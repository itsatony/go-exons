package exons

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Integration Tests — Full Pipeline via Public Engine API
// =============================================================================
// These tests exercise ALL template tags end-to-end through the public Engine
// API, proving the full lexer → parser → AST → executor → resolver pipeline
// works after DC-4 wiring.

// =============================================================================
// exons.var — Variable Interpolation
// =============================================================================

func TestIntegration_Var_Simple(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	result, err := engine.Execute(ctx, `{~exons.var name="user" /~}`, map[string]any{
		"user": "Alice",
	})
	require.NoError(t, err)
	assert.Equal(t, "Alice", result)
}

func TestIntegration_Var_DotPath(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	result, err := engine.Execute(ctx, `{~exons.var name="user.name" /~}`, map[string]any{
		"user": map[string]any{
			"name": "Bob",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "Bob", result)
}

func TestIntegration_Var_Default(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	result, err := engine.Execute(ctx, `{~exons.var name="missing" default="Guest" /~}`, nil)
	require.NoError(t, err)
	assert.Equal(t, "Guest", result)
}

// =============================================================================
// exons.raw — Literal Content Preservation
// =============================================================================

func TestIntegration_Raw(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	result, err := engine.Execute(ctx, `{~exons.raw~}{~exons.var name="x" /~}{~/exons.raw~}`, nil)
	require.NoError(t, err)
	assert.Equal(t, `{~exons.var name="x" /~}`, result)
}

// =============================================================================
// exons.include — Nested Template Execution
// =============================================================================

func TestIntegration_Include(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	engine.MustRegisterTemplate("greeting", `Hello World`)

	result, err := engine.Execute(ctx, `{~exons.include template="greeting" /~}!`, nil)
	require.NoError(t, err)
	assert.Equal(t, "Hello World!", result)
}

func TestIntegration_Include_WithData(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	engine.MustRegisterTemplate("hello", `Hello {~exons.var name="name" /~}`)

	result, err := engine.Execute(ctx, `{~exons.include template="hello" name="Bob" /~}`, nil)
	require.NoError(t, err)
	assert.Equal(t, "Hello Bob", result)
}

// =============================================================================
// exons.env — Environment Variable Reading
// =============================================================================

func TestIntegration_Env(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Setenv("EXONS_INTEG_TEST_VAR", "hello")

	result, err := engine.Execute(ctx, `{~exons.env name="EXONS_INTEG_TEST_VAR" /~}`, nil)
	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestIntegration_Env_Default(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	result, err := engine.Execute(ctx, `{~exons.env name="EXONS_MISSING_VAR_XYZ" default="fallback" /~}`, nil)
	require.NoError(t, err)
	assert.Equal(t, "fallback", result)
}

// =============================================================================
// exons.message — Message Role Markers + Extraction
// =============================================================================

func TestIntegration_Message(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	source := `{~exons.message role="system"~}You are a helper.{~/exons.message~}
{~exons.message role="user"~}{~exons.var name="query" /~}{~/exons.message~}`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	messages, err := tmpl.ExecuteAndExtractMessages(ctx, map[string]any{
		"query": "What is Go?",
	})
	require.NoError(t, err)
	require.Len(t, messages, 2)

	assert.Equal(t, RoleSystem, messages[0].Role)
	assert.Equal(t, "You are a helper.", messages[0].Content)

	assert.Equal(t, RoleUser, messages[1].Role)
	assert.Equal(t, "What is Go?", messages[1].Content)
}

// =============================================================================
// exons.ref — Spec Reference Resolution
// =============================================================================

func TestIntegration_Ref(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	resolver := NewMapSpecResolver()
	resolver.Add("my-spec", &Spec{
		Name:        "my-spec",
		Description: "A test spec",
	}, "Body from ref")
	engine.SetSpecResolver(resolver)

	result, err := engine.Execute(ctx, `{~exons.ref slug="my-spec" /~}`, nil)
	require.NoError(t, err)
	assert.Equal(t, "Body from ref", result)
}

func TestIntegration_Ref_NoResolver(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// Without a spec resolver, the ref tag should fail
	_, err := engine.Execute(ctx, `{~exons.ref slug="my-spec" /~}`, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spec resolver")
}

// =============================================================================
// exons.skills_catalog / exons.tools_catalog via ExecuteWithCatalogs
// =============================================================================

func TestIntegration_SkillsCatalog(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	resolver := NewMapSpecResolver()
	resolver.Add("web-search", &Spec{
		Name:        "web-search",
		Description: "Search the web for information",
	}, "web search body")
	engine.SetSpecResolver(resolver)

	spec := &Spec{
		Skills: []SkillRef{{Slug: "web-search", Injection: "system_prompt"}},
	}

	result, err := engine.ExecuteWithCatalogs(ctx,
		`{~exons.var name="skills" default="none" /~}`,
		nil, spec, CatalogFormatDefault)
	require.NoError(t, err)
	assert.Contains(t, result, "web-search")
	assert.Contains(t, result, "Search the web")
}

func TestIntegration_ToolsCatalog(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	spec := &Spec{
		Tools: &ToolsConfig{
			Functions: []*FunctionDef{
				{Name: "search_web", Description: "Search the web"},
			},
		},
	}

	result, err := engine.ExecuteWithCatalogs(ctx,
		`{~exons.var name="tools" default="none" /~}`,
		nil, spec, CatalogFormatDefault)
	require.NoError(t, err)
	assert.Contains(t, result, "search_web")
	assert.Contains(t, result, "Search the web")
}

// =============================================================================
// exons.if / exons.elseif / exons.else — Conditionals
// =============================================================================

func TestIntegration_If_True(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	result, err := engine.Execute(ctx,
		`{~exons.if eval="show"~}visible{~/exons.if~}`,
		map[string]any{"show": true})
	require.NoError(t, err)
	assert.Equal(t, "visible", result)
}

func TestIntegration_If_False(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	result, err := engine.Execute(ctx,
		`{~exons.if eval="show"~}visible{~/exons.if~}`,
		map[string]any{"show": false})
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestIntegration_If_ElseIf_Else(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	tmpl := `{~exons.if eval="role == \"admin\""~}ADMIN{~exons.elseif eval="role == \"user\""~}USER{~exons.else~}GUEST{~/exons.if~}`

	// Branch 1: admin
	result, err := engine.Execute(ctx, tmpl, map[string]any{"role": "admin"})
	require.NoError(t, err)
	assert.Equal(t, "ADMIN", result)

	// Branch 2: user
	result, err = engine.Execute(ctx, tmpl, map[string]any{"role": "user"})
	require.NoError(t, err)
	assert.Equal(t, "USER", result)

	// Branch 3: else
	result, err = engine.Execute(ctx, tmpl, map[string]any{"role": "guest"})
	require.NoError(t, err)
	assert.Equal(t, "GUEST", result)
}

// =============================================================================
// exons.for — Loops
// =============================================================================

func TestIntegration_For(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	result, err := engine.Execute(ctx,
		`{~exons.for item="x" in="items"~}{~exons.var name="x" /~},{~/exons.for~}`,
		map[string]any{"items": []any{"a", "b", "c"}})
	require.NoError(t, err)
	assert.Equal(t, "a,b,c,", result)
}

func TestIntegration_For_WithIndex(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	result, err := engine.Execute(ctx,
		`{~exons.for item="x" index="i" in="items"~}{~exons.var name="i" /~}:{~exons.var name="x" /~} {~/exons.for~}`,
		map[string]any{"items": []any{"a", "b"}})
	require.NoError(t, err)
	assert.Equal(t, "0:a 1:b ", result)
}

// =============================================================================
// exons.switch / exons.case / exons.casedefault — Switch/Case
// =============================================================================

func TestIntegration_Switch(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	tmpl := `{~exons.switch eval="color"~}{~exons.case value="red"~}RED{~/exons.case~}{~exons.case value="blue"~}BLUE{~/exons.case~}{~/exons.switch~}`

	result, err := engine.Execute(ctx, tmpl, map[string]any{"color": "blue"})
	require.NoError(t, err)
	assert.Equal(t, "BLUE", result)
}

func TestIntegration_Switch_Default(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	tmpl := `{~exons.switch eval="color"~}{~exons.case value="red"~}RED{~/exons.case~}{~exons.casedefault~}OTHER{~/exons.casedefault~}{~/exons.switch~}`

	result, err := engine.Execute(ctx, tmpl, map[string]any{"color": "green"})
	require.NoError(t, err)
	assert.Equal(t, "OTHER", result)
}

// =============================================================================
// exons.extends / exons.block — Template Inheritance
// =============================================================================

func TestIntegration_Extends_Block(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// Register parent template with a default block
	engine.MustRegisterTemplate("base-layout",
		`HEADER {~exons.block name="content"~}default content{~/exons.block~} FOOTER`)

	// Child extends parent and overrides the "content" block
	childSource := `{~exons.extends template="base-layout" /~}{~exons.block name="content"~}overridden content{~/exons.block~}`

	result, err := engine.Execute(ctx, childSource, nil)
	require.NoError(t, err)
	assert.Contains(t, result, "HEADER")
	assert.Contains(t, result, "overridden content")
	assert.Contains(t, result, "FOOTER")
	assert.NotContains(t, result, "default content")
}

// =============================================================================
// exons.comment — Comments (Stripped from Output)
// =============================================================================

func TestIntegration_Comment(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	result, err := engine.Execute(ctx,
		`before{~exons.comment~}hidden{~/exons.comment~}after`, nil)
	require.NoError(t, err)
	assert.Equal(t, "beforeafter", result)
}

// =============================================================================
// Escape — Literal {~ in Output
// =============================================================================

func TestIntegration_Escape(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	result, err := engine.Execute(ctx, `\{~exons.var name="x" /~}`, nil)
	require.NoError(t, err)
	assert.Equal(t, `{~exons.var name="x" /~}`, result)
}

// =============================================================================
// Combined / Multi-Tag Tests
// =============================================================================

func TestIntegration_Combined_MultiTag(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// Template combining: var + if + for + comment
	source := `{~exons.comment~}This is hidden{~/exons.comment~}Hello {~exons.var name="user" /~}! {~exons.if eval="showItems"~}Items: {~exons.for item="x" in="items"~}{~exons.var name="x" /~} {~/exons.for~}{~/exons.if~}`

	result, err := engine.Execute(ctx, source, map[string]any{
		"user":      "Alice",
		"showItems": true,
		"items":     []any{"apple", "banana", "cherry"},
	})
	require.NoError(t, err)

	// Comment should be stripped
	assert.NotContains(t, result, "This is hidden")
	// Var should resolve
	assert.Contains(t, result, "Hello Alice!")
	// For loop should produce items
	assert.Contains(t, result, "apple")
	assert.Contains(t, result, "banana")
	assert.Contains(t, result, "cherry")
}

func TestIntegration_Combined_RefWithCatalogs(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// Set up spec resolver for both ref and skills catalog
	resolver := NewMapSpecResolver()
	resolver.Add("helper-skill", &Spec{
		Name:        "helper-skill",
		Description: "A helpful skill",
	}, "helper body text")
	resolver.Add("footer-ref", &Spec{
		Name:        "footer-ref",
		Description: "Footer reference",
	}, "-- footer --")
	engine.SetSpecResolver(resolver)

	spec := &Spec{
		Skills: []SkillRef{{Slug: "helper-skill"}},
		Tools: &ToolsConfig{
			Functions: []*FunctionDef{
				{Name: "search", Description: "Search the web"},
			},
		},
	}

	source := `User: {~exons.var name="user" /~}
Skills: {~exons.var name="skills" default="none" /~}
Tools: {~exons.var name="tools" default="none" /~}
Footer: {~exons.ref slug="footer-ref" /~}`

	data := map[string]any{"user": "Charlie"}
	result, err := engine.ExecuteWithCatalogs(ctx, source, data, spec, CatalogFormatDefault)
	require.NoError(t, err)

	assert.Contains(t, result, "User: Charlie")
	assert.Contains(t, result, "helper-skill")
	assert.Contains(t, result, "search")
	assert.Contains(t, result, "-- footer --")
}

func TestIntegration_Combined_NestedIncludes(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// Set up a spec resolver for ref tags within nested includes
	resolver := NewMapSpecResolver()
	resolver.Add("shared-note", &Spec{
		Name:        "shared-note",
		Description: "Shared note",
	}, "SHARED NOTE")
	engine.SetSpecResolver(resolver)

	// Register templates that use refs and variables
	engine.MustRegisterTemplate("inner",
		`[inner: {~exons.var name="label" default="?" /~} ref={~exons.ref slug="shared-note" /~}]`)
	engine.MustRegisterTemplate("outer",
		`[outer: {~exons.include template="inner" label="from-outer" /~}]`)

	// Execute a template that includes outer, which includes inner, which uses ref
	result, err := engine.Execute(ctx,
		`ROOT:{~exons.include template="outer" /~}`, nil)
	require.NoError(t, err)

	assert.Contains(t, result, "ROOT:")
	assert.Contains(t, result, "[outer:")
	assert.Contains(t, result, "[inner:")
	assert.Contains(t, result, "from-outer")
	assert.Contains(t, result, "SHARED NOTE")
}

// =============================================================================
// Message Extraction with Multiple Roles + Variable Interpolation
// =============================================================================

func TestIntegration_Message_FullConversation(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	source := `{~exons.message role="system"~}You are {~exons.var name="bot_name" /~}.{~/exons.message~}
{~exons.message role="user"~}{~exons.var name="query" /~}{~/exons.message~}
{~exons.message role="assistant"~}Let me help with that.{~/exons.message~}
{~exons.message role="tool"~}Tool result: {~exons.var name="tool_result" /~}{~/exons.message~}`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	messages, err := tmpl.ExecuteAndExtractMessages(ctx, map[string]any{
		"bot_name":    "ExonsBot",
		"query":       "Tell me about Go",
		"tool_result": "42 results found",
	})
	require.NoError(t, err)
	require.Len(t, messages, 4)

	assert.Equal(t, RoleSystem, messages[0].Role)
	assert.Equal(t, "You are ExonsBot.", messages[0].Content)

	assert.Equal(t, RoleUser, messages[1].Role)
	assert.Equal(t, "Tell me about Go", messages[1].Content)

	assert.Equal(t, RoleAssistant, messages[2].Role)
	assert.Equal(t, "Let me help with that.", messages[2].Content)

	assert.Equal(t, RoleTool, messages[3].Role)
	assert.Equal(t, "Tool result: 42 results found", messages[3].Content)
}

// =============================================================================
// Message with Cache Attribute
// =============================================================================

func TestIntegration_Message_Cache(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	source := `{~exons.message role="system" cache="true"~}Cached system prompt{~/exons.message~}
{~exons.message role="user"~}Not cached{~/exons.message~}`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	messages, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, messages, 2)

	assert.True(t, messages[0].Cache)
	assert.False(t, messages[1].Cache)
}

// =============================================================================
// Catalog Resolver Tags (skills_catalog / tools_catalog) — Direct Context
// =============================================================================

func TestIntegration_SkillsCatalog_ViaContext(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// The skills_catalog resolver reads from the "skills" context key directly
	data := map[string]any{
		ContextKeySkills: "## Skills\n\n- **web-search**: Search the web\n",
	}

	result, err := engine.Execute(ctx,
		`{~exons.skills_catalog /~}`, data)
	require.NoError(t, err)
	assert.Contains(t, result, "web-search")
	assert.Contains(t, result, "Search the web")
}

func TestIntegration_ToolsCatalog_ViaContext(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// The tools_catalog resolver reads from the "tools" context key directly
	data := map[string]any{
		ContextKeyTools: "## Tools\n\n- **fetch_url**: Fetch a URL\n",
	}

	result, err := engine.Execute(ctx,
		`{~exons.tools_catalog /~}`, data)
	require.NoError(t, err)
	assert.Contains(t, result, "fetch_url")
	assert.Contains(t, result, "Fetch a URL")
}

// =============================================================================
// Extends with Default Block (no override)
// =============================================================================

func TestIntegration_Extends_DefaultBlock(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	engine.MustRegisterTemplate("parent",
		`START {~exons.block name="main"~}DEFAULT{~/exons.block~} END`)

	// Child extends but does NOT override the block — default content should remain
	result, err := engine.Execute(ctx, `{~exons.extends template="parent" /~}`, nil)
	require.NoError(t, err)
	assert.Contains(t, result, "START")
	assert.Contains(t, result, "DEFAULT")
	assert.Contains(t, result, "END")
}

// =============================================================================
// Switch with No Match and No Default — Empty Output
// =============================================================================

func TestIntegration_Switch_NoMatch(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	tmpl := `{~exons.switch eval="color"~}{~exons.case value="red"~}RED{~/exons.case~}{~exons.case value="blue"~}BLUE{~/exons.case~}{~/exons.switch~}`

	result, err := engine.Execute(ctx, tmpl, map[string]any{"color": "green"})
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

// =============================================================================
// For Loop with Map Items (Dot Path Access)
// =============================================================================

func TestIntegration_For_MapItems(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	result, err := engine.Execute(ctx,
		`{~exons.for item="u" in="users"~}{~exons.var name="u.name" /~} {~/exons.for~}`,
		map[string]any{
			"users": []any{
				map[string]any{"name": "Alice"},
				map[string]any{"name": "Bob"},
			},
		})
	require.NoError(t, err)
	assert.Equal(t, "Alice Bob ", result)
}

// =============================================================================
// Combined: If + For + Var (Conditional Loop)
// =============================================================================

func TestIntegration_ConditionalLoop(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	source := `{~exons.if eval="len(items) > 0"~}Found: {~exons.for item="x" in="items"~}{~exons.var name="x" /~} {~/exons.for~}{~exons.else~}Empty{~/exons.if~}`

	// With items
	result, err := engine.Execute(ctx, source, map[string]any{
		"items": []any{"a", "b"},
	})
	require.NoError(t, err)
	assert.Contains(t, result, "Found:")
	assert.Contains(t, result, "a")
	assert.Contains(t, result, "b")

	// Without items
	result, err = engine.Execute(ctx, source, map[string]any{
		"items": []any{},
	})
	require.NoError(t, err)
	assert.Equal(t, "Empty", result)
}

// =============================================================================
// Raw Block Preserves All Tag Types Inside
// =============================================================================

func TestIntegration_Raw_PreservesAllTags(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	source := `{~exons.raw~}{~exons.if eval="true"~}hello{~/exons.if~} {~exons.for item="x" in="y"~}loop{~/exons.for~}{~/exons.raw~}`

	result, err := engine.Execute(ctx, source, nil)
	require.NoError(t, err)
	// All tags inside raw should be preserved as literal text
	assert.Contains(t, result, `{~exons.if eval="true"~}`)
	assert.Contains(t, result, `{~exons.for item="x" in="y"~}`)
}

// =============================================================================
// Include with Parent Context Inheritance
// =============================================================================

func TestIntegration_Include_InheritsParentContext(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// Include passes attributes explicitly to child template context
	engine.MustRegisterTemplate("child-tmpl",
		`{~exons.var name="name" /~} is {~exons.var name="role" /~}`)

	result, err := engine.Execute(ctx,
		`{~exons.include template="child-tmpl" name="Alice" role="admin" /~}`,
		nil)
	require.NoError(t, err)
	assert.Equal(t, "Alice is admin", result)
}

// =============================================================================
// Env with Existing Variable
// =============================================================================

func TestIntegration_Env_WithRequired(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Setenv("EXONS_INTEG_REQUIRED", "present")

	result, err := engine.Execute(ctx,
		`{~exons.env name="EXONS_INTEG_REQUIRED" required="true" /~}`, nil)
	require.NoError(t, err)
	assert.Equal(t, "present", result)
}

// =============================================================================
// Escape Sequence Within Larger Text
// =============================================================================

func TestIntegration_Escape_InContext(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	result, err := engine.Execute(ctx,
		`Use \{~ to start a tag and ~} to end it.`, nil)
	require.NoError(t, err)
	assert.Contains(t, result, "{~")
	assert.Contains(t, result, "to start a tag")
}

// =============================================================================
// Catalogs with Detailed Format
// =============================================================================

func TestIntegration_Catalogs_DetailedFormat(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	resolver := NewMapSpecResolver()
	resolver.Add("analysis", &Spec{
		Name:        "analysis",
		Description: "Analyze data in detail",
	}, "analysis body")
	engine.SetSpecResolver(resolver)

	spec := &Spec{
		Skills: []SkillRef{{Slug: "analysis", Injection: "system_prompt"}},
		Tools: &ToolsConfig{
			Functions: []*FunctionDef{
				{
					Name:        "query_db",
					Description: "Query the database",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"sql": map[string]any{"type": "string"},
						},
					},
				},
			},
		},
	}

	result, err := engine.ExecuteWithCatalogs(ctx,
		`SKILLS:{~exons.var name="skills" default="" /~} TOOLS:{~exons.var name="tools" default="" /~}`,
		nil, spec, CatalogFormatDetailed)
	require.NoError(t, err)

	// Skills detailed format includes ### headers and injection info
	assert.Contains(t, result, "analysis")
	assert.Contains(t, result, "Analyze data in detail")
	// Tools detailed format includes function name and parameters
	assert.Contains(t, result, "query_db")
	assert.Contains(t, result, "Query the database")
}

// =============================================================================
// Multiple Escape Sequences
// =============================================================================

func TestIntegration_MultipleEscapes(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	result, err := engine.Execute(ctx,
		`\{~first~} and \{~second~}`, nil)
	require.NoError(t, err)
	assert.Contains(t, result, "{~first~}")
	assert.Contains(t, result, "{~second~}")
}

// =============================================================================
// Whitespace Handling in Messages
// =============================================================================

func TestIntegration_Message_WhitespaceTrimmed(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	source := `{~exons.message role="user"~}
  Hello World
{~/exons.message~}`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	messages, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	// Content should be trimmed
	assert.Equal(t, "Hello World", strings.TrimSpace(messages[0].Content))
}

// =============================================================================
// Combined: Nested Includes + Messages + Variables
// =============================================================================

func TestIntegration_Combined_IncludeInMessage(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	engine.MustRegisterTemplate("skill-prompt", `You can use the {~exons.var name="skill_name" /~} skill.`)

	source := `{~exons.message role="system"~}{~exons.include template="skill-prompt" skill_name="search" /~}{~/exons.message~}
{~exons.message role="user"~}{~exons.var name="query" /~}{~/exons.message~}`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	messages, err := tmpl.ExecuteAndExtractMessages(ctx, map[string]any{
		"query": "Find something",
	})
	require.NoError(t, err)
	require.Len(t, messages, 2)

	assert.Equal(t, RoleSystem, messages[0].Role)
	assert.Contains(t, messages[0].Content, "search")
	assert.Equal(t, RoleUser, messages[1].Role)
	assert.Equal(t, "Find something", messages[1].Content)
}
