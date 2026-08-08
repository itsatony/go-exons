package internal

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
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
// recursing through slices, arrays, maps, pointers and STRUCT FIELDS to the same bound
// renderValue uses.
//
// Strings are untouched: a string is text the caller chose to bind, and withholding it would
// break every ordinary input. Only a byte slice — the shape a file body arrives in — is
// replaced. That includes a byte slice holding UTF-8 text: a text file read into a []byte is
// exactly as much of an accidental paste as a PDF, and the shape carries no evidence of which
// it is. A caller that means to inline a document's text binds a string.
//
// ⚠ THIS FUNCTION MUST TRAVERSE EVERYTHING renderValue TRAVERSES. The guarantee is not "we
// redact the shapes we thought of", it is "no byte slice renderValue can reach survives". Its
// first version handled Pointer/Slice/Map only, so a struct with an exported []byte field fell
// to the `default` arm untouched — and renderValue has an explicit reflect.Struct case that
// walks exported fields and renders a uint8-element slice as string(rv.Bytes()). A caller
// binding `struct{ Name string; Body []byte }` therefore pasted the entire file body into the
// prompt, through the one function written to stop precisely that.
//
// This is the same defect maxRenderDepth's comment records against the ORIGINAL renderValue:
// there too the untraversed arm was the struct one, and there too the test that should have
// caught it used a shape that never reached the hole. Traversal parity with renderValue is the
// invariant; TestWithholdBinaryTraversesEveryKindRenderValueDoes pins it.
func withholdBinary(val any, depth int) any {
	if val == nil {
		return val
	}

	// ⚠ AT THE BOUND, ELIDE — DO NOT RETURN THE VALUE UNSWEPT.
	//
	// Returning val here was fail-OPEN. This function REBUILDS the value, and its pointer arm
	// flattens each indirection away, so the rebuilt structure can be shallower than the one
	// swept: a byte slice behind eight pointers was reached at depth 8 here (bound hit, handed
	// back raw) and at depth 0 by renderValue, which then rendered it as text. Eliding instead
	// costs an input nested deeper than the bound its content — which renderValue would elide
	// anyway — and removes the whole depth-misalignment class.
	if depth >= inputMaxSweepDepth {
		return elidedValue
	}

	rv := reflect.ValueOf(val)

	// The binary rule is absolute and is therefore decided BEFORE the self-rendering check
	// below: a `type Blob []byte` that also implements fmt.Stringer is still a byte slice, and
	// the guarantee does not have a "unless the type would rather print itself" clause.
	//
	// The ELEMENT-KIND match is the same one renderValue uses, so a named type — json.RawMessage,
	// `type Blob []byte` — is caught too, not just the literal []byte. Arrays are matched
	// alongside slices: a [32]byte digest is as much a byte sequence as a []byte, and gating the
	// check on Kind()==Slice let it through to be rendered as a list of small integers.
	switch rv.Kind() {
	case reflect.Slice:
		if rv.IsNil() {
			return val // renderValue emits "" for a nil slice; withholding would be noise
		}
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return withheldBinaryValue
		}
	case reflect.Array:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return withheldBinaryValue
		}
	}

	// A value renderValue renders through its OWN method never exposes a field, so there is
	// nothing to sweep — and rebuilding it would destroy the rendering. time.Time is the case
	// that makes this mandatory rather than tidy: it is a struct of unexported fields, so the
	// struct arm below would turn every bound timestamp into an empty map.
	if isSelfRenderingValue(val) {
		return val
	}

	switch rv.Kind() {
	// No reflect.Interface arm: reflect.ValueOf takes an `any` and always reports the
	// DYNAMIC kind, so Kind() == Interface is unreachable here. The old arm was dead code
	// that read like coverage.
	case reflect.Pointer:
		if rv.IsNil() {
			return val
		}
		return withholdBinary(rv.Elem().Interface(), depth+1)

	case reflect.Slice, reflect.Array:
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = withholdBinary(rv.Index(i).Interface(), depth+1)
		}
		return out

	case reflect.Map:
		if rv.IsNil() {
			return val
		}
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			key := renderValue(iter.Key().Interface(), DefaultValueSeparator)
			out[key] = withholdBinary(iter.Value().Interface(), depth+1)
		}
		return out

	case reflect.Struct:
		// Rebuilt as a map, which costs the struct its DECLARATION-order rendering (a map
		// renders sorted). That is a deliberate trade: the alternative — pre-scanning for
		// binary and returning the struct untouched when clean — needs a second traversal
		// that mirrors this one, and a mirror that drifts fails OPEN, which is the one
		// direction this function may not fail in. exons.var still renders a struct in
		// declaration order; only a declared INPUT pays the ordering.
		//
		// Unexported fields are skipped: renderValue skips them too (so they cannot leak),
		// and calling Interface() on one panics.
		rt := rv.Type()
		out := make(map[string]any, rv.NumField())
		for i := 0; i < rv.NumField(); i++ {
			f := rt.Field(i)
			if !f.IsExported() {
				continue
			}
			out[f.Name] = withholdBinary(rv.Field(i).Interface(), depth+1)
		}
		return out

	default:
		// Only kinds that cannot contain another value reach here — the scalars, plus chan,
		// func, complex and unsafe.Pointer, none of which can hide a byte slice from
		// renderValue either.
		return val
	}
}

