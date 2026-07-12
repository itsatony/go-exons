package internal

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// textValues concatenates the values of all TEXT tokens (ignoring EOF and
// structural tokens) — convenient for asserting literal output of fences.
func textValues(tokens []Token) string {
	var sb strings.Builder
	for _, tok := range tokens {
		if tok.Type == TokenTypeText {
			sb.WriteString(tok.Value)
		}
	}
	return sb.String()
}

func tokenTypes(tokens []Token) []TokenType {
	types := make([]TokenType, 0, len(tokens))
	for _, tok := range tokens {
		types = append(types, tok.Type)
	}
	return types
}

func TestLexer_VerbatimFence(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantText string // concatenated TEXT token values
	}{
		{
			name:     "simple fence with tag inside",
			input:    "A{~~ {~x~} ~~}B",
			wantText: "A {~x~} B",
		},
		{
			name:     "fence body single space",
			input:    "{~~ ~~}",
			wantText: " ",
		},
		{
			name:     "fence body single char",
			input:    "{~~a~~}",
			wantText: "a",
		},
		{
			name:     "longer close run does not close k=2",
			input:    "{~~ a ~~~} b ~~}",
			wantText: " a ~~~} b ",
		},
		{
			name:     "k=3 fence contains k=2 close",
			input:    "{~~~ x ~~} y ~~~}",
			wantText: " x ~~} y ",
		},
		{
			name:     "fence contains block close and escape",
			input:    `{~~ {~/exons.if~} and \{~ raw ~~}`,
			wantText: ` {~/exons.if~} and \{~ raw `,
		},
		{
			name:     "fence contains raw block syntax",
			input:    "{~~ {~exons.raw~}x{~/exons.raw~} ~~}",
			wantText: " {~exons.raw~}x{~/exons.raw~} ",
		},
		{
			name:     "escaped fence open is literal",
			input:    `\{~~x~~}`,
			wantText: "{~~x~~}",
		},
		{
			name:     "crlf body preserved",
			input:    "{~~ line1\r\nline2 ~~}",
			wantText: " line1\r\nline2 ",
		},
		{
			name:     "adjacent fences",
			input:    "{~~a~~}{~~b~~}",
			wantText: "ab",
		},
		{
			name:     "fence right after tag close",
			input:    `{~exons.var name="x" /~}{~~ {~y~} ~~}`,
			wantText: " {~y~} ",
		},
		{
			name:     "whole input is one fence",
			input:    "{~~ everything ~~}",
			wantText: " everything ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, nil)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)
			assert.Equal(t, tt.wantText, textValues(tokens))
		})
	}
}

func TestLexer_VerbatimFence_SingleTextToken(t *testing.T) {
	// The fence body arrives as exactly one TEXT token
	lexer := NewLexer("{~~ {~a~} {~b~} ~~}", nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)
	require.Len(t, tokens, 2) // TEXT + EOF
	assert.Equal(t, TokenTypeText, tokens[0].Type)
	assert.Equal(t, " {~a~} {~b~} ", tokens[0].Value)
}

func TestLexer_VerbatimFence_Positions(t *testing.T) {
	// Body position points at the first body byte; positions after the
	// fence account for consumed newlines.
	lexer := NewLexer("{~~\nbody\n~~}X", nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)
	require.Len(t, tokens, 3) // TEXT(body), TEXT(X), EOF
	assert.Equal(t, "\nbody\n", tokens[0].Value)
	assert.Equal(t, Position{Offset: 3, Line: 1, Column: 4}, tokens[0].Position)
	assert.Equal(t, "X", tokens[1].Value)
	assert.Equal(t, 3, tokens[1].Position.Line)
}

func TestLexer_VerbatimFence_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "immediately closed brace is unterminated", input: "{~~}"},
		{name: "merged runs cannot express empty body", input: "{~~~~}"},
		{name: "fence at EOF", input: "{~~ abc"},
		{name: "only longer close present", input: "{~~ abc ~~~}"},
		{name: "only shorter close present", input: "{~~~ abc ~~}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, nil)
			_, err := lexer.Tokenize()
			require.Error(t, err)
			assert.Contains(t, err.Error(), ErrMsgUnterminatedFence)
		})
	}
}

func TestLexer_VerbatimFence_ErrorPositionIsOpen(t *testing.T) {
	lexer := NewLexer("text\n{~~ never closed", nil)
	_, err := lexer.Tokenize()
	require.Error(t, err)
	lexErr := &LexerError{}
	require.ErrorAs(t, err, &lexErr)
	assert.Equal(t, 2, lexErr.Position.Line)
	assert.Contains(t, lexErr.Message, "2 tildes")
}

func TestLexer_VerbatimFence_CustomDelimitersInert(t *testing.T) {
	// Under custom delimiters the fence family is disabled: "{~~ x ~~}" is
	// plain text, not a fence and not an error.
	config := LexerConfig{OpenDelim: "<<", CloseDelim: ">>"}
	lexer := NewLexerWithConfig("{~~ x ~~} and <<exons.var name='y' />> end", config, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)
	assert.Contains(t, textValues(tokens), "{~~ x ~~}")
}

func TestLexer_VerbatimFence_InsideAttrValue(t *testing.T) {
	// "{~~" inside a quoted attribute value is ordinary value content.
	lexer := NewLexer(`{~exons.var name="{~~" /~}`, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)
	require.Len(t, tokens, 7)
	assert.Equal(t, TokenTypeAttrValue, tokens[4].Type)
	assert.Equal(t, "{~~", tokens[4].Value)
}

