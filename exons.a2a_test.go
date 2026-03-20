package exons

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/itsatony/go-exons/a2a"
	"github.com/itsatony/go-exons/execution"
	"github.com/itsatony/go-exons/genspec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- CompileAgentCard: error cases ---

func TestCompileAgentCard_NilSpec(t *testing.T) {
	var s *Spec
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{URL: "https://example.com"})
	assert.Nil(t, card)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgA2ACardNilSpec)
}

func TestCompileAgentCard_NilOpts(t *testing.T) {
	s := &Spec{Name: "test-agent", Description: "test"}
	card, err := s.CompileAgentCard(context.Background(), nil)
	assert.Nil(t, card)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgA2ACardMissingURL)
}

func TestCompileAgentCard_MissingURL(t *testing.T) {
	s := &Spec{Name: "test-agent", Description: "test"}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{})
	assert.Nil(t, card)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgA2ACardMissingURL)
}

func TestCompileAgentCard_MissingName(t *testing.T) {
	s := &Spec{Description: "test"}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{URL: "https://example.com"})
	assert.Nil(t, card)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgA2ACardMissingName)
}

// --- CompileAgentCard: minimal valid ---

func TestCompileAgentCard_MinimalValid(t *testing.T) {
	s := &Spec{Name: "test-agent", Description: "A test agent"}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://agents.example.com/test",
	})
	require.NoError(t, err)
	require.NotNil(t, card)

	assert.Equal(t, "test-agent", card.Name)
	assert.Equal(t, "A test agent", card.Description)
	assert.Equal(t, "https://agents.example.com/test", card.URL)
	assert.Equal(t, A2AVersionDefault, card.Version)
	assert.Equal(t, A2AProtocolVersionDefault, card.ProtocolVersion)
	assert.Nil(t, card.Provider)
	assert.NotNil(t, card.Capabilities)
	assert.False(t, card.Capabilities.Streaming)
	assert.Equal(t, []string{A2AMIMETextPlain}, card.DefaultInputModes)
	assert.Equal(t, []string{A2AMIMETextPlain}, card.DefaultOutputModes)
	assert.Nil(t, card.SecuritySchemes)
	assert.Nil(t, card.Security)
	assert.Nil(t, card.Metadata)
	assert.Nil(t, card.Skills)
}

// --- Version resolution ---

func TestCompileAgentCard_VersionFromOpts(t *testing.T) {
	s := &Spec{Name: "test-agent", Description: "test"}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL:     "https://example.com",
		Version: "2.0.0",
	})
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", card.Version)
}

func TestCompileAgentCard_VersionFromGenSpecRegistry(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		GenSpec: &genspec.GenSpec{
			Registry: &genspec.RegistrySpec{
				Version: "3.1.0",
			},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "3.1.0", card.Version)
}

func TestCompileAgentCard_VersionOptsOverridesGenSpec(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		GenSpec: &genspec.GenSpec{
			Registry: &genspec.RegistrySpec{
				Version: "3.1.0",
			},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL:     "https://example.com",
		Version: "override",
	})
	require.NoError(t, err)
	assert.Equal(t, "override", card.Version)
}

func TestCompileAgentCard_VersionDefaultFallback(t *testing.T) {
	s := &Spec{Name: "test-agent", Description: "test"}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, A2AVersionDefault, card.Version)
}

// --- Protocol version ---

func TestCompileAgentCard_ProtocolVersionOverride(t *testing.T) {
	s := &Spec{Name: "test-agent", Description: "test"}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL:             "https://example.com",
		ProtocolVersion: "0.4.0",
	})
	require.NoError(t, err)
	assert.Equal(t, "0.4.0", card.ProtocolVersion)
}

func TestCompileAgentCard_ProtocolVersionDefault(t *testing.T) {
	s := &Spec{Name: "test-agent", Description: "test"}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, A2AProtocolVersionDefault, card.ProtocolVersion)
}

