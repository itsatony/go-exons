package internal

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExprParser_Parse_Literal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		literal any
		kind    LiteralKind
	}{
		{"string", `"hello"`, "hello", LiteralKindString},
		{"number", "42", 42.0, LiteralKindNumber},
		{"bool true", "true", true, LiteralKindBool},
		{"bool false", "false", false, LiteralKindBool},
		{"nil", "nil", nil, LiteralKindNil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ParseExpression(tt.input)

			require.NoError(t, err)
			literal, ok := node.(*LiteralNode)
			require.True(t, ok)
			assert.Equal(t, tt.literal, literal.Value)
			assert.Equal(t, tt.kind, literal.Kind)
		})
	}
}

func TestExprParser_Parse_Identifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"foo", "foo"},
		{"user.name", "user.name"},
		{"user.profile.avatar", "user.profile.avatar"},
		{"_private", "_private"},
		{"camelCase", "camelCase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := ParseExpression(tt.input)

			require.NoError(t, err)
			ident, ok := node.(*IdentifierNode)
			require.True(t, ok)
			assert.Equal(t, tt.expected, ident.Name)
		})
	}
}

func TestExprParser_Parse_UnaryNot(t *testing.T) {
	node, err := ParseExpression("!isAdmin")

	require.NoError(t, err)
	unary, ok := node.(*UnaryNode)
	require.True(t, ok)
	assert.Equal(t, ExprTokenTypeNot, unary.Op)

	ident, ok := unary.Right.(*IdentifierNode)
	require.True(t, ok)
	assert.Equal(t, "isAdmin", ident.Name)
}

func TestExprParser_Parse_DoubleNot(t *testing.T) {
	node, err := ParseExpression("!!flag")

	require.NoError(t, err)
	outer, ok := node.(*UnaryNode)
	require.True(t, ok)
	assert.Equal(t, ExprTokenTypeNot, outer.Op)

	inner, ok := outer.Right.(*UnaryNode)
	require.True(t, ok)
	assert.Equal(t, ExprTokenTypeNot, inner.Op)
}

func TestExprParser_Parse_Comparison(t *testing.T) {
	tests := []struct {
		name  string
		input string
		op    ExprTokenType
	}{
		{"equals", `x == 1`, ExprTokenTypeEq},
		{"not equals", `x != 1`, ExprTokenTypeNeq},
		{"less than", `x < 1`, ExprTokenTypeLt},
		{"greater than", `x > 1`, ExprTokenTypeGt},
		{"less than or equal", `x <= 1`, ExprTokenTypeLte},
		{"greater than or equal", `x >= 1`, ExprTokenTypeGte},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ParseExpression(tt.input)

			require.NoError(t, err)
			binary, ok := node.(*BinaryNode)
			require.True(t, ok)
			assert.Equal(t, tt.op, binary.Op)
		})
	}
}

func TestExprParser_Parse_LogicalAnd(t *testing.T) {
	node, err := ParseExpression("a && b")

	require.NoError(t, err)
	binary, ok := node.(*BinaryNode)
	require.True(t, ok)
	assert.Equal(t, ExprTokenTypeAnd, binary.Op)

	left, ok := binary.Left.(*IdentifierNode)
	require.True(t, ok)
	assert.Equal(t, "a", left.Name)

	right, ok := binary.Right.(*IdentifierNode)
	require.True(t, ok)
	assert.Equal(t, "b", right.Name)
}

func TestExprParser_Parse_LogicalOr(t *testing.T) {
	node, err := ParseExpression("a || b")

	require.NoError(t, err)
	binary, ok := node.(*BinaryNode)
	require.True(t, ok)
	assert.Equal(t, ExprTokenTypeOr, binary.Op)
}

func TestExprParser_Parse_Precedence_OrOverAnd(t *testing.T) {
	// a || b && c should parse as a || (b && c)
	node, err := ParseExpression("a || b && c")

	require.NoError(t, err)
	binary, ok := node.(*BinaryNode)
	require.True(t, ok)
	assert.Equal(t, ExprTokenTypeOr, binary.Op)

	// Right side should be (b && c)
	right, ok := binary.Right.(*BinaryNode)
	require.True(t, ok)
	assert.Equal(t, ExprTokenTypeAnd, right.Op)
}

func TestExprParser_Parse_Precedence_ComparisonOverLogical(t *testing.T) {
	// a > 1 && b < 2 should parse as (a > 1) && (b < 2)
	node, err := ParseExpression("a > 1 && b < 2")

	require.NoError(t, err)
	binary, ok := node.(*BinaryNode)
	require.True(t, ok)
	assert.Equal(t, ExprTokenTypeAnd, binary.Op)

	left, ok := binary.Left.(*BinaryNode)
	require.True(t, ok)
	assert.Equal(t, ExprTokenTypeGt, left.Op)

	right, ok := binary.Right.(*BinaryNode)
	require.True(t, ok)
	assert.Equal(t, ExprTokenTypeLt, right.Op)
}

func TestExprParser_Parse_Parentheses(t *testing.T) {
	// (a || b) && c - parentheses override precedence
	node, err := ParseExpression("(a || b) && c")

	require.NoError(t, err)
	binary, ok := node.(*BinaryNode)
	require.True(t, ok)
	assert.Equal(t, ExprTokenTypeAnd, binary.Op)

	// Left side should be (a || b)
	left, ok := binary.Left.(*BinaryNode)
	require.True(t, ok)
	assert.Equal(t, ExprTokenTypeOr, left.Op)
}

