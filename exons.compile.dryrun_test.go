package exons

import (
	"context"
	"testing"

	"github.com/itsatony/go-exons/execution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// AgentDryRunResult methods
// =============================================================================

func TestDryRunResult_OK(t *testing.T) {
	t.Run("empty result is OK", func(t *testing.T) {
		r := &AgentDryRunResult{}
		assert.True(t, r.OK())
		assert.False(t, r.HasErrors())
	})

	t.Run("result with issues is not OK", func(t *testing.T) {
		r := &AgentDryRunResult{
			Issues: []AgentDryRunIssue{
				{Category: AgentDryRunCategoryValidation, Message: "test", Location: DryRunLocationSpec},
			},
		}
		assert.False(t, r.OK())
		assert.True(t, r.HasErrors())
	})
}

func TestAgentDryRunResult_String(t *testing.T) {
	t.Run("OK summary", func(t *testing.T) {
		r := &AgentDryRunResult{
			SkillsResolved: 2,
			ToolsDefined:   3,
			MessageCount:   4,
		}
		s := r.String()
		assert.Contains(t, s, "2 skills resolved")
		assert.Contains(t, s, "3 tools defined")
		assert.Contains(t, s, "4 messages")
		assert.NotContains(t, s, "issue")
	})

	t.Run("issues summary", func(t *testing.T) {
		r := &AgentDryRunResult{
			Issues: []AgentDryRunIssue{
				{Category: AgentDryRunCategoryValidation, Message: "bad spec", Location: DryRunLocationSpec},
				{Category: AgentDryRunCategoryResolver, Message: "not found", Location: DryRunLocationSkillPrefix + "web-search"},
			},
		}
		s := r.String()
		assert.Contains(t, s, "2 issue(s)")
		assert.Contains(t, s, AgentDryRunCategoryValidation)
		assert.Contains(t, s, DryRunLocationSpec)
		assert.Contains(t, s, "bad spec")
		assert.Contains(t, s, AgentDryRunCategoryResolver)
		assert.Contains(t, s, "web-search")
	})
}

// =============================================================================
// Spec.AgentDryRun — nil spec
// =============================================================================

func TestDryRun_NilSpec(t *testing.T) {
	var spec *Spec
	result := spec.AgentDryRun(context.Background(), nil)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	require.Len(t, result.Issues, 1)
	assert.Equal(t, AgentDryRunCategoryValidation, result.Issues[0].Category)
	assert.Equal(t, ErrMsgAgentDryRunNilSpec, result.Issues[0].Message)
	assert.Equal(t, DryRunLocationSpec, result.Issues[0].Location)
}

// =============================================================================
// Spec.AgentDryRun — valid agent
// =============================================================================

func TestDryRun_ValidAgent(t *testing.T) {
	spec := &Spec{
		Name:        "test-agent",
		Description: "A test agent",
		Type:        DocumentTypeAgent,
		Execution: &execution.Config{
			Provider: ProviderOpenAI,
			Model:    "gpt-4",
		},
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: "You are helpful."},
			{Role: RoleUser, Content: "Hello"},
		},
		Tools: &ToolsConfig{
			Functions: []*FunctionDef{
				{Name: "search", Description: "Search the web"},
			},
			MCPServers: []*MCPServer{
				{Name: "mcp1", URL: "http://localhost:8080"},
			},
		},
	}

	result := spec.AgentDryRun(context.Background(), nil)
	require.NotNil(t, result)
	assert.True(t, result.OK())
	assert.False(t, result.HasErrors())
	assert.Equal(t, 0, result.SkillsResolved) // no resolver provided
	assert.Equal(t, 2, result.ToolsDefined)   // 1 function + 1 MCP
	assert.Equal(t, 2, result.MessageCount)
}

// =============================================================================
// Spec.AgentDryRun — non-agent type
// =============================================================================

func TestDryRun_NonAgentType(t *testing.T) {
	spec := &Spec{
		Name:        "my-skill",
		Description: "A test skill",
		Type:        DocumentTypeSkill,
		Execution: &execution.Config{
			Provider: ProviderOpenAI,
			Model:    "gpt-4",
		},
		Body: "skill body",
	}

	result := spec.AgentDryRun(context.Background(), nil)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())

	// Should have validation issue but still continue
	found := false
	for _, issue := range result.Issues {
		if issue.Category == AgentDryRunCategoryValidation && issue.Location == DryRunLocationSpec {
			found = true
			assert.Contains(t, issue.Message, ErrMsgNotAnAgent)
		}
	}
	assert.True(t, found, "expected validation issue for non-agent type")
}

// =============================================================================
// Spec.AgentDryRun — missing execution
// =============================================================================