// --- Provider ---

func TestCompileAgentCard_ProviderFromOpts(t *testing.T) {
	s := &Spec{Name: "test-agent", Description: "test"}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL:                  "https://example.com",
		ProviderOrganization: "Acme Corp",
		ProviderURL:          "https://acme.example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, card.Provider)
	assert.Equal(t, "Acme Corp", card.Provider.Organization)
	assert.Equal(t, "https://acme.example.com", card.Provider.URL)
}

func TestCompileAgentCard_NoProvider(t *testing.T) {
	s := &Spec{Name: "test-agent", Description: "test"}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	assert.Nil(t, card.Provider)
}

// --- Capabilities ---

func TestCompileAgentCard_CapabilitiesAutoDetectStreamingEnabled(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Execution: &execution.Config{
			Streaming: &execution.StreamingConfig{Enabled: true},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, card.Capabilities)
	assert.True(t, card.Capabilities.Streaming)
}

func TestCompileAgentCard_CapabilitiesAutoDetectStreamingDisabled(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Execution: &execution.Config{
			Streaming: &execution.StreamingConfig{Enabled: false},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, card.Capabilities)
	assert.False(t, card.Capabilities.Streaming)
}

func TestCompileAgentCard_CapabilitiesOverride(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Execution: &execution.Config{
			Streaming: &execution.StreamingConfig{Enabled: true},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL:          "https://example.com",
		Capabilities: &a2a.Capabilities{PushNotifications: true},
	})
	require.NoError(t, err)
	require.NotNil(t, card.Capabilities)
	assert.False(t, card.Capabilities.Streaming)
	assert.True(t, card.Capabilities.PushNotifications)
}

func TestCompileAgentCard_CapabilitiesNoExecution(t *testing.T) {
	s := &Spec{Name: "test-agent", Description: "test"}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, card.Capabilities)
	assert.False(t, card.Capabilities.Streaming)
	assert.False(t, card.Capabilities.PushNotifications)
}

// --- Skills ---

func TestCompileAgentCard_SkillsFromSkillRefs(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Skills: []SkillRef{
			{Slug: "web-search"},
			{Slug: "summarizer"},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	require.Len(t, card.Skills, 2)
	assert.Equal(t, "web-search", card.Skills[0].ID)
	assert.Equal(t, "web-search", card.Skills[0].Name)
	assert.Empty(t, card.Skills[0].Description)
	assert.Equal(t, "summarizer", card.Skills[1].ID)
	assert.Equal(t, "summarizer", card.Skills[1].Name)
}

func TestCompileAgentCard_SkillsWithResolver(t *testing.T) {
	resolver := NewMapSpecResolver()
	resolver.Add("web-search", &Spec{
		Name:        "web-search",
		Description: "Searches the web for information",
	}, "")
	resolver.Add("summarizer", &Spec{
		Name:        "smart-summarizer",
		Description: "Summarizes long content",
	}, "")

	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Skills: []SkillRef{
			{Slug: "web-search"},
			{Slug: "summarizer"},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL:      "https://example.com",
		Resolver: resolver,
	})
	require.NoError(t, err)
	require.Len(t, card.Skills, 2)

	assert.Equal(t, "web-search", card.Skills[0].ID)
	assert.Equal(t, "web-search", card.Skills[0].Name)
	assert.Equal(t, "Searches the web for information", card.Skills[0].Description)

	assert.Equal(t, "summarizer", card.Skills[1].ID)
	assert.Equal(t, "smart-summarizer", card.Skills[1].Name)
	assert.Equal(t, "Summarizes long content", card.Skills[1].Description)
}

