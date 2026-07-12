package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanMarkdownFences_Basic(t *testing.T) {
	source := "before\n```go\ncode {~x~}\n```\nafter\n"
	regions := ScanMarkdownFences(source)
	require.Len(t, regions, 1)

	r := regions[0]
	assert.Equal(t, InertRegionFence, r.Kind)
	assert.Equal(t, byte(CharBacktick), r.FenceChar)
	assert.Equal(t, 3, r.FenceLen)
	assert.Equal(t, "go", r.InfoString)
	assert.False(t, r.Live)
	assert.False(t, r.Unclosed)
	assert.Equal(t, 7, r.Start)                                     // start of "```go" line
	assert.Equal(t, 13, r.BodyStart)                                // after "```go\n"
	assert.Equal(t, len("before\n```go\ncode {~x~}\n```\n"), r.End) // includes closer line
	assert.Equal(t, 2, r.OpenPos.Line)
}

func TestScanMarkdownFences_LiveInfoString(t *testing.T) {
	tests := []struct {
		name string
		info string
		live bool
	}{
		{name: "exact exons", info: "exons", live: true},
		{name: "exons with extra words", info: `exons title="x"`, live: true},
		{name: "prefix does not match", info: "exonsx", live: false},
		{name: "other language", info: "go", live: false},
		{name: "empty info", info: "", live: false},
		{name: "exons as second word", info: "go exons", live: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := "```" + tt.info + "\nbody\n```\n"
			regions := ScanMarkdownFences(source)
			require.Len(t, regions, 1)
			assert.Equal(t, tt.live, regions[0].Live)
		})
	}
}

func TestScanMarkdownFences_OpenerRules(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		wantCount int
	}{
		{name: "tilde fence", source: "~~~\nbody\n~~~\n", wantCount: 1},
		{name: "longer opener run", source: "`````\nbody\n`````\n", wantCount: 1},
		{name: "two-char run is not a fence", source: "``\nbody\n``\n", wantCount: 0},
		{name: "indent up to three spaces ok", source: "   ```\nbody\n```\n", wantCount: 1},
		{name: "four-space indent is not a fence", source: "    ```\nbody\n    ```\n", wantCount: 0},
		{name: "backtick info with backtick is not a fence", source: "```go`x\nbody\nplain\n", wantCount: 0},
		{name: "tilde info may contain backtick", source: "~~~go`x\nbody\n~~~\n", wantCount: 1},
		{name: "two sequential fences", source: "```\na\n```\n```\nb\n```\n", wantCount: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regions := ScanMarkdownFences(tt.source)
			assert.Len(t, regions, tt.wantCount)
		})
	}
}

func TestScanMarkdownFences_CloserRules(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		wantUnclosed bool
	}{
		{name: "same length closes", source: "```\nbody\n```\n", wantUnclosed: false},
		{name: "longer run closes", source: "```\nbody\n`````\n", wantUnclosed: false},
		{name: "shorter run is content", source: "````\nbody\n```\n", wantUnclosed: true},
		{name: "other char is content", source: "```\nbody\n~~~\n", wantUnclosed: true},
		{name: "closer with info text is content", source: "```\nbody\n``` info\n", wantUnclosed: true},
		{name: "closer with trailing spaces closes", source: "```\nbody\n```   \n", wantUnclosed: false},
		{name: "closer indented three spaces closes", source: "```\nbody\n   ```\n", wantUnclosed: false},
		{name: "no closer runs to EOF", source: "```\nbody without end", wantUnclosed: true},
		{name: "closer at EOF without newline", source: "```\nbody\n```", wantUnclosed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regions := ScanMarkdownFences(tt.source)
			require.Len(t, regions, 1)
			assert.Equal(t, tt.wantUnclosed, regions[0].Unclosed)
			if tt.wantUnclosed {
				assert.Equal(t, len(tt.source), regions[0].End)
			}
		})
	}
}

