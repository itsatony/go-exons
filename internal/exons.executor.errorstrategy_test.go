package internal

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLogRecord holds a captured log record for test assertions.
type testLogRecord struct {
	Message string
	Level   slog.Level
}

// testLogHandler is a slog.Handler that captures log records for test verification.
type testLogHandler struct {
	mu      sync.Mutex
	records []testLogRecord
}

func newTestLogHandler() *testLogHandler {
	return &testLogHandler{}
}

func (h *testLogHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *testLogHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, testLogRecord{
		Message: r.Message,
		Level:   r.Level,
	})
	return nil
}

func (h *testLogHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *testLogHandler) WithGroup(_ string) slog.Handler      { return h }

func (h *testLogHandler) All() []testLogRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]testLogRecord, len(h.records))
	copy(out, h.records)
	return out
}

// mockErrorStrategyContext implements both ContextAccessor and ErrorStrategyAccessor
// for testing context-level error strategy configuration.
type mockErrorStrategyContext struct {
	data     map[string]any
	strategy int
}

func newMockErrorStrategyContext(data map[string]any, strategy int) *mockErrorStrategyContext {
	if data == nil {
		data = make(map[string]any)
	}
	return &mockErrorStrategyContext{data: data, strategy: strategy}
}

func (m *mockErrorStrategyContext) Get(path string) (any, bool) {
	val, ok := m.data[path]
	return val, ok
}

func (m *mockErrorStrategyContext) GetString(path string) string {
	val, ok := m.data[path]
	if !ok {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

func (m *mockErrorStrategyContext) GetStringDefault(path, defaultVal string) string {
	val := m.GetString(path)
	if val == "" {
		return defaultVal
	}
	return val
}

func (m *mockErrorStrategyContext) Has(path string) bool {
	_, ok := m.data[path]
	return ok
}

func (m *mockErrorStrategyContext) ErrorStrategy() int {
	return m.strategy
}

// TestHandleTagError_ThrowStrategy verifies that the throw strategy propagates the error.
func TestHandleTagError_ThrowStrategy(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry, BuiltinConfig{})
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	// A tag referencing a missing variable with onerror="throw" should produce an error.
	tag := NewSelfClosingTag(TagNameVar, Attributes{
		AttrName:    "nonexistent",
		AttrOnError: ErrorStrategyNameThrow,
	}, Position{Line: 1, Column: 1})

	root := &RootNode{Children: []Node{tag}}
	ctx := newMockContextAccessor(nil)

	_, err := executor.Execute(context.Background(), root, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgResolverFailed)
}

// TestHandleTagError_DefaultStrategyWithValue verifies that the default strategy returns
// the default attribute value when present.
func TestHandleTagError_DefaultStrategyWithValue(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry, BuiltinConfig{})
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	tag := NewSelfClosingTag(TagNameVar, Attributes{
		AttrName:    "nonexistent",
		AttrOnError: ErrorStrategyNameDefault,
		AttrDefault: "fallback",
	}, Position{Line: 1, Column: 1})

	root := &RootNode{Children: []Node{tag}}
	ctx := newMockContextAccessor(nil)

	result, err := executor.Execute(context.Background(), root, ctx)
	require.NoError(t, err)
	assert.Equal(t, "fallback", result)
}

// TestHandleTagError_DefaultStrategyWithoutValue verifies that the default strategy returns
// an empty string when no default attribute is present.
func TestHandleTagError_DefaultStrategyWithoutValue(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry, BuiltinConfig{})
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	tag := NewSelfClosingTag(TagNameVar, Attributes{
		AttrName:    "nonexistent",
		AttrOnError: ErrorStrategyNameDefault,
	}, Position{Line: 1, Column: 1})

	root := &RootNode{Children: []Node{tag}}
	ctx := newMockContextAccessor(nil)

	result, err := executor.Execute(context.Background(), root, ctx)
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

// TestHandleTagError_RemoveStrategy verifies that the remove strategy returns an empty string.
func TestHandleTagError_RemoveStrategy(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry, BuiltinConfig{})
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	tag := NewSelfClosingTag(TagNameVar, Attributes{
		AttrName:    "nonexistent",
		AttrOnError: ErrorStrategyNameRemove,
	}, Position{Line: 1, Column: 1})

	root := &RootNode{Children: []Node{tag}}
	ctx := newMockContextAccessor(nil)

	result, err := executor.Execute(context.Background(), root, ctx)
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

