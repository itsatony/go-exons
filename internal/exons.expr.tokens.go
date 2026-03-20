package internal

import (
	"fmt"
	"strings"
	"unicode"
)

// ExprTokenType represents the type of an expression token
type ExprTokenType string

// Expression token type constants
const (
	ExprTokenTypeIdentifier ExprTokenType = "IDENT"
	ExprTokenTypeString     ExprTokenType = "STRING"
	ExprTokenTypeNumber     ExprTokenType = "NUMBER"
	ExprTokenTypeBool       ExprTokenType = "BOOL"
	ExprTokenTypeNil        ExprTokenType = "NIL"
	ExprTokenTypeLParen     ExprTokenType = "LPAREN"
	ExprTokenTypeRParen     ExprTokenType = "RPAREN"
	ExprTokenTypeComma      ExprTokenType = "COMMA"

	// Operators
	ExprTokenTypeAnd ExprTokenType = "AND"
	ExprTokenTypeOr  ExprTokenType = "OR"
	ExprTokenTypeNot ExprTokenType = "NOT"
	ExprTokenTypeEq  ExprTokenType = "EQ"
	ExprTokenTypeNeq ExprTokenType = "NEQ"
	ExprTokenTypeLt  ExprTokenType = "LT"
	ExprTokenTypeGt  ExprTokenType = "GT"
	ExprTokenTypeLte ExprTokenType = "LTE"
	ExprTokenTypeGte ExprTokenType = "GTE"

	ExprTokenTypeEOF ExprTokenType = "EOF"
)

// Expression operator strings
const (
	ExprOpAnd = "&&"
	ExprOpOr  = "||"
	ExprOpNot = "!"
	ExprOpEq  = "=="
	ExprOpNeq = "!="
	ExprOpLt  = "<"
	ExprOpGt  = ">"
	ExprOpLte = "<="
	ExprOpGte = ">="
)

// Expression keyword constants
const (
	ExprKeywordTrue  = "true"
	ExprKeywordFalse = "false"
	ExprKeywordNil   = "nil"
)

// ExprToken represents a token in an expression
type ExprToken struct {
	Type    ExprTokenType
	Value   string
	Pos     int
	Literal any // Parsed value for literals (string, float64, bool, nil)
}

// String returns the string representation of the token
func (t ExprToken) String() string {
	if t.Value != "" {
		return fmt.Sprintf("%s(%s)", t.Type, t.Value)
	}
	return string(t.Type)
}

// ExprTokenizer tokenizes expression strings
type ExprTokenizer struct {
	input string
	pos   int
	len   int
}

// NewExprTokenizer creates a new expression tokenizer
func NewExprTokenizer(input string) *ExprTokenizer {
	return &ExprTokenizer{
		input: input,
		pos:   0,
		len:   len(input),
	}
}

