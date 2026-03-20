package exons

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// HookRegistry Creation
// =============================================================================

func TestNewHookRegistry(t *testing.T) {
	registry := NewHookRegistry()
	assert.NotNil(t, registry)
}

// =============================================================================
// Register + Fire Hooks
// =============================================================================

func TestHookRegistry_Register(t *testing.T) {
	t.Run("register and run before hook", func(t *testing.T) {
		registry := NewHookRegistry()
		called := false
		registry.Register(HookBeforeExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
			called = true
			assert.Equal(t, HookBeforeExecute, point)
			return nil
		})

		err := registry.Run(context.Background(), HookBeforeExecute, NewHookData("execute", "test"))
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("register and run after hook", func(t *testing.T) {
		registry := NewHookRegistry()
		called := false
		registry.Register(HookAfterExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
			called = true
			return nil
		})

		err := registry.Run(context.Background(), HookAfterExecute, NewHookData("execute", "test"))
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("run with no hooks", func(t *testing.T) {
		registry := NewHookRegistry()
		err := registry.Run(context.Background(), HookBeforeLoad, NewHookData("load", "test"))
		require.NoError(t, err)
	})

	t.Run("multiple hooks same point", func(t *testing.T) {
		registry := NewHookRegistry()
		order := make([]int, 0)
		registry.Register(HookAfterExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
			order = append(order, 1)
			return nil
		})
		registry.Register(HookAfterExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
			order = append(order, 2)
			return nil
		})

		err := registry.Run(context.Background(), HookAfterExecute, NewHookData("execute", "test"))
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2}, order)
	})
}

// =============================================================================
// Before Hooks Abort on Error
// =============================================================================

func TestHookRegistry_BeforeHooksAbort(t *testing.T) {
	registry := NewHookRegistry()
	hookErr := fmt.Errorf("abort operation")

	secondCalled := false
	registry.Register(HookBeforeExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
		return hookErr
	})
	registry.Register(HookBeforeExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
		secondCalled = true
		return nil
	})

	err := registry.Run(context.Background(), HookBeforeExecute, NewHookData("execute", "test"))
	assert.Equal(t, hookErr, err)
	assert.False(t, secondCalled, "second hook should not be called after first error in before hook")
}

// =============================================================================
// After Hooks Continue on Error
// =============================================================================

func TestHookRegistry_AfterHooksContinue(t *testing.T) {
	registry := NewHookRegistry()

	secondCalled := false
	registry.Register(HookAfterExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
		return fmt.Errorf("non-fatal error")
	})
	registry.Register(HookAfterExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
		secondCalled = true
		return nil
	})

	err := registry.Run(context.Background(), HookAfterExecute, NewHookData("execute", "test"))
	// After hooks don't return errors from Run
	require.NoError(t, err)
	assert.True(t, secondCalled, "second hook should be called even after first error in after hook")
}

// =============================================================================
// RunWithErrors
// =============================================================================

func TestHookRegistry_RunWithErrors(t *testing.T) {
	t.Run("collects all errors", func(t *testing.T) {
		registry := NewHookRegistry()
		registry.Register(HookAfterExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
			return fmt.Errorf("error 1")
		})
		registry.Register(HookAfterExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
			return fmt.Errorf("error 2")
		})
		registry.Register(HookAfterExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
			return nil // no error
		})

		errs := registry.RunWithErrors(context.Background(), HookAfterExecute, NewHookData("execute", "test"))
		assert.Len(t, errs, 2)
	})

	t.Run("returns nil for no hooks", func(t *testing.T) {
		registry := NewHookRegistry()
		errs := registry.RunWithErrors(context.Background(), HookBeforeLoad, NewHookData("load", "test"))
		assert.Nil(t, errs)
	})

	t.Run("returns nil for no errors", func(t *testing.T) {
		registry := NewHookRegistry()
		registry.Register(HookAfterSave, func(ctx context.Context, point HookPoint, data *HookData) error {
			return nil
		})
		errs := registry.RunWithErrors(context.Background(), HookAfterSave, NewHookData("save", "test"))
		assert.Nil(t, errs)
	})
}

// =============================================================================
// RegisterMultiple
// =============================================================================

func TestHookRegistry_RegisterMultiple(t *testing.T) {
	registry := NewHookRegistry()
	called := make(map[HookPoint]bool)
	hook := func(ctx context.Context, point HookPoint, data *HookData) error {
		called[point] = true
		return nil
	}

	registry.RegisterMultiple(hook, HookBeforeLoad, HookAfterLoad, HookBeforeExecute)

	err := registry.Run(context.Background(), HookBeforeLoad, NewHookData("load", "test"))
	require.NoError(t, err)
	assert.True(t, called[HookBeforeLoad])

	err = registry.Run(context.Background(), HookAfterLoad, NewHookData("load", "test"))
	require.NoError(t, err)
	assert.True(t, called[HookAfterLoad])

	err = registry.Run(context.Background(), HookBeforeExecute, NewHookData("execute", "test"))
	require.NoError(t, err)
	assert.True(t, called[HookBeforeExecute])
}

