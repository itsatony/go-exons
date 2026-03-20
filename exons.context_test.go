package exons

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// NewContext
// =============================================================================

func TestNewContext(t *testing.T) {
	t.Run("with data", func(t *testing.T) {
		ctx := NewContext(map[string]any{"key": "value"})
		assert.NotNil(t, ctx)
		val, ok := ctx.Get("key")
		assert.True(t, ok)
		assert.Equal(t, "value", val)
	})

	t.Run("with nil data", func(t *testing.T) {
		ctx := NewContext(nil)
		assert.NotNil(t, ctx)
		val, ok := ctx.Get("key")
		assert.False(t, ok)
		assert.Nil(t, val)
	})

	t.Run("default error strategy is throw", func(t *testing.T) {
		ctx := NewContext(nil)
		assert.Equal(t, ErrorStrategyThrow, ctx.ErrorStrategyValue())
	})
}

func TestNewContextWithStrategy(t *testing.T) {
	ctx := NewContextWithStrategy(map[string]any{"a": 1}, ErrorStrategyRemove)
	assert.Equal(t, ErrorStrategyRemove, ctx.ErrorStrategyValue())
	assert.Equal(t, int(ErrorStrategyRemove), ctx.ErrorStrategy())
	val, ok := ctx.Get("a")
	assert.True(t, ok)
	assert.Equal(t, 1, val)
}

// =============================================================================
// Get / Set / Has
// =============================================================================

func TestContext_GetSetHas(t *testing.T) {
	ctx := NewContext(map[string]any{"existing": "value"})

	t.Run("get existing key", func(t *testing.T) {
		val, ok := ctx.Get("existing")
		assert.True(t, ok)
		assert.Equal(t, "value", val)
	})

	t.Run("get missing key", func(t *testing.T) {
		val, ok := ctx.Get("missing")
		assert.False(t, ok)
		assert.Nil(t, val)
	})

	t.Run("has existing key", func(t *testing.T) {
		assert.True(t, ctx.Has("existing"))
	})

	t.Run("has missing key", func(t *testing.T) {
		assert.False(t, ctx.Has("missing"))
	})

	t.Run("set key", func(t *testing.T) {
		ctx.Set("new_key", "new_value")
		val, ok := ctx.Get("new_key")
		assert.True(t, ok)
		assert.Equal(t, "new_value", val)
	})

	t.Run("set overrides existing key", func(t *testing.T) {
		ctx.Set("existing", "updated")
		val, ok := ctx.Get("existing")
		assert.True(t, ok)
		assert.Equal(t, "updated", val)
	})

	t.Run("get empty path", func(t *testing.T) {
		val, ok := ctx.Get("")
		assert.False(t, ok)
		assert.Nil(t, val)
	})
}

// =============================================================================
// GetString
// =============================================================================

func TestContext_GetString(t *testing.T) {
	ctx := NewContext(map[string]any{
		"str": "hello",
		"num": 42,
	})

	assert.Equal(t, "hello", ctx.GetString("str"))
	assert.Equal(t, "", ctx.GetString("num")) // not a string
	assert.Equal(t, "", ctx.GetString("missing"))
}

// =============================================================================
// GetDefault
// =============================================================================

func TestContext_GetDefault(t *testing.T) {
	ctx := NewContext(map[string]any{"key": "value"})

	assert.Equal(t, "value", ctx.GetDefault("key", "fallback"))
	assert.Equal(t, "fallback", ctx.GetDefault("missing", "fallback"))
}

// =============================================================================
// GetStringDefault
// =============================================================================

func TestContext_GetStringDefault(t *testing.T) {
	ctx := NewContext(map[string]any{
		"str": "hello",
		"num": 42,
	})

	assert.Equal(t, "hello", ctx.GetStringDefault("str", "default"))
	assert.Equal(t, "default", ctx.GetStringDefault("missing", "default"))
	assert.Equal(t, "default", ctx.GetStringDefault("num", "default")) // not a string
}

