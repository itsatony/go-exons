package internal

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutor_NewExecutor(t *testing.T) {
	registry := NewRegistry(nil)
	config := DefaultExecutorConfig()

	executor := NewExecutor(registry, config, nil)
	require.NotNil(t, executor)
}

func TestExecutor_ExecutePlainText(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	root := &RootNode{
		Children: []Node{
			NewTextNode("Hello, World!", Position{Line: 1, Column: 1}),
		},
	}

	ctx := newMockContextAccessor(nil)
	result, err := executor.Execute(context.Background(), root, ctx)

	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", result)
}

func TestExecutor_ExecuteMultipleTextNodes(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	root := &RootNode{
		Children: []Node{
			NewTextNode("Hello, ", Position{Line: 1, Column: 1}),
			NewTextNode("World", Position{Line: 1, Column: 8}),
			NewTextNode("!", Position{Line: 1, Column: 13}),
		},
	}

	ctx := newMockContextAccessor(nil)
	result, err := executor.Execute(context.Background(), root, ctx)

	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", result)
}

func TestExecutor_ExecuteVarTag(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	root := &RootNode{
		Children: []Node{
			NewTextNode("Hello, ", Position{Line: 1, Column: 1}),
			NewSelfClosingTag(TagNameVar, Attributes{"name": "user"}, Position{Line: 1, Column: 8}),
			NewTextNode("!", Position{Line: 1, Column: 30}),
		},
	}

	ctx := newMockContextAccessor(map[string]any{
		"user": "Alice",
	})

	result, err := executor.Execute(context.Background(), root, ctx)

	require.NoError(t, err)
	assert.Equal(t, "Hello, Alice!", result)
}

func TestExecutor_ExecuteVarWithDefault(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	root := &RootNode{
		Children: []Node{
			NewSelfClosingTag(TagNameVar, Attributes{
				"name":    "missing",
				"default": "Guest",
			}, Position{Line: 1, Column: 1}),
		},
	}

	ctx := newMockContextAccessor(nil)
	result, err := executor.Execute(context.Background(), root, ctx)

	require.NoError(t, err)
	assert.Equal(t, "Guest", result)
}

func TestExecutor_ExecuteRawBlock(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	// Create a raw block with literal tag syntax inside
	rawTag := NewRawBlockTag(`{~exons.var name="x" /~}`, Position{Line: 1, Column: 1})

	root := &RootNode{
		Children: []Node{rawTag},
	}

	ctx := newMockContextAccessor(nil)
	result, err := executor.Execute(context.Background(), root, ctx)

	require.NoError(t, err)
	// Raw content should be preserved literally
	assert.Equal(t, `{~exons.var name="x" /~}`, result)
}

func TestExecutor_ExecuteNestedTags(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry)

	// Register a custom block tag resolver
	registry.MustRegister(&testBlockResolver{name: "wrapper"})

	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	// Create a block tag with children
	blockTag := NewBlockTag("wrapper", Attributes{}, []Node{
		NewTextNode("Inner content", Position{Line: 1, Column: 10}),
	}, Position{Line: 1, Column: 1})

	root := &RootNode{
		Children: []Node{blockTag},
	}

	ctx := newMockContextAccessor(nil)
	result, err := executor.Execute(context.Background(), root, ctx)

	require.NoError(t, err)
	// Block resolver returns "[" and children append
	assert.Equal(t, "[Inner content", result)
}

func TestExecutor_ExecuteUnknownTag(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	root := &RootNode{
		Children: []Node{
			NewSelfClosingTag("unknown.tag", Attributes{}, Position{Line: 1, Column: 1}),
		},
	}

	ctx := newMockContextAccessor(nil)
	_, err := executor.Execute(context.Background(), root, ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgUnknownTag)
	assert.Contains(t, err.Error(), "unknown.tag")
}

func TestExecutor_ExecuteMissingVariable(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	root := &RootNode{
		Children: []Node{
			NewSelfClosingTag(TagNameVar, Attributes{"name": "missing"}, Position{Line: 1, Column: 1}),
		},
	}

	ctx := newMockContextAccessor(nil)
	_, err := executor.Execute(context.Background(), root, ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgResolverFailed)
}

func TestExecutor_MaxDepthExceeded(t *testing.T) {
	registry := NewRegistry(nil)
	registry.MustRegister(&testBlockResolver{name: "block"})

	config := ExecutorConfig{MaxDepth: 2}
	executor := NewExecutor(registry, config, nil)

	// Create deeply nested structure
	innermost := NewBlockTag("block", Attributes{}, []Node{
		NewTextNode("deep", Position{Line: 1, Column: 1}),
	}, Position{Line: 1, Column: 1})

	middle := NewBlockTag("block", Attributes{}, []Node{innermost}, Position{Line: 1, Column: 1})
	outer := NewBlockTag("block", Attributes{}, []Node{middle}, Position{Line: 1, Column: 1})

	root := &RootNode{Children: []Node{outer}}

	ctx := newMockContextAccessor(nil)
	_, err := executor.Execute(context.Background(), root, ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgMaxDepthExceeded)
}

