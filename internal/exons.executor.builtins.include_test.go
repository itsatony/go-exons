package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTemplateExecutor implements TemplateExecutor for testing
type mockTemplateExecutor struct {
	templates     map[string]string
	executeFunc   func(ctx context.Context, name string, data map[string]any) (string, error)
	maxDepth      int
	executeCalled bool
	lastData      map[string]any
}

func newMockTemplateExecutor() *mockTemplateExecutor {
	return &mockTemplateExecutor{
		templates: make(map[string]string),
		maxDepth:  DefaultMaxDepth,
	}
}

func (m *mockTemplateExecutor) ExecuteTemplate(ctx context.Context, name string, data map[string]any) (string, error) {
	m.executeCalled = true
	m.lastData = data
	if m.executeFunc != nil {
		return m.executeFunc(ctx, name, data)
	}
	if result, ok := m.templates[name]; ok {
		return result, nil
	}
	return "", errors.New(ErrMsgTemplateNotFound)
}

func (m *mockTemplateExecutor) HasTemplate(name string) bool {
	_, ok := m.templates[name]
	return ok
}

func (m *mockTemplateExecutor) MaxDepth() int {
	return m.maxDepth
}

func (m *mockTemplateExecutor) GetTemplateSource(name string) (string, bool) {
	source, ok := m.templates[name]
	return source, ok
}

func (m *mockTemplateExecutor) RegisterTemplate(name, result string) {
	m.templates[name] = result
}

// mockTemplateContextAccessor implements TemplateContextAccessor for testing
type mockTemplateContextAccessor struct {
	data   map[string]any
	engine TemplateExecutor
	depth  int
}

func newMockTemplateContextAccessor(data map[string]any) *mockTemplateContextAccessor {
	if data == nil {
		data = make(map[string]any)
	}
	return &mockTemplateContextAccessor{
		data:  data,
		depth: 0,
	}
}

func (m *mockTemplateContextAccessor) Get(path string) (any, bool) {
	val, ok := m.data[path]
	return val, ok
}

