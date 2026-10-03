package exons

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requireRefused asserts a PatchSource refusal with the given reason and no bytes.
func requireRefused(t *testing.T, out []byte, err error, reason error) {
	t.Helper()
	require.Error(t, err)
	assert.Nil(t, out)
	assert.True(t, errors.Is(err, ErrPatchRefused), "matches ErrPatchRefused: %v", err)
	assert.True(t, errors.Is(err, reason), "matches %v: %v", reason, err)
}

// H1: yaml.v3 breaks lines on U+0085, U+2028, U+2029 and a lone "\r"; a "\n" split does not, so
// node line numbers and text lines disagreed and slicing panicked. Refused up front now.
func TestPatchSource_ReviewH1_ForeignLineBreaksAreRefusedNotPanicked(t *testing.T) {
	cases := map[string]string{
		"U+2028 in a comment":        "---\n# a  # b\nname: x\ndescription: dd\n---\nbody\n",
		"U+2029, U+0085, lone CR":    "---\n# a  # b\n# c\u0085# d\n# e\r# f\nname: x\ndescription: dd\n---\nbody\n",
		"U+0085 in a quoted value":   "---\nname: x\ndescription: \"d\u0085d\"\n---\nbody\n",
		"U+2028 in a trailing block": "---\nname: x\ndescription: dd\ndisplay_name: a\n# x # y # z\n---\nbody\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			var out []byte
			var err error
			require.NotPanics(t, func() { out, err = PatchSource([]byte(src), SetDisplayName("N")) })
			requireRefused(t, out, err, ErrPatchUnsupportedShape)
		})
	}

	// The reviewer's literal reproducer (a "#" inside a comment line) is ordinary text and patches.
	got := mustPatch(t, "---\n# a # b\nname: x\ndescription: dd\n---\nbody\n", SetDisplayName("New"))
	assert.Equal(t, "---\n# a # b\nname: x\ndescription: dd\ndisplay_name: New\n---\nbody\n", got)
}

func TestPatchSource_ReviewH1_LineRangesAreBoundsChecked(t *testing.T) {
	f := &fmLines{lines: []string{"a", "b"}}
	for _, r := range [][2]int{{0, 1}, {2, 3}, {3, 1}, {-1, -1}} {
		err := f.replace(r[0], r[1], []string{"x"})
		require.Error(t, err, "range %v", r)
		assert.True(t, errors.Is(err, ErrPatchInternal))
	}
	require.NoError(t, f.insertAfter(2, []string{"c"}))
	require.NoError(t, f.insertAfter(0, []string{"z"}))
	assert.Equal(t, []string{"z", "a", "b", "c"}, f.lines)
}

func TestPatchSource_ReviewH1_APanicBecomesARefusal(t *testing.T) {
	boom := SetDisplayName("x")
	boom.apply = func(*Spec) { panic("boom") }
	var out []byte
	var err error
	require.NotPanics(t, func() {
		out, err = PatchSource([]byte("---\nname: a\ndescription: d\n---\n"), boom)
	})
	requireRefused(t, out, err, ErrPatchInternal)
}

// H2: a "#" line inside a block scalar is CONTENT; it must leave with its value.
func TestPatchSource_ReviewH2_BlockScalarHashLinesLeaveWithTheValue(t *testing.T) {
	head := "---\nname: a\ndescription: d\ntype: agent\n"
	tail := "x-k: 1\n---\nbody\n"
	cases := []struct {
		name, value string
		edit        SourceEdit
		want        string
	}{
		{"literal, replaced", "display_name: |\n  Team Bot\n  # internal codename: falcon\n", SetDisplayName("New"), "display_name: New\n"},
		{"literal, removed", "display_name: |\n  Team Bot\n  # internal codename: falcon\n", SetDisplayName(""), ""},
		{"folded with a blank line, replaced", "display_name: >\n  Team\n\n  # falcon\n", SetDisplayName("New"), "display_name: New\n"},
		{"nested literal, replaced", "execution:\n  model: |\n    m\n    # falcon\n  provider: p\n", SetExecutionModel("m2"), "execution:\n  model: m2\n  provider: p\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mustPatch(t, head+tc.value+tail, tc.edit)
			assert.Equal(t, head+tc.want+tail, got)
			assert.NotContains(t, got, "falcon")
		})
	}
}

