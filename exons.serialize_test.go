package exons

import (
	"testing"

	"github.com/itsatony/go-exons/execution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Round-trip tests
// =============================================================================

func TestSerialize_RoundTrip_Basic(t *testing.T) {
	yamlDoc := "---\nname: test-spec\ndescription: A test spec\ntype: skill\n---\nBody content here.\n"
	spec1, err := Parse([]byte(yamlDoc))
	require.NoError(t, err)

	serialized, err := spec1.Serialize(nil)
	require.NoError(t, err)
	require.NotNil(t, serialized)

	spec2, err := Parse(serialized)
	require.NoError(t, err)

	assert.Equal(t, spec1.Name, spec2.Name)
	assert.Equal(t, spec1.Description, spec2.Description)
	assert.Equal(t, spec1.Type, spec2.Type)
	assert.Equal(t, spec1.Body, spec2.Body)
}

func TestSerialize_RoundTrip_WithExecution(t *testing.T) {
	temp := 0.7
	spec1 := &Spec{
		Name:        "test-spec",
		Description: "desc",
		Type:        DocumentTypeAgent,
		Execution: &execution.Config{
			Provider:    "openai",
			Model:       "gpt-4",
			Temperature: &temp,
		},
		Body: "Some body",
	}

	serialized, err := spec1.Serialize(nil)
	require.NoError(t, err)

	spec2, err := Parse(serialized)
	require.NoError(t, err)
	require.NotNil(t, spec2.Execution)
	assert.Equal(t, "openai", spec2.Execution.Provider)
	assert.Equal(t, "gpt-4", spec2.Execution.Model)
}

func TestSerialize_RoundTrip_WithAgentFields(t *testing.T) {
	spec1 := &Spec{
		Name:        "test-agent",
		Description: "desc",
		Type:        DocumentTypeAgent,
		Skills: []SkillRef{
			{Slug: "web-search", Injection: "system_prompt"},
		},
		Tools: &ToolsConfig{
			Functions: []*FunctionDef{
				{Name: "search", Description: "Search the web"},
			},
		},
		Constraints: &ConstraintsConfig{
			Behavioral: []string{"Be polite"},
		},
		Messages: []MessageTemplate{
			{Role: "system", Content: "You are helpful."},
		},
		Context: map[string]any{"company": "Acme"},
	}

	serialized, err := spec1.Serialize(nil)
	require.NoError(t, err)

	spec2, err := Parse(serialized)
	require.NoError(t, err)
	assert.Len(t, spec2.Skills, 1)
	assert.Equal(t, "web-search", spec2.Skills[0].Slug)
	assert.NotNil(t, spec2.Tools)
	assert.NotNil(t, spec2.Constraints)
	assert.Len(t, spec2.Messages, 1)
	assert.Equal(t, "Acme", spec2.Context["company"])
}

// =============================================================================
// Option toggle tests
// =============================================================================

func TestSerialize_DefaultOptions(t *testing.T) {
	opts := DefaultSerializeOptions()
	assert.True(t, opts.IncludeExecution)
	assert.True(t, opts.IncludeExtensions)
	assert.True(t, opts.IncludeAgentFields)
	assert.True(t, opts.IncludeContext)
	assert.False(t, opts.IncludeCredentials)
	assert.True(t, opts.IncludeMetadata)
}

func TestSerialize_IncludeExecutionFalse(t *testing.T) {
	temp := 0.7
	spec := &Spec{
		Name:        "test-spec",
		Description: "desc",
		Execution: &execution.Config{
			Provider:    "openai",
			Temperature: &temp,
		},
	}
	opts := &SerializeOptions{IncludeExecution: false}
	serialized, err := spec.Serialize(opts)
	require.NoError(t, err)
	assert.NotContains(t, string(serialized), "execution:")
}

func TestSerialize_IncludeAgentFieldsFalse(t *testing.T) {
	spec := &Spec{
		Name:        "test-spec",
		Description: "desc",
		Type:        DocumentTypeAgent,
		Skills:      []SkillRef{{Slug: "s"}},
		Messages:    []MessageTemplate{{Role: "user", Content: "hi"}},
	}
	opts := &SerializeOptions{IncludeAgentFields: false, IncludeExecution: true}
	serialized, err := spec.Serialize(opts)
	require.NoError(t, err)
	s := string(serialized)
	assert.NotContains(t, s, "skills:")
	assert.NotContains(t, s, "messages:")
	assert.NotContains(t, s, "type:")
}

func TestSerialize_IncludeContextFalse(t *testing.T) {
	spec := &Spec{
		Name:        "test-spec",
		Description: "desc",
		Context:     map[string]any{"key": "value"},
	}
	opts := &SerializeOptions{IncludeContext: false}
	serialized, err := spec.Serialize(opts)
	require.NoError(t, err)
	assert.NotContains(t, string(serialized), "context:")
}

func TestSerialize_IncludeCredentialsTrue(t *testing.T) {
	spec := &Spec{
		Name:        "test-spec",
		Description: "desc",
		Credentials: map[string]*CredentialRef{
			"main": {Provider: "openai", Ref: "${OPENAI_KEY}"},
		},
		Credential: "main",
	}
	opts := FullExportWithCredentials()
	serialized, err := spec.Serialize(opts)
	require.NoError(t, err)
	s := string(serialized)
	assert.Contains(t, s, "credentials:")
	assert.Contains(t, s, "credential:")
}

func TestSerialize_CredentialExcludedByDefault(t *testing.T) {
	spec := &Spec{
		Name:        "test-spec",
		Description: "desc",
		Credentials: map[string]*CredentialRef{
			"main": {Provider: "openai"},
		},
		Credential: "main",
	}
	serialized, err := spec.Serialize(nil)
	require.NoError(t, err)
	s := string(serialized)
	// credentials and credential should not appear with default options
	assert.NotContains(t, s, "credential:")
}

func TestSerialize_AgentSkillsExportOptions(t *testing.T) {
	opts := AgentSkillsExportOptions()
	assert.False(t, opts.IncludeExecution)
	assert.False(t, opts.IncludeExtensions)
	assert.False(t, opts.IncludeAgentFields)
	assert.False(t, opts.IncludeContext)
	assert.False(t, opts.IncludeCredentials)
	assert.False(t, opts.IncludeMetadata)
}

func TestSerialize_FullExportWithCredentials(t *testing.T) {
	opts := FullExportWithCredentials()
	assert.True(t, opts.IncludeExecution)
	assert.True(t, opts.IncludeExtensions)
	assert.True(t, opts.IncludeAgentFields)
	assert.True(t, opts.IncludeContext)
	assert.True(t, opts.IncludeCredentials)
	assert.True(t, opts.IncludeMetadata)
}

// =============================================================================
// Extension key conflict guard
// =============================================================================

func TestSerialize_ExtensionKeyConflict(t *testing.T) {
	spec := &Spec{
		Name:        "test-spec",
		Description: "desc",
		Extensions:  map[string]any{"name": "should-not-overwrite"},
	}
	serialized, err := spec.Serialize(nil)
	require.NoError(t, err)
	// The spec's Name field should win over the extension "name" key
	parsed, err := Parse(serialized)
	require.NoError(t, err)
	assert.Equal(t, "test-spec", parsed.Name)
}

// =============================================================================
// Nil spec
// =============================================================================

func TestSerialize_NilSpec(t *testing.T) {
	var s *Spec
	result, err := s.Serialize(nil)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// =============================================================================
// ExportAgentSkill and ExportFull
// =============================================================================

func TestSpec_ExportAgentSkill(t *testing.T) {
	spec := &Spec{
		Name:        "test-spec",
		Description: "desc",
		Inputs:      map[string]*InputDef{"q": {Type: "string"}},
	}
	result, err := spec.ExportAgentSkill()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSpec_ExportFull(t *testing.T) {
	spec := &Spec{
		Name:        "test-spec",
		Description: "desc",
	}
	result, err := spec.ExportFull()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// =============================================================================
// Body preservation
// =============================================================================

func TestSerialize_BodyPreserved(t *testing.T) {
	spec := &Spec{
		Name:        "test-spec",
		Description: "desc",
		Body:        "Hello, world!\nSecond line.",
	}
	serialized, err := spec.Serialize(nil)
	require.NoError(t, err)
	assert.Contains(t, string(serialized), "Hello, world!\nSecond line.")
}

// =============================================================================
// IncludeMetadata
// =============================================================================

func TestSerialize_IncludeMetadataFalse(t *testing.T) {
	spec := &Spec{
		Name:        "test-spec",
		Description: "desc",
		Memory: &MemorySpec{
			Scope: "agent-memory",
		},
		Dispatch: &DispatchSpec{
			TriggerKeywords: []string{"search"},
		},
		Registry: &RegistrySpec{
			Version: "1.0.0",
		},
		Safety: &SafetyConfig{
			Guardrails: GuardrailsEnabled,
		},
	}
	opts := &SerializeOptions{IncludeMetadata: false}
	serialized, err := spec.Serialize(opts)
	require.NoError(t, err)
	s := string(serialized)
	assert.NotContains(t, s, "memory:")
	assert.NotContains(t, s, "dispatch:")
	assert.NotContains(t, s, "registry:")
	assert.NotContains(t, s, "safety:")
}