// =============================================================================
// GetInt, GetIntDefault
// =============================================================================

func TestContext_GetInt(t *testing.T) {
	ctx := NewContext(map[string]any{
		"int":     42,
		"int64":   int64(100),
		"int32":   int32(50),
		"float64": float64(3.14),
		"float32": float32(2.5),
		"str":     "not a number",
		"uint":    uint(10),
	})

	t.Run("int", func(t *testing.T) {
		val, ok := ctx.GetInt("int")
		assert.True(t, ok)
		assert.Equal(t, 42, val)
	})

	t.Run("int64", func(t *testing.T) {
		val, ok := ctx.GetInt("int64")
		assert.True(t, ok)
		assert.Equal(t, 100, val)
	})

	t.Run("int32", func(t *testing.T) {
		val, ok := ctx.GetInt("int32")
		assert.True(t, ok)
		assert.Equal(t, 50, val)
	})

	t.Run("float64 truncated", func(t *testing.T) {
		val, ok := ctx.GetInt("float64")
		assert.True(t, ok)
		assert.Equal(t, 3, val)
	})

	t.Run("float32 truncated", func(t *testing.T) {
		val, ok := ctx.GetInt("float32")
		assert.True(t, ok)
		assert.Equal(t, 2, val)
	})

	t.Run("string is not int", func(t *testing.T) {
		_, ok := ctx.GetInt("str")
		assert.False(t, ok)
	})

	t.Run("uint via reflection", func(t *testing.T) {
		val, ok := ctx.GetInt("uint")
		assert.True(t, ok)
		assert.Equal(t, 10, val)
	})

	t.Run("missing key", func(t *testing.T) {
		val, ok := ctx.GetInt("missing")
		assert.False(t, ok)
		assert.Equal(t, 0, val)
	})
}

func TestContext_GetIntDefault(t *testing.T) {
	ctx := NewContext(map[string]any{"count": 5})

	assert.Equal(t, 5, ctx.GetIntDefault("count", 0))
	assert.Equal(t, 99, ctx.GetIntDefault("missing", 99))
}

// =============================================================================
// GetFloat, GetFloatDefault
// =============================================================================

func TestContext_GetFloat(t *testing.T) {
	ctx := NewContext(map[string]any{
		"f64": float64(3.14),
		"f32": float32(2.5),
		"int": 42,
		"i64": int64(100),
		"str": "not a number",
	})

	t.Run("float64", func(t *testing.T) {
		val, ok := ctx.GetFloat("f64")
		assert.True(t, ok)
		assert.InDelta(t, 3.14, val, 0.001)
	})

	t.Run("float32", func(t *testing.T) {
		val, ok := ctx.GetFloat("f32")
		assert.True(t, ok)
		assert.InDelta(t, 2.5, val, 0.001)
	})

	t.Run("int converted", func(t *testing.T) {
		val, ok := ctx.GetFloat("int")
		assert.True(t, ok)
		assert.Equal(t, float64(42), val)
	})

	t.Run("int64 converted", func(t *testing.T) {
		val, ok := ctx.GetFloat("i64")
		assert.True(t, ok)
		assert.Equal(t, float64(100), val)
	})

	t.Run("string fails", func(t *testing.T) {
		_, ok := ctx.GetFloat("str")
		assert.False(t, ok)
	})

	t.Run("missing", func(t *testing.T) {
		val, ok := ctx.GetFloat("missing")
		assert.False(t, ok)
		assert.Equal(t, float64(0), val)
	})
}

func TestContext_GetFloatDefault(t *testing.T) {
	ctx := NewContext(map[string]any{"pi": 3.14})
	assert.InDelta(t, 3.14, ctx.GetFloatDefault("pi", 0), 0.001)
	assert.InDelta(t, 9.99, ctx.GetFloatDefault("missing", 9.99), 0.001)
}

// =============================================================================
// GetBool, GetBoolDefault
// =============================================================================

