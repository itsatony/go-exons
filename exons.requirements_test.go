package exons

import (
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
