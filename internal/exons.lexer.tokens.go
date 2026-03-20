package internal

import "fmt"

// Position represents a location in the source template
type Position struct {
	Offset int // Byte offset from start
	Line   int // 1-indexed line number
	Column int // 1-indexed column number
}

// String returns a human-readable position string
func (p Position) String() string {
	return fmt.Sprintf("line %d, column %d", p.Line, p.Column)
}

// Token represents a lexical token produced by the lexer
type Token struct {
	Type     TokenType // The type of token
	Value    string    // The token's value/content
	Position Position  // Source position
}

// String returns a human-readable representation of the token
func (t Token) String() string {
	if t.Value == "" {
		return fmt.Sprintf("Token{%s @ %s}", t.Type, t.Position)
	}
	return fmt.Sprintf("Token{%s: %q @ %s}", t.Type, t.Value, t.Position)
}

// NewEOFToken creates an EOF token at the given position
func NewEOFToken(pos Position) Token {
	return Token{
		Type:     TokenTypeEOF,
		Position: pos,
	}
}

// NewTextToken creates a text token with the given content
func NewTextToken(content string, pos Position) Token {
	return Token{
		Type:     TokenTypeText,
		Value:    content,
		Position: pos,
	}
}

// NewOpenTagToken creates an open tag token
func NewOpenTagToken(pos Position) Token {
	return Token{
		Type:     TokenTypeOpenTag,
		Position: pos,
	}
}

// NewCloseTagToken creates a close tag token
func NewCloseTagToken(pos Position) Token {
	return Token{
		Type:     TokenTypeCloseTag,
		Position: pos,
	}
}

// NewSelfCloseToken creates a self-close token
func NewSelfCloseToken(pos Position) Token {
	return Token{
		Type:     TokenTypeSelfClose,
		Position: pos,
	}
}

// NewBlockCloseToken creates a block close token
func NewBlockCloseToken(pos Position) Token {
	return Token{
		Type:     TokenTypeBlockClose,
		Position: pos,
	}
}

// NewTagNameToken creates a tag name token
func NewTagNameToken(name string, pos Position) Token {
	return Token{
		Type:     TokenTypeTagName,
		Value:    name,
		Position: pos,
	}
}

// NewAttrNameToken creates an attribute name token
func NewAttrNameToken(name string, pos Position) Token {
	return Token{
		Type:     TokenTypeAttrName,
		Value:    name,
		Position: pos,
	}
}

// NewAttrValueToken creates an attribute value token
func NewAttrValueToken(value string, pos Position) Token {
	return Token{
		Type:     TokenTypeAttrValue,
		Value:    value,
		Position: pos,
	}
}

// NewEqualsToken creates an equals token
func NewEqualsToken(pos Position) Token {
	return Token{
		Type:     TokenTypeEquals,
		Position: pos,
	}
}
