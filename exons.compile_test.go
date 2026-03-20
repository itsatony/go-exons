package exons

import (
	"context"
	"strings"
	"testing"

	"github.com/itsatony/go-exons/execution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// CompileOptions — functional options, nil opts, default engine
// =============================================================================

func TestCompileOptions_NewCompileOptions_Empty(t *testing.T) {
	opts := NewCompileOptions()
	require.NotNil(t, opts)
	assert.Nil(t, opts.Resolver)
	assert.Nil(t, opts.Engine)
	assert.Equal(t, CatalogFormat(""), opts.SkillsCatalogFormat)
	assert.Equal(t, CatalogFormat(""), opts.ToolsCatalogFormat)
}

func TestCompileOptions_WithResolver(t *testing.T) {
	resolver := NewMapSpecResolver()
	opts := NewCompileOptions(WithResolver(resolver))
	require.NotNil(t, opts)
	assert.Equal(t, resolver, opts.Resolver)
}

func TestCompileOptions_WithCompileEngine(t *testing.T) {
	engine := MustNew()
	opts := NewCompileOptions(WithCompileEngine(engine))
	require.NotNil(t, opts)
	assert.Equal(t, engine, opts.Engine)
}

func TestCompileOptions_WithSkillsCatalogFormat(t *testing.T) {
	opts := NewCompileOptions(WithSkillsCatalogFormat(CatalogFormatDetailed))
	require.NotNil(t, opts)
	assert.Equal(t, CatalogFormatDetailed, opts.SkillsCatalogFormat)
}

func TestCompileOptions_WithToolsCatalogFormat(t *testing.T) {
	opts := NewCompileOptions(WithToolsCatalogFormat(CatalogFormatFunctionCalling))
	require.NotNil(t, opts)
	assert.Equal(t, CatalogFormatFunctionCalling, opts.ToolsCatalogFormat)
}

func TestCompileOptions_MultipleOptions(t *testing.T) {
	resolver := NewMapSpecResolver()
	engine := MustNew()
	opts := NewCompileOptions(
		WithResolver(resolver),
		WithCompileEngine(engine),
		WithSkillsCatalogFormat(CatalogFormatCompact),
		WithToolsCatalogFormat(CatalogFormatDetailed),
	)
	require.NotNil(t, opts)
	assert.Equal(t, resolver, opts.Resolver)
	assert.Equal(t, engine, opts.Engine)
	assert.Equal(t, CatalogFormatCompact, opts.SkillsCatalogFormat)
	assert.Equal(t, CatalogFormatDetailed, opts.ToolsCatalogFormat)
}

func TestCompileEngine_NilOpts(t *testing.T) {
	engine := compileEngine(nil)
	require.NotNil(t, engine, "compileEngine should create a default engine when opts is nil")
}

func TestCompileEngine_NilEngine(t *testing.T) {
	opts := &CompileOptions{Engine: nil}
	engine := compileEngine(opts)
	require.NotNil(t, engine, "compileEngine should create a default engine when opts.Engine is nil")
}

func TestCompileEngine_ProvidedEngine(t *testing.T) {
	provided := MustNew()
	opts := &CompileOptions{Engine: provided}
	engine := compileEngine(opts)
	assert.Equal(t, provided, engine, "compileEngine should return the provided engine")
}

// =============================================================================
// Spec.Compile — nil spec, empty body, variable resolution, context injection
// =============================================================================

func TestCompile_NilSpec(t *testing.T) {
	var s *Spec
	result, err := s.Compile(context.Background(), nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgCompileNotAgent)
	assert.Empty(t, result)
}

func TestCompile_EmptyBody(t *testing.T) {
	s := &Spec{
		Name: "empty-body",
		Body: "",
	}
	result, err := s.Compile(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestCompile_PlainTextBody(t *testing.T) {
	s := &Spec{
		Name: "plain-body",
		Body: "Hello, world!",
	}
	result, err := s.Compile(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "Hello, world!", result)
}

func TestCompile_VariableResolution(t *testing.T) {
	s := &Spec{
		Name: "var-body",
		Body: `Hello, {~exons.var name="name" default="World" /~}!`,
	}
	input := map[string]any{
		"name": "Alice",
	}
	result, err := s.Compile(context.Background(), input, nil)
	require.NoError(t, err)
	assert.Equal(t, "Hello, Alice!", result)
}

func TestCompile_VariableWithDefault(t *testing.T) {
	s := &Spec{
		Name: "var-default",
		Body: `Hello, {~exons.var name="name" default="World" /~}!`,
	}
	result, err := s.Compile(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", result)
}

func TestCompile_ContextInjection(t *testing.T) {
	s := &Spec{
		Name: "ctx-body",
		Body: `Company: {~exons.var name="company" default="N/A" /~}`,
		Context: map[string]any{
			"company": "Acme Corp",
		},
	}
	result, err := s.Compile(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "Company: Acme Corp", result)
}

func TestCompile_InputOverridesContext(t *testing.T) {
	// Input is flattened at top level AND under "input" key.
	// Context values do NOT overwrite existing top-level keys from input.
	s := &Spec{
		Name: "override-body",
		Body: `Value: {~exons.var name="key" /~}`,
		Context: map[string]any{
			"key": "from-context",
		},
	}
	input := map[string]any{
		"key": "from-input",
	}
	result, err := s.Compile(context.Background(), input, nil)
	require.NoError(t, err)
	assert.Equal(t, "Value: from-input", result)
}

func TestCompile_NilOpts(t *testing.T) {
	s := &Spec{
		Name: "nil-opts",
		Body: "Static content",
	}
	result, err := s.Compile(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "Static content", result)
}

func TestCompile_WithProvidedEngine(t *testing.T) {
	engine := MustNew()
	s := &Spec{
		Name: "with-engine",
		Body: `{~exons.var name="x" default="ok" /~}`,
	}
	opts := NewCompileOptions(WithCompileEngine(engine))
	result, err := s.Compile(context.Background(), nil, opts)
	require.NoError(t, err)
	assert.Equal(t, "ok", result)
}

// =============================================================================
// Spec.CompileAgent — full agent compilation tests
// =============================================================================

func TestCompileAgent_NilSpec(t *testing.T) {
	var s *Spec
	compiled, err := s.CompileAgent(context.Background(), nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgCompileNotAgent)
	assert.Nil(t, compiled)
}

func TestCompileAgent_NonAgentType(t *testing.T) {
	s := &Spec{
		Name:      "not-agent",
		Type:      DocumentTypeSkill,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Body:      "Some body",
	}
	compiled, err := s.CompileAgent(context.Background(), nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgNotAnAgent)
	assert.Nil(t, compiled)
}

func TestCompileAgent_MissingExecutionConfig(t *testing.T) {
	s := &Spec{
		Name: "no-exec",
		Type: DocumentTypeAgent,
		Body: "Some body",
	}
	compiled, err := s.CompileAgent(context.Background(), nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgNoExecutionConfig)
	assert.Nil(t, compiled)
}

func TestCompileAgent_NoBodyOrMessages(t *testing.T) {
	s := &Spec{
		Name:      "no-body-msgs",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
	}
	compiled, err := s.CompileAgent(context.Background(), nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgAgentNoBodyOrMessages)
	assert.Nil(t, compiled)
}

func TestCompileAgent_BodyOnly_DefaultMessages(t *testing.T) {
	s := &Spec{
		Name:      "body-only",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Body:      "You are a helpful assistant.",
	}
	compiled, err := s.CompileAgent(context.Background(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	// Body becomes system message
	require.Len(t, compiled.Messages, 1)
	assert.Equal(t, RoleSystem, compiled.Messages[0].Role)
	assert.Equal(t, "You are a helpful assistant.", compiled.Messages[0].Content)
}

func TestCompileAgent_BodyOnly_WithQueryInput(t *testing.T) {
	s := &Spec{
		Name:      "body-query",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Body:      "You are a helpful assistant.",
	}
	input := map[string]any{
		"query": "What is the weather?",
	}
	compiled, err := s.CompileAgent(context.Background(), input, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	// Body becomes system message, query becomes user message
	require.Len(t, compiled.Messages, 2)
	assert.Equal(t, RoleSystem, compiled.Messages[0].Role)
	assert.Equal(t, "You are a helpful assistant.", compiled.Messages[0].Content)
	assert.Equal(t, RoleUser, compiled.Messages[1].Role)
	assert.Equal(t, "What is the weather?", compiled.Messages[1].Content)
}

func TestCompileAgent_BodyOnly_WithMessageInput(t *testing.T) {
	s := &Spec{
		Name:      "body-message",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Body:      "You are a helpful assistant.",
	}
	input := map[string]any{
		"message": "Hello there!",
	}
	compiled, err := s.CompileAgent(context.Background(), input, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	require.Len(t, compiled.Messages, 2)
	assert.Equal(t, RoleSystem, compiled.Messages[0].Role)
	assert.Equal(t, RoleUser, compiled.Messages[1].Role)
	assert.Equal(t, "Hello there!", compiled.Messages[1].Content)
}

func TestCompileAgent_BodyOnly_QueryTakesPrecedenceOverMessage(t *testing.T) {
	s := &Spec{
		Name:      "body-query-precedence",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Body:      "System prompt.",
	}
	input := map[string]any{
		"query":   "Query wins",
		"message": "Message loses",
	}
	compiled, err := s.CompileAgent(context.Background(), input, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	// "query" is checked first in buildDefaultMessages
	require.Len(t, compiled.Messages, 2)
	assert.Equal(t, RoleUser, compiled.Messages[1].Role)
	assert.Equal(t, "Query wins", compiled.Messages[1].Content)
}

func TestCompileAgent_ExplicitMessages(t *testing.T) {
	s := &Spec{
		Name:      "explicit-msgs",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: "You are a test assistant."},
			{Role: RoleUser, Content: "Hello"},
		},
	}
	compiled, err := s.CompileAgent(context.Background(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	require.Len(t, compiled.Messages, 2)
	assert.Equal(t, RoleSystem, compiled.Messages[0].Role)
	assert.Equal(t, "You are a test assistant.", compiled.Messages[0].Content)
	assert.Equal(t, RoleUser, compiled.Messages[1].Role)
	assert.Equal(t, "Hello", compiled.Messages[1].Content)
}

func TestCompileAgent_MessagesWithTemplateResolution(t *testing.T) {
	s := &Spec{
		Name:      "msg-templates",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: `You are an assistant for {~exons.var name="company" default="Acme" /~}.`},
			{Role: RoleUser, Content: `{~exons.var name="input.query" default="hello" /~}`},
		},
	}
	input := map[string]any{
		"query": "What is 2+2?",
	}
	compiled, err := s.CompileAgent(context.Background(), input, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	require.Len(t, compiled.Messages, 2)
	assert.Equal(t, "You are an assistant for Acme.", compiled.Messages[0].Content)
	assert.Equal(t, "What is 2+2?", compiled.Messages[1].Content)
}

func TestCompileAgent_ContextInjectionIntoMessages(t *testing.T) {
	s := &Spec{
		Name:      "ctx-msgs",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Context: map[string]any{
			"company": "TestCo",
		},
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: `Hello from {~exons.var name="company" /~}`},
		},
	}
	compiled, err := s.CompileAgent(context.Background(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	require.Len(t, compiled.Messages, 1)
	assert.Equal(t, "Hello from TestCo", compiled.Messages[0].Content)
}

func TestCompileAgent_EmptyMessageContentSkipped(t *testing.T) {
	s := &Spec{
		Name:      "skip-empty",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: "Valid system message"},
			{Role: RoleUser, Content: "   "}, // becomes empty after trim
			{Role: RoleUser, Content: "Valid user message"},
		},
	}
	compiled, err := s.CompileAgent(context.Background(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	// Only non-empty messages are kept
	require.Len(t, compiled.Messages, 2)
	assert.Equal(t, RoleSystem, compiled.Messages[0].Role)
	assert.Equal(t, RoleUser, compiled.Messages[1].Role)
	assert.Equal(t, "Valid user message", compiled.Messages[1].Content)
}

func TestCompileAgent_CacheFlagPropagation(t *testing.T) {
	s := &Spec{
		Name:      "cache-flag",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: "Cached system message", Cache: true},
			{Role: RoleUser, Content: "Non-cached user message", Cache: false},
		},
	}
	compiled, err := s.CompileAgent(context.Background(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	require.Len(t, compiled.Messages, 2)
	assert.True(t, compiled.Messages[0].Cache, "System message should have cache=true")
	assert.False(t, compiled.Messages[1].Cache, "User message should have cache=false")
}

func TestCompileAgent_ExecutionConfigCloned(t *testing.T) {
	temp := 0.7
	original := &execution.Config{
		Provider:    ProviderOpenAI,
		Model:       "gpt-4",
		Temperature: &temp,
	}
	s := &Spec{
		Name:      "exec-clone",
		Type:      DocumentTypeAgent,
		Execution: original,
		Body:      "Hello agent",
	}
	compiled, err := s.CompileAgent(context.Background(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)
	require.NotNil(t, compiled.Execution)

	// Verify it is a clone, not a reference to the original
	assert.Equal(t, ProviderOpenAI, compiled.Execution.Provider)
	assert.Equal(t, "gpt-4", compiled.Execution.Model)
	require.NotNil(t, compiled.Execution.Temperature)
	assert.InDelta(t, 0.7, *compiled.Execution.Temperature, 0.001)

	// Mutate the compiled execution and verify original is unaffected
	newTemp := 0.9
	compiled.Execution.Temperature = &newTemp
	assert.InDelta(t, 0.7, *original.Temperature, 0.001, "Original should be unaffected by mutation of clone")
}

func TestCompileAgent_ToolsCloned(t *testing.T) {
	original := &ToolsConfig{
		Functions: []*FunctionDef{
			{Name: "search", Description: "Search the web", Parameters: map[string]any{"type": "object"}},
		},
	}
	s := &Spec{
		Name:      "tools-clone",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Tools:     original,
		Body:      "Hello agent",
	}
	compiled, err := s.CompileAgent(context.Background(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)
	require.NotNil(t, compiled.Tools)
	require.Len(t, compiled.Tools.Functions, 1)
	assert.Equal(t, "search", compiled.Tools.Functions[0].Name)

	// Mutate compiled tools and verify original is unaffected
	compiled.Tools.Functions[0].Name = "modified"
	assert.Equal(t, "search", original.Functions[0].Name, "Original tool should be unaffected")
}

func TestCompileAgent_ConstraintsCloned(t *testing.T) {
	original := &ConstraintsConfig{
		Behavioral: []string{"Be polite", "Be concise"},
		Safety:     []string{"No harmful content"},
	}
	s := &Spec{
		Name:        "constraints-clone",
		Type:        DocumentTypeAgent,
		Execution:   &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Constraints: original,
		Body:        "Hello agent",
	}
	compiled, err := s.CompileAgent(context.Background(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)
	require.NotNil(t, compiled.Constraints)
	require.Len(t, compiled.Constraints.Behavioral, 2)
	require.Len(t, compiled.Constraints.Safety, 1)

	// Mutate compiled constraints and verify original is unaffected
	compiled.Constraints.Behavioral[0] = "modified"
	assert.Equal(t, "Be polite", original.Behavioral[0], "Original constraint should be unaffected")
}

func TestCompileAgent_NilToolsAndConstraints(t *testing.T) {
	s := &Spec{
		Name:      "nil-tc",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Body:      "Hello agent",
	}
	compiled, err := s.CompileAgent(context.Background(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)
	assert.Nil(t, compiled.Tools, "Tools should be nil when spec has no tools")
	assert.Nil(t, compiled.Constraints, "Constraints should be nil when spec has no constraints")
}

func TestCompileAgent_BodyWithVariableResolution(t *testing.T) {
	s := &Spec{
		Name:      "body-var",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Body:      `You are {~exons.var name="role" default="an assistant" /~}.`,
	}
	input := map[string]any{
		"role": "a researcher",
	}
	compiled, err := s.CompileAgent(context.Background(), input, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	// Body-only means default messages
	require.Len(t, compiled.Messages, 1)
	assert.Equal(t, RoleSystem, compiled.Messages[0].Role)
	assert.Equal(t, "You are a researcher.", compiled.Messages[0].Content)
}

func TestCompileAgent_BodyAndMessages(t *testing.T) {
	// When messages are present, they take precedence over buildDefaultMessages
	s := &Spec{
		Name:      "body-and-msgs",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Body:      "This is the body content.",
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: "System from messages"},
		},
	}
	compiled, err := s.CompileAgent(context.Background(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	// Messages defined in spec override the buildDefaultMessages path
	require.Len(t, compiled.Messages, 1)
	assert.Equal(t, RoleSystem, compiled.Messages[0].Role)
	assert.Equal(t, "System from messages", compiled.Messages[0].Content)
}

func TestCompileAgent_SkillsCatalogInContext(t *testing.T) {
	resolver := NewMapSpecResolver()
	resolver.Add("web-search", &Spec{
		Name:        "web-search",
		Description: "Search the web",
	}, "")

	s := &Spec{
		Name:      "skills-ctx",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Skills: []SkillRef{
			{Slug: "web-search", Injection: string(SkillInjectionSystemPrompt)},
		},
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: `Skills: {~exons.var name="skills" default="none" /~}`},
		},
	}

	opts := NewCompileOptions(WithResolver(resolver))
	compiled, err := s.CompileAgent(context.Background(), nil, opts)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	// The skills catalog should be injected into the template context
	require.Len(t, compiled.Messages, 1)
	assert.Contains(t, compiled.Messages[0].Content, "web-search")
}

func TestCompileAgent_ToolsCatalogInContext(t *testing.T) {
	s := &Spec{
		Name:      "tools-ctx",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Tools: &ToolsConfig{
			Functions: []*FunctionDef{
				{Name: "search", Description: "Search function"},
			},
		},
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: `Tools: {~exons.var name="tools" default="none" /~}`},
		},
	}

	compiled, err := s.CompileAgent(context.Background(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	require.Len(t, compiled.Messages, 1)
	assert.Contains(t, compiled.Messages[0].Content, "search")
}

// =============================================================================
// Spec.ActivateSkill — skill activation with injection modes
// =============================================================================

func TestActivateSkill_NilSpec(t *testing.T) {
	var s *Spec
	compiled, err := s.ActivateSkill(context.Background(), "some-skill", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgCompileNotAgent)
	assert.Nil(t, compiled)
}

func TestActivateSkill_SkillNotFound(t *testing.T) {
	s := &Spec{
		Name:      "agent",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Body:      "Agent body",
		Skills: []SkillRef{
			{Slug: "existing-skill"},
		},
	}
	compiled, err := s.ActivateSkill(context.Background(), "missing-skill", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgActivateSkillNotFound)
	assert.Nil(t, compiled)
}

func TestActivateSkill_NoResolver(t *testing.T) {
	s := &Spec{
		Name:      "agent",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Body:      "Agent system prompt",
		Skills: []SkillRef{
			{Slug: "test-skill"},
		},
	}
	// No resolver means skill is found but not resolved — returns compiled agent without injection
	compiled, err := s.ActivateSkill(context.Background(), "test-skill", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)
	require.Len(t, compiled.Messages, 1)
	assert.Equal(t, RoleSystem, compiled.Messages[0].Role)
	assert.Equal(t, "Agent system prompt", compiled.Messages[0].Content)
}

func TestActivateSkill_SystemPromptInjection_Default(t *testing.T) {
	resolver := NewMapSpecResolver()
	resolver.Add("helper-skill", &Spec{
		Name:        "helper-skill",
		Description: "A helper skill",
	}, "You can help with tasks.")

	s := &Spec{
		Name:      "agent",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Body:      "You are an agent.",
		Skills: []SkillRef{
			{Slug: "helper-skill"}, // No injection set => defaults to system_prompt
		},
	}

	opts := NewCompileOptions(WithResolver(resolver))
	compiled, err := s.ActivateSkill(context.Background(), "helper-skill", nil, opts)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	// Skill should be injected into the system message
	require.True(t, len(compiled.Messages) >= 1)
	systemMsg := compiled.Messages[0]
	assert.Equal(t, RoleSystem, systemMsg.Role)
	assert.Contains(t, systemMsg.Content, "You are an agent.")
	assert.Contains(t, systemMsg.Content, "You can help with tasks.")
	assert.Contains(t, systemMsg.Content, SkillInjectionMarkerStart+"helper-skill")
	assert.Contains(t, systemMsg.Content, SkillInjectionMarkerEnd+"helper-skill")
}

func TestActivateSkill_SystemPromptInjection_Explicit(t *testing.T) {
	resolver := NewMapSpecResolver()
	resolver.Add("explicit-skill", &Spec{
		Name:        "explicit-skill",
		Description: "Explicit injection skill",
	}, "Skill content here.")

	s := &Spec{
		Name:      "agent",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Body:      "Base system prompt.",
		Skills: []SkillRef{
			{Slug: "explicit-skill", Injection: string(SkillInjectionSystemPrompt)},
		},
	}

	opts := NewCompileOptions(WithResolver(resolver))
	compiled, err := s.ActivateSkill(context.Background(), "explicit-skill", nil, opts)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	require.True(t, len(compiled.Messages) >= 1)
	systemMsg := compiled.Messages[0]
	assert.Equal(t, RoleSystem, systemMsg.Role)
	assert.Contains(t, systemMsg.Content, "Base system prompt.")
	assert.Contains(t, systemMsg.Content, "Skill content here.")
}

func TestActivateSkill_UserContextInjection(t *testing.T) {
	resolver := NewMapSpecResolver()
	resolver.Add("user-skill", &Spec{
		Name:        "user-skill",
		Description: "User context skill",
	}, "User-injected skill content.")

	s := &Spec{
		Name:      "agent",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: "System message."},
			{Role: RoleUser, Content: "User question."},
		},
		Skills: []SkillRef{
			{Slug: "user-skill", Injection: string(SkillInjectionUserContext)},
		},
	}

	opts := NewCompileOptions(WithResolver(resolver))
	compiled, err := s.ActivateSkill(context.Background(), "user-skill", nil, opts)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	// Skill should be injected into the last user message
	var userMsg *CompiledMessage
	for i := len(compiled.Messages) - 1; i >= 0; i-- {
		if compiled.Messages[i].Role == RoleUser {
			userMsg = &compiled.Messages[i]
			break
		}
	}
	require.NotNil(t, userMsg, "Should have a user message")
	assert.Contains(t, userMsg.Content, "User question.")
	assert.Contains(t, userMsg.Content, "User-injected skill content.")
	assert.Contains(t, userMsg.Content, SkillInjectionMarkerStart+"user-skill")
}

func TestActivateSkill_NoneInjection(t *testing.T) {
	resolver := NewMapSpecResolver()
	resolver.Add("silent-skill", &Spec{
		Name:        "silent-skill",
		Description: "No injection skill",
	}, "This should not appear in messages.")

	s := &Spec{
		Name:      "agent",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Body:      "System message.",
		Skills: []SkillRef{
			{Slug: "silent-skill", Injection: string(SkillInjectionNone)},
		},
	}

	opts := NewCompileOptions(WithResolver(resolver))
	compiled, err := s.ActivateSkill(context.Background(), "silent-skill", nil, opts)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	// With "none" injection, skill content should NOT appear in any messages
	for _, msg := range compiled.Messages {
		assert.NotContains(t, msg.Content, "This should not appear in messages.")
		assert.NotContains(t, msg.Content, SkillInjectionMarkerStart+"silent-skill")
	}
}

func TestActivateSkill_EmptySkillBody(t *testing.T) {
	resolver := NewMapSpecResolver()
	resolver.Add("empty-skill", &Spec{
		Name:        "empty-skill",
		Description: "Skill with empty body",
	}, "") // Empty body

	s := &Spec{
		Name:      "agent",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Body:      "System message content.",
		Skills: []SkillRef{
			{Slug: "empty-skill", Injection: string(SkillInjectionSystemPrompt)},
		},
	}

	opts := NewCompileOptions(WithResolver(resolver))
	compiled, err := s.ActivateSkill(context.Background(), "empty-skill", nil, opts)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	// Empty skill body means no injection markers
	require.True(t, len(compiled.Messages) >= 1)
	assert.NotContains(t, compiled.Messages[0].Content, SkillInjectionMarkerStart)
}

func TestActivateSkill_SkillBodyWithTemplateExecution(t *testing.T) {
	resolver := NewMapSpecResolver()
	resolver.Add("template-skill", &Spec{
		Name:        "template-skill",
		Description: "Templated skill",
	}, `Skill for {~exons.var name="name" default="someone" /~}`)

	s := &Spec{
		Name:      "agent",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Body:      "Agent prompt.",
		Skills: []SkillRef{
			{Slug: "template-skill", Injection: string(SkillInjectionSystemPrompt)},
		},
	}

	input := map[string]any{"name": "Alice"}
	opts := NewCompileOptions(WithResolver(resolver))
	compiled, err := s.ActivateSkill(context.Background(), "template-skill", input, opts)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	require.True(t, len(compiled.Messages) >= 1)
	assert.Contains(t, compiled.Messages[0].Content, "Skill for Alice")
}

func TestActivateSkill_MergesSkillExecutionConfig(t *testing.T) {
	skillTemp := 0.9
	resolver := NewMapSpecResolver()
	resolver.Add("config-skill", &Spec{
		Name:        "config-skill",
		Description: "Skill with execution config",
		Execution:   &execution.Config{Model: "gpt-4-turbo", Temperature: &skillTemp},
	}, "Skill body content.")

	agentTemp := 0.5
	s := &Spec{
		Name:      "agent",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4", Temperature: &agentTemp},
		Body:      "System prompt.",
		Skills: []SkillRef{
			{Slug: "config-skill", Injection: string(SkillInjectionSystemPrompt)},
		},
	}

	opts := NewCompileOptions(WithResolver(resolver))
	compiled, err := s.ActivateSkill(context.Background(), "config-skill", nil, opts)
	require.NoError(t, err)
	require.NotNil(t, compiled)
	require.NotNil(t, compiled.Execution)

	// Skill execution config should override agent's: model should be overridden
	assert.Equal(t, "gpt-4-turbo", compiled.Execution.Model)
	assert.Equal(t, ProviderOpenAI, compiled.Execution.Provider) // Provider preserved from agent
	require.NotNil(t, compiled.Execution.Temperature)
	assert.InDelta(t, 0.9, *compiled.Execution.Temperature, 0.001) // Skill overrides agent
}

// =============================================================================
// buildDefaultMessages — unit tests for the unexported helper
// =============================================================================

func TestBuildDefaultMessages_EmptyBodyNilInput(t *testing.T) {
	msgs := buildDefaultMessages("", nil)
	assert.Empty(t, msgs)
}

func TestBuildDefaultMessages_BodyOnly(t *testing.T) {
	msgs := buildDefaultMessages("System content", nil)
	require.Len(t, msgs, 1)
	assert.Equal(t, RoleSystem, msgs[0].Role)
	assert.Equal(t, "System content", msgs[0].Content)
	assert.False(t, msgs[0].Cache)
}

func TestBuildDefaultMessages_BodyWithQuery(t *testing.T) {
	input := map[string]any{"query": "User query"}
	msgs := buildDefaultMessages("System content", input)
	require.Len(t, msgs, 2)
	assert.Equal(t, RoleSystem, msgs[0].Role)
	assert.Equal(t, "System content", msgs[0].Content)
	assert.Equal(t, RoleUser, msgs[1].Role)
	assert.Equal(t, "User query", msgs[1].Content)
}

func TestBuildDefaultMessages_BodyWithMessage(t *testing.T) {
	input := map[string]any{"message": "User message"}
	msgs := buildDefaultMessages("System content", input)
	require.Len(t, msgs, 2)
	assert.Equal(t, RoleUser, msgs[1].Role)
	assert.Equal(t, "User message", msgs[1].Content)
}

func TestBuildDefaultMessages_EmptyBodyWithQuery(t *testing.T) {
	input := map[string]any{"query": "User query"}
	msgs := buildDefaultMessages("", input)
	// No body -> no system message, but still get user message from query
	require.Len(t, msgs, 1)
	assert.Equal(t, RoleUser, msgs[0].Role)
	assert.Equal(t, "User query", msgs[0].Content)
}

func TestBuildDefaultMessages_EmptyStringQuery(t *testing.T) {
	input := map[string]any{"query": ""}
	msgs := buildDefaultMessages("Body", input)
	// Empty query string should NOT create a user message
	require.Len(t, msgs, 1)
	assert.Equal(t, RoleSystem, msgs[0].Role)
}

func TestBuildDefaultMessages_NonStringQuery(t *testing.T) {
	input := map[string]any{"query": 42}
	msgs := buildDefaultMessages("Body", input)
	// Non-string query should NOT create a user message
	require.Len(t, msgs, 1)
	assert.Equal(t, RoleSystem, msgs[0].Role)
}

func TestBuildDefaultMessages_QueryPrecedenceOverMessage(t *testing.T) {
	input := map[string]any{
		"query":   "From query",
		"message": "From message",
	}
	msgs := buildDefaultMessages("Body", input)
	require.Len(t, msgs, 2)
	assert.Equal(t, RoleUser, msgs[1].Role)
	assert.Equal(t, "From query", msgs[1].Content) // query checked first
}

// =============================================================================
// Skill injection helpers — injectSkillIntoSystemPrompt, injectSkillIntoUserContext
// =============================================================================

func TestInjectSkillIntoSystemPrompt_ExistingSystemMessage(t *testing.T) {
	compiled := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleSystem, Content: "Original system"},
			{Role: RoleUser, Content: "User msg"},
		},
	}

	injectSkillIntoSystemPrompt(compiled, "my-skill", "Skill content")

	require.Len(t, compiled.Messages, 2)
	assert.Equal(t, RoleSystem, compiled.Messages[0].Role)
	assert.Contains(t, compiled.Messages[0].Content, "Original system")
	assert.Contains(t, compiled.Messages[0].Content, "Skill content")
	assert.Contains(t, compiled.Messages[0].Content, SkillInjectionMarkerStart+"my-skill"+SkillInjectionMarkerClose)
	assert.Contains(t, compiled.Messages[0].Content, SkillInjectionMarkerEnd+"my-skill"+SkillInjectionMarkerClose)
}

func TestInjectSkillIntoSystemPrompt_NoSystemMessage(t *testing.T) {
	compiled := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleUser, Content: "User message only"},
		},
	}

	injectSkillIntoSystemPrompt(compiled, "new-skill", "New skill content")

	// System message should be prepended
	require.Len(t, compiled.Messages, 2)
	assert.Equal(t, RoleSystem, compiled.Messages[0].Role)
	assert.Contains(t, compiled.Messages[0].Content, "New skill content")
	assert.Contains(t, compiled.Messages[0].Content, SkillInjectionMarkerStart+"new-skill")
	assert.Equal(t, RoleUser, compiled.Messages[1].Role)
	assert.Equal(t, "User message only", compiled.Messages[1].Content)
}

func TestInjectSkillIntoSystemPrompt_EmptyMessages(t *testing.T) {
	compiled := &CompiledSpec{
		Messages: []CompiledMessage{},
	}

	injectSkillIntoSystemPrompt(compiled, "empty-skill", "Skill content")

	require.Len(t, compiled.Messages, 1)
	assert.Equal(t, RoleSystem, compiled.Messages[0].Role)
	assert.Contains(t, compiled.Messages[0].Content, "Skill content")
}

func TestInjectSkillIntoSystemPrompt_MarkerFormat(t *testing.T) {
	compiled := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleSystem, Content: "System"},
		},
	}

	injectSkillIntoSystemPrompt(compiled, "test-skill", "Content here")

	content := compiled.Messages[0].Content
	// Verify marker format: \n\n<!-- SKILL_START:test-skill -->\nContent here\n<!-- SKILL_END:test-skill -->
	assert.Contains(t, content, "<!-- SKILL_START:test-skill -->")
	assert.Contains(t, content, "Content here")
	assert.Contains(t, content, "<!-- SKILL_END:test-skill -->")
}

