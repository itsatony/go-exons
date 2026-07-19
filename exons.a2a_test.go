package exons

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/itsatony/go-exons/a2a"
	"github.com/itsatony/go-exons/execution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// declIface is a declaration-only supported interface, as aigentverse supplies.
func declIface() []a2a.AgentInterface {
	return []a2a.AgentInterface{{
		URL:             "https://reg.example.com/@org/agent",
		ProtocolBinding: A2AProtocolBindingHTTPS,
	}}
}

func baseOpts() *A2ACardOptions {
	return &A2ACardOptions{SupportedInterfaces: declIface()}
}

// goExonsMetaExt returns the go-exons enrichment extension from a card, if present.
func goExonsMetaExt(card *a2a.AgentCard) *a2a.AgentExtension {
	if card.Capabilities == nil {
		return nil
	}
	for i := range card.Capabilities.Extensions {
		if card.Capabilities.Extensions[i].URI == A2AExtensionURIGoExonsMetadata {
			return &card.Capabilities.Extensions[i]
		}
	}
	return nil
}

// --- CompileAgentCard: error + edge cases ---

func TestCompileAgentCard_NilSpec(t *testing.T) {
	var s *Spec
	card, err := s.CompileAgentCard(context.Background(), baseOpts())
	assert.Nil(t, card)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgA2ACardNilSpec)
}

func TestCompileAgentCard_MissingName(t *testing.T) {
	s := &Spec{Description: "test"}
	card, err := s.CompileAgentCard(context.Background(), baseOpts())
	assert.Nil(t, card)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgA2ACardMissingName)
}

// v1.0.1 has no required top-level URL, so nil opts no longer errors — it yields a
// card whose (missing) interfaces Validate would flag, but compilation succeeds.
func TestCompileAgentCard_NilOptsSucceeds(t *testing.T) {
	s := &Spec{Name: "test-agent", Description: "A test agent"}
	card, err := s.CompileAgentCard(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, card)
	assert.Empty(t, card.SupportedInterfaces)
	// A synthesized skill keeps the required skills[] non-empty.
	require.Len(t, card.Skills, 1)
	assert.Equal(t, "test-agent", card.Skills[0].ID)
}

// --- CompileAgentCard: minimal valid + conformance ---

func TestCompileAgentCard_MinimalValidIsConformant(t *testing.T) {
	s := &Spec{Name: "test-agent", Description: "A test agent"}
	card, err := s.CompileAgentCard(context.Background(), baseOpts())
	require.NoError(t, err)
	require.NotNil(t, card)

	assert.Equal(t, "test-agent", card.Name)
	assert.Equal(t, "A test agent", card.Description)
	require.Len(t, card.SupportedInterfaces, 1)
	assert.Equal(t, A2AProtocolBindingHTTPS, card.SupportedInterfaces[0].ProtocolBinding)
	// Per-interface protocol version defaulted.
	assert.Equal(t, A2AProtocolVersionDefault, card.SupportedInterfaces[0].ProtocolVersion)
	assert.Equal(t, A2AVersionDefault, card.Version)
	assert.Nil(t, card.Provider)
	require.NotNil(t, card.Capabilities)
	assert.Nil(t, card.Capabilities.Streaming) // unset ⇒ omitted, not explicit false
	assert.Equal(t, []string{A2AMIMETextPlain}, card.DefaultInputModes)
	assert.Equal(t, []string{A2AMIMETextPlain}, card.DefaultOutputModes)

	// The synthesized card must be A2A-conformant.
	assert.Empty(t, card.Validate())
}

// --- Version resolution ---

func TestCompileAgentCard_VersionResolution(t *testing.T) {
	t.Run("from opts", func(t *testing.T) {
		s := &Spec{Name: "a", Description: "d"}
		opts := baseOpts()
		opts.Version = "2.0.0"
		card, err := s.CompileAgentCard(context.Background(), opts)
		require.NoError(t, err)
		assert.Equal(t, "2.0.0", card.Version)
	})
	t.Run("from registry", func(t *testing.T) {
		s := &Spec{Name: "a", Description: "d", Registry: &RegistrySpec{Version: "3.1.0"}}
		card, err := s.CompileAgentCard(context.Background(), baseOpts())
		require.NoError(t, err)
		assert.Equal(t, "3.1.0", card.Version)
	})
	t.Run("opts overrides registry", func(t *testing.T) {
		s := &Spec{Name: "a", Description: "d", Registry: &RegistrySpec{Version: "3.1.0"}}
		opts := baseOpts()
		opts.Version = "override"
		card, err := s.CompileAgentCard(context.Background(), opts)
		require.NoError(t, err)
		assert.Equal(t, "override", card.Version)
	})
	t.Run("default fallback", func(t *testing.T) {
		s := &Spec{Name: "a", Description: "d"}
		card, err := s.CompileAgentCard(context.Background(), baseOpts())
		require.NoError(t, err)
		assert.Equal(t, A2AVersionDefault, card.Version)
	})
}

