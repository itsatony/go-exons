package exons

import (
	"fmt"
	"strings"
)

// This file makes the frontmatter `inputs:` block MEAN something at execution time.
//
// Until v0.21.0 Spec.Inputs was entirely inert: nothing in Execute ever read it, and
// InputDef.Default had no application site anywhere in the library. A declared input was a
// promise to a form builder and nothing more, so the only way to actually USE one was
// {~exons.var~} — the same verb that reads arbitrary runtime data, which is why no tool could
// tell a mistyped input from a legitimate context variable.
//
// Injection under a reserved root fixes that at the root. Because `input` is an ordinary
// context path, exons.if and exons.for reach declared inputs with no grammar change at all:
// eval="input.verbose" and in="input.sources" simply work.

// contextWithInputs returns a context in which every input this template DECLARES is readable
// under the reserved `input` root, with InputDef.Default supplied wherever the caller bound
// nothing. It never mutates the given context or the caller's data map.
//
// Four rules, each load-bearing:
//
//  1. NO-OP WHEN data["input"] IS NOT A MAP. `input` is an extremely likely ordinary variable
//     name in a prompt — data["input"] = "the user's question" is idiomatic — and an include
//     copies its non-reserved attributes into the child data as STRINGS, so
//     {~exons.include template="x" input="y" /~} lets a DOCUMENT put a string there. Rather
//     than fail, such a template renders exactly as it did before this feature existed.
//
//     An explicit NIL is not that case and must not take this exit. data["input"] = nil says
//     "nothing is bound", not "input is my own variable", and bailing on it skipped injection
//     entirely — so a document's declared defaults silently did not apply, which is the one
//     outcome this whole file exists to prevent. A nil root proceeds as an empty binding.
//
//  2. MERGE PER KEY, CALLER WINS. Only absent (or nil, or empty-string) keys receive the
//     declared default. Replacing the caller's map wholesale would mean that adding an input
//     to a spec silently unbinds it for every existing caller.
//
//  3. DEEP-COPY EVERY DEFAULT. A *Template is built to be executed many times, concurrently.
//     InputDef.Default holds YAML-decoded values, so aliasing a map or slice default into a
//     live context would share it across renders — a cross-request contamination bug, not a
//     tidiness issue.
//
//  4. PRESENT-AS-NIL IS THE UNIVERSAL ZERO. A declared input with neither a bound value nor a
//     default lands as nil, PRESENT. Every consumer already does the right thing with nil: it
//     renders as the empty string, evaluates falsy in a condition, and iterates zero times in
//     a loop — so an unbound optional multiselect behaves sanely with no executor change. The
//     payoff is the equivalence PRESENT ⇔ DECLARED, which is what lets exons.input report an
//     undeclared name as the author error it is.
func (t *Template) contextWithInputs(execCtx *Context) *Context {
	if execCtx == nil || t.spec == nil || len(t.spec.Inputs) == 0 {
		return execCtx
	}

	// Read through the parent chain: when ExecuteWithContext is handed a child context whose
	// parent carries the bindings, writing a fresh map into the child would SHADOW them
	// entirely — path lookup is all-or-nothing per scope, not a per-key overlay.
	bound, hasBinding := execCtx.Get(ContextKeyInput)
	binding, ok := asBindingMap(bound)
	if hasBinding && bound != nil && !ok {
		return execCtx // rule 1
	}

	// Deep-copied for the same reason the defaults below are, and it was an omission that these
	// were not: Context.Get returns the LIVE value, not a copy, so aliasing a caller's nested map
	// in here would share it across every render that context feeds. The map this replaces came
	// from Context.Data(), which deep-copies — so a shallow merge here silently WEAKENED a
	// thread-safety property the context documents. Spec.BindInputs, the sibling path, deep-copies
	// caller values already; the two now agree.
	//
	// ⚠ "Agree" is the accurate claim, NOT "safe for every shape". deepCopyValue copies the
	// YAML/JSON shapes exhaustively and returns everything else — a struct, a pointer, a []byte, a
	// map with non-string keys — BY REFERENCE, as its own comment says. A Go caller binding one of
	// those to a declared input therefore still shares it live across concurrent renders. That is
	// a property of deepCopyValue, unchanged here and identical on both paths; widening it belongs
	// there, not in two call sites that would then have to agree by memory.
	merged := make(map[string]any, len(binding)+len(t.spec.Inputs))
	for k, v := range binding {
		merged[k] = deepCopyValue(v)
	}
	for name, def := range t.spec.Inputs {
		if def == nil {
			continue
		}
		if v, present := merged[name]; present && !isUnboundInputValue(v) {
			continue // rule 2
		}
		merged[name] = deepCopyValue(def.Default) // rules 3 and 4
	}

	data := execCtx.Data() // already a deep copy of the direct data
	if data == nil {
		data = make(map[string]any, 1)
	}
	data[ContextKeyInput] = merged
	return execCtx.withData(data)
}

// asBindingMap normalises the value found under the reserved root. A map[string]string is
// accepted as well as map[string]any because path lookup traverses both and callers do
// produce the narrower one. Anything else — including nil — yields ok=false.
func asBindingMap(val any) (map[string]any, bool) {
	switch m := val.(type) {
	case nil:
		return map[string]any{}, false
	case map[string]any:
		out := make(map[string]any, len(m))
		for k, v := range m {
			out[k] = v
		}
		return out, true
	case map[string]string:
		out := make(map[string]any, len(m))
		for k, v := range m {
			out[k] = v
		}
		return out, true
	default:
		return nil, false
	}
}

