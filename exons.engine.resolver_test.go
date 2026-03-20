package exons

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Engine SpecResolver Wiring
// =============================================================================

func TestEngine_SetGetSpecResolver(t *testing.T) {
	t.Run("nil by default", func(t *testing.T) {
		engine := MustNew()
		assert.Nil(t, engine.GetSpecResolver())
	})

	t.Run("stores and returns resolver", func(t *testing.T) {
		engine := MustNew()
		resolver := NewMapSpecResolver()
		engine.SetSpecResolver(resolver)
		assert.Equal(t, resolver, engine.GetSpecResolver())
	})

	t.Run("can clear resolver with nil", func(t *testing.T) {
		engine := MustNew()
		resolver := NewMapSpecResolver()
		engine.SetSpecResolver(resolver)
		assert.NotNil(t, engine.GetSpecResolver())

		engine.SetSpecResolver(nil)
		assert.Nil(t, engine.GetSpecResolver())
	})
}

func TestEngine_Execute_WithSpecResolver(t *testing.T) {
	ctx := context.Background()

	t.Run("ref tag resolves with SpecResolver", func(t *testing.T) {
		resolver := NewMapSpecResolver()
		resolver.Add("greeting", &Spec{
			Name:        "greeting",
			Description: "A greeting skill",
		}, "Hello from ref!")

		engine := MustNew()
		engine.SetSpecResolver(resolver)

		result, err := engine.Execute(ctx, `{~exons.ref slug="greeting" /~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, "Hello from ref!", result)
	})

	t.Run("works without SpecResolver set", func(t *testing.T) {
		engine := MustNew()

		result, err := engine.Execute(ctx, `Hello {~exons.var name="user" default="World" /~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, "Hello World", result)
	})

	t.Run("works with data and SpecResolver", func(t *testing.T) {
		resolver := NewMapSpecResolver()
		resolver.Add("footer", &Spec{
			Name:        "footer",
			Description: "A footer",
		}, "-- end --")

		engine := MustNew()
		engine.SetSpecResolver(resolver)

		data := map[string]any{"name": "Alice"}
		result, err := engine.Execute(ctx, `Hi {~exons.var name="name" /~}! {~exons.ref slug="footer" /~}`, data)
		require.NoError(t, err)
		assert.Equal(t, "Hi Alice! -- end --", result)
	})
}

func TestEngine_ExecuteTemplate_WithSpecResolver(t *testing.T) {
	ctx := context.Background()

	t.Run("injects SpecResolver into nested template context", func(t *testing.T) {
		resolver := NewMapSpecResolver()
		resolver.Add("shared-note", &Spec{
			Name:        "shared-note",
			Description: "Shared note",
		}, "This is a shared note.")

		engine := MustNew()
		engine.SetSpecResolver(resolver)

		// Register a template that uses exons.ref
		err := engine.RegisterTemplate("with-ref", `Note: {~exons.ref slug="shared-note" /~}`)
		require.NoError(t, err)

		// ExecuteTemplate should also inject the resolver
		result, err := engine.ExecuteTemplate(ctx, "with-ref", nil)
		require.NoError(t, err)
		assert.Equal(t, "Note: This is a shared note.", result)
	})

	t.Run("works without SpecResolver", func(t *testing.T) {
		engine := MustNew()
		err := engine.RegisterTemplate("simple", `Hello {~exons.var name="who" default="World" /~}`)
		require.NoError(t, err)

		result, err := engine.ExecuteTemplate(ctx, "simple", nil)
		require.NoError(t, err)
		assert.Equal(t, "Hello World", result)
	})
}

// =============================================================================
// ExecuteWithCatalogs
// =============================================================================

func TestEngine_ExecuteWithCatalogs(t *testing.T) {
	ctx := context.Background()

	t.Run("injects skills catalog", func(t *testing.T) {
		resolver := NewMapSpecResolver()
		resolver.Add("web-search", &Spec{
			Name:        "web-search",
			Description: "Search the web for information",
		}, "web search body")

		engine := MustNew()
		engine.SetSpecResolver(resolver)

		spec := &Spec{
			Skills: []SkillRef{{Slug: "web-search"}},
		}

		result, err := engine.ExecuteWithCatalogs(ctx,
			`{~exons.var name="skills" default="none" /~}`,
			nil, spec, CatalogFormatDefault)
		require.NoError(t, err)
		assert.Contains(t, result, "web-search")
		assert.Contains(t, result, "Search the web")
	})

	t.Run("injects tools catalog", func(t *testing.T) {
		engine := MustNew()

		spec := &Spec{
			Tools: &ToolsConfig{
				Functions: []*FunctionDef{
					{Name: "search", Description: "Search the web"},
				},
			},
		}

		result, err := engine.ExecuteWithCatalogs(ctx,
			`{~exons.var name="tools" default="none" /~}`,
			nil, spec, CatalogFormatDefault)
		require.NoError(t, err)
		assert.Contains(t, result, "search")
		assert.Contains(t, result, "Search the web")
	})

	t.Run("injects both skills and tools", func(t *testing.T) {
		resolver := NewMapSpecResolver()
		resolver.Add("summarize", &Spec{
			Name:        "summarize",
			Description: "Summarize text",
		}, "summarize body")

		engine := MustNew()
		engine.SetSpecResolver(resolver)

		spec := &Spec{
			Skills: []SkillRef{{Slug: "summarize"}},
			Tools: &ToolsConfig{
				Functions: []*FunctionDef{
					{Name: "fetch_url", Description: "Fetch a URL"},
				},
			},
		}

		result, err := engine.ExecuteWithCatalogs(ctx,
			`Skills: {~exons.var name="skills" default="" /~} | Tools: {~exons.var name="tools" default="" /~}`,
			nil, spec, CatalogFormatDefault)
		require.NoError(t, err)
		assert.Contains(t, result, "summarize")
		assert.Contains(t, result, "fetch_url")
	})

	t.Run("nil spec works without error", func(t *testing.T) {
		engine := MustNew()

		result, err := engine.ExecuteWithCatalogs(ctx,
			`{~exons.var name="skills" default="none" /~}`,
			nil, nil, CatalogFormatDefault)
		require.NoError(t, err)
		assert.Equal(t, "none", result)
	})

	t.Run("nil data is initialized", func(t *testing.T) {
		engine := MustNew()

		spec := &Spec{
			Tools: &ToolsConfig{
				Functions: []*FunctionDef{
					{Name: "my_tool", Description: "A tool"},
				},
			},
		}

		result, err := engine.ExecuteWithCatalogs(ctx,
			`{~exons.var name="tools" default="none" /~}`,
			nil, spec, CatalogFormatDefault)
		require.NoError(t, err)
		assert.Contains(t, result, "my_tool")
	})

	t.Run("empty skills and tools", func(t *testing.T) {
		engine := MustNew()

		spec := &Spec{
			Skills: []SkillRef{},
			Tools:  &ToolsConfig{},
		}

		result, err := engine.ExecuteWithCatalogs(ctx,
			`{~exons.var name="skills" default="no-skills" /~} {~exons.var name="tools" default="no-tools" /~}`,
			nil, spec, CatalogFormatDefault)
		require.NoError(t, err)
		assert.Equal(t, "no-skills no-tools", result)
	})

	t.Run("uses compact format", func(t *testing.T) {
		resolver := NewMapSpecResolver()
		resolver.Add("analyze", &Spec{
			Name:        "analyze",
			Description: "Analyze data",
		}, "body")

		engine := MustNew()
		engine.SetSpecResolver(resolver)

		spec := &Spec{
			Skills: []SkillRef{{Slug: "analyze"}},
		}

		result, err := engine.ExecuteWithCatalogs(ctx,
			`{~exons.var name="skills" default="" /~}`,
			nil, spec, CatalogFormatCompact)
		require.NoError(t, err)
		assert.Contains(t, result, "analyze")
		// Compact format should not contain markdown headers
		assert.NotContains(t, result, "## Skills")
	})

	t.Run("does not overwrite existing data keys", func(t *testing.T) {
		engine := MustNew()

		spec := &Spec{
			Tools: &ToolsConfig{
				Functions: []*FunctionDef{
					{Name: "tool_a", Description: "Tool A"},
				},
			},
		}

		// Pre-set the tools key — ExecuteWithCatalogs will overwrite it
		data := map[string]any{
			"greeting": "hello",
		}

		result, err := engine.ExecuteWithCatalogs(ctx,
			`{~exons.var name="greeting" /~} {~exons.var name="tools" default="none" /~}`,
			data, spec, CatalogFormatDefault)
		require.NoError(t, err)
		assert.Contains(t, result, "hello")
		assert.Contains(t, result, "tool_a")
	})

	t.Run("SpecResolver is also injected for ref tags", func(t *testing.T) {
		resolver := NewMapSpecResolver()
		resolver.Add("helper", &Spec{
			Name:        "helper",
			Description: "Helper skill",
		}, "helper content here")

		engine := MustNew()
		engine.SetSpecResolver(resolver)

		spec := &Spec{
			Skills: []SkillRef{{Slug: "helper"}},
		}

		// Use both catalog variable and ref tag
		result, err := engine.ExecuteWithCatalogs(ctx,
			`Ref: {~exons.ref slug="helper" /~}`,
			nil, spec, CatalogFormatDefault)
		require.NoError(t, err)
		assert.Equal(t, "Ref: helper content here", result)
	})
}