// --- Per-interface protocol version ---

func TestCompileAgentCard_ProtocolVersionPerInterface(t *testing.T) {
	s := &Spec{Name: "a", Description: "d"}

	t.Run("explicit on interface preserved", func(t *testing.T) {
		card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
			SupportedInterfaces: []a2a.AgentInterface{{URL: "https://x", ProtocolBinding: "GRPC", ProtocolVersion: "0.3"}},
		})
		require.NoError(t, err)
		assert.Equal(t, "0.3", card.SupportedInterfaces[0].ProtocolVersion)
	})
	t.Run("opts default fills empty", func(t *testing.T) {
		card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
			SupportedInterfaces: []a2a.AgentInterface{{URL: "https://x", ProtocolBinding: "GRPC"}},
			ProtocolVersion:     "1.1",
		})
		require.NoError(t, err)
		assert.Equal(t, "1.1", card.SupportedInterfaces[0].ProtocolVersion)
	})
	t.Run("package default fills empty", func(t *testing.T) {
		card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
			SupportedInterfaces: []a2a.AgentInterface{{URL: "https://x", ProtocolBinding: "GRPC"}},
		})
		require.NoError(t, err)
		assert.Equal(t, A2AProtocolVersionDefault, card.SupportedInterfaces[0].ProtocolVersion)
	})
}

// --- Provider ---

func TestCompileAgentCard_Provider(t *testing.T) {
	s := &Spec{Name: "a", Description: "d"}
	t.Run("from opts", func(t *testing.T) {
		opts := baseOpts()
		opts.ProviderOrganization = "Acme Corp"
		opts.ProviderURL = "https://acme.example.com"
		card, err := s.CompileAgentCard(context.Background(), opts)
		require.NoError(t, err)
		require.NotNil(t, card.Provider)
		assert.Equal(t, "Acme Corp", card.Provider.Organization)
		assert.Equal(t, "https://acme.example.com", card.Provider.URL)
	})
	t.Run("absent", func(t *testing.T) {
		card, err := s.CompileAgentCard(context.Background(), baseOpts())
		require.NoError(t, err)
		assert.Nil(t, card.Provider)
	})
}

// --- Capabilities ---

func TestCompileAgentCard_Capabilities(t *testing.T) {
	t.Run("auto-detect streaming enabled", func(t *testing.T) {
		s := &Spec{Name: "a", Description: "d", Execution: &execution.Config{Streaming: &execution.StreamingConfig{Enabled: true}}}
		card, err := s.CompileAgentCard(context.Background(), baseOpts())
		require.NoError(t, err)
		require.NotNil(t, card.Capabilities)
		require.NotNil(t, card.Capabilities.Streaming)
		assert.True(t, *card.Capabilities.Streaming)
	})
	t.Run("streaming disabled ⇒ nil (omitted)", func(t *testing.T) {
		s := &Spec{Name: "a", Description: "d", Execution: &execution.Config{Streaming: &execution.StreamingConfig{Enabled: false}}}
		card, err := s.CompileAgentCard(context.Background(), baseOpts())
		require.NoError(t, err)
		assert.Nil(t, card.Capabilities.Streaming)
	})
	t.Run("override capabilities preserved + extensions merged", func(t *testing.T) {
		push := true
		s := &Spec{Name: "a", Description: "d"}
		opts := baseOpts()
		opts.Capabilities = &a2a.AgentCapabilities{PushNotifications: &push}
		opts.Extensions = []a2a.AgentExtension{{URI: "https://aiv/ext/provenance", Params: map[string]any{"did": "did:tessera:org:a"}}}
		card, err := s.CompileAgentCard(context.Background(), opts)
		require.NoError(t, err)
		require.NotNil(t, card.Capabilities.PushNotifications)
		assert.True(t, *card.Capabilities.PushNotifications)
		// The provenance extension from opts is present.
		var found bool
		for _, e := range card.Capabilities.Extensions {
			if e.URI == "https://aiv/ext/provenance" {
				found = true
				assert.Equal(t, "did:tessera:org:a", e.Params["did"])
			}
		}
		assert.True(t, found, "opts.Extensions must be merged into capabilities.extensions")
	})
}

// --- Skills ---