// The self-check's comment arm, on its own: a result carrying a comment the source did not have.
func TestPatchSource_ReviewH2_NoNewCommentsCatchesResidue(t *testing.T) {
	source := []byte("display_name: |\n  Team Bot\n  # internal codename: falcon\nx-k: 1 # kept")
	residue := []byte("display_name: New\n  # internal codename: falcon\nx-k: 1 # kept")
	err := noNewComments(source, residue)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgPatchNewComment)

	require.NoError(t, noNewComments(source, []byte("display_name: New\nx-k: 1 # kept")), "dropping is fine")
	require.NoError(t, noNewComments([]byte("a: 1 # c\nb: 2 # c"), []byte("a: 1 # c\nb: 3 # c")), "a multiset, not a set")
	require.Error(t, noNewComments([]byte("a: 1 # c"), []byte("a: 1 # c\nb: 3 # c")), "one more copy is new")
}

// H3: Parse is tolerant; PatchSource runs the strict check on its result, so a patch repairs.
func TestPatchSource_ReviewH3_APatchRepairsAToleratedDocument(t *testing.T) {
	src := "---\nname: a\ndescription: d\ntype: agent\ntools:\n  allow: [a, a]\n---\nbody\n"
	_, err := Parse([]byte(src))
	require.NoError(t, err, "Parse tolerates the duplicate")

	got := mustPatch(t, src, SetToolsAllow([]string{"a"}))
	assert.Equal(t, "---\nname: a\ndescription: d\ntype: agent\ntools:\n  allow:\n    - a\n---\nbody\n", got)

	out, err := PatchSource([]byte(src), SetDisplayName("A"))
	requireRefused(t, out, err, ErrPatchResultInvalid)
	assert.Contains(t, err.Error(), ErrMsgToolAllowEntryDup, "names what must be repaired")
}

// M3: the source refusal is neutral and carries the parse error; an intermediate failure is ours.
func TestPatchSource_ReviewM3_SourceRefusalIsNeutral(t *testing.T) {
	out, err := PatchSource([]byte("---\nname: a\n---\n"), SetDisplayName("x"))
	requireRefused(t, out, err, ErrPatchSourceInvalid)
	assert.Contains(t, err.Error(), ErrMsgPatchSourceInvalid)
	assert.Contains(t, err.Error(), ErrMsgSpecDescriptionRequired, "wraps the cause")
	assert.False(t, errors.Is(err, ErrPatchInternal))
}

// L2: inserted lines follow the DOMINANT line ending.
func TestPatchSource_ReviewL2_DominantLineEnding(t *testing.T) {
	src := "---\nname: a\r\ndescription: d\ntype: agent\n---\nbody\n" // one CRLF among LFs
	got := mustPatch(t, src, SetDisplayName("A"))
	assert.Equal(t, "---\nname: a\r\ndescription: d\ntype: agent\ndisplay_name: A\n---\nbody\n", got)
}

// L3: the key-line comment survives even when the new value contains its text.
func TestPatchSource_ReviewL3_KeyLineCommentIsNotDroppedByAValueThatContainsIt(t *testing.T) {
	src := "---\nname: a\ndescription: d\ntype: agent\ndisplay_name: a # legacy\n---\nbody\n"
	got := mustPatch(t, src, SetDisplayName("# legacy"))
	assert.Equal(t, "---\nname: a\ndescription: d\ntype: agent\ndisplay_name: '# legacy' # legacy\n---\nbody\n", got)
}

// L4: a key that reads as a YAML 1.1 boolean is quoted.
func TestPatchSource_ReviewL4_BooleanLookingKeysAreQuoted(t *testing.T) {
	src := "---\nname: a\ndescription: d\ntype: agent\n---\nbody\n"
	got := mustPatch(t, src, SetResourceMode("no", ResourceModeNone), SetResourceMode("on", ResourceModeAll))
	assert.Contains(t, got, "\n    \"no\": none\n")
	assert.Contains(t, got, "\n    \"on\": all\n")
	spec, err := Parse([]byte(got))
	require.NoError(t, err)
	assert.Equal(t, ResourceModeNone, spec.Requirements.ResourceMode("no"))
}

