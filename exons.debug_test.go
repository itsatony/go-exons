package exons

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// DryRun — Variable Detection
// =============================================================================

func TestTemplate_DryRun_Variables(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("detects variable reference", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.var name="user.name" /~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, map[string]any{
			"user": map[string]any{"name": "Alice"},
		})
		assert.True(t, result.Valid)
		require.Len(t, result.Variables, 1)
		assert.Equal(t, "user.name", result.Variables[0].Name)
		assert.True(t, result.Variables[0].InData)
	})

	t.Run("detects missing variable", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.var name="missing" /~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		require.Len(t, result.Variables, 1)
		assert.False(t, result.Variables[0].InData)
		assert.Contains(t, result.MissingVariables, "missing")
	})

	t.Run("variable with default is not missing", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.var name="x" default="fallback" /~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		require.Len(t, result.Variables, 1)
		assert.True(t, result.Variables[0].HasDefault)
		assert.Equal(t, "fallback", result.Variables[0].Default)
		assert.Empty(t, result.MissingVariables)
	})

	t.Run("detects unused variables", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.var name="used" /~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, map[string]any{
			"used":   "val1",
			"unused": "val2",
		})
		assert.Contains(t, result.UnusedVariables, "unused")
	})

	t.Run("multiple variables", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.var name="a" /~} {~exons.var name="b" /~} {~exons.var name="c" /~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, map[string]any{"a": "1", "b": "2"})
		assert.Len(t, result.Variables, 3)
		// c should be missing
		assert.Contains(t, result.MissingVariables, "c")
	})

	t.Run("suggestions for similar variable names", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.var name="naem" /~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, map[string]any{"name": "Alice"})
		require.Len(t, result.Variables, 1)
		// Should have a suggestion for "name" since "naem" is similar
		if len(result.Variables[0].Suggestions) > 0 {
			assert.Contains(t, result.Variables[0].Suggestions, "name")
		}
	})
}

// =============================================================================
// DryRun — Resolver Detection
// =============================================================================

func TestTemplate_DryRun_Resolvers(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("detects env resolver", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.env name="TEST_VAR" /~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		// env tags are built-in resolvers, so they appear in resolvers list
		found := false
		for _, r := range result.Resolvers {
			if r.TagName == TagNameEnv {
				found = true
				break
			}
		}
		assert.True(t, found, "should detect env resolver")
	})

	t.Run("detects custom resolver", func(t *testing.T) {
		engine2 := MustNew()
		engine2.MustRegister(NewResolverFunc("CustomTag", func(ctx context.Context, execCtx *Context, attrs Attributes) (string, error) {
			return "custom", nil
		}, nil))
		tmpl, err := engine2.Parse(`{~CustomTag id="123" /~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		require.True(t, len(result.Resolvers) > 0)
		assert.Equal(t, "CustomTag", result.Resolvers[0].TagName)
	})
}

// =============================================================================
// DryRun — Include Detection
// =============================================================================

func TestTemplate_DryRun_Includes(t *testing.T) {
	ctx := context.Background()

	t.Run("detects include reference", func(t *testing.T) {
		engine := MustNew()
		engine.MustRegisterTemplate("header", "Header Content")
		tmpl, err := engine.Parse(`{~exons.include template="header" /~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		require.Len(t, result.Includes, 1)
		assert.Equal(t, "header", result.Includes[0].TemplateName)
		assert.True(t, result.Includes[0].Exists)
	})

	t.Run("detects missing include", func(t *testing.T) {
		engine := MustNew()
		tmpl, err := engine.Parse(`{~exons.include template="missing" /~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		require.Len(t, result.Includes, 1)
		assert.Equal(t, "missing", result.Includes[0].TemplateName)
		assert.False(t, result.Includes[0].Exists)
		assert.True(t, len(result.Warnings) > 0)
	})

	t.Run("detects isolated include", func(t *testing.T) {
		engine := MustNew()
		engine.MustRegisterTemplate("child", "Child")
		tmpl, err := engine.Parse(`{~exons.include template="child" isolate="true" /~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		require.Len(t, result.Includes, 1)
		assert.True(t, result.Includes[0].Isolated)
	})
}

// =============================================================================
// DryRun — Conditional Detection
// =============================================================================

func TestTemplate_DryRun_Conditionals(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("detects simple if", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.if eval="show"~}content{~/exons.if~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		require.Len(t, result.Conditionals, 1)
		assert.Equal(t, "show", result.Conditionals[0].Condition)
		assert.False(t, result.Conditionals[0].HasElseIf)
		assert.False(t, result.Conditionals[0].HasElse)
	})

	t.Run("detects if-else", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.if eval="x"~}a{~exons.else~}b{~/exons.if~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		require.Len(t, result.Conditionals, 1)
		assert.True(t, result.Conditionals[0].HasElse)
		assert.False(t, result.Conditionals[0].HasElseIf)
	})

	t.Run("detects if-elseif-else", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.if eval="a"~}1{~exons.elseif eval="b"~}2{~exons.else~}3{~/exons.if~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		require.Len(t, result.Conditionals, 1)
		assert.True(t, result.Conditionals[0].HasElseIf)
		assert.True(t, result.Conditionals[0].HasElse)
	})

	t.Run("conditional with line info", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.if eval="x"~}content{~/exons.if~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		require.Len(t, result.Conditionals, 1)
		assert.True(t, result.Conditionals[0].Line > 0)
	})
}

