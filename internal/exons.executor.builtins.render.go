package internal

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Value rendering — how a bound value becomes the text that replaces a
// {~exons.var~} tag in the rendered prompt.
//
// ⚠ THIS USED TO BE `fmt.Sprintf("%v")` FOR EVERY NON-SCALAR, AND THAT IS WHAT A
// MODEL READ. Go's default composite formatting is a debugging aid, not prose: a
// `multiselect`/`sort` value arrived as `[cost speed]` and an `associate` value as
// `[map[left:x right:y]]`. Both are legal Go and both are gibberish in a sentence
// that says "rank these criteria:". The input kinds that bind a LIST or a set of
// PAIRS (InputTypeMultiselect, InputTypeSort, InputTypeAssociate — see
// exons.constants.go) are exactly the kinds this hit, so the vocabulary the v0.19.0
// input types promised could not be rendered legibly by the executor that consumes
// it.
//
// The rendering is deliberately PROSE-shaped rather than JSON-shaped, because the
// consumer is a language model reading a sentence, and the overwhelmingly common
// case is a flat list of scalars. A template that wants JSON has the value in its
// own hands upstream; a template that wants a different separator has `join`.
//
// Nested composites keep delimiters (`[a, b]`, `(left: x, right: y)`) so a list of
// pairs does not smear into one comma-run — the top level is unwrapped because a
// bare `cost, speed` is what reads correctly inside a sentence.
//
// Two shapes deliberately keep the old `%v` fallback. A plain STRUCT renders as
// `{a b}` — it is unreachable from a JSON-decoded value (which yields maps), and
// inventing a field-name rendering for it would guess at a Go caller's intent. And
// a non-string MAP KEY sorts by its rendered form, so map[int]string orders 1, 10,
// 2; JSON object keys are strings, so this too is a Go-caller-only edge. Both are
// recorded here as choices rather than left to be rediscovered.

const (
	// DefaultValueSeparator joins the elements of a top-level list, and the entries
	// of a top-level map, when the tag declares no `join` attribute.
	DefaultValueSeparator = ", "

	// nestedValueSeparator joins inside a nested composite. Not author-controllable:
	// `join` names the separator between the values the author is talking about, and
	// re-using it one level down would make `join=" > "` produce `a > b > c > d` for
	// a list of pairs — two different relations spelled the same way.
	nestedValueSeparator = ", "

	// maxRenderDepth bounds recursion. Values reaching a template come from JSON in
	// every current caller, so they are acyclic — but valueToString takes `any` from
	// a caller-supplied Context, and a Go caller CAN hand it a cycle. Past the bound
	// we fall back to %v, which terminates.
	//
	// ⚠ EVERY RECURSIVE ARM MUST INCREMENT IT, INCLUDING THE POINTER DEREF. The first
	// version of this file incremented only on slice elements and map entries, so the
	// simplest cycle of all — `var i any; i = &i` — never reached the bound and
	// recursed until the stack died. The test that was supposed to cover it used a
	// STRUCT field, which lands on the untraversed `%v` arm, so it passed against the
	// hole. A depth bound is only a bound if it counts every step.
	maxRenderDepth = 8

	// elidedValue replaces anything nested deeper than maxRenderDepth.
	elidedValue = "…"
)

// valueToString converts a bound value to the text substituted into the template,
// using the default separator. See renderValue for the rules.
func valueToString(val any) string {
	return renderValue(val, DefaultValueSeparator)
}

// renderValue converts a bound value to substitutable text, joining a top-level
// list's elements (or a top-level map's entries) with sep.
//
// Scalars render as themselves. A list renders as its elements joined by sep. A map
// renders as `key: value` entries joined by sep, with the keys SORTED — a Go map has
// no order, so rendering it in range order would make the same value produce
// different prompts on different runs, which is worse than an order the author did
// not choose. Anything else falls back to %v, unchanged from before: the value
// vocabulary is open and an unrecognised shape is better shown than swallowed.
func renderValue(val any, sep string) string {
	return renderAtDepth(val, sep, false, 0)
}