// L6: one call accepts at most MaxPatchEdits edits.
func TestPatchSource_ReviewL6_EditCountIsBounded(t *testing.T) {
	src := []byte("---\nname: a\ndescription: d\ntype: agent\n---\n")
	edits := make([]SourceEdit, MaxPatchEdits+1)
	for i := range edits {
		edits[i] = SetDisplayName("A")
	}
	out, err := PatchSource(src, edits...)
	requireRefused(t, out, err, ErrPatchEditInvalid)
	_, err = PatchSource(src, edits[:MaxPatchEdits]...)
	require.NoError(t, err)
	assert.True(t, strings.Contains(ErrMsgPatchTooManyEdits, "MaxPatchEdits"))
}

// The comment arm wired into PatchSource: a splice that leaves value text behind as a comment is
// refused even when everything else about the result checks out.
func TestPatchSource_ReviewH2_SelfCheckRefusesLeftoverComment(t *testing.T) {
	orig := patchOneEdit
	t.Cleanup(func() { patchOneEdit = orig })
	patchOneEdit = func(fm []byte, e SourceEdit) ([]byte, []string, error) {
		out, pruned, err := orig(fm, e)
		return append(out, []byte("\n# internal codename: falcon")...), pruned, err
	}
	out, err := PatchSource([]byte("---\nname: a\ndescription: d\ntype: agent\n---\n"), SetDisplayName("A"))
	requireRefused(t, out, err, ErrPatchSelfCheck)
	assert.Contains(t, err.Error(), ErrMsgPatchNewComment)
}

// M3: a frontmatter an EARLIER edit produced that no longer parses is our fault, not the source's.
func TestPatchSource_ReviewM3_IntermediateFailureIsInternal(t *testing.T) {
	_, _, err := patchFrontmatter([]byte("name: [unclosed"), SetDisplayName("A"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrPatchInternal), "%v", err)
	assert.False(t, errors.Is(err, ErrPatchSourceInvalid))
}

// Re-review MEDIUM: a quoted scalar's continuation line may sit at any indentation and start with
// "#", so it can pass as a comment and balance the comment self-check. An edited entry holding a
// multi-line quoted scalar is refused; one elsewhere in the document is left alone.
func TestPatchSource_ReReview_MultiLineQuotedScalarInAnEditedEntryIsRefused(t *testing.T) {
	head := "---\nname: a\ndescription: d\ntype: agent\n"
	reproducer := head + "tools:\n  allow:\n    # x\"\n    - \"a\n# x\"\nx-k: 1\n---\nbody\n"
	_, err := Parse([]byte(reproducer))
	require.NoError(t, err, "the reproducer is a valid document")
	out, err := PatchSource([]byte(reproducer), SetToolsAllow([]string{"z"}))
	requireRefused(t, out, err, ErrPatchUnsupportedShape)
	assert.Contains(t, err.Error(), ErrMsgPatchMultiLineQuoted)

	// (A multi-line quoted value was an H2 golden case; since the re-review it is refused instead.)
	double := head + "display_name: \"Team\n  # falcon\"\nx-k: 1\n---\nbody\n"
	out, err = PatchSource([]byte(double), SetDisplayName(""))
	requireRefused(t, out, err, ErrPatchUnsupportedShape)

	single := head + "display_name: 'Team\n  # falcon'\nx-k: 1\n---\nbody\n"
	out, err = PatchSource([]byte(single), SetDisplayName(""))
	requireRefused(t, out, err, ErrPatchUnsupportedShape)

	// Escapes keep a one-line scalar one line: \" in double quotes, '' in single quotes.
	oneLine := head + "display_name: \"Te\\\"am\"\nx-a: 'it''s'\nx-k: 1\n---\nbody\n"
	assert.Equal(t, head+"display_name: New\nx-a: 'it''s'\nx-k: 1\n---\nbody\n", mustPatch(t, oneLine, SetDisplayName("New")))

	// A multi-line quoted scalar OUTSIDE the edited entries is untouched and does not block.
	elsewhere := head + "x-note: \"two\n  lines\"\nexecution:\n  model: m\n---\nbody\n"
	assert.Equal(t, head+"x-note: \"two\n  lines\"\nexecution:\n  model: n2\n---\nbody\n", mustPatch(t, elsewhere, SetExecutionModel("n2")))
}