func TestExecutor_NoDepthLimitWhenZero(t *testing.T) {
	registry := NewRegistry(nil)
	registry.MustRegister(&testBlockResolver{name: "block"})

	config := ExecutorConfig{MaxDepth: 0} // Unlimited
	executor := NewExecutor(registry, config, nil)

	// Create nested structure
	inner := NewBlockTag("block", Attributes{}, []Node{
		NewTextNode("deep", Position{Line: 1, Column: 1}),
	}, Position{Line: 1, Column: 1})

	outer := NewBlockTag("block", Attributes{}, []Node{inner}, Position{Line: 1, Column: 1})
	root := &RootNode{Children: []Node{outer}}

	ctx := newMockContextAccessor(nil)
	result, err := executor.Execute(context.Background(), root, ctx)

	require.NoError(t, err)
	assert.Equal(t, "[[deep", result) // Two "[" from the two block resolvers
}

func TestExecutor_ResolverError(t *testing.T) {
	registry := NewRegistry(nil)
	registry.MustRegister(&testErrorResolver{})

	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	root := &RootNode{
		Children: []Node{
			NewSelfClosingTag("error.tag", Attributes{}, Position{Line: 5, Column: 10}),
		},
	}

	ctx := newMockContextAccessor(nil)
	_, err := executor.Execute(context.Background(), root, ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgResolverFailed)
}

func TestExecutor_ComplexTemplate(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	// Simulate a parsed template like:
	// Hello, {~exons.var name="user" /~}!
	// Your items: {~exons.raw~}{{ items }}{~/exons.raw~}
	root := &RootNode{
		Children: []Node{
			NewTextNode("Hello, ", Position{Line: 1, Column: 1}),
			NewSelfClosingTag(TagNameVar, Attributes{"name": "user"}, Position{Line: 1, Column: 8}),
			NewTextNode("!\nYour items: ", Position{Line: 1, Column: 35}),
			NewRawBlockTag("{{ items }}", Position{Line: 2, Column: 13}),
		},
	}

	ctx := newMockContextAccessor(map[string]any{
		"user": "Bob",
	})

	result, err := executor.Execute(context.Background(), root, ctx)

	require.NoError(t, err)
	assert.Equal(t, "Hello, Bob!\nYour items: {{ items }}", result)
}

func TestExecutor_EmptyRoot(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	root := &RootNode{Children: []Node{}}

	ctx := newMockContextAccessor(nil)
	result, err := executor.Execute(context.Background(), root, ctx)

	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestExecutorError_Error(t *testing.T) {
	t.Run("with tag name", func(t *testing.T) {
		err := NewExecutorError("test error", "my.tag", Position{Line: 5, Column: 10})
		errStr := err.Error()
		assert.Contains(t, errStr, "test error")
		assert.Contains(t, errStr, "5")
	})

	t.Run("without tag name", func(t *testing.T) {
		err := NewExecutorError("test error", "", Position{Line: 1, Column: 1})
		errStr := err.Error()
		assert.Contains(t, errStr, "test error")
	})
}

func TestDefaultExecutorConfig(t *testing.T) {
	config := DefaultExecutorConfig()
	assert.Equal(t, DefaultMaxDepth, config.MaxDepth)
}

// Test helpers

// testBlockResolver is a test resolver that returns "[" for block tags
type testBlockResolver struct {
	name string
}

func (r *testBlockResolver) TagName() string { return r.name }

func (r *testBlockResolver) Resolve(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error) {
	return "[", nil
}

func (r *testBlockResolver) Validate(attrs Attributes) error {
	return nil
}

// testErrorResolver is a test resolver that always returns an error
type testErrorResolver struct{}

func (r *testErrorResolver) TagName() string { return "error.tag" }

func (r *testErrorResolver) Resolve(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error) {
	return "", errors.New("intentional error")
}

func (r *testErrorResolver) Validate(attrs Attributes) error {
	return nil
}

// Tests for Executor function wrappers
func TestExecutor_RegisterFunc(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	fn := &Func{
		Name:    "testfunc",
		MinArgs: 0,
		MaxArgs: 0,
		Fn:      func(args []any) (any, error) { return "test", nil },
	}

	err := executor.RegisterFunc(fn)
	require.NoError(t, err)
	assert.True(t, executor.HasFunc("testfunc"))
}

func TestExecutor_RegisterFunc_Duplicate(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	fn := &Func{
		Name:    "testfunc",
		MinArgs: 0,
		MaxArgs: 0,
		Fn:      func(args []any) (any, error) { return "test", nil },
	}

	err := executor.RegisterFunc(fn)
	require.NoError(t, err)

	// Second registration should fail
	err = executor.RegisterFunc(fn)
	require.Error(t, err)
}

func TestExecutor_MustRegisterFunc(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	fn := &Func{
		Name:    "testfunc",
		MinArgs: 0,
		MaxArgs: 0,
		Fn:      func(args []any) (any, error) { return "test", nil },
	}

	assert.NotPanics(t, func() {
		executor.MustRegisterFunc(fn)
	})
	assert.True(t, executor.HasFunc("testfunc"))
}

func TestExecutor_MustRegisterFunc_Panic(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	fn := &Func{
		Name:    "testfunc",
		MinArgs: 0,
		MaxArgs: 0,
		Fn:      func(args []any) (any, error) { return "test", nil },
	}

	executor.MustRegisterFunc(fn)

	assert.Panics(t, func() {
		executor.MustRegisterFunc(fn) // duplicate
	})
}

func TestExecutor_HasFunc(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	assert.False(t, executor.HasFunc("nonexistent"))

	fn := &Func{
		Name:    "exists",
		MinArgs: 0,
		MaxArgs: 0,
		Fn:      func(args []any) (any, error) { return nil, nil },
	}
	executor.MustRegisterFunc(fn)

	assert.True(t, executor.HasFunc("exists"))
}

func TestExecutor_ListFuncs(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	executor.MustRegisterFunc(&Func{Name: "customfunc1", MinArgs: 0, MaxArgs: 0, Fn: func(args []any) (any, error) { return nil, nil }})
	executor.MustRegisterFunc(&Func{Name: "customfunc2", MinArgs: 0, MaxArgs: 0, Fn: func(args []any) (any, error) { return nil, nil }})

	funcs := executor.ListFuncs()
	// Should have built-in functions plus custom ones
	assert.True(t, len(funcs) > 2, "should have multiple functions")
	assert.Contains(t, funcs, "customfunc1")
	assert.Contains(t, funcs, "customfunc2")
	// Verify some built-in functions are present
	assert.Contains(t, funcs, "len")
	assert.Contains(t, funcs, "upper")
}

func TestExecutor_FuncCount(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	// Executor starts with built-in functions registered
	initialCount := executor.FuncCount()
	assert.True(t, initialCount > 0, "should have built-in functions")

	executor.MustRegisterFunc(&Func{Name: "customfunc1", MinArgs: 0, MaxArgs: 0, Fn: func(args []any) (any, error) { return nil, nil }})
	assert.Equal(t, initialCount+1, executor.FuncCount())

	executor.MustRegisterFunc(&Func{Name: "customfunc2", MinArgs: 0, MaxArgs: 0, Fn: func(args []any) (any, error) { return nil, nil }})
	assert.Equal(t, initialCount+2, executor.FuncCount())
}

func TestExecutorError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	execErr := &ExecutorError{
		Message:  "execution failed",
		TagName:  "test.tag",
		Position: Position{Line: 1, Column: 1},
		Cause:    cause,
	}

	assert.Equal(t, cause, execErr.Unwrap())
}

