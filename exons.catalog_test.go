package exons

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- generateSkillsCatalog tests ---

func Test_generateSkillsCatalog_DefaultFormat(t *testing.T) {
	ctx := context.Background()
	resolver := NewMapSpecResolver()
	resolver.Add("web-search", &Spec{
		Name:        "web-search",
		Description: "Search the web for information",
	}, "")
	resolver.Add("summarizer", &Spec{
		Name:        "summarizer",
		Description: "Summarize text content",
	}, "")

	skills := []SkillRef{
		{Slug: "web-search", Injection: string(SkillInjectionSystemPrompt)},
		{Slug: "summarizer", Injection: string(SkillInjectionUserContext)},
	}

	result, err := generateSkillsCatalog(ctx, skills, resolver, CatalogFormatDefault)
	require.NoError(t, err)
	assert.Contains(t, result, "## Skills")
	assert.Contains(t, result, "**web-search**")
	assert.Contains(t, result, "Search the web for information")
	assert.Contains(t, result, "**summarizer**")
	assert.Contains(t, result, "Summarize text content")
}

func Test_generateSkillsCatalog_DetailedFormat(t *testing.T) {
	ctx := context.Background()
	resolver := NewMapSpecResolver()
	resolver.Add("web-search", &Spec{
		Name:        "web-search",
		Description: "Search the web for information",
	}, "")

	skills := []SkillRef{
		{Slug: "web-search", Injection: string(SkillInjectionSystemPrompt)},
	}

	result, err := generateSkillsCatalog(ctx, skills, resolver, CatalogFormatDetailed)
	require.NoError(t, err)
	assert.Contains(t, result, "### web-search")
	assert.Contains(t, result, "Search the web for information")
	assert.Contains(t, result, "Injection: system_prompt")
}

func Test_generateSkillsCatalog_CompactFormat(t *testing.T) {
	ctx := context.Background()
	resolver := NewMapSpecResolver()
	resolver.Add("web-search", &Spec{
		Name:        "web-search",
		Description: "Search the web for information",
	}, "")

	skills := []SkillRef{
		{Slug: "web-search"},
	}

	result, err := generateSkillsCatalog(ctx, skills, resolver, CatalogFormatCompact)
	require.NoError(t, err)
	assert.Contains(t, result, "web-search - Search the web for information")
}

func Test_generateSkillsCatalog_FunctionCallingReturnsError(t *testing.T) {
	ctx := context.Background()
	skills := []SkillRef{
		{Slug: "some-skill"},
	}

	result, err := generateSkillsCatalog(ctx, skills, nil, CatalogFormatFunctionCalling)
	require.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), ErrMsgCatalogFuncCallingSkills)
}

func Test_generateSkillsCatalog_EmptySkillsReturnsEmpty(t *testing.T) {
	ctx := context.Background()

	result, err := generateSkillsCatalog(ctx, []SkillRef{}, nil, CatalogFormatDefault)
	require.NoError(t, err)
	assert.Empty(t, result)

	result, err = generateSkillsCatalog(ctx, nil, nil, CatalogFormatDefault)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func Test_generateSkillsCatalog_ResolverErrorsAreNonFatal(t *testing.T) {
	ctx := context.Background()
	// Use a noop resolver that always returns not-found
	resolver := NewNoopSpecResolver()

	skills := []SkillRef{
		{Slug: "missing-skill"},
	}

	result, err := generateSkillsCatalog(ctx, skills, resolver, CatalogFormatDefault)
	require.NoError(t, err)
	assert.Contains(t, result, "**missing-skill**")
	// No description since resolver failed
	assert.NotContains(t, result, ": ")
}

func Test_generateSkillsCatalog_NilResolverStillWorks(t *testing.T) {
	ctx := context.Background()
	skills := []SkillRef{
		{Slug: "my-skill", Injection: string(SkillInjectionNone)},
	}

	result, err := generateSkillsCatalog(ctx, skills, nil, CatalogFormatDefault)
	require.NoError(t, err)
	assert.Contains(t, result, "**my-skill**")
}

func Test_generateSkillsCatalog_UnknownFormatReturnsError(t *testing.T) {
	ctx := context.Background()
	skills := []SkillRef{
		{Slug: "some-skill"},
	}

	result, err := generateSkillsCatalog(ctx, skills, nil, CatalogFormat("unknown_format"))
	require.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), ErrMsgCatalogUnknownFormat)
}