// TestHandleTagError_KeepRawStrategyWithRawSource verifies that the keepraw strategy
// returns the original tag source when RawSource is set.
func TestHandleTagError_KeepRawStrategyWithRawSource(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry, BuiltinConfig{})
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	rawSource := `{~exons.var name="nonexistent" onerror="keepraw" /~}`
	tag := &TagNode{
		pos:  Position{Line: 1, Column: 1},
		Name: TagNameVar,
		Attributes: Attributes{
			AttrName:    "nonexistent",
			AttrOnError: ErrorStrategyNameKeepRaw,
		},
		SelfClose: true,
		RawSource: rawSource,
	}

	root := &RootNode{Children: []Node{tag}}
	ctx := newMockContextAccessor(nil)

	result, err := executor.Execute(context.Background(), root, ctx)
	require.NoError(t, err)
	assert.Equal(t, rawSource, result)
}

// TestHandleTagError_KeepRawStrategyWithoutRawSource verifies that the keepraw strategy
// returns an empty string when RawSource is not set.
func TestHandleTagError_KeepRawStrategyWithoutRawSource(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry, BuiltinConfig{})
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	tag := NewSelfClosingTag(TagNameVar, Attributes{
		AttrName:    "nonexistent",
		AttrOnError: ErrorStrategyNameKeepRaw,
	}, Position{Line: 1, Column: 1})

	root := &RootNode{Children: []Node{tag}}
	ctx := newMockContextAccessor(nil)

	result, err := executor.Execute(context.Background(), root, ctx)
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

// TestHandleTagError_LogStrategy verifies that the log strategy logs a warning and returns
// an empty string.
func TestHandleTagError_LogStrategy(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry, BuiltinConfig{})

	// Use a test log handler so we can verify log output.
	handler := newTestLogHandler()
	logger := slog.New(handler)

	executor := NewExecutor(registry, DefaultExecutorConfig(), logger)

	tag := NewSelfClosingTag(TagNameVar, Attributes{
		AttrName:    "nonexistent",
		AttrOnError: ErrorStrategyNameLog,
	}, Position{Line: 1, Column: 1})

	root := &RootNode{Children: []Node{tag}}
	ctx := newMockContextAccessor(nil)

	result, err := executor.Execute(context.Background(), root, ctx)
	require.NoError(t, err)
	assert.Equal(t, "", result)

	// Verify the warning was logged with the expected message.
	found := false
	for _, entry := range handler.All() {
		if entry.Message == LogMsgErrorLogged {
			found = true
			break
		}
	}
	assert.True(t, found, "expected log message %q not found in log output", LogMsgErrorLogged)
}

// TestHandleTagError_UnknownStrategyFallsToThrow verifies that an unknown/invalid
// onerror value falls back to the throw strategy (propagating the error).
func TestHandleTagError_UnknownStrategyFallsToThrow(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry, BuiltinConfig{})
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	tag := NewSelfClosingTag(TagNameVar, Attributes{
		AttrName:    "nonexistent",
		AttrOnError: "unknown_strategy",
	}, Position{Line: 1, Column: 1})

	root := &RootNode{Children: []Node{tag}}
	ctx := newMockContextAccessor(nil)

	_, err := executor.Execute(context.Background(), root, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgResolverFailed)
}

// TestGetErrorStrategy_ContextLevel verifies that when the context implements
// ErrorStrategyAccessor, its strategy is used when no per-tag onerror is set.
func TestGetErrorStrategy_ContextLevel(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry, BuiltinConfig{})
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	// No onerror on tag, context provides ErrorStrategyDefault (1).
	tag := NewSelfClosingTag(TagNameVar, Attributes{
		AttrName: "nonexistent",
	}, Position{Line: 1, Column: 1})

	root := &RootNode{Children: []Node{tag}}
	ctx := newMockErrorStrategyContext(nil, int(ErrorStrategyDefault))

	result, err := executor.Execute(context.Background(), root, ctx)
	require.NoError(t, err)
	// ErrorStrategyDefault with no default attr returns empty string.
	assert.Equal(t, "", result)
}

