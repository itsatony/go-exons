package internal

import (
	"fmt"
	"log/slog"
	"strings"
)

// LexerConfig holds lexer configuration
type LexerConfig struct {
	OpenDelim      string // Opening delimiter (default: "{~")
	CloseDelim     string // Closing delimiter (default: "~}")
	MarkdownFences bool   // Treat markdown code fences as inert regions
}

// DefaultLexerConfig returns the default lexer configuration
func DefaultLexerConfig() LexerConfig {
	return LexerConfig{
		OpenDelim:  StrOpenDelim,
		CloseDelim: StrCloseDelim,
	}
}

// selfClose returns the self-close pattern for this config (e.g., "/~}" for "~}")
func (c LexerConfig) selfClose() string {
	return "/" + c.CloseDelim
}

// blockClose returns the block-close pattern for this config (e.g., "{~/" for "{~")
func (c LexerConfig) blockClose() string {
	return c.OpenDelim + "/"
}

// escapeOpen returns the escape pattern for this config (e.g., "\{~" for "{~")
func (c LexerConfig) escapeOpen() string {
	return "\\" + c.OpenDelim
}

// verbatimFencesActive reports whether {~~ ... ~~} verbatim fences are
// recognized. The fence family extends the default delimiter alphabet only;
// under custom delimiters the sequence "{~~" remains plain text.
func (c LexerConfig) verbatimFencesActive() bool {
	return c.OpenDelim == StrOpenDelim && c.CloseDelim == StrCloseDelim
}

// Lexer tokenizes template source into a token stream
type Lexer struct {
	source       string
	config       LexerConfig
	pos          int // Current byte position
	line         int // Current line (1-indexed)
	column       int // Current column (1-indexed)
	logger       *slog.Logger
	fencesActive bool          // cached LexerConfig.verbatimFencesActive()
	inertRegions []InertRegion // markdown fence regions (MarkdownFences mode)
	inertIdx     int           // cursor into inertRegions (monotonic)
}

// NewLexer creates a new lexer with default configuration
func NewLexer(source string, logger *slog.Logger) *Lexer {
	return NewLexerWithConfig(source, DefaultLexerConfig(), logger)
}

// NewLexerWithConfig creates a lexer with custom configuration
func NewLexerWithConfig(source string, config LexerConfig, logger *slog.Logger) *Lexer {
	if logger == nil {
		logger = slog.Default()
	}
	logger.Debug(LogMsgLexerCreated, slog.Int(LogFieldSource, len(source)))
	l := &Lexer{
		source:       source,
		config:       config,
		pos:          0,
		line:         1,
		column:       1,
		logger:       logger,
		fencesActive: config.verbatimFencesActive(),
	}
	if config.MarkdownFences {
		l.inertRegions = ScanMarkdownFences(source)
	}
	return l
}

