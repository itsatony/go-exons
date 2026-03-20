package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldShowHint_NoDefaultNoOnerror(t *testing.T) {
	attrs := Attributes{"name": "foo"}
	assert.True(t, ShouldShowHint(attrs))
}

func TestShouldShowHint_HasDefault(t *testing.T) {
	attrs := Attributes{"name": "foo", "default": "bar"}
	assert.False(t, ShouldShowHint(attrs))
}

func TestShouldShowHint_HasOnerror(t *testing.T) {
	attrs := Attributes{"name": "foo", "onerror": "remove"}
	assert.False(t, ShouldShowHint(attrs))
}

func TestShouldShowHint_HasBoth(t *testing.T) {
	attrs := Attributes{"name": "foo", "default": "bar", "onerror": "remove"}
	assert.False(t, ShouldShowHint(attrs))
}

func TestShouldShowHint_NilAttrs(t *testing.T) {
	assert.True(t, ShouldShowHint(nil))
}

func TestAppendHint(t *testing.T) {
	result := AppendHint("variable not found", "Hint: use default")
	assert.Equal(t, "variable not found\nHint: use default", result)
}

func TestAppendHint_EmptyHint(t *testing.T) {
	result := AppendHint("variable not found", "")
	assert.Equal(t, "variable not found", result)
}

func TestVarResolver_HintInError(t *testing.T) {
	resolver := NewVarResolver()
	ctx := context.Background()

	mockCtx := newMockContextAccessor(nil)

	// No default, no onerror — should show hint
	attrs := Attributes{"name": "nonexistent"}
	_, err := resolver.Resolve(ctx, mockCtx, attrs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), HintVarNotFound)
}

func TestVarResolver_NoHintWhenDefault(t *testing.T) {
	resolver := NewVarResolver()
	ctx := context.Background()

	mockCtx := newMockContextAccessor(nil)

	// Has default — should NOT show hint (and should return default value)
	attrs := Attributes{"name": "nonexistent", "default": "fallback"}
	result, err := resolver.Resolve(ctx, mockCtx, attrs)
	require.NoError(t, err)
	assert.Equal(t, "fallback", result)
}

func TestVarResolver_NoHintWhenOnerror(t *testing.T) {
	resolver := NewVarResolver()
	ctx := context.Background()

	mockCtx := newMockContextAccessor(nil)

	// Has onerror — should NOT show hint
	attrs := Attributes{"name": "nonexistent", "onerror": "remove"}
	_, err := resolver.Resolve(ctx, mockCtx, attrs)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), HintVarNotFound)
}

func TestIncludeResolver_HintInError(t *testing.T) {
	resolver := NewIncludeResolver()
	ctx := context.Background()

	mockEngine := newMockTemplateExecutor()
	mockCtx := newMockTemplateContextAccessor(nil).WithEngine(mockEngine)

	attrs := Attributes{"template": "nonexistent"}
	_, err := resolver.Resolve(ctx, mockCtx, attrs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), HintTemplateNotFound)
}

func TestRefResolver_NoResolverHint(t *testing.T) {
	resolver := NewRefResolver()
	ctx := context.Background()

	// mockContextAccessor does NOT implement SpecResolverAccessor
	mockCtx := newMockContextAccessor(nil)

	attrs := Attributes{"slug": "my-prompt"}
	_, err := resolver.Resolve(ctx, mockCtx, attrs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), HintRefNoResolver)
}

func TestRefResolver_NotFoundHint(t *testing.T) {
	resolver := NewRefResolver()
	ctx := context.Background()

	// Use a mock ref context with a resolver that returns errors for all lookups
	mockCtx := &mockRefContext{
		resolver: newMockSpecBodyResolver(nil),
	}

	attrs := Attributes{"slug": "missing-prompt"}
	_, err := resolver.Resolve(ctx, mockCtx, attrs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), HintRefNotFound)
}
