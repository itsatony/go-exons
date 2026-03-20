package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExprTokenizer_Tokenize_SimpleIdentifier(t *testing.T) {
	tokenizer := NewExprTokenizer("foo")
	tokens, err := tokenizer.Tokenize()

	require.NoError(t, err)
	require.Len(t, tokens, 2) // IDENT + EOF
	assert.Equal(t, ExprTokenTypeIdentifier, tokens[0].Type)
	assert.Equal(t, "foo", tokens[0].Value)
	assert.Equal(t, ExprTokenTypeEOF, tokens[1].Type)
}

func TestExprTokenizer_Tokenize_DottedIdentifier(t *testing.T) {
	tokenizer := NewExprTokenizer("user.name")
	tokens, err := tokenizer.Tokenize()

	require.NoError(t, err)
	require.Len(t, tokens, 2)
	assert.Equal(t, ExprTokenTypeIdentifier, tokens[0].Type)
	assert.Equal(t, "user.name", tokens[0].Value)
}

func TestExprTokenizer_Tokenize_StringLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"double quotes", `"hello"`, "hello"},
		{"single quotes", `'hello'`, "hello"},
		{"with spaces", `"hello world"`, "hello world"},
		{"empty string", `""`, ""},
		{"escape newline", `"hello\nworld"`, "hello\nworld"},
		{"escape tab", `"hello\tworld"`, "hello\tworld"},
		{"escape quote", `"say \"hi\""`, `say "hi"`},
		{"escape backslash", `"a\\b"`, `a\b`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewExprTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()

			require.NoError(t, err)
			require.Len(t, tokens, 2)
			assert.Equal(t, ExprTokenTypeString, tokens[0].Type)
			assert.Equal(t, tt.expected, tokens[0].Value)
			assert.Equal(t, tt.expected, tokens[0].Literal)
		})
	}
}

func TestExprTokenizer_Tokenize_NumberLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{"integer", "42", 42.0},
		{"decimal", "3.14", 3.14},
		{"leading decimal", ".5", 0.5},
		{"zero", "0", 0.0},
		{"large number", "1000000", 1000000.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewExprTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()

			require.NoError(t, err)
			require.Len(t, tokens, 2)
			assert.Equal(t, ExprTokenTypeNumber, tokens[0].Type)
			assert.Equal(t, tt.expected, tokens[0].Literal)
		})
	}
}

func TestExprTokenizer_Tokenize_BoolLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokenizer := NewExprTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()

			require.NoError(t, err)
			require.Len(t, tokens, 2)
			assert.Equal(t, ExprTokenTypeBool, tokens[0].Type)
			assert.Equal(t, tt.expected, tokens[0].Literal)
		})
	}
}

func TestExprTokenizer_Tokenize_NilLiteral(t *testing.T) {
	tokenizer := NewExprTokenizer("nil")
	tokens, err := tokenizer.Tokenize()

	require.NoError(t, err)
	require.Len(t, tokens, 2)
	assert.Equal(t, ExprTokenTypeNil, tokens[0].Type)
	assert.Nil(t, tokens[0].Literal)
}

func TestExprTokenizer_Tokenize_Operators(t *testing.T) {
	tests := []struct {
		input    string
		expected ExprTokenType
	}{
		{"&&", ExprTokenTypeAnd},
		{"||", ExprTokenTypeOr},
		{"!", ExprTokenTypeNot},
		{"==", ExprTokenTypeEq},
		{"!=", ExprTokenTypeNeq},
		{"<", ExprTokenTypeLt},
		{">", ExprTokenTypeGt},
		{"<=", ExprTokenTypeLte},
		{">=", ExprTokenTypeGte},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokenizer := NewExprTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()

			require.NoError(t, err)
			require.Len(t, tokens, 2)
			assert.Equal(t, tt.expected, tokens[0].Type)
		})
	}
}

func TestExprTokenizer_Tokenize_Punctuation(t *testing.T) {
	tests := []struct {
		input    string
		expected ExprTokenType
	}{
		{"(", ExprTokenTypeLParen},
		{")", ExprTokenTypeRParen},
		{",", ExprTokenTypeComma},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokenizer := NewExprTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()

			require.NoError(t, err)
			require.Len(t, tokens, 2)
			assert.Equal(t, tt.expected, tokens[0].Type)
		})
	}
}

