package internal

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock types for RefResolver tests ---

// mockSpecBodyResolver implements SpecBodyResolver for testing.
type mockSpecBodyResolver struct {
	bodies map[string]string // key: "slug:version"
	err    error
}

func newMockSpecBodyResolver(bodies map[string]string) *mockSpecBodyResolver {
	if bodies == nil {
		bodies = make(map[string]string)
	}
	return &mockSpecBodyResolver{bodies: bodies}
}

func (m *mockSpecBodyResolver) ResolveSpecBody(ctx context.Context, slug, version string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	key := slug + ":" + version
	if body, ok := m.bodies[key]; ok {
		return body, nil
	}
	return "", fmt.Errorf("not found: %s", key)
}

// mockRefContext implements SpecResolverAccessor, RefDepthAccessor, and RefChainAccessor.
type mockRefContext struct {
	resolver SpecBodyResolver
	depth    int
	chain    []string
}

func (m *mockRefContext) SpecResolver() interface{} {
	if m.resolver == nil {
		return nil
	}
	return m.resolver
}

func (m *mockRefContext) RefDepth() int {
	return m.depth
}

func (m *mockRefContext) RefChain() []string {
	return m.chain
}

// mockRefContextNilResolver returns a non-nil interface{} wrapping a nil value
// to test the nil-resolver path inside getSpecResolver.
type mockRefContextNilResolver struct{}

func (m *mockRefContextNilResolver) SpecResolver() interface{} {
	return nil
}

// mockRefContextWrongType returns a resolver value that does not implement SpecBodyResolver.
type mockRefContextWrongType struct{}

func (m *mockRefContextWrongType) SpecResolver() interface{} {
	return "not a SpecBodyResolver"
}

// --- Tests ---

func TestRefResolver_TagName(t *testing.T) {
	resolver := NewRefResolver()
	assert.Equal(t, TagNameRef, resolver.TagName())
}

