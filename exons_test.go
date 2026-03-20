package exons

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Engine Creation
// =============================================================================

func TestNew(t *testing.T) {
	t.Run("creates engine with defaults", func(t *testing.T) {
		engine, err := New()
		require.NoError(t, err)
		assert.NotNil(t, engine)
	})

	t.Run("creates engine with custom delimiters", func(t *testing.T) {
		engine, err := New(WithDelimiters("<<", ">>"))
		require.NoError(t, err)
		assert.NotNil(t, engine)
	})

	t.Run("creates engine with error strategy", func(t *testing.T) {
		engine, err := New(WithErrorStrategy(ErrorStrategyRemove))
		require.NoError(t, err)
		assert.NotNil(t, engine)
	})

	t.Run("creates engine with max depth", func(t *testing.T) {
		engine, err := New(WithMaxDepth(5))
		require.NoError(t, err)
		assert.Equal(t, 5, engine.MaxDepth())
	})

	t.Run("creates engine with logger", func(t *testing.T) {
		engine, err := New(WithLogger(nil))
		require.NoError(t, err)
		assert.NotNil(t, engine)
	})
}

func TestMustNew(t *testing.T) {
	t.Run("creates engine without panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			engine := MustNew()
			assert.NotNil(t, engine)
		})
	})
}

// =============================================================================
// Parse + Execute with Variables
// =============================================================================

func TestEngine_Execute_Variables(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("simple variable", func(t *testing.T) {
		result, err := engine.Execute(ctx, `Hello {~exons.var name="name" /~}!`, map[string]any{
			"name": "World",
		})
		require.NoError(t, err)
		assert.Equal(t, "Hello World!", result)
	})

	t.Run("variable with default", func(t *testing.T) {
		result, err := engine.Execute(ctx, `Hello {~exons.var name="name" default="Guest" /~}!`, nil)
		require.NoError(t, err)
		assert.Equal(t, "Hello Guest!", result)
	})

	t.Run("variable with default overridden by data", func(t *testing.T) {
		result, err := engine.Execute(ctx, `Hello {~exons.var name="name" default="Guest" /~}!`, map[string]any{
			"name": "Alice",
		})
		require.NoError(t, err)
		assert.Equal(t, "Hello Alice!", result)
	})

	t.Run("dot notation path", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~exons.var name="user.profile.name" /~}`, map[string]any{
			"user": map[string]any{
				"profile": map[string]any{
					"name": "Bob",
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "Bob", result)
	})

	t.Run("numeric variable", func(t *testing.T) {
		result, err := engine.Execute(ctx, `Count: {~exons.var name="count" /~}`, map[string]any{
			"count": 42,
		})
		require.NoError(t, err)
		assert.Equal(t, "Count: 42", result)
	})

	t.Run("boolean variable", func(t *testing.T) {
		result, err := engine.Execute(ctx, `Active: {~exons.var name="active" /~}`, map[string]any{
			"active": true,
		})
		require.NoError(t, err)
		assert.Equal(t, "Active: true", result)
	})

	t.Run("multiple variables", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~exons.var name="first" /~} {~exons.var name="last" /~}`, map[string]any{
			"first": "John",
			"last":  "Doe",
		})
		require.NoError(t, err)
		assert.Equal(t, "John Doe", result)
	})

	t.Run("missing variable with throw strategy", func(t *testing.T) {
		_, err := engine.Execute(ctx, `{~exons.var name="missing" /~}`, nil)
		assert.Error(t, err)
	})
}

// =============================================================================
// Conditionals
// =============================================================================

func TestEngine_Execute_Conditionals(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("simple if true", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~exons.if eval="show"~}visible{~/exons.if~}`, map[string]any{
			"show": true,
		})
		require.NoError(t, err)
		assert.Equal(t, "visible", result)
	})

	t.Run("simple if false", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~exons.if eval="show"~}visible{~/exons.if~}`, map[string]any{
			"show": false,
		})
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("if else", func(t *testing.T) {
		tmpl := `{~exons.if eval="isAdmin"~}admin{~exons.else~}user{~/exons.if~}`

		result, err := engine.Execute(ctx, tmpl, map[string]any{"isAdmin": true})
		require.NoError(t, err)
		assert.Equal(t, "admin", result)

		result, err = engine.Execute(ctx, tmpl, map[string]any{"isAdmin": false})
		require.NoError(t, err)
		assert.Equal(t, "user", result)
	})

	t.Run("if elseif else", func(t *testing.T) {
		tmpl := `{~exons.if eval="role == \"admin\""~}A{~exons.elseif eval="role == \"user\""~}U{~exons.else~}G{~/exons.if~}`

		result, err := engine.Execute(ctx, tmpl, map[string]any{"role": "admin"})
		require.NoError(t, err)
		assert.Equal(t, "A", result)

		result, err = engine.Execute(ctx, tmpl, map[string]any{"role": "user"})
		require.NoError(t, err)
		assert.Equal(t, "U", result)

		result, err = engine.Execute(ctx, tmpl, map[string]any{"role": "guest"})
		require.NoError(t, err)
		assert.Equal(t, "G", result)
	})

	t.Run("if with comparison", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~exons.if eval="count > 5"~}many{~exons.else~}few{~/exons.if~}`, map[string]any{
			"count": 10,
		})
		require.NoError(t, err)
		assert.Equal(t, "many", result)
	})

	t.Run("nested conditional", func(t *testing.T) {
		tmpl := `{~exons.if eval="a"~}{~exons.if eval="b"~}both{~/exons.if~}{~/exons.if~}`
		result, err := engine.Execute(ctx, tmpl, map[string]any{"a": true, "b": true})
		require.NoError(t, err)
		assert.Equal(t, "both", result)
	})

	t.Run("conditional with expression functions", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~exons.if eval="len(items) > 0"~}has items{~exons.else~}empty{~/exons.if~}`, map[string]any{
			"items": []any{"a", "b"},
		})
		require.NoError(t, err)
		assert.Equal(t, "has items", result)
	})
}