func TestContext_GetBool(t *testing.T) {
	ctx := NewContext(map[string]any{
		"yes": true,
		"no":  false,
		"str": "true",
	})

	t.Run("true", func(t *testing.T) {
		val, ok := ctx.GetBool("yes")
		assert.True(t, ok)
		assert.True(t, val)
	})

	t.Run("false", func(t *testing.T) {
		val, ok := ctx.GetBool("no")
		assert.True(t, ok)
		assert.False(t, val)
	})

	t.Run("string is not bool", func(t *testing.T) {
		_, ok := ctx.GetBool("str")
		assert.False(t, ok)
	})

	t.Run("missing", func(t *testing.T) {
		val, ok := ctx.GetBool("missing")
		assert.False(t, ok)
		assert.False(t, val)
	})
}

func TestContext_GetBoolDefault(t *testing.T) {
	ctx := NewContext(map[string]any{"active": true})
	assert.True(t, ctx.GetBoolDefault("active", false))
	assert.True(t, ctx.GetBoolDefault("missing", true))
	assert.False(t, ctx.GetBoolDefault("missing", false))
}

// =============================================================================
// GetSlice, GetSliceDefault
// =============================================================================

func TestContext_GetSlice(t *testing.T) {
	ctx := NewContext(map[string]any{
		"any_slice":    []any{"a", "b"},
		"string_slice": []string{"x", "y"},
		"not_slice":    "hello",
	})

	t.Run("any slice", func(t *testing.T) {
		val, ok := ctx.GetSlice("any_slice")
		assert.True(t, ok)
		assert.Len(t, val, 2)
	})

	t.Run("string slice converted via reflection", func(t *testing.T) {
		val, ok := ctx.GetSlice("string_slice")
		assert.True(t, ok)
		assert.Len(t, val, 2)
	})

	t.Run("not a slice", func(t *testing.T) {
		_, ok := ctx.GetSlice("not_slice")
		assert.False(t, ok)
	})

	t.Run("missing", func(t *testing.T) {
		_, ok := ctx.GetSlice("missing")
		assert.False(t, ok)
	})
}

func TestContext_GetSliceDefault(t *testing.T) {
	ctx := NewContext(map[string]any{"items": []any{"a"}})
	def := []any{"default"}
	assert.Len(t, ctx.GetSliceDefault("items", def), 1)
	assert.Equal(t, def, ctx.GetSliceDefault("missing", def))
}

// =============================================================================
// GetMap, GetMapDefault
// =============================================================================

func TestContext_GetMap(t *testing.T) {
	ctx := NewContext(map[string]any{
		"any_map":    map[string]any{"a": 1},
		"string_map": map[string]string{"b": "2"},
		"not_map":    "hello",
	})

	t.Run("any map", func(t *testing.T) {
		val, ok := ctx.GetMap("any_map")
		assert.True(t, ok)
		assert.Equal(t, 1, val["a"])
	})

	t.Run("string map converted", func(t *testing.T) {
		val, ok := ctx.GetMap("string_map")
		assert.True(t, ok)
		assert.Equal(t, "2", val["b"])
	})

	t.Run("not a map", func(t *testing.T) {
		_, ok := ctx.GetMap("not_map")
		assert.False(t, ok)
	})

	t.Run("missing", func(t *testing.T) {
		_, ok := ctx.GetMap("missing")
		assert.False(t, ok)
	})
}

func TestContext_GetMapDefault(t *testing.T) {
	ctx := NewContext(map[string]any{"m": map[string]any{"k": "v"}})
	def := map[string]any{"default": true}
	assert.Equal(t, "v", ctx.GetMapDefault("m", def)["k"])
	assert.Equal(t, def, ctx.GetMapDefault("missing", def))
}

// =============================================================================
// GetStringSlice, GetStringSliceDefault
// =============================================================================