func TestCompileAgentCard_SkillsResolverFailureNonFatal(t *testing.T) {
	resolver := NewMapSpecResolver()
	// Only add one skill — the other will fail resolution
	resolver.Add("web-search", &Spec{
		Name:        "web-search",
		Description: "Searches the web",
	}, "")

	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Skills: []SkillRef{
			{Slug: "web-search"},
			{Slug: "missing-skill"},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL:      "https://example.com",
		Resolver: resolver,
	})
	require.NoError(t, err)
	require.Len(t, card.Skills, 2)

	assert.Equal(t, "Searches the web", card.Skills[0].Description)
	assert.Empty(t, card.Skills[1].Description)
	assert.Equal(t, "missing-skill", card.Skills[1].Name)
}

func TestCompileAgentCard_SkillsNilResolver(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Skills: []SkillRef{
			{Slug: "web-search"},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	require.Len(t, card.Skills, 1)
	assert.Empty(t, card.Skills[0].Description)
}

func TestCompileAgentCard_SkillsResolverOutputModes(t *testing.T) {
	resolver := NewMapSpecResolver()
	resolver.Add("image-gen", &Spec{
		Name:        "image-gen",
		Description: "Generates images",
		Execution: &execution.Config{
			Modality: ModalityImage,
		},
	}, "")

	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Skills: []SkillRef{
			{Slug: "image-gen"},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL:      "https://example.com",
		Resolver: resolver,
	})
	require.NoError(t, err)
	require.Len(t, card.Skills, 1)
	assert.Equal(t, []string{A2AMIMEImagePNG}, card.Skills[0].OutputModes)
}

func TestCompileAgentCard_NoSkills(t *testing.T) {
	s := &Spec{Name: "test-agent", Description: "test"}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	assert.Nil(t, card.Skills)
}

// --- GenSpec dispatch keywords as skill tags ---

func TestCompileAgentCard_DispatchKeywordsAsSkillTags(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Skills: []SkillRef{
			{Slug: "web-search"},
			{Slug: "summarizer"},
		},
		GenSpec: &genspec.GenSpec{
			Dispatch: &genspec.DispatchSpec{
				TriggerKeywords: []string{"research", "search", "find"},
			},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	require.Len(t, card.Skills, 2)

	assert.Equal(t, []string{"research", "search", "find"}, card.Skills[0].Tags)
	assert.Equal(t, []string{"research", "search", "find"}, card.Skills[1].Tags)
}

func TestCompileAgentCard_NoDispatchKeywords(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Skills:      []SkillRef{{Slug: "web-search"}},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	require.Len(t, card.Skills, 1)
	assert.Nil(t, card.Skills[0].Tags)
}

// --- Input modes ---

func TestCompileAgentCard_InputModesFromStringInputs(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Inputs: map[string]*InputDef{
			"query": {Type: SchemaTypeString},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{A2AMIMETextPlain}, card.DefaultInputModes)
}

func TestCompileAgentCard_InputModesFromObjectInputs(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Inputs: map[string]*InputDef{
			"data": {Type: SchemaTypeObject},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{A2AMIMEApplicationJSON}, card.DefaultInputModes)
}

func TestCompileAgentCard_InputModesMixed(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Inputs: map[string]*InputDef{
			"query": {Type: SchemaTypeString},
			"data":  {Type: SchemaTypeObject},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	// Sorted: application/json, text/plain
	assert.Equal(t, []string{A2AMIMEApplicationJSON, A2AMIMETextPlain}, card.DefaultInputModes)
}

func TestCompileAgentCard_InputModesOverride(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Inputs: map[string]*InputDef{
			"query": {Type: SchemaTypeString},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL:               "https://example.com",
		DefaultInputModes: []string{A2AMIMETextMarkdown},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{A2AMIMETextMarkdown}, card.DefaultInputModes)
}

func TestCompileAgentCard_InputModesDefaultNoInputs(t *testing.T) {
	s := &Spec{Name: "test-agent", Description: "test"}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{A2AMIMETextPlain}, card.DefaultInputModes)
}

func TestCompileAgentCard_InputModesNilDefs(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Inputs: map[string]*InputDef{
			"query": nil,
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{A2AMIMETextPlain}, card.DefaultInputModes)
}

// --- Output modes ---

func TestCompileAgentCard_OutputModesFromTextModality(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Execution: &execution.Config{
			Modality: ModalityText,
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{A2AMIMETextPlain}, card.DefaultOutputModes)
}

func TestCompileAgentCard_OutputModesFromImageModality(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Execution: &execution.Config{
			Modality: ModalityImage,
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{A2AMIMEImagePNG}, card.DefaultOutputModes)
}

func TestCompileAgentCard_OutputModesFromAudioModality(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Execution: &execution.Config{
			Modality: ModalityAudioSpeech,
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{A2AMIMEAudioMPEG}, card.DefaultOutputModes)
}

func TestCompileAgentCard_OutputModesFromEmbeddingModality(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Execution: &execution.Config{
			Modality: ModalityEmbedding,
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{A2AMIMEApplicationJSON}, card.DefaultOutputModes)
}

func TestCompileAgentCard_OutputModesFromVideoModality(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Execution: &execution.Config{
			Modality: ModalityVideo,
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{A2AMIMEApplicationJSON}, card.DefaultOutputModes)
}

func TestCompileAgentCard_OutputModesUnknownModalityFallback(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Execution: &execution.Config{
			Modality: "hologram",
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{A2AMIMETextPlain}, card.DefaultOutputModes)
}

func TestCompileAgentCard_OutputModesOverride(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Execution: &execution.Config{
			Modality: ModalityImage,
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL:                "https://example.com",
		DefaultOutputModes: []string{A2AMIMETextMarkdown},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{A2AMIMETextMarkdown}, card.DefaultOutputModes)
}

func TestCompileAgentCard_OutputModesDefaultNoModality(t *testing.T) {
	s := &Spec{Name: "test-agent", Description: "test"}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{A2AMIMETextPlain}, card.DefaultOutputModes)
}