func TestRefResolver_Validate(t *testing.T) {
	resolver := NewRefResolver()

	t.Run("missing slug attribute returns error", func(t *testing.T) {
		err := resolver.Validate(Attributes{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgRefMissingSlug)
	})

	t.Run("nil attributes returns error", func(t *testing.T) {
		err := resolver.Validate(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgRefMissingSlug)
	})

	t.Run("present slug attribute returns nil", func(t *testing.T) {
		err := resolver.Validate(Attributes{AttrSlug: "my-prompt"})
		assert.NoError(t, err)
	})

	t.Run("empty slug value still passes validate", func(t *testing.T) {
		// Validate only checks presence via Has, not emptiness
		err := resolver.Validate(Attributes{AttrSlug: ""})
		assert.NoError(t, err)
	})
}

func TestRefResolver_Resolve_BasicSlug(t *testing.T) {
	resolver := NewRefResolver()
	bodies := map[string]string{
		"my-prompt:latest": "Hello from my-prompt",
	}
	mockResolver := newMockSpecBodyResolver(bodies)
	execCtx := &mockRefContext{resolver: mockResolver, depth: 0, chain: nil}
	attrs := Attributes{AttrSlug: "my-prompt"}

	result, err := resolver.Resolve(context.Background(), execCtx, attrs)
	require.NoError(t, err)
	assert.Equal(t, "Hello from my-prompt", result)
}

func TestRefResolver_Resolve_SlugAtVersionSyntax(t *testing.T) {
	resolver := NewRefResolver()
	bodies := map[string]string{
		"my-prompt:v2": "Hello from v2",
	}
	mockResolver := newMockSpecBodyResolver(bodies)
	execCtx := &mockRefContext{resolver: mockResolver, depth: 0, chain: nil}
	attrs := Attributes{AttrSlug: "my-prompt@v2"}

	result, err := resolver.Resolve(context.Background(), execCtx, attrs)
	require.NoError(t, err)
	assert.Equal(t, "Hello from v2", result)
}

func TestRefResolver_Resolve_SlugAtVersionMultipleAtSigns(t *testing.T) {
	// Uses LastIndex of "@", so "a@b@v3" should parse as slug="a@b", version="v3"
	// But "a@b" will fail slug validation since @ is not allowed
	resolver := NewRefResolver()
	mockResolver := newMockSpecBodyResolver(nil)
	execCtx := &mockRefContext{resolver: mockResolver, depth: 0, chain: nil}
	attrs := Attributes{AttrSlug: "a@b@v3"}

	_, err := resolver.Resolve(context.Background(), execCtx, attrs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgRefInvalidSlug)
}

func TestRefResolver_Resolve_ExplicitVersionOverridesAtVersion(t *testing.T) {
	resolver := NewRefResolver()
	bodies := map[string]string{
		"my-prompt:v3": "Hello from v3",
	}
	mockResolver := newMockSpecBodyResolver(bodies)
	execCtx := &mockRefContext{resolver: mockResolver, depth: 0, chain: nil}
	// slug@v2 would parse version=v2, but explicit version=v3 overrides
	attrs := Attributes{
		AttrSlug:    "my-prompt@v2",
		AttrVersion: "v3",
	}

	result, err := resolver.Resolve(context.Background(), execCtx, attrs)
	require.NoError(t, err)
	assert.Equal(t, "Hello from v3", result)
}

func TestRefResolver_Resolve_ExplicitVersionAttribute(t *testing.T) {
	resolver := NewRefResolver()
	bodies := map[string]string{
		"my-prompt:v5": "Hello from v5",
	}
	mockResolver := newMockSpecBodyResolver(bodies)
	execCtx := &mockRefContext{resolver: mockResolver, depth: 0, chain: nil}
	attrs := Attributes{
		AttrSlug:    "my-prompt",
		AttrVersion: "v5",
	}

	result, err := resolver.Resolve(context.Background(), execCtx, attrs)
	require.NoError(t, err)
	assert.Equal(t, "Hello from v5", result)
}

func TestRefResolver_Resolve_EmptyVersionAttributeDoesNotOverride(t *testing.T) {
	resolver := NewRefResolver()
	bodies := map[string]string{
		"my-prompt:v2": "Hello from v2",
	}
	mockResolver := newMockSpecBodyResolver(bodies)
	execCtx := &mockRefContext{resolver: mockResolver, depth: 0, chain: nil}
	// slug@v2 gives version="v2"; empty version attribute should not override
	attrs := Attributes{
		AttrSlug:    "my-prompt@v2",
		AttrVersion: "",
	}

	result, err := resolver.Resolve(context.Background(), execCtx, attrs)
	require.NoError(t, err)
	assert.Equal(t, "Hello from v2", result)
}

func TestRefResolver_Resolve_InvalidSlugFormat(t *testing.T) {
	resolver := NewRefResolver()
	mockResolver := newMockSpecBodyResolver(nil)
	execCtx := &mockRefContext{resolver: mockResolver, depth: 0, chain: nil}

	testCases := []struct {
		name string
		slug string
	}{
		{"starts with digit", "9abc"},
		{"uppercase letters", "MyPrompt"},
		{"underscores", "my_prompt"},
		{"spaces", "my prompt"},
		{"special characters", "my!prompt"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			attrs := Attributes{AttrSlug: tc.slug}
			_, err := resolver.Resolve(context.Background(), execCtx, attrs)
			require.Error(t, err)
			assert.Contains(t, err.Error(), ErrMsgRefInvalidSlug)
		})
	}
}

func TestRefResolver_Resolve_EmptySlug(t *testing.T) {
	resolver := NewRefResolver()
	execCtx := &mockRefContext{}

	t.Run("empty slug value", func(t *testing.T) {
		attrs := Attributes{AttrSlug: ""}
		_, err := resolver.Resolve(context.Background(), execCtx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgRefMissingSlug)
	})

	t.Run("missing slug key", func(t *testing.T) {
		attrs := Attributes{}
		_, err := resolver.Resolve(context.Background(), execCtx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgRefMissingSlug)
	})
}