// isSelfRenderingValue reports whether renderValue would render val through the value's own
// method rather than by walking its structure.
//
// It mirrors renderAtDepth's leading type switch, and the mirroring is the point: a value that
// short-circuits there must short-circuit here, or withholdBinary rebuilds something whose
// rendering it has just destroyed.
func isSelfRenderingValue(val any) bool {
	switch val.(type) {
	case time.Time, fmt.Stringer, error:
		return true
	default:
		return false
	}
}

// fileManifestEntries recognises a value as a list of uploaded files and renders one bullet
// line per file. It returns ok=false for every other shape, so an ordinary list value falls
// through to renderValue untouched.
//
// The recognised shape is a non-empty slice in which EVERY element is a map whose keys are all
// drawn from the file vocabulary {name, mime_type, size_bytes}, carrying a non-empty string
// under InputFileKeyName — AND in which at least one element actually carries a file-specific
// key (mime_type or size_bytes).
//
// ⚠ BOTH EXTRA CONDITIONS EARN THEIR KEEP; "every element is a map with a name" DOES NOT
// DESCRIBE A FILE. It describes a list of named objects, which is one of the most ordinary
// shapes a caller can bind — `[{"name":"GPT-4","provider":"openai"}, …]` was rendered as a
// bullet list with `provider` SILENTLY DROPPED from the prompt. Requiring the key set to be a
// subset of the vocabulary is what rejects that, and requiring one file-specific key is what
// keeps a list of bare `{"name": …}` maps — genuinely ambiguous, and no more a file list than a
// person list — falling through to renderValue, which preserves it whole.
//
// The bar is deliberately on the side of NOT claiming the value: a missed manifest renders as
// an ordinary map list, which is merely less pretty, while a false manifest destroys data.
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
	sawFileKey := false
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		name, ok := m[InputFileKeyName].(string)
		if !ok || name == "" {
			return nil, false
		}
		for k := range m {
			if !isFileManifestKey(k) {
				return nil, false
			}
			if k != InputFileKeyName {
				sawFileKey = true
			}
		}
		entries = append(entries, inputManifestBullet+name+fileManifestDetail(m))
	}
	if !sawFileKey {
		return nil, false
	}
	return entries, true
}

// isFileManifestKey reports whether a key belongs to the file-descriptor vocabulary. A map
// carrying anything else is not a file descriptor, and rendering it as one would discard the
// keys the manifest has no place for.
func isFileManifestKey(key string) bool {
	switch key {
	case InputFileKeyName, InputFileKeyMimeType, InputFileKeySizeBytes:
		return true
	default:
		return false
	}
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
