package exons

import (
	"strings"
	"testing"

	"github.com/itsatony/go-exons/execution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// ParseYAMLSpec
// =============================================================================

func TestParseYAMLSpec_Empty(t *testing.T) {
	spec, err := ParseYAMLSpec("")
	assert.Error(t, err)
	assert.Nil(t, spec)
	assert.Contains(t, err.Error(), ErrMsgFrontmatterEmpty)
}

func TestParseYAMLSpec_Basic(t *testing.T) {
	yaml := `
name: test-prompt
description: A test prompt
type: prompt
`
	spec, err := ParseYAMLSpec(yaml)
	require.NoError(t, err)
	require.NotNil(t, spec)
	assert.Equal(t, "test-prompt", spec.Name)
	assert.Equal(t, "A test prompt", spec.Description)
	assert.Equal(t, DocumentTypePrompt, spec.Type)
}

func TestParseYAMLSpec_AllFields(t *testing.T) {
	yaml := `
name: full-agent
description: A fully configured agent
type: agent
execution:
  provider: openai
  model: gpt-4
inputs:
  query:
    type: string
    required: true
    description: The user query
outputs:
  result:
    type: string
    description: The response
sample:
  query: What is Go?
skills:
  - slug: web-search
    injection: system_prompt
tools:
  functions:
    - name: search
      description: Search the web
      parameters:
        type: object
        properties:
          q:
            type: string
constraints:
  behavioral:
    - Be polite
  safety:
    - No harmful content
messages:
  - role: system
    content: You are helpful.
  - role: user
    content: Hello
context:
  company: Acme
credentials:
  main:
    provider: openai
    ref: "${OPENAI_KEY}"
credential: main
`
	spec, err := ParseYAMLSpec(yaml)
	require.NoError(t, err)
	require.NotNil(t, spec)

	assert.Equal(t, "full-agent", spec.Name)
	assert.Equal(t, DocumentTypeAgent, spec.Type)
	assert.NotNil(t, spec.Execution)
	assert.Equal(t, "openai", spec.Execution.Provider)
	assert.Equal(t, "gpt-4", spec.Execution.Model)

	require.NotNil(t, spec.Inputs)
	assert.Contains(t, spec.Inputs, "query")
	assert.True(t, spec.Inputs["query"].Required)

	require.NotNil(t, spec.Outputs)
	assert.Contains(t, spec.Outputs, "result")

	require.NotNil(t, spec.Sample)
	assert.Equal(t, "What is Go?", spec.Sample["query"])

	require.Len(t, spec.Skills, 1)
	assert.Equal(t, "web-search", spec.Skills[0].Slug)
	assert.Equal(t, "system_prompt", spec.Skills[0].Injection)

	require.NotNil(t, spec.Tools)
	require.Len(t, spec.Tools.Functions, 1)
	assert.Equal(t, "search", spec.Tools.Functions[0].Name)

	require.NotNil(t, spec.Constraints)
	assert.Len(t, spec.Constraints.Behavioral, 1)
	assert.Len(t, spec.Constraints.Safety, 1)

	require.Len(t, spec.Messages, 2)
	assert.Equal(t, "system", spec.Messages[0].Role)
	assert.Equal(t, "user", spec.Messages[1].Role)

	assert.Equal(t, "Acme", spec.Context["company"])

	require.NotNil(t, spec.Credentials)
	assert.Contains(t, spec.Credentials, "main")
	assert.Equal(t, "openai", spec.Credentials["main"].Provider)
	assert.Equal(t, "main", spec.Credential)
}

func TestParseYAMLSpec_InvalidYAML(t *testing.T) {
	_, err := ParseYAMLSpec("invalid: [yaml: content")
	assert.Error(t, err)
}

func TestParseYAMLSpec_TooLarge(t *testing.T) {
	large := strings.Repeat("x", int(DefaultMaxFrontmatterSize)+1)
	_, err := ParseYAMLSpec(large)
	assert.Error(t, err)
}

// =============================================================================
// Validate
// =============================================================================

func TestSpec_Validate(t *testing.T) {
	t.Run("nil spec fails", func(t *testing.T) {
		var s *Spec
		err := s.Validate()
		assert.Error(t, err)
	})

	t.Run("empty name fails", func(t *testing.T) {
		s := &Spec{Description: "desc"}
		err := s.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgSpecNameRequired)
	})

	t.Run("name too long fails", func(t *testing.T) {
		s := &Spec{
			Name:        strings.Repeat("a", SpecNameMaxLength+1),
			Description: "desc",
		}
		err := s.Validate()
		assert.Error(t, err)
	})

	t.Run("invalid name format fails", func(t *testing.T) {
		s := &Spec{
			Name:        "Invalid-Name",
			Description: "desc",
		}
		err := s.Validate()
		assert.Error(t, err)
	})

	t.Run("name starting with digit fails", func(t *testing.T) {
		s := &Spec{
			Name:        "1abc",
			Description: "desc",
		}
		err := s.Validate()
		assert.Error(t, err)
	})

	t.Run("valid slug format passes", func(t *testing.T) {
		s := &Spec{
			Name:        "my-valid-name",
			Description: "A valid description",
		}
		err := s.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing description fails", func(t *testing.T) {
		s := &Spec{
			Name: "test-name",
		}
		err := s.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgSpecDescriptionRequired)
	})

	t.Run("description too long fails", func(t *testing.T) {
		s := &Spec{
			Name:        "test-name",
			Description: strings.Repeat("a", SpecDescriptionMaxLength+1),
		}
		err := s.Validate()
		assert.Error(t, err)
	})

	t.Run("invalid document type fails", func(t *testing.T) {
		s := &Spec{
			Name:        "test-name",
			Description: "desc",
			Type:        DocumentType("invalid"),
		}
		err := s.Validate()
		assert.Error(t, err)
	})

	t.Run("valid document types pass", func(t *testing.T) {
		for _, dt := range []DocumentType{DocumentTypePrompt, DocumentTypeSkill, DocumentTypeAgent, ""} {
			s := &Spec{
				Name:        "test-name",
				Description: "desc",
				Type:        dt,
			}
			err := s.Validate()
			assert.NoError(t, err, "type %q should be valid", dt)
		}
	})
}

