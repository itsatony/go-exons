package internal

import "fmt"

// ExprParser parses expression tokens into an AST
type ExprParser struct {
	tokens []ExprToken
	pos    int
}

// NewExprParser creates a new expression parser
func NewExprParser(tokens []ExprToken) *ExprParser {
	return &ExprParser{
		tokens: tokens,
		pos:    0,
	}
}

// Parse parses the expression and returns the root AST node
func (p *ExprParser) Parse() (ExprNode, error) {
	if len(p.tokens) == 0 || (len(p.tokens) == 1 && p.tokens[0].Type == ExprTokenTypeEOF) {
		return nil, NewExprParseError(ErrMsgExprEmptyExpression, 0, "")
	}

	node, err := p.parseOr()
	if err != nil {
		return nil, err
	}

	// Ensure we consumed all tokens
	if !p.isAtEnd() && p.peek().Type != ExprTokenTypeEOF {
		return nil, NewExprParseError(ErrMsgExprUnexpectedToken, p.peek().Pos, p.peek().Value)
	}

	return node, nil
}

// parseOr parses OR expressions (lowest precedence)
func (p *ExprParser) parseOr() (ExprNode, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.match(ExprTokenTypeOr) {
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = NewBinary(left, ExprTokenTypeOr, right)
	}

	return left, nil
}

// parseAnd parses AND expressions
func (p *ExprParser) parseAnd() (ExprNode, error) {
	left, err := p.parseEquality()
	if err != nil {
		return nil, err
	}

	for p.match(ExprTokenTypeAnd) {
		right, err := p.parseEquality()
		if err != nil {
			return nil, err
		}
		left = NewBinary(left, ExprTokenTypeAnd, right)
	}

	return left, nil
}

// parseEquality parses equality expressions (==, !=)
func (p *ExprParser) parseEquality() (ExprNode, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}

	for p.matchAny(ExprTokenTypeEq, ExprTokenTypeNeq) {
		op := p.previous().Type
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = NewBinary(left, op, right)
	}

	return left, nil
}

// parseComparison parses comparison expressions (<, >, <=, >=)
func (p *ExprParser) parseComparison() (ExprNode, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for p.matchAny(ExprTokenTypeLt, ExprTokenTypeGt, ExprTokenTypeLte, ExprTokenTypeGte) {
		op := p.previous().Type
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = NewBinary(left, op, right)
	}

	return left, nil
}

// parseUnary parses unary expressions (!)
func (p *ExprParser) parseUnary() (ExprNode, error) {
	if p.match(ExprTokenTypeNot) {
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return NewUnary(ExprTokenTypeNot, right), nil
	}

	return p.parseCall()
}

// parseCall parses function calls and primary expressions
func (p *ExprParser) parseCall() (ExprNode, error) {
	node, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	// Check if this is a function call (identifier followed by parenthesis)
	if ident, ok := node.(*IdentifierNode); ok {
		if p.match(ExprTokenTypeLParen) {
			return p.finishCall(ident.Name)
		}
	}

	return node, nil
}

// finishCall finishes parsing a function call after the opening paren
func (p *ExprParser) finishCall(name string) (ExprNode, error) {
	var args []ExprNode

	// Handle no arguments case
	if !p.check(ExprTokenTypeRParen) {
		for {
			arg, err := p.parseOr()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)

			if !p.match(ExprTokenTypeComma) {
				break
			}
		}
	}

	if !p.match(ExprTokenTypeRParen) {
		return nil, NewExprParseError(ErrMsgExprExpectedRParen, p.currentPos(), "")
	}

	return NewCall(name, args), nil
}