// =============================================================================
// Clear, ClearAll
// =============================================================================

func TestHookRegistry_Clear(t *testing.T) {
	t.Run("clear specific point", func(t *testing.T) {
		registry := NewHookRegistry()
		registry.Register(HookBeforeLoad, func(ctx context.Context, point HookPoint, data *HookData) error {
			return nil
		})
		registry.Register(HookAfterLoad, func(ctx context.Context, point HookPoint, data *HookData) error {
			return nil
		})

		assert.True(t, registry.HasHooks(HookBeforeLoad))
		assert.True(t, registry.HasHooks(HookAfterLoad))

		registry.Clear(HookBeforeLoad)
		assert.False(t, registry.HasHooks(HookBeforeLoad))
		assert.True(t, registry.HasHooks(HookAfterLoad))
	})

	t.Run("clear all", func(t *testing.T) {
		registry := NewHookRegistry()
		registry.Register(HookBeforeLoad, func(ctx context.Context, point HookPoint, data *HookData) error {
			return nil
		})
		registry.Register(HookAfterLoad, func(ctx context.Context, point HookPoint, data *HookData) error {
			return nil
		})

		registry.ClearAll()
		assert.False(t, registry.HasHooks(HookBeforeLoad))
		assert.False(t, registry.HasHooks(HookAfterLoad))
	})
}

// =============================================================================
// Count, HasHooks
// =============================================================================

func TestHookRegistry_Count(t *testing.T) {
	registry := NewHookRegistry()
	assert.Equal(t, 0, registry.Count(HookBeforeExecute))

	registry.Register(HookBeforeExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
		return nil
	})
	assert.Equal(t, 1, registry.Count(HookBeforeExecute))

	registry.Register(HookBeforeExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
		return nil
	})
	assert.Equal(t, 2, registry.Count(HookBeforeExecute))
}

func TestHookRegistry_HasHooks(t *testing.T) {
	registry := NewHookRegistry()
	assert.False(t, registry.HasHooks(HookBeforeValidate))

	registry.Register(HookBeforeValidate, func(ctx context.Context, point HookPoint, data *HookData) error {
		return nil
	})
	assert.True(t, registry.HasHooks(HookBeforeValidate))
}

// =============================================================================
// LoggingHook Factory
// =============================================================================

func TestLoggingHook(t *testing.T) {
	var logged HookPoint
	var loggedData *HookData

	hook := LoggingHook(func(point HookPoint, data *HookData) {
		logged = point
		loggedData = data
	})

	hookData := NewHookData("execute", "test-template")
	err := hook(context.Background(), HookBeforeExecute, hookData)
	require.NoError(t, err)
	assert.Equal(t, HookBeforeExecute, logged)
	assert.Equal(t, "test-template", loggedData.TemplateName)
}

// =============================================================================
// TimingHook Factory
// =============================================================================

func TestTimingHook(t *testing.T) {
	hook, getElapsed := TimingHook()

	data := NewHookData("execute", "test")

	// Simulate before hook
	err := hook(context.Background(), HookBeforeExecute, data)
	require.NoError(t, err)

	// Simulate after hook
	err = hook(context.Background(), HookAfterExecute, data)
	require.NoError(t, err)

	elapsed := getElapsed(data)
	assert.True(t, elapsed >= 0, "elapsed time should be non-negative")
}

func TestTimingHook_NoStart(t *testing.T) {
	_, getElapsed := TimingHook()
	data := NewHookData("execute", "test")
	// Without calling the before hook, elapsed should be 0
	elapsed := getElapsed(data)
	assert.Equal(t, int64(0), elapsed)
}

// =============================================================================
// Thread Safety
// =============================================================================

func TestHookRegistry_ThreadSafety(t *testing.T) {
	registry := NewHookRegistry()
	var wg sync.WaitGroup

	// Concurrent registrations and runs
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			registry.Register(HookBeforeExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
				return nil
			})
		}()
		go func() {
			defer wg.Done()
			_ = registry.Run(context.Background(), HookBeforeExecute, NewHookData("execute", "test"))
		}()
	}
	wg.Wait()
}

// =============================================================================
// HookData Tests
// =============================================================================

