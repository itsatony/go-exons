package internal

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockContextAccessor implements ContextAccessor for testing
type mockContextAccessor struct {
	data map[string]any
}

func newMockContextAccessor(data map[string]any) *mockContextAccessor {
	if data == nil {
		data = make(map[string]any)
	}
	return &mockContextAccessor{data: data}
}

func (m *mockContextAccessor) Get(path string) (any, bool) {
	val, ok := m.data[path]
	return val, ok
}

func (m *mockContextAccessor) GetString(path string) string {
	val, ok := m.data[path]
	if !ok {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

func (m *mockContextAccessor) GetStringDefault(path, defaultVal string) string {
	val := m.GetString(path)
	if val == "" {
		return defaultVal
	}
	return val
}

func (m *mockContextAccessor) Has(path string) bool {
	_, ok := m.data[path]
	return ok
}

func TestVarResolver_TagName(t *testing.T) {
	resolver := NewVarResolver()
	assert.Equal(t, TagNameVar, resolver.TagName())
}

func TestVarResolver_Resolve(t *testing.T) {
	t.Run("string variable", func(t *testing.T) {
		resolver := NewVarResolver()
		ctx := newMockContextAccessor(map[string]any{
			"username": "Alice",
		})
		attrs := Attributes{"name": "username"}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "Alice", result)
	})

	t.Run("integer variable", func(t *testing.T) {
		resolver := NewVarResolver()
		ctx := newMockContextAccessor(map[string]any{
			"age": 30,
		})
		attrs := Attributes{"name": "age"}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "30", result)
	})

	t.Run("int64 variable", func(t *testing.T) {
		resolver := NewVarResolver()
		ctx := newMockContextAccessor(map[string]any{
			"bignum": int64(9223372036854775807),
		})
		attrs := Attributes{"name": "bignum"}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "9223372036854775807", result)
	})

	t.Run("float variable", func(t *testing.T) {
		resolver := NewVarResolver()
		ctx := newMockContextAccessor(map[string]any{
			"price": 19.99,
		})
		attrs := Attributes{"name": "price"}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "19.99", result)
	})

	t.Run("boolean variable", func(t *testing.T) {
		resolver := NewVarResolver()
		ctx := newMockContextAccessor(map[string]any{
			"active": true,
		})
		attrs := Attributes{"name": "active"}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "true", result)
	})

	t.Run("nil variable", func(t *testing.T) {
		resolver := NewVarResolver()
		ctx := newMockContextAccessor(map[string]any{
			"empty": nil,
		})
		attrs := Attributes{"name": "empty"}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("stringer variable", func(t *testing.T) {
		resolver := NewVarResolver()
		ctx := newMockContextAccessor(map[string]any{
			"custom": customStringer{"test"},
		})
		attrs := Attributes{"name": "custom"}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "custom:test", result)
	})

	t.Run("fallback fmt.Sprintf", func(t *testing.T) {
		resolver := NewVarResolver()
		ctx := newMockContextAccessor(map[string]any{
			"slice": []string{"a", "b"},
		})
		attrs := Attributes{"name": "slice"}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "[a b]", result)
	})

	t.Run("missing variable with default", func(t *testing.T) {
		resolver := NewVarResolver()
		ctx := newMockContextAccessor(nil)
		attrs := Attributes{
			"name":    "missing",
			"default": "fallback",
		}

		result, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.NoError(t, err)
		assert.Equal(t, "fallback", result)
	})

	t.Run("missing variable without default", func(t *testing.T) {
		resolver := NewVarResolver()
		ctx := newMockContextAccessor(nil)
		attrs := Attributes{"name": "missing"}

		_, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("missing name attribute", func(t *testing.T) {
		resolver := NewVarResolver()
		ctx := newMockContextAccessor(nil)
		attrs := Attributes{}

		_, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgMissingNameAttr)
	})

	t.Run("invalid context type", func(t *testing.T) {
		resolver := NewVarResolver()
		attrs := Attributes{"name": "test"}

		_, err := resolver.Resolve(context.Background(), "not a context", attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgInvalidContext)
	})
}