// =============================================================================
// Loops
// =============================================================================

func TestEngine_Execute_Loops(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("simple for loop", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~exons.for item="x" in="items"~}{~exons.var name="x" /~},{~/exons.for~}`, map[string]any{
			"items": []any{"a", "b", "c"},
		})
		require.NoError(t, err)
		assert.Equal(t, "a,b,c,", result)
	})

	t.Run("for loop with index", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~exons.for item="x" index="i" in="items"~}{~exons.var name="i" /~}:{~exons.var name="x" /~} {~/exons.for~}`, map[string]any{
			"items": []any{"a", "b"},
		})
		require.NoError(t, err)
		assert.Equal(t, "0:a 1:b ", result)
	})

	t.Run("for loop with map items", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~exons.for item="u" in="users"~}{~exons.var name="u.name" /~} {~/exons.for~}`, map[string]any{
			"users": []any{
				map[string]any{"name": "Alice"},
				map[string]any{"name": "Bob"},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "Alice Bob ", result)
	})

	t.Run("for loop with limit", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~exons.for item="x" in="items" limit="2"~}{~exons.var name="x" /~}{~/exons.for~}`, map[string]any{
			"items": []any{"a", "b", "c", "d"},
		})
		require.NoError(t, err)
		assert.Equal(t, "ab", result)
	})

	t.Run("for loop with empty collection", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~exons.for item="x" in="items"~}item{~/exons.for~}`, map[string]any{
			"items": []any{},
		})
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})
}

// =============================================================================
// Raw Blocks
// =============================================================================

func TestEngine_Execute_Raw(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("raw block preserves content", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~exons.raw~}{~exons.var name="x" /~}{~/exons.raw~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, `{~exons.var name="x" /~}`, result)
	})

	t.Run("raw block with XML content", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~exons.raw~}<tag attr="val">{~/exons.raw~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, `<tag attr="val">`, result)
	})

	t.Run("raw block with JSON content", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~exons.raw~}{"key": "value"}{~/exons.raw~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, `{"key": "value"}`, result)
	})
}

// =============================================================================
// Comments
// =============================================================================

func TestEngine_Execute_Comments(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("comment is removed", func(t *testing.T) {
		result, err := engine.Execute(ctx, `before{~exons.comment~}hidden{~/exons.comment~}after`, nil)
		require.NoError(t, err)
		assert.Equal(t, "beforeafter", result)
	})

	t.Run("multiple comments", func(t *testing.T) {
		result, err := engine.Execute(ctx, `a{~exons.comment~}1{~/exons.comment~}b{~exons.comment~}2{~/exons.comment~}c`, nil)
		require.NoError(t, err)
		assert.Equal(t, "abc", result)
	})
}

// =============================================================================
// Message Tags + Extraction
// =============================================================================

func TestEngine_Execute_Messages(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("single message", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="user"~}Hello{~/exons.message~}`)
		require.NoError(t, err)
		messages, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
		require.NoError(t, err)
		require.Len(t, messages, 1)
		assert.Equal(t, RoleUser, messages[0].Role)
		assert.Equal(t, "Hello", messages[0].Content)
	})

	t.Run("system and user messages", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="system"~}You are a helper.{~/exons.message~}
{~exons.message role="user"~}Hi{~/exons.message~}`)
		require.NoError(t, err)
		messages, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
		require.NoError(t, err)
		require.Len(t, messages, 2)
		assert.Equal(t, RoleSystem, messages[0].Role)
		assert.Equal(t, RoleUser, messages[1].Role)
	})

	t.Run("message with variable interpolation", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="user"~}{~exons.var name="query" /~}{~/exons.message~}`)
		require.NoError(t, err)
		messages, err := tmpl.ExecuteAndExtractMessages(ctx, map[string]any{"query": "What is Go?"})
		require.NoError(t, err)
		require.Len(t, messages, 1)
		assert.Equal(t, "What is Go?", messages[0].Content)
	})

	t.Run("message with cache attribute", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="system" cache="true"~}Cached content{~/exons.message~}`)
		require.NoError(t, err)
		messages, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
		require.NoError(t, err)
		require.Len(t, messages, 1)
		assert.True(t, messages[0].Cache)
	})
}

// =============================================================================
// Include / Nested Templates
// =============================================================================

func TestEngine_Execute_Include(t *testing.T) {
	ctx := context.Background()

	t.Run("simple include", func(t *testing.T) {
		engine := MustNew()
		engine.MustRegisterTemplate("greeting", `Hello {~exons.var name="name" default="World" /~}`)
		result, err := engine.Execute(ctx, `{~exons.include template="greeting" /~}!`, nil)
		require.NoError(t, err)
		assert.Equal(t, "Hello World!", result)
	})

	t.Run("include with passed attributes", func(t *testing.T) {
		engine := MustNew()
		engine.MustRegisterTemplate("greeting", `Hello {~exons.var name="name" /~}`)
		result, err := engine.Execute(ctx, `{~exons.include template="greeting" name="Alice" /~}!`, nil)
		require.NoError(t, err)
		assert.Equal(t, "Hello Alice!", result)
	})

	t.Run("include with isolation", func(t *testing.T) {
		engine := MustNew()
		engine.MustRegisterTemplate("child", `{~exons.var name="x" default="none" /~}`)
		result, err := engine.Execute(ctx, `{~exons.include template="child" isolate="true" /~}`, map[string]any{
			"x": "parent_value",
		})
		require.NoError(t, err)
		assert.Equal(t, "none", result)
	})

	t.Run("include missing template", func(t *testing.T) {
		engine := MustNew()
		_, err := engine.Execute(ctx, `{~exons.include template="nonexistent" /~}`, nil)
		assert.Error(t, err)
	})

	t.Run("nested includes", func(t *testing.T) {
		engine := MustNew()
		engine.MustRegisterTemplate("inner", "INNER")
		engine.MustRegisterTemplate("outer", `OUTER:{~exons.include template="inner" /~}`)
		result, err := engine.Execute(ctx, `{~exons.include template="outer" /~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, "OUTER:INNER", result)
	})
}