// parsePrimary parses primary expressions (literals, identifiers, parenthesized expressions)
func (p *ExprParser) parsePrimary() (ExprNode, error) {
	// Literals
	if p.match(ExprTokenTypeString) {
		return NewLiteralString(p.previous().Literal.(string)), nil
	}

	if p.match(ExprTokenTypeNumber) {
		return NewLiteralNumber(p.previous().Literal.(float64)), nil
	}

	if p.match(ExprTokenTypeBool) {
		return NewLiteralBool(p.previous().Literal.(bool)), nil
	}

	if p.match(ExprTokenTypeNil) {
		return NewLiteralNil(), nil
	}

	// Identifiers (variable references)
	if p.match(ExprTokenTypeIdentifier) {
		return NewIdentifier(p.previous().Value), nil
	}

	// Parenthesized expressions
	if p.match(ExprTokenTypeLParen) {
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}

		if !p.match(ExprTokenTypeRParen) {
			return nil, NewExprParseError(ErrMsgExprExpectedRParen, p.currentPos(), "")
		}

		return expr, nil
	}

	// Unexpected token
	if p.isAtEnd() {
		return nil, NewExprParseError(ErrMsgExprUnexpectedEOF, p.currentPos(), "")
	}

	return nil, NewExprParseError(ErrMsgExprUnexpectedToken, p.peek().Pos, p.peek().Value)
}

// Helper methods

// match checks if the current token matches and advances if so
func (p *ExprParser) match(tokenType ExprTokenType) bool {
	if p.check(tokenType) {
		p.advance()
		return true
	}
	return false
}

// matchAny checks if the current token matches any of the given types
func (p *ExprParser) matchAny(types ...ExprTokenType) bool {
	for _, t := range types {
		if p.match(t) {
			return true
		}
	}
	return false
}

// check returns true if the current token is of the given type
func (p *ExprParser) check(tokenType ExprTokenType) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek().Type == tokenType
}

// advance moves to the next token and returns the previous one
func (p *ExprParser) advance() ExprToken {
	if !p.isAtEnd() {
		p.pos++
	}
	return p.previous()
}

// peek returns the current token
func (p *ExprParser) peek() ExprToken {
	if p.pos >= len(p.tokens) {
		return ExprToken{Type: ExprTokenTypeEOF, Pos: p.currentPos()}
	}
	return p.tokens[p.pos]
}

// previous returns the previous token
func (p *ExprParser) previous() ExprToken {
	if p.pos == 0 {
		return p.tokens[0]
	}
	return p.tokens[p.pos-1]
}

// isAtEnd returns true if we've consumed all tokens
func (p *ExprParser) isAtEnd() bool {
	return p.pos >= len(p.tokens) || p.peek().Type == ExprTokenTypeEOF
}

// currentPos returns the current position for error reporting
func (p *ExprParser) currentPos() int {
	if p.pos >= len(p.tokens) {
		if len(p.tokens) > 0 {
			return p.tokens[len(p.tokens)-1].Pos
		}
		return 0
	}
	return p.tokens[p.pos].Pos
}

// ExprParseError represents an error during expression parsing
type ExprParseError struct {
	Message string
	Pos     int
	Detail  string
}

// NewExprParseError creates a new expression parse error
func NewExprParseError(message string, pos int, detail string) *ExprParseError {
	return &ExprParseError{
		Message: message,
		Pos:     pos,
		Detail:  detail,
	}
}

// Error implements the error interface
func (e *ExprParseError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s at position %d: %s", e.Message, e.Pos, e.Detail)
	}
	return fmt.Sprintf("%s at position %d", e.Message, e.Pos)
}

// Expression parser error messages
const (
	ErrMsgExprEmptyExpression = "empty expression"
	ErrMsgExprUnexpectedToken = "unexpected token"
	ErrMsgExprExpectedRParen  = "expected closing parenthesis"
	ErrMsgExprUnexpectedEOF   = "unexpected end of expression"
)

// ParseExpression is a convenience function that tokenizes and parses an expression string
func ParseExpression(expr string) (ExprNode, error) {
	tokenizer := NewExprTokenizer(expr)
	tokens, err := tokenizer.Tokenize()
	if err != nil {
		return nil, err
	}

	parser := NewExprParser(tokens)
	return parser.Parse()
}