func (m *mockTemplateContextAccessor) GetString(path string) string {
	val, ok := m.data[path]
	if !ok {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

func (m *mockTemplateContextAccessor) GetStringDefault(path, defaultVal string) string {
	val := m.GetString(path)
	if val == "" {
		return defaultVal
	}
	return val
}

func (m *mockTemplateContextAccessor) Has(path string) bool {
	_, ok := m.data[path]
	return ok
}

func (m *mockTemplateContextAccessor) Engine() interface{} {
	return m.engine
}

func (m *mockTemplateContextAccessor) Depth() int {
	return m.depth
}

func (m *mockTemplateContextAccessor) WithEngine(engine TemplateExecutor) *mockTemplateContextAccessor {
	m.engine = engine
	return m
}

func (m *mockTemplateContextAccessor) WithDepth(depth int) *mockTemplateContextAccessor {
	m.depth = depth
	return m
}

// TestIncludeResolver_TagName verifies the resolver returns the correct tag name
func TestIncludeResolver_TagName(t *testing.T) {
	resolver := NewIncludeResolver()
	assert.Equal(t, TagNameInclude, resolver.TagName())
}

// TestIncludeResolver_Validate tests attribute validation
func TestIncludeResolver_Validate(t *testing.T) {
	resolver := NewIncludeResolver()

	t.Run("valid attributes with template", func(t *testing.T) {
		err := resolver.Validate(Attributes{AttrTemplate: "mytemplate"})
		assert.NoError(t, err)
	})

	t.Run("valid attributes with extra attrs", func(t *testing.T) {
		err := resolver.Validate(Attributes{
			AttrTemplate: "mytemplate",
			AttrWith:     "context.path",
			AttrIsolate:  AttrValueTrue,
		})
		assert.NoError(t, err)
	})

	t.Run("missing template attribute", func(t *testing.T) {
		err := resolver.Validate(Attributes{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgMissingTemplateAttr)
	})

	t.Run("missing template with other attrs", func(t *testing.T) {
		err := resolver.Validate(Attributes{AttrWith: "path"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgMissingTemplateAttr)
	})
}

// TestIncludeResolver_Resolve tests the resolution logic
func TestIncludeResolver_Resolve(t *testing.T) {
	t.Run("basic include", func(t *testing.T) {
		resolver := NewIncludeResolver()
		engine := newMockTemplateExecutor()
		engine.RegisterTemplate("footer", "Copyright 2024")

		ctx := newMockTemplateContextAccessor(nil).WithEngine(engine)
		attrs := Attributes{AttrTemplate: "footer"}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "Copyright 2024", result)
		assert.True(t, engine.executeCalled)
	})

	t.Run("include with custom attributes passed to child", func(t *testing.T) {
		resolver := NewIncludeResolver()
		engine := newMockTemplateExecutor()
		engine.RegisterTemplate("greeting", "Hello")

		ctx := newMockTemplateContextAccessor(nil).WithEngine(engine)
		attrs := Attributes{
			AttrTemplate: "greeting",
			"user":       "Alice",
			"role":       "admin",
		}

		_, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)

		// Verify custom attributes were passed to child
		assert.Equal(t, "Alice", engine.lastData["user"])
		assert.Equal(t, "admin", engine.lastData["role"])
	})

	t.Run("include with parent depth tracking", func(t *testing.T) {
		resolver := NewIncludeResolver()
		engine := newMockTemplateExecutor()
		engine.RegisterTemplate("child", "Child content")

		ctx := newMockTemplateContextAccessor(nil).WithEngine(engine).WithDepth(2)
		attrs := Attributes{AttrTemplate: "child"}

		_, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)

		// Verify depth was passed
		assert.Equal(t, 2, engine.lastData[MetaKeyParentDepth])
	})

	t.Run("invalid context type - not TemplateContextAccessor", func(t *testing.T) {
		resolver := NewIncludeResolver()
		ctx := newMockContextAccessor(nil) // Not a TemplateContextAccessor
		attrs := Attributes{AttrTemplate: "test"}

		_, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgEngineNotAvailable)
	})

	t.Run("invalid context type - string", func(t *testing.T) {
		resolver := NewIncludeResolver()
		attrs := Attributes{AttrTemplate: "test"}

		_, err := resolver.Resolve(context.Background(), "invalid context", attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgEngineNotAvailable)
	})

	t.Run("missing template attribute", func(t *testing.T) {
		resolver := NewIncludeResolver()
		engine := newMockTemplateExecutor()
		ctx := newMockTemplateContextAccessor(nil).WithEngine(engine)
		attrs := Attributes{}

		_, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgMissingTemplateAttr)
	})

	t.Run("nil engine in context", func(t *testing.T) {
		resolver := NewIncludeResolver()
		ctx := newMockTemplateContextAccessor(nil) // No engine set
		attrs := Attributes{AttrTemplate: "test"}

		_, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgEngineNotAvailable)
	})

	t.Run("template not found", func(t *testing.T) {
		resolver := NewIncludeResolver()
		engine := newMockTemplateExecutor()
		// Don't register any template

		ctx := newMockTemplateContextAccessor(nil).WithEngine(engine)
		attrs := Attributes{AttrTemplate: "nonexistent"}

		_, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "template not found")
		assert.Contains(t, err.Error(), "nonexistent")
	})

	t.Run("depth limit exceeded", func(t *testing.T) {
		resolver := NewIncludeResolver()
		engine := newMockTemplateExecutor()
		engine.maxDepth = 3
		engine.RegisterTemplate("test", "content")

		ctx := newMockTemplateContextAccessor(nil).WithEngine(engine).WithDepth(3)
		attrs := Attributes{AttrTemplate: "test"}

		_, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgDepthExceeded)
	})

	t.Run("depth exactly at limit", func(t *testing.T) {
		resolver := NewIncludeResolver()
		engine := newMockTemplateExecutor()
		engine.maxDepth = 3
		engine.RegisterTemplate("test", "content")

		// Depth 2 is still valid (0, 1, 2 allowed, 3 exceeds)
		ctx := newMockTemplateContextAccessor(nil).WithEngine(engine).WithDepth(2)
		attrs := Attributes{AttrTemplate: "test"}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "content", result)
	})

	t.Run("unlimited depth when maxDepth is 0", func(t *testing.T) {
		resolver := NewIncludeResolver()
		engine := newMockTemplateExecutor()
		engine.maxDepth = 0 // Unlimited
		engine.RegisterTemplate("test", "content")

		ctx := newMockTemplateContextAccessor(nil).WithEngine(engine).WithDepth(100)
		attrs := Attributes{AttrTemplate: "test"}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "content", result)
	})

	t.Run("execution error propagation", func(t *testing.T) {
		resolver := NewIncludeResolver()
		engine := newMockTemplateExecutor()
		engine.RegisterTemplate("test", "")
		engine.executeFunc = func(ctx context.Context, name string, data map[string]any) (string, error) {
			return "", errors.New("execution failed")
		}

		ctx := newMockTemplateContextAccessor(nil).WithEngine(engine)
		attrs := Attributes{AttrTemplate: "test"}

		_, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "execution failed")
	})
}

