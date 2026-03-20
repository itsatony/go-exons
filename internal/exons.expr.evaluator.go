package internal

import (
	"context"
	"errors"
	"fmt"
)

// ExprEvaluator evaluates expression AST nodes
type ExprEvaluator struct {
	funcs   *FuncRegistry
	ctx     ContextAccessor
	evalCtx context.Context // Execution context for cancellation/timeout
}

// NewExprEvaluator creates a new expression evaluator
func NewExprEvaluator(funcs *FuncRegistry, ctx ContextAccessor) *ExprEvaluator {
	return &ExprEvaluator{
		funcs:   funcs,
		ctx:     ctx,
		evalCtx: context.Background(),
	}
}

// NewExprEvaluatorWithContext creates a new expression evaluator with cancellation context.
// The provided context is checked during evaluation to support timeouts and cancellation.
func NewExprEvaluatorWithContext(evalCtx context.Context, funcs *FuncRegistry, ctx ContextAccessor) *ExprEvaluator {
	if evalCtx == nil {
		evalCtx = context.Background()
	}
	return &ExprEvaluator{
		funcs:   funcs,
		ctx:     ctx,
		evalCtx: evalCtx,
	}
}

// Evaluate evaluates an expression and returns the result.
// It checks for context cancellation before evaluating each node to support
// timeouts and cancellation of long-running or deeply-nested expressions.
func (e *ExprEvaluator) Evaluate(node ExprNode) (any, error) {
	// Check for context cancellation before evaluation
	if err := e.checkContext(); err != nil {
		return nil, err
	}

	if node == nil {
		return nil, NewExprEvalError(ErrMsgExprNilNode, "")
	}

	switch n := node.(type) {
	case *LiteralNode:
		return n.Value, nil

	case *IdentifierNode:
		return e.evaluateIdentifier(n)

	case *UnaryNode:
		return e.evaluateUnary(n)

	case *BinaryNode:
		return e.evaluateBinary(n)

	case *CallNode:
		return e.evaluateCall(n)

	default:
		return nil, NewExprEvalError(ErrMsgExprUnknownNodeType, fmt.Sprintf("%T", node))
	}
}

// checkContext checks if the evaluation context has been cancelled or timed out.
func (e *ExprEvaluator) checkContext() error {
	if e.evalCtx == nil {
		return nil
	}
	select {
	case <-e.evalCtx.Done():
		err := e.evalCtx.Err()
		if errors.Is(err, context.Canceled) {
			return NewExprEvalError(ErrMsgExprCancelled, "")
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return NewExprEvalError(ErrMsgExprTimeout, "")
		}
		return NewExprEvalError(ErrMsgExprContextDone, err.Error())
	default:
		return nil
	}
}

// EvaluateBool evaluates an expression and coerces the result to a boolean
func (e *ExprEvaluator) EvaluateBool(node ExprNode) (bool, error) {
	result, err := e.Evaluate(node)
	if err != nil {
		return false, err
	}
	return isTruthy(result), nil
}

// evaluateIdentifier looks up a variable from the context
func (e *ExprEvaluator) evaluateIdentifier(node *IdentifierNode) (any, error) {
	if e.ctx == nil {
		return nil, NewExprEvalError(ErrMsgExprNoContext, node.Name)
	}

	val, found := e.ctx.Get(node.Name)
	if !found {
		return nil, nil // Return nil for missing variables (not an error)
	}
	return val, nil
}

// evaluateUnary evaluates a unary operation
func (e *ExprEvaluator) evaluateUnary(node *UnaryNode) (any, error) {
	right, err := e.Evaluate(node.Right)
	if err != nil {
		return nil, err
	}

	switch node.Op {
	case ExprTokenTypeNot:
		return !isTruthy(right), nil
	default:
		return nil, NewExprEvalError(ErrMsgExprUnknownOperator, string(node.Op))
	}
}