// --- generateToolsCatalog tests ---

func Test_generateToolsCatalog_DefaultFormatWithFunctions(t *testing.T) {
	tools := &ToolsConfig{
		Functions: []*FunctionDef{
			{
				Name:        "search_web",
				Description: "Search the web for information",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{"type": "string"},
					},
				},
			},
			{
				Name:        "get_weather",
				Description: "Get weather for a location",
			},
		},
	}

	result, err := generateToolsCatalog(tools, CatalogFormatDefault)
	require.NoError(t, err)
	assert.Contains(t, result, "## Tools")
	assert.Contains(t, result, "**search_web**")
	assert.Contains(t, result, "Search the web for information")
	assert.Contains(t, result, "**get_weather**")
}

func Test_generateToolsCatalog_DefaultFormatWithMCPServers(t *testing.T) {
	tools := &ToolsConfig{
		MCPServers: []*MCPServer{
			{Name: "code-executor", URL: "https://mcp.example.com/code"},
		},
	}

	result, err := generateToolsCatalog(tools, CatalogFormatDefault)
	require.NoError(t, err)
	assert.Contains(t, result, "[MCP] code-executor")
	assert.Contains(t, result, "https://mcp.example.com/code")
}

func Test_generateToolsCatalog_DetailedFormatWithFunctions(t *testing.T) {
	tools := &ToolsConfig{
		Functions: []*FunctionDef{
			{
				Name:        "search_web",
				Description: "Search the web",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{"type": "string"},
					},
					"required": []string{"query"},
				},
			},
		},
	}

	result, err := generateToolsCatalog(tools, CatalogFormatDetailed)
	require.NoError(t, err)
	assert.Contains(t, result, "### search_web")
	assert.Contains(t, result, "Search the web")
	assert.Contains(t, result, "```json")
	assert.Contains(t, result, "query")
}

func Test_generateToolsCatalog_CompactFormat(t *testing.T) {
	tools := &ToolsConfig{
		Functions: []*FunctionDef{
			{Name: "search_web", Description: "Search the web for information"},
		},
		MCPServers: []*MCPServer{
			{Name: "code-runner", URL: "https://mcp.example.com"},
		},
	}

	result, err := generateToolsCatalog(tools, CatalogFormatCompact)
	require.NoError(t, err)
	assert.Contains(t, result, "search_web - Search the web for information")
	assert.Contains(t, result, "[MCP] code-runner")
}

func Test_generateToolsCatalog_FunctionCallingFormatProducesValidJSON(t *testing.T) {
	tools := &ToolsConfig{
		Functions: []*FunctionDef{
			{
				Name:        "search_web",
				Description: "Search the web",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{"type": "string"},
					},
				},
			},
			{
				Name:        "get_weather",
				Description: "Get weather",
			},
		},
	}

	result, err := generateToolsCatalog(tools, CatalogFormatFunctionCalling)
	require.NoError(t, err)

	// Verify it's valid JSON
	var parsed []map[string]any
	err = json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err)
	assert.Len(t, parsed, 2)

	// Verify OpenAI tool format
	assert.Equal(t, "function", parsed[0]["type"])
	fn, ok := parsed[0]["function"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "search_web", fn["name"])
}

func Test_generateToolsCatalog_NilToolsReturnsEmpty(t *testing.T) {
	result, err := generateToolsCatalog(nil, CatalogFormatDefault)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func Test_generateToolsCatalog_EmptyToolsReturnsEmpty(t *testing.T) {
	tools := &ToolsConfig{}
	result, err := generateToolsCatalog(tools, CatalogFormatDefault)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func Test_generateToolsCatalog_UnknownFormatReturnsError(t *testing.T) {
	tools := &ToolsConfig{
		Functions: []*FunctionDef{
			{Name: "search_web", Description: "Search"},
		},
	}

	result, err := generateToolsCatalog(tools, CatalogFormat("weird_format"))
	require.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), ErrMsgCatalogUnknownFormat)
}