// =============================================================================
// Template Management
// =============================================================================

func TestEngine_TemplateManagement(t *testing.T) {
	t.Run("register and get template", func(t *testing.T) {
		engine := MustNew()
		err := engine.RegisterTemplate("test", "Hello")
		require.NoError(t, err)

		tmpl, ok := engine.GetTemplate("test")
		assert.True(t, ok)
		assert.NotNil(t, tmpl)
	})

	t.Run("has template", func(t *testing.T) {
		engine := MustNew()
		assert.False(t, engine.HasTemplate("test"))
		engine.MustRegisterTemplate("test", "Hello")
		assert.True(t, engine.HasTemplate("test"))
	})

	t.Run("list templates", func(t *testing.T) {
		engine := MustNew()
		engine.MustRegisterTemplate("b", "B")
		engine.MustRegisterTemplate("a", "A")
		names := engine.ListTemplates()
		assert.Equal(t, []string{"a", "b"}, names)
	})

	t.Run("template count", func(t *testing.T) {
		engine := MustNew()
		assert.Equal(t, 0, engine.TemplateCount())
		engine.MustRegisterTemplate("test", "Hello")
		assert.Equal(t, 1, engine.TemplateCount())
	})

	t.Run("unregister template", func(t *testing.T) {
		engine := MustNew()
		engine.MustRegisterTemplate("test", "Hello")
		assert.True(t, engine.UnregisterTemplate("test"))
		assert.False(t, engine.HasTemplate("test"))
		assert.False(t, engine.UnregisterTemplate("test")) // already gone
	})

	t.Run("duplicate template registration fails", func(t *testing.T) {
		engine := MustNew()
		engine.MustRegisterTemplate("test", "Hello")
		err := engine.RegisterTemplate("test", "World")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgTemplateAlreadyExists)
	})

	t.Run("empty template name fails", func(t *testing.T) {
		engine := MustNew()
		err := engine.RegisterTemplate("", "Hello")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgEmptyTemplateName)
	})

	t.Run("reserved namespace fails", func(t *testing.T) {
		engine := MustNew()
		err := engine.RegisterTemplate("exons.test", "Hello")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgReservedTemplateName)
	})

	t.Run("get template source", func(t *testing.T) {
		engine := MustNew()
		engine.MustRegisterTemplate("test", "Hello World")
		src, ok := engine.GetTemplateSource("test")
		assert.True(t, ok)
		assert.Equal(t, "Hello World", src)
	})

	t.Run("get template source not found", func(t *testing.T) {
		engine := MustNew()
		src, ok := engine.GetTemplateSource("missing")
		assert.False(t, ok)
		assert.Equal(t, "", src)
	})
}

// =============================================================================
// Custom Resolvers
// =============================================================================