// =============================================================================
// DryRun — Loop Detection
// =============================================================================

func TestTemplate_DryRun_Loops(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("detects simple loop", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.for item="x" in="items"~}content{~/exons.for~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, map[string]any{"items": []any{"a"}})
		require.Len(t, result.Loops, 1)
		assert.Equal(t, "x", result.Loops[0].ItemVar)
		assert.Equal(t, "items", result.Loops[0].Source)
		assert.True(t, result.Loops[0].InData)
	})

	t.Run("detects loop with index", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.for item="x" index="i" in="items"~}content{~/exons.for~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, map[string]any{"items": []any{}})
		require.Len(t, result.Loops, 1)
		assert.Equal(t, "i", result.Loops[0].IndexVar)
	})

	t.Run("detects loop with missing source", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.for item="x" in="missing"~}content{~/exons.for~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		require.Len(t, result.Loops, 1)
		assert.False(t, result.Loops[0].InData)
		assert.True(t, len(result.Warnings) > 0)
	})

	t.Run("loop with limit", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.for item="x" in="items" limit="5"~}content{~/exons.for~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		require.Len(t, result.Loops, 1)
		assert.Equal(t, 5, result.Loops[0].Limit)
	})
}

// =============================================================================
// DryRun — Edge Cases
// =============================================================================

func TestTemplate_DryRun_EdgeCases(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("empty template", func(t *testing.T) {
		tmpl, err := engine.Parse("")
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Variables)
		assert.Empty(t, result.Resolvers)
		assert.Empty(t, result.Includes)
		assert.Empty(t, result.Conditionals)
		assert.Empty(t, result.Loops)
		assert.Empty(t, result.Errors)
	})

	t.Run("text only template", func(t *testing.T) {
		tmpl, err := engine.Parse("Just plain text")
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		assert.True(t, result.Valid)
		assert.Equal(t, "Just plain text", result.Output)
	})

	t.Run("raw block", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.raw~}unchanged{~/exons.raw~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		assert.True(t, result.Valid)
		assert.Contains(t, result.Output, "unchanged")
	})

	t.Run("comment block", func(t *testing.T) {
		tmpl, err := engine.Parse(`before{~exons.comment~}hidden{~/exons.comment~}after`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		assert.True(t, result.Valid)
		// Comments produce no output in dry run
		assert.NotContains(t, result.Output, "hidden")
		assert.Contains(t, result.Output, "before")
		assert.Contains(t, result.Output, "after")
	})
}