func TestCompileAgentCard_Skills(t *testing.T) {
	t.Run("from refs, description falls back to name", func(t *testing.T) {
		s := &Spec{Name: "a", Description: "d", Skills: []SkillRef{{Slug: "web-search"}, {Slug: "summarizer"}}}
		card, err := s.CompileAgentCard(context.Background(), baseOpts())
		require.NoError(t, err)
		require.Len(t, card.Skills, 2)
		assert.Equal(t, "web-search", card.Skills[0].ID)
		assert.Equal(t, "web-search", card.Skills[0].Name)
		assert.Equal(t, "web-search", card.Skills[0].Description) // fallback → name (never blank)
		assert.NotNil(t, card.Skills[0].Tags)                     // always a non-nil slice
	})
	t.Run("with resolver", func(t *testing.T) {
		resolver := NewMapSpecResolver()
		resolver.Add("web-search", &Spec{Name: "web-search", Description: "Searches the web"}, "")
		resolver.Add("summarizer", &Spec{Name: "smart-summarizer", Description: "Summarizes"}, "")
		s := &Spec{Name: "a", Description: "d", Skills: []SkillRef{{Slug: "web-search"}, {Slug: "summarizer"}}}
		opts := baseOpts()
		opts.Resolver = resolver
		card, err := s.CompileAgentCard(context.Background(), opts)
		require.NoError(t, err)
		require.Len(t, card.Skills, 2)
		assert.Equal(t, "Searches the web", card.Skills[0].Description)
		assert.Equal(t, "smart-summarizer", card.Skills[1].Name)
		assert.Equal(t, "Summarizes", card.Skills[1].Description)
	})
	t.Run("resolver failure non-fatal, description falls back", func(t *testing.T) {
		resolver := NewMapSpecResolver()
		resolver.Add("web-search", &Spec{Name: "web-search", Description: "Searches"}, "")
		s := &Spec{Name: "a", Description: "d", Skills: []SkillRef{{Slug: "web-search"}, {Slug: "missing"}}}
		opts := baseOpts()
		opts.Resolver = resolver
		card, err := s.CompileAgentCard(context.Background(), opts)
		require.NoError(t, err)
		require.Len(t, card.Skills, 2)
		assert.Equal(t, "Searches", card.Skills[0].Description)
		assert.Equal(t, "missing", card.Skills[1].Description) // fallback → slug/name
	})
	t.Run("resolver output modes", func(t *testing.T) {
		resolver := NewMapSpecResolver()
		resolver.Add("image-gen", &Spec{Name: "image-gen", Description: "images", Execution: &execution.Config{Modality: ModalityImage}}, "")
		s := &Spec{Name: "a", Description: "d", Skills: []SkillRef{{Slug: "image-gen"}}}
		opts := baseOpts()
		opts.Resolver = resolver
		card, err := s.CompileAgentCard(context.Background(), opts)
		require.NoError(t, err)
		assert.Equal(t, []string{A2AMIMEImagePNG}, card.Skills[0].OutputModes)
	})
	t.Run("no skills synthesizes agent skill", func(t *testing.T) {
		s := &Spec{Name: "solo-agent", Description: "does things"}
		card, err := s.CompileAgentCard(context.Background(), baseOpts())
		require.NoError(t, err)
		require.Len(t, card.Skills, 1)
		assert.Equal(t, "solo-agent", card.Skills[0].ID)
		assert.Equal(t, "solo-agent", card.Skills[0].Name)
		assert.Equal(t, "does things", card.Skills[0].Description)
		assert.NotNil(t, card.Skills[0].Tags)
	})
	t.Run("dispatch keywords become skill tags", func(t *testing.T) {
		s := &Spec{Name: "a", Description: "d", Skills: []SkillRef{{Slug: "web-search"}},
			Dispatch: &DispatchSpec{TriggerKeywords: []string{"research", "search"}}}
		card, err := s.CompileAgentCard(context.Background(), baseOpts())
		require.NoError(t, err)
		assert.Equal(t, []string{"research", "search"}, card.Skills[0].Tags)
	})
}

// --- Input / output modes (helpers unchanged from v0.3) ---