func TestEngine_CustomResolvers(t *testing.T) {
	ctx := context.Background()

	t.Run("register and use custom resolver", func(t *testing.T) {
		engine := MustNew()
		resolver := NewResolverFunc(
			"MyTag",
			func(ctx context.Context, execCtx *Context, attrs Attributes) (string, error) {
				name, _ := attrs.Get("name")
				return "CUSTOM:" + name, nil
			},
			nil,
		)
		err := engine.Register(resolver)
		require.NoError(t, err)

		result, err := engine.Execute(ctx, `{~MyTag name="test" /~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, "CUSTOM:test", result)
	})

	t.Run("resolver with validation", func(t *testing.T) {
		engine := MustNew()
		resolver := NewResolverFunc(
			"Required",
			func(ctx context.Context, execCtx *Context, attrs Attributes) (string, error) {
				name, _ := attrs.Get("name")
				return name, nil
			},
			func(attrs Attributes) error {
				if !attrs.Has("name") {
					return fmt.Errorf("name attribute required")
				}
				return nil
			},
		)
		err := engine.Register(resolver)
		require.NoError(t, err)
		assert.True(t, engine.HasResolver("Required"))
	})

	t.Run("has resolver", func(t *testing.T) {
		engine := MustNew()
		assert.True(t, engine.HasResolver(TagNameVar))
		assert.False(t, engine.HasResolver("NonExistent"))
	})

	t.Run("list resolvers", func(t *testing.T) {
		engine := MustNew()
		resolvers := engine.ListResolvers()
		assert.True(t, len(resolvers) > 0)
		// Should include built-in resolvers
		assert.Contains(t, resolvers, TagNameVar)
	})

	t.Run("resolver count", func(t *testing.T) {
		engine := MustNew()
		count := engine.ResolverCount()
		assert.True(t, count > 0)
	})

	t.Run("duplicate resolver fails", func(t *testing.T) {
		engine := MustNew()
		resolver := NewResolverFunc("DupTag", func(ctx context.Context, execCtx *Context, attrs Attributes) (string, error) {
			return "", nil
		}, nil)
		err := engine.Register(resolver)
		require.NoError(t, err)
		err = engine.Register(resolver)
		assert.Error(t, err)
	})

	t.Run("must register panics on duplicate", func(t *testing.T) {
		engine := MustNew()
		resolver := NewResolverFunc("PanicTag", func(ctx context.Context, execCtx *Context, attrs Attributes) (string, error) {
			return "", nil
		}, nil)
		engine.MustRegister(resolver)
		assert.Panics(t, func() {
			engine.MustRegister(resolver)
		})
	})
}

// =============================================================================
// Custom Functions
// =============================================================================

func TestEngine_CustomFunctions(t *testing.T) {
	ctx := context.Background()

	t.Run("register and use custom function", func(t *testing.T) {
		engine := MustNew()
		err := engine.RegisterFunc(&Func{
			Name:    "double",
			MinArgs: 1,
			MaxArgs: 1,
			Fn: func(args []any) (any, error) {
				if n, ok := args[0].(int); ok {
					return n * 2, nil
				}
				return nil, fmt.Errorf("expected int")
			},
		})
		require.NoError(t, err)

		result, err := engine.Execute(ctx, `{~exons.if eval="double(count) > 5"~}big{~exons.else~}small{~/exons.if~}`, map[string]any{
			"count": 10,
		})
		require.NoError(t, err)
		assert.Equal(t, "big", result)
	})

	t.Run("nil function fails", func(t *testing.T) {
		engine := MustNew()
		err := engine.RegisterFunc(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgFuncNilFunc)
	})

	t.Run("empty function name fails", func(t *testing.T) {
		engine := MustNew()
		err := engine.RegisterFunc(&Func{
			Name: "",
			Fn:   func(args []any) (any, error) { return nil, nil },
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgFuncEmptyName)
	})

	t.Run("has func", func(t *testing.T) {
		engine := MustNew()
		assert.True(t, engine.HasFunc("len"))    // built-in
		assert.False(t, engine.HasFunc("custom")) // not registered
	})

	t.Run("list funcs", func(t *testing.T) {
		engine := MustNew()
		funcs := engine.ListFuncs()
		assert.True(t, len(funcs) > 0)
	})

	t.Run("func count", func(t *testing.T) {
		engine := MustNew()
		count := engine.FuncCount()
		assert.True(t, count > 0)
	})

	t.Run("must register func panics on nil", func(t *testing.T) {
		engine := MustNew()
		assert.Panics(t, func() {
			engine.MustRegisterFunc(nil)
		})
	})
}

// =============================================================================
// Error Strategies
// =============================================================================

func TestEngine_ErrorStrategies(t *testing.T) {
	ctx := context.Background()

	t.Run("throw strategy returns error", func(t *testing.T) {
		engine := MustNew(WithErrorStrategy(ErrorStrategyThrow))
		_, err := engine.Execute(ctx, `{~exons.var name="missing" /~}`, nil)
		assert.Error(t, err)
	})

	t.Run("default strategy uses default value", func(t *testing.T) {
		engine := MustNew(WithErrorStrategy(ErrorStrategyDefault))
		result, err := engine.Execute(ctx, `{~exons.var name="missing" default="fallback" /~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, "fallback", result)
	})

	t.Run("remove strategy removes tag", func(t *testing.T) {
		engine := MustNew(WithErrorStrategy(ErrorStrategyRemove))
		result, err := engine.Execute(ctx, `before{~exons.var name="missing" /~}after`, nil)
		require.NoError(t, err)
		assert.Equal(t, "beforeafter", result)
	})

	t.Run("per-tag onerror attribute", func(t *testing.T) {
		engine := MustNew(WithErrorStrategy(ErrorStrategyThrow))
		result, err := engine.Execute(ctx, `{~exons.var name="missing" onerror="remove" /~}OK`, nil)
		require.NoError(t, err)
		assert.Equal(t, "OK", result)
	})

	t.Run("keepraw strategy preserves tag", func(t *testing.T) {
		engine := MustNew(WithErrorStrategy(ErrorStrategyKeepRaw))
		result, err := engine.Execute(ctx, `{~exons.var name="missing" /~}`, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "exons.var")
	})

	t.Run("log strategy continues", func(t *testing.T) {
		engine := MustNew(WithErrorStrategy(ErrorStrategyLog))
		result, err := engine.Execute(ctx, `before{~exons.var name="missing" /~}after`, nil)
		require.NoError(t, err)
		assert.Equal(t, "beforeafter", result)
	})
}

// =============================================================================
// YAML Frontmatter / Spec Parsing
// =============================================================================

func TestEngine_Execute_YAMLFrontmatter(t *testing.T) {
	ctx := context.Background()

	t.Run("template with frontmatter", func(t *testing.T) {
		engine := MustNew()
		source := `---
name: test-prompt
description: A test prompt
type: prompt
---
Hello {~exons.var name="name" default="World" /~}!`
		tmpl, err := engine.Parse(source)
		require.NoError(t, err)
		assert.True(t, tmpl.HasSpec())

		spec := tmpl.Spec()
		assert.Equal(t, "test-prompt", spec.Name)
		assert.Equal(t, "A test prompt", spec.Description)
		assert.Equal(t, DocumentTypePrompt, spec.Type)

		result, err := tmpl.Execute(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, "Hello World!", result)
	})

	t.Run("template without frontmatter", func(t *testing.T) {
		engine := MustNew()
		tmpl, err := engine.Parse(`Hello World`)
		require.NoError(t, err)
		assert.False(t, tmpl.HasSpec())
		assert.Nil(t, tmpl.Spec())
	})

	t.Run("frontmatter with execution config", func(t *testing.T) {
		engine := MustNew()
		source := `---
name: test-agent
description: Test agent
type: agent
execution:
  provider: openai
  model: gpt-4
---
Body`
		tmpl, err := engine.Parse(source)
		require.NoError(t, err)
		spec := tmpl.Spec()
		require.NotNil(t, spec.Execution)
		assert.Equal(t, "openai", spec.Execution.Provider)
		assert.Equal(t, "gpt-4", spec.Execution.Model)
	})

	t.Run("frontmatter with inputs", func(t *testing.T) {
		engine := MustNew()
		source := `---
name: test-skill
description: Test skill
type: skill
inputs:
  query:
    type: string
    required: true
---
{~exons.var name="query" /~}`
		tmpl, err := engine.Parse(source)
		require.NoError(t, err)
		spec := tmpl.Spec()
		require.NotNil(t, spec.Inputs)
		assert.Contains(t, spec.Inputs, "query")
		assert.True(t, spec.Inputs["query"].Required)
	})
}

// =============================================================================
// Environment Variables
// =============================================================================

func TestEngine_Execute_Env(t *testing.T) {
	ctx := context.Background()

	t.Run("env variable found", func(t *testing.T) {
		os.Setenv("TEST_EXONS_VAR", "hello_env")
		defer os.Unsetenv("TEST_EXONS_VAR")

		engine := MustNew()
		result, err := engine.Execute(ctx, `{~exons.env name="TEST_EXONS_VAR" /~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, "hello_env", result)
	})

	t.Run("env variable with default", func(t *testing.T) {
		engine := MustNew()
		result, err := engine.Execute(ctx, `{~exons.env name="NONEXISTENT_EXONS_VAR_12345" default="fallback" /~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, "fallback", result)
	})

	t.Run("env required variable missing", func(t *testing.T) {
		engine := MustNew()
		_, err := engine.Execute(ctx, `{~exons.env name="NONEXISTENT_EXONS_VAR_12345" required="true" /~}`, nil)
		assert.Error(t, err)
	})
}

// =============================================================================
// Escape Sequences
// =============================================================================

func TestEngine_Execute_EscapeSequence(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("escaped delimiters", func(t *testing.T) {
		result, err := engine.Execute(ctx, `\{~exons.var name="x" /~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, `{~exons.var name="x" /~}`, result)
	})
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestEngine_Execute_EdgeCases(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("empty template", func(t *testing.T) {
		result, err := engine.Execute(ctx, "", nil)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("nil data", func(t *testing.T) {
		result, err := engine.Execute(ctx, "Hello", nil)
		require.NoError(t, err)
		assert.Equal(t, "Hello", result)
	})

	t.Run("plain text no tags", func(t *testing.T) {
		result, err := engine.Execute(ctx, "Just plain text", nil)
		require.NoError(t, err)
		assert.Equal(t, "Just plain text", result)
	})

	t.Run("unicode content", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~exons.var name="msg" /~}`, map[string]any{
			"msg": "Hello, World! Hola mundo!",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "Hello")
	})

	t.Run("template source and body", func(t *testing.T) {
		tmpl, err := engine.Parse("Hello World")
		require.NoError(t, err)
		assert.Equal(t, "Hello World", tmpl.Source())
		assert.Equal(t, "Hello World", tmpl.TemplateBody())
	})
}

// =============================================================================
// Switch/Case
// =============================================================================

func TestEngine_Execute_Switch(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("switch with matching case", func(t *testing.T) {
		tmpl := `{~exons.switch eval="color"~}{~exons.case value="red"~}RED{~/exons.case~}{~exons.case value="blue"~}BLUE{~/exons.case~}{~/exons.switch~}`
		result, err := engine.Execute(ctx, tmpl, map[string]any{"color": "red"})
		require.NoError(t, err)
		assert.Equal(t, "RED", result)
	})

	t.Run("switch with default case", func(t *testing.T) {
		tmpl := `{~exons.switch eval="color"~}{~exons.case value="red"~}RED{~/exons.case~}{~exons.casedefault~}OTHER{~/exons.casedefault~}{~/exons.switch~}`
		result, err := engine.Execute(ctx, tmpl, map[string]any{"color": "green"})
		require.NoError(t, err)
		assert.Equal(t, "OTHER", result)
	})

	t.Run("switch with no match and no default", func(t *testing.T) {
		tmpl := `{~exons.switch eval="color"~}{~exons.case value="red"~}RED{~/exons.case~}{~/exons.switch~}`
		result, err := engine.Execute(ctx, tmpl, map[string]any{"color": "green"})
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})
}

// =============================================================================
// Template Inheritance
// =============================================================================

func TestEngine_Execute_Inheritance(t *testing.T) {
	ctx := context.Background()

	t.Run("extends with block override", func(t *testing.T) {
		engine := MustNew()
		engine.MustRegisterTemplate("base", `Header {~exons.block name="content"~}default{~/exons.block~} Footer`)

		result, err := engine.Execute(ctx, `{~exons.extends template="base" /~}{~exons.block name="content"~}custom{~/exons.block~}`, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "Header")
		assert.Contains(t, result, "custom")
		assert.Contains(t, result, "Footer")
		assert.NotContains(t, result, "default")
	})

	t.Run("extends with default block content", func(t *testing.T) {
		engine := MustNew()
		engine.MustRegisterTemplate("base", `Header {~exons.block name="content"~}default{~/exons.block~} Footer`)

		result, err := engine.Execute(ctx, `{~exons.extends template="base" /~}`, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "Header")
		assert.Contains(t, result, "default")
		assert.Contains(t, result, "Footer")
	})
}

// =============================================================================
// Ref Tags
// =============================================================================

func TestEngine_Execute_RefTag(t *testing.T) {
	ctx := context.Background()

	t.Run("ref tag without resolver returns error", func(t *testing.T) {
		engine := MustNew()
		// Without a spec resolver configured, ref should fail
		_, err := engine.Execute(ctx, `{~exons.ref slug="my-prompt" /~}`, nil)
		assert.Error(t, err)
	})
}

// =============================================================================
// Parse
// =============================================================================

func TestEngine_Parse(t *testing.T) {
	t.Run("parse valid template", func(t *testing.T) {
		engine := MustNew()
		tmpl, err := engine.Parse(`Hello {~exons.var name="x" /~}`)
		require.NoError(t, err)
		assert.NotNil(t, tmpl)
	})

	t.Run("parse invalid syntax", func(t *testing.T) {
		engine := MustNew()
		_, err := engine.Parse(`{~exons.if~}no closing tag`)
		assert.Error(t, err)
	})
}

// =============================================================================
// Template Compile stubs
// =============================================================================

func TestTemplate_Compile(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse("Hello")
	require.NoError(t, err)

	t.Run("Compile returns not available", func(t *testing.T) {
		_, err := tmpl.Compile(context.Background(), nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgCompileNotAvailable)
	})

	t.Run("CompileAgent returns not available", func(t *testing.T) {
		_, err := tmpl.CompileAgent(context.Background(), nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgCompileNotAvailable)
	})
}

// =============================================================================
// ErrorStrategy
// =============================================================================

func TestErrorStrategy_String(t *testing.T) {
	tests := []struct {
		strategy ErrorStrategy
		expected string
	}{
		{ErrorStrategyThrow, ErrorStrategyNameThrow},
		{ErrorStrategyDefault, ErrorStrategyNameDefault},
		{ErrorStrategyRemove, ErrorStrategyNameRemove},
		{ErrorStrategyKeepRaw, ErrorStrategyNameKeepRaw},
		{ErrorStrategyLog, ErrorStrategyNameLog},
		{ErrorStrategy(99), ErrorStrategyNameThrow}, // unknown defaults to throw
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.strategy.String())
		})
	}
}