// Tokenize processes the source and returns a token stream
func (l *Lexer) Tokenize() ([]Token, error) {
	l.logger.Debug(LogMsgTokenizerStart)
	var tokens []Token

	for !l.isAtEnd() {
		// Consume markdown-inert regions wholesale (MarkdownFences mode).
		// Must run before any delimiter matching so fenced examples stay text.
		if region, ok := l.currentInertRegion(); ok {
			pos := l.currentPosition()
			body := l.source[l.pos:region.End]
			l.advanceN(len(body))
			tokens = append(tokens, NewTextToken(body, pos))
			continue
		}

		// Check for escape sequence first
		if l.isEscapedOpenDelim() {
			// Handle escape: consume \{~ and emit {~ as text
			pos := l.currentPosition()
			l.advanceN(len(l.config.escapeOpen())) // Skip escaped open delim
			tokens = append(tokens, NewTextToken(l.config.OpenDelim, pos))
			continue
		}

		// Check for verbatim tilde fence {~~ ... ~~} (before block-close/open:
		// "{~~" prefix-matches the open delimiter but is a fence of order >= 2)
		if l.fencesActive && l.isFenceOpen() {
			fenceToken, err := l.scanVerbatimFence()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, fenceToken)
			continue
		}

		// Check for block close delimiter (e.g., {~/)
		blockClosePattern := l.config.blockClose()
		if l.matchStr(blockClosePattern) {
			pos := l.currentPosition()
			l.advanceN(len(blockClosePattern))
			tokens = append(tokens, NewBlockCloseToken(pos))
			// Now scan tag name
			tagTokens, err := l.scanTagContent(true)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tagTokens...)
			continue
		}

		// Check for open delimiter {~
		if l.matchStr(l.config.OpenDelim) {
			pos := l.currentPosition()
			l.advanceN(len(l.config.OpenDelim))
			tokens = append(tokens, NewOpenTagToken(pos))
			// Scan tag content (name, attributes, close)
			tagTokens, err := l.scanTagContent(false)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tagTokens...)

			// Raw/comment block openers switch to verbatim scanning: the body
			// runs byte-for-byte until the canonical close sequence, so it
			// need not be lexically valid exons.
			if name, isVerbatim := verbatimBlockTag(tagTokens); isVerbatim {
				blockTokens, err := l.scanVerbatimBlock(name, pos)
				if err != nil {
					return nil, err
				}
				tokens = append(tokens, blockTokens...)
			}
			continue
		}

		// Scan regular text until next delimiter or escape
		textToken, err := l.scanText()
		if err != nil {
			return nil, err
		}
		if textToken.Value != "" {
			tokens = append(tokens, textToken)
		}
	}

	// Add EOF token
	tokens = append(tokens, NewEOFToken(l.currentPosition()))
	l.logger.Debug(LogMsgTokenizerEnd, slog.Int(LogFieldTokens, len(tokens)))
	return tokens, nil
}

// scanText scans text content until the next delimiter or escape sequence
func (l *Lexer) scanText() (Token, error) {
	startPos := l.currentPosition()
	var sb strings.Builder

	blockClosePattern := l.config.blockClose()
	for !l.isAtEnd() {
		// Stop at a markdown-inert region boundary
		if _, inRegion := l.currentInertRegion(); inRegion {
			break
		}
		// Stop at escape sequence
		if l.isEscapedOpenDelim() {
			break
		}
		// Stop at block close
		if l.matchStr(blockClosePattern) {
			break
		}
		// Stop at open delimiter
		if l.matchStr(l.config.OpenDelim) {
			break
		}

		ch := l.advance()
		sb.WriteByte(ch)
	}

	return NewTextToken(sb.String(), startPos), nil
}

// currentInertRegion returns the markdown-inert region containing the current
// position, advancing the region cursor past regions already behind us. Live
// fences (info string "exons") are skipped: their contents lex normally.
func (l *Lexer) currentInertRegion() (InertRegion, bool) {
	for l.inertIdx < len(l.inertRegions) {
		region := l.inertRegions[l.inertIdx]
		if l.pos >= region.End {
			l.inertIdx++
			continue
		}
		// Live fences (info string "exons") lex normally: not inert.
		if !region.Live && l.pos >= region.Start {
			return region, true
		}
		return InertRegion{}, false
	}
	return InertRegion{}, false
}

// tildeRunLenAt returns the length of the tilde run starting at pos.
func (l *Lexer) tildeRunLenAt(pos int) int {
	n := 0
	for pos+n < len(l.source) && l.source[pos+n] == CharTilde {
		n++
	}
	return n
}

// isFenceOpen reports whether the current position starts a verbatim tilde
// fence: '{' followed by a run of at least MinVerbatimFenceTildes tildes.
func (l *Lexer) isFenceOpen() bool {
	return l.peek() == CharOpenBrace && l.tildeRunLenAt(l.pos+1) >= MinVerbatimFenceTildes
}