func TestRefResolver_Resolve_NoResolverInContext(t *testing.T) {
	resolver := NewRefResolver()

	t.Run("plain struct without accessor", func(t *testing.T) {
		attrs := Attributes{AttrSlug: "my-prompt"}
		_, err := resolver.Resolve(context.Background(), "not a context", attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgRefNoResolver)
	})

	t.Run("accessor returns nil resolver", func(t *testing.T) {
		execCtx := &mockRefContextNilResolver{}
		attrs := Attributes{AttrSlug: "my-prompt"}
		_, err := resolver.Resolve(context.Background(), execCtx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgRefNoResolver)
	})

	t.Run("accessor returns wrong type", func(t *testing.T) {
		execCtx := &mockRefContextWrongType{}
		attrs := Attributes{AttrSlug: "my-prompt"}
		_, err := resolver.Resolve(context.Background(), execCtx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgRefNoResolver)
	})
}

func TestRefResolver_Resolve_DepthExceeded(t *testing.T) {
	resolver := NewRefResolver()
	mockResolver := newMockSpecBodyResolver(nil)

	t.Run("depth equals RefMaxDepth", func(t *testing.T) {
		execCtx := &mockRefContext{resolver: mockResolver, depth: RefMaxDepth, chain: nil}
		attrs := Attributes{AttrSlug: "my-prompt"}

		_, err := resolver.Resolve(context.Background(), execCtx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgRefDepthExceeded)
	})

	t.Run("depth exceeds RefMaxDepth", func(t *testing.T) {
		execCtx := &mockRefContext{resolver: mockResolver, depth: RefMaxDepth + 5, chain: nil}
		attrs := Attributes{AttrSlug: "my-prompt"}

		_, err := resolver.Resolve(context.Background(), execCtx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgRefDepthExceeded)
	})

	t.Run("depth just below RefMaxDepth succeeds", func(t *testing.T) {
		bodies := map[string]string{
			"my-prompt:latest": "ok",
		}
		mockResolverWithBody := newMockSpecBodyResolver(bodies)
		execCtx := &mockRefContext{resolver: mockResolverWithBody, depth: RefMaxDepth - 1, chain: nil}
		attrs := Attributes{AttrSlug: "my-prompt"}

		result, err := resolver.Resolve(context.Background(), execCtx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "ok", result)
	})
}

func TestRefResolver_Resolve_CircularReference(t *testing.T) {
	resolver := NewRefResolver()
	mockResolver := newMockSpecBodyResolver(nil)

	t.Run("direct circular reference", func(t *testing.T) {
		execCtx := &mockRefContext{
			resolver: mockResolver,
			depth:    1,
			chain:    []string{"my-prompt"},
		}
		attrs := Attributes{AttrSlug: "my-prompt"}

		_, err := resolver.Resolve(context.Background(), execCtx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgRefCircular)
		assert.Contains(t, err.Error(), "my-prompt")
	})

	t.Run("indirect circular reference", func(t *testing.T) {
		execCtx := &mockRefContext{
			resolver: mockResolver,
			depth:    2,
			chain:    []string{"prompt-a", "prompt-b"},
		}
		attrs := Attributes{AttrSlug: "prompt-a"}

		_, err := resolver.Resolve(context.Background(), execCtx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgRefCircular)
		assert.Contains(t, err.Error(), "prompt-a")
	})

	t.Run("chain with ref_chain metadata", func(t *testing.T) {
		execCtx := &mockRefContext{
			resolver: mockResolver,
			depth:    3,
			chain:    []string{"prompt-a", "prompt-b", "prompt-c"},
		}
		attrs := Attributes{AttrSlug: "prompt-a"}

		_, err := resolver.Resolve(context.Background(), execCtx, attrs)
		require.Error(t, err)
		// The chain should include the full path
		assert.Contains(t, err.Error(), "prompt-a -> prompt-b -> prompt-c -> prompt-a")
	})

	t.Run("no circular reference when slug not in chain", func(t *testing.T) {
		bodies := map[string]string{
			"prompt-d:latest": "Hello from D",
		}
		mockResolverWithBody := newMockSpecBodyResolver(bodies)
		execCtx := &mockRefContext{
			resolver: mockResolverWithBody,
			depth:    2,
			chain:    []string{"prompt-a", "prompt-b"},
		}
		attrs := Attributes{AttrSlug: "prompt-d"}

		result, err := resolver.Resolve(context.Background(), execCtx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "Hello from D", result)
	})
}

