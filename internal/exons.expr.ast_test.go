package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExprNodeType_String(t *testing.T) {
	tests := []struct {
		nodeType ExprNodeType
		expected string
	}{
		{ExprNodeTypeLiteral, ExprNodeTypeNameLiteral},
		{ExprNodeTypeIdentifier, ExprNodeTypeNameIdentifier},
		{ExprNodeTypeUnary, ExprNodeTypeNameUnary},
		{ExprNodeTypeBinary, ExprNodeTypeNameBinary},
		{ExprNodeTypeCall, ExprNodeTypeNameCall},
		{ExprNodeType(99), ExprNodeTypeNameLiteral}, // unknown defaults to LITERAL
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.nodeType.String())
		})
	}
}

func TestLiteralNode_String(t *testing.T) {
	tests := []struct {
		name     string
		node     *LiteralNode
		expected string
	}{
		{
			name:     "string literal",
			node:     NewLiteralString("hello"),
			expected: `"hello"`,
		},
		{
			name:     "number literal",
			node:     NewLiteralNumber(42.5),
			expected: "42.5",
		},
		{
			name:     "bool literal true",
			node:     NewLiteralBool(true),
			expected: "true",
		},
		{
			name:     "bool literal false",
			node:     NewLiteralBool(false),
			expected: "false",
		},
		{
			name:     "nil literal",
			node:     NewLiteralNil(),
			expected: "nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.node.String())
			assert.Equal(t, ExprNodeTypeLiteral, tt.node.Type())
		})
	}
}

func TestLiteralNode_exprNode(t *testing.T) {
	node := NewLiteralString("test")
	// Just ensure the marker method exists and doesn't panic
	node.exprNode()
}

func TestIdentifierNode_String(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"foo", "foo"},
		{"user.name", "user.name"},
		{"data.items.0.value", "data.items.0.value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &IdentifierNode{Name: tt.name}
			assert.Equal(t, tt.expected, node.String())
			assert.Equal(t, ExprNodeTypeIdentifier, node.Type())
		})
	}
}

func TestIdentifierNode_exprNode(t *testing.T) {
	node := &IdentifierNode{Name: "test"}
	node.exprNode()
}

func TestUnaryNode_String(t *testing.T) {
	node := &UnaryNode{
		Op:    ExprTokenTypeNot,
		Right: NewLiteralBool(true),
	}
	result := node.String()
	assert.Contains(t, result, "true")
	assert.Equal(t, ExprNodeTypeUnary, node.Type())
}

func TestUnaryNode_exprNode(t *testing.T) {
	node := &UnaryNode{Op: ExprTokenTypeNot, Right: NewLiteralBool(true)}
	node.exprNode()
}

func TestBinaryNode_String(t *testing.T) {
	node := &BinaryNode{
		Left:  NewLiteralNumber(1),
		Op:    ExprTokenTypeEq,
		Right: NewLiteralNumber(2),
	}
	result := node.String()
	assert.Contains(t, result, "1")
	assert.Contains(t, result, "2")
	assert.Equal(t, ExprNodeTypeBinary, node.Type())
}

func TestBinaryNode_exprNode(t *testing.T) {
	node := &BinaryNode{
		Left:  NewLiteralNumber(1),
		Op:    ExprTokenTypeEq,
		Right: NewLiteralNumber(2),
	}
	node.exprNode()
}

func TestCallNode_String(t *testing.T) {
	tests := []struct {
		name     string
		node     *CallNode
		contains []string
	}{
		{
			name: "no args",
			node: &CallNode{
				Name: "now",
				Args: []ExprNode{},
			},
			contains: []string{"now()"},
		},
		{
			name: "one arg",
			node: &CallNode{
				Name: "len",
				Args: []ExprNode{&IdentifierNode{Name: "items"}},
			},
			contains: []string{"len(", "items"},
		},
		{
			name: "multiple args",
			node: &CallNode{
				Name: "contains",
				Args: []ExprNode{
					&IdentifierNode{Name: "list"},
					NewLiteralString("value"),
				},
			},
			contains: []string{"contains(", "list", "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.node.String()
			for _, c := range tt.contains {
				assert.Contains(t, result, c)
			}
			assert.Equal(t, ExprNodeTypeCall, tt.node.Type())
		})
	}
}

func TestCallNode_exprNode(t *testing.T) {
	node := &CallNode{Name: "test", Args: []ExprNode{}}
	node.exprNode()
}

func TestNewLiteralHelpers(t *testing.T) {
	t.Run("NewLiteralString", func(t *testing.T) {
		node := NewLiteralString("test")
		assert.Equal(t, "test", node.Value)
		assert.Equal(t, LiteralKindString, node.Kind)
	})

	t.Run("NewLiteralNumber", func(t *testing.T) {
		node := NewLiteralNumber(3.14)
		assert.Equal(t, 3.14, node.Value)
		assert.Equal(t, LiteralKindNumber, node.Kind)
	})

	t.Run("NewLiteralBool", func(t *testing.T) {
		node := NewLiteralBool(true)
		assert.Equal(t, true, node.Value)
		assert.Equal(t, LiteralKindBool, node.Kind)
	})

	t.Run("NewLiteralNil", func(t *testing.T) {
		node := NewLiteralNil()
		assert.Nil(t, node.Value)
		assert.Equal(t, LiteralKindNil, node.Kind)
	})
}
