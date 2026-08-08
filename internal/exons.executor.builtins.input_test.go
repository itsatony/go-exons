package internal

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// withholdBinary — the guarantee, tested where the marker constant lives
// =============================================================================

// binarySecret is the payload every case below hides somewhere. Its first bytes are also used
// to catch a NUMERIC leak, which is the failure mode a NotContains-on-the-string test misses.
const binarySecret = "SENSITIVE FILE BODY"

// withholdProbe is a struct with an exported byte-slice field — the shape that defeated the
// first version of withholdBinary, and the one renderValue walks with an explicit
// reflect.Struct arm.
type withholdProbe struct {
	Name string
	Body []byte
}

// withholdUnexported hides its bytes in an unexported field. renderValue cannot read it either,
// so the correct behaviour is that nothing leaks WITHOUT the sweep having to reach it.
type withholdUnexported struct {
	Name string
	body []byte //nolint:unused // read reflectively by neither renderValue nor the sweep — that is the assertion
}

// withholdStringer renders itself, so withholdBinary must leave it alone rather than rebuild
// it — but it is a struct, not a byte slice, so the absolute binary rule does not apply.
type withholdStringer struct{ N int }

func (w withholdStringer) String() string { return "stringer:" + strconv.Itoa(w.N) }

// withholdBlob is a named byte-slice type that ALSO renders itself. The binary rule is absolute
// and must win over the self-rendering short-circuit.
type withholdBlob []byte

func (b withholdBlob) String() string { return string(b) }

// TestWithholdBinaryTraversesEveryKindRenderValueDoes is the traversal-parity pin named in
// withholdBinary's doc comment.
//
// ⚠ THE ASSERTION IS POSITIVE ON PURPOSE. Its predecessor asserted only that the secret STRING
// was absent from the output, and that is not the same claim: an untraversed byte array renders
// as `83, 69, 78, …`, which contains no secret substring and passes a NotContains test while
// leaking every byte. So each case asserts the withheld MARKER is present and that the numeric
// form is absent.
func TestWithholdBinaryTraversesEveryKindRenderValueDoes(t *testing.T) {
	t.Parallel()

	// The numeric rendering of the first two bytes, as renderValue would emit them for a
	// uint8-element sequence it walked instead of withholding.
	numeric := strconv.Itoa(int(binarySecret[0])) + ", " + strconv.Itoa(int(binarySecret[1]))

	var arr [len(binarySecret)]byte
	copy(arr[:], binarySecret)
	probe := withholdProbe{Name: "q3.pdf", Body: []byte(binarySecret)}

	cases := map[string]any{
		"a bare byte slice":                []byte(binarySecret),
		"a byte array":                     arr,
		"a byte slice in a list":           []any{[]byte(binarySecret)},
		"a byte slice in a map":            map[string]any{"body": []byte(binarySecret)},
		"a byte array in a map":            map[string]any{"digest": arr},
		"an exported struct field":         probe,
		"a pointer to a struct":            &probe,
		"a struct inside a list":           []any{probe},
		"a struct inside a map":            map[string]any{"upload": probe},
		"a list of structs":                []withholdProbe{probe, probe},
		"a self-rendering byte slice":      withholdBlob(binarySecret),
		"a byte slice behind two pointers": ptr(ptr([]byte(binarySecret))),
	}

	for name, val := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			out := renderValue(withholdBinary(val, 0), DefaultValueSeparator)

			assert.Contains(t, out, withheldBinaryValue,
				"the sweep did not reach the byte sequence — renderValue rendered it instead")
			assert.NotContains(t, out, binarySecret, "raw file bytes reached the output as text")
			assert.NotContains(t, out, numeric, "raw file bytes reached the output as numbers")
		})
	}
}

// TestWithholdBinaryLeavesOrdinaryValuesIntact is the other half of the guarantee: a sweep that
// redacted everything would also pass the test above.
func TestWithholdBinaryLeavesOrdinaryValuesIntact(t *testing.T) {
	t.Parallel()

	stamp := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

	cases := map[string]struct {
		val  any
		want string
	}{
		"a string is text the caller chose to bind": {"hello", "hello"},
		"a list of strings":                         {[]any{"a", "b"}, "a, b"},
		"an int":                                    {42, "42"},
		// time.Time is a struct of UNEXPORTED fields. Rebuilding it as a map — which the struct
		// arm does — would render every bound timestamp as the empty string, which is why the
		// self-rendering short-circuit is mandatory rather than tidy.
		"a timestamp survives the struct arm": {stamp, stamp.Format(time.RFC3339)},
		"a Stringer renders itself":           {withholdStringer{N: 7}, "stringer:7"},
		"an error renders itself":             {errors.New("boom"), "boom"},
		"a nil byte slice is not withheld":    {[]byte(nil), ""},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, renderValue(withholdBinary(tc.val, 0), DefaultValueSeparator))
		})
	}

	t.Run("an unexported byte field leaks through neither path", func(t *testing.T) {
		t.Parallel()
		out := renderValue(withholdBinary(withholdUnexported{Name: "f", body: []byte(binarySecret)}, 0),
			DefaultValueSeparator)
		assert.NotContains(t, out, binarySecret)
		assert.Contains(t, out, "f")
	})
}

