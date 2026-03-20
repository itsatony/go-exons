package internal

import (
	"fmt"
	"strings"
)

// ExprNodeType identifies the type of expression AST node
type ExprNodeType int

// Expression node type constants
const (
	ExprNodeTypeLiteral ExprNodeType = iota
	ExprNodeTypeIdentifier
	ExprNodeTypeUnary
	ExprNodeTypeBinary
	ExprNodeTypeCall
)

// Expression node type names for debugging
const (
	ExprNodeTypeNameLiteral    = "LITERAL"
	ExprNodeTypeNameIdentifier = "IDENTIFIER"
	ExprNodeTypeNameUnary      = "UNARY"
	ExprNodeTypeNameBinary     = "BINARY"
	ExprNodeTypeNameCall       = "CALL"
)

// String returns the string representation of the node type
func (t ExprNodeType) String() string {
	switch t {
	case ExprNodeTypeLiteral:
		return ExprNodeTypeNameLiteral
	case ExprNodeTypeIdentifier:
		return ExprNodeTypeNameIdentifier
	case ExprNodeTypeUnary:
		return ExprNodeTypeNameUnary
	case ExprNodeTypeBinary:
		return ExprNodeTypeNameBinary
	case ExprNodeTypeCall:
		return ExprNodeTypeNameCall
	default:
		return ExprNodeTypeNameLiteral
	}
}

// ExprNode is the interface for all expression AST nodes
type ExprNode interface {
	// Type returns the node type
	Type() ExprNodeType
	// String returns a string representation for debugging
	String() string
	// exprNode is a marker method to ensure type safety
	exprNode()
}

// LiteralKind identifies the kind of literal value
type LiteralKind int

// Literal kind constants
const (
	LiteralKindString LiteralKind = iota
	LiteralKindNumber
	LiteralKindBool
	LiteralKindNil
)

// LiteralNode represents a literal value (string, number, bool, nil)
type LiteralNode struct {
	Value any
	Kind  LiteralKind
}

func (n *LiteralNode) Type() ExprNodeType { return ExprNodeTypeLiteral }
func (n *LiteralNode) exprNode()          {}

func (n *LiteralNode) String() string {
	switch n.Kind {
	case LiteralKindString:
		return fmt.Sprintf("%q", n.Value)
	case LiteralKindNil:
		return ExprKeywordNil
	default:
		return fmt.Sprintf("%v", n.Value)
	}
}

// IdentifierNode represents a variable reference (may include dot notation)
type IdentifierNode struct {
	Name string
}

func (n *IdentifierNode) Type() ExprNodeType { return ExprNodeTypeIdentifier }
func (n *IdentifierNode) exprNode()          {}

func (n *IdentifierNode) String() string {
	return n.Name
}

// UnaryNode represents a unary operation (e.g., !x)
type UnaryNode struct {
	Op    ExprTokenType
	Right ExprNode
}

func (n *UnaryNode) Type() ExprNodeType { return ExprNodeTypeUnary }
func (n *UnaryNode) exprNode()          {}

func (n *UnaryNode) String() string {
	return fmt.Sprintf("(%s%s)", n.Op, n.Right.String())
}

// BinaryNode represents a binary operation (e.g., a && b)
type BinaryNode struct {
	Left  ExprNode
	Op    ExprTokenType
	Right ExprNode
}

func (n *BinaryNode) Type() ExprNodeType { return ExprNodeTypeBinary }
func (n *BinaryNode) exprNode()          {}

func (n *BinaryNode) String() string {
	return fmt.Sprintf("(%s %s %s)", n.Left.String(), n.Op, n.Right.String())
}

// CallNode represents a function call (e.g., len(items))
type CallNode struct {
	Name string
	Args []ExprNode
}

func (n *CallNode) Type() ExprNodeType { return ExprNodeTypeCall }
func (n *CallNode) exprNode()          {}

func (n *CallNode) String() string {
	args := make([]string, len(n.Args))
	for i, arg := range n.Args {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", n.Name, strings.Join(args, ", "))
}

// NewLiteralString creates a string literal node
func NewLiteralString(value string) *LiteralNode {
	return &LiteralNode{Value: value, Kind: LiteralKindString}
}

// NewLiteralNumber creates a number literal node
func NewLiteralNumber(value float64) *LiteralNode {
	return &LiteralNode{Value: value, Kind: LiteralKindNumber}
}

// NewLiteralBool creates a boolean literal node
func NewLiteralBool(value bool) *LiteralNode {
	return &LiteralNode{Value: value, Kind: LiteralKindBool}
}

// NewLiteralNil creates a nil literal node
func NewLiteralNil() *LiteralNode {
	return &LiteralNode{Value: nil, Kind: LiteralKindNil}
}

// NewIdentifier creates an identifier node
func NewIdentifier(name string) *IdentifierNode {
	return &IdentifierNode{Name: name}
}

// NewUnary creates a unary operation node
func NewUnary(op ExprTokenType, right ExprNode) *UnaryNode {
	return &UnaryNode{Op: op, Right: right}
}

// NewBinary creates a binary operation node
func NewBinary(left ExprNode, op ExprTokenType, right ExprNode) *BinaryNode {
	return &BinaryNode{Left: left, Op: op, Right: right}
}

// NewCall creates a function call node
func NewCall(name string, args []ExprNode) *CallNode {
	return &CallNode{Name: name, Args: args}
}