func TestInjectSkillIntoUserContext_ExistingUserMessage(t *testing.T) {
	compiled := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleSystem, Content: "System msg"},
			{Role: RoleUser, Content: "First user msg"},
			{Role: RoleUser, Content: "Last user msg"},
		},
	}

	injectSkillIntoUserContext(compiled, "user-skill", "User skill content")

	require.Len(t, compiled.Messages, 3)
	// Injection goes into the LAST user message
	assert.NotContains(t, compiled.Messages[1].Content, "User skill content")
	assert.Contains(t, compiled.Messages[2].Content, "Last user msg")
	assert.Contains(t, compiled.Messages[2].Content, "User skill content")
	assert.Contains(t, compiled.Messages[2].Content, SkillInjectionMarkerStart+"user-skill")
}

func TestInjectSkillIntoUserContext_NoUserMessage(t *testing.T) {
	compiled := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleSystem, Content: "System only"},
		},
	}

	injectSkillIntoUserContext(compiled, "ctx-skill", "Context skill content")

	// Should append a new user message
	require.Len(t, compiled.Messages, 2)
	assert.Equal(t, RoleSystem, compiled.Messages[0].Role)
	assert.Equal(t, RoleUser, compiled.Messages[1].Role)
	assert.Contains(t, compiled.Messages[1].Content, "Context skill content")
	assert.Contains(t, compiled.Messages[1].Content, SkillInjectionMarkerStart+"ctx-skill")
}