// scanVerbatimFence scans a verbatim tilde fence {~~ ... ~~} of order k
// (k = length of the opening tilde run, k >= 2). The body is emitted
// byte-for-byte as a single text token with no escape, tag, or nested-fence
// processing. The fence closes at the first maximal run of exactly k tildes
// immediately followed by '}'; a longer run never closes (so "~~~}" is body
// content inside a k=2 fence). An unterminated fence is a hard lexer error.
func (l *Lexer) scanVerbatimFence() (Token, error) {
	openPos := l.currentPosition()
	l.advance() // consume '{'
	k := 0
	for !l.isAtEnd() && l.peek() == CharTilde {
		l.advance()
		k++
	}

	bodyPos := l.currentPosition()
	bodyStart := l.pos

	// Scan forward over whole tilde runs; the body always starts with a
	// non-tilde byte (the opening run was maximal), so every run below is
	// left-bounded by a non-tilde and therefore maximal.
	i := bodyStart
	for i < len(l.source) {
		if l.source[i] != CharTilde {
			i++
			continue
		}
		runStart := i
		for i < len(l.source) && l.source[i] == CharTilde {
			i++
		}
		if i-runStart == k && i < len(l.source) && l.source[i] == CharCloseBrace {
			body := l.source[bodyStart:runStart]
			l.advanceN(len(body)) // move over body (maintains line/column)
			l.advanceN(k + 1)     // move over closing run + '}'
			l.logger.Debug(LogMsgFenceScanned, slog.Int(LogFieldFenceLen, k), slog.Int(LogFieldSource, len(body)))
			return NewTextToken(body, bodyPos), nil
		}
	}
	return Token{}, l.newUnterminatedFenceError(k, openPos)
}

// verbatimBlockTag inspects the token stream of a just-scanned open tag and
// returns the tag name if it opens a verbatim block: exons.raw or
// exons.comment in block form (CLOSE_TAG, not SELF_CLOSE).
func verbatimBlockTag(tagTokens []Token) (string, bool) {
	if len(tagTokens) == 0 || tagTokens[0].Type != TokenTypeTagName {
		return "", false
	}
	name := tagTokens[0].Value
	if name != TagNameRaw && name != TagNameComment {
		return "", false
	}
	if tagTokens[len(tagTokens)-1].Type != TokenTypeCloseTag {
		return "", false
	}
	return name, true
}

// scanVerbatimBlock scans the body of a raw/comment block byte-for-byte until
// the canonical close sequence (e.g. "{~/exons.raw~}", built from the
// configured delimiters) and emits TEXT (omitted when empty) + BLOCK_CLOSE +
// TAG_NAME + CLOSE_TAG. First close wins; the {~~ ... ~~} fence is the escape
// hatch for content containing the close sequence itself.
func (l *Lexer) scanVerbatimBlock(tagName string, openPos Position) ([]Token, error) {
	closeSeq := l.config.blockClose() + tagName + l.config.CloseDelim
	idx := strings.Index(l.source[l.pos:], closeSeq)
	if idx < 0 {
		return nil, l.newUnterminatedVerbatimBlockError(closeSeq, openPos)
	}

	var tokens []Token
	if idx > 0 {
		bodyPos := l.currentPosition()
		body := l.source[l.pos : l.pos+idx]
		l.advanceN(idx)
		tokens = append(tokens, NewTextToken(body, bodyPos))
	}

	blockClosePos := l.currentPosition()
	l.advanceN(len(l.config.blockClose()))
	tokens = append(tokens, NewBlockCloseToken(blockClosePos))

	namePos := l.currentPosition()
	l.advanceN(len(tagName))
	tokens = append(tokens, NewTagNameToken(tagName, namePos))

	closePos := l.currentPosition()
	l.advanceN(len(l.config.CloseDelim))
	tokens = append(tokens, NewCloseTagToken(closePos))

	l.logger.Debug(LogMsgVerbatimBlockScanned, slog.String(LogFieldTag, tagName), slog.Int(LogFieldSource, idx))
	return tokens, nil
}