// TestWithholdBinaryIsBoundedAndFailsClosed pins the depth bound's DIRECTION.
//
// Returning the value unswept at the bound was fail-open: this function flattens pointers, so
// the rebuilt structure can be shallower than the one swept, and a byte slice the sweep handed
// back raw at its bound was reached by renderValue well inside its own.
func TestWithholdBinaryIsBoundedAndFailsClosed(t *testing.T) {
	t.Parallel()

	t.Run("a byte slice behind more pointers than the bound never renders as text", func(t *testing.T) {
		t.Parallel()
		var deep any = []byte(binarySecret)
		for i := 0; i < inputMaxSweepDepth+4; i++ {
			deep = ptr(deep)
		}
		out := renderValue(withholdBinary(deep, 0), DefaultValueSeparator)
		assert.NotContains(t, out, binarySecret)
	})

	t.Run("a self-referential value terminates", func(t *testing.T) {
		t.Parallel()
		cyclic := map[string]any{"name": "loop"}
		cyclic["self"] = cyclic
		require.NotPanics(t, func() {
			_ = renderValue(withholdBinary(cyclic, 0), DefaultValueSeparator)
		})
	})
}

// =============================================================================
// fileManifestEntries — what it must NOT claim
// =============================================================================

// TestFileManifestEntriesDoesNotClaimOrdinaryLists is the regression pin for the recognizer's
// original rule, which was "every element is a map with a non-empty `name`".
//
// That does not describe a file. It describes a list of NAMED OBJECTS — one of the most ordinary
// shapes a caller can bind — and claiming it rendered a bullet list with every other key
// silently dropped from the prompt.
func TestFileManifestEntriesDoesNotClaimOrdinaryLists(t *testing.T) {
	t.Parallel()

	t.Run("a list of named objects is not a manifest", func(t *testing.T) {
		t.Parallel()
		_, ok := fileManifestEntries([]any{
			map[string]any{"name": "GPT-4", "provider": "openai"},
			map[string]any{"name": "Claude", "provider": "anthropic"},
		})
		assert.False(t, ok, "a foreign key on every element means the value is not a file list")
	})

	t.Run("a list of named objects keeps every key when rendered", func(t *testing.T) {
		t.Parallel()
		out := renderInputValue([]any{
			map[string]any{"name": "GPT-4", "provider": "openai"},
		}, "", false)
		assert.Contains(t, out, "provider", "the manifest dropped a key the caller bound")
		assert.Contains(t, out, "openai")
	})

	t.Run("bare named maps alone are too ambiguous to claim", func(t *testing.T) {
		t.Parallel()
		_, ok := fileManifestEntries([]any{
			map[string]any{"name": "a"}, map[string]any{"name": "b"},
		})
		assert.False(t, ok, "no file-specific key means this is as much a person list as a file list")
	})

	t.Run("a heterogeneous list is not a manifest", func(t *testing.T) {
		t.Parallel()
		_, ok := fileManifestEntries([]any{map[string]any{"name": "a"}, "plain"})
		assert.False(t, ok)
	})

	t.Run("an empty list is not a manifest", func(t *testing.T) {
		t.Parallel()
		_, ok := fileManifestEntries([]any{})
		assert.False(t, ok)
	})

	t.Run("a real upload list still is one", func(t *testing.T) {
		t.Parallel()
		entries, ok := fileManifestEntries([]any{
			map[string]any{"name": "q3.pdf", "mime_type": "application/pdf", "size_bytes": 1048576},
			map[string]any{"name": "notes.txt"},
		})
		require.True(t, ok, "one file-specific key on one element is enough to identify the list")
		assert.Equal(t, []string{"- q3.pdf (application/pdf, 1.0 MB)", "- notes.txt"}, entries)
	})

	t.Run("a file list carrying binary renders the manifest, never the bytes", func(t *testing.T) {
		t.Parallel()
		out := renderInputValue([]any{
			map[string]any{"name": "q3.pdf", "size_bytes": 12, "content": []byte(binarySecret)},
		}, "", false)
		assert.NotContains(t, out, binarySecret,
			"a `content` key takes the value out of the manifest vocabulary — it must still be swept")
		assert.Contains(t, out, withheldBinaryValue)
		assert.True(t, strings.Contains(out, "q3.pdf"))
	})
}

// ptr is a generic address-of helper, so the pointer cases above read as the shapes they test.
func ptr[T any](v T) *T { return &v }