func TestInjectSkillIntoUserContext_EmptyMessages(t *testing.T) {
	compiled := &CompiledSpec{
		Messages: []CompiledMessage{},
	}

	injectSkillIntoUserContext(compiled, "new-skill", "New skill content")

	require.Len(t, compiled.Messages, 1)
	assert.Equal(t, RoleUser, compiled.Messages[0].Role)
	assert.Contains(t, compiled.Messages[0].Content, "New skill content")
}

func TestInjectSkillIntoUserContext_MarkerFormat(t *testing.T) {
	compiled := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleUser, Content: "User question"},
		},
	}

	injectSkillIntoUserContext(compiled, "fmt-skill", "Formatted content")

	content := compiled.Messages[0].Content
	assert.Contains(t, content, "<!-- SKILL_START:fmt-skill -->")
	assert.Contains(t, content, "Formatted content")
	assert.Contains(t, content, "<!-- SKILL_END:fmt-skill -->")
}

// =============================================================================
// buildCompileContext — context construction tests
// =============================================================================

func TestBuildCompileContext_InputFlattened(t *testing.T) {
	s := &Spec{Name: "test"}
	input := map[string]any{
		"key1": "value1",
		"key2": 42,
	}
	data := buildCompileContext(context.Background(), s, input, nil)

	// Input should be under "input" key AND flattened at top level
	assert.Equal(t, input, data[ContextKeyInput])
	assert.Equal(t, "value1", data["key1"])
	assert.Equal(t, 42, data["key2"])
}

