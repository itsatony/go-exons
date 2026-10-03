package exons

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// patchFixture carries everything a Parse→Serialize round trip damages: comments (head, line,
// trailing), a non-alphabetical key order, an unknown top-level key (Extensions), credentials, a
// templated value that Engine.Parse would RENDER, a flow list, blank lines, and a body with a
// leading horizontal rule and a tag.
const patchFixture = `---
# Churn analyst — owned by the data team.
name: churn-analyst
description: finds churn drivers
type: agent
x-team-note: keep this key   # unknown key, lands in Extensions

execution:
  provider: anthropic
  model: '{~exons.env name="CHURN_MODEL" default="claude-sonnet-5" /~}'  # templated, never rendered
  temperature: 0.2

credentials:
  warehouse:
    provider: snowflake
    ref: vault://data/warehouse
credential: warehouse

tools:
  # the analyst only queries
  allow: [sql_query, chart]
  tool_choice: auto

requirements:
  resources:
    - ref: churn-docs
      kind: corpus
# trailing comment
---
---
Body keeps {~exons.var name="x" /~} and the rule above, byte for byte.
`

func mustPatch(t *testing.T, src string, edits ...SourceEdit) string {
	t.Helper()
	out, err := PatchSource([]byte(src), edits...)
	require.NoError(t, err)
	return string(out)
}