// =============================================================================
// Validate — Type-Specific
// =============================================================================

func TestSpec_Validate_TypeSpecific(t *testing.T) {
	t.Run("prompt with skills fails", func(t *testing.T) {
		s := &Spec{
			Name:        "test-prompt",
			Description: "desc",
			Type:        DocumentTypePrompt,
			Skills:      []SkillRef{{Slug: "some-skill"}},
		}
		err := s.Validate()
		assert.Error(t, err)
	})

	t.Run("prompt with tools fails", func(t *testing.T) {
		s := &Spec{
			Name:        "test-prompt",
			Description: "desc",
			Type:        DocumentTypePrompt,
			Tools:       &ToolsConfig{Functions: []*FunctionDef{{Name: "fn"}}},
		}
		err := s.Validate()
		assert.Error(t, err)
	})

	t.Run("prompt with constraints fails", func(t *testing.T) {
		s := &Spec{
			Name:        "test-prompt",
			Description: "desc",
			Type:        DocumentTypePrompt,
			Constraints: &ConstraintsConfig{Behavioral: []string{"be nice"}},
		}
		err := s.Validate()
		assert.Error(t, err)
	})

	t.Run("skill with skills fails", func(t *testing.T) {
		s := &Spec{
			Name:        "test-skill",
			Description: "desc",
			Type:        DocumentTypeSkill,
			Skills:      []SkillRef{{Slug: "nested"}},
		}
		err := s.Validate()
		assert.Error(t, err)
	})

	t.Run("agent with skills passes", func(t *testing.T) {
		s := &Spec{
			Name:        "test-agent",
			Description: "desc",
			Type:        DocumentTypeAgent,
			Skills:      []SkillRef{{Slug: "some-skill"}},
		}
		err := s.Validate()
		assert.NoError(t, err)
	})
}

// =============================================================================
// ValidateOptional
// =============================================================================

func TestSpec_ValidateOptional(t *testing.T) {
	t.Run("nil passes", func(t *testing.T) {
		var s *Spec
		assert.NoError(t, s.ValidateOptional())
	})

	t.Run("empty spec passes", func(t *testing.T) {
		s := &Spec{}
		assert.NoError(t, s.ValidateOptional())
	})

	t.Run("spec with name triggers validation", func(t *testing.T) {
		s := &Spec{Name: "Invalid-Name"}
		err := s.ValidateOptional()
		assert.Error(t, err) // triggers full validate, which fails on slug format
	})

	t.Run("spec with type triggers validation", func(t *testing.T) {
		s := &Spec{Type: DocumentTypeAgent}
		err := s.ValidateOptional()
		assert.Error(t, err) // triggers full validate, which fails on name
	})

	t.Run("spec with credentials triggers validation", func(t *testing.T) {
		s := &Spec{
			Credentials: map[string]*CredentialRef{
				"main": {Provider: "openai"},
			},
		}
		err := s.ValidateOptional()
		assert.Error(t, err) // name required
	})
}