func TestBuildCompileContext_MetaPopulated(t *testing.T) {
	s := &Spec{Name: "test-agent", Type: DocumentTypeAgent}
	data := buildCompileContext(context.Background(), s, nil, nil)

	meta, ok := data[ContextKeyMeta].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, string(DocumentTypeAgent), meta[MetaKeyDocumentType])
	assert.Equal(t, "test-agent", meta[MetaKeySpecName])
}

func TestBuildCompileContext_SpecContext(t *testing.T) {
	s := &Spec{
		Name: "test",
		Context: map[string]any{
			"company": "Acme",
			"region":  "US",
		},
	}
	data := buildCompileContext(context.Background(), s, nil, nil)

	assert.Equal(t, s.Context, data[ContextKeyContext])
	assert.Equal(t, "Acme", data["company"])
	assert.Equal(t, "US", data["region"])
}

func TestBuildCompileContext_InputDoesNotOverrideContext(t *testing.T) {
	// Input is set first, context values only fill gaps
	s := &Spec{
		Name: "test",
		Context: map[string]any{
			"key": "from-context",
		},
	}
	input := map[string]any{
		"key": "from-input",
	}
	data := buildCompileContext(context.Background(), s, input, nil)
	// Input was set first, context should NOT overwrite
	assert.Equal(t, "from-input", data["key"])
}