func TestContext_GetStringSlice(t *testing.T) {
	ctx := NewContext(map[string]any{
		"str_slice": []string{"a", "b"},
		"any_slice": []any{"x", "y"},
		"mixed_any": []any{"a", 1},
		"not_slice": "hello",
	})

	t.Run("string slice", func(t *testing.T) {
		val, ok := ctx.GetStringSlice("str_slice")
		assert.True(t, ok)
		assert.Equal(t, []string{"a", "b"}, val)
	})

	t.Run("any slice of strings", func(t *testing.T) {
		val, ok := ctx.GetStringSlice("any_slice")
		assert.True(t, ok)
		assert.Equal(t, []string{"x", "y"}, val)
	})

	t.Run("mixed any slice fails", func(t *testing.T) {
		_, ok := ctx.GetStringSlice("mixed_any")
		assert.False(t, ok)
	})

	t.Run("not a slice", func(t *testing.T) {
		_, ok := ctx.GetStringSlice("not_slice")
		assert.False(t, ok)
	})

	t.Run("missing", func(t *testing.T) {
		_, ok := ctx.GetStringSlice("missing")
		assert.False(t, ok)
	})
}

func TestContext_GetStringSliceDefault(t *testing.T) {
	ctx := NewContext(map[string]any{"tags": []string{"go", "rust"}})
	def := []string{"default"}
	assert.Equal(t, []string{"go", "rust"}, ctx.GetStringSliceDefault("tags", def))
	assert.Equal(t, def, ctx.GetStringSliceDefault("missing", def))
}

// =============================================================================
// Dot-Notation Paths
// =============================================================================

func TestContext_DotNotation(t *testing.T) {
	ctx := NewContext(map[string]any{
		"user": map[string]any{
			"profile": map[string]any{
				"name": "Alice",
				"age":  30,
			},
		},
		"headers": map[string]string{
			"content-type": "application/json",
		},
	})

	t.Run("nested map[string]any", func(t *testing.T) {
		val, ok := ctx.Get("user.profile.name")
		assert.True(t, ok)
		assert.Equal(t, "Alice", val)
	})

	t.Run("deep nested int", func(t *testing.T) {
		val, ok := ctx.GetInt("user.profile.age")
		assert.True(t, ok)
		assert.Equal(t, 30, val)
	})

	t.Run("map[string]string path", func(t *testing.T) {
		val, ok := ctx.Get("headers.content-type")
		assert.True(t, ok)
		assert.Equal(t, "application/json", val)
	})

	t.Run("non-existent nested path", func(t *testing.T) {
		_, ok := ctx.Get("user.profile.email")
		assert.False(t, ok)
	})

	t.Run("path through non-map", func(t *testing.T) {
		_, ok := ctx.Get("user.profile.name.extra")
		assert.False(t, ok)
	})
}

// =============================================================================
// Parent-Child Inheritance
// =============================================================================

func TestContext_ParentChild(t *testing.T) {
	parent := NewContext(map[string]any{
		"inherited": "parent_value",
		"shared":    "parent_shared",
	})

	childCtx := parent.Child(map[string]any{
		"shared": "child_shared",
		"own":    "child_own",
	}).(*Context)

	t.Run("child sees own data", func(t *testing.T) {
		val, ok := childCtx.Get("own")
		assert.True(t, ok)
		assert.Equal(t, "child_own", val)
	})

	t.Run("child sees parent data", func(t *testing.T) {
		val, ok := childCtx.Get("inherited")
		assert.True(t, ok)
		assert.Equal(t, "parent_value", val)
	})

	t.Run("child shadows parent data", func(t *testing.T) {
		val, ok := childCtx.Get("shared")
		assert.True(t, ok)
		assert.Equal(t, "child_shared", val)
	})

	t.Run("parent unchanged", func(t *testing.T) {
		val, ok := parent.Get("shared")
		assert.True(t, ok)
		assert.Equal(t, "parent_shared", val)
	})

	t.Run("parent is accessible", func(t *testing.T) {
		assert.Equal(t, parent, childCtx.Parent())
	})

	t.Run("root has no parent", func(t *testing.T) {
		assert.Nil(t, parent.Parent())
	})

	t.Run("child with nil data", func(t *testing.T) {
		c := parent.Child(nil).(*Context)
		assert.NotNil(t, c)
		// Should still see parent data
		val, ok := c.Get("inherited")
		assert.True(t, ok)
		assert.Equal(t, "parent_value", val)
	})

	t.Run("child inherits error strategy", func(t *testing.T) {
		parentWithStrat := NewContextWithStrategy(nil, ErrorStrategyRemove)
		child := parentWithStrat.Child(nil).(*Context)
		assert.Equal(t, ErrorStrategyRemove, child.ErrorStrategyValue())
	})
}

