package internal

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExprEvaluator_Evaluate_Literals(t *testing.T) {
	funcs := NewFuncRegistry()
	RegisterBuiltinFuncs(funcs)
	ctx := newMockContextAccessor(nil)

	tests := []struct {
		name     string
		input    string
		expected any
	}{
		{"string", `"hello"`, "hello"},
		{"number", "42", 42.0},
		{"float", "3.14", 3.14},
		{"bool true", "true", true},
		{"bool false", "false", false},
		{"nil", "nil", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateExpression(tt.input, funcs, ctx)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExprEvaluator_Evaluate_Identifier(t *testing.T) {
	funcs := NewFuncRegistry()
	RegisterBuiltinFuncs(funcs)
	ctx := newMockContextAccessor(map[string]any{
		"name":     "Alice",
		"count":    42,
		"isActive": true,
	})

	tests := []struct {
		name     string
		input    string
		expected any
	}{
		{"string var", "name", "Alice"},
		{"number var", "count", 42},
		{"bool var", "isActive", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateExpression(tt.input, funcs, ctx)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExprEvaluator_Evaluate_DottedIdentifier(t *testing.T) {
	funcs := NewFuncRegistry()
	RegisterBuiltinFuncs(funcs)
	// Note: The mock context accessor doesn't support dotted path resolution.
	// It does direct key lookup. So we use keys that match the identifier strings.
	ctx := newMockContextAccessor(map[string]any{
		"user.name":           "Bob",
		"user.profile.avatar": "photo.jpg",
	})

	tests := []struct {
		input    string
		expected any
	}{
		{"user.name", "Bob"},
		{"user.profile.avatar", "photo.jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := EvaluateExpression(tt.input, funcs, ctx)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExprEvaluator_Evaluate_UnaryNot(t *testing.T) {
	funcs := NewFuncRegistry()
	RegisterBuiltinFuncs(funcs)

	tests := []struct {
		name     string
		input    string
		data     map[string]any
		expected bool
	}{
		{"not true", "!true", nil, false},
		{"not false", "!false", nil, true},
		{"not truthy string", `!"hello"`, nil, false},
		{"not falsy string", `!""`, nil, true},
		{"not nil", "!nil", nil, true},
		{"not var true", "!isActive", map[string]any{"isActive": true}, false},
		{"not var false", "!isActive", map[string]any{"isActive": false}, true},
		{"double not", "!!true", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newMockContextAccessor(tt.data)
			result, err := EvaluateExpressionBool(tt.input, funcs, ctx)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExprEvaluator_Evaluate_Comparison(t *testing.T) {
	funcs := NewFuncRegistry()
	RegisterBuiltinFuncs(funcs)
	ctx := newMockContextAccessor(map[string]any{
		"x": 10,
		"y": 20,
		"s": "hello",
	})

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"number equal", "x == 10", true},
		{"number not equal", "x != 10", false},
		{"number less than true", "x < y", true},
		{"number less than false", "y < x", false},
		{"number greater than true", "y > x", true},
		{"number greater than false", "x > y", false},
		{"number lte true", "x <= 10", true},
		{"number lte false", "y <= 10", false},
		{"number gte true", "y >= 20", true},
		{"number gte false", "x >= 20", false},
		{"string equal", `s == "hello"`, true},
		{"string not equal", `s != "hello"`, false},
		{"compare nil", "x != nil", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateExpressionBool(tt.input, funcs, ctx)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExprEvaluator_Evaluate_LogicalAnd(t *testing.T) {
	funcs := NewFuncRegistry()
	RegisterBuiltinFuncs(funcs)
	ctx := newMockContextAccessor(nil)

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"true and true", "true && true", true},
		{"true and false", "true && false", false},
		{"false and true", "false && true", false},
		{"false and false", "false && false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateExpressionBool(tt.input, funcs, ctx)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExprEvaluator_Evaluate_LogicalOr(t *testing.T) {
	funcs := NewFuncRegistry()
	RegisterBuiltinFuncs(funcs)
	ctx := newMockContextAccessor(nil)

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"true or true", "true || true", true},
		{"true or false", "true || false", true},
		{"false or true", "false || true", true},
		{"false or false", "false || false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateExpressionBool(tt.input, funcs, ctx)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExprEvaluator_Evaluate_FunctionCalls(t *testing.T) {
	funcs := NewFuncRegistry()
	RegisterBuiltinFuncs(funcs)
	ctx := newMockContextAccessor(map[string]any{
		"items": []any{1, 2, 3},
		"name":  "  Alice  ",
		"roles": []any{"user", "admin"},
	})

	tests := []struct {
		name     string
		input    string
		expected any
	}{
		{"len array", "len(items)", 3},
		{"trim string", "trim(name)", "Alice"},
		{"upper", `upper("hello")`, "HELLO"},
		{"lower", `lower("HELLO")`, "hello"},
		{"contains true", `contains(roles, "admin")`, true},
		{"contains false", `contains(roles, "superuser")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateExpression(tt.input, funcs, ctx)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExprEvaluator_Evaluate_NestedFunctionCalls(t *testing.T) {
	funcs := NewFuncRegistry()
	RegisterBuiltinFuncs(funcs)
	ctx := newMockContextAccessor(map[string]any{
		"name": "  Alice  ",
	})

	result, err := EvaluateExpression("upper(trim(name))", funcs, ctx)

	require.NoError(t, err)
	assert.Equal(t, "ALICE", result)
}

func TestExprEvaluator_Evaluate_ComplexExpression(t *testing.T) {
	funcs := NewFuncRegistry()
	RegisterBuiltinFuncs(funcs)
	ctx := newMockContextAccessor(map[string]any{
		"items":   []any{1, 2, 3},
		"isAdmin": true,
	})

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			"len and bool",
			"len(items) > 0 && isAdmin",
			true,
		},
		{
			"comparison chain",
			"len(items) >= 1 && len(items) <= 10",
			true,
		},
		{
			"mixed operators",
			"isAdmin || len(items) == 0",
			true,
		},
		{
			"negation and comparison",
			"!isAdmin || len(items) > 5",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateExpressionBool(tt.input, funcs, ctx)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExprEvaluator_Evaluate_Truthiness(t *testing.T) {
	funcs := NewFuncRegistry()
	RegisterBuiltinFuncs(funcs)

	tests := []struct {
		name     string
		data     map[string]any
		input    string
		expected bool
	}{
		{"nil is falsy", nil, "nil", false},
		{"true is truthy", nil, "true", true},
		{"false is falsy", nil, "false", false},
		{"non-empty string truthy", map[string]any{"s": "hello"}, "s", true},
		{"empty string falsy", map[string]any{"s": ""}, "s", false},
		{"positive number truthy", map[string]any{"n": 1}, "n", true},
		{"zero is falsy", map[string]any{"n": 0}, "n", false},
		{"negative number truthy", map[string]any{"n": -1}, "n", true},
		{"non-empty array truthy", map[string]any{"a": []any{1}}, "a", true},
		{"empty array falsy", map[string]any{"a": []any{}}, "a", false},
		{"non-empty map truthy", map[string]any{"m": map[string]any{"k": "v"}}, "m", true},
		{"empty map falsy", map[string]any{"m": map[string]any{}}, "m", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newMockContextAccessor(tt.data)
			result, err := EvaluateExpressionBool(tt.input, funcs, ctx)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExprEvaluator_Evaluate_Error_UnknownFunction(t *testing.T) {
	funcs := NewFuncRegistry()
	ctx := newMockContextAccessor(nil)

	_, err := EvaluateExpression("unknownFunc()", funcs, ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgFuncNotFound)
}

func TestExprEvaluator_Evaluate_Error_UndefinedVariable(t *testing.T) {
	funcs := NewFuncRegistry()
	RegisterBuiltinFuncs(funcs)
	ctx := newMockContextAccessor(nil)

	result, err := EvaluateExpression("undefined", funcs, ctx)

	// Undefined variables return nil, not an error
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestExprEvaluator_EvaluateBool(t *testing.T) {
	funcs := NewFuncRegistry()
	RegisterBuiltinFuncs(funcs)
	ctx := newMockContextAccessor(nil)

	t.Run("returns bool for bool expression", func(t *testing.T) {
		result, err := EvaluateExpressionBool("true && false", funcs, ctx)
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("converts string to bool", func(t *testing.T) {
		result, err := EvaluateExpressionBool(`"hello"`, funcs, ctx)
		require.NoError(t, err)
		assert.True(t, result) // non-empty string is truthy
	})
}

func TestExprEvalError_Error(t *testing.T) {
	err := NewExprEvalError("test error", "detail")
	assert.Equal(t, "test error: detail", err.Error())

	err2 := NewExprEvalError("simple error", "")
	assert.Equal(t, "simple error", err2.Error())
}

func TestEvaluateExpressionBool_ParseError(t *testing.T) {
	funcs := NewFuncRegistry()
	ctx := newMockContextAccessor(nil)

	_, err := EvaluateExpressionBool("(unclosed", funcs, ctx)

	require.Error(t, err)
}

func TestEvaluateExpression_ParseError(t *testing.T) {
	funcs := NewFuncRegistry()
	ctx := newMockContextAccessor(nil)

	_, err := EvaluateExpression("@@invalid", funcs, ctx)

	require.Error(t, err)
}

func TestEvaluateExpressionWithContext_Cancellation(t *testing.T) {
	funcs := NewFuncRegistry()
	RegisterBuiltinFuncs(funcs)
	ctx := newMockContextAccessor(map[string]any{
		"value": 42,
	})

	t.Run("cancelled context returns error", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := EvaluateExpressionWithContext(cancelledCtx, "value > 10", funcs, ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgExprCancelled)
	})

	t.Run("timed out context returns error", func(t *testing.T) {
		timedOutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond) // Wait for timeout

		_, err := EvaluateExpressionWithContext(timedOutCtx, "value > 10", funcs, ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgExprTimeout)
	})

	t.Run("valid context works normally", func(t *testing.T) {
		validCtx := context.Background()

		result, err := EvaluateExpressionWithContext(validCtx, "value > 10", funcs, ctx)

		require.NoError(t, err)
		assert.Equal(t, true, result)
	})
}

func TestEvaluateExpressionBoolWithContext_Cancellation(t *testing.T) {
	funcs := NewFuncRegistry()
	RegisterBuiltinFuncs(funcs)
	ctx := newMockContextAccessor(map[string]any{
		"flag": true,
	})

	t.Run("cancelled context returns error", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := EvaluateExpressionBoolWithContext(cancelledCtx, "flag", funcs, ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgExprCancelled)
	})

	t.Run("valid context works normally", func(t *testing.T) {
		result, err := EvaluateExpressionBoolWithContext(context.Background(), "flag", funcs, ctx)

		require.NoError(t, err)
		assert.True(t, result)
	})
}

func TestNewExprEvaluatorWithContext_NilContext(t *testing.T) {
	funcs := NewFuncRegistry()
	ctx := newMockContextAccessor(nil)

	// Should not panic with nil context - pass context.TODO for the test
	evaluator := NewExprEvaluatorWithContext(context.TODO(), funcs, ctx)
	require.NotNil(t, evaluator)

	// Should work normally with nil context (uses background)
	node, err := ParseExpression("true")
	require.NoError(t, err)

	result, err := evaluator.Evaluate(node)
	require.NoError(t, err)
	assert.Equal(t, true, result)
}

func TestExprEvaluator_CheckContext(t *testing.T) {
	funcs := NewFuncRegistry()
	ctx := newMockContextAccessor(nil)

	t.Run("nil context is ok", func(t *testing.T) {
		evaluator := &ExprEvaluator{
			funcs:   funcs,
			ctx:     ctx,
			evalCtx: nil,
		}

		err := evaluator.checkContext()
		require.NoError(t, err)
	})

	t.Run("non-cancelled context is ok", func(t *testing.T) {
		evaluator := NewExprEvaluatorWithContext(context.Background(), funcs, ctx)

		err := evaluator.checkContext()
		require.NoError(t, err)
	})

	t.Run("cancelled context returns error", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		evaluator := NewExprEvaluatorWithContext(cancelledCtx, funcs, ctx)

		err := evaluator.checkContext()
		require.Error(t, err)
	})
}
