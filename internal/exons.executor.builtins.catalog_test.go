package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// SkillsCatalogResolver tests
// ---------------------------------------------------------------------------

func TestSkillsCatalogResolver_TagName(t *testing.T) {
	resolver := NewSkillsCatalogResolver()
	assert.Equal(t, TagNameSkillsCatalog, resolver.TagName())
}

func TestSkillsCatalogResolver_Validate(t *testing.T) {
	resolver := NewSkillsCatalogResolver()

	t.Run("nil attributes", func(t *testing.T) {
		err := resolver.Validate(nil)
		assert.NoError(t, err)
	})

	t.Run("empty attributes", func(t *testing.T) {
		err := resolver.Validate(Attributes{})
		assert.NoError(t, err)
	})

	t.Run("arbitrary attributes", func(t *testing.T) {
		err := resolver.Validate(Attributes{"format": "detailed", "extra": "value"})
		assert.NoError(t, err)
	})
}

func TestSkillsCatalogResolver_Resolve(t *testing.T) {
	resolver := NewSkillsCatalogResolver()

	t.Run("returns catalog string from context", func(t *testing.T) {
		catalog := "## Skills\n- web-search: Search the web\n- summarizer: Summarize content"
		ctx := newMockContextAccessor(map[string]any{
			ContextKeySkills: catalog,
		})
		attrs := Attributes{}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, catalog, result)
	})

	t.Run("returns empty string when skills key not found", func(t *testing.T) {
		ctx := newMockContextAccessor(map[string]any{})
		attrs := Attributes{}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("returns empty string when skills value is not a string", func(t *testing.T) {
		ctx := newMockContextAccessor(map[string]any{
			ContextKeySkills: 42,
		})
		attrs := Attributes{}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("returns empty string when skills value is nil", func(t *testing.T) {
		ctx := newMockContextAccessor(map[string]any{
			ContextKeySkills: nil,
		})
		attrs := Attributes{}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("returns empty string when skills value is a slice", func(t *testing.T) {
		ctx := newMockContextAccessor(map[string]any{
			ContextKeySkills: []string{"skill1", "skill2"},
		})
		attrs := Attributes{}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("returns error when execCtx is not a ContextAccessor", func(t *testing.T) {
		attrs := Attributes{}

		result, err := resolver.Resolve(context.Background(), "not-a-context-accessor", attrs)
		require.Error(t, err)
		assert.Equal(t, "", result)
		assert.Contains(t, err.Error(), ErrMsgInvalidContext)
		assert.Contains(t, err.Error(), TagNameSkillsCatalog)
	})

	t.Run("returns error when execCtx is nil", func(t *testing.T) {
		attrs := Attributes{}

		result, err := resolver.Resolve(context.Background(), nil, attrs)
		require.Error(t, err)
		assert.Equal(t, "", result)
		assert.Contains(t, err.Error(), ErrMsgInvalidContext)
	})

	t.Run("returns empty catalog string", func(t *testing.T) {
		ctx := newMockContextAccessor(map[string]any{
			ContextKeySkills: "",
		})
		attrs := Attributes{}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("does not read tools key", func(t *testing.T) {
		ctx := newMockContextAccessor(map[string]any{
			ContextKeyTools: "tools catalog content",
		})
		attrs := Attributes{}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})
}

// ---------------------------------------------------------------------------
// ToolsCatalogResolver tests
// ---------------------------------------------------------------------------

func TestToolsCatalogResolver_TagName(t *testing.T) {
	resolver := NewToolsCatalogResolver()
	assert.Equal(t, TagNameToolsCatalog, resolver.TagName())
}

func TestToolsCatalogResolver_Validate(t *testing.T) {
	resolver := NewToolsCatalogResolver()

	t.Run("nil attributes", func(t *testing.T) {
		err := resolver.Validate(nil)
		assert.NoError(t, err)
	})

	t.Run("empty attributes", func(t *testing.T) {
		err := resolver.Validate(Attributes{})
		assert.NoError(t, err)
	})

	t.Run("arbitrary attributes", func(t *testing.T) {
		err := resolver.Validate(Attributes{"format": "function_calling", "extra": "value"})
		assert.NoError(t, err)
	})
}

func TestToolsCatalogResolver_Resolve(t *testing.T) {
	resolver := NewToolsCatalogResolver()

	t.Run("returns catalog string from context", func(t *testing.T) {
		catalog := "## Tools\n- search_web(query: string): Search the web\n- calculate(expr: string): Evaluate math"
		ctx := newMockContextAccessor(map[string]any{
			ContextKeyTools: catalog,
		})
		attrs := Attributes{}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, catalog, result)
	})

	t.Run("returns empty string when tools key not found", func(t *testing.T) {
		ctx := newMockContextAccessor(map[string]any{})
		attrs := Attributes{}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("returns empty string when tools value is not a string", func(t *testing.T) {
		ctx := newMockContextAccessor(map[string]any{
			ContextKeyTools: 123,
		})
		attrs := Attributes{}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("returns empty string when tools value is nil", func(t *testing.T) {
		ctx := newMockContextAccessor(map[string]any{
			ContextKeyTools: nil,
		})
		attrs := Attributes{}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("returns empty string when tools value is a map", func(t *testing.T) {
		ctx := newMockContextAccessor(map[string]any{
			ContextKeyTools: map[string]string{"tool1": "desc1"},
		})
		attrs := Attributes{}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("returns error when execCtx is not a ContextAccessor", func(t *testing.T) {
		attrs := Attributes{}

		result, err := resolver.Resolve(context.Background(), "not-a-context-accessor", attrs)
		require.Error(t, err)
		assert.Equal(t, "", result)
		assert.Contains(t, err.Error(), ErrMsgInvalidContext)
		assert.Contains(t, err.Error(), TagNameToolsCatalog)
	})

	t.Run("returns error when execCtx is nil", func(t *testing.T) {
		attrs := Attributes{}

		result, err := resolver.Resolve(context.Background(), nil, attrs)
		require.Error(t, err)
		assert.Equal(t, "", result)
		assert.Contains(t, err.Error(), ErrMsgInvalidContext)
	})

	t.Run("returns empty catalog string", func(t *testing.T) {
		ctx := newMockContextAccessor(map[string]any{
			ContextKeyTools: "",
		})
		attrs := Attributes{}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("does not read skills key", func(t *testing.T) {
		ctx := newMockContextAccessor(map[string]any{
			ContextKeySkills: "skills catalog content",
		})
		attrs := Attributes{}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})
}