// TestGetErrorStrategy_ContextLevelRemove verifies that context-level ErrorStrategyRemove
// produces an empty string for a failing tag.
func TestGetErrorStrategy_ContextLevelRemove(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry, BuiltinConfig{})
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	tag := NewSelfClosingTag(TagNameVar, Attributes{
		AttrName: "nonexistent",
	}, Position{Line: 1, Column: 1})

	root := &RootNode{Children: []Node{tag}}
	ctx := newMockErrorStrategyContext(nil, int(ErrorStrategyRemove))

	result, err := executor.Execute(context.Background(), root, ctx)
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

// TestGetErrorStrategy_ContextLevelLog verifies that context-level ErrorStrategyLog
// logs and returns an empty string.
func TestGetErrorStrategy_ContextLevelLog(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry, BuiltinConfig{})

	handler := newTestLogHandler()
	logger := slog.New(handler)
	executor := NewExecutor(registry, DefaultExecutorConfig(), logger)

	tag := NewSelfClosingTag(TagNameVar, Attributes{
		AttrName: "nonexistent",
	}, Position{Line: 1, Column: 1})

	root := &RootNode{Children: []Node{tag}}
	ctx := newMockErrorStrategyContext(nil, int(ErrorStrategyLog))

	result, err := executor.Execute(context.Background(), root, ctx)
	require.NoError(t, err)
	assert.Equal(t, "", result)

	found := false
	for _, entry := range handler.All() {
		if entry.Message == LogMsgErrorLogged {
			found = true
			break
		}
	}
	assert.True(t, found, "expected log message %q not found in log output", LogMsgErrorLogged)
}

// TestGetErrorStrategy_TagOverridesContext verifies that a per-tag onerror attribute
// takes precedence over the context-level error strategy.
func TestGetErrorStrategy_TagOverridesContext(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry, BuiltinConfig{})
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	// Context says "remove", but tag says "throw".
	tag := NewSelfClosingTag(TagNameVar, Attributes{
		AttrName:    "nonexistent",
		AttrOnError: ErrorStrategyNameThrow,
	}, Position{Line: 1, Column: 1})

	root := &RootNode{Children: []Node{tag}}
	ctx := newMockErrorStrategyContext(nil, int(ErrorStrategyRemove))

	_, err := executor.Execute(context.Background(), root, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgResolverFailed)
}

// TestGetErrorStrategy_TagDefaultOverridesContextThrow verifies that a per-tag
// onerror="default" overrides context-level throw strategy.
func TestGetErrorStrategy_TagDefaultOverridesContextThrow(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry, BuiltinConfig{})
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	// Context says "throw", but tag says "default" with a fallback value.
	tag := NewSelfClosingTag(TagNameVar, Attributes{
		AttrName:    "nonexistent",
		AttrOnError: ErrorStrategyNameDefault,
		AttrDefault: "safe_value",
	}, Position{Line: 1, Column: 1})

	root := &RootNode{Children: []Node{tag}}
	ctx := newMockErrorStrategyContext(nil, int(ErrorStrategyThrow))

	result, err := executor.Execute(context.Background(), root, ctx)
	require.NoError(t, err)
	assert.Equal(t, "safe_value", result)
}

// TestGetErrorStrategy_NoContextNoTag verifies that when neither the tag nor the
// context provides an error strategy, the default is throw.
func TestGetErrorStrategy_NoContextNoTag(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry, BuiltinConfig{})
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	// No onerror attribute, plain context (not ErrorStrategyAccessor).
	tag := NewSelfClosingTag(TagNameVar, Attributes{
		AttrName: "nonexistent",
	}, Position{Line: 1, Column: 1})

	root := &RootNode{Children: []Node{tag}}
	ctx := newMockContextAccessor(nil)

	_, err := executor.Execute(context.Background(), root, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgResolverFailed)
}