// =============================================================================
// Deep Copy Isolation
// =============================================================================

func TestContext_DeepCopy(t *testing.T) {
	t.Run("WithEngine creates isolated copy", func(t *testing.T) {
		ctx := NewContext(map[string]any{
			"nested": map[string]any{"key": "original"},
		})
		newCtx := ctx.WithEngine(nil)

		// Mutate original
		ctx.Set("new_key", "value")
		// New context should not see mutation
		_, ok := newCtx.Get("new_key")
		assert.False(t, ok)
	})

	t.Run("WithDepth creates isolated copy", func(t *testing.T) {
		ctx := NewContext(map[string]any{"a": "b"})
		newCtx := ctx.WithDepth(5)
		assert.Equal(t, 5, newCtx.Depth())

		ctx.Set("c", "d")
		_, ok := newCtx.Get("c")
		assert.False(t, ok)
	})

	t.Run("Data returns deep copy", func(t *testing.T) {
		original := map[string]any{
			"nested": map[string]any{"inner": "value"},
		}
		ctx := NewContext(original)
		data := ctx.Data()

		// Modify the copy
		data["extra"] = "added"
		nestedCopy, _ := data["nested"].(map[string]any)
		nestedCopy["inner"] = "modified"

		// Original should be unchanged
		val, ok := ctx.Get("nested.inner")
		assert.True(t, ok)
		assert.Equal(t, "value", val)
		_, ok = ctx.Get("extra")
		assert.False(t, ok)
	})
}

// =============================================================================
// WithEngine, WithDepth, WithSpecResolver, WithRefDepth, WithRefChain
// =============================================================================

func TestContext_WithMethods(t *testing.T) {
	t.Run("WithEngine", func(t *testing.T) {
		ctx := NewContext(nil)
		assert.Nil(t, ctx.Engine())

		engine := MustNew()
		newCtx := ctx.WithEngine(engine)
		assert.Equal(t, engine, newCtx.Engine())
		// Original unchanged
		assert.Nil(t, ctx.Engine())
	})

	t.Run("WithDepth", func(t *testing.T) {
		ctx := NewContext(nil)
		assert.Equal(t, 0, ctx.Depth())

		newCtx := ctx.WithDepth(3)
		assert.Equal(t, 3, newCtx.Depth())
		assert.Equal(t, 0, ctx.Depth())
	})

	t.Run("WithSpecResolver", func(t *testing.T) {
		ctx := NewContext(nil)
		assert.Nil(t, ctx.SpecResolver())

		mockResolver := &mockBodyResolver{}
		newCtx := ctx.WithSpecResolver(mockResolver)
		assert.Equal(t, mockResolver, newCtx.SpecResolver())
		assert.Nil(t, ctx.SpecResolver())
	})

	t.Run("WithRefDepth", func(t *testing.T) {
		ctx := NewContext(nil)
		assert.Equal(t, 0, ctx.RefDepth())

		newCtx := ctx.WithRefDepth(5)
		assert.Equal(t, 5, newCtx.RefDepth())
		assert.Equal(t, 0, ctx.RefDepth())
	})

	t.Run("WithRefChain", func(t *testing.T) {
		ctx := NewContext(nil)
		assert.Nil(t, ctx.RefChain())

		chain := []string{"a", "b", "c"}
		newCtx := ctx.WithRefChain(chain)
		assert.Equal(t, chain, newCtx.RefChain())
		assert.Nil(t, ctx.RefChain())

		// Verify deep copy
		chain[0] = "modified"
		assert.Equal(t, "a", newCtx.RefChain()[0])
	})

	t.Run("WithRefChain nil", func(t *testing.T) {
		ctx := NewContext(nil)
		newCtx := ctx.WithRefChain(nil)
		assert.Nil(t, newCtx.RefChain())
	})
}