func TestScanMarkdownFences_CRLF(t *testing.T) {
	source := "```go\r\ncode {~x~}\r\n```\r\nafter"
	regions := ScanMarkdownFences(source)
	require.Len(t, regions, 1)
	assert.Equal(t, "go", regions[0].InfoString)
	assert.False(t, regions[0].Unclosed)
	assert.Equal(t, len("```go\r\ncode {~x~}\r\n```\r\n"), regions[0].End)
}

func TestScanMarkdownFences_FenceLineInsideLiveFence(t *testing.T) {
	// A "~~~" line inside a live backtick fence is fence body, not a new
	// opener: the whole live region is consumed as one unit.
	source := "```exons\n~~~\n{~x~}\n```\n~~~\nreal tilde fence\n~~~\n"
	regions := ScanMarkdownFences(source)
	require.Len(t, regions, 2)
	assert.True(t, regions[0].Live)
	assert.Equal(t, byte(CharBacktick), regions[0].FenceChar)
	assert.Equal(t, byte(CharTilde), regions[1].FenceChar)
	assert.False(t, regions[1].Live)
}

func TestScanMarkdownFences_NoFences(t *testing.T) {
	assert.Empty(t, ScanMarkdownFences("plain {~exons.var name=\"x\" /~} text\nwith `inline code`\n"))
	assert.Empty(t, ScanMarkdownFences(""))
}

// --- Lexer integration (MarkdownFences mode) ---

func mdLexerConfig() LexerConfig {
	config := DefaultLexerConfig()
	config.MarkdownFences = true
	return config
}

func TestLexer_MarkdownFences_InertFence(t *testing.T) {
	source := "before {~a~}\n```\n{~exons.var name=\"x\" /~} \\{~ {~~f~~}\n```\nafter"
	lexer := NewLexerWithConfig(source, mdLexerConfig(), nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	// The fence contents (incl. escapes and tilde fences) are literal text;
	// the tag before the fence still lexes.
	assert.Contains(t, textValues(tokens), "{~exons.var name=\"x\" /~} \\{~ {~~f~~}")
	assert.Equal(t, TokenTypeOpenTag, tokens[1].Type)
}

func TestLexer_MarkdownFences_LiveFence(t *testing.T) {
	source := "```exons\n{~exons.var name=\"x\" /~}\n```\n"
	lexer := NewLexerWithConfig(source, mdLexerConfig(), nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	types := tokenTypes(tokens)
	assert.Contains(t, types, TokenTypeOpenTag)
	assert.Contains(t, types, TokenTypeSelfClose)
}

func TestLexer_MarkdownFences_UnclosedInertToEOF(t *testing.T) {
	source := "text\n```\n{~exons.var name=\"x\" /~}\nno closer"
	lexer := NewLexerWithConfig(source, mdLexerConfig(), nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)
	assert.NotContains(t, tokenTypes(tokens), TokenTypeOpenTag)
	assert.Equal(t, source, textValues(tokens))
}

func TestLexer_MarkdownFences_TildeFence(t *testing.T) {
	source := "~~~\n{~x~}\n~~~\n{~exons.var name=\"y\" /~}"
	lexer := NewLexerWithConfig(source, mdLexerConfig(), nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)
	assert.Contains(t, textValues(tokens), "{~x~}")
	assert.Contains(t, tokenTypes(tokens), TokenTypeSelfClose)
}

func TestLexer_MarkdownFences_ModeOffUnchanged(t *testing.T) {
	// Without the mode, fence lines are plain text and tags inside lex live.
	source := "```\n{~exons.var name=\"x\" /~}\n```\n"
	lexer := NewLexer(source, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)
	assert.Contains(t, tokenTypes(tokens), TokenTypeSelfClose)
}

func TestLexer_MarkdownFences_RawOpenedBeforeFenceWins(t *testing.T) {
	// A raw block opened before a fence consumes through it: the close
	// inside the fence body still ends the raw block (first-close-wins).
	source := "{~exons.raw~}\n```\n{~/exons.raw~}\n```\nafter"
	lexer := NewLexerWithConfig(source, mdLexerConfig(), nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(tokens), 4)
	assert.Equal(t, "\n```\n", tokens[3].Value) // raw body up to the close inside the fence
}