// scanTagContent scans the content inside a tag (name, attributes, closing)
// isBlockClose indicates if this is a closing tag ({~/...)
func (l *Lexer) scanTagContent(isBlockClose bool) ([]Token, error) {
	var tokens []Token

	l.skipWhitespace()

	// Scan tag name
	nameToken, err := l.scanTagName()
	if err != nil {
		return nil, err
	}
	tokens = append(tokens, nameToken)

	l.skipWhitespace()

	// For block close tags, just need the close delimiter
	if isBlockClose {
		if !l.matchStr(l.config.CloseDelim) {
			return nil, l.newUnterminatedTagError()
		}
		pos := l.currentPosition()
		l.advanceN(len(l.config.CloseDelim))
		tokens = append(tokens, NewCloseTagToken(pos))
		return tokens, nil
	}

	// Scan attributes
	selfClosePattern := l.config.selfClose()
	for !l.isAtEnd() {
		l.skipWhitespace()

		// Check for self-close (e.g., /~})
		if l.matchStr(selfClosePattern) {
			pos := l.currentPosition()
			l.advanceN(len(selfClosePattern))
			tokens = append(tokens, NewSelfCloseToken(pos))
			return tokens, nil
		}

		// Check for close delimiter ~}
		if l.matchStr(l.config.CloseDelim) {
			pos := l.currentPosition()
			l.advanceN(len(l.config.CloseDelim))
			tokens = append(tokens, NewCloseTagToken(pos))
			return tokens, nil
		}

		// Scan attribute
		attrTokens, err := l.scanAttribute()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, attrTokens...)
	}

	return nil, l.newUnterminatedTagError()
}

// scanTagName scans an identifier for a tag name
func (l *Lexer) scanTagName() (Token, error) {
	startPos := l.currentPosition()
	var sb strings.Builder

	// First character must be letter or underscore
	if !l.isAtEnd() && (isLetter(l.peek()) || l.peek() == '_') {
		sb.WriteByte(l.advance())
	} else {
		return Token{}, l.newInvalidTagNameError()
	}

	// Subsequent characters can be letter, digit, underscore, hyphen, or dot
	for !l.isAtEnd() {
		ch := l.peek()
		if isLetter(ch) || isDigit(ch) || ch == '_' || ch == '-' || ch == '.' {
			sb.WriteByte(l.advance())
		} else {
			break
		}
	}

	return NewTagNameToken(sb.String(), startPos), nil
}

// scanAttribute scans an attribute name=value pair
func (l *Lexer) scanAttribute() ([]Token, error) {
	var tokens []Token

	// Scan attribute name
	nameToken, err := l.scanAttrName()
	if err != nil {
		return nil, err
	}
	tokens = append(tokens, nameToken)

	l.skipWhitespace()

	// Expect equals sign
	if l.isAtEnd() || l.peek() != CharEquals {
		return nil, l.newUnexpectedCharError()
	}
	tokens = append(tokens, NewEqualsToken(l.currentPosition()))
	l.advance()

	l.skipWhitespace()

	// Scan attribute value
	valueToken, err := l.scanAttrValue()
	if err != nil {
		return nil, err
	}
	tokens = append(tokens, valueToken)

	return tokens, nil
}

// scanAttrName scans an attribute name identifier
func (l *Lexer) scanAttrName() (Token, error) {
	startPos := l.currentPosition()
	var sb strings.Builder

	// First character must be letter or underscore
	if !l.isAtEnd() && (isLetter(l.peek()) || l.peek() == '_') {
		sb.WriteByte(l.advance())
	} else {
		return Token{}, l.newUnexpectedCharError()
	}

	// Subsequent characters can be letter, digit, underscore, or hyphen
	for !l.isAtEnd() {
		ch := l.peek()
		if isLetter(ch) || isDigit(ch) || ch == '_' || ch == '-' {
			sb.WriteByte(l.advance())
		} else {
			break
		}
	}

	return NewAttrNameToken(sb.String(), startPos), nil
}