// =============================================================================
// Keys(), AllKeys()
// =============================================================================

func TestContext_Keys(t *testing.T) {
	ctx := NewContext(map[string]any{
		"a": 1,
		"b": 2,
	})

	keys := ctx.Keys()
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, "a")
	assert.Contains(t, keys, "b")
}

func TestContext_AllKeys(t *testing.T) {
	parent := NewContext(map[string]any{
		"parent_key": 1,
		"shared":     2,
	})
	child := parent.Child(map[string]any{
		"child_key": 3,
		"shared":    4,
	}).(*Context)

	t.Run("child keys only", func(t *testing.T) {
		keys := child.Keys()
		assert.Contains(t, keys, "child_key")
		assert.Contains(t, keys, "shared")
		assert.NotContains(t, keys, "parent_key")
	})

	t.Run("all keys includes parent", func(t *testing.T) {
		allKeys := child.AllKeys()
		assert.Contains(t, allKeys, "child_key")
		assert.Contains(t, allKeys, "shared")
		assert.Contains(t, allKeys, "parent_key")
	})
}

// =============================================================================
// ErrorStrategy, ErrorStrategyValue
// =============================================================================

func TestContext_ErrorStrategyMethods(t *testing.T) {
	ctx := NewContextWithStrategy(nil, ErrorStrategyKeepRaw)
	assert.Equal(t, int(ErrorStrategyKeepRaw), ctx.ErrorStrategy())
	assert.Equal(t, ErrorStrategyKeepRaw, ctx.ErrorStrategyValue())
}

// =============================================================================
// Thread Safety
// =============================================================================

func TestContext_ThreadSafety(t *testing.T) {
	ctx := NewContext(map[string]any{"counter": 0})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			ctx.Set("counter", n)
		}(i)
		go func() {
			defer wg.Done()
			_, _ = ctx.Get("counter")
			_ = ctx.GetString("counter")
			_ = ctx.Has("counter")
			_ = ctx.Keys()
			_ = ctx.AllKeys()
			_ = ctx.Data()
		}()
	}
	wg.Wait()
}

// =============================================================================
// Deep Copy Functions
// =============================================================================

func TestDeepCopyValue(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, deepCopyValue(nil))
	})

	t.Run("string", func(t *testing.T) {
		assert.Equal(t, "hello", deepCopyValue("hello"))
	})

	t.Run("int", func(t *testing.T) {
		assert.Equal(t, 42, deepCopyValue(42))
	})

	t.Run("bool", func(t *testing.T) {
		assert.Equal(t, true, deepCopyValue(true))
	})

	t.Run("map[string]any", func(t *testing.T) {
		m := map[string]any{"a": "b"}
		copied := deepCopyValue(m).(map[string]any)
		assert.Equal(t, "b", copied["a"])
		m["a"] = "changed"
		assert.Equal(t, "b", copied["a"])
	})

	t.Run("map[string]string", func(t *testing.T) {
		m := map[string]string{"a": "b"}
		copied := deepCopyValue(m).(map[string]string)
		assert.Equal(t, "b", copied["a"])
		m["a"] = "changed"
		assert.Equal(t, "b", copied["a"])
	})

	t.Run("[]any", func(t *testing.T) {
		s := []any{"a", "b"}
		copied := deepCopyValue(s).([]any)
		assert.Equal(t, "a", copied[0])
		s[0] = "changed"
		assert.Equal(t, "a", copied[0])
	})

	t.Run("[]string", func(t *testing.T) {
		s := []string{"x", "y"}
		copied := deepCopyValue(s).([]string)
		assert.Equal(t, "x", copied[0])
		s[0] = "changed"
		assert.Equal(t, "x", copied[0])
	})

	t.Run("[]int", func(t *testing.T) {
		s := []int{1, 2}
		copied := deepCopyValue(s).([]int)
		assert.Equal(t, 1, copied[0])
		s[0] = 99
		assert.Equal(t, 1, copied[0])
	})

	t.Run("[]float64", func(t *testing.T) {
		s := []float64{1.1, 2.2}
		copied := deepCopyValue(s).([]float64)
		assert.InDelta(t, 1.1, copied[0], 0.001)
	})

	t.Run("[]bool", func(t *testing.T) {
		s := []bool{true, false}
		copied := deepCopyValue(s).([]bool)
		assert.True(t, copied[0])
	})
}