func TestParseErrorStrategy(t *testing.T) {
	tests := []struct {
		input    string
		expected ErrorStrategy
	}{
		{ErrorStrategyNameThrow, ErrorStrategyThrow},
		{ErrorStrategyNameDefault, ErrorStrategyDefault},
		{ErrorStrategyNameRemove, ErrorStrategyRemove},
		{ErrorStrategyNameKeepRaw, ErrorStrategyKeepRaw},
		{ErrorStrategyNameLog, ErrorStrategyLog},
		{"unknown", ErrorStrategyThrow},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, ParseErrorStrategy(tt.input))
		})
	}
}

func TestIsValidErrorStrategy(t *testing.T) {
	assert.True(t, IsValidErrorStrategy(ErrorStrategyNameThrow))
	assert.True(t, IsValidErrorStrategy(ErrorStrategyNameDefault))
	assert.True(t, IsValidErrorStrategy(ErrorStrategyNameRemove))
	assert.True(t, IsValidErrorStrategy(ErrorStrategyNameKeepRaw))
	assert.True(t, IsValidErrorStrategy(ErrorStrategyNameLog))
	assert.False(t, IsValidErrorStrategy("invalid"))
}

// =============================================================================
// ValidationSeverity
// =============================================================================

func TestValidationSeverity_String(t *testing.T) {
	assert.Equal(t, SeverityNameError, SeverityError.String())
	assert.Equal(t, SeverityNameWarning, SeverityWarning.String())
	assert.Equal(t, SeverityNameInfo, SeverityInfo.String())
	assert.Equal(t, SeverityNameError, ValidationSeverity(99).String())
}