func TestExecutorError_UnwrapNil(t *testing.T) {
	execErr := &ExecutorError{
		Message:  "execution failed",
		TagName:  "test.tag",
		Position: Position{Line: 1, Column: 1},
		Cause:    nil,
	}

	assert.Nil(t, execErr.Unwrap())
}

// mockContextAccessorWithChild is a mock that supports child context creation for loop tests
type mockContextAccessorWithChild struct {
	data   map[string]any
	parent *mockContextAccessorWithChild
}

func newMockContextAccessorWithChild(data map[string]any) *mockContextAccessorWithChild {
	if data == nil {
		data = make(map[string]any)
	}
	return &mockContextAccessorWithChild{data: data}
}

func (m *mockContextAccessorWithChild) Get(path string) (any, bool) {
	// Support dot notation paths like "entry.key"
	if strings.Contains(path, ".") {
		parts := strings.Split(path, ".")
		var current any = m.data

		for _, part := range parts {
			if part == "" {
				continue
			}

			switch v := current.(type) {
			case map[string]any:
				val, ok := v[part]
				if !ok {
					// Try parent context if not found
					if m.parent != nil {
						return m.parent.Get(path)
					}
					return nil, false
				}
				current = val
			default:
				// Can't traverse further
				if m.parent != nil {
					return m.parent.Get(path)
				}
				return nil, false
			}
		}
		return current, true
	}

	// Simple path - check current data first
	val, ok := m.data[path]
	if ok {
		return val, true
	}
	// Check parent if available
	if m.parent != nil {
		return m.parent.Get(path)
	}
	return nil, false
}