// --- Security ---

func TestCompileAgentCard_SecurityPassthrough(t *testing.T) {
	s := &Spec{Name: "test-agent", Description: "test"}
	schemes := map[string]any{
		"bearer": map[string]any{
			"type":   "http",
			"scheme": "bearer",
		},
	}
	security := []map[string][]string{
		{"bearer": {}},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL:             "https://example.com",
		SecuritySchemes: schemes,
		Security:        security,
	})
	require.NoError(t, err)
	assert.Equal(t, schemes, card.SecuritySchemes)
	assert.Equal(t, security, card.Security)
}

// --- Metadata ---

func TestCompileAgentCard_MetadataFromA2AExtensions(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Extensions: map[string]any{
			"a2a.custom_field":  "custom_value",
			"a2a.contact_email": "agent@example.com",
			"other_extension":   "ignored",
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, card.Metadata)
	assert.Equal(t, "custom_value", card.Metadata["a2a.custom_field"])
	assert.Equal(t, "agent@example.com", card.Metadata["a2a.contact_email"])
	_, hasOther := card.Metadata["other_extension"]
	assert.False(t, hasOther)
}

func TestCompileAgentCard_MetadataFromGenSpecSafety(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		GenSpec: &genspec.GenSpec{
			Safety: &genspec.SafetyConfig{
				Guardrails:             genspec.GuardrailsEnabled,
				DenyTools:              []string{"delete_all", "drop_table"},
				RequireConfirmationFor: []string{"send_email"},
			},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, card.Metadata)
	assert.Equal(t, genspec.GuardrailsEnabled, card.Metadata[A2AMetaKeySafetyGuardrails])
	assert.Equal(t, []string{"delete_all", "drop_table"}, card.Metadata[A2AMetaKeySafetyDenyTools])
	assert.Equal(t, []string{"send_email"}, card.Metadata[A2AMetaKeySafetyConfirmation])
}