// evaluateBinary evaluates a binary operation
func (e *ExprEvaluator) evaluateBinary(node *BinaryNode) (any, error) {
	// Short-circuit evaluation for logical operators
	if node.Op == ExprTokenTypeAnd {
		left, err := e.Evaluate(node.Left)
		if err != nil {
			return nil, err
		}
		if !isTruthy(left) {
			return false, nil // Short-circuit: false && x = false
		}
		right, err := e.Evaluate(node.Right)
		if err != nil {
			return nil, err
		}
		return isTruthy(right), nil
	}

	if node.Op == ExprTokenTypeOr {
		left, err := e.Evaluate(node.Left)
		if err != nil {
			return nil, err
		}
		if isTruthy(left) {
			return true, nil // Short-circuit: true || x = true
		}
		right, err := e.Evaluate(node.Right)
		if err != nil {
			return nil, err
		}
		return isTruthy(right), nil
	}

	// Evaluate both sides for other operators
	left, err := e.Evaluate(node.Left)
	if err != nil {
		return nil, err
	}

	right, err := e.Evaluate(node.Right)
	if err != nil {
		return nil, err
	}

	switch node.Op {
	case ExprTokenTypeEq:
		return compareEqual(left, right), nil
	case ExprTokenTypeNeq:
		return !compareEqual(left, right), nil
	case ExprTokenTypeLt:
		return compareLess(left, right)
	case ExprTokenTypeGt:
		return compareGreater(left, right)
	case ExprTokenTypeLte:
		result, err := compareGreater(left, right)
		if err != nil {
			return nil, err
		}
		return !result, nil
	case ExprTokenTypeGte:
		result, err := compareLess(left, right)
		if err != nil {
			return nil, err
		}
		return !result, nil
	default:
		return nil, NewExprEvalError(ErrMsgExprUnknownOperator, string(node.Op))
	}
}

// evaluateCall evaluates a function call
func (e *ExprEvaluator) evaluateCall(node *CallNode) (any, error) {
	if e.funcs == nil {
		return nil, NewExprEvalError(ErrMsgExprNoFuncRegistry, node.Name)
	}

	// Evaluate arguments
	args := make([]any, len(node.Args))
	for i, argNode := range node.Args {
		val, err := e.Evaluate(argNode)
		if err != nil {
			return nil, err
		}
		args[i] = val
	}

	// Call the function
	return e.funcs.Call(node.Name, args)
}

// Comparison helper functions

// compareEqual checks if two values are equal
func compareEqual(a, b any) bool {
	// Handle nil cases
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Try numeric comparison
	aNum, aIsNum := toNumber(a)
	bNum, bIsNum := toNumber(b)
	if aIsNum && bIsNum {
		return aNum == bNum
	}

	// Try string comparison
	aStr, aIsStr := toString(a)
	bStr, bIsStr := toString(b)
	if aIsStr && bIsStr {
		return aStr == bStr
	}

	// Try boolean comparison
	aBool, aIsBool := a.(bool)
	bBool, bIsBool := b.(bool)
	if aIsBool && bIsBool {
		return aBool == bBool
	}

	// Fallback to direct comparison
	return a == b
}

// compareLess checks if a < b
func compareLess(a, b any) (bool, error) {
	// Try numeric comparison
	aNum, aIsNum := toNumber(a)
	bNum, bIsNum := toNumber(b)
	if aIsNum && bIsNum {
		return aNum < bNum, nil
	}

	// Try string comparison
	aStr, aIsStr := toString(a)
	bStr, bIsStr := toString(b)
	if aIsStr && bIsStr {
		return aStr < bStr, nil
	}

	return false, NewTypeComparisonError(fmt.Sprintf("%T", a), fmt.Sprintf("%T", b))
}

// compareGreater checks if a > b
func compareGreater(a, b any) (bool, error) {
	// Try numeric comparison
	aNum, aIsNum := toNumber(a)
	bNum, bIsNum := toNumber(b)
	if aIsNum && bIsNum {
		return aNum > bNum, nil
	}

	// Try string comparison
	aStr, aIsStr := toString(a)
	bStr, bIsStr := toString(b)
	if aIsStr && bIsStr {
		return aStr > bStr, nil
	}

	return false, NewTypeComparisonError(fmt.Sprintf("%T", a), fmt.Sprintf("%T", b))
}