func TestExprParser_Parse_FunctionCall(t *testing.T) {
	node, err := ParseExpression(`len(items)`)

	require.NoError(t, err)
	call, ok := node.(*CallNode)
	require.True(t, ok)
	assert.Equal(t, "len", call.Name)
	require.Len(t, call.Args, 1)

	arg, ok := call.Args[0].(*IdentifierNode)
	require.True(t, ok)
	assert.Equal(t, "items", arg.Name)
}

func TestExprParser_Parse_FunctionCallMultipleArgs(t *testing.T) {
	node, err := ParseExpression(`contains(roles, "admin")`)

	require.NoError(t, err)
	call, ok := node.(*CallNode)
	require.True(t, ok)
	assert.Equal(t, "contains", call.Name)
	require.Len(t, call.Args, 2)

	arg0, ok := call.Args[0].(*IdentifierNode)
	require.True(t, ok)
	assert.Equal(t, "roles", arg0.Name)

	arg1, ok := call.Args[1].(*LiteralNode)
	require.True(t, ok)
	assert.Equal(t, "admin", arg1.Value)
}

func TestExprParser_Parse_FunctionCallNoArgs(t *testing.T) {
	node, err := ParseExpression(`now()`)

	require.NoError(t, err)
	call, ok := node.(*CallNode)
	require.True(t, ok)
	assert.Equal(t, "now", call.Name)
	assert.Empty(t, call.Args)
}

func TestExprParser_Parse_NestedFunctionCalls(t *testing.T) {
	node, err := ParseExpression(`upper(trim(name))`)

	require.NoError(t, err)
	outer, ok := node.(*CallNode)
	require.True(t, ok)
	assert.Equal(t, "upper", outer.Name)
	require.Len(t, outer.Args, 1)

	inner, ok := outer.Args[0].(*CallNode)
	require.True(t, ok)
	assert.Equal(t, "trim", inner.Name)
}

func TestExprParser_Parse_ComplexExpression(t *testing.T) {
	// len(items) > 0 && (isAdmin || hasRole("editor"))
	node, err := ParseExpression(`len(items) > 0 && (isAdmin || hasRole("editor"))`)

	require.NoError(t, err)
	and, ok := node.(*BinaryNode)
	require.True(t, ok)
	assert.Equal(t, ExprTokenTypeAnd, and.Op)

	// Left: len(items) > 0
	left, ok := and.Left.(*BinaryNode)
	require.True(t, ok)
	assert.Equal(t, ExprTokenTypeGt, left.Op)

	leftCall, ok := left.Left.(*CallNode)
	require.True(t, ok)
	assert.Equal(t, "len", leftCall.Name)

	// Right: (isAdmin || hasRole("editor"))
	right, ok := and.Right.(*BinaryNode)
	require.True(t, ok)
	assert.Equal(t, ExprTokenTypeOr, right.Op)
}

func TestExprParser_Parse_Error_UnexpectedToken(t *testing.T) {
	_, err := ParseExpression("1 +")

	require.Error(t, err)
	// The error could be about unexpected character or unexpected token
	assert.True(t,
		strings.Contains(err.Error(), ErrMsgExprUnexpectedToken) ||
			strings.Contains(err.Error(), ErrMsgExprUnexpectedChar),
		"expected error to contain 'unexpected token' or 'unexpected character', got: %s", err.Error())
}

func TestExprParser_Parse_Error_UnclosedParen(t *testing.T) {
	_, err := ParseExpression("(a && b")

	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgExprExpectedRParen)
}

func TestExprParser_Parse_Error_UnclosedFunctionCall(t *testing.T) {
	_, err := ParseExpression("len(items")

	require.Error(t, err)
}

func TestExprParser_Parse_EmptyExpression(t *testing.T) {
	_, err := ParseExpression("")

	require.Error(t, err)
}

func TestExprNodeTypes(t *testing.T) {
	t.Run("LiteralNode type", func(t *testing.T) {
		node := &LiteralNode{Value: 42, Kind: LiteralKindNumber}
		assert.Equal(t, ExprNodeTypeLiteral, node.Type())
		assert.Contains(t, node.String(), "42")
	})

	t.Run("IdentifierNode type", func(t *testing.T) {
		node := &IdentifierNode{Name: "foo"}
		assert.Equal(t, ExprNodeTypeIdentifier, node.Type())
		assert.Contains(t, node.String(), "foo")
	})

	t.Run("UnaryNode type", func(t *testing.T) {
		node := &UnaryNode{Op: ExprTokenTypeNot, Right: &LiteralNode{Value: true}}
		assert.Equal(t, ExprNodeTypeUnary, node.Type())
		assert.Contains(t, node.String(), "!")
	})

	t.Run("BinaryNode type", func(t *testing.T) {
		node := &BinaryNode{
			Left:  &LiteralNode{Value: 1},
			Op:    ExprTokenTypeEq,
			Right: &LiteralNode{Value: 1},
		}
		assert.Equal(t, ExprNodeTypeBinary, node.Type())
		// The String method outputs the ExprTokenType which is "EQ", not "=="
		assert.Contains(t, node.String(), string(ExprTokenTypeEq))
	})

	t.Run("CallNode type", func(t *testing.T) {
		node := &CallNode{Name: "len", Args: []ExprNode{}}
		assert.Equal(t, ExprNodeTypeCall, node.Type())
		assert.Contains(t, node.String(), "len")
	})
}

func TestExprParseError_Error(t *testing.T) {
	t.Run("with detail", func(t *testing.T) {
		err := NewExprParseError("test message", 10, "detail info")
		assert.Equal(t, "test message at position 10: detail info", err.Error())
	})

	t.Run("without detail", func(t *testing.T) {
		err := NewExprParseError("test message", 10, "")
		assert.Equal(t, "test message at position 10", err.Error())
	})
}