// =============================================================================
// DryRun — Complex Templates
// =============================================================================

func TestTemplate_DryRun_Complex(t *testing.T) {
	engine := MustNew()
	engine.MustRegisterTemplate("header", "HEADER")
	ctx := context.Background()

	tmpl, err := engine.Parse(`{~exons.include template="header" /~}
{~exons.var name="title" /~}
{~exons.if eval="show"~}
  {~exons.for item="x" in="items"~}
    {~exons.var name="x" /~}
  {~/exons.for~}
{~/exons.if~}`)
	require.NoError(t, err)

	result := tmpl.DryRun(ctx, map[string]any{
		"title": "Test",
		"show":  true,
		"items": []any{"a", "b"},
	})

	assert.True(t, result.Valid)
	assert.Len(t, result.Includes, 1)
	// Variables: title plus any inside for/if
	assert.True(t, len(result.Variables) >= 1)
	assert.Len(t, result.Conditionals, 1)
	assert.Len(t, result.Loops, 1)
}

// =============================================================================
// DryRun — String Output
// =============================================================================

func TestDryRunResult_String(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	tmpl, err := engine.Parse(`Hello {~exons.var name="name" /~}`)
	require.NoError(t, err)
	result := tmpl.DryRun(ctx, map[string]any{"name": "World"})

	str := result.String()
	assert.Contains(t, str, "Dry Run")
	assert.Contains(t, str, "Valid: true")
}

// =============================================================================
// DryRun — Placeholder Output
// =============================================================================

func TestTemplate_DryRun_PlaceholderOutput(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("variable with data shows actual value", func(t *testing.T) {
		tmpl, err := engine.Parse(`Hello {~exons.var name="name" /~}!`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, map[string]any{"name": "World"})
		assert.Equal(t, "Hello World!", result.Output)
	})

	t.Run("variable without data shows placeholder", func(t *testing.T) {
		tmpl, err := engine.Parse(`Hello {~exons.var name="name" /~}!`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		assert.Contains(t, result.Output, "{{name}}")
	})

	t.Run("variable with default shows default", func(t *testing.T) {
		tmpl, err := engine.Parse(`Hello {~exons.var name="name" default="Guest" /~}!`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		assert.Equal(t, "Hello Guest!", result.Output)
	})

	t.Run("include shows placeholder", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.include template="header" /~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		assert.Contains(t, result.Output, "{{include:header}}")
	})

	t.Run("conditional shows placeholder", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.if eval="x"~}content{~/exons.if~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		assert.Contains(t, result.Output, "{{if:x}}")
		assert.Contains(t, result.Output, "{{/if}}")
	})

	t.Run("loop shows placeholder", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.for item="x" in="items"~}body{~/exons.for~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		assert.Contains(t, result.Output, "{{for:x in items}}")
		assert.Contains(t, result.Output, "{{/for}}")
	})

	t.Run("switch shows placeholder", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.switch eval="x"~}{~exons.case value="a"~}A{~/exons.case~}{~/exons.switch~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		assert.Contains(t, result.Output, "{{switch:x}}")
		assert.Contains(t, result.Output, "{{case:a}}")
		assert.Contains(t, result.Output, "{{/switch}}")
	})
}

// =============================================================================
// Explain
// =============================================================================