func TestDeepCopyMap(t *testing.T) {
	t.Run("nil map", func(t *testing.T) {
		assert.Nil(t, deepCopyMap(nil))
	})

	t.Run("nested map", func(t *testing.T) {
		m := map[string]any{
			"a": map[string]any{
				"b": "value",
			},
		}
		copied := deepCopyMap(m)
		inner := m["a"].(map[string]any)
		inner["b"] = "changed"
		assert.Equal(t, "value", copied["a"].(map[string]any)["b"])
	})
}

func TestDeepCopySlice(t *testing.T) {
	t.Run("nil slice", func(t *testing.T) {
		assert.Nil(t, deepCopySlice(nil))
	})

	t.Run("nested slice", func(t *testing.T) {
		s := []any{map[string]any{"k": "v"}}
		copied := deepCopySlice(s)
		s[0].(map[string]any)["k"] = "changed"
		assert.Equal(t, "v", copied[0].(map[string]any)["k"])
	})
}

// =============================================================================
// Mock types for testing
// =============================================================================

type mockBodyResolver struct{}

func (m *mockBodyResolver) ResolveSpecBody(ctx context.Context, slug string, version string) (string, error) {
	return "body", nil
}

// =============================================================================
// Child propagates engine, depth, specResolver, refDepth, refChain
// =============================================================================

func TestContext_ChildPropagation(t *testing.T) {
	engine := MustNew()
	mock := &mockBodyResolver{}

	parent := NewContext(map[string]any{"a": 1})
	parent = parent.WithEngine(engine)
	parent = parent.WithDepth(3)
	parent = parent.WithSpecResolver(mock)
	parent = parent.WithRefDepth(2)
	parent = parent.WithRefChain([]string{"slug1"})

	child := parent.Child(map[string]any{"b": 2}).(*Context)

	assert.Equal(t, engine, child.Engine())
	assert.Equal(t, 3, child.Depth())
	assert.Equal(t, mock, child.SpecResolver())
	assert.Equal(t, 2, child.RefDepth())
	assert.Equal(t, []string{"slug1"}, child.RefChain())
}

// TestContext_WithMethods_ConcurrentAccess verifies that all With* methods
// are safe for concurrent access with -race. They use RLock (not Lock)
// so multiple goroutines can create derived contexts simultaneously.
func TestContext_WithMethods_ConcurrentAccess(t *testing.T) {
	ctx := NewContext(map[string]any{
		"key1": "value1",
		"key2": map[string]any{"nested": "data"},
	})

	engine := MustNew()
	var wg sync.WaitGroup
	const goroutines = 50

	for i := 0; i < goroutines; i++ {
		wg.Add(5)
		go func() {
			defer wg.Done()
			_ = ctx.WithEngine(engine)
		}()
		go func() {
			defer wg.Done()
			_ = ctx.WithDepth(i)
		}()
		go func() {
			defer wg.Done()
			_ = ctx.WithSpecResolver(nil)
		}()
		go func() {
			defer wg.Done()
			_ = ctx.WithRefDepth(i)
		}()
		go func() {
			defer wg.Done()
			_ = ctx.WithRefChain([]string{"slug1", "slug2"})
		}()
	}
	wg.Wait()

	// Original context should be unmodified
	val, ok := ctx.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)
}