func (m *mockContextAccessorWithChild) GetString(path string) string {
	val, ok := m.Get(path)
	if !ok {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

func (m *mockContextAccessorWithChild) GetStringDefault(path, defaultVal string) string {
	val := m.GetString(path)
	if val == "" {
		return defaultVal
	}
	return val
}

func (m *mockContextAccessorWithChild) Has(path string) bool {
	_, ok := m.Get(path)
	return ok
}

func (m *mockContextAccessorWithChild) Child(data map[string]any) interface{} {
	child := newMockContextAccessorWithChild(data)
	child.parent = m
	return child
}

// TestExecutor_ExecuteConditional tests the executeConditional function
func TestExecutor_ExecuteConditional(t *testing.T) {
	t.Run("if branch evaluates true", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"isAdmin": true,
		})

		// Create conditional: if isAdmin
		cond := NewConditionalNode([]ConditionalBranch{
			NewConditionalBranch("isAdmin", []Node{
				NewTextNode("Admin content", Position{Line: 1, Column: 1}),
			}, false, Position{Line: 1, Column: 1}),
		}, Position{Line: 1, Column: 1})

		result, err := executor.executeConditional(context.Background(), cond, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "Admin content", result)
	})

	t.Run("if branch false, else executes", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"isAdmin": false,
		})

		// Create conditional: if isAdmin / else
		cond := NewConditionalNode([]ConditionalBranch{
			NewConditionalBranch("isAdmin", []Node{
				NewTextNode("Admin content", Position{Line: 1, Column: 1}),
			}, false, Position{Line: 1, Column: 1}),
			NewConditionalBranch("", []Node{
				NewTextNode("Guest content", Position{Line: 2, Column: 1}),
			}, true, Position{Line: 2, Column: 1}),
		}, Position{Line: 1, Column: 1})

		result, err := executor.executeConditional(context.Background(), cond, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "Guest content", result)
	})

	t.Run("multiple elseif branches, middle one matches", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"role": "user",
		})

		// Create conditional: if role == "admin" / elseif role == "user" / else
		cond := NewConditionalNode([]ConditionalBranch{
			NewConditionalBranch("role == \"admin\"", []Node{
				NewTextNode("Admin", Position{Line: 1, Column: 1}),
			}, false, Position{Line: 1, Column: 1}),
			NewConditionalBranch("role == \"user\"", []Node{
				NewTextNode("User", Position{Line: 2, Column: 1}),
			}, false, Position{Line: 2, Column: 1}),
			NewConditionalBranch("", []Node{
				NewTextNode("Guest", Position{Line: 3, Column: 1}),
			}, true, Position{Line: 3, Column: 1}),
		}, Position{Line: 1, Column: 1})

		result, err := executor.executeConditional(context.Background(), cond, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "User", result)
	})

	t.Run("no branch matches, else executes", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"role": "guest",
		})

		cond := NewConditionalNode([]ConditionalBranch{
			NewConditionalBranch("role == \"admin\"", []Node{
				NewTextNode("Admin", Position{Line: 1, Column: 1}),
			}, false, Position{Line: 1, Column: 1}),
			NewConditionalBranch("role == \"user\"", []Node{
				NewTextNode("User", Position{Line: 2, Column: 1}),
			}, false, Position{Line: 2, Column: 1}),
			NewConditionalBranch("", []Node{
				NewTextNode("Guest", Position{Line: 3, Column: 1}),
			}, true, Position{Line: 3, Column: 1}),
		}, Position{Line: 1, Column: 1})

		result, err := executor.executeConditional(context.Background(), cond, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "Guest", result)
	})

	t.Run("no branch matches, no else (empty result)", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"value": false,
		})

		cond := NewConditionalNode([]ConditionalBranch{
			NewConditionalBranch("value", []Node{
				NewTextNode("Content", Position{Line: 1, Column: 1}),
			}, false, Position{Line: 1, Column: 1}),
		}, Position{Line: 1, Column: 1})

		result, err := executor.executeConditional(context.Background(), cond, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("condition evaluation error propagation", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(nil)

		// Invalid expression
		cond := NewConditionalNode([]ConditionalBranch{
			NewConditionalBranch("invalidFunc()", []Node{
				NewTextNode("Content", Position{Line: 1, Column: 1}),
			}, false, Position{Line: 1, Column: 1}),
		}, Position{Line: 1, Column: 1})

		_, err := executor.executeConditional(context.Background(), cond, ctx, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgCondExprFailed)
	})

	t.Run("nested conditionals evaluation", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"outer": true,
			"inner": true,
		})

		// Outer conditional with nested conditional in its body
		innerCond := NewConditionalNode([]ConditionalBranch{
			NewConditionalBranch("inner", []Node{
				NewTextNode("Inner true", Position{Line: 2, Column: 1}),
			}, false, Position{Line: 2, Column: 1}),
		}, Position{Line: 2, Column: 1})

		outerCond := NewConditionalNode([]ConditionalBranch{
			NewConditionalBranch("outer", []Node{
				NewTextNode("Outer true, ", Position{Line: 1, Column: 1}),
				innerCond,
			}, false, Position{Line: 1, Column: 1}),
		}, Position{Line: 1, Column: 1})

		result, err := executor.executeConditional(context.Background(), outerCond, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "Outer true, Inner true", result)
	})

	t.Run("complex boolean expressions", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"isLoggedIn": true,
			"isAdmin":    false,
			"count":      5,
		})

		// Test && operator
		cond := NewConditionalNode([]ConditionalBranch{
			NewConditionalBranch("isLoggedIn && !isAdmin", []Node{
				NewTextNode("Logged in user", Position{Line: 1, Column: 1}),
			}, false, Position{Line: 1, Column: 1}),
		}, Position{Line: 1, Column: 1})

		result, err := executor.executeConditional(context.Background(), cond, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "Logged in user", result)

		// Test || and comparison
		cond2 := NewConditionalNode([]ConditionalBranch{
			NewConditionalBranch("isAdmin || count > 3", []Node{
				NewTextNode("Match", Position{Line: 1, Column: 1}),
			}, false, Position{Line: 1, Column: 1}),
		}, Position{Line: 1, Column: 1})

		result2, err := executor.executeConditional(context.Background(), cond2, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "Match", result2)
	})
}

// TestExecutor_EvaluateCondition tests the evaluateCondition function
func TestExecutor_EvaluateCondition(t *testing.T) {
	t.Run("simple boolean variable", func(t *testing.T) {
		registry := NewRegistry(nil)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"flag": true,
		})

		result, err := executor.evaluateCondition(context.Background(), "flag", ctx)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("comparison expression", func(t *testing.T) {
		registry := NewRegistry(nil)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"count": 10,
		})

		result, err := executor.evaluateCondition(context.Background(), "count > 5", ctx)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("invalid expression returns error", func(t *testing.T) {
		registry := NewRegistry(nil)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(nil)

		_, err := executor.evaluateCondition(context.Background(), "undefined_func()", ctx)
		require.Error(t, err)
	})
}