// =============================================================================
// Position
// =============================================================================

func TestPosition_String(t *testing.T) {
	pos := Position{Line: 5, Column: 10}
	assert.Equal(t, "line 5, column 10", pos.String())
}

// =============================================================================
// TemplateRunner Interface
// =============================================================================

func TestEngine_ImplementsTemplateRunner(t *testing.T) {
	var _ TemplateRunner = (*Engine)(nil)
}

// =============================================================================
// Concurrent Safety
// =============================================================================

func TestEngine_ConcurrentExecution(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()
	engine.MustRegisterTemplate("greet", `Hello {~exons.var name="name" /~}`)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			name := fmt.Sprintf("User%d", n)
			result, err := engine.Execute(ctx, `{~exons.var name="name" /~}`, map[string]any{"name": name})
			assert.NoError(t, err)
			assert.Equal(t, name, result)
		}(i)
	}
	wg.Wait()
}

func TestEngine_ConcurrentTemplateRegistration(t *testing.T) {
	engine := MustNew()
	var wg sync.WaitGroup

	// Concurrent reads and writes
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			name := fmt.Sprintf("tmpl_%d", n)
			_ = engine.RegisterTemplate(name, "Hello")
		}(i)
		go func(n int) {
			defer wg.Done()
			_ = engine.ListTemplates()
			_ = engine.TemplateCount()
		}(i)
	}
	wg.Wait()
}