// =============================================================================
// Getter Methods
// =============================================================================

func TestSpec_Getters(t *testing.T) {
	t.Run("GetSlug", func(t *testing.T) {
		s := &Spec{Name: "my-slug"}
		assert.Equal(t, "my-slug", s.GetSlug())

		var nilSpec *Spec
		assert.Equal(t, "", nilSpec.GetSlug())
	})

	t.Run("EffectiveType", func(t *testing.T) {
		s := &Spec{Type: DocumentTypeAgent}
		assert.Equal(t, DocumentTypeAgent, s.EffectiveType())

		sEmpty := &Spec{}
		assert.Equal(t, DocumentTypeSkill, sEmpty.EffectiveType()) // default

		var nilSpec *Spec
		assert.Equal(t, DocumentTypeSkill, nilSpec.EffectiveType())
	})

	t.Run("HasExecution", func(t *testing.T) {
		assert.False(t, (&Spec{}).HasExecution())
		assert.False(t, (*Spec)(nil).HasExecution())
	})

	t.Run("HasTools", func(t *testing.T) {
		assert.False(t, (&Spec{}).HasTools())
		assert.False(t, (&Spec{Tools: &ToolsConfig{}}).HasTools()) // empty tools
		assert.True(t, (&Spec{Tools: &ToolsConfig{Functions: []*FunctionDef{{Name: "fn"}}}}).HasTools())
	})

	t.Run("HasSkills", func(t *testing.T) {
		assert.False(t, (&Spec{}).HasSkills())
		assert.True(t, (&Spec{Skills: []SkillRef{{Slug: "s"}}}).HasSkills())
	})

	t.Run("HasConstraints", func(t *testing.T) {
		assert.False(t, (&Spec{}).HasConstraints())
		assert.True(t, (&Spec{Constraints: &ConstraintsConfig{}}).HasConstraints())
	})

	t.Run("HasExtensions", func(t *testing.T) {
		assert.False(t, (&Spec{}).HasExtensions())
		assert.True(t, (&Spec{Extensions: map[string]any{"k": "v"}}).HasExtensions())
	})

	t.Run("HasCredentials", func(t *testing.T) {
		assert.False(t, (&Spec{}).HasCredentials())
		assert.True(t, (&Spec{Credentials: map[string]*CredentialRef{"k": {}}}).HasCredentials())
	})
}

// =============================================================================
// IsAgentSkillsCompatible
// =============================================================================