func TestExprTokenizer_Tokenize_ComplexExpression(t *testing.T) {
	tokenizer := NewExprTokenizer(`len(items) > 0 && user.isAdmin == true`)
	tokens, err := tokenizer.Tokenize()

	require.NoError(t, err)
	require.Len(t, tokens, 11) // len ( items ) > 0 && user.isAdmin == true EOF

	assert.Equal(t, ExprTokenTypeIdentifier, tokens[0].Type)
	assert.Equal(t, "len", tokens[0].Value)
	assert.Equal(t, ExprTokenTypeLParen, tokens[1].Type)
	assert.Equal(t, ExprTokenTypeIdentifier, tokens[2].Type)
	assert.Equal(t, "items", tokens[2].Value)
	assert.Equal(t, ExprTokenTypeRParen, tokens[3].Type)
	assert.Equal(t, ExprTokenTypeGt, tokens[4].Type)
	assert.Equal(t, ExprTokenTypeNumber, tokens[5].Type)
	assert.Equal(t, ExprTokenTypeAnd, tokens[6].Type)
	assert.Equal(t, ExprTokenTypeIdentifier, tokens[7].Type)
	assert.Equal(t, "user.isAdmin", tokens[7].Value)
	assert.Equal(t, ExprTokenTypeEq, tokens[8].Type)
	assert.Equal(t, ExprTokenTypeBool, tokens[9].Type)
	assert.Equal(t, ExprTokenTypeEOF, tokens[10].Type)
}

func TestExprTokenizer_Tokenize_WhitespaceHandling(t *testing.T) {
	tokenizer := NewExprTokenizer("  foo   &&   bar  ")
	tokens, err := tokenizer.Tokenize()

	require.NoError(t, err)
	require.Len(t, tokens, 4) // foo && bar EOF
}

func TestExprTokenizer_Tokenize_FunctionCall(t *testing.T) {
	tokenizer := NewExprTokenizer(`contains(roles, "admin")`)
	tokens, err := tokenizer.Tokenize()

	require.NoError(t, err)
	require.Len(t, tokens, 7) // contains ( roles , "admin" ) EOF

	assert.Equal(t, "contains", tokens[0].Value)
	assert.Equal(t, ExprTokenTypeLParen, tokens[1].Type)
	assert.Equal(t, "roles", tokens[2].Value)
	assert.Equal(t, ExprTokenTypeComma, tokens[3].Type)
	assert.Equal(t, "admin", tokens[4].Value)
	assert.Equal(t, ExprTokenTypeRParen, tokens[5].Type)
	assert.Equal(t, ExprTokenTypeEOF, tokens[6].Type)
}

func TestExprTokenizer_Tokenize_Error_UnexpectedChar(t *testing.T) {
	tokenizer := NewExprTokenizer("foo @ bar")
	_, err := tokenizer.Tokenize()

	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgExprUnexpectedChar)
}

func TestExprTokenizer_Tokenize_Error_UnterminatedString(t *testing.T) {
	tokenizer := NewExprTokenizer(`"unterminated`)
	_, err := tokenizer.Tokenize()

	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgExprUnterminatedStr)
}

func TestExprTokenizer_Tokenize_EmptyInput(t *testing.T) {
	tokenizer := NewExprTokenizer("")
	tokens, err := tokenizer.Tokenize()

	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, ExprTokenTypeEOF, tokens[0].Type)
}

func TestExprToken_String(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		token := ExprToken{Type: ExprTokenTypeIdentifier, Value: "foo"}
		assert.Equal(t, "IDENT(foo)", token.String())
	})

	t.Run("without value", func(t *testing.T) {
		token := ExprToken{Type: ExprTokenTypeLParen}
		assert.Equal(t, "LPAREN", token.String())
	})
}

func TestExprTokenError_Error(t *testing.T) {
	t.Run("with detail", func(t *testing.T) {
		err := NewExprTokenError("test error", 5, "detail")
		assert.Equal(t, "test error at position 5: detail", err.Error())
	})

	t.Run("without detail", func(t *testing.T) {
		err := NewExprTokenError("test error", 5, "")
		assert.Equal(t, "test error at position 5", err.Error())
	})
}