// --- truncateString tests ---

func TestTruncateString_ShortStringPassesThrough(t *testing.T) {
	result := truncateString("short", 80)
	assert.Equal(t, "short", result)
}

func TestTruncateString_LongStringTruncated(t *testing.T) {
	long := strings.Repeat("a", 100)
	result := truncateString(long, 80)
	assert.Len(t, result, 80)
	assert.True(t, strings.HasSuffix(result, "..."))
}

func TestTruncateString_ExactLengthPassesThrough(t *testing.T) {
	exact := strings.Repeat("x", 80)
	result := truncateString(exact, 80)
	assert.Equal(t, exact, result)
	assert.Len(t, result, 80)
}

func TestTruncateString_VerySmallMaxLen(t *testing.T) {
	result := truncateString("hello world", 3)
	assert.Equal(t, "hel", result)
}

func Test_generateToolsCatalog_DetailedFormatWithMCPServers(t *testing.T) {
	tools := &ToolsConfig{
		MCPServers: []*MCPServer{
			{Name: "code-exec", URL: "https://mcp.example.com/exec"},
		},
	}

	result, err := generateToolsCatalog(tools, CatalogFormatDetailed)
	require.NoError(t, err)
	assert.Contains(t, result, "### [MCP] code-exec")
	assert.Contains(t, result, "URL: https://mcp.example.com/exec")
}

func Test_generateSkillsCatalog_DetailedFormatNoInjection(t *testing.T) {
	ctx := context.Background()
	resolver := NewMapSpecResolver()
	resolver.Add("plain-skill", &Spec{
		Name:        "plain-skill",
		Description: "A plain skill",
	}, "")

	skills := []SkillRef{
		{Slug: "plain-skill"},
	}

	result, err := generateSkillsCatalog(ctx, skills, resolver, CatalogFormatDetailed)
	require.NoError(t, err)
	assert.Contains(t, result, "### plain-skill")
	assert.Contains(t, result, "A plain skill")
	assert.NotContains(t, result, "Injection:")
}

func Test_generateSkillsCatalog_CompactLongDescription(t *testing.T) {
	ctx := context.Background()
	resolver := NewMapSpecResolver()
	longDesc := strings.Repeat("a", 200)
	resolver.Add("verbose-skill", &Spec{
		Name:        "verbose-skill",
		Description: longDesc,
	}, "")

	skills := []SkillRef{
		{Slug: "verbose-skill"},
	}

	result, err := generateSkillsCatalog(ctx, skills, resolver, CatalogFormatCompact)
	require.NoError(t, err)
	// Description should be truncated
	assert.True(t, len(result) < 200+30) // slug + separator + truncated desc + newline
	assert.Contains(t, result, "...")
}

func Test_generateToolsCatalog_FunctionCallingIgnoresMCPServers(t *testing.T) {
	tools := &ToolsConfig{
		Functions: []*FunctionDef{
			{Name: "search", Description: "Search"},
		},
		MCPServers: []*MCPServer{
			{Name: "code-exec", URL: "https://mcp.example.com"},
		},
	}

	result, err := generateToolsCatalog(tools, CatalogFormatFunctionCalling)
	require.NoError(t, err)

	var parsed []map[string]any
	err = json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err)
	// Only functions, no MCP servers
	assert.Len(t, parsed, 1)
	assert.Equal(t, "function", parsed[0]["type"])
}

func Test_generateToolsCatalog_NilFunctionDefsSkipped(t *testing.T) {
	tools := &ToolsConfig{
		Functions: []*FunctionDef{
			{Name: "valid", Description: "Valid function"},
			nil,
			{Name: "also-valid", Description: "Also valid"},
		},
	}

	result, err := generateToolsCatalog(tools, CatalogFormatDefault)
	require.NoError(t, err)
	assert.Contains(t, result, "**valid**")
	assert.Contains(t, result, "**also-valid**")
}