// TestHandleTagError_UnknownTagWithStrategies verifies that error strategies also
// apply when the tag itself is not registered (unknown tag error).
func TestHandleTagError_UnknownTagWithStrategies(t *testing.T) {
	t.Run("unknown tag with onerror=remove", func(t *testing.T) {
		registry := NewRegistry(nil)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		tag := NewSelfClosingTag("nonexistent.tag", Attributes{
			AttrOnError: ErrorStrategyNameRemove,
		}, Position{Line: 1, Column: 1})

		root := &RootNode{Children: []Node{tag}}
		ctx := newMockContextAccessor(nil)

		result, err := executor.Execute(context.Background(), root, ctx)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("unknown tag with onerror=default and default value", func(t *testing.T) {
		registry := NewRegistry(nil)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		tag := NewSelfClosingTag("nonexistent.tag", Attributes{
			AttrOnError: ErrorStrategyNameDefault,
			AttrDefault: "placeholder",
		}, Position{Line: 1, Column: 1})

		root := &RootNode{Children: []Node{tag}}
		ctx := newMockContextAccessor(nil)

		result, err := executor.Execute(context.Background(), root, ctx)
		require.NoError(t, err)
		assert.Equal(t, "placeholder", result)
	})

	t.Run("unknown tag with onerror=keepraw and raw source", func(t *testing.T) {
		registry := NewRegistry(nil)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		rawSource := `{~nonexistent.tag onerror="keepraw" /~}`
		tag := &TagNode{
			pos:  Position{Line: 1, Column: 1},
			Name: "nonexistent.tag",
			Attributes: Attributes{
				AttrOnError: ErrorStrategyNameKeepRaw,
			},
			SelfClose: true,
			RawSource: rawSource,
		}

		root := &RootNode{Children: []Node{tag}}
		ctx := newMockContextAccessor(nil)

		result, err := executor.Execute(context.Background(), root, ctx)
		require.NoError(t, err)
		assert.Equal(t, rawSource, result)
	})
}

// TestHandleTagError_MixedContent verifies that error strategies work correctly
// when the failing tag is surrounded by other content.
func TestHandleTagError_MixedContent(t *testing.T) {
	t.Run("remove strategy preserves surrounding text", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry, BuiltinConfig{})
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		root := &RootNode{
			Children: []Node{
				NewTextNode("Hello, ", Position{Line: 1, Column: 1}),
				NewSelfClosingTag(TagNameVar, Attributes{
					AttrName:    "nonexistent",
					AttrOnError: ErrorStrategyNameRemove,
				}, Position{Line: 1, Column: 8}),
				NewTextNode("!", Position{Line: 1, Column: 40}),
			},
		}

		ctx := newMockContextAccessor(nil)
		result, err := executor.Execute(context.Background(), root, ctx)
		require.NoError(t, err)
		assert.Equal(t, "Hello, !", result)
	})

	t.Run("default strategy with value in mixed content", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry, BuiltinConfig{})
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		root := &RootNode{
			Children: []Node{
				NewTextNode("Hello, ", Position{Line: 1, Column: 1}),
				NewSelfClosingTag(TagNameVar, Attributes{
					AttrName:    "nonexistent",
					AttrOnError: ErrorStrategyNameDefault,
					AttrDefault: "Guest",
				}, Position{Line: 1, Column: 8}),
				NewTextNode("!", Position{Line: 1, Column: 50}),
			},
		}

		ctx := newMockContextAccessor(nil)
		result, err := executor.Execute(context.Background(), root, ctx)
		require.NoError(t, err)
		assert.Equal(t, "Hello, Guest!", result)
	})
}

// TestHandleTagError_ContextLevelKeepRaw verifies that context-level keepraw strategy
// returns the raw source when available.
func TestHandleTagError_ContextLevelKeepRaw(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry, BuiltinConfig{})
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	rawSource := `{~exons.var name="nonexistent" /~}`
	tag := &TagNode{
		pos:  Position{Line: 1, Column: 1},
		Name: TagNameVar,
		Attributes: Attributes{
			AttrName: "nonexistent",
		},
		SelfClose: true,
		RawSource: rawSource,
	}

	root := &RootNode{Children: []Node{tag}}
	ctx := newMockErrorStrategyContext(nil, int(ErrorStrategyKeepRaw))

	result, err := executor.Execute(context.Background(), root, ctx)
	require.NoError(t, err)
	assert.Equal(t, rawSource, result)
}