func TestCompileAgentCard_MetadataFromGenSpecVersion(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		GenSpec: &genspec.GenSpec{
			Version: "1",
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, card.Metadata)
	assert.Equal(t, "1", card.Metadata[A2AMetaKeyGenSpecVersion])
}

func TestCompileAgentCard_MetadataFromGenSpecDispatchDescription(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		GenSpec: &genspec.GenSpec{
			Dispatch: &genspec.DispatchSpec{
				TriggerDescription: "Route tasks about research and web search",
			},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, card.Metadata)
	assert.Equal(t, "Route tasks about research and web search",
		card.Metadata[A2AMetaKeyDispatchDescription])
}

func TestCompileAgentCard_MetadataDenyToolsOnly(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		GenSpec: &genspec.GenSpec{
			Safety: &genspec.SafetyConfig{
				DenyTools: []string{"rm_rf"},
			},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, card.Metadata)
	assert.Equal(t, []string{"rm_rf"}, card.Metadata[A2AMetaKeySafetyDenyTools])
	_, hasGuardrails := card.Metadata[A2AMetaKeySafetyGuardrails]
	assert.False(t, hasGuardrails)
}

func TestCompileAgentCard_MetadataRequireConfirmationOnly(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		GenSpec: &genspec.GenSpec{
			Safety: &genspec.SafetyConfig{
				RequireConfirmationFor: []string{"send_email", "delete_user"},
			},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, card.Metadata)
	assert.Equal(t, []string{"send_email", "delete_user"}, card.Metadata[A2AMetaKeySafetyConfirmation])
	_, hasGuardrails := card.Metadata[A2AMetaKeySafetyGuardrails]
	assert.False(t, hasGuardrails)
	_, hasDenyTools := card.Metadata[A2AMetaKeySafetyDenyTools]
	assert.False(t, hasDenyTools)
}

func TestCompileAgentCard_MetadataCombinesExtensionsAndGenSpec(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Extensions: map[string]any{
			"a2a.team": "platform",
		},
		GenSpec: &genspec.GenSpec{
			Version: "1",
			Safety: &genspec.SafetyConfig{
				Guardrails: genspec.GuardrailsEnabled,
			},
		},
	}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, card.Metadata)
	assert.Equal(t, "platform", card.Metadata["a2a.team"])
	assert.Equal(t, genspec.GuardrailsEnabled, card.Metadata[A2AMetaKeySafetyGuardrails])
	assert.Equal(t, "1", card.Metadata[A2AMetaKeyGenSpecVersion])
}

func TestCompileAgentCard_MetadataNilWithNoExtensionsOrGenSpec(t *testing.T) {
	s := &Spec{Name: "test-agent", Description: "test"}
	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL: "https://example.com",
	})
	require.NoError(t, err)
	assert.Nil(t, card.Metadata)
}

// --- JSON serialization ---

func TestAgentCard_ToJSON(t *testing.T) {
	card := &a2a.AgentCard{
		Name:            "test-agent",
		URL:             "https://example.com",
		ProtocolVersion: A2AProtocolVersionDefault,
	}
	data, err := card.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, string(data), `"name":"test-agent"`)
	assert.Contains(t, string(data), `"url":"https://example.com"`)

	// Verify it's valid JSON
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))
}

func TestAgentCard_ToJSONPretty(t *testing.T) {
	card := &a2a.AgentCard{
		Name:            "test-agent",
		URL:             "https://example.com",
		ProtocolVersion: A2AProtocolVersionDefault,
	}
	data, err := card.ToJSONPretty()
	require.NoError(t, err)
	assert.Contains(t, string(data), "\"name\": \"test-agent\"")
	// Indented output should have newlines
	assert.Contains(t, string(data), "\n")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))
}

func TestAgentCard_ToJSON_Nil(t *testing.T) {
	var card *a2a.AgentCard
	data, err := card.ToJSON()
	assert.NoError(t, err)
	assert.Nil(t, data)
}