func TestSpec_IsAgentSkillsCompatible(t *testing.T) {
	t.Run("nil spec is compatible", func(t *testing.T) {
		var s *Spec
		assert.True(t, s.IsAgentSkillsCompatible())
	})

	t.Run("empty spec is compatible", func(t *testing.T) {
		s := &Spec{}
		assert.True(t, s.IsAgentSkillsCompatible())
	})

	t.Run("spec with name and description is compatible", func(t *testing.T) {
		s := &Spec{Name: "test", Description: "desc"}
		assert.True(t, s.IsAgentSkillsCompatible())
	})

	t.Run("spec with execution is not compatible", func(t *testing.T) {
		s := &Spec{Execution: &execution.Config{Provider: "openai"}}
		assert.False(t, s.IsAgentSkillsCompatible())
	})

	t.Run("spec with extensions is not compatible", func(t *testing.T) {
		s := &Spec{Extensions: map[string]any{"k": "v"}}
		assert.False(t, s.IsAgentSkillsCompatible())
	})

	t.Run("spec with type is not compatible", func(t *testing.T) {
		s := &Spec{Type: DocumentTypeAgent}
		assert.False(t, s.IsAgentSkillsCompatible())
	})

	t.Run("spec with skills is not compatible", func(t *testing.T) {
		s := &Spec{Skills: []SkillRef{{Slug: "s"}}}
		assert.False(t, s.IsAgentSkillsCompatible())
	})

	t.Run("spec with tools is not compatible", func(t *testing.T) {
		s := &Spec{Tools: &ToolsConfig{}}
		assert.False(t, s.IsAgentSkillsCompatible())
	})

	t.Run("spec with constraints is not compatible", func(t *testing.T) {
		s := &Spec{Constraints: &ConstraintsConfig{}}
		assert.False(t, s.IsAgentSkillsCompatible())
	})

	t.Run("spec with messages is not compatible", func(t *testing.T) {
		s := &Spec{Messages: []MessageTemplate{{Role: "user", Content: "hi"}}}
		assert.False(t, s.IsAgentSkillsCompatible())
	})

	t.Run("spec with credentials is not compatible", func(t *testing.T) {
		s := &Spec{Credentials: map[string]*CredentialRef{"k": {}}}
		assert.False(t, s.IsAgentSkillsCompatible())
	})

	t.Run("spec with credential is not compatible", func(t *testing.T) {
		s := &Spec{Credential: "main"}
		assert.False(t, s.IsAgentSkillsCompatible())
	})

	t.Run("spec with memory is not compatible", func(t *testing.T) {
		s := &Spec{Memory: &MemorySpec{Scope: "agent-memory"}}
		assert.False(t, s.IsAgentSkillsCompatible())
	})

	t.Run("spec with dispatch is not compatible", func(t *testing.T) {
		s := &Spec{Dispatch: &DispatchSpec{}}
		assert.False(t, s.IsAgentSkillsCompatible())
	})

	t.Run("spec with verifications is not compatible", func(t *testing.T) {
		s := &Spec{Verifications: []VerificationCase{{Name: "test-case", Input: map[string]any{"q": "test"}}}}
		assert.False(t, s.IsAgentSkillsCompatible())
	})

	t.Run("spec with registry is not compatible", func(t *testing.T) {
		s := &Spec{Registry: &RegistrySpec{Version: "1.0"}}
		assert.False(t, s.IsAgentSkillsCompatible())
	})

	t.Run("spec with safety is not compatible", func(t *testing.T) {
		s := &Spec{Safety: &SafetyConfig{Guardrails: GuardrailsEnabled}}
		assert.False(t, s.IsAgentSkillsCompatible())
	})
}

// =============================================================================
// Type Helpers
// =============================================================================

func TestSpec_TypeHelpers(t *testing.T) {
	t.Run("IsAgent", func(t *testing.T) {
		assert.True(t, (&Spec{Type: DocumentTypeAgent}).IsAgent())
		assert.False(t, (&Spec{Type: DocumentTypeSkill}).IsAgent())
		assert.False(t, (*Spec)(nil).IsAgent())
	})

	t.Run("IsSkill", func(t *testing.T) {
		assert.True(t, (&Spec{Type: DocumentTypeSkill}).IsSkill())
		assert.False(t, (&Spec{Type: DocumentTypeAgent}).IsSkill())
		assert.False(t, (*Spec)(nil).IsSkill())
	})

	t.Run("IsPrompt", func(t *testing.T) {
		assert.True(t, (&Spec{Type: DocumentTypePrompt}).IsPrompt())
		assert.False(t, (&Spec{Type: DocumentTypeAgent}).IsPrompt())
		assert.False(t, (*Spec)(nil).IsPrompt())
	})
}

// =============================================================================
// Clone
// =============================================================================

