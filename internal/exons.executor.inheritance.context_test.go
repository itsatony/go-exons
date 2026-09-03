package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockContextSourceResolver implements ContextTemplateSourceResolver for testing.
type mockContextSourceResolver struct {
	sources  map[string]string
	failWith error
	seenCtx  context.Context
}

func (m *mockContextSourceResolver) ResolveTemplateSource(ctx context.Context, name string) (string, bool, error) {
	m.seenCtx = ctx
	if m.failWith != nil {
		return "", false, m.failWith
	}
	src, ok := m.sources[name]
	return src, ok, nil
}

type ctxKey struct{}

func TestNewContextInheritanceResolver(t *testing.T) {
	t.Run("resolves through the context resolver with the request context", func(t *testing.T) {
		mock := &mockContextSourceResolver{sources: map[string]string{
			"base": `A{~exons.block name="c"~}p{~/exons.block~}B`,
		}}
		r := NewContextInheritanceResolver(mock, 5, DefaultLexerConfig())
		child, info := parseChildForTest(t, `{~exons.extends template="base" /~}{~exons.block name="c"~}x{~/exons.block~}`)

		ctx := context.WithValue(context.Background(), ctxKey{}, "tenant")
		root, err := r.ResolveInheritance(ctx, child, info, 0)
		require.NoError(t, err)
		require.NotNil(t, root)
		assert.Equal(t, "tenant", mock.seenCtx.Value(ctxKey{}), "the caller's context reaches the resolver")
	})

	t.Run("default depth applies for a non-positive bound", func(t *testing.T) {
		r := NewContextInheritanceResolver(&mockContextSourceResolver{}, 0, DefaultLexerConfig())
		assert.Equal(t, DefaultMaxInheritanceDepth, r.maxDepth)
	})

	t.Run("found=false is template-not-found naming the parent", func(t *testing.T) {
		r := NewContextInheritanceResolver(&mockContextSourceResolver{sources: map[string]string{}}, 5, DefaultLexerConfig())
		child, info := parseChildForTest(t, `{~exons.extends template="gone" /~}`)
		_, err := r.ResolveInheritance(context.Background(), child, info, 0)
		require.Error(t, err)
		var be *BuiltinError
		require.True(t, errors.As(err, &be))
		assert.Equal(t, ErrMsgTemplateNotFound, be.Message)
		assert.Equal(t, TagNameExtends, be.TagName)
		assert.Equal(t, "gone", be.Metadata[MetaKeyTemplateName])
		assert.Nil(t, be.Unwrap(), "an absent parent has no external cause")
	})

	t.Run("a lookup error is its own failure, carrying the cause", func(t *testing.T) {
		cause := errors.New("store unreachable")
		r := NewContextInheritanceResolver(&mockContextSourceResolver{failWith: cause}, 5, DefaultLexerConfig())
		child, info := parseChildForTest(t, `{~exons.extends template="base" /~}`)
		_, err := r.ResolveInheritance(context.Background(), child, info, 0)
		require.Error(t, err)
		var be *BuiltinError
		require.True(t, errors.As(err, &be))
		assert.Equal(t, ErrMsgTemplateSourceLookupFailed, be.Message, "not rewritten as not-found")
		assert.Equal(t, "base", be.Metadata[MetaKeyTemplateName])
		assert.True(t, errors.Is(err, cause), "errors.Is reaches the cause through Unwrap")
		assert.Contains(t, err.Error(), cause.Error(), "and the message shows it")
	})

	t.Run("the context resolver is the only source — the registry resolver is not consulted", func(t *testing.T) {
		registry := newMockTemplateSourceResolver(map[string]string{"base": "FROM REGISTRY"})
		r := NewContextInheritanceResolver(&mockContextSourceResolver{sources: map[string]string{}}, 5, DefaultLexerConfig())
		r.templateResolver = registry
		child, info := parseChildForTest(t, `{~exons.extends template="base" /~}`)
		_, err := r.ResolveInheritance(context.Background(), child, info, 0)
		require.Error(t, err, "the registry knows the name and must not answer for the store")
		assert.Contains(t, err.Error(), ErrMsgTemplateNotFound)
	})
}

// parseChildForTest lexes and parses a child template and extracts its inheritance info.
func parseChildForTest(t *testing.T, source string) (*RootNode, *InheritanceInfo) {
	t.Helper()
	lexer := NewLexerWithConfig(source, DefaultLexerConfig(), nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)
	root, err := NewParserWithSource(tokens, source, DefaultLexerConfig(), nil).Parse()
	require.NoError(t, err)
	info, err := ExtractInheritanceInfo(root)
	require.NoError(t, err)
	require.NotNil(t, info)
	return root, info
}