// =============================================================================
// Built-in Expression Functions
// =============================================================================

func TestEngine_BuiltInFunctions(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("len function", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~exons.if eval="len(items) > 0"~}yes{~exons.else~}no{~/exons.if~}`, map[string]any{
			"items": []any{"a"},
		})
		require.NoError(t, err)
		assert.Equal(t, "yes", result)
	})

	t.Run("contains function", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~exons.if eval="contains(name, \"ello\")"~}found{~exons.else~}nope{~/exons.if~}`, map[string]any{
			"name": "Hello",
		})
		require.NoError(t, err)
		assert.Equal(t, "found", result)
	})

	t.Run("upper function", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~exons.if eval="upper(name) == \"HELLO\""~}yes{~exons.else~}no{~/exons.if~}`, map[string]any{
			"name": "hello",
		})
		require.NoError(t, err)
		assert.Equal(t, "yes", result)
	})
}

// =============================================================================
// Complex Integration Tests
// =============================================================================

func TestEngine_ComplexTemplate(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("mixed content", func(t *testing.T) {
		tmpl := `Welcome, {~exons.var name="user.name" /~}!
{~exons.if eval="user.isAdmin"~}
You have admin access.
{~exons.else~}
Standard user access.
{~/exons.if~}
Your items:
{~exons.for item="item" index="i" in="items"~}
  {~exons.var name="i" /~}. {~exons.var name="item" /~}
{~/exons.for~}`

		data := map[string]any{
			"user": map[string]any{
				"name":    "Alice",
				"isAdmin": true,
			},
			"items": []any{"Task1", "Task2"},
		}

		result, err := engine.Execute(ctx, tmpl, data)
		require.NoError(t, err)
		assert.Contains(t, result, "Alice")
		assert.Contains(t, result, "admin access")
		assert.Contains(t, result, "Task1")
		assert.Contains(t, result, "Task2")
	})
}

// =============================================================================
// ResolverFunc Tests
// =============================================================================

func TestResolverFunc(t *testing.T) {
	t.Run("tag name", func(t *testing.T) {
		r := NewResolverFunc("TestTag", nil, nil)
		assert.Equal(t, "TestTag", r.TagName())
	})

	t.Run("resolve", func(t *testing.T) {
		r := NewResolverFunc("TestTag",
			func(ctx context.Context, execCtx *Context, attrs Attributes) (string, error) {
				return "result", nil
			},
			nil,
		)
		result, err := r.Resolve(context.Background(), NewContext(nil), nil)
		require.NoError(t, err)
		assert.Equal(t, "result", result)
	})

	t.Run("validate nil returns nil", func(t *testing.T) {
		r := NewResolverFunc("TestTag", nil, nil)
		assert.NoError(t, r.Validate(nil))
	})

	t.Run("validate with function", func(t *testing.T) {
		r := NewResolverFunc("TestTag", nil, func(attrs Attributes) error {
			return fmt.Errorf("validation error")
		})
		assert.Error(t, r.Validate(nil))
	})
}

// =============================================================================
// SpecResolverAdapter Tests
// =============================================================================

type mockSpecResolver struct {
	spec *Spec
	body string
	err  error
}

func (m *mockSpecResolver) ResolveSpec(ctx context.Context, slug string, version string) (*Spec, string, error) {
	return m.spec, m.body, m.err
}

func TestSpecResolverAdapter(t *testing.T) {
	t.Run("resolves body from spec resolver", func(t *testing.T) {
		mock := &mockSpecResolver{
			spec: &Spec{Name: "test"},
			body: "template body",
		}
		adapter := NewSpecResolverAdapter(mock)
		body, err := adapter.ResolveSpecBody(context.Background(), "test", "latest")
		require.NoError(t, err)
		assert.Equal(t, "template body", body)
	})

	t.Run("returns error from spec resolver", func(t *testing.T) {
		mock := &mockSpecResolver{
			err: fmt.Errorf("not found"),
		}
		adapter := NewSpecResolverAdapter(mock)
		_, err := adapter.ResolveSpecBody(context.Background(), "missing", "latest")
		assert.Error(t, err)
	})
}

// =============================================================================
// Error Types
// =============================================================================

func TestErrorTypes(t *testing.T) {
	t.Run("configBlockError", func(t *testing.T) {
		err := NewConfigBlockError("test", Position{Line: 1, Column: 2}, nil)
		assert.Contains(t, err.Error(), "test")
	})

	t.Run("configBlockError with cause", func(t *testing.T) {
		cause := fmt.Errorf("root cause")
		err := NewConfigBlockError("test", Position{}, cause)
		assert.Contains(t, err.Error(), "test")
	})

	t.Run("parseError", func(t *testing.T) {
		err := NewParseError("parse failed", Position{}, nil)
		assert.Contains(t, err.Error(), "parse failed")

		err2 := NewParseError("parse failed", Position{}, fmt.Errorf("cause"))
		assert.Contains(t, err2.Error(), "parse failed")
	})

	t.Run("frontmatterError", func(t *testing.T) {
		err := NewFrontmatterError("fm error", Position{}, nil)
		assert.Contains(t, err.Error(), "fm error")
	})

	t.Run("executionError", func(t *testing.T) {
		err := NewExecutionError("exec failed", "myTag", Position{}, nil)
		assert.Contains(t, err.Error(), "exec failed")
	})

	t.Run("templateNotFoundError", func(t *testing.T) {
		err := NewTemplateNotFoundError("missing")
		assert.Contains(t, err.Error(), ErrMsgTemplateNotFound)
	})

	t.Run("templateExistsError", func(t *testing.T) {
		err := NewTemplateExistsError("dup")
		assert.Contains(t, err.Error(), ErrMsgTemplateAlreadyExists)
	})

	t.Run("emptyTemplateNameError", func(t *testing.T) {
		err := NewEmptyTemplateNameError()
		assert.Contains(t, err.Error(), ErrMsgEmptyTemplateName)
	})

	t.Run("reservedTemplateNameError", func(t *testing.T) {
		err := NewReservedTemplateNameError("exons.test")
		assert.Contains(t, err.Error(), ErrMsgReservedTemplateName)
	})

	t.Run("compileNotAvailableError", func(t *testing.T) {
		err := NewCompileNotAvailableError()
		assert.Contains(t, err.Error(), ErrMsgCompileNotAvailable)
	})

	t.Run("funcRegistrationError", func(t *testing.T) {
		err := NewFuncRegistrationError("test error", "myFunc")
		assert.Contains(t, err.Error(), "test error")

		err2 := NewFuncRegistrationError("test error", "")
		assert.Contains(t, err2.Error(), "test error")
	})
}

// =============================================================================
// ExtractMessagesFromOutput
// =============================================================================

func TestExtractMessagesFromOutput(t *testing.T) {
	t.Run("nil for no messages", func(t *testing.T) {
		messages := ExtractMessagesFromOutput("just plain text")
		assert.Nil(t, messages)
	})
}

// =============================================================================
// Max Depth
// =============================================================================

func TestEngine_MaxDepth(t *testing.T) {
	engine := MustNew(WithMaxDepth(3))
	assert.Equal(t, 3, engine.MaxDepth())
}

// =============================================================================
// ExecuteTemplate
// =============================================================================

func TestEngine_ExecuteTemplate(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("execute registered template", func(t *testing.T) {
		engine.MustRegisterTemplate("hello", "Hello World")
		result, err := engine.ExecuteTemplate(ctx, "hello", nil)
		require.NoError(t, err)
		assert.Equal(t, "Hello World", result)
	})

	t.Run("execute non-existent template", func(t *testing.T) {
		_, err := engine.ExecuteTemplate(ctx, "nonexistent", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgTemplateNotFound)
	})

	t.Run("execute template with data", func(t *testing.T) {
		engine2 := MustNew()
		engine2.MustRegisterTemplate("greet", `Hi {~exons.var name="who" /~}`)
		result, err := engine2.ExecuteTemplate(ctx, "greet", map[string]any{"who": "Bob"})
		require.NoError(t, err)
		assert.Equal(t, "Hi Bob", result)
	})
}

// =============================================================================
// DocumentType Constants
// =============================================================================

func TestDocumentType(t *testing.T) {
	assert.Equal(t, DocumentType("prompt"), DocumentTypePrompt)
	assert.Equal(t, DocumentType("skill"), DocumentTypeSkill)
	assert.Equal(t, DocumentType("agent"), DocumentTypeAgent)
}

// =============================================================================
// Complex Nested Templates with Variables
// =============================================================================

func TestEngine_NestedLoopWithConditional(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	tmpl := `{~exons.for item="user" in="users"~}{~exons.if eval="user.active"~}{~exons.var name="user.name" /~} {~/exons.if~}{~/exons.for~}`

	data := map[string]any{
		"users": []any{
			map[string]any{"name": "Alice", "active": true},
			map[string]any{"name": "Bob", "active": false},
			map[string]any{"name": "Carol", "active": true},
		},
	}

	result, err := engine.Execute(ctx, tmpl, data)
	require.NoError(t, err)
	assert.Contains(t, result, "Alice")
	assert.NotContains(t, result, "Bob")
	assert.Contains(t, result, "Carol")
}

// =============================================================================
// Large Data Sets
// =============================================================================

func TestEngine_LargeLoop(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	items := make([]any, 100)
	for i := range items {
		items[i] = fmt.Sprintf("item_%d", i)
	}

	result, err := engine.Execute(ctx, `{~exons.for item="x" in="items"~}{~exons.var name="x" /~},{~/exons.for~}`, map[string]any{
		"items": items,
	})
	require.NoError(t, err)
	assert.Contains(t, result, "item_0")
	assert.Contains(t, result, "item_99")
	assert.Equal(t, 100, strings.Count(result, ","))
}