func TestCompileAgentCard_InputModes(t *testing.T) {
	cases := []struct {
		name   string
		inputs map[string]*InputDef
		over   []string
		want   []string
	}{
		{"string", map[string]*InputDef{"q": {Type: SchemaTypeString}}, nil, []string{A2AMIMETextPlain}},
		{"object", map[string]*InputDef{"d": {Type: SchemaTypeObject}}, nil, []string{A2AMIMEApplicationJSON}},
		{"mixed sorted", map[string]*InputDef{"q": {Type: SchemaTypeString}, "d": {Type: SchemaTypeObject}}, nil, []string{A2AMIMEApplicationJSON, A2AMIMETextPlain}},
		{"override", map[string]*InputDef{"q": {Type: SchemaTypeString}}, []string{A2AMIMETextMarkdown}, []string{A2AMIMETextMarkdown}},
		{"none", nil, nil, []string{A2AMIMETextPlain}},
		{"nil def", map[string]*InputDef{"q": nil}, nil, []string{A2AMIMETextPlain}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &Spec{Name: "a", Description: "d", Inputs: tc.inputs}
			opts := baseOpts()
			opts.DefaultInputModes = tc.over
			card, err := s.CompileAgentCard(context.Background(), opts)
			require.NoError(t, err)
			assert.Equal(t, tc.want, card.DefaultInputModes)
		})
	}
}

func TestCompileAgentCard_OutputModes(t *testing.T) {
	cases := []struct {
		name     string
		modality string
		over     []string
		want     []string
	}{
		{"text", ModalityText, nil, []string{A2AMIMETextPlain}},
		{"image", ModalityImage, nil, []string{A2AMIMEImagePNG}},
		{"audio", ModalityAudioSpeech, nil, []string{A2AMIMEAudioMPEG}},
		{"embedding", ModalityEmbedding, nil, []string{A2AMIMEApplicationJSON}},
		{"video", ModalityVideo, nil, []string{A2AMIMEApplicationJSON}},
		{"unknown fallback", "hologram", nil, []string{A2AMIMETextPlain}},
		{"override", ModalityImage, []string{A2AMIMETextMarkdown}, []string{A2AMIMETextMarkdown}},
		{"none", "", nil, []string{A2AMIMETextPlain}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &Spec{Name: "a", Description: "d"}
			if tc.modality != "" {
				s.Execution = &execution.Config{Modality: tc.modality}
			}
			opts := baseOpts()
			opts.DefaultOutputModes = tc.over
			card, err := s.CompileAgentCard(context.Background(), opts)
			require.NoError(t, err)
			assert.Equal(t, tc.want, card.DefaultOutputModes)
		})
	}
}

// --- Security ---

func TestCompileAgentCard_SecurityPassthrough(t *testing.T) {
	s := &Spec{Name: "a", Description: "d"}
	schemes := map[string]any{"bearer": map[string]any{"type": "http", "scheme": "bearer"}}
	reqs := []map[string][]string{{"bearer": {}}}
	opts := baseOpts()
	opts.SecuritySchemes = schemes
	opts.SecurityRequirements = reqs
	card, err := s.CompileAgentCard(context.Background(), opts)
	require.NoError(t, err)
	assert.Equal(t, schemes, card.SecuritySchemes)
	assert.Equal(t, reqs, card.SecurityRequirements)
}

// --- Enrichment extension (safety/dispatch/a2a-ext → go-exons metadata extension) ---

func TestCompileAgentCard_EnrichmentExtension(t *testing.T) {
	t.Run("a2a-prefixed extensions only", func(t *testing.T) {
		s := &Spec{Name: "a", Description: "d", Extensions: map[string]any{
			"a2a.custom": "v", "a2a.contact": "x@y", "other": "ignored"}}
		card, err := s.CompileAgentCard(context.Background(), baseOpts())
		require.NoError(t, err)
		ext := goExonsMetaExt(card)
		require.NotNil(t, ext)
		assert.Equal(t, "v", ext.Params["a2a.custom"])
		assert.Equal(t, "x@y", ext.Params["a2a.contact"])
		_, has := ext.Params["other"]
		assert.False(t, has)
	})
	t.Run("safety + dispatch", func(t *testing.T) {
		s := &Spec{Name: "a", Description: "d",
			Safety:   &SafetyConfig{Guardrails: GuardrailsEnabled, DenyTools: []string{"drop_table"}, RequireConfirmationFor: []string{"send_email"}},
			Dispatch: &DispatchSpec{TriggerDescription: "route research"}}
		card, err := s.CompileAgentCard(context.Background(), baseOpts())
		require.NoError(t, err)
		ext := goExonsMetaExt(card)
		require.NotNil(t, ext)
		assert.Equal(t, GuardrailsEnabled, ext.Params[A2AMetaKeySafetyGuardrails])
		assert.Equal(t, []string{"drop_table"}, ext.Params[A2AMetaKeySafetyDenyTools])
		assert.Equal(t, []string{"send_email"}, ext.Params[A2AMetaKeySafetyConfirmation])
		assert.Equal(t, "route research", ext.Params[A2AMetaKeyDispatchDescription])
	})
	t.Run("no enrichment ⇒ no metadata extension", func(t *testing.T) {
		s := &Spec{Name: "a", Description: "d"}
		card, err := s.CompileAgentCard(context.Background(), baseOpts())
		require.NoError(t, err)
		assert.Nil(t, goExonsMetaExt(card))
	})
}