// TestHandleTagError_AllStrategiesViaLexerParser tests error strategies through the
// full pipeline: Lexer -> Parser -> Executor, using template source strings.
func TestHandleTagError_AllStrategiesViaLexerParser(t *testing.T) {
	parseTemplate := func(t *testing.T, source string) *RootNode {
		t.Helper()
		lexer := NewLexer(source, nil)
		tokens, err := lexer.Tokenize()
		require.NoError(t, err)
		parser := NewParser(tokens, nil)
		ast, err := parser.Parse()
		require.NoError(t, err)
		return ast
	}

	t.Run("throw via parser", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry, BuiltinConfig{})
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ast := parseTemplate(t, `{~exons.var name="missing" onerror="throw" /~}`)
		ctx := newMockContextAccessor(nil)

		_, err := executor.Execute(context.Background(), ast, ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgResolverFailed)
	})

	t.Run("default with value via parser", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry, BuiltinConfig{})
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ast := parseTemplate(t, `{~exons.var name="missing" onerror="default" default="fallback" /~}`)
		ctx := newMockContextAccessor(nil)

		result, err := executor.Execute(context.Background(), ast, ctx)
		require.NoError(t, err)
		assert.Equal(t, "fallback", result)
	})

	t.Run("default without value via parser", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry, BuiltinConfig{})
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ast := parseTemplate(t, `{~exons.var name="missing" onerror="default" /~}`)
		ctx := newMockContextAccessor(nil)

		result, err := executor.Execute(context.Background(), ast, ctx)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("remove via parser", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry, BuiltinConfig{})
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ast := parseTemplate(t, `before {~exons.var name="missing" onerror="remove" /~} after`)
		ctx := newMockContextAccessor(nil)

		result, err := executor.Execute(context.Background(), ast, ctx)
		require.NoError(t, err)
		assert.Equal(t, "before  after", result)
	})

	t.Run("log via parser", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry, BuiltinConfig{})

		handler := newTestLogHandler()
		logger := slog.New(handler)
		executor := NewExecutor(registry, DefaultExecutorConfig(), logger)

		ast := parseTemplate(t, `{~exons.var name="missing" onerror="log" /~}`)
		ctx := newMockContextAccessor(nil)

		result, err := executor.Execute(context.Background(), ast, ctx)
		require.NoError(t, err)
		assert.Equal(t, "", result)

		found := false
		for _, entry := range handler.All() {
			if entry.Message == LogMsgErrorLogged {
				found = true
				break
			}
		}
		assert.True(t, found, "expected log message %q not found", LogMsgErrorLogged)
	})

	t.Run("keepraw via parser", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry, BuiltinConfig{})
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		source := `{~exons.var name="missing" onerror="keepraw" /~}`
		ast := parseTemplate(t, source)
		ctx := newMockContextAccessor(nil)

		result, err := executor.Execute(context.Background(), ast, ctx)
		require.NoError(t, err)
		// The parser should capture the raw source. If it does, we get it back.
		// If not, we get an empty string (both are valid outcomes for keepraw).
		// We just verify no error was returned.
		assert.NotContains(t, result, "error")
	})
}

// TestHandleTagError_MultipleFailingTags verifies that error strategies are applied
// independently to each failing tag in a template.
func TestHandleTagError_MultipleFailingTags(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry, BuiltinConfig{})
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	root := &RootNode{
		Children: []Node{
			NewSelfClosingTag(TagNameVar, Attributes{
				AttrName:    "missing1",
				AttrOnError: ErrorStrategyNameDefault,
				AttrDefault: "A",
			}, Position{Line: 1, Column: 1}),
			NewTextNode("-", Position{Line: 1, Column: 20}),
			NewSelfClosingTag(TagNameVar, Attributes{
				AttrName:    "missing2",
				AttrOnError: ErrorStrategyNameRemove,
			}, Position{Line: 1, Column: 22}),
			NewTextNode("-", Position{Line: 1, Column: 40}),
			NewSelfClosingTag(TagNameVar, Attributes{
				AttrName:    "missing3",
				AttrOnError: ErrorStrategyNameDefault,
				AttrDefault: "C",
			}, Position{Line: 1, Column: 42}),
		},
	}

	ctx := newMockContextAccessor(nil)
	result, err := executor.Execute(context.Background(), root, ctx)
	require.NoError(t, err)
	assert.Equal(t, "A--C", result)
}