func TestSpec_Clone(t *testing.T) {
	t.Run("nil clone", func(t *testing.T) {
		var s *Spec
		assert.Nil(t, s.Clone())
	})

	t.Run("basic fields", func(t *testing.T) {
		s := &Spec{
			Name:        "test",
			Description: "desc",
			Type:        DocumentTypeAgent,
			Credential:  "main",
			Body:        "body content",
		}
		clone := s.Clone()
		assert.Equal(t, s.Name, clone.Name)
		assert.Equal(t, s.Description, clone.Description)
		assert.Equal(t, s.Type, clone.Type)
		assert.Equal(t, s.Credential, clone.Credential)
		assert.Equal(t, s.Body, clone.Body)
	})

	t.Run("deep copy isolation", func(t *testing.T) {
		s := &Spec{
			Name:        "test",
			Description: "desc",
			Inputs: map[string]*InputDef{
				"q": {Type: "string", Required: true},
			},
			Outputs: map[string]*OutputDef{
				"r": {Type: "string"},
			},
			Sample:  map[string]any{"key": "value"},
			Skills:  []SkillRef{{Slug: "skill1"}},
			Context: map[string]any{"ctx": "val"},
			Messages: []MessageTemplate{
				{Role: "user", Content: "hi"},
			},
			Credentials: map[string]*CredentialRef{
				"main": {Provider: "openai"},
			},
			Extensions: map[string]any{"ext": "val"},
		}

		clone := s.Clone()

		// Modify original
		s.Inputs["q"].Required = false
		s.Sample["key"] = "changed"
		s.Skills[0].Slug = "changed"
		s.Context["ctx"] = "changed"
		s.Messages[0].Content = "changed"
		s.Credentials["main"].Provider = "changed"
		s.Extensions["ext"] = "changed"

		// Clone should be unaffected
		assert.True(t, clone.Inputs["q"].Required)
		assert.Equal(t, "value", clone.Sample["key"])
		assert.Equal(t, "skill1", clone.Skills[0].Slug)
		assert.Equal(t, "val", clone.Context["ctx"])
		assert.Equal(t, "hi", clone.Messages[0].Content)
		assert.Equal(t, "openai", clone.Credentials["main"].Provider)
		assert.Equal(t, "val", clone.Extensions["ext"])
	})

	t.Run("tools deep copy", func(t *testing.T) {
		s := &Spec{
			Name:        "test",
			Description: "desc",
			Tools: &ToolsConfig{
				Functions: []*FunctionDef{
					{Name: "fn1", Description: "desc1"},
				},
				MCPServers: []*MCPServer{
					{Name: "mcp1", URL: "http://example.com"},
				},
			},
		}
		clone := s.Clone()
		s.Tools.Functions[0].Name = "changed"
		s.Tools.MCPServers[0].Name = "changed"
		assert.Equal(t, "fn1", clone.Tools.Functions[0].Name)
		assert.Equal(t, "mcp1", clone.Tools.MCPServers[0].Name)
	})

	t.Run("constraints deep copy", func(t *testing.T) {
		maxTurns := 10
		s := &Spec{
			Name:        "test",
			Description: "desc",
			Constraints: &ConstraintsConfig{
				Behavioral: []string{"be nice"},
				Safety:     []string{"no harm"},
				Operational: &OperationalConstraints{
					MaxTurns: &maxTurns,
				},
			},
		}
		clone := s.Clone()
		s.Constraints.Behavioral[0] = "changed"
		s.Constraints.Safety[0] = "changed"
		assert.Equal(t, "be nice", clone.Constraints.Behavioral[0])
		assert.Equal(t, "no harm", clone.Constraints.Safety[0])
		assert.NotNil(t, clone.Constraints.Operational)
	})
}

// =============================================================================
// isValidDocumentType
// =============================================================================

func TestIsValidDocumentType(t *testing.T) {
	assert.True(t, isValidDocumentType(DocumentTypePrompt))
	assert.True(t, isValidDocumentType(DocumentTypeSkill))
	assert.True(t, isValidDocumentType(DocumentTypeAgent))
	assert.False(t, isValidDocumentType(DocumentType("invalid")))
	assert.False(t, isValidDocumentType(DocumentType("")))
}

// =============================================================================
// HasMetadata
// =============================================================================

func TestSpec_HasMetadata(t *testing.T) {
	assert.False(t, (*Spec)(nil).HasMetadata())
	assert.False(t, (&Spec{}).HasMetadata())

	// With metadata fields set, should return true
	s := &Spec{Memory: &MemorySpec{Scope: "agent-memory"}}
	assert.True(t, s.HasMetadata())

	// Verify all metadata fields individually trigger HasMetadata
	assert.True(t, (&Spec{Dispatch: &DispatchSpec{}}).HasMetadata())
	assert.True(t, (&Spec{Verifications: []VerificationCase{{Name: "test-case", Input: map[string]any{"q": "test"}}}}).HasMetadata())
	assert.True(t, (&Spec{Registry: &RegistrySpec{Version: "1.0"}}).HasMetadata())
	assert.True(t, (&Spec{Safety: &SafetyConfig{Guardrails: GuardrailsEnabled}}).HasMetadata())

	// Empty spec
	assert.False(t, (&Spec{}).HasMetadata())
	assert.False(t, (*Spec)(nil).HasMetadata())
}

// =============================================================================
// ToolsConfig.Clone
// =============================================================================