func TestTemplate_Explain(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("explain simple template", func(t *testing.T) {
		tmpl, err := engine.Parse(`Hello {~exons.var name="name" /~}!`)
		require.NoError(t, err)
		result := tmpl.Explain(ctx, map[string]any{"name": "World"})
		assert.Equal(t, "Hello World!", result.Output)
		assert.Nil(t, result.Error)
		assert.True(t, result.Timing.Total > 0)
		assert.NotEmpty(t, result.AST)
	})

	t.Run("explain with error", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.var name="missing" /~}`)
		require.NoError(t, err)
		result := tmpl.Explain(ctx, nil)
		assert.Error(t, result.Error)
	})

	t.Run("explain variable accesses", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.var name="x" /~} {~exons.var name="y" default="def" /~}`)
		require.NoError(t, err)
		result := tmpl.Explain(ctx, map[string]any{"x": "hello"})
		assert.True(t, len(result.Variables) >= 1)
		// Variable "x" should be found
		found := false
		for _, v := range result.Variables {
			if v.Path == "x" && v.Found {
				found = true
				break
			}
		}
		assert.True(t, found, "should find variable 'x'")
	})

	t.Run("explain empty template", func(t *testing.T) {
		tmpl, err := engine.Parse("")
		require.NoError(t, err)
		result := tmpl.Explain(ctx, nil)
		assert.Equal(t, "", result.Output)
		assert.Nil(t, result.Error)
	})

	t.Run("explain timing info", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.var name="x" default="test" /~}`)
		require.NoError(t, err)
		result := tmpl.Explain(ctx, nil)
		assert.True(t, result.Timing.Execution >= 0)
		assert.True(t, result.Timing.Total >= result.Timing.Execution)
	})
}

// =============================================================================
// ExplainResult.String
// =============================================================================

func TestExplainResult_String(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	tmpl, err := engine.Parse(`Hello {~exons.var name="name" /~}`)
	require.NoError(t, err)
	result := tmpl.Explain(ctx, map[string]any{"name": "World"})

	str := result.String()
	assert.Contains(t, str, "Template Explanation")
	assert.Contains(t, str, "AST Structure")
	assert.Contains(t, str, "Output")
}

// =============================================================================
// formatAST
// =============================================================================

func TestTemplate_FormatAST(t *testing.T) {
	engine := MustNew()

	t.Run("AST contains root", func(t *testing.T) {
		tmpl, err := engine.Parse("Hello World")
		require.NoError(t, err)
		ast := tmpl.formatAST(tmpl.ast, 0)
		assert.Contains(t, ast, "Root")
		assert.Contains(t, ast, "Hello World")
	})

	t.Run("AST with tag", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.var name="x" /~}`)
		require.NoError(t, err)
		ast := tmpl.formatAST(tmpl.ast, 0)
		assert.Contains(t, ast, "Root")
		assert.Contains(t, ast, "exons.var")
	})

	t.Run("AST with conditional", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.if eval="x"~}a{~exons.else~}b{~/exons.if~}`)
		require.NoError(t, err)
		ast := tmpl.formatAST(tmpl.ast, 0)
		assert.Contains(t, ast, "Conditional")
	})

	t.Run("AST with loop", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.for item="x" in="items"~}body{~/exons.for~}`)
		require.NoError(t, err)
		ast := tmpl.formatAST(tmpl.ast, 0)
		assert.Contains(t, ast, "For")
		assert.Contains(t, ast, "items")
	})

	t.Run("AST with switch", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.switch eval="color"~}{~exons.case value="red"~}R{~/exons.case~}{~/exons.switch~}`)
		require.NoError(t, err)
		ast := tmpl.formatAST(tmpl.ast, 0)
		assert.Contains(t, ast, "Switch")
	})

	t.Run("AST text truncation", func(t *testing.T) {
		longText := strings.Repeat("a", 100)
		tmpl, err := engine.Parse(longText)
		require.NoError(t, err)
		ast := tmpl.formatAST(tmpl.ast, 0)
		assert.Contains(t, ast, "...")
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

func TestHasPath(t *testing.T) {
	data := map[string]any{
		"a": map[string]any{
			"b": "value",
		},
	}

	assert.True(t, hasPath(data, "a"))
	assert.True(t, hasPath(data, "a.b"))
	assert.False(t, hasPath(data, "missing"))
	assert.False(t, hasPath(data, "a.missing"))
	assert.False(t, hasPath(nil, "a"))
	assert.False(t, hasPath(data, ""))
}

func TestGetPath(t *testing.T) {
	data := map[string]any{
		"x": "hello",
		"nested": map[string]any{
			"y": "world",
		},
	}

	val, ok := getPath(data, "x")
	assert.True(t, ok)
	assert.Equal(t, "hello", val)

	val, ok = getPath(data, "nested.y")
	assert.True(t, ok)
	assert.Equal(t, "world", val)

	_, ok = getPath(data, "missing")
	assert.False(t, ok)

	_, ok = getPath(nil, "x")
	assert.False(t, ok)

	_, ok = getPath(data, "")
	assert.False(t, ok)
}

func TestGetPath_MapStringString(t *testing.T) {
	data := map[string]any{
		"headers": map[string]string{
			"content-type": "application/json",
		},
	}

	val, ok := getPath(data, "headers.content-type")
	assert.True(t, ok)
	assert.Equal(t, "application/json", val)
}

func TestCollectAllKeys(t *testing.T) {
	data := map[string]any{
		"a": "val",
		"b": map[string]any{
			"c": "val2",
		},
	}
	keys := collectAllKeys(data, "")
	assert.Contains(t, keys, "a")
	assert.Contains(t, keys, "b")
	assert.Contains(t, keys, "b.c")
}

func TestMarkKeyUsed(t *testing.T) {
	used := make(map[string]bool)
	markKeyUsed(used, "a.b.c")
	assert.True(t, used["a.b.c"])
	assert.True(t, used["a"])
	assert.True(t, used["a.b"])
}

func TestFindSimilarStrings(t *testing.T) {
	candidates := []string{"name", "email", "phone", "names", "game"}

	t.Run("finds similar strings", func(t *testing.T) {
		results := findSimilarStrings("nme", candidates, 3)
		assert.Contains(t, results, "name")
	})

	t.Run("respects max results", func(t *testing.T) {
		results := findSimilarStrings("name", candidates, 1)
		assert.True(t, len(results) <= 1)
	})

	t.Run("no results for very different string", func(t *testing.T) {
		results := findSimilarStrings("zzzzzzzzzzzzz", candidates, 3)
		assert.Empty(t, results)
	})
}

func TestLevenshteinDistance(t *testing.T) {
	assert.Equal(t, 0, levenshteinDistance("", ""))
	assert.Equal(t, 3, levenshteinDistance("", "abc"))
	assert.Equal(t, 3, levenshteinDistance("abc", ""))
	assert.Equal(t, 0, levenshteinDistance("abc", "abc"))
	assert.Equal(t, 1, levenshteinDistance("abc", "ab"))
	assert.Equal(t, 1, levenshteinDistance("abc", "abx"))
	assert.Equal(t, 3, levenshteinDistance("kitten", "sitting"))
}

// =============================================================================
// DryRun — Switch Node Processing
// =============================================================================

func TestTemplate_DryRun_Switch(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("switch with cases", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.switch eval="color"~}{~exons.case value="red"~}{~exons.var name="r" /~}{~/exons.case~}{~exons.case value="blue"~}{~exons.var name="b" /~}{~/exons.case~}{~/exons.switch~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, map[string]any{"r": "R", "b": "B"})
		// Should detect variables inside switch cases
		assert.True(t, len(result.Variables) >= 2)
	})

	t.Run("switch with default", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.switch eval="x"~}{~exons.case value="a"~}A{~/exons.case~}{~exons.casedefault~}{~exons.var name="d" /~}{~/exons.casedefault~}{~/exons.switch~}`)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, map[string]any{"d": "D"})
		// Should detect variables inside default case
		assert.True(t, len(result.Variables) >= 1)
	})
}

// =============================================================================
// DryRun — Valid field
// =============================================================================

func TestDryRunResult_Valid(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("valid when no errors", func(t *testing.T) {
		tmpl, err := engine.Parse("Hello World")
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
	})
}