func TestRefResolver_Resolve_ResolverReturnsError(t *testing.T) {
	resolver := NewRefResolver()
	bodies := map[string]string{} // empty: no prompts registered
	mockResolver := newMockSpecBodyResolver(bodies)
	execCtx := &mockRefContext{resolver: mockResolver, depth: 0, chain: nil}

	t.Run("prompt not found", func(t *testing.T) {
		attrs := Attributes{AttrSlug: "nonexistent"}
		_, err := resolver.Resolve(context.Background(), execCtx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgRefNotFound)
		assert.Contains(t, err.Error(), "nonexistent")
	})

	t.Run("resolver configured with error", func(t *testing.T) {
		errResolver := &mockSpecBodyResolver{
			bodies: nil,
			err:    fmt.Errorf("connection failed"),
		}
		execCtx := &mockRefContext{resolver: errResolver, depth: 0, chain: nil}
		attrs := Attributes{AttrSlug: "my-prompt"}

		_, err := resolver.Resolve(context.Background(), execCtx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgRefNotFound)
	})
}

func TestIsValidSpecSlug(t *testing.T) {
	validSlugs := []struct {
		name string
		slug string
	}{
		{"single letter", "a"},
		{"simple slug", "my-prompt"},
		{"slug with digits", "test-123"},
		{"all lowercase", "abcdef"},
		{"mixed alphanumeric", "a1b2c3"},
		{"hyphenated", "my-long-prompt-name"},
		{"letter then digits", "x99"},
	}

	for _, tc := range validSlugs {
		t.Run("valid: "+tc.name, func(t *testing.T) {
			assert.True(t, isValidSpecSlug(tc.slug), "expected slug %q to be valid", tc.slug)
		})
	}

	invalidSlugs := []struct {
		name string
		slug string
	}{
		{"empty string", ""},
		{"starts with digit", "9abc"},
		{"uppercase letters", "UPPER"},
		{"mixed case", "MyPrompt"},
		{"underscores", "my_prompt"},
		{"spaces", "my prompt"},
		{"starts with hyphen", "-prompt"},
		{"special characters", "my!prompt"},
		{"dots", "my.prompt"},
		{"at sign", "my@prompt"},
		{"leading space", " abc"},
	}

	for _, tc := range invalidSlugs {
		t.Run("invalid: "+tc.name, func(t *testing.T) {
			assert.False(t, isValidSpecSlug(tc.slug), "expected slug %q to be invalid", tc.slug)
		})
	}
}

func TestGetSpecResolver(t *testing.T) {
	t.Run("with valid accessor and resolver", func(t *testing.T) {
		mockResolver := newMockSpecBodyResolver(nil)
		execCtx := &mockRefContext{resolver: mockResolver}

		resolver, ok := getSpecResolver(execCtx)
		assert.True(t, ok)
		assert.NotNil(t, resolver)
		assert.Equal(t, mockResolver, resolver)
	})

	t.Run("without accessor interface", func(t *testing.T) {
		resolver, ok := getSpecResolver("not an accessor")
		assert.False(t, ok)
		assert.Nil(t, resolver)
	})

	t.Run("nil execCtx", func(t *testing.T) {
		resolver, ok := getSpecResolver(nil)
		assert.False(t, ok)
		assert.Nil(t, resolver)
	})

	t.Run("accessor returns nil resolver", func(t *testing.T) {
		execCtx := &mockRefContextNilResolver{}
		resolver, ok := getSpecResolver(execCtx)
		assert.False(t, ok)
		assert.Nil(t, resolver)
	})

	t.Run("accessor returns wrong type", func(t *testing.T) {
		execCtx := &mockRefContextWrongType{}
		resolver, ok := getSpecResolver(execCtx)
		assert.False(t, ok)
		assert.Nil(t, resolver)
	})

	t.Run("accessor with nil SpecBodyResolver field", func(t *testing.T) {
		execCtx := &mockRefContext{resolver: nil}
		resolver, ok := getSpecResolver(execCtx)
		assert.False(t, ok)
		assert.Nil(t, resolver)
	})
}