// TestPatchSource_GoldenEdits pins each edit as an exact expected document: every byte outside the
// edited entry is the fixture's.
func TestPatchSource_GoldenEdits(t *testing.T) {
	replace := func(old, new string) string {
		require.Equal(t, 1, strings.Count(patchFixture, old), "fixture anchor %q", old)
		return strings.Replace(patchFixture, old, new, 1)
	}
	cases := []struct {
		name  string
		edits []SourceEdit
		want  string
	}{
		{
			"tools.allow listed → other list",
			[]SourceEdit{SetToolsAllow([]string{"sql_query"})},
			replace("  allow: [sql_query, chart]\n", "  allow:\n    - sql_query\n"),
		},
		{
			"tools.allow → none ([])",
			[]SourceEdit{SetToolsAllow([]string{})},
			replace("  allow: [sql_query, chart]\n", "  allow: []\n"),
		},
		{
			"tools.allow → all (key removed, block kept: it has tool_choice)",
			[]SourceEdit{SetToolsAllow(nil)},
			replace("  allow: [sql_query, chart]\n", ""),
		},
		{
			"execution.model replaces the templated value, keeps the line comment",
			[]SourceEdit{SetExecutionModel("gpt-5")},
			replace(`  model: '{~exons.env name="CHURN_MODEL" default="claude-sonnet-5" /~}'  # templated, never rendered`+"\n",
				"  model: gpt-5 # templated, never rendered\n"),
		},
		{
			"execution.provider untouched model stays templated",
			[]SourceEdit{SetExecutionProvider("openai")},
			replace("  provider: anthropic\n", "  provider: openai\n"),
		},
		{
			"execution.reasoning_effort inserted at the end of the execution block",
			[]SourceEdit{SetExecutionReasoningEffort("high")},
			replace("  temperature: 0.2\n", "  temperature: 0.2\n  reasoning_effort: high\n"),
		},
		{
			"display_name inserted at the end of the root, before the trailing comment",
			[]SourceEdit{SetDisplayName("Churn Analyst")},
			replace("      kind: corpus\n# trailing comment\n", "      kind: corpus\ndisplay_name: Churn Analyst\n# trailing comment\n"),
		},
		{
			"resource_modes inserted into an existing requirements block",
			[]SourceEdit{SetResourceMode("mcp_server", ResourceModeNone)},
			replace("      kind: corpus\n# trailing", "      kind: corpus\n  resource_modes:\n    mcp_server: none\n# trailing"),
		},
		{
			// The intermediate state (none beside a corpus entry) is invalid; only the result is judged.
			// Mode FIRST keeps the requirements block in place: removing its only key first would
			// prune the block and the mode would be re-created at the end of the root.
			"corpus mode set to none and its entries removed in one patch",
			[]SourceEdit{SetResourceMode("corpus", ResourceModeNone), SetRequirementsResources(nil)},
			replace("requirements:\n  resources:\n    - ref: churn-docs\n      kind: corpus\n", "requirements:\n  resource_modes:\n    corpus: none\n"),
		},
		{
			"skills inserted",
			[]SourceEdit{SetSkills([]SkillRef{{Slug: "@acme/sql-helper", Version: "2"}})},
			replace("      kind: corpus\n# trailing", "      kind: corpus\nskills:\n  - slug: '@acme/sql-helper'\n    version: \"2\"\n# trailing"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mustPatch(t, patchFixture, tc.edits...)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestPatchSource_PreservesWhatARoundTripLoses is the reason the function exists, stated as the
// three losses: credentials, comments, templated values — next to proof that Serialize loses them.
func TestPatchSource_PreservesWhatARoundTripLoses(t *testing.T) {
	spec, err := Parse([]byte(patchFixture))
	require.NoError(t, err)
	spec.Tools.Allow = []string{"sql_query"}
	roundTrip, err := spec.ExportFull()
	require.NoError(t, err)
	assert.NotContains(t, string(roundTrip), "vault://data/warehouse", "ExportFull drops credentials — the damage PatchSource avoids")
	assert.NotContains(t, string(roundTrip), "# the analyst only queries", "and comments")

	got := mustPatch(t, patchFixture, SetToolsAllow([]string{"sql_query"}))
	for _, keep := range []string{
		"vault://data/warehouse", "credential: warehouse",
		"# Churn analyst — owned by the data team.", "# the analyst only queries", "# trailing comment",
		"x-team-note: keep this key   # unknown key, lands in Extensions",
		`'{~exons.env name="CHURN_MODEL" default="claude-sonnet-5" /~}'`,
		"---\nBody keeps {~exons.var name=\"x\" /~} and the rule above, byte for byte.\n",
	} {
		assert.Contains(t, got, keep)
	}
	reparsed, err := Parse([]byte(got))
	require.NoError(t, err)
	assert.Equal(t, []string{"sql_query"}, reparsed.Tools.Allow)
	assert.Equal(t, "auto", reparsed.Tools.ToolChoice)
	assert.Equal(t, spec.Body, reparsed.Body)
}

func TestPatchSource_PrunesAnEmptiedBlock(t *testing.T) {
	src := "---\nname: a\ndescription: d\ntype: agent\ntools:\n  allow: [x]\nexecution:\n  model: m\n---\nbody\n"
	got := mustPatch(t, src, SetToolsAllow(nil), SetExecutionModel(""))
	assert.Equal(t, "---\nname: a\ndescription: d\ntype: agent\n---\nbody\n", got)

	// Two levels: the only mode in the only requirements key.
	src = "---\nname: a\ndescription: d\ntype: agent\nrequirements:\n  resource_modes:\n    corpus: none\n---\nbody\n"
	got = mustPatch(t, src, SetResourceMode("corpus", ""))
	assert.Equal(t, "---\nname: a\ndescription: d\ntype: agent\n---\nbody\n", got)

	// A block re-created after an earlier edit emptied it.
	src = "---\nname: a\ndescription: d\ntype: agent\ntools:\n  allow: [x]\n---\nbody\n"
	got = mustPatch(t, src, SetToolsAllow(nil), SetToolsAllow([]string{}))
	assert.Equal(t, "---\nname: a\ndescription: d\ntype: agent\ntools:\n  allow: []\n---\nbody\n", got)
}

func TestPatchSource_Shapes(t *testing.T) {
	head := "---\nname: a\ndescription: d\ntype: agent\n"
	cases := []struct {
		name, fm string
		edit     SourceEdit
		want     string
	}{
		{"compact block sequence", "tools:\n  allow:\n  - a\n  - b\n  tool_choice: auto\n", SetToolsAllow([]string{"c"}), "tools:\n  allow:\n    - c\n  tool_choice: auto\n"},
		{"null parent", "tools:\n", SetToolsAllow([]string{"c"}), "tools:\n  allow:\n    - c\n"},
		{"flow parent stays flow", "tools: {allow: [a], tool_choice: auto}\n", SetToolsAllow([]string{}), "tools: {allow: [], tool_choice: auto}\n"},
		{"four-space indentation is the block's own", "execution:\n    model: m\n", SetExecutionProvider("p"), "execution:\n    model: m\n    provider: p\n"},
		{"remove an absent key is a no-op", "", SetDisplayName(""), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mustPatch(t, head+tc.fm+"---\nbody\n", tc.edit)
			assert.Equal(t, head+tc.want+"---\nbody\n", got)
		})
	}

	t.Run("CRLF document stays CRLF", func(t *testing.T) {
		src := "---\r\nname: a\r\ndescription: d\r\ntype: agent\r\n---\r\nbody\r\n"
		got := mustPatch(t, src, SetDisplayName("A"))
		assert.Equal(t, "---\r\nname: a\r\ndescription: d\r\ntype: agent\r\ndisplay_name: A\r\n---\r\nbody\r\n", got)
	})
}

func TestPatchSource_Refusals(t *testing.T) {
	agentDoc := "---\nname: a\ndescription: d\ntype: agent\n---\nbody\n"
	cases := []struct {
		name   string
		src    string
		edits  []SourceEdit
		reason error
	}{
		{"no frontmatter", "just a body", []SourceEdit{SetDisplayName("x")}, ErrPatchNoFrontmatter},
		{"source does not parse", "---\nname: a\n---\nbody\n", []SourceEdit{SetDisplayName("x")}, ErrPatchSourceInvalid},
		{"unquoted template is YAML only after rendering", "---\nname: a\ndescription: d\nexecution:\n  model: {~exons.env name=\"M\" /~}\n---\n", []SourceEdit{SetDisplayName("x")}, ErrPatchSourceInvalid},
		{"zero edit", agentDoc, []SourceEdit{{}}, ErrPatchEditInvalid},
		{"bad resource kind", agentDoc, []SourceEdit{SetResourceMode("Corpus", ResourceModeNone)}, ErrPatchEditInvalid},
		{"bad resource mode", agentDoc, []SourceEdit{SetResourceMode("corpus", "some")}, ErrPatchEditInvalid},
		{"result invalid: none beside entries", "---\nname: a\ndescription: d\ntype: agent\nrequirements:\n  resources:\n    - ref: r\n      kind: corpus\n---\n", []SourceEdit{SetResourceMode("corpus", ResourceModeNone)}, ErrPatchResultInvalid},
		{"result invalid: resource_modes on a prompt", "---\nname: a\ndescription: d\ntype: prompt\n---\n", []SourceEdit{SetResourceMode("corpus", ResourceModeNone)}, ErrPatchResultInvalid},
		{"result invalid: duplicate tool", agentDoc, []SourceEdit{SetToolsAllow([]string{"a", "a"})}, ErrPatchResultInvalid},
		{"anchor in the edited entry", "---\nname: a\ndescription: d\ntype: agent\ntools:\n  allow: &t [a]\n---\n", []SourceEdit{SetToolsAllow([]string{"b"})}, ErrPatchUnsupportedShape},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := PatchSource([]byte(tc.src), tc.edits...)
			require.Error(t, err)
			assert.Nil(t, out)
			assert.True(t, errors.Is(err, ErrPatchRefused), "matches ErrPatchRefused: %v", err)
			assert.True(t, errors.Is(err, tc.reason), "matches %v: %v", tc.reason, err)
			var pe *PatchError
			require.True(t, errors.As(err, &pe))
		})
	}

	t.Run("result invalid keeps the validation cause", func(t *testing.T) {
		_, err := PatchSource([]byte(agentDoc), SetToolsAllow([]string{"a", "a"}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgToolAllowEntryDup)
	})
}

// TestPatchSource_SelfCheckRefusesAWrongSplice proves the self-check is live: an edit whose Go
// twin disagrees with its text edit is refused, not shipped.
func TestPatchSource_SelfCheckRefusesAWrongSplice(t *testing.T) {
	lying := SetDisplayName("Shown")
	lying.apply = func(s *Spec) { s.DisplayName = "Something else" }
	_, err := PatchSource([]byte("---\nname: a\ndescription: d\ntype: agent\n---\n"), lying)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrPatchSelfCheck), "%v", err)
}

// An unquoted tag in a key the parser does not type (an extension) is YAML as written — a flow
// mapping — so the document Parses, and PatchSource carries it through untouched and unrendered.
func TestPatchSource_UnquotedTemplateInAnExtensionIsCarried(t *testing.T) {
	src := "---\nname: a\ndescription: d\ntype: agent\nx-model: {~exons.env name=\"M\" /~}\n---\nbody\n"
	got := mustPatch(t, src, SetDisplayName("A"))
	assert.Equal(t, "---\nname: a\ndescription: d\ntype: agent\nx-model: {~exons.env name=\"M\" /~}\ndisplay_name: A\n---\nbody\n", got)
}

func TestPatchSource_NoEditsIsIdentity(t *testing.T) {
	assert.Equal(t, patchFixture, mustPatch(t, patchFixture))
}