func TestToolsConfig_Clone(t *testing.T) {
	t.Run("nil_returns_nil", func(t *testing.T) {
		var tc *ToolsConfig
		assert.Nil(t, tc.Clone())
	})

	t.Run("empty", func(t *testing.T) {
		tc := &ToolsConfig{}
		clone := tc.Clone()
		require.NotNil(t, clone)
		assert.Empty(t, clone.Functions)
		assert.Empty(t, clone.MCPServers)
		assert.Empty(t, clone.Allow)
		assert.Nil(t, clone.ParallelToolCalls)
		assert.Empty(t, clone.ToolChoice)
	})

	t.Run("full_deep_copy", func(t *testing.T) {
		ptc := true
		tc := &ToolsConfig{
			Functions: []*FunctionDef{
				{
					Name:        "search",
					Description: "Search the web",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"query": map[string]any{"type": "string"},
						},
					},
				},
				{
					Name:        "calc",
					Description: "Calculate",
					Parameters:  map[string]any{"type": "number"},
				},
			},
			MCPServers: []*MCPServer{
				{Name: "server1", URL: "http://example.com"},
			},
			ToolChoice:        "auto",
			ParallelToolCalls: &ptc,
			Allow:             []string{"search", "calc"},
		}

		clone := tc.Clone()
		require.NotNil(t, clone)

		// Verify all fields match
		require.Len(t, clone.Functions, 2)
		assert.Equal(t, "search", clone.Functions[0].Name)
		assert.Equal(t, "Search the web", clone.Functions[0].Description)
		assert.Equal(t, "calc", clone.Functions[1].Name)
		require.Len(t, clone.MCPServers, 1)
		assert.Equal(t, "server1", clone.MCPServers[0].Name)
		assert.Equal(t, "http://example.com", clone.MCPServers[0].URL)
		assert.Equal(t, "auto", clone.ToolChoice)
		require.NotNil(t, clone.ParallelToolCalls)
		assert.True(t, *clone.ParallelToolCalls)
		assert.Equal(t, []string{"search", "calc"}, clone.Allow)

		// Mutate clone and verify original unchanged
		clone.Functions[0].Name = "modified"
		clone.MCPServers[0].Name = "modified"
		clone.Allow[0] = "modified"
		*clone.ParallelToolCalls = false
		clone.ToolChoice = "none"

		assert.Equal(t, "search", tc.Functions[0].Name)
		assert.Equal(t, "server1", tc.MCPServers[0].Name)
		assert.Equal(t, "search", tc.Allow[0])
		assert.True(t, *tc.ParallelToolCalls)
		assert.Equal(t, "auto", tc.ToolChoice)
	})

	t.Run("functions_deep_copy", func(t *testing.T) {
		tc := &ToolsConfig{
			Functions: []*FunctionDef{
				{
					Name: "fn1",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"name": map[string]any{"type": "string"},
						},
					},
				},
			},
		}

		clone := tc.Clone()

		// Modify cloned function's Parameters map
		clonedProps := clone.Functions[0].Parameters["properties"].(map[string]any)
		clonedProps["name"] = map[string]any{"type": "integer"}

		// Original should be unchanged
		origProps := tc.Functions[0].Parameters["properties"].(map[string]any)
		origName := origProps["name"].(map[string]any)
		assert.Equal(t, "string", origName["type"])
	})

	t.Run("parallel_tool_calls_deep_copy", func(t *testing.T) {
		val := true
		tc := &ToolsConfig{
			ParallelToolCalls: &val,
		}

		clone := tc.Clone()
		require.NotNil(t, clone.ParallelToolCalls)
		assert.True(t, *clone.ParallelToolCalls)

		// Modify cloned pointer
		*clone.ParallelToolCalls = false

		// Original should be unchanged
		assert.True(t, *tc.ParallelToolCalls)
	})
}

// =============================================================================
// ConstraintsConfig.Clone
// =============================================================================