// TestIncludeResolver_BuildChildData tests the child data building logic
func TestIncludeResolver_BuildChildData(t *testing.T) {
	resolver := NewIncludeResolver()

	t.Run("basic attributes passed through", func(t *testing.T) {
		ctx := newMockTemplateContextAccessor(nil)
		attrs := Attributes{
			AttrTemplate: "test",
			"user":       "Alice",
			"count":      "42",
		}

		data := resolver.buildChildData(ctx, attrs)

		assert.Equal(t, "Alice", data["user"])
		assert.Equal(t, "42", data["count"])
		// Reserved attributes should not be passed
		_, hasTemplate := data[AttrTemplate]
		assert.False(t, hasTemplate)
	})

	t.Run("reserved attributes excluded", func(t *testing.T) {
		ctx := newMockTemplateContextAccessor(nil)
		attrs := Attributes{
			AttrTemplate: "test",
			AttrWith:     "path",
			AttrIsolate:  AttrValueTrue,
			"custom":     "value",
		}

		data := resolver.buildChildData(ctx, attrs)

		assert.Equal(t, "value", data["custom"])
		_, hasTemplate := data[AttrTemplate]
		assert.False(t, hasTemplate)
		_, hasWith := data[AttrWith]
		assert.False(t, hasWith)
		_, hasIsolate := data[AttrIsolate]
		assert.False(t, hasIsolate)
	})

	t.Run("with attribute - map value", func(t *testing.T) {
		ctx := newMockTemplateContextAccessor(map[string]any{
			"user": map[string]any{
				"name": "Alice",
				"age":  30,
			},
		})
		attrs := Attributes{
			AttrTemplate: "test",
			AttrWith:     "user",
		}

		data := resolver.buildChildData(ctx, attrs)

		assert.Equal(t, "Alice", data["name"])
		assert.Equal(t, 30, data["age"])
	})

	t.Run("with attribute - non-map value", func(t *testing.T) {
		ctx := newMockTemplateContextAccessor(map[string]any{
			"username": "Alice",
		})
		attrs := Attributes{
			AttrTemplate: "test",
			AttrWith:     "username",
		}

		data := resolver.buildChildData(ctx, attrs)

		assert.Equal(t, "Alice", data[MetaKeyValue])
	})

	t.Run("with attribute - path not found", func(t *testing.T) {
		ctx := newMockTemplateContextAccessor(nil)
		attrs := Attributes{
			AttrTemplate: "test",
			AttrWith:     "nonexistent",
		}

		data := resolver.buildChildData(ctx, attrs)

		// Should still work, just no data from with
		_, hasValue := data[MetaKeyValue]
		assert.False(t, hasValue)
	})

	t.Run("isolate mode ignores with", func(t *testing.T) {
		ctx := newMockTemplateContextAccessor(map[string]any{
			"user": map[string]any{
				"name": "Alice",
			},
		})
		attrs := Attributes{
			AttrTemplate: "test",
			AttrWith:     "user",
			AttrIsolate:  AttrValueTrue,
		}

		data := resolver.buildChildData(ctx, attrs)

		// With attribute should be ignored when isolate is true
		_, hasName := data["name"]
		assert.False(t, hasName)
	})

	t.Run("depth tracking included", func(t *testing.T) {
		ctx := newMockTemplateContextAccessor(nil).WithDepth(5)
		attrs := Attributes{AttrTemplate: "test"}

		data := resolver.buildChildData(ctx, attrs)

		assert.Equal(t, 5, data[MetaKeyParentDepth])
	})
}

// TestIncludeResolver_EngineInterfaceCheck tests engine interface validation
func TestIncludeResolver_EngineInterfaceCheck(t *testing.T) {
	t.Run("engine not implementing TemplateExecutor", func(t *testing.T) {
		resolver := NewIncludeResolver()

		// Create a mock that returns something that isn't a TemplateExecutor
		ctx := &mockTemplateContextAccessorWithInvalidEngine{
			data:   make(map[string]any),
			engine: "not a template executor",
		}
		attrs := Attributes{AttrTemplate: "test"}

		_, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgEngineNotAvailable)
	})
}

// mockTemplateContextAccessorWithInvalidEngine for testing invalid engine type
type mockTemplateContextAccessorWithInvalidEngine struct {
	data   map[string]any
	engine interface{}
}

func (m *mockTemplateContextAccessorWithInvalidEngine) Get(path string) (any, bool) {
	val, ok := m.data[path]
	return val, ok
}

func (m *mockTemplateContextAccessorWithInvalidEngine) GetString(path string) string {
	return ""
}

func (m *mockTemplateContextAccessorWithInvalidEngine) GetStringDefault(path, defaultVal string) string {
	return defaultVal
}

func (m *mockTemplateContextAccessorWithInvalidEngine) Has(path string) bool {
	_, ok := m.data[path]
	return ok
}

func (m *mockTemplateContextAccessorWithInvalidEngine) Engine() interface{} {
	return m.engine
}

func (m *mockTemplateContextAccessorWithInvalidEngine) Depth() int {
	return 0
}