func TestHookData(t *testing.T) {
	t.Run("new hook data", func(t *testing.T) {
		data := NewHookData("save", "my-template")
		assert.Equal(t, "save", data.OperationName)
		assert.Equal(t, "my-template", data.TemplateName)
		assert.NotNil(t, data.Metadata)
	})

	t.Run("with execution data", func(t *testing.T) {
		data := NewHookData("execute", "test").WithExecutionData(map[string]any{"key": "val"})
		assert.Equal(t, "val", data.ExecutionData["key"])
	})

	t.Run("with result", func(t *testing.T) {
		data := NewHookData("execute", "test").WithResult("output")
		assert.Equal(t, "output", data.Result)
	})

	t.Run("with error", func(t *testing.T) {
		err := fmt.Errorf("test error")
		data := NewHookData("execute", "test").WithError(err)
		assert.Equal(t, err, data.Error)
	})

	t.Run("with validation result", func(t *testing.T) {
		vr := &ValidationResult{}
		data := NewHookData("validate", "test").WithValidationResult(vr)
		assert.Equal(t, vr, data.ValidationResult)
	})

	t.Run("metadata set and get", func(t *testing.T) {
		data := NewHookData("execute", "test")
		data.SetMetadata("key", "value")
		val, ok := data.GetMetadata("key")
		assert.True(t, ok)
		assert.Equal(t, "value", val)
	})

	t.Run("metadata get missing", func(t *testing.T) {
		data := NewHookData("execute", "test")
		val, ok := data.GetMetadata("missing")
		assert.False(t, ok)
		assert.Nil(t, val)
	})

	t.Run("metadata set on nil map", func(t *testing.T) {
		data := &HookData{}
		data.SetMetadata("key", "value")
		val, ok := data.GetMetadata("key")
		assert.True(t, ok)
		assert.Equal(t, "value", val)
	})

	t.Run("metadata get on nil map", func(t *testing.T) {
		data := &HookData{}
		val, ok := data.GetMetadata("key")
		assert.False(t, ok)
		assert.Nil(t, val)
	})
}

// =============================================================================
// HookError Tests
// =============================================================================

func TestHookError(t *testing.T) {
	t.Run("basic hook error", func(t *testing.T) {
		err := NewHookError(HookBeforeExecute, fmt.Errorf("cause"))
		assert.Contains(t, err.Error(), ErrMsgHookFailed)
		assert.Contains(t, err.Error(), string(HookBeforeExecute))
		assert.Contains(t, err.Error(), "cause")
	})

	t.Run("hook error without cause", func(t *testing.T) {
		err := NewHookError(HookAfterSave, nil)
		assert.Contains(t, err.Error(), ErrMsgHookFailed)
		assert.Contains(t, err.Error(), string(HookAfterSave))
		assert.Nil(t, err.Unwrap())
	})

	t.Run("hook error unwrap", func(t *testing.T) {
		cause := fmt.Errorf("root cause")
		err := NewHookError(HookBeforeLoad, cause)
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("hook error empty point", func(t *testing.T) {
		err := &HookError{Message: ErrMsgHookFailed}
		assert.Equal(t, ErrMsgHookFailed, err.Error())
	})
}

// =============================================================================
// HookPoint Constants
// =============================================================================

func TestHookPointConstants(t *testing.T) {
	assert.Equal(t, HookPoint("before_load"), HookBeforeLoad)
	assert.Equal(t, HookPoint("after_load"), HookAfterLoad)
	assert.Equal(t, HookPoint("before_execute"), HookBeforeExecute)
	assert.Equal(t, HookPoint("after_execute"), HookAfterExecute)
	assert.Equal(t, HookPoint("before_save"), HookBeforeSave)
	assert.Equal(t, HookPoint("after_save"), HookAfterSave)
	assert.Equal(t, HookPoint("before_delete"), HookBeforeDelete)
	assert.Equal(t, HookPoint("after_delete"), HookAfterDelete)
	assert.Equal(t, HookPoint("before_validate"), HookBeforeValidate)
	assert.Equal(t, HookPoint("after_validate"), HookAfterValidate)
}

// =============================================================================
// isBeforeHook Tests
// =============================================================================

func TestIsBeforeHook(t *testing.T) {
	beforePoints := []HookPoint{
		HookBeforeLoad, HookBeforeExecute, HookBeforeSave, HookBeforeDelete, HookBeforeValidate,
	}
	afterPoints := []HookPoint{
		HookAfterLoad, HookAfterExecute, HookAfterSave, HookAfterDelete, HookAfterValidate,
	}

	for _, p := range beforePoints {
		assert.True(t, isBeforeHook(p), "expected %s to be a before hook", p)
	}
	for _, p := range afterPoints {
		assert.False(t, isBeforeHook(p), "expected %s to NOT be a before hook", p)
	}
}