// Tokenize converts the input string into a slice of tokens
func (t *ExprTokenizer) Tokenize() ([]ExprToken, error) {
	var tokens []ExprToken

	for {
		t.skipWhitespace()

		if t.pos >= t.len {
			tokens = append(tokens, ExprToken{Type: ExprTokenTypeEOF, Pos: t.pos})
			break
		}

		token, err := t.nextToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

// nextToken reads the next token from the input
func (t *ExprTokenizer) nextToken() (ExprToken, error) {
	startPos := t.pos
	ch := t.peek()

	// String literals
	if ch == '"' || ch == '\'' {
		return t.readString()
	}

	// Numbers
	if unicode.IsDigit(rune(ch)) || (ch == '.' && t.pos+1 < t.len && unicode.IsDigit(rune(t.input[t.pos+1]))) {
		return t.readNumber()
	}

	// Identifiers and keywords
	if unicode.IsLetter(rune(ch)) || ch == '_' {
		return t.readIdentifier()
	}

	// Two-character operators
	if t.pos+1 < t.len {
		twoChar := t.input[t.pos : t.pos+2]
		switch twoChar {
		case ExprOpAnd:
			t.pos += 2
			return ExprToken{Type: ExprTokenTypeAnd, Value: ExprOpAnd, Pos: startPos}, nil
		case ExprOpOr:
			t.pos += 2
			return ExprToken{Type: ExprTokenTypeOr, Value: ExprOpOr, Pos: startPos}, nil
		case ExprOpEq:
			t.pos += 2
			return ExprToken{Type: ExprTokenTypeEq, Value: ExprOpEq, Pos: startPos}, nil
		case ExprOpNeq:
			t.pos += 2
			return ExprToken{Type: ExprTokenTypeNeq, Value: ExprOpNeq, Pos: startPos}, nil
		case ExprOpLte:
			t.pos += 2
			return ExprToken{Type: ExprTokenTypeLte, Value: ExprOpLte, Pos: startPos}, nil
		case ExprOpGte:
			t.pos += 2
			return ExprToken{Type: ExprTokenTypeGte, Value: ExprOpGte, Pos: startPos}, nil
		}
	}

	// Single-character tokens
	t.pos++
	switch ch {
	case '(':
		return ExprToken{Type: ExprTokenTypeLParen, Value: "(", Pos: startPos}, nil
	case ')':
		return ExprToken{Type: ExprTokenTypeRParen, Value: ")", Pos: startPos}, nil
	case ',':
		return ExprToken{Type: ExprTokenTypeComma, Value: ",", Pos: startPos}, nil
	case '!':
		return ExprToken{Type: ExprTokenTypeNot, Value: ExprOpNot, Pos: startPos}, nil
	case '<':
		return ExprToken{Type: ExprTokenTypeLt, Value: ExprOpLt, Pos: startPos}, nil
	case '>':
		return ExprToken{Type: ExprTokenTypeGt, Value: ExprOpGt, Pos: startPos}, nil
	}

	return ExprToken{}, NewExprTokenError(ErrMsgExprUnexpectedChar, startPos, string(ch))
}

// readString reads a string literal
func (t *ExprTokenizer) readString() (ExprToken, error) {
	startPos := t.pos
	quote := t.input[t.pos]
	t.pos++ // skip opening quote

	var sb strings.Builder
	for t.pos < t.len {
		ch := t.input[t.pos]
		if ch == quote {
			t.pos++ // skip closing quote
			value := sb.String()
			return ExprToken{
				Type:    ExprTokenTypeString,
				Value:   value,
				Pos:     startPos,
				Literal: value,
			}, nil
		}
		if ch == '\\' && t.pos+1 < t.len {
			t.pos++
			escaped := t.input[t.pos]
			switch escaped {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			case '\'':
				sb.WriteByte('\'')
			default:
				sb.WriteByte(escaped)
			}
			t.pos++
			continue
		}
		sb.WriteByte(ch)
		t.pos++
	}

	return ExprToken{}, NewExprTokenError(ErrMsgExprUnterminatedStr, startPos, "")
}

// readNumber reads a numeric literal
func (t *ExprTokenizer) readNumber() (ExprToken, error) {
	startPos := t.pos
	hasDecimal := false

	for t.pos < t.len {
		ch := t.input[t.pos]
		if ch == '.' {
			if hasDecimal {
				break
			}
			hasDecimal = true
			t.pos++
			continue
		}
		if !unicode.IsDigit(rune(ch)) {
			break
		}
		t.pos++
	}

	value := t.input[startPos:t.pos]

	// Parse the number
	var literal float64
	_, err := fmt.Sscanf(value, "%f", &literal)
	if err != nil {
		return ExprToken{}, NewExprTokenError(ErrMsgExprInvalidNumber, startPos, value)
	}

	return ExprToken{
		Type:    ExprTokenTypeNumber,
		Value:   value,
		Pos:     startPos,
		Literal: literal,
	}, nil
}

// readIdentifier reads an identifier or keyword
func (t *ExprTokenizer) readIdentifier() (ExprToken, error) {
	startPos := t.pos

	for t.pos < t.len {
		ch := rune(t.input[t.pos])
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' && ch != '.' {
			break
		}
		t.pos++
	}

	value := t.input[startPos:t.pos]

	// Check for keywords
	switch value {
	case ExprKeywordTrue:
		return ExprToken{Type: ExprTokenTypeBool, Value: value, Pos: startPos, Literal: true}, nil
	case ExprKeywordFalse:
		return ExprToken{Type: ExprTokenTypeBool, Value: value, Pos: startPos, Literal: false}, nil
	case ExprKeywordNil:
		return ExprToken{Type: ExprTokenTypeNil, Value: value, Pos: startPos, Literal: nil}, nil
	}

	return ExprToken{Type: ExprTokenTypeIdentifier, Value: value, Pos: startPos}, nil
}

// peek returns the current character without advancing
func (t *ExprTokenizer) peek() byte {
	if t.pos >= t.len {
		return 0
	}
	return t.input[t.pos]
}

// skipWhitespace skips whitespace characters
func (t *ExprTokenizer) skipWhitespace() {
	for t.pos < t.len && unicode.IsSpace(rune(t.input[t.pos])) {
		t.pos++
	}
}

// ExprTokenError represents an error during expression tokenization
type ExprTokenError struct {
	Message string
	Pos     int
	Detail  string
}

// NewExprTokenError creates a new expression token error
func NewExprTokenError(message string, pos int, detail string) *ExprTokenError {
	return &ExprTokenError{
		Message: message,
		Pos:     pos,
		Detail:  detail,
	}
}

// Error implements the error interface
func (e *ExprTokenError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s at position %d: %s", e.Message, e.Pos, e.Detail)
	}
	return fmt.Sprintf("%s at position %d", e.Message, e.Pos)
}

// Expression tokenizer error messages
const (
	ErrMsgExprUnexpectedChar  = "unexpected character"
	ErrMsgExprUnterminatedStr = "unterminated string literal"
	ErrMsgExprInvalidNumber   = "invalid number format"
)