// --- Full integration ---

func TestCompileAgentCard_FullIntegration(t *testing.T) {
	resolver := NewMapSpecResolver()
	resolver.Add("web-search", &Spec{Name: "web-search", Description: "Searches the web", Execution: &execution.Config{Modality: ModalityText}}, "")
	resolver.Add("image-gen", &Spec{Name: "image-gen", Description: "Generates images", Execution: &execution.Config{Modality: ModalityImage}}, "")

	s := &Spec{
		Name:        "research-agent",
		Description: "AI research assistant",
		Type:        DocumentTypeAgent,
		Execution:   &execution.Config{Provider: ProviderAnthropic, Model: "claude-sonnet-4-5", Modality: ModalityText, Streaming: &execution.StreamingConfig{Enabled: true}},
		Inputs:      map[string]*InputDef{"query": {Type: SchemaTypeString, Required: true}, "context": {Type: SchemaTypeObject}},
		Skills:      []SkillRef{{Slug: "web-search"}, {Slug: "image-gen"}},
		Extensions:  map[string]any{"a2a.team": "platform", "internal": "ignored"},
		Dispatch:    &DispatchSpec{TriggerKeywords: []string{"research", "analyze"}, TriggerDescription: "Route research tasks"},
		Registry:    &RegistrySpec{Version: "2.5.0", Namespace: "acme/research"},
		Safety:      &SafetyConfig{Guardrails: GuardrailsEnabled, DenyTools: []string{"delete_all"}},
	}

	card, err := s.CompileAgentCard(context.Background(), &A2ACardOptions{
		SupportedInterfaces:  declIface(),
		ProviderOrganization: "Acme Corp",
		ProviderURL:          "https://acme.com",
		Resolver:             resolver,
	})
	require.NoError(t, err)
	require.NotNil(t, card)

	assert.Equal(t, "research-agent", card.Name)
	assert.Equal(t, "2.5.0", card.Version)
	require.NotNil(t, card.Provider)
	assert.Equal(t, "Acme Corp", card.Provider.Organization)
	require.NotNil(t, card.Capabilities.Streaming)
	assert.True(t, *card.Capabilities.Streaming)

	require.Len(t, card.Skills, 2)
	assert.Equal(t, "Searches the web", card.Skills[0].Description)
	assert.Equal(t, []string{"research", "analyze"}, card.Skills[0].Tags)
	assert.Equal(t, []string{A2AMIMEImagePNG}, card.Skills[1].OutputModes)

	assert.Equal(t, []string{A2AMIMEApplicationJSON, A2AMIMETextPlain}, card.DefaultInputModes)
	assert.Equal(t, []string{A2AMIMETextPlain}, card.DefaultOutputModes)

	ext := goExonsMetaExt(card)
	require.NotNil(t, ext)
	assert.Equal(t, "platform", ext.Params["a2a.team"])
	assert.Equal(t, GuardrailsEnabled, ext.Params[A2AMetaKeySafetyGuardrails])
	assert.Equal(t, "Route research tasks", ext.Params[A2AMetaKeyDispatchDescription])

	// The fully-populated card is conformant + serializes to valid JSON.
	assert.Empty(t, card.Validate())
	jsonBytes, err := card.ToJSONPretty()
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(jsonBytes, &parsed))
	assert.Contains(t, string(jsonBytes), "research-agent")
}

// --- Helper functions ---

func TestModalityToMIME(t *testing.T) {
	tests := []struct{ modality, expected string }{
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
	tests := []struct{ inputType, expected string }{
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
	assert.Nil(t, sortedStringKeys(nil))
	assert.Nil(t, sortedStringKeys(map[string]bool{}))
	assert.Equal(t, []string{"a"}, sortedStringKeys(map[string]bool{"a": true}))
	assert.Equal(t, []string{"a", "b", "c"}, sortedStringKeys(map[string]bool{"c": true, "a": true, "b": true}))
}

func TestNewA2AError(t *testing.T) {
	err := NewA2AError(ErrMsgA2ACardNilSpec, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgA2ACardNilSpec)

	err = NewA2AError(ErrMsgA2ACardMissingName, assert.AnError)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgA2ACardMissingName)
}