func TestAgentCard_ToJSONPretty_Nil(t *testing.T) {
	var card *a2a.AgentCard
	data, err := card.ToJSONPretty()
	assert.NoError(t, err)
	assert.Nil(t, data)
}

// --- Helper functions ---

func TestModalityToMIME(t *testing.T) {
	tests := []struct {
		modality string
		expected string
	}{
		{ModalityText, A2AMIMETextPlain},
		{ModalityImage, A2AMIMEImagePNG},
		{ModalityImageEdit, A2AMIMEImagePNG},
		{ModalityAudioSpeech, A2AMIMEAudioMPEG},
		{ModalityAudioTranscription, A2AMIMEAudioMPEG},
		{ModalityMusic, A2AMIMEAudioMPEG},
		{ModalitySoundEffects, A2AMIMEAudioMPEG},
		{ModalityEmbedding, A2AMIMEApplicationJSON},
		{ModalityVideo, A2AMIMEApplicationJSON},
		{"unknown", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.modality, func(t *testing.T) {
			assert.Equal(t, tt.expected, modalityToMIME(tt.modality))
		})
	}
}

func TestInputTypeToMIME(t *testing.T) {
	tests := []struct {
		inputType string
		expected  string
	}{
		{SchemaTypeString, A2AMIMETextPlain},
		{SchemaTypeObject, A2AMIMEApplicationJSON},
		{SchemaTypeArray, A2AMIMEApplicationJSON},
		{SchemaTypeNumber, A2AMIMETextPlain},
		{SchemaTypeBoolean, A2AMIMETextPlain},
		{"unknown", A2AMIMETextPlain},
		{"", A2AMIMETextPlain},
	}
	for _, tt := range tests {
		t.Run(tt.inputType, func(t *testing.T) {
			assert.Equal(t, tt.expected, inputTypeToMIME(tt.inputType))
		})
	}
}

func TestSortedStringKeys(t *testing.T) {
	t.Run("nil map", func(t *testing.T) {
		assert.Nil(t, sortedStringKeys(nil))
	})
	t.Run("empty map", func(t *testing.T) {
		assert.Nil(t, sortedStringKeys(map[string]bool{}))
	})
	t.Run("single key", func(t *testing.T) {
		assert.Equal(t, []string{"a"}, sortedStringKeys(map[string]bool{"a": true}))
	})
	t.Run("multiple keys sorted", func(t *testing.T) {
		m := map[string]bool{"c": true, "a": true, "b": true}
		assert.Equal(t, []string{"a", "b", "c"}, sortedStringKeys(m))
	})
}

// --- Full integration test ---