// scanAttrValue scans a quoted attribute value
func (l *Lexer) scanAttrValue() (Token, error) {
	startPos := l.currentPosition()

	// Must start with a quote
	if l.isAtEnd() {
		return Token{}, l.newUnterminatedStrError()
	}

	quote := l.peek()
	if quote != CharDoubleQuote && quote != CharSingleQuote {
		return Token{}, l.newUnexpectedCharError()
	}
	l.advance() // consume opening quote

	var sb strings.Builder
	for !l.isAtEnd() {
		ch := l.peek()

		// Check for closing quote
		if ch == quote {
			l.advance() // consume closing quote
			return NewAttrValueToken(sb.String(), startPos), nil
		}

		// Handle escape sequences within strings
		if ch == CharBackslash && l.pos+1 < len(l.source) {
			nextCh := l.source[l.pos+1]
			if nextCh == quote || nextCh == CharBackslash {
				l.advance() // skip backslash
				sb.WriteByte(l.advance())
				continue
			}
		}

		sb.WriteByte(l.advance())
	}

	return Token{}, l.newUnterminatedStrError()
}

// Helper methods

// currentPosition returns the current position
func (l *Lexer) currentPosition() Position {
	return Position{
		Offset: l.pos,
		Line:   l.line,
		Column: l.column,
	}
}

// isAtEnd returns true if we've reached the end of source
func (l *Lexer) isAtEnd() bool {
	return l.pos >= len(l.source)
}

// peek returns the current character without advancing
func (l *Lexer) peek() byte {
	if l.isAtEnd() {
		return 0
	}
	return l.source[l.pos]
}

// advance consumes and returns the current character
func (l *Lexer) advance() byte {
	if l.isAtEnd() {
		return 0
	}
	ch := l.source[l.pos]
	l.pos++
	if ch == CharNewline {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	return ch
}

// advanceN advances by n characters
func (l *Lexer) advanceN(n int) {
	for i := 0; i < n && !l.isAtEnd(); i++ {
		l.advance()
	}
}

// matchStr returns true if the remaining source starts with s
func (l *Lexer) matchStr(s string) bool {
	return strings.HasPrefix(l.source[l.pos:], s)
}

// isEscapedOpenDelim returns true if we're at an escaped open delimiter
func (l *Lexer) isEscapedOpenDelim() bool {
	return l.matchStr(l.config.escapeOpen())
}

// skipWhitespace skips whitespace characters
func (l *Lexer) skipWhitespace() {
	for !l.isAtEnd() {
		ch := l.peek()
		if ch == CharSpace || ch == CharTab || ch == CharNewline || ch == CharCarriageRet {
			l.advance()
		} else {
			break
		}
	}
}

// Character classification helpers

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

// Error helpers - these create errors with proper position context

func (l *Lexer) newUnterminatedTagError() error {
	return &LexerError{
		Message:  ErrMsgUnterminatedTag,
		Position: l.currentPosition(),
	}
}

func (l *Lexer) newUnterminatedStrError() error {
	return &LexerError{
		Message:  ErrMsgUnterminatedStr,
		Position: l.currentPosition(),
	}
}

func (l *Lexer) newInvalidTagNameError() error {
	return &LexerError{
		Message:  ErrMsgInvalidTagName,
		Position: l.currentPosition(),
	}
}

func (l *Lexer) newUnexpectedCharError() error {
	return &LexerError{
		Message:  ErrMsgUnexpectedChar,
		Position: l.currentPosition(),
	}
}

func (l *Lexer) newUnterminatedFenceError(tildes int, openPos Position) error {
	return &LexerError{
		Message:  fmt.Sprintf(ErrFmtUnterminatedFence, ErrMsgUnterminatedFence, tildes),
		Position: openPos,
	}
}

func (l *Lexer) newUnterminatedVerbatimBlockError(closeSeq string, openPos Position) error {
	return &LexerError{
		Message:  fmt.Sprintf(ErrFmtUnterminatedVerbatimBlock, ErrMsgUnterminatedVerbatimBlock, closeSeq),
		Position: openPos,
	}
}

// LexerError represents a lexer error with position
type LexerError struct {
	Message  string
	Position Position
}

func (e *LexerError) Error() string {
	return fmt.Sprintf(ErrFmtWithPosition, e.Message, e.Position.String())
}