func TestGetRefDepth(t *testing.T) {
	t.Run("with depth accessor", func(t *testing.T) {
		execCtx := &mockRefContext{depth: 5}
		depth := getRefDepth(execCtx)
		assert.Equal(t, 5, depth)
	})

	t.Run("with zero depth", func(t *testing.T) {
		execCtx := &mockRefContext{depth: 0}
		depth := getRefDepth(execCtx)
		assert.Equal(t, 0, depth)
	})

	t.Run("without depth accessor returns 0", func(t *testing.T) {
		depth := getRefDepth("not an accessor")
		assert.Equal(t, 0, depth)
	})

	t.Run("nil execCtx returns 0", func(t *testing.T) {
		depth := getRefDepth(nil)
		assert.Equal(t, 0, depth)
	})
}

func TestGetRefChain(t *testing.T) {
	t.Run("with chain accessor", func(t *testing.T) {
		chain := []string{"prompt-a", "prompt-b"}
		execCtx := &mockRefContext{chain: chain}
		result := getRefChain(execCtx)
		assert.Equal(t, chain, result)
	})

	t.Run("with nil chain", func(t *testing.T) {
		execCtx := &mockRefContext{chain: nil}
		result := getRefChain(execCtx)
		assert.Nil(t, result)
	})

	t.Run("with empty chain", func(t *testing.T) {
		execCtx := &mockRefContext{chain: []string{}}
		result := getRefChain(execCtx)
		assert.Empty(t, result)
	})

	t.Run("without chain accessor returns nil", func(t *testing.T) {
		result := getRefChain("not an accessor")
		assert.Nil(t, result)
	})

	t.Run("nil execCtx returns nil", func(t *testing.T) {
		result := getRefChain(nil)
		assert.Nil(t, result)
	})
}

func TestNewRefCircularError(t *testing.T) {
	t.Run("formats chain correctly", func(t *testing.T) {
		err := NewRefCircularError("prompt-a", []string{"prompt-a", "prompt-b", "prompt-a"})
		errStr := err.Error()
		assert.Contains(t, errStr, ErrMsgRefCircular)
		assert.Contains(t, errStr, TagNameRef)
		assert.Contains(t, errStr, "prompt-a -> prompt-b -> prompt-a")
		assert.Contains(t, errStr, LogFieldSpecSlug)
		assert.Contains(t, errStr, LogFieldRefChain)
	})

	t.Run("single element chain", func(t *testing.T) {
		err := NewRefCircularError("x", []string{"x"})
		errStr := err.Error()
		assert.Contains(t, errStr, ErrMsgRefCircular)
		assert.Contains(t, errStr, "x")
	})
}

func TestRefResolver_Resolve_SlugOnlyAtEnd(t *testing.T) {
	// Ensure "@" at position 0 does not split (atIdx > 0 check)
	resolver := NewRefResolver()
	mockResolver := newMockSpecBodyResolver(nil)
	execCtx := &mockRefContext{resolver: mockResolver, depth: 0, chain: nil}

	// "@v2" has @ at index 0, so atIdx > 0 is false, slug stays "@v2"
	// "@v2" is invalid slug format (starts with @)
	attrs := Attributes{AttrSlug: "@v2"}
	_, err := resolver.Resolve(context.Background(), execCtx, attrs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgRefInvalidSlug)
}

func TestRefResolver_Resolve_ContextCancellation(t *testing.T) {
	// The resolver passes ctx to ResolveSpecBody, so we verify it propagates
	resolver := NewRefResolver()
	bodies := map[string]string{
		"my-prompt:latest": "ok",
	}
	mockResolver := newMockSpecBodyResolver(bodies)
	execCtx := &mockRefContext{resolver: mockResolver, depth: 0, chain: nil}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// The mock doesn't check ctx, so it still returns the body.
	// This test verifies the ctx is at least passed through without panic.
	attrs := Attributes{AttrSlug: "my-prompt"}
	result, err := resolver.Resolve(ctx, execCtx, attrs)
	require.NoError(t, err)
	assert.Equal(t, "ok", result)
}