// TestExecutor_ExecuteFor tests the executeFor function
func TestExecutor_ExecuteFor(t *testing.T) {
	t.Run("iteration over []string", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessorWithChild(map[string]any{
			"items": []string{"apple", "banana", "cherry"},
		})

		forNode := NewForNode("item", "", "items", 0, []Node{
			NewSelfClosingTag(TagNameVar, Attributes{"name": "item"}, Position{Line: 1, Column: 1}),
			NewTextNode(",", Position{Line: 1, Column: 2}),
		}, Position{Line: 1, Column: 1})

		result, err := executor.executeFor(context.Background(), forNode, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "apple,banana,cherry,", result)
	})

	t.Run("iteration over []int", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessorWithChild(map[string]any{
			"numbers": []int{1, 2, 3},
		})

		forNode := NewForNode("num", "", "numbers", 0, []Node{
			NewSelfClosingTag(TagNameVar, Attributes{"name": "num"}, Position{Line: 1, Column: 1}),
			NewTextNode(" ", Position{Line: 1, Column: 2}),
		}, Position{Line: 1, Column: 1})

		result, err := executor.executeFor(context.Background(), forNode, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "1 2 3 ", result)
	})

	t.Run("iteration over []any", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessorWithChild(map[string]any{
			"mixed": []any{"text", 42, true},
		})

		forNode := NewForNode("val", "", "mixed", 0, []Node{
			NewSelfClosingTag(TagNameVar, Attributes{"name": "val"}, Position{Line: 1, Column: 1}),
			NewTextNode("|", Position{Line: 1, Column: 2}),
		}, Position{Line: 1, Column: 1})

		result, err := executor.executeFor(context.Background(), forNode, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "text|42|true|", result)
	})

	t.Run("iteration over map[string]any (sorted keys)", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessorWithChild(map[string]any{
			"data": map[string]any{
				"z": "last",
				"a": "first",
				"m": "middle",
			},
		})

		forNode := NewForNode("entry", "", "data", 0, []Node{
			NewSelfClosingTag(TagNameVar, Attributes{"name": "entry.key"}, Position{Line: 1, Column: 1}),
			NewTextNode(":", Position{Line: 1, Column: 2}),
			NewSelfClosingTag(TagNameVar, Attributes{"name": "entry.value"}, Position{Line: 1, Column: 3}),
			NewTextNode(",", Position{Line: 1, Column: 4}),
		}, Position{Line: 1, Column: 1})

		result, err := executor.executeFor(context.Background(), forNode, ctx, 0)
		require.NoError(t, err)
		// Keys should be sorted alphabetically
		assert.Equal(t, "a:first,m:middle,z:last,", result)
	})

	t.Run("empty collection (zero iterations)", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessorWithChild(map[string]any{
			"items": []string{},
		})

		forNode := NewForNode("item", "", "items", 0, []Node{
			NewTextNode("should not appear", Position{Line: 1, Column: 1}),
		}, Position{Line: 1, Column: 1})

		result, err := executor.executeFor(context.Background(), forNode, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("nil collection (treated as empty)", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessorWithChild(map[string]any{
			"items": nil,
		})

		forNode := NewForNode("item", "", "items", 0, []Node{
			NewTextNode("should not appear", Position{Line: 1, Column: 1}),
		}, Position{Line: 1, Column: 1})

		result, err := executor.executeFor(context.Background(), forNode, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("limit applied (fewer iterations)", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessorWithChild(map[string]any{
			"items": []string{"a", "b", "c", "d", "e"},
		})

		forNode := NewForNode("item", "", "items", 3, []Node{
			NewSelfClosingTag(TagNameVar, Attributes{"name": "item"}, Position{Line: 1, Column: 1}),
		}, Position{Line: 1, Column: 1})

		result, err := executor.executeFor(context.Background(), forNode, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "abc", result)
	})

	t.Run("index variable populated correctly", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessorWithChild(map[string]any{
			"items": []string{"a", "b", "c"},
		})

		forNode := NewForNode("item", "i", "items", 0, []Node{
			NewSelfClosingTag(TagNameVar, Attributes{"name": "i"}, Position{Line: 1, Column: 1}),
			NewTextNode(":", Position{Line: 1, Column: 2}),
			NewSelfClosingTag(TagNameVar, Attributes{"name": "item"}, Position{Line: 1, Column: 3}),
			NewTextNode(",", Position{Line: 1, Column: 4}),
		}, Position{Line: 1, Column: 1})

		result, err := executor.executeFor(context.Background(), forNode, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "0:a,1:b,2:c,", result)
	})

	t.Run("child context isolation", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessorWithChild(map[string]any{
			"items":  []string{"a", "b"},
			"global": "parent",
		})

		forNode := NewForNode("item", "", "items", 0, []Node{
			NewSelfClosingTag(TagNameVar, Attributes{"name": "item"}, Position{Line: 1, Column: 1}),
			NewTextNode("-", Position{Line: 1, Column: 2}),
			NewSelfClosingTag(TagNameVar, Attributes{"name": "global"}, Position{Line: 1, Column: 3}),
			NewTextNode(",", Position{Line: 1, Column: 4}),
		}, Position{Line: 1, Column: 1})

		result, err := executor.executeFor(context.Background(), forNode, ctx, 0)
		require.NoError(t, err)
		// Child should have access to both loop var and parent var
		assert.Equal(t, "a-parent,b-parent,", result)
	})

	t.Run("collection path not found (error)", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessorWithChild(map[string]any{})

		forNode := NewForNode("item", "", "nonexistent", 0, []Node{
			NewTextNode("content", Position{Line: 1, Column: 1}),
		}, Position{Line: 1, Column: 1})

		_, err := executor.executeFor(context.Background(), forNode, ctx, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgForCollectionPath)
	})

	t.Run("non-iterable type (error)", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessorWithChild(map[string]any{
			"notIterable": "string value",
		})

		forNode := NewForNode("item", "", "notIterable", 0, []Node{
			NewTextNode("content", Position{Line: 1, Column: 1}),
		}, Position{Line: 1, Column: 1})

		_, err := executor.executeFor(context.Background(), forNode, ctx, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgForNotIterable)
	})

	t.Run("nested loops", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessorWithChild(map[string]any{
			"outer": []string{"A", "B"},
			"inner": []int{1, 2},
		})

		innerFor := NewForNode("num", "", "inner", 0, []Node{
			NewSelfClosingTag(TagNameVar, Attributes{"name": "letter"}, Position{Line: 2, Column: 1}),
			NewSelfClosingTag(TagNameVar, Attributes{"name": "num"}, Position{Line: 2, Column: 2}),
			NewTextNode(" ", Position{Line: 2, Column: 3}),
		}, Position{Line: 2, Column: 1})

		outerFor := NewForNode("letter", "", "outer", 0, []Node{
			innerFor,
		}, Position{Line: 1, Column: 1})

		result, err := executor.executeFor(context.Background(), outerFor, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "A1 A2 B1 B2 ", result)
	})

	t.Run("context does not support child creation", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		// Use regular mock without child support
		ctx := newMockContextAccessor(map[string]any{
			"items": []string{"a", "b"},
		})

		forNode := NewForNode("item", "", "items", 0, []Node{
			NewTextNode("content", Position{Line: 1, Column: 1}),
		}, Position{Line: 1, Column: 1})

		_, err := executor.executeFor(context.Background(), forNode, ctx, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgForContextNoChild)
	})
}