// isUnboundInputValue reports whether a caller-supplied value should still receive the
// declared default. A bound empty string counts as unbound, matching how {~exons.env~} treats
// an empty variable: for the form-driven callers this exists for, an untouched field and a
// cleared field are the same gesture.
func isUnboundInputValue(val any) bool {
	if val == nil {
		return true
	}
	s, ok := val.(string)
	return ok && s == ""
}

// declaresInput reports whether the template's frontmatter declares the given input name.
// A dotted reference (name="user.email" on a structured input) is judged by its first
// segment, since that is the key the frontmatter declares.
func (t *Template) declaresInput(name string) bool {
	if t.spec == nil || name == "" {
		return false
	}
	root, _, _ := strings.Cut(name, PathSeparator)
	_, ok := t.spec.Inputs[root]
	return ok
}

// BindInputs applies this spec's declared defaults to a set of caller-supplied values and
// returns the map to place under the reserved `input` key of Execute's data — so a caller
// never has to hand-roll the namespace or re-implement the default rules:
//
//	data := map[string]any{exons.ContextKeyInput: spec.BindInputs(formValues)}
//
// The returned map is independent of both the spec and the input map. Passing nil is fine and
// yields the defaults alone. Undeclared keys are preserved, not dropped — validating them is
// ValidateInputBinding's job, and silently discarding a caller's value would be worse than
// reporting it.
func (s *Spec) BindInputs(values map[string]any) map[string]any {
	out := make(map[string]any, len(values)+len(s.Inputs))
	for k, v := range values {
		out[k] = deepCopyValue(v)
	}
	for name, def := range s.Inputs {
		if def == nil {
			continue
		}
		if v, present := out[name]; present && !isUnboundInputValue(v) {
			continue
		}
		out[name] = deepCopyValue(def.Default)
	}
	return out
}

// ValidateInputBinding checks caller-supplied values against what the spec declared, and is
// the supported way to enforce a required input.
//
// It runs BEFORE any render, on purpose. Render time is far too late to ask a user for a
// missing value, and a check expressed as a tag attribute would fire or not depending on
// whether the tag sat inside a branch that happened to be taken — the same document would be
// "valid" or not depending on data. That is why {~exons.input~} refuses a `required=`
// attribute and points here instead.
//
// Returns every violation rather than the first, so a form can mark all its bad fields in one
// pass. An empty (or nil) result means the binding is acceptable.
func (s *Spec) ValidateInputBinding(values map[string]any) []error {
	var errs []error
	for _, name := range s.OrderedInputKeys() {
		def := s.Inputs[name]
		if def == nil {
			continue
		}
		val, present := values[name]
		if !present || isUnboundInputValue(val) {
			// A declared default satisfies requiredness — the value is never actually
			// absent at render, so refusing here would make an unsubmittable form.
			if def.Required && def.Default == nil {
				errs = append(errs, fmt.Errorf(ErrFmtInputRequired, name))
			}
			continue
		}
		errs = append(errs, validateInputValue(name, def, val)...)
	}
	return errs
}

// validateInputValue checks one bound value against one declaration. Only constraints the
// SPEC states are enforced; the input-kind vocabulary itself stays open, because a document
// may be written against a newer version of this library than the one reading it.
func validateInputValue(name string, def *InputDef, val any) []error {
	var errs []error

	switch def.Type {
	case InputTypeSelect:
		if s, ok := val.(string); ok {
			if !optionAllows(def.Options, s) {
				errs = append(errs, fmt.Errorf(ErrFmtInputNotAnOption, name, s))
			}
		}
	case InputTypeMultiselect, InputTypeSort:
		for _, item := range asStringList(val) {
			if !optionAllows(def.Options, item) {
				errs = append(errs, fmt.Errorf(ErrFmtInputNotAnOption, name, item))
			}
		}
	case InputTypeFileUpload:
		if def.MaxFiles > 0 {
			if n, ok := boundLength(val); ok && n > def.MaxFiles {
				errs = append(errs, fmt.Errorf(ErrFmtInputTooManyFiles, name, n, def.MaxFiles))
			}
		}
	}
	return errs
}

// optionAllows reports whether v is one of the declared option values. An input that declares
// no options constrains nothing — a consumer degrades such a kind to free text, so validating
// against an empty set would reject every value.
func optionAllows(options []InputOption, v string) bool {
	if len(options) == 0 {
		return true
	}
	for _, opt := range options {
		if opt.Value == v {
			return true
		}
	}
	return false
}

// asStringList coerces a bound list value to the strings it contains, ignoring non-string
// elements rather than reporting them: the value vocabulary is open, and an unrecognised
// element shape is not evidence of an option violation.
func asStringList(val any) []string {
	switch v := val.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// boundLength returns the element count of a bound list value.
func boundLength(val any) (int, bool) {
	switch v := val.(type) {
	case []any:
		return len(v), true
	case []string:
		return len(v), true
	default:
		return 0, false
	}
}
