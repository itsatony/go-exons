package exons

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequirementsValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     *SpecRequirements
		wantErr bool
	}{
		{"nil is valid", nil, false},
		{"empty is valid", &SpecRequirements{}, false},
		{
			"valid mcp + credentials",
			&SpecRequirements{
				MCP:         []MCPRequirement{{Capability: "dns-management", CredentialRef: "cloudflare-api", Scope: RequirementScopeOrg}},
				Credentials: []CredentialRequirement{{Ref: "slack-bot", Provider: "slack", Scope: RequirementScopeUser}},
			},
			false,
		},
		{"empty scope allowed", &SpecRequirements{MCP: []MCPRequirement{{Capability: "x"}}}, false},
		{"per_call scope", &SpecRequirements{Credentials: []CredentialRequirement{{Ref: "r", Scope: RequirementScopePerCall}}}, false},
		{"missing capability", &SpecRequirements{MCP: []MCPRequirement{{Capability: ""}}}, true},
		{"duplicate capability", &SpecRequirements{MCP: []MCPRequirement{{Capability: "a"}, {Capability: "a"}}}, true},
		{"invalid mcp scope", &SpecRequirements{MCP: []MCPRequirement{{Capability: "a", Scope: "global"}}}, true},
		{"missing ref", &SpecRequirements{Credentials: []CredentialRequirement{{Ref: ""}}}, true},
		{"duplicate ref", &SpecRequirements{Credentials: []CredentialRequirement{{Ref: "r"}, {Ref: "r"}}}, true},
		{"invalid cred scope", &SpecRequirements{Credentials: []CredentialRequirement{{Ref: "r", Scope: "team"}}}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRequirementsBounds(t *testing.T) {
	t.Run("too many mcp entries", func(t *testing.T) {
		mcp := make([]MCPRequirement, MaxRequirementEntries+1)
		for i := range mcp {
			mcp[i] = MCPRequirement{Capability: "cap-" + string(rune('a'+i%26)) + "-" + string(rune('a'+(i/26)%26)) + "-x"}
		}
		// Distinct capabilities avoid the duplicate check; the count cap fires.
		err := (&SpecRequirements{MCP: mcp[:MaxRequirementEntries+1]}).Validate()
		assert.Error(t, err)
	})
	t.Run("field too long", func(t *testing.T) {
		long := make([]byte, MaxRequirementFieldLen+1)
		for i := range long {
			long[i] = 'x'
		}
		err := (&SpecRequirements{MCP: []MCPRequirement{{Capability: string(long)}}}).Validate()
		assert.Error(t, err)
	})
}

func TestSpecValidateRequirements(t *testing.T) {
	t.Run("nil spec", func(t *testing.T) {
		var s *Spec
		assert.NoError(t, s.ValidateRequirements())
	})
	t.Run("spec without requirements", func(t *testing.T) {
		s := &Spec{Name: "x", Description: "d"}
		assert.NoError(t, s.ValidateRequirements())
	})
	t.Run("spec with invalid requirements", func(t *testing.T) {
		s := &Spec{Requirements: &SpecRequirements{MCP: []MCPRequirement{{Capability: ""}}}}
		assert.Error(t, s.ValidateRequirements())
	})
}

func TestRequirementsParsedFromYAML(t *testing.T) {
	yamlData := `
name: dns-agent
description: manages DNS
type: agent
requirements:
  mcp:
    - capability: dns-management
      credential_ref: cloudflare-api
      scope: org
  credentials:
    - ref: slack-bot
      provider: slack
      scope: user
`
	spec, err := ParseYAMLSpec(yamlData)
	require.NoError(t, err)
	require.NotNil(t, spec.Requirements)
	require.Len(t, spec.Requirements.MCP, 1)
	assert.Equal(t, "dns-management", spec.Requirements.MCP[0].Capability)
	assert.Equal(t, "cloudflare-api", spec.Requirements.MCP[0].CredentialRef)
	assert.Equal(t, RequirementScopeOrg, spec.Requirements.MCP[0].Scope)
	require.Len(t, spec.Requirements.Credentials, 1)
	assert.Equal(t, "slack-bot", spec.Requirements.Credentials[0].Ref)
	assert.NoError(t, spec.ValidateRequirements())
}

func TestRequirementsClone(t *testing.T) {
	orig := &SpecRequirements{
		MCP:         []MCPRequirement{{Capability: "a", Scope: RequirementScopeOrg}},
		Credentials: []CredentialRequirement{{Ref: "r"}},
	}
	clone := orig.Clone()
	clone.MCP[0].Capability = "b"
	assert.Equal(t, "a", orig.MCP[0].Capability, "clone must be independent")
}

func TestResourceRequirementsValidate(t *testing.T) {
	res := func(rs ...ResourceRequirement) *SpecRequirements { return &SpecRequirements{Resources: rs} }
	tests := []struct {
		name    string
		req     *SpecRequirements
		wantMsg string // empty = valid
	}{
		{"minimal", res(ResourceRequirement{Ref: "product-docs", Kind: "corpus"}), ""},
		{"every field", res(ResourceRequirement{Ref: "r", Kind: "folder", Access: ResourceAccessWrite, Scope: RequirementScopePerCall, Purpose: "why"}), ""},
		{"kind token characters", res(ResourceRequirement{Ref: "r", Kind: "vai.corpus_v2-beta"}), ""},
		{"ref missing", res(ResourceRequirement{Kind: "corpus"}), ErrMsgRequirementResourceRefEmpty},
		{"kind missing", res(ResourceRequirement{Ref: "r"}), ErrMsgRequirementResourceKindEmpty},
		{"kind uppercase", res(ResourceRequirement{Ref: "r", Kind: "Corpus"}), ErrMsgRequirementResourceKindForm},
		{"kind leading digit", res(ResourceRequirement{Ref: "r", Kind: "2corpus"}), ErrMsgRequirementResourceKindForm},
		{"kind with a space", res(ResourceRequirement{Ref: "r", Kind: "my corpus"}), ErrMsgRequirementResourceKindForm},
		{"ref is a coordinate", res(ResourceRequirement{Ref: "s3://tenant-42/docs", Kind: "corpus"}), ErrMsgRequirementResourceCoordinate},
		// The coordinate sentence wins over the shape complaint: an author who pasted
		// a location into kind needs to be told where it goes.
		{"kind is a coordinate", res(ResourceRequirement{Ref: "r", Kind: "s3://corpus"}), ErrMsgRequirementResourceCoordinate},
		{"access out of vocabulary", res(ResourceRequirement{Ref: "r", Kind: "corpus", Access: "admin"}), ErrMsgRequirementResourceAccess},
		{"scope out of vocabulary", res(ResourceRequirement{Ref: "r", Kind: "corpus", Scope: "team"}), ErrMsgRequirementScopeInvalid},
		{"duplicate ref", res(ResourceRequirement{Ref: "r", Kind: "corpus"}, ResourceRequirement{Ref: "r", Kind: "folder"}), ErrMsgRequirementResourceRefDup},
		// Verbatim, like the other lists: " r" and "r" are different refs and a
		// whitespace-only ref is not empty. Consumers key bindings on the exact string.
		{"refs compared verbatim", res(ResourceRequirement{Ref: "r", Kind: "corpus"}, ResourceRequirement{Ref: " r", Kind: "corpus"}), ""},
		{"purpose too long", res(ResourceRequirement{Ref: "r", Kind: "corpus", Purpose: strings.Repeat("p", MaxRequirementFieldLen+1)}), ErrMsgRequirementFieldTooLong},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.wantMsg == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantMsg)
		})
	}
}

