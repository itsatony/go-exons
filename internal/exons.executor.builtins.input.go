package internal

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// InputResolver handles the exons.input built-in tag: a reference to a value the DOCUMENT
// declared in its frontmatter `inputs:` block.
//
// WHY THIS IS NOT A SYNONYM FOR exons.var. Before this verb, a declared input and an
// arbitrary runtime context variable were spelled identically — both {~exons.var name="x"~}.
// exons.var reaches the WHOLE context: the caller's data map, an include's `with=`, an
// inherited parent scope from a for-iteration. So no tool could tell a typo'd input from a
// legitimate context variable, and "this input is declared but never referenced" was
// undecidable no matter how good the analysis. Splitting the verb makes the question
// answerable by construction.
//
// exons.var is UNCHANGED and is not deprecated. It remains the context-variable verb; the
// defect was that it was ALSO the input verb.
//
// The lookup is deliberately plain: exons.input name="tone" resolves the context path
// "input.tone" and nothing more. Declared inputs are injected under that reserved root before
// execution (see Template.ExecuteWithContext), which is why eval="input.verbose" and
// in="input.sources" work with no grammar change to exons.if / exons.for. A private lookup
// space would have made the verb invisible to control flow — a half-language where you can
// print an input but cannot branch on it.
type InputResolver struct{}

// NewInputResolver creates a new InputResolver.
func NewInputResolver() *InputResolver {
	return &InputResolver{}
}

// TagName returns the tag name for this resolver.
func (r *InputResolver) TagName() string {
	return TagNameInput
}

const (
	// InputFileKeyName is the only REQUIRED key of a file-descriptor map. Its presence on
	// every element is what identifies a value as an upload manifest.
	InputFileKeyName = "name"
	// InputFileKeyMimeType is the optional declared media type of an uploaded file.
	InputFileKeyMimeType = "mime_type"
	// InputFileKeySizeBytes is the optional size of an uploaded file, in bytes.
	InputFileKeySizeBytes = "size_bytes"

	// inputManifestSeparator joins manifest entries when the tag declares no `join`. A
	// filename list reads as a list, not as prose, so it defaults to a newline where an
	// ordinary list value defaults to ", ".
	inputManifestSeparator = "\n"
	// inputManifestBullet prefixes each manifest entry.
	inputManifestBullet = "- "

	// withheldBinaryValue replaces any byte slice reached while rendering an input.
	//
	// ⚠ THIS IS A GUARANTEE, NOT A FORMATTING CHOICE. renderValue matches a slice on its
	// ELEMENT KIND and returns string(rv.Bytes()) for uint8 — deliberately, so that
	// json.RawMessage renders as text. That arm is correct for exons.var and catastrophic
	// here: a caller binding an uploaded file's bytes to a declared input would paste the
	// entire file body into the prompt, by accident rather than by misuse. Every value an
	// exons.input renders is swept for byte slices first, at every depth.
	withheldBinaryValue = "«binary value withheld»"

	// inputMaxSweepDepth bounds the binary sweep. It matches maxRenderDepth for the same
	// reason that bound exists: the value comes from a caller-supplied context and a Go
	// caller can hand it a cycle.
	inputMaxSweepDepth = maxRenderDepth

	// byteUnitStep is the divisor between adjacent size units in a manifest entry.
	byteUnitStep = 1024
)

// byteUnits are the suffixes used when rendering a file size in a manifest entry.
var byteUnits = []string{"B", "KB", "MB", "GB", "TB"}