func TestCompileAgentCard_FullIntegration(t *testing.T) {
	resolver := NewMapSpecResolver()
	resolver.Add("web-search", &Spec{
		Name:        "web-search",
		Description: "Searches the web for current information",
		Execution: &execution.Config{
			Modality: ModalityText,
		},
	}, "")
	resolver.Add("image-gen", &Spec{
		Name:        "image-gen",
		Description: "Generates images from text prompts",
		Execution: &execution.Config{
			Modality: ModalityImage,
		},
	}, "")

	s := &Spec{
		Name:        "research-agent",
		Description: "AI research assistant with multi-modal capabilities",
		Type:        DocumentTypeAgent,
		Execution: &execution.Config{
			Provider:  ProviderAnthropic,
			Model:     "claude-sonnet-4-5",
			Modality:  ModalityText,
			Streaming: &execution.StreamingConfig{Enabled: true},
		},
		Inputs: map[string]*InputDef{
			"query":   {Type: SchemaTypeString, Required: true},
			"context": {Type: SchemaTypeObject},
		},
		Skills: []SkillRef{
			{Slug: "web-search", Injection: string(SkillInjectionSystemPrompt)},
			{Slug: "image-gen"},
		},
		Extensions: map[string]any{
			"a2a.team":    "platform",
			"a2a.version": "beta",
			"internal":    "ignored",
		},
		GenSpec: &genspec.GenSpec{
			Version: "1",
			Dispatch: &genspec.DispatchSpec{
				TriggerKeywords:    []string{"research", "search", "analyze"},
				TriggerDescription: "Route research and analysis tasks here",
			},
			Registry: &genspec.RegistrySpec{
				Version:   "2.5.0",
				Namespace: "acme/research",
			},
			Safety: &genspec.SafetyConfig{
				Guardrails: genspec.GuardrailsEnabled,
				DenyTools:  []string{"delete_all"},
			},
		},
	}

	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		URL:                  "https://agents.acme.com/research",
		ProviderOrganization: "Acme Corp",
		ProviderURL:          "https://acme.com",
		Resolver:             resolver,
	})
	require.NoError(t, err)
	require.NotNil(t, card)

	// Identity
	assert.Equal(t, "research-agent", card.Name)
	assert.Equal(t, "AI research assistant with multi-modal capabilities", card.Description)
	assert.Equal(t, "https://agents.acme.com/research", card.URL)

	// Version from GenSpec registry
	assert.Equal(t, "2.5.0", card.Version)
	assert.Equal(t, A2AProtocolVersionDefault, card.ProtocolVersion)

	// Provider
	require.NotNil(t, card.Provider)
	assert.Equal(t, "Acme Corp", card.Provider.Organization)
	assert.Equal(t, "https://acme.com", card.Provider.URL)

	// Capabilities (streaming auto-detected)
	require.NotNil(t, card.Capabilities)
	assert.True(t, card.Capabilities.Streaming)

	// Skills with resolved descriptions and dispatch tags
	require.Len(t, card.Skills, 2)
	assert.Equal(t, "web-search", card.Skills[0].ID)
	assert.Equal(t, "Searches the web for current information", card.Skills[0].Description)
	assert.Equal(t, []string{"research", "search", "analyze"}, card.Skills[0].Tags)
	assert.Equal(t, []string{A2AMIMETextPlain}, card.Skills[0].OutputModes)

	assert.Equal(t, "image-gen", card.Skills[1].ID)
	assert.Equal(t, "Generates images from text prompts", card.Skills[1].Description)
	assert.Equal(t, []string{A2AMIMEImagePNG}, card.Skills[1].OutputModes)

	// Input modes (sorted: application/json, text/plain)
	assert.Equal(t, []string{A2AMIMEApplicationJSON, A2AMIMETextPlain}, card.DefaultInputModes)

	// Output modes (text modality)
	assert.Equal(t, []string{A2AMIMETextPlain}, card.DefaultOutputModes)

	// Metadata
	require.NotNil(t, card.Metadata)
	assert.Equal(t, "platform", card.Metadata["a2a.team"])
	assert.Equal(t, "beta", card.Metadata["a2a.version"])
	_, hasInternal := card.Metadata["internal"]
	assert.False(t, hasInternal)
	assert.Equal(t, genspec.GuardrailsEnabled, card.Metadata[A2AMetaKeySafetyGuardrails])
	assert.Equal(t, []string{"delete_all"}, card.Metadata[A2AMetaKeySafetyDenyTools])
	assert.Equal(t, "1", card.Metadata[A2AMetaKeyGenSpecVersion])
	assert.Equal(t, "Route research and analysis tasks here", card.Metadata[A2AMetaKeyDispatchDescription])

	// JSON output
	jsonBytes, err := card.ToJSONPretty()
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), "research-agent")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(jsonBytes, &parsed))
}

// --- NewA2AError ---

func TestNewA2AError(t *testing.T) {
	t.Run("without cause", func(t *testing.T) {
		err := NewA2AError(ErrMsgA2ACardNilSpec, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgA2ACardNilSpec)
	})

	t.Run("with cause", func(t *testing.T) {
		cause := assert.AnError
		err := NewA2AError(ErrMsgA2ACardMissingURL, cause)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgA2ACardMissingURL)
	})
}