func TestVarResolver_Validate(t *testing.T) {
	resolver := NewVarResolver()

	t.Run("valid attributes", func(t *testing.T) {
		err := resolver.Validate(Attributes{"name": "test"})
		assert.NoError(t, err)
	})

	t.Run("missing name attribute", func(t *testing.T) {
		err := resolver.Validate(Attributes{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgMissingNameAttr)
	})
}

func TestRawResolver_TagName(t *testing.T) {
	resolver := NewRawResolver()
	assert.Equal(t, TagNameRaw, resolver.TagName())
}

func TestRawResolver_Resolve(t *testing.T) {
	resolver := NewRawResolver()

	// Resolve should return an error since raw blocks should be
	// handled specially by the executor
	_, err := resolver.Resolve(context.Background(), nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgRawResolverCalled)
}

func TestRawResolver_Validate(t *testing.T) {
	resolver := NewRawResolver()

	// Should always return nil
	err := resolver.Validate(nil)
	assert.NoError(t, err)

	err = resolver.Validate(Attributes{"random": "attr"})
	assert.NoError(t, err)
}

func TestRegisterBuiltins(t *testing.T) {
	registry := NewRegistry(nil)

	RegisterBuiltins(registry)

	// Verify all built-ins are registered
	assert.True(t, registry.Has(TagNameVar))
	assert.True(t, registry.Has(TagNameRaw))
	assert.True(t, registry.Has(TagNameInclude))
	assert.True(t, registry.Has(TagNameEnv))
	assert.True(t, registry.Has(TagNameMessage))
	assert.True(t, registry.Has(TagNameRef))
	assert.True(t, registry.Has(TagNameSkillsCatalog))
	assert.True(t, registry.Has(TagNameToolsCatalog))
	assert.Equal(t, 8, registry.Count())

	// Verify we can get them
	varResolver, ok := registry.Get(TagNameVar)
	require.True(t, ok)
	assert.Equal(t, TagNameVar, varResolver.TagName())

	rawResolver, ok := registry.Get(TagNameRaw)
	require.True(t, ok)
	assert.Equal(t, TagNameRaw, rawResolver.TagName())

	includeResolver, ok := registry.Get(TagNameInclude)
	require.True(t, ok)
	assert.Equal(t, TagNameInclude, includeResolver.TagName())
}

func TestBuiltinError_Error(t *testing.T) {
	err := NewBuiltinError("test message", "test.tag")
	assert.Equal(t, "test.tag: test message", err.Error())
}

// customStringer implements fmt.Stringer for testing
type customStringer struct {
	value string
}

func (c customStringer) String() string {
	return fmt.Sprintf("custom:%s", c.value)
}

// mockContextAccessorWithKeys implements ContextAccessor and KeyLister for testing
type mockContextAccessorWithKeys struct {
	data map[string]any
}

func newMockContextAccessorWithKeys(data map[string]any) *mockContextAccessorWithKeys {
	if data == nil {
		data = make(map[string]any)
	}
	return &mockContextAccessorWithKeys{data: data}
}

func (m *mockContextAccessorWithKeys) Get(path string) (any, bool) {
	val, ok := m.data[path]
	return val, ok
}

func (m *mockContextAccessorWithKeys) GetString(path string) string {
	val, ok := m.data[path]
	if !ok {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

func (m *mockContextAccessorWithKeys) GetStringDefault(path, defaultVal string) string {
	val := m.GetString(path)
	if val == "" {
		return defaultVal
	}
	return val
}

func (m *mockContextAccessorWithKeys) Has(path string) bool {
	_, ok := m.data[path]
	return ok
}

func (m *mockContextAccessorWithKeys) Keys() []string {
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys
}

func TestVarResolver_Resolve_WithSuggestions(t *testing.T) {
	t.Run("shows 'did you mean' when similar key exists", func(t *testing.T) {
		resolver := NewVarResolver()
		ctx := newMockContextAccessorWithKeys(map[string]any{
			"name":  "Alice",
			"email": "alice@example.com",
		})
		attrs := Attributes{"name": "nam"} // similar to "name" (distance 1, within threshold)

		_, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Did you mean")
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("shows available keys when no similar keys found", func(t *testing.T) {
		resolver := NewVarResolver()
		ctx := newMockContextAccessorWithKeys(map[string]any{
			"name":  "Alice",
			"email": "alice@example.com",
			"age":   30,
		})
		attrs := Attributes{"name": "xyz_completely_different"}

		_, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Available keys")
	})

	t.Run("handles empty context gracefully", func(t *testing.T) {
		resolver := NewVarResolver()
		ctx := newMockContextAccessorWithKeys(map[string]any{})
		attrs := Attributes{"name": "missing"}

		_, err := resolver.Resolve(context.Background(), ctx, attrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
		// Should not contain suggestions or available keys since context is empty
		assert.NotContains(t, err.Error(), "Did you mean")
		assert.NotContains(t, err.Error(), "Available keys")
	})
}

func TestNewVariableNotFoundWithAvailableKeysError(t *testing.T) {
	t.Run("formats available keys correctly", func(t *testing.T) {
		err := NewVariableNotFoundWithAvailableKeysError("missing", []string{"name", "email", "age"}, false)
		errStr := err.Error()
		assert.Contains(t, errStr, "variable not found")
		assert.Contains(t, errStr, "Available keys")
		assert.Contains(t, errStr, "'name'")
		assert.Contains(t, errStr, "'email'")
		assert.Contains(t, errStr, "'age'")
		assert.Contains(t, errStr, "[path=missing]")
	})

	t.Run("handles empty keys", func(t *testing.T) {
		err := NewVariableNotFoundWithAvailableKeysError("missing", []string{}, false)
		errStr := err.Error()
		assert.Contains(t, errStr, "variable not found")
		assert.NotContains(t, errStr, "Available keys")
	})

	t.Run("truncates long key lists", func(t *testing.T) {
		keys := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
		err := NewVariableNotFoundWithAvailableKeysError("missing", keys, false)
		errStr := err.Error()
		assert.Contains(t, errStr, "Available keys")
		assert.Contains(t, errStr, "(3 more)")
	})
}