// Resolve renders the value of a declared input.
//
// The resolution order is deliberate and differs from exons.var in one load-bearing way:
//
//  1. ABSENT under the input root → an error, routed by the executor through the configured
//     error strategy. Because injection makes every DECLARED input present (nil when it has
//     neither a bound value nor a frontmatter default), absent means UNDECLARED — which is
//     an author typo, and the single most valuable diagnostic this verb offers.
//  2. Present and non-empty → rendered.
//  3. Present but nil or empty → the tag's `default=`, else the empty string, never an error.
//
// Step 3 is why this cannot delegate to VarResolver: VarResolver consults `default=` only on
// a lookup MISS, and after injection a declared input never misses.
func (r *InputResolver) Resolve(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error) {
	accessor, ok := execCtx.(ContextAccessor)
	if !ok {
		return "", NewBuiltinError(ErrMsgInvalidContext, TagNameInput)
	}

	// Checked HERE and not only in Validate, because the executor never calls a resolver's
	// Validate — the method is on the InternalResolver interface and nothing in the engine
	// invokes it. A refusal expressed only there would be dead code, which is the same defect
	// class as `inputs:` itself being inert before this release. Both entry points delegate to
	// one helper so the two answers cannot drift.
	if err := validateInputAttrs(attrs); err != nil {
		return "", err
	}

	name, _ := attrs.Get(AttrName)
	val, found := accessor.Get(ContextKeyInput + PathSeparator + name)
	if !found {
		return "", newInputNotDeclaredError(name, declaredInputNames(accessor), ShouldShowHint(attrs))
	}

	if isEmptyInputValue(val) {
		if defaultVal, hasDefault := attrs.Get(AttrDefault); hasDefault {
			return defaultVal, nil
		}
		return "", nil
	}

	sep, sepDeclared := attrs.Get(AttrJoin)
	return renderInputValue(val, sep, sepDeclared), nil
}

// Validate checks the attributes at parse time.
//
// `required=` is refused rather than ignored. InputDef.Required in the frontmatter is already
// the single source of truth, render time is far too late to ask a user for a value, and a
// required= sitting inside an unreached exons.if branch would enforce inconsistently — the
// same document would be "valid" or not depending on data. Spec.ValidateInputBinding is the
// supported way to enforce requiredness, and it runs BEFORE any render.
func (r *InputResolver) Validate(attrs Attributes) error {
	return validateInputAttrs(attrs)
}

// validateInputAttrs is the single statement of what attributes exons.input accepts, shared
// by Validate and Resolve so a caller that validates ahead of time gets the same verdict the
// executor will.
func validateInputAttrs(attrs Attributes) error {
	if !attrs.Has(AttrName) {
		return NewBuiltinError(ErrMsgMissingNameAttr, TagNameInput)
	}
	if attrs.Has(AttrRequired) {
		return NewBuiltinError(ErrMsgInputRequiredAttr, TagNameInput)
	}
	return nil
}

// declaredInputNames lists the keys under the reserved input root, for did-you-mean
// suggestions. It reads them through the accessor rather than through KeyLister because
// KeyLister.Keys() returns only TOP-LEVEL keys — under which "input" is a single entry, so
// every suggestion in every input-declaring document would collapse to the word "input".
func declaredInputNames(accessor ContextAccessor) []string {
	root, ok := accessor.Get(ContextKeyInput)
	if !ok {
		return nil
	}
	var names []string
	switch m := root.(type) {
	case map[string]any:
		for k := range m {
			names = append(names, k)
		}
	case map[string]string:
		for k := range m {
			names = append(names, k)
		}
	default:
		return nil
	}
	sort.Strings(names)
	return names
}

// newInputNotDeclaredError builds the undeclared-input diagnostic, with suggestions drawn
// from the names the document actually declared.
func newInputNotDeclaredError(name string, declared []string, showHint bool) *BuiltinError {
	message := ErrMsgInputNotDeclared
	if suggestions := FindSimilarStrings(ExtractPathPrefix(name), declared, 3); len(suggestions) > 0 {
		message += FormatSuggestions(suggestions)
	} else if len(declared) > 0 {
		message += FormatAvailableKeys(declared, 5)
	}
	if showHint {
		message = AppendHint(message, HintInputNotDeclared)
	}
	return NewBuiltinError(message, TagNameInput).WithMetadata(MetaKeyPath, name)
}

// isEmptyInputValue reports whether a bound value should fall back to the tag's `default=`.
//
// A bound empty string counts as unbound, matching EnvResolver. The consequence is that a
// caller cannot force-blank an input that declares a default — which is right for the
// form-driven callers this verb exists for, where an untouched field and a cleared field are
// the same gesture from the user.
func isEmptyInputValue(val any) bool {
	if val == nil {
		return true
	}
	s, ok := val.(string)
	return ok && s == ""
}