func TestBuildCompileContext_ConstraintsPopulated(t *testing.T) {
	s := &Spec{
		Name: "test",
		Constraints: &ConstraintsConfig{
			Behavioral: []string{"Be polite"},
			Safety:     []string{"No harmful content"},
		},
	}
	data := buildCompileContext(context.Background(), s, nil, nil)

	constraintsRaw, ok := data[ContextKeyConstraints]
	require.True(t, ok)
	constraints, ok := constraintsRaw.(map[string]any)
	require.True(t, ok)
	assert.Contains(t, constraints, "behavioral")
	assert.Contains(t, constraints, "safety")
}

func TestBuildCompileContext_NilInput(t *testing.T) {
	s := &Spec{Name: "test"}
	data := buildCompileContext(context.Background(), s, nil, nil)

	_, hasInput := data[ContextKeyInput]
	assert.False(t, hasInput, "ContextKeyInput should not be present when input is nil")
}

// =============================================================================
// compileMessages — unit tests for message compilation
// =============================================================================

func TestCompileMessages_PlainText(t *testing.T) {
	engine := MustNew()
	templates := []MessageTemplate{
		{Role: RoleSystem, Content: "Static system message"},
		{Role: RoleUser, Content: "Static user message"},
	}
	data := map[string]any{}

	msgs, err := compileMessages(context.Background(), engine, templates, data, "")
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	assert.Equal(t, "Static system message", msgs[0].Content)
	assert.Equal(t, "Static user message", msgs[1].Content)
}