// toNumber attempts to convert a value to float64
func toNumber(v any) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case float64:
		return val, true
	case float32:
		return float64(val), true
	default:
		return 0, false
	}
}

// ExprEvalError represents an expression evaluation error
type ExprEvalError struct {
	Message  string
	Detail   string
	Metadata map[string]string
}

// NewExprEvalError creates a new expression evaluation error
func NewExprEvalError(message, detail string) *ExprEvalError {
	return &ExprEvalError{
		Message:  message,
		Detail:   detail,
		Metadata: make(map[string]string),
	}
}

// WithMetadata adds a metadata key-value pair and returns the error for chaining.
func (e *ExprEvalError) WithMetadata(key, value string) *ExprEvalError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}

// Error implements the error interface
func (e *ExprEvalError) Error() string {
	var result string
	if e.Detail != "" {
		result = fmt.Sprintf("%s: %s", e.Message, e.Detail)
	} else {
		result = e.Message
	}
	if len(e.Metadata) > 0 {
		for k, v := range e.Metadata {
			result += fmt.Sprintf(" [%s=%s]", k, v)
		}
	}
	return result
}

// NewTypeComparisonError creates an error for type mismatch in comparisons.
func NewTypeComparisonError(typeA, typeB string) *ExprEvalError {
	return NewExprEvalError(ErrMsgExprTypeMismatch, "").
		WithMetadata(MetaKeyFromType, typeA).
		WithMetadata(MetaKeyToType, typeB)
}

// Expression evaluator error messages
const (
	ErrMsgExprNilNode         = "nil expression node"
	ErrMsgExprUnknownNodeType = "unknown expression node type"
	ErrMsgExprNoContext       = "no context available for variable lookup"
	ErrMsgExprUnknownOperator = "unknown operator"
	ErrMsgExprNoFuncRegistry  = "no function registry available"
	ErrMsgExprTypeMismatch    = "type mismatch in comparison"
	ErrMsgExprCancelled       = "expression evaluation cancelled"
	ErrMsgExprTimeout         = "expression evaluation timed out"
	ErrMsgExprContextDone     = "expression evaluation context done"
)

// EvaluateExpression is a convenience function that parses and evaluates an expression string.
// It uses a background context. For cancellation/timeout support, use EvaluateExpressionWithContext.
func EvaluateExpression(expr string, funcs *FuncRegistry, ctx ContextAccessor) (any, error) {
	return EvaluateExpressionWithContext(context.Background(), expr, funcs, ctx)
}

// EvaluateExpressionWithContext parses and evaluates an expression string with context support.
// The context is checked during evaluation to support timeouts and cancellation.
func EvaluateExpressionWithContext(evalCtx context.Context, expr string, funcs *FuncRegistry, ctx ContextAccessor) (any, error) {
	node, err := ParseExpression(expr)
	if err != nil {
		return nil, err
	}

	evaluator := NewExprEvaluatorWithContext(evalCtx, funcs, ctx)
	return evaluator.Evaluate(node)
}

// EvaluateExpressionBool is a convenience function that parses and evaluates an expression as a boolean.
// It uses a background context. For cancellation/timeout support, use EvaluateExpressionBoolWithContext.
func EvaluateExpressionBool(expr string, funcs *FuncRegistry, ctx ContextAccessor) (bool, error) {
	return EvaluateExpressionBoolWithContext(context.Background(), expr, funcs, ctx)
}

// EvaluateExpressionBoolWithContext parses and evaluates an expression as a boolean with context support.
// The context is checked during evaluation to support timeouts and cancellation.
func EvaluateExpressionBoolWithContext(evalCtx context.Context, expr string, funcs *FuncRegistry, ctx ContextAccessor) (bool, error) {
	node, err := ParseExpression(expr)
	if err != nil {
		return false, err
	}

	evaluator := NewExprEvaluatorWithContext(evalCtx, funcs, ctx)
	return evaluator.EvaluateBool(node)
}