func TestLexer_VerbatimRawBlock(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantBody string
	}{
		{
			name:     "simple body",
			input:    "{~exons.raw~}hello{~/exons.raw~}",
			wantBody: "hello",
		},
		{
			name:     "lexically invalid body now legal",
			input:    "{~exons.raw~}{~ 5 ~} and a lone {~ here{~/exons.raw~}",
			wantBody: "{~ 5 ~} and a lone {~ here",
		},
		{
			name:     "escape preserved byte-for-byte",
			input:    `{~exons.raw~}\{~ stays{~/exons.raw~}`,
			wantBody: `\{~ stays`,
		},
		{
			name:     "first close wins over inner opener",
			input:    "{~exons.raw~}a{~exons.raw~}b{~/exons.raw~}",
			wantBody: "a{~exons.raw~}b",
		},
		{
			name:     "non-canonical close is body content",
			input:    "{~exons.raw~}a{~/ exons.raw ~}b{~/exons.raw~}",
			wantBody: "a{~/ exons.raw ~}b",
		},
		{
			name:     "tilde fence syntax inside raw is literal",
			input:    "{~exons.raw~}{~~ x ~~}{~/exons.raw~}",
			wantBody: "{~~ x ~~}",
		},
		{
			name:     "crlf body",
			input:    "{~exons.raw~}line1\r\nline2{~/exons.raw~}",
			wantBody: "line1\r\nline2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, nil)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)

			// OPEN, TAG_NAME, CLOSE, TEXT(body), BLOCK_CLOSE, TAG_NAME, CLOSE, EOF
			require.Len(t, tokens, 8)
			assert.Equal(t, []TokenType{
				TokenTypeOpenTag, TokenTypeTagName, TokenTypeCloseTag,
				TokenTypeText,
				TokenTypeBlockClose, TokenTypeTagName, TokenTypeCloseTag,
				TokenTypeEOF,
			}, tokenTypes(tokens))
			assert.Equal(t, tt.wantBody, tokens[3].Value)
		})
	}
}

func TestLexer_VerbatimRawBlock_EmptyBody(t *testing.T) {
	// Empty body emits no TEXT token between open and close.
	lexer := NewLexer("{~exons.raw~}{~/exons.raw~}", nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)
	assert.Equal(t, []TokenType{
		TokenTypeOpenTag, TokenTypeTagName, TokenTypeCloseTag,
		TokenTypeBlockClose, TokenTypeTagName, TokenTypeCloseTag,
		TokenTypeEOF,
	}, tokenTypes(tokens))
}

func TestLexer_VerbatimRawBlock_AttributesLexNormally(t *testing.T) {
	// Attributes on the raw opener remain syntactically legal.
	lexer := NewLexer(`{~exons.raw foo="bar"~}x{~/exons.raw~}`, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)
	assert.Equal(t, TokenTypeAttrName, tokens[2].Type)
	assert.Equal(t, "foo", tokens[2].Value)
	assert.Equal(t, "x", textValues(tokens))
}

func TestLexer_VerbatimRawBlock_SelfCloseNoScan(t *testing.T) {
	// Self-closing raw has no body: no verbatim scan is triggered, the
	// following tag lexes normally.
	lexer := NewLexer(`{~exons.raw /~}{~exons.var name="x" /~}`, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)
	assert.Equal(t, []TokenType{
		TokenTypeOpenTag, TokenTypeTagName, TokenTypeSelfClose,
		TokenTypeOpenTag, TokenTypeTagName, TokenTypeAttrName, TokenTypeEquals, TokenTypeAttrValue, TokenTypeSelfClose,
		TokenTypeEOF,
	}, tokenTypes(tokens))
}

func TestLexer_VerbatimRawBlock_Unterminated(t *testing.T) {
	for _, input := range []string{
		"{~exons.raw~}never closes",
		"{~exons.comment~}never closes",
	} {
		lexer := NewLexer(input, nil)
		_, err := lexer.Tokenize()
		require.Error(t, err, input)
		assert.Contains(t, err.Error(), ErrMsgUnterminatedVerbatimBlock)
	}
}

func TestLexer_VerbatimCommentBlock(t *testing.T) {
	// Comment bodies may contain broken syntax; they scan verbatim too.
	lexer := NewLexer("{~exons.comment~}{~broken {~/exons.comment~}", nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)
	require.Len(t, tokens, 8)
	assert.Equal(t, "{~broken ", tokens[3].Value)
}

func TestLexer_VerbatimRawBlock_CustomDelimiters(t *testing.T) {
	// The close sequence derives from the configured delimiters.
	config := LexerConfig{OpenDelim: "<<", CloseDelim: ">>"}
	lexer := NewLexerWithConfig("<<exons.raw>>{~x~} body<</exons.raw>>", config, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)
	require.Len(t, tokens, 8)
	assert.Equal(t, "{~x~} body", tokens[3].Value)
}

func TestLexer_VerbatimBlock_OuterWins(t *testing.T) {
	// comment inside raw and raw inside comment: the outer block owns the
	// bytes until its own canonical close.
	lexer := NewLexer("{~exons.raw~}{~exons.comment~}x{~/exons.comment~}{~/exons.raw~}", nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)
	assert.Equal(t, "{~exons.comment~}x{~/exons.comment~}", tokens[3].Value)

	lexer = NewLexer("{~exons.comment~}{~exons.raw~}x{~/exons.raw~}{~/exons.comment~}", nil)
	tokens, err = lexer.Tokenize()
	require.NoError(t, err)
	assert.Equal(t, "{~exons.raw~}x{~/exons.raw~}", tokens[3].Value)
}
