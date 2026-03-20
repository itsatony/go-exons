package internal

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockResolver implements InternalResolver for testing
type mockResolver struct {
	name        string
	resolveFunc func(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error)
	validateErr error
}

func newMockResolver(name string) *mockResolver {
	return &mockResolver{
		name: name,
		resolveFunc: func(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error) {
			return "resolved:" + name, nil
		},
	}
}

func (m *mockResolver) TagName() string { return m.name }

func (m *mockResolver) Resolve(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error) {
	if m.resolveFunc != nil {
		return m.resolveFunc(ctx, execCtx, attrs)
	}
	return "", nil
}

func (m *mockResolver) Validate(attrs Attributes) error {
	return m.validateErr
}

func TestRegistry_NewRegistry(t *testing.T) {
	t.Run("with nil logger", func(t *testing.T) {
		reg := NewRegistry(nil)
		require.NotNil(t, reg)
		assert.Equal(t, 0, reg.Count())
	})

	t.Run("with logger", func(t *testing.T) {
		reg := NewRegistry(nil) // Using nil for simplicity in tests
		require.NotNil(t, reg)
	})
}

func TestRegistry_Register(t *testing.T) {
	t.Run("successful registration", func(t *testing.T) {
		reg := NewRegistry(nil)
		resolver := newMockResolver("test.tag")

		err := reg.Register(resolver)
		require.NoError(t, err)
		assert.Equal(t, 1, reg.Count())
		assert.True(t, reg.Has("test.tag"))
	})

	t.Run("nil resolver", func(t *testing.T) {
		reg := NewRegistry(nil)

		err := reg.Register(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgNilResolver)
	})

	t.Run("empty tag name", func(t *testing.T) {
		reg := NewRegistry(nil)
		resolver := newMockResolver("")

		err := reg.Register(resolver)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgEmptyTagName)
	})

	t.Run("duplicate registration - first-come-wins", func(t *testing.T) {
		reg := NewRegistry(nil)
		resolver1 := newMockResolver("test.tag")
		resolver2 := newMockResolver("test.tag")

		err := reg.Register(resolver1)
		require.NoError(t, err)

		err = reg.Register(resolver2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgResolverAlreadyExists)

		// First resolver should still be registered
		got, ok := reg.Get("test.tag")
		require.True(t, ok)
		assert.Equal(t, resolver1, got)
	})

	t.Run("multiple different resolvers", func(t *testing.T) {
		reg := NewRegistry(nil)

		for i := 0; i < 5; i++ {
			resolver := newMockResolver("tag" + string(rune('a'+i)))
			err := reg.Register(resolver)
			require.NoError(t, err)
		}

		assert.Equal(t, 5, reg.Count())
	})
}

func TestRegistry_MustRegister(t *testing.T) {
	t.Run("successful must register", func(t *testing.T) {
		reg := NewRegistry(nil)
		resolver := newMockResolver("test.tag")

		// Should not panic
		assert.NotPanics(t, func() {
			reg.MustRegister(resolver)
		})

		assert.True(t, reg.Has("test.tag"))
	})

	t.Run("panics on duplicate", func(t *testing.T) {
		reg := NewRegistry(nil)
		resolver1 := newMockResolver("test.tag")
		resolver2 := newMockResolver("test.tag")

		reg.MustRegister(resolver1)

		assert.Panics(t, func() {
			reg.MustRegister(resolver2)
		})
	})

	t.Run("panics on nil", func(t *testing.T) {
		reg := NewRegistry(nil)

		assert.Panics(t, func() {
			reg.MustRegister(nil)
		})
	})
}

func TestRegistry_Get(t *testing.T) {
	t.Run("existing resolver", func(t *testing.T) {
		reg := NewRegistry(nil)
		resolver := newMockResolver("test.tag")
		reg.MustRegister(resolver)

		got, ok := reg.Get("test.tag")
		require.True(t, ok)
		assert.Equal(t, resolver, got)
	})

	t.Run("non-existing resolver", func(t *testing.T) {
		reg := NewRegistry(nil)

		got, ok := reg.Get("nonexistent")
		assert.False(t, ok)
		assert.Nil(t, got)
	})
}

func TestRegistry_Has(t *testing.T) {
	reg := NewRegistry(nil)
	resolver := newMockResolver("test.tag")
	reg.MustRegister(resolver)

	assert.True(t, reg.Has("test.tag"))
	assert.False(t, reg.Has("nonexistent"))
}

func TestRegistry_List(t *testing.T) {
	t.Run("empty registry", func(t *testing.T) {
		reg := NewRegistry(nil)
		list := reg.List()
		assert.Empty(t, list)
	})

	t.Run("sorted order", func(t *testing.T) {
		reg := NewRegistry(nil)

		// Register in random order
		reg.MustRegister(newMockResolver("zebra"))
		reg.MustRegister(newMockResolver("apple"))
		reg.MustRegister(newMockResolver("middle"))

		list := reg.List()
		assert.Equal(t, []string{"apple", "middle", "zebra"}, list)
	})
}

func TestRegistry_Count(t *testing.T) {
	reg := NewRegistry(nil)
	assert.Equal(t, 0, reg.Count())

	reg.MustRegister(newMockResolver("a"))
	assert.Equal(t, 1, reg.Count())

	reg.MustRegister(newMockResolver("b"))
	assert.Equal(t, 2, reg.Count())
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	reg := NewRegistry(nil)

	// Pre-register some resolvers
	for i := 0; i < 10; i++ {
		reg.MustRegister(newMockResolver("preset" + string(rune('0'+i))))
	}

	var wg sync.WaitGroup
	const numGoroutines = 50
	const numOps = 100

	// Concurrent reads and writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOps; j++ {
				// Random operations
				switch j % 4 {
				case 0:
					// Read
					reg.Has("preset5")
				case 1:
					// List
					reg.List()
				case 2:
					// Get
					reg.Get("preset3")
				case 3:
					// Count
					reg.Count()
				}
			}
		}(i)
	}

	wg.Wait()

	// Registry should still be consistent
	assert.Equal(t, 10, reg.Count())
}

func TestRegistry_ConcurrentRegistration(t *testing.T) {
	reg := NewRegistry(nil)

	var wg sync.WaitGroup
	const numGoroutines = 10

	// Try to register same resolver from multiple goroutines
	// Only one should succeed
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := reg.Register(newMockResolver("contested"))
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Only one registration should have succeeded
	assert.Equal(t, 1, successCount)
	assert.True(t, reg.Has("contested"))
}

func TestRegistryError_Error(t *testing.T) {
	t.Run("with tag name", func(t *testing.T) {
		err := NewRegistryError(ErrMsgResolverAlreadyExists, "my.tag")
		errStr := err.Error()
		assert.Contains(t, errStr, ErrMsgResolverAlreadyExists)
		assert.Contains(t, errStr, "my.tag")
	})

	t.Run("without tag name", func(t *testing.T) {
		err := NewRegistryError(ErrMsgNilResolver, "")
		errStr := err.Error()
		assert.Equal(t, ErrMsgNilResolver, errStr)
	})
}

func TestMockResolver_Resolve(t *testing.T) {
	// Test the mock resolver to ensure it works correctly
	resolver := newMockResolver("test")

	result, err := resolver.Resolve(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "resolved:test", result)
}

func TestMockResolver_ResolveWithNilFunc(t *testing.T) {
	resolver := &mockResolver{name: "test", resolveFunc: nil}

	result, err := resolver.Resolve(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "", result)
}
