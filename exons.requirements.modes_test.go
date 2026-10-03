package exons

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func modesDoc(typ, requirements string) []byte {
	return []byte("---\nname: doc\ndescription: a document\ntype: " + typ + "\n" + requirements + "---\nbody\n")
}

func TestResourceMode_DeclaredAndDerived(t *testing.T) {
	spec, err := Parse(modesDoc("agent", `requirements:
  resources:
    - ref: docs
      kind: corpus
  resource_modes:
    mcp_server: none
    corpus: listed
`))
	require.NoError(t, err)
	r := spec.Requirements
	assert.Equal(t, ResourceModeNone, r.ResourceMode("mcp_server"), "declared none")
	assert.Equal(t, ResourceModeListed, r.ResourceMode("corpus"), "declared listed")
	assert.Equal(t, ResourceModeAll, r.ResourceMode("folder"), "absent kind, no entries → all")

	// A document written before v0.40.0: no resource_modes key at all.
	old, err := Parse(modesDoc("agent", "requirements:\n  resources:\n    - ref: docs\n      kind: corpus\n"))
	require.NoError(t, err)
	assert.Equal(t, ResourceModeListed, old.Requirements.ResourceMode("corpus"), "entries present → listed")
	assert.Equal(t, ResourceModeAll, old.Requirements.ResourceMode("toolset"), "no entries → all")

	var nilReqs *SpecRequirements
	assert.Equal(t, ResourceModeAll, nilReqs.ResourceMode("corpus"), "nil requirements → all")
}