func TestConstraintsConfig_Clone(t *testing.T) {
	t.Run("nil_returns_nil", func(t *testing.T) {
		var cc *ConstraintsConfig
		assert.Nil(t, cc.Clone())
	})

	t.Run("empty", func(t *testing.T) {
		cc := &ConstraintsConfig{}
		clone := cc.Clone()
		require.NotNil(t, clone)
		assert.Empty(t, clone.Behavioral)
		assert.Empty(t, clone.Safety)
		assert.Nil(t, clone.Operational)
	})

	t.Run("full_deep_copy", func(t *testing.T) {
		maxTurns := 10
		timeout := 30
		cc := &ConstraintsConfig{
			Behavioral: []string{"be polite", "cite sources"},
			Safety:     []string{"no harmful content", "no PII"},
			Operational: &OperationalConstraints{
				MaxTurns:       &maxTurns,
				TimeoutSeconds: &timeout,
			},
		}

		clone := cc.Clone()
		require.NotNil(t, clone)

		// Verify all fields match
		assert.Equal(t, []string{"be polite", "cite sources"}, clone.Behavioral)
		assert.Equal(t, []string{"no harmful content", "no PII"}, clone.Safety)
		require.NotNil(t, clone.Operational)
		require.NotNil(t, clone.Operational.MaxTurns)
		assert.Equal(t, 10, *clone.Operational.MaxTurns)
		require.NotNil(t, clone.Operational.TimeoutSeconds)
		assert.Equal(t, 30, *clone.Operational.TimeoutSeconds)

		// Mutate clone and verify original unchanged
		clone.Behavioral[0] = "modified"
		clone.Safety[0] = "modified"
		*clone.Operational.MaxTurns = 99

		assert.Equal(t, "be polite", cc.Behavioral[0])
		assert.Equal(t, "no harmful content", cc.Safety[0])
		assert.Equal(t, 10, *cc.Operational.MaxTurns)
	})

	t.Run("operational_deep_copy", func(t *testing.T) {
		maxTurns := 5
		cc := &ConstraintsConfig{
			Operational: &OperationalConstraints{
				MaxTurns:       &maxTurns,
				AllowedDomains: []string{"example.com"},
			},
		}

		clone := cc.Clone()
		require.NotNil(t, clone.Operational)

		// Modify cloned Operational fields
		*clone.Operational.MaxTurns = 100
		clone.Operational.AllowedDomains[0] = "modified.com"

		// Original should be unchanged
		assert.Equal(t, 5, *cc.Operational.MaxTurns)
		assert.Equal(t, "example.com", cc.Operational.AllowedDomains[0])
	})
}

// =============================================================================
// OperationalConstraints.Clone
// =============================================================================

func TestOperationalConstraints_Clone(t *testing.T) {
	t.Run("nil_returns_nil", func(t *testing.T) {
		var oc *OperationalConstraints
		assert.Nil(t, oc.Clone())
	})

	t.Run("full_deep_copy", func(t *testing.T) {
		maxTurns := 10
		maxTokens := 4096
		timeout := 60
		maxToolCalls := 50

		oc := &OperationalConstraints{
			MaxTurns:         &maxTurns,
			MaxTokensPerTurn: &maxTokens,
			TimeoutSeconds:   &timeout,
			MaxToolCalls:     &maxToolCalls,
			AllowedDomains:   []string{"example.com", "api.example.com"},
			BlockedDomains:   []string{"evil.com"},
		}

		clone := oc.Clone()
		require.NotNil(t, clone)

		// Verify all fields match
		require.NotNil(t, clone.MaxTurns)
		assert.Equal(t, 10, *clone.MaxTurns)
		require.NotNil(t, clone.MaxTokensPerTurn)
		assert.Equal(t, 4096, *clone.MaxTokensPerTurn)
		require.NotNil(t, clone.TimeoutSeconds)
		assert.Equal(t, 60, *clone.TimeoutSeconds)
		require.NotNil(t, clone.MaxToolCalls)
		assert.Equal(t, 50, *clone.MaxToolCalls)
		assert.Equal(t, []string{"example.com", "api.example.com"}, clone.AllowedDomains)
		assert.Equal(t, []string{"evil.com"}, clone.BlockedDomains)

		// Mutate clone and verify original unchanged
		*clone.MaxTurns = 99
		*clone.MaxTokensPerTurn = 1
		*clone.TimeoutSeconds = 1
		*clone.MaxToolCalls = 1
		clone.AllowedDomains[0] = "modified.com"
		clone.BlockedDomains[0] = "modified.com"

		assert.Equal(t, 10, *oc.MaxTurns)
		assert.Equal(t, 4096, *oc.MaxTokensPerTurn)
		assert.Equal(t, 60, *oc.TimeoutSeconds)
		assert.Equal(t, 50, *oc.MaxToolCalls)
		assert.Equal(t, "example.com", oc.AllowedDomains[0])
		assert.Equal(t, "evil.com", oc.BlockedDomains[0])
	})
}

// =============================================================================
// CredentialRef.Validate
// =============================================================================