func TestCompileMessages_WithTemplateExecution(t *testing.T) {
	engine := MustNew()
	templates := []MessageTemplate{
		{Role: RoleSystem, Content: `Hello {~exons.var name="name" default="World" /~}`},
	}
	data := map[string]any{
		"name": "Alice",
	}

	msgs, err := compileMessages(context.Background(), engine, templates, data, "")
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "Hello Alice", msgs[0].Content)
}

func TestCompileMessages_EmptyContentSkipped(t *testing.T) {
	engine := MustNew()
	templates := []MessageTemplate{
		{Role: RoleSystem, Content: "Valid"},
		{Role: RoleUser, Content: "  "},
		{Role: RoleAssistant, Content: "Also valid"},
	}
	data := map[string]any{}

	msgs, err := compileMessages(context.Background(), engine, templates, data, "")
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	assert.Equal(t, "Valid", msgs[0].Content)
	assert.Equal(t, "Also valid", msgs[1].Content)
}

func TestCompileMessages_CachePropagated(t *testing.T) {
	engine := MustNew()
	templates := []MessageTemplate{
		{Role: RoleSystem, Content: "Cached", Cache: true},
		{Role: RoleUser, Content: "Not cached", Cache: false},
	}
	data := map[string]any{}

	msgs, err := compileMessages(context.Background(), engine, templates, data, "")
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	assert.True(t, msgs[0].Cache)
	assert.False(t, msgs[1].Cache)
}