func TestResourceModes_Validation(t *testing.T) {
	cases := []struct {
		name    string
		typ     string
		reqs    string
		wantErr string
	}{
		{"none with no entries", "agent", "requirements:\n  resource_modes:\n    corpus: none\n", ""},
		{"all with no entries", "skill", "requirements:\n  resource_modes:\n    corpus: all\n", ""},
		{"listed with an entry", "agent", "requirements:\n  resources:\n    - ref: d\n      kind: corpus\n  resource_modes:\n    corpus: listed\n", ""},
		{"none beside entries of the kind", "agent", "requirements:\n  resources:\n    - ref: d\n      kind: corpus\n  resource_modes:\n    corpus: none\n", ErrMsgResourceModeWithEntries},
		{"all beside entries of the kind", "agent", "requirements:\n  resources:\n    - ref: d\n      kind: corpus\n  resource_modes:\n    corpus: all\n", ErrMsgResourceModeWithEntries},
		{"none beside entries of another kind", "agent", "requirements:\n  resources:\n    - ref: d\n      kind: corpus\n  resource_modes:\n    folder: none\n", ""},
		{"listed with zero entries", "agent", "requirements:\n  resource_modes:\n    corpus: listed\n", ErrMsgResourceModeListedEmpty},
		{"listed with entries only of another kind", "agent", "requirements:\n  resources:\n    - ref: d\n      kind: folder\n  resource_modes:\n    corpus: listed\n", ErrMsgResourceModeListedEmpty},
		{"value out of vocabulary", "agent", "requirements:\n  resource_modes:\n    corpus: some\n", ErrMsgResourceModeInvalid},
		{"value empty", "agent", "requirements:\n  resource_modes:\n    corpus: \"\"\n", ErrMsgResourceModeInvalid},
		{"kind uppercase", "agent", "requirements:\n  resource_modes:\n    Corpus: none\n", ErrMsgResourceModeKindForm},
		{"kind is a coordinate", "agent", "requirements:\n  resource_modes:\n    \"s3://x\": none\n", ErrMsgResourceModeKindForm},
		{"prompt refuses it", "prompt", "requirements:\n  resource_modes:\n    corpus: none\n", ErrMsgPromptNoResourceModes},
		{"prompt refuses an empty mapping", "prompt", "requirements:\n  resource_modes: {}\n", ErrMsgPromptNoResourceModes},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(modesDoc(tc.typ, tc.reqs))
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestResourceModes_TooManyKeys(t *testing.T) {
	r := &SpecRequirements{ResourceModes: map[string]ResourceMode{}}
	for i := 0; i <= MaxRequirementEntries; i++ {
		r.ResourceModes["k"+strconv.Itoa(i)] = ResourceModeNone
	}
	err := r.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgRequirementTooManyEntries)
	delete(r.ResourceModes, "k0")
	require.NoError(t, r.Validate(), "exactly MaxRequirementEntries keys is valid")
}

func TestResourceModes_RoundTripAndClone(t *testing.T) {
	src := modesDoc("agent", "requirements:\n  resource_modes:\n    corpus: none\n    mcp_server: all\n")
	spec, err := Parse(src)
	require.NoError(t, err)

	out, err := spec.ExportFull()
	require.NoError(t, err)
	again, err := Parse(out)
	require.NoError(t, err)
	assert.Equal(t, spec.Requirements.ResourceModes, again.Requirements.ResourceModes, "ExportFull keeps resource_modes")

	clone := spec.Clone()
	clone.Requirements.ResourceModes["corpus"] = ResourceModeAll
	assert.Equal(t, ResourceModeNone, spec.Requirements.ResourceModes["corpus"], "Clone deep-copies the map")
}

func TestToolMode_DerivedFromAllow(t *testing.T) {
	var nilSpec *Spec
	assert.Equal(t, ResourceModeAll, nilSpec.ToolMode())
	assert.Equal(t, ResourceModeAll, (&Spec{}).ToolMode(), "no tools block")
	assert.Equal(t, ResourceModeAll, (&Spec{Tools: &ToolsConfig{}}).ToolMode(), "tools block, no allow")
	assert.Equal(t, ResourceModeNone, (&Spec{Tools: &ToolsConfig{Allow: []string{}}}).ToolMode(), "allow: []")
	assert.Equal(t, ResourceModeListed, (&Spec{Tools: &ToolsConfig{Allow: []string{"a"}}}).ToolMode(), "allow: [a]")

	// And through the parser, where the nil-vs-empty distinction is made.
	none, err := Parse(modesDoc("agent", "tools:\n  allow: []\n"))
	require.NoError(t, err)
	assert.Equal(t, ResourceModeNone, none.ToolMode())
}

func TestToolsConfig_ValidateAllowLists(t *testing.T) {
	many := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = "t" + strconv.Itoa(i)
		}
		return out
	}
	cases := []struct {
		name    string
		tc      *ToolsConfig
		wantErr string
	}{
		{"nil", nil, ""},
		{"absent allow", &ToolsConfig{}, ""},
		{"empty allow", &ToolsConfig{Allow: []string{}}, ""},
		{"populated allow", &ToolsConfig{Allow: []string{"a", "b"}}, ""},
		{"case differs is not a duplicate", &ToolsConfig{Allow: []string{"a", "A"}}, ""},
		{"empty entry", &ToolsConfig{Allow: []string{"a", ""}}, ErrMsgToolAllowEntryEmpty},
		{"duplicate", &ToolsConfig{Allow: []string{"a", "a"}}, ErrMsgToolAllowEntryDup},
		{"at the bound", &ToolsConfig{Allow: many(MaxToolAllowEntries)}, ""},
		{"over the bound", &ToolsConfig{Allow: many(MaxToolAllowEntries + 1)}, ErrMsgToolAllowTooMany},
		{"server tools duplicate", &ToolsConfig{MCPServers: []*MCPServer{{Name: "s", URL: "u", Tools: []string{"x", "x"}}}}, ErrMsgToolAllowEntryDup},
		{"server tools empty entry", &ToolsConfig{MCPServers: []*MCPServer{{Name: "s", URL: "u", Tools: []string{""}}}}, ErrMsgToolAllowEntryEmpty},
		{"server tools over the bound", &ToolsConfig{MCPServers: []*MCPServer{{Name: "s", URL: "u", Tools: many(MaxToolAllowEntries + 1)}}}, ErrMsgToolAllowTooMany},
		{"server tools empty list", &ToolsConfig{MCPServers: []*MCPServer{{Name: "s", URL: "u", Tools: []string{}}}}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.tc.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}

	// Spec.Validate runs it: the duplicate is refused at Parse.
	_, err := Parse(modesDoc("agent", "tools:\n  allow: [a, a]\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgToolAllowEntryDup)
}