// renderInputValue converts a declared input's bound value to substitutable text.
//
// It differs from renderValue in exactly two ways, both of which are safety properties rather
// than formatting preferences:
//
//   - every byte slice reached at any depth is withheld (see withheldBinaryValue);
//   - a value shaped like a list of uploaded files renders as a filename MANIFEST.
//
// The manifest is selected by the value's SHAPE, not by the input's declared `type:`.
// Shape-based dispatch fails closed: a value carrying file bytes is withheld whatever the
// frontmatter claims the input is, and an author who mislabels an upload as `type: text` is
// still protected. Type-based dispatch would have had to thread the declared types down
// through every child context the executor creates, and a single missed propagation — into a
// for-loop body, say — would have silently restored the leak.
func renderInputValue(val any, sep string, sepDeclared bool) string {
	safe := withholdBinary(val, 0)

	if entries, ok := fileManifestEntries(safe); ok {
		separator := inputManifestSeparator
		if sepDeclared {
			separator = sep
		}
		return strings.Join(entries, separator)
	}

	if !sepDeclared {
		sep = DefaultValueSeparator
	}
	return renderValue(safe, sep)
}

// withholdBinary returns val with every byte slice replaced by withheldBinaryValue,
// recursing through slices, maps and pointers to the same bound renderValue uses.
//
// Strings are untouched: a string is text the caller chose to bind, and withholding it would
// break every ordinary input. Only a byte slice — the shape a file body arrives in — is
// replaced.
func withholdBinary(val any, depth int) any {
	if val == nil || depth >= inputMaxSweepDepth {
		return val
	}

	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return val
		}
		return withholdBinary(rv.Elem().Interface(), depth+1)

	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			return val
		}
		// The same ELEMENT-KIND match renderValue uses, so a named type — json.RawMessage,
		// `type Blob []byte` — is caught too, not just the literal []byte.
		if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
			return withheldBinaryValue
		}
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = withholdBinary(rv.Index(i).Interface(), depth+1)
		}
		return out

	case reflect.Map:
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			key := renderValue(iter.Key().Interface(), DefaultValueSeparator)
			out[key] = withholdBinary(iter.Value().Interface(), depth+1)
		}
		return out

	default:
		return val
	}
}

// fileManifestEntries recognises a value as a list of uploaded files and renders one bullet
// line per file. It returns ok=false for every other shape, so an ordinary list value falls
// through to renderValue untouched.
//
// The recognised shape is a non-empty slice whose EVERY element is a map carrying a non-empty
// string under InputFileKeyName. Requiring every element keeps the rule from firing on a
// heterogeneous list that merely happens to contain one named map.
//
// A plain []string is NOT recognised, on purpose: it is indistinguishable from an ordinary
// multiselect value, and rendering `a, b` for a multiselect matters more than bulleting two
// filenames. A runtime that has uploaded files has their metadata and can bind the map form.
func fileManifestEntries(val any) ([]string, bool) {
	items, ok := val.([]any)
	if !ok || len(items) == 0 {
		return nil, false
	}

	entries := make([]string, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		name, ok := m[InputFileKeyName].(string)
		if !ok || name == "" {
			return nil, false
		}
		entries = append(entries, inputManifestBullet+name+fileManifestDetail(m))
	}
	return entries, true
}

// fileManifestDetail renders the optional parenthesised detail after a filename. Absent both
// optional keys it returns the empty string, so a bare {"name": …} entry reads as just the
// filename rather than as an empty pair of brackets.
func fileManifestDetail(m map[string]any) string {
	var parts []string
	if mime, ok := m[InputFileKeyMimeType].(string); ok && mime != "" {
		parts = append(parts, mime)
	}
	if size, ok := toByteSize(m[InputFileKeySizeBytes]); ok {
		parts = append(parts, humanizeBytes(size))
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

// toByteSize coerces a size to a non-negative count of bytes. JSON decodes a number to
// float64 and a Go caller is as likely to pass an int, so both are accepted.
func toByteSize(val any) (float64, bool) {
	if val == nil {
		return 0, false
	}
	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if rv.Int() < 0 {
			return 0, false
		}
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		if rv.Float() < 0 {
			return 0, false
		}
		return rv.Float(), true
	default:
		return 0, false
	}
}

// humanizeBytes renders a byte count the way a person reads it. Bytes are shown whole; every
// larger unit carries one decimal, so a manifest line stays short enough to scan.
func humanizeBytes(size float64) string {
	unit := 0
	for size >= byteUnitStep && unit < len(byteUnits)-1 {
		size /= byteUnitStep
		unit++
	}
	if unit == 0 {
		return strconv.FormatFloat(size, 'f', 0, 64) + " " + byteUnits[unit]
	}
	return fmt.Sprintf("%.1f %s", size, byteUnits[unit])
}