// TestExecutor_ExecuteSwitch tests the executeSwitch function
func TestExecutor_ExecuteSwitch(t *testing.T) {
	t.Run("case matches via value comparison", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"status": "active",
		})

		switchNode := NewSwitchNode("status", []SwitchCase{
			NewSwitchCase("active", "", []Node{
				NewTextNode("Active status", Position{Line: 1, Column: 1}),
			}, false, Position{Line: 1, Column: 1}),
			NewSwitchCase("inactive", "", []Node{
				NewTextNode("Inactive status", Position{Line: 2, Column: 1}),
			}, false, Position{Line: 2, Column: 1}),
		}, nil, Position{Line: 1, Column: 1})

		result, err := executor.executeSwitch(context.Background(), switchNode, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "Active status", result)
	})

	t.Run("case matches via eval expression", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"count": 15,
		})

		switchNode := NewSwitchNode("count", []SwitchCase{
			NewSwitchCase("", "count < 10", []Node{
				NewTextNode("Low", Position{Line: 1, Column: 1}),
			}, false, Position{Line: 1, Column: 1}),
			NewSwitchCase("", "count >= 10 && count < 20", []Node{
				NewTextNode("Medium", Position{Line: 2, Column: 1}),
			}, false, Position{Line: 2, Column: 1}),
		}, nil, Position{Line: 1, Column: 1})

		result, err := executor.executeSwitch(context.Background(), switchNode, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "Medium", result)
	})

	t.Run("second case matches (first doesn't)", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"role": "user",
		})

		switchNode := NewSwitchNode("role", []SwitchCase{
			NewSwitchCase("admin", "", []Node{
				NewTextNode("Admin", Position{Line: 1, Column: 1}),
			}, false, Position{Line: 1, Column: 1}),
			NewSwitchCase("user", "", []Node{
				NewTextNode("User", Position{Line: 2, Column: 1}),
			}, false, Position{Line: 2, Column: 1}),
			NewSwitchCase("guest", "", []Node{
				NewTextNode("Guest", Position{Line: 3, Column: 1}),
			}, false, Position{Line: 3, Column: 1}),
		}, nil, Position{Line: 1, Column: 1})

		result, err := executor.executeSwitch(context.Background(), switchNode, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "User", result)
	})

	t.Run("no case matches, default executes", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"status": "unknown",
		})

		defaultCase := NewSwitchCase("", "", []Node{
			NewTextNode("Unknown status", Position{Line: 3, Column: 1}),
		}, true, Position{Line: 3, Column: 1})

		switchNode := NewSwitchNode("status", []SwitchCase{
			NewSwitchCase("active", "", []Node{
				NewTextNode("Active", Position{Line: 1, Column: 1}),
			}, false, Position{Line: 1, Column: 1}),
			NewSwitchCase("inactive", "", []Node{
				NewTextNode("Inactive", Position{Line: 2, Column: 1}),
			}, false, Position{Line: 2, Column: 1}),
		}, &defaultCase, Position{Line: 1, Column: 1})

		result, err := executor.executeSwitch(context.Background(), switchNode, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "Unknown status", result)
	})

	t.Run("no case matches, no default (empty result)", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"status": "pending",
		})

		switchNode := NewSwitchNode("status", []SwitchCase{
			NewSwitchCase("active", "", []Node{
				NewTextNode("Active", Position{Line: 1, Column: 1}),
			}, false, Position{Line: 1, Column: 1}),
		}, nil, Position{Line: 1, Column: 1})

		result, err := executor.executeSwitch(context.Background(), switchNode, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("integer switch values", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"code": 200,
		})

		switchNode := NewSwitchNode("code", []SwitchCase{
			NewSwitchCase("200", "", []Node{
				NewTextNode("OK", Position{Line: 1, Column: 1}),
			}, false, Position{Line: 1, Column: 1}),
			NewSwitchCase("404", "", []Node{
				NewTextNode("Not Found", Position{Line: 2, Column: 1}),
			}, false, Position{Line: 2, Column: 1}),
		}, nil, Position{Line: 1, Column: 1})

		result, err := executor.executeSwitch(context.Background(), switchNode, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "OK", result)
	})

	t.Run("boolean switch values", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"enabled": true,
		})

		switchNode := NewSwitchNode("enabled", []SwitchCase{
			NewSwitchCase("true", "", []Node{
				NewTextNode("Enabled", Position{Line: 1, Column: 1}),
			}, false, Position{Line: 1, Column: 1}),
			NewSwitchCase("false", "", []Node{
				NewTextNode("Disabled", Position{Line: 2, Column: 1}),
			}, false, Position{Line: 2, Column: 1}),
		}, nil, Position{Line: 1, Column: 1})

		result, err := executor.executeSwitch(context.Background(), switchNode, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "Enabled", result)
	})

	t.Run("string switch values", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"color": "blue",
		})

		switchNode := NewSwitchNode("color", []SwitchCase{
			NewSwitchCase("red", "", []Node{
				NewTextNode("Red", Position{Line: 1, Column: 1}),
			}, false, Position{Line: 1, Column: 1}),
			NewSwitchCase("blue", "", []Node{
				NewTextNode("Blue", Position{Line: 2, Column: 1}),
			}, false, Position{Line: 2, Column: 1}),
		}, nil, Position{Line: 1, Column: 1})

		result, err := executor.executeSwitch(context.Background(), switchNode, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "Blue", result)
	})

	t.Run("nested path expressions in switch", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"user.role": "admin",
		})

		switchNode := NewSwitchNode("user.role", []SwitchCase{
			NewSwitchCase("admin", "", []Node{
				NewTextNode("Admin user", Position{Line: 1, Column: 1}),
			}, false, Position{Line: 1, Column: 1}),
		}, nil, Position{Line: 1, Column: 1})

		result, err := executor.executeSwitch(context.Background(), switchNode, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "Admin user", result)
	})

	t.Run("case evaluation error propagation", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"value": 10,
		})

		switchNode := NewSwitchNode("value", []SwitchCase{
			NewSwitchCase("", "invalidFunc()", []Node{
				NewTextNode("Content", Position{Line: 1, Column: 1}),
			}, false, Position{Line: 1, Column: 1}),
		}, nil, Position{Line: 1, Column: 1})

		_, err := executor.executeSwitch(context.Background(), switchNode, ctx, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgCondExprFailed)
	})

	t.Run("nested switch blocks", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(map[string]any{
			"outer": "a",
			"inner": "1",
		})

		innerSwitch := NewSwitchNode("inner", []SwitchCase{
			NewSwitchCase("1", "", []Node{
				NewTextNode("One", Position{Line: 2, Column: 1}),
			}, false, Position{Line: 2, Column: 1}),
		}, nil, Position{Line: 2, Column: 1})

		outerSwitch := NewSwitchNode("outer", []SwitchCase{
			NewSwitchCase("a", "", []Node{
				NewTextNode("A-", Position{Line: 1, Column: 1}),
				innerSwitch,
			}, false, Position{Line: 1, Column: 1}),
		}, nil, Position{Line: 1, Column: 1})

		result, err := executor.executeSwitch(context.Background(), outerSwitch, ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, "A-One", result)
	})

	t.Run("switch expression evaluation error", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry)
		executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

		ctx := newMockContextAccessor(nil)

		switchNode := NewSwitchNode("undefinedFunc()", []SwitchCase{
			NewSwitchCase("value", "", []Node{
				NewTextNode("Content", Position{Line: 1, Column: 1}),
			}, false, Position{Line: 1, Column: 1}),
		}, nil, Position{Line: 1, Column: 1})

		_, err := executor.executeSwitch(context.Background(), switchNode, ctx, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgCondExprFailed)
	})
}