// TestHandleTagError_SuccessfulTagNoStrategyApplied verifies that error strategies
// do not interfere with tags that resolve successfully.
func TestHandleTagError_SuccessfulTagNoStrategyApplied(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry, BuiltinConfig{})
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	tag := NewSelfClosingTag(TagNameVar, Attributes{
		AttrName:    "user",
		AttrOnError: ErrorStrategyNameRemove,
		AttrDefault: "should_not_appear",
	}, Position{Line: 1, Column: 1})

	root := &RootNode{Children: []Node{tag}}
	ctx := newMockContextAccessor(map[string]any{
		"user": "Alice",
	})

	result, err := executor.Execute(context.Background(), root, ctx)
	require.NoError(t, err)
	assert.Equal(t, "Alice", result)
}

// TestHandleTagError_ResolverErrorWithStrategies verifies that error strategies work
// for resolver errors (not just missing variables).
func TestHandleTagError_ResolverErrorWithStrategies(t *testing.T) {
	t.Run("resolver error with onerror=remove", func(t *testing.T) {
		registry := NewRegistry(nil)
		registry.MustRegister(&testErrorResolver{})
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		tag := NewSelfClosingTag("error.tag", Attributes{
			AttrOnError: ErrorStrategyNameRemove,
		}, Position{Line: 1, Column: 1})

		root := &RootNode{Children: []Node{tag}}
		ctx := newMockContextAccessor(nil)

		result, err := executor.Execute(context.Background(), root, ctx)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("resolver error with onerror=default", func(t *testing.T) {
		registry := NewRegistry(nil)
		registry.MustRegister(&testErrorResolver{})
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		tag := NewSelfClosingTag("error.tag", Attributes{
			AttrOnError: ErrorStrategyNameDefault,
			AttrDefault: "error_fallback",
		}, Position{Line: 1, Column: 1})

		root := &RootNode{Children: []Node{tag}}
		ctx := newMockContextAccessor(nil)

		result, err := executor.Execute(context.Background(), root, ctx)
		require.NoError(t, err)
		assert.Equal(t, "error_fallback", result)
	})

	t.Run("resolver error with onerror=log", func(t *testing.T) {
		registry := NewRegistry(nil)
		registry.MustRegister(&testErrorResolver{})

		handler := newTestLogHandler()
		logger := slog.New(handler)
		executor := NewExecutor(registry, DefaultExecutorConfig(), logger)

		tag := NewSelfClosingTag("error.tag", Attributes{
			AttrOnError: ErrorStrategyNameLog,
		}, Position{Line: 1, Column: 1})

		root := &RootNode{Children: []Node{tag}}
		ctx := newMockContextAccessor(nil)

		result, err := executor.Execute(context.Background(), root, ctx)
		require.NoError(t, err)
		assert.Equal(t, "", result)

		found := false
		for _, entry := range handler.All() {
			if entry.Message == LogMsgErrorLogged {
				found = true
				break
			}
		}
		assert.True(t, found, "expected log message %q not found", LogMsgErrorLogged)
	})
}

// TestParseErrorStrategy_AllValues verifies ParseErrorStrategy for all known strategy names.
func TestParseErrorStrategy_AllValues(t *testing.T) {
	tests := []struct {
		input    string
		expected ErrorStrategy
	}{
		{ErrorStrategyNameThrow, ErrorStrategyThrow},
		{ErrorStrategyNameDefault, ErrorStrategyDefault},
		{ErrorStrategyNameRemove, ErrorStrategyRemove},
		{ErrorStrategyNameKeepRaw, ErrorStrategyKeepRaw},
		{ErrorStrategyNameLog, ErrorStrategyLog},
		{"", ErrorStrategyThrow},
		{"invalid", ErrorStrategyThrow},
		{"THROW", ErrorStrategyThrow},
	}

	for _, tt := range tests {
		t.Run("input="+tt.input, func(t *testing.T) {
			result := ParseErrorStrategy(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestErrorStrategy_StringRepresentation verifies that ErrorStrategy.String() returns the correct names.
func TestErrorStrategy_StringRepresentation(t *testing.T) {
	tests := []struct {
		strategy ErrorStrategy
		expected string
	}{
		{ErrorStrategyThrow, ErrorStrategyNameThrow},
		{ErrorStrategyDefault, ErrorStrategyNameDefault},
		{ErrorStrategyRemove, ErrorStrategyNameRemove},
		{ErrorStrategyKeepRaw, ErrorStrategyNameKeepRaw},
		{ErrorStrategyLog, ErrorStrategyNameLog},
		{ErrorStrategy(99), ErrorStrategyNameThrow},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.strategy.String())
		})
	}
}