func TestDryRun_MissingExecution(t *testing.T) {
	spec := &Spec{
		Name:        "no-exec",
		Description: "Missing execution",
		Type:        DocumentTypeAgent,
		Body:        "some body",
	}

	result := spec.AgentDryRun(context.Background(), nil)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())

	found := false
	for _, issue := range result.Issues {
		if issue.Category == AgentDryRunCategoryValidation {
			found = true
			assert.Contains(t, issue.Message, ErrMsgNoExecutionConfig)
		}
	}
	assert.True(t, found, "expected validation issue for missing execution")
}

// =============================================================================
// Spec.AgentDryRun — skill resolution failures
// =============================================================================

func TestDryRun_SkillResolutionFailures(t *testing.T) {
	resolver := NewMapSpecResolver()
	resolver.Add("found-skill", &Spec{
		Name:        "found-skill",
		Description: "A found skill",
	}, "skill body")

	spec := &Spec{
		Name:        "agent-with-skills",
		Description: "Agent with skills",
		Type:        DocumentTypeAgent,
		Execution: &execution.Config{
			Provider: ProviderOpenAI,
			Model:    "gpt-4",
		},
		Body: "agent body",
		Skills: []SkillRef{
			{Slug: "found-skill"},
			{Slug: "missing-skill"},
			{Slug: "also-missing"},
		},
	}

	opts := &CompileOptions{Resolver: resolver}
	result := spec.AgentDryRun(context.Background(), opts)
	require.NotNil(t, result)

	assert.Equal(t, 1, result.SkillsResolved)

	// Should have 2 resolver issues
	resolverIssues := 0
	for _, issue := range result.Issues {
		if issue.Category == AgentDryRunCategoryResolver {
			resolverIssues++
			assert.True(t,
				issue.Location == DryRunLocationSkillPrefix+"missing-skill" ||
					issue.Location == DryRunLocationSkillPrefix+"also-missing",
				"unexpected resolver issue location: %s", issue.Location)
		}
	}
	assert.Equal(t, 2, resolverIssues)
}

// =============================================================================
// Spec.AgentDryRun — no resolver skips skills
// =============================================================================

func TestDryRun_NoResolver_SkillsSkipped(t *testing.T) {
	spec := &Spec{
		Name:        "agent-no-resolver",
		Description: "Agent without resolver",
		Type:        DocumentTypeAgent,
		Execution: &execution.Config{
			Provider: ProviderOpenAI,
			Model:    "gpt-4",
		},
		Body: "body",
		Skills: []SkillRef{
			{Slug: "some-skill"},
		},
	}

	// No resolver in opts
	result := spec.AgentDryRun(context.Background(), nil)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.SkillsResolved)

	// Should have no resolver issues
	for _, issue := range result.Issues {
		assert.NotEqual(t, AgentDryRunCategoryResolver, issue.Category)
	}
}

// =============================================================================
// Spec.AgentDryRun — credential validation issues
// =============================================================================

func TestDryRun_CredentialValidation(t *testing.T) {
	spec := &Spec{
		Name:        "cred-agent",
		Description: "Agent with credential issues",
		Type:        DocumentTypeAgent,
		Execution: &execution.Config{
			Provider: ProviderOpenAI,
			Model:    "gpt-4",
		},
		Body:       "body",
		Credential: "nonexistent-label",
		Credentials: map[string]*CredentialRef{
			"main": {Provider: ProviderOpenAI, Ref: "key"},
		},
	}

	result := spec.AgentDryRun(context.Background(), nil)
	require.NotNil(t, result)

	found := false
	for _, issue := range result.Issues {
		if issue.Category == AgentDryRunCategoryCredential {
			found = true
			assert.Equal(t, DryRunLocationCredentials, issue.Location)
		}
	}
	assert.True(t, found, "expected credential validation issue")
}

// =============================================================================
// Spec.AgentDryRun — invalid message template
// =============================================================================

func TestDryRun_InvalidMessageTemplate(t *testing.T) {
	spec := &Spec{
		Name:        "msg-agent",
		Description: "Agent with invalid message template",
		Type:        DocumentTypeAgent,
		Execution: &execution.Config{
			Provider: ProviderOpenAI,
			Model:    "gpt-4",
		},
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: "Valid content without tags"},
			{Role: RoleUser, Content: "{~exons.var name=\"query\""},                    // unterminated
			{Role: RoleAssistant, Content: "Also valid {~exons.var name=\"x\" /~} ok"}, // valid
		},
	}

	result := spec.AgentDryRun(context.Background(), nil)
	require.NotNil(t, result)

	templateIssues := 0
	for _, issue := range result.Issues {
		if issue.Category == AgentDryRunCategoryTemplate {
			templateIssues++
			assert.Equal(t, DryRunLocationMessagePrefix+"1"+DryRunLocationMessageSuffix, issue.Location)
		}
	}
	assert.Equal(t, 1, templateIssues)
	assert.Equal(t, 3, result.MessageCount)
}

// =============================================================================
// Spec.AgentDryRun — invalid body template
// =============================================================================