// TestToIterableSlice tests the toIterableSlice helper function
func TestToIterableSlice(t *testing.T) {
	t.Run("nil returns empty slice", func(t *testing.T) {
		result, err := toIterableSlice(nil)
		require.NoError(t, err)
		assert.Equal(t, []any{}, result)
	})

	t.Run("[]any passes through", func(t *testing.T) {
		input := []any{"a", 1, true}
		result, err := toIterableSlice(input)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("[]string converts to []any", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result, err := toIterableSlice(input)
		require.NoError(t, err)
		assert.Equal(t, []any{"a", "b", "c"}, result)
	})

	t.Run("[]int converts to []any", func(t *testing.T) {
		input := []int{1, 2, 3}
		result, err := toIterableSlice(input)
		require.NoError(t, err)
		assert.Equal(t, []any{1, 2, 3}, result)
	})

	t.Run("[]int64 converts to []any", func(t *testing.T) {
		input := []int64{100, 200, 300}
		result, err := toIterableSlice(input)
		require.NoError(t, err)
		assert.Equal(t, []any{int64(100), int64(200), int64(300)}, result)
	})

	t.Run("[]float64 converts to []any", func(t *testing.T) {
		input := []float64{1.1, 2.2, 3.3}
		result, err := toIterableSlice(input)
		require.NoError(t, err)
		assert.Equal(t, []any{1.1, 2.2, 3.3}, result)
	})

	t.Run("[]bool converts to []any", func(t *testing.T) {
		input := []bool{true, false, true}
		result, err := toIterableSlice(input)
		require.NoError(t, err)
		assert.Equal(t, []any{true, false, true}, result)
	})

	t.Run("[]map[string]any converts to []any", func(t *testing.T) {
		input := []map[string]any{
			{"a": 1},
			{"b": 2},
		}
		result, err := toIterableSlice(input)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("map[string]any converts with sorted keys", func(t *testing.T) {
		input := map[string]any{
			"zebra":  "z",
			"alpha":  "a",
			"middle": "m",
		}
		result, err := toIterableSlice(input)
		require.NoError(t, err)
		assert.Len(t, result, 3)

		// Verify keys are sorted
		first := result[0].(map[string]any)
		assert.Equal(t, "alpha", first[ForMapKeyField])
		assert.Equal(t, "a", first[ForMapValueField])

		second := result[1].(map[string]any)
		assert.Equal(t, "middle", second[ForMapKeyField])

		third := result[2].(map[string]any)
		assert.Equal(t, "zebra", third[ForMapKeyField])
	})

	t.Run("non-iterable type returns error", func(t *testing.T) {
		input := "not iterable"
		_, err := toIterableSlice(input)
		require.Error(t, err)
		execErr, ok := err.(*ExecutorError)
		require.True(t, ok)
		assert.Equal(t, ErrMsgTypeNotIterable, execErr.Message)
	})

	t.Run("struct is not iterable", func(t *testing.T) {
		type testStruct struct {
			Field string
		}
		input := testStruct{Field: "value"}
		_, err := toIterableSlice(input)
		require.Error(t, err)
	})
}

// TestToSwitchString tests the toSwitchString helper function
func TestToSwitchString(t *testing.T) {
	t.Run("nil returns empty string", func(t *testing.T) {
		result := toSwitchString(nil)
		assert.Equal(t, "", result)
	})

	t.Run("string returns as-is", func(t *testing.T) {
		result := toSwitchString("test")
		assert.Equal(t, "test", result)
	})

	t.Run("bool true returns 'true'", func(t *testing.T) {
		result := toSwitchString(true)
		assert.Equal(t, AttrValueTrue, result)
	})

	t.Run("bool false returns 'false'", func(t *testing.T) {
		result := toSwitchString(false)
		assert.Equal(t, AttrValueFalse, result)
	})

	t.Run("int converts to string", func(t *testing.T) {
		result := toSwitchString(42)
		assert.Equal(t, "42", result)
	})

	t.Run("int64 converts to string", func(t *testing.T) {
		result := toSwitchString(int64(9223372036854775807))
		assert.Equal(t, "9223372036854775807", result)
	})

	t.Run("float64 converts to string", func(t *testing.T) {
		result := toSwitchString(3.14)
		assert.Equal(t, "3.14", result)
	})

	t.Run("float64 with no decimals uses %g format", func(t *testing.T) {
		result := toSwitchString(42.0)
		assert.Equal(t, "42", result)
	})

	t.Run("other types use fmt.Sprintf", func(t *testing.T) {
		result := toSwitchString([]string{"a", "b"})
		assert.Equal(t, "[a b]", result)
	})
}

// TestSortStrings tests the sortStrings helper function
func TestSortStrings(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		input := []string{}
		sortStrings(input)
		assert.Equal(t, []string{}, input)
	})

	t.Run("single element", func(t *testing.T) {
		input := []string{"single"}
		sortStrings(input)
		assert.Equal(t, []string{"single"}, input)
	})

	t.Run("already sorted", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		sortStrings(input)
		assert.Equal(t, []string{"a", "b", "c"}, input)
	})

	t.Run("reverse order", func(t *testing.T) {
		input := []string{"c", "b", "a"}
		sortStrings(input)
		assert.Equal(t, []string{"a", "b", "c"}, input)
	})

	t.Run("random order", func(t *testing.T) {
		input := []string{"zebra", "apple", "middle", "banana"}
		sortStrings(input)
		assert.Equal(t, []string{"apple", "banana", "middle", "zebra"}, input)
	})

	t.Run("duplicates", func(t *testing.T) {
		input := []string{"b", "a", "b", "a"}
		sortStrings(input)
		assert.Equal(t, []string{"a", "a", "b", "b"}, input)
	})
}