// TestRequirementFieldLengthCountsCharacters pins the v0.31.0 loosening: the cap is
// in runes, as the schema's maxLength is, so a 512-character non-ASCII value (1024
// bytes) is accepted and 513 characters is refused.
func TestRequirementFieldLengthCountsCharacters(t *testing.T) {
	atCap := strings.Repeat("ä", MaxRequirementFieldLen)
	overCap := atCap + "ä"
	for _, req := range []*SpecRequirements{
		{MCP: []MCPRequirement{{Capability: atCap}}},
		{Credentials: []CredentialRequirement{{Ref: atCap}}},
		{Resources: []ResourceRequirement{{Ref: atCap, Kind: "corpus"}}},
	} {
		assert.NoError(t, req.Validate())
	}
	for _, req := range []*SpecRequirements{
		{MCP: []MCPRequirement{{Capability: overCap}}},
		{Credentials: []CredentialRequirement{{Ref: overCap}}},
		{Resources: []ResourceRequirement{{Ref: overCap, Kind: "corpus"}}},
	} {
		assert.Error(t, req.Validate())
	}
}

func TestResourceRequirementsTooManyEntries(t *testing.T) {
	rs := make([]ResourceRequirement, MaxRequirementEntries+1)
	for i := range rs {
		rs[i] = ResourceRequirement{Ref: "r" + strconv.Itoa(i), Kind: "corpus"}
	}
	err := (&SpecRequirements{Resources: rs}).Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgRequirementTooManyEntries)
	assert.NoError(t, (&SpecRequirements{Resources: rs[:MaxRequirementEntries]}).Validate())
}