func TestCredentialRef_Validate(t *testing.T) {
	t.Run("nil_returns_nil", func(t *testing.T) {
		var cr *CredentialRef
		assert.NoError(t, cr.Validate())
	})

	t.Run("missing_provider_returns_error", func(t *testing.T) {
		cr := &CredentialRef{
			Label: "my-cred",
			Ref:   "${API_KEY}",
		}
		err := cr.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgCredentialMissingProvider)
	})

	t.Run("valid", func(t *testing.T) {
		cr := &CredentialRef{
			Provider: "openai",
			Label:    "main",
			Ref:      "${OPENAI_KEY}",
			Scopes:   []string{"chat", "images"},
		}
		assert.NoError(t, cr.Validate())
	})
}

// =============================================================================
// CredentialRef.Clone
// =============================================================================

func TestCredentialRef_Clone(t *testing.T) {
	t.Run("nil_returns_nil", func(t *testing.T) {
		var cr *CredentialRef
		assert.Nil(t, cr.Clone())
	})

	t.Run("full_deep_copy", func(t *testing.T) {
		cr := &CredentialRef{
			Provider: "anthropic",
			Label:    "main-cred",
			Ref:      "${ANTHROPIC_KEY}",
			Scopes:   []string{"chat", "completion"},
		}

		clone := cr.Clone()
		require.NotNil(t, clone)

		// Verify all fields match
		assert.Equal(t, "anthropic", clone.Provider)
		assert.Equal(t, "main-cred", clone.Label)
		assert.Equal(t, "${ANTHROPIC_KEY}", clone.Ref)
		assert.Equal(t, []string{"chat", "completion"}, clone.Scopes)

		// Mutate clone Scopes and verify original unchanged
		clone.Scopes[0] = "modified"
		clone.Provider = "modified"

		assert.Equal(t, "chat", cr.Scopes[0])
		assert.Equal(t, "anthropic", cr.Provider)
	})
}

// =============================================================================
// Spec.ValidateCredentialRefs
// =============================================================================

func TestSpec_ValidateCredentialRefs(t *testing.T) {
	t.Run("nil_spec_returns_nil", func(t *testing.T) {
		var s *Spec
		assert.NoError(t, s.ValidateCredentialRefs())
	})

	t.Run("empty_credentials_returns_nil", func(t *testing.T) {
		s := &Spec{}
		assert.NoError(t, s.ValidateCredentialRefs())
	})

	t.Run("valid_credentials", func(t *testing.T) {
		s := &Spec{
			Credentials: map[string]*CredentialRef{
				"main": {Provider: "openai", Ref: "${OPENAI_KEY}"},
				"alt":  {Provider: "anthropic", Ref: "${ANTHROPIC_KEY}"},
			},
			Credential: "main",
		}
		assert.NoError(t, s.ValidateCredentialRefs())
	})

	t.Run("credential_missing_provider", func(t *testing.T) {
		s := &Spec{
			Credentials: map[string]*CredentialRef{
				"bad-cred": {Ref: "${API_KEY}"},
			},
		}
		err := s.ValidateCredentialRefs()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgCredentialMissingProvider)
	})

	t.Run("default_label_not_in_map", func(t *testing.T) {
		s := &Spec{
			Credentials: map[string]*CredentialRef{
				"main": {Provider: "openai", Ref: "${KEY}"},
			},
			Credential: "nonexistent",
		}
		err := s.ValidateCredentialRefs()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgCredentialMissingRef)
	})

	t.Run("skill_credential_not_in_map", func(t *testing.T) {
		s := &Spec{
			Credentials: map[string]*CredentialRef{
				"main": {Provider: "openai", Ref: "${KEY}"},
			},
			Skills: []SkillRef{
				{Slug: "image-gen", Credential: "missing-label"},
			},
		}
		err := s.ValidateCredentialRefs()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgCredentialMissingRef)
	})

	t.Run("valid_with_skills", func(t *testing.T) {
		s := &Spec{
			Credentials: map[string]*CredentialRef{
				"main":   {Provider: "openai", Ref: "${OPENAI_KEY}"},
				"images": {Provider: "openai", Ref: "${IMAGES_KEY}", Scopes: []string{"images"}},
			},
			Credential: "main",
			Skills: []SkillRef{
				{Slug: "image-gen", Credential: "images"},
				{Slug: "chat", Credential: "main"},
			},
		}
		assert.NoError(t, s.ValidateCredentialRefs())
	})
}