func TestCompileMessages_SelfBodyInjected(t *testing.T) {
	engine := MustNew()
	templates := []MessageTemplate{
		{Role: RoleSystem, Content: `Body: {~exons.var name="_selfBody" default="none" /~}`},
	}
	data := map[string]any{}
	compiledBody := "Compiled body content"

	msgs, err := compileMessages(context.Background(), engine, templates, data, compiledBody)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "Body: Compiled body content", msgs[0].Content)
}

// =============================================================================
// Integration tests — MapSpecResolver with ActivateSkill
// =============================================================================

func TestActivateSkill_ResolverIntegration(t *testing.T) {
	ctx := context.Background()
	resolver := NewMapSpecResolver()

	// Add multiple skills
	resolver.Add("search", &Spec{
		Name:        "search",
		Description: "Web search capability",
	}, "You can search the web for information.")

	resolver.Add("calculator", &Spec{
		Name:        "calculator",
		Description: "Math operations",
	}, "You can perform mathematical calculations.")

	s := &Spec{
		Name:      "multi-skill-agent",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Body:      "You are a multi-skilled assistant.",
		Skills: []SkillRef{
			{Slug: "search", Injection: string(SkillInjectionSystemPrompt)},
			{Slug: "calculator", Injection: string(SkillInjectionUserContext)},
		},
	}

	opts := NewCompileOptions(WithResolver(resolver))

	t.Run("activate search skill", func(t *testing.T) {
		compiled, err := s.ActivateSkill(ctx, "search", nil, opts)
		require.NoError(t, err)
		require.NotNil(t, compiled)

		// search is system_prompt injection
		systemFound := false
		for _, msg := range compiled.Messages {
			if msg.Role == RoleSystem && strings.Contains(msg.Content, "search the web") {
				systemFound = true
			}
		}
		assert.True(t, systemFound, "Search skill content should be in system message")
	})

	t.Run("activate calculator skill", func(t *testing.T) {
		// Need to provide a query or message for user context injection
		input := map[string]any{"query": "What is 2+2?"}
		compiled, err := s.ActivateSkill(ctx, "calculator", input, opts)
		require.NoError(t, err)
		require.NotNil(t, compiled)

		// calculator is user_context injection
		userFound := false
		for _, msg := range compiled.Messages {
			if msg.Role == RoleUser && strings.Contains(msg.Content, "mathematical calculations") {
				userFound = true
			}
		}
		assert.True(t, userFound, "Calculator skill content should be in user message")
	})
}