func TestDryRun_InvalidBody(t *testing.T) {
	spec := &Spec{
		Name:        "body-agent",
		Description: "Agent with invalid body",
		Type:        DocumentTypeAgent,
		Execution: &execution.Config{
			Provider: ProviderOpenAI,
			Model:    "gpt-4",
		},
		Body: "{~exons.var name=\"x\"", // unterminated
	}

	result := spec.AgentDryRun(context.Background(), nil)
	require.NotNil(t, result)

	found := false
	for _, issue := range result.Issues {
		if issue.Category == AgentDryRunCategoryTemplate && issue.Location == DryRunLocationBody {
			found = true
		}
	}
	assert.True(t, found, "expected template issue for invalid body")
}

// =============================================================================
// Spec.AgentDryRun — body without tags is not parsed
// =============================================================================

func TestDryRun_BodyWithoutTags(t *testing.T) {
	spec := &Spec{
		Name:        "plain-agent",
		Description: "Agent with plain body",
		Type:        DocumentTypeAgent,
		Execution: &execution.Config{
			Provider: ProviderOpenAI,
			Model:    "gpt-4",
		},
		Body: "Just plain text with no template tags",
	}

	result := spec.AgentDryRun(context.Background(), nil)
	require.NotNil(t, result)
	assert.True(t, result.OK())
}

// =============================================================================
// Spec.AgentDryRun — multiple issues collected simultaneously
// =============================================================================

func TestDryRun_MultipleIssues(t *testing.T) {
	spec := &Spec{
		Name:        "broken-agent",
		Description: "Agent with multiple issues",
		Type:        DocumentTypeSkill,        // wrong type
		Body:        "{~exons.var name=\"x\"", // unterminated tag
		Credential:  "missing-label",
		Credentials: map[string]*CredentialRef{
			"main": {Provider: ProviderOpenAI},
		},
		Skills: []SkillRef{
			{Slug: "missing-skill"},
		},
	}

	resolver := NewMapSpecResolver()
	opts := &CompileOptions{Resolver: resolver}

	result := spec.AgentDryRun(context.Background(), opts)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())

	// Expect at minimum: validation (not agent), resolver (missing skill), credential, template (body)
	categories := make(map[string]int)
	for _, issue := range result.Issues {
		categories[issue.Category]++
	}

	assert.Greater(t, categories[AgentDryRunCategoryValidation], 0, "expected validation issue")
	assert.Greater(t, categories[AgentDryRunCategoryResolver], 0, "expected resolver issue")
	assert.Greater(t, categories[AgentDryRunCategoryCredential], 0, "expected credential issue")
	assert.Greater(t, categories[AgentDryRunCategoryTemplate], 0, "expected template issue")
}

// =============================================================================
// Spec.AgentDryRun — tools counted correctly
// =============================================================================

func TestDryRun_ToolsCounted(t *testing.T) {
	t.Run("no tools", func(t *testing.T) {
		spec := &Spec{
			Name:        "no-tools",
			Description: "Agent without tools",
			Type:        DocumentTypeAgent,
			Execution: &execution.Config{
				Provider: ProviderOpenAI,
				Model:    "gpt-4",
			},
			Body: "body",
		}
		result := spec.AgentDryRun(context.Background(), nil)
		assert.Equal(t, 0, result.ToolsDefined)
	})

	t.Run("functions and mcp servers", func(t *testing.T) {
		spec := &Spec{
			Name:        "tools-agent",
			Description: "Agent with tools",
			Type:        DocumentTypeAgent,
			Execution: &execution.Config{
				Provider: ProviderOpenAI,
				Model:    "gpt-4",
			},
			Body: "body",
			Tools: &ToolsConfig{
				Functions: []*FunctionDef{
					{Name: "fn1"},
					{Name: "fn2"},
				},
				MCPServers: []*MCPServer{
					{Name: "mcp1", URL: "http://localhost"},
				},
			},
		}
		result := spec.AgentDryRun(context.Background(), nil)
		assert.Equal(t, 3, result.ToolsDefined)
	})
}

// =============================================================================
// Spec.AgentDryRun — with compile engine option
// =============================================================================

func TestDryRun_WithCompileEngine(t *testing.T) {
	engine := MustNew()
	spec := &Spec{
		Name:        "engine-agent",
		Description: "Agent with custom engine",
		Type:        DocumentTypeAgent,
		Execution: &execution.Config{
			Provider: ProviderOpenAI,
			Model:    "gpt-4",
		},
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: "Hello {~exons.var name=\"name\" default=\"World\" /~}"},
		},
	}

	opts := &CompileOptions{Engine: engine}
	result := spec.AgentDryRun(context.Background(), opts)
	require.NotNil(t, result)
	assert.True(t, result.OK())
	assert.Equal(t, 1, result.MessageCount)
}