func TestResourceRequirementEffectiveValues(t *testing.T) {
	assert.Equal(t, ResourceAccessRead, ResourceRequirement{}.EffectiveAccess())
	assert.Equal(t, ResourceAccessWrite, ResourceRequirement{Access: ResourceAccessWrite}.EffectiveAccess())
	assert.Equal(t, RequirementScopeOrg, ResourceRequirement{}.EffectiveScope())
	assert.Equal(t, RequirementScopeUser, ResourceRequirement{Scope: RequirementScopeUser}.EffectiveScope())
	// Verbatim, never corrected: Validate is what refuses an unknown value.
	assert.Equal(t, "admin", ResourceRequirement{Access: "admin"}.EffectiveAccess())
}

func TestResourceRequirementsParsedFromDocument(t *testing.T) {
	doc := `---
name: grounded-agent
description: answers from the product docs
type: agent
requirements:
  resources:
    - ref: product-docs
      kind: corpus
      access: read
      scope: org
      purpose: grounding
---
body
`
	spec, err := Parse([]byte(doc))
	require.NoError(t, err)
	require.NotNil(t, spec.Requirements)
	require.Len(t, spec.Requirements.Resources, 1)
	assert.Equal(t, ResourceRequirement{Ref: "product-docs", Kind: "corpus", Access: "read", Scope: "org", Purpose: "grounding"}, spec.Requirements.Resources[0])
	_, isExtension := spec.Extensions[SpecFieldRequirements]
	assert.False(t, isExtension, "requirements is a typed field and must not also land in Extensions")

	// Parse calls Validate, so a coordinate in a resource ref refuses the document.
	_, err = Parse([]byte(strings.Replace(doc, "ref: product-docs", "ref: s3://tenant-42/docs", 1)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgRequirementResourceCoordinate)
}

func TestResourceRequirementsClone(t *testing.T) {
	orig := &SpecRequirements{Resources: []ResourceRequirement{{Ref: "r", Kind: "corpus"}}}
	clone := orig.Clone()
	require.Len(t, clone.Resources, 1)
	clone.Resources[0].Ref = "other"
	assert.Equal(t, "r", orig.Resources[0].Ref, "clone must be independent")
}

// TestResourceRequirementsSurviveEveryFullExport is the v0.27.0 lesson applied to
// the new list: a typed value that no export writes survives Parse and dies at
// Serialize. Every full-export path is driven and re-parsed — never
// yaml.Marshal(spec), where the struct tag would do the work and hide a
// buildSerializeMap that emitted nothing.
func TestResourceRequirementsSurviveEveryFullExport(t *testing.T) {
	want := []ResourceRequirement{
		{Ref: "product-docs", Kind: "corpus", Access: ResourceAccessRead, Scope: RequirementScopeOrg, Purpose: "grounding"},
		{Ref: "scratch", Kind: "folder", Access: ResourceAccessWrite},
	}
	spec := &Spec{
		Name:         "grounded",
		Description:  "a grounded agent",
		Type:         DocumentTypeAgent,
		Requirements: &SpecRequirements{Resources: want},
		Body:         "body",
	}

	exports := map[string]func() ([]byte, error){
		"ExportFull": spec.ExportFull,
		"Serialize(nil)": func() ([]byte, error) {
			return spec.Serialize(nil)
		},
		"Serialize(FullExportWithCredentials)": func() ([]byte, error) {
			return spec.Serialize(FullExportWithCredentials())
		},
	}
	for name, export := range exports {
		t.Run(name, func(t *testing.T) {
			out, err := export()
			require.NoError(t, err)
			reparsed, err := Parse(out)
			require.NoError(t, err)
			require.NotNil(t, reparsed.Requirements)
			assert.Equal(t, want, reparsed.Requirements.Resources)
		})
	}

	t.Run("ExportDirectory", func(t *testing.T) {
		archive, err := ExportDirectory(spec, nil)
		require.NoError(t, err)
		result, err := ImportDirectory(archive)
		require.NoError(t, err)
		require.NotNil(t, result.Spec.Requirements)
		assert.Equal(t, want, result.Spec.Requirements.Resources)
	})

	// The Agent-Skills card keeps requirements OUT, resources with it — paired with
	// a positive assertion so an empty export cannot pass the absence check.
	t.Run("ExportAgentSkill excludes it", func(t *testing.T) {
		out, err := spec.ExportAgentSkill()
		require.NoError(t, err)
		card := string(out)
		assert.Contains(t, card, "grounded")
		assert.NotContains(t, card, SpecFieldRequirements)
		assert.NotContains(t, card, "product-docs")
	})
}