func TestActivateSkill_SystemPromptCreatesSystemWhenMissing(t *testing.T) {
	// Agent with only explicit user messages, no system message.
	// system_prompt injection should create a new system message.
	resolver := NewMapSpecResolver()
	resolver.Add("inject-skill", &Spec{
		Name:        "inject-skill",
		Description: "Injected",
	}, "Injected skill content")

	s := &Spec{
		Name:      "user-only-agent",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Messages: []MessageTemplate{
			{Role: RoleUser, Content: "Hello"},
		},
		Skills: []SkillRef{
			{Slug: "inject-skill", Injection: string(SkillInjectionSystemPrompt)},
		},
	}

	opts := NewCompileOptions(WithResolver(resolver))
	compiled, err := s.ActivateSkill(context.Background(), "inject-skill", nil, opts)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	// Should have 2 messages: new system + existing user
	require.Len(t, compiled.Messages, 2)
	assert.Equal(t, RoleSystem, compiled.Messages[0].Role)
	assert.Contains(t, compiled.Messages[0].Content, "Injected skill content")
	assert.Equal(t, RoleUser, compiled.Messages[1].Role)
	assert.Equal(t, "Hello", compiled.Messages[1].Content)
}

func TestActivateSkill_UserContextCreatesUserWhenMissing(t *testing.T) {
	// Agent with only system message, no user message.
	// user_context injection should create a new user message.
	resolver := NewMapSpecResolver()
	resolver.Add("user-inject", &Spec{
		Name:        "user-inject",
		Description: "User injected",
	}, "Skill for user context")

	s := &Spec{
		Name:      "system-only-agent",
		Type:      DocumentTypeAgent,
		Execution: &execution.Config{Provider: ProviderOpenAI, Model: "gpt-4"},
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: "System message only"},
		},
		Skills: []SkillRef{
			{Slug: "user-inject", Injection: string(SkillInjectionUserContext)},
		},
	}

	opts := NewCompileOptions(WithResolver(resolver))
	compiled, err := s.ActivateSkill(context.Background(), "user-inject", nil, opts)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	// Should have 2 messages: system + new user
	require.Len(t, compiled.Messages, 2)
	assert.Equal(t, RoleSystem, compiled.Messages[0].Role)
	assert.Equal(t, RoleUser, compiled.Messages[1].Role)
	assert.Contains(t, compiled.Messages[1].Content, "Skill for user context")
}