// renderAtDepth is the recursive worker. nested=true asks for the delimited form,
// used for a composite that sits INSIDE another composite.
func renderAtDepth(val any, sep string, nested bool, depth int) string {
	switch v := val.(type) {
	case nil:
		return ""
	case string:
		return v
	case time.Time:
		// A deliberate choice, not a fallthrough: time.Time IS a fmt.Stringer, and its
		// String() is Go's debug form ("2009-11-10 23:00:00 +0000 UTC"). RFC 3339 is
		// what a model reads unambiguously and what every other date surface in this
		// library emits.
		return v.Format(time.RFC3339)
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int8, int16, int32, int64:
		return strconv.FormatInt(reflect.ValueOf(v).Int(), 10)
	case uint, uint8, uint16, uint32, uint64:
		return strconv.FormatUint(reflect.ValueOf(v).Uint(), 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case fmt.Stringer:
		// After the concrete scalars so a Stringer cannot shadow them, before the
		// composites so a caller's own type keeps the rendering it declared.
		return v.String()
	case error:
		return v.Error()
	}

	// ⚠ AT THE BOUND, ELIDE — DO NOT HAND THE VALUE TO fmt. `%v` was the original
	// escape here and it is not a safe one: fmt does not detect a slice that contains
	// itself, so the "fallback that terminates" recursed until the stack died. An
	// ellipsis is also the more honest answer — it says something was left out,
	// where a truncated debug dump says nothing at all.
	if depth >= maxRenderDepth {
		return elidedValue
	}

	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return ""
		}
		// depth+1, not depth — see maxRenderDepth.
		return renderAtDepth(rv.Elem().Interface(), sep, nested, depth+1)

	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			return ""
		}
		// A byte slice is text far more often than a list of small integers. Matched
		// on the ELEMENT KIND rather than on `case []byte`, so a named type —
		// json.RawMessage, `type Blob []byte` — renders as text too instead of as
		// `104, 105`.
		if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
			return string(rv.Bytes())
		}
		inner := sep
		if nested {
			inner = nestedValueSeparator
		}
		parts := make([]string, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			parts = append(parts, renderAtDepth(rv.Index(i).Interface(), inner, true, depth+1))
		}
		joined := strings.Join(parts, inner)
		if nested {
			return "[" + joined + "]"
		}
		return joined

	case reflect.Map:
		if rv.IsNil() {
			return ""
		}
		inner := sep
		if nested {
			inner = nestedValueSeparator
		}
		// ⚠ SORT THE KEYS, NOT THE RENDERED ENTRIES. Sorting `"k: v"` strings looks
		// equivalent and is not: ':' sorts after digits and space, so {"a":1,"a1":2}
		// came out as `a1: 2, a: 1` and a map[int]string ordered 1, 10, 2. Still
		// deterministic — which was the requirement — but in an order no one asked
		// for and the comment claimed was key order.
		type entry struct{ key, value string }
		entries := make([]entry, 0, rv.Len())
		for _, k := range rv.MapKeys() {
			entries = append(entries, entry{
				key:   renderAtDepth(k.Interface(), nestedValueSeparator, true, depth+1),
				value: renderAtDepth(rv.MapIndex(k).Interface(), nestedValueSeparator, true, depth+1),
			})
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].key < entries[j].key })
		parts := make([]string, 0, len(entries))
		for _, e := range entries {
			parts = append(parts, e.key+": "+e.value)
		}
		joined := strings.Join(parts, inner)
		if nested {
			return "(" + joined + ")"
		}
		return joined

	case reflect.Struct:
		// A struct renders like a map, but in DECLARATION order rather than sorted —
		// a struct has an order and it is the author's. Unexported fields are skipped
		// (they cannot be read reflectively anyway).
		//
		// Handled explicitly for two reasons. `%v` gave `{a b}`, the same gibberish
		// this whole file exists to stop for lists and maps. And `%v` on a struct
		// holding a self-referential slice does not terminate, which would have left
		// one unbounded path after the depth fix.
		inner := sep
		if nested {
			inner = nestedValueSeparator
		}
		rt := rv.Type()
		parts := make([]string, 0, rv.NumField())
		for i := 0; i < rv.NumField(); i++ {
			f := rt.Field(i)
			if !f.IsExported() {
				continue
			}
			parts = append(parts, f.Name+": "+renderAtDepth(rv.Field(i).Interface(), nestedValueSeparator, true, depth+1))
		}
		joined := strings.Join(parts, inner)
		if nested {
			return "(" + joined + ")"
		}
		return joined

	default:
		// Only the kinds that cannot contain another value reach here — chan, func,
		// complex, unsafe.Pointer — so `%v` is bounded by construction.
		return fmt.Sprintf("%v", val)
	}
}
