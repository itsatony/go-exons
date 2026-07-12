package internal

import (
	"fmt"
	"log/slog"
	"strconv"
)

// Parser produces an AST from a token stream
type Parser struct {
	tokens []Token
	source string // Original source for raw text extraction
	pos    int
	logger *slog.Logger
	config LexerConfig // Delimiter config for raw-source offset math
}

// NewParser creates a new parser for the given token stream
func NewParser(tokens []Token, logger *slog.Logger) *Parser {
	return NewParserWithSource(tokens, StringValueEmpty, DefaultLexerConfig(), logger)
}

// NewParserWithSource creates a new parser with source for raw text capture.
// The config must match the lexer config the tokens were produced with so
// RawSource offsets line up under custom delimiters.
func NewParserWithSource(tokens []Token, source string, config LexerConfig, logger *slog.Logger) *Parser {
	if logger == nil {
		logger = slog.Default()
	}
	logger.Debug(LogMsgParserCreated, slog.Int(LogFieldTokens, len(tokens)))
	return &Parser{
		tokens: tokens,
		source: source,
		pos:    0,
		logger: logger,
		config: config,
	}
}

// extractRawSource extracts the original source text between two positions
func (p *Parser) extractRawSource(startOffset, endOffset int) string {
	if p.source == StringValueEmpty {
		return StringValueEmpty
	}
	if startOffset < 0 || endOffset > len(p.source) || startOffset >= endOffset {
		return StringValueEmpty
	}
	return p.source[startOffset:endOffset]
}

// Parse produces the AST root node from the token stream
func (p *Parser) Parse() (*RootNode, error) {
	p.logger.Debug(LogMsgParserStart)

	nodes, err := p.parseNodes()
	if err != nil {
		return nil, err
	}

	// A stray top-level block close (e.g. "{~/x~}" with no open) stops
	// parseNodes early; surface it instead of silently dropping the rest.
	if !p.isAtEnd() {
		return nil, p.newUnexpectedTokenError(p.current())
	}

	root := &RootNode{Children: nodes}
	p.logger.Debug(LogMsgParserEnd, slog.Int(LogFieldNodes, len(nodes)))
	return root, nil
}

// parseNodes parses a sequence of nodes until EOF or a closing tag
func (p *Parser) parseNodes() ([]Node, error) {
	var nodes []Node

	for !p.isAtEnd() && !p.isBlockClose() {
		node, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		if node != nil {
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

// parseNode parses a single node (text or tag)
func (p *Parser) parseNode() (Node, error) {
	tok := p.current()

	switch tok.Type {
	case TokenTypeText:
		return p.parseText()
	case TokenTypeOpenTag:
		return p.parseTag()
	case TokenTypeBlockClose:
		// Block close is handled by parseBlockTag
		return nil, nil
	case TokenTypeEOF:
		return nil, nil
	default:
		return nil, p.newUnexpectedTokenError(tok)
	}
}

// parseText parses a text node
func (p *Parser) parseText() (*TextNode, error) {
	tok := p.advance()
	return NewTextNode(tok.Value, tok.Position), nil
}

// parseTag parses a tag (self-closing or block)
func (p *Parser) parseTag() (Node, error) {
	openTok := p.advance() // consume OPEN_TAG

	// Get tag name
	nameTok := p.current()
	if nameTok.Type != TokenTypeTagName {
		return nil, p.newExpectedTokenError(TokenTypeTagName, nameTok)
	}
	p.advance() // consume TAG_NAME

	tagName := nameTok.Value
	pos := openTok.Position

	// Parse attributes
	attrs, err := p.parseAttributes()
	if err != nil {
		return nil, err
	}

	// Check how the tag ends
	endTok := p.current()

	switch endTok.Type {
	case TokenTypeSelfClose:
		p.advance() // consume SELF_CLOSE
		tag := NewSelfClosingTag(tagName, attrs, pos)
		// Capture raw source for keepRaw error strategy
		endOffset := endTok.Position.Offset + len(p.config.selfClose())
		tag.RawSource = p.extractRawSource(pos.Offset, endOffset)
		return tag, nil

	case TokenTypeCloseTag:
		p.advance() // consume CLOSE_TAG
		// This is a block tag - parse content and closing
		return p.parseBlockTag(tagName, attrs, pos)

	default:
		return nil, p.newUnexpectedTokenError(endTok)
	}
}

// parseBlockTag parses the content and closing of a block tag
func (p *Parser) parseBlockTag(tagName string, attrs Attributes, pos Position) (Node, error) {
	// Special handling for raw blocks
	if tagName == TagNameRaw {
		return p.parseRawBlock(pos)
	}

	// Special handling for conditionals
	if tagName == TagNameIf {
		return p.parseConditional(attrs, pos)
	}

	// Special handling for comments - discard content entirely
	if tagName == TagNameComment {
		return p.parseCommentBlock()
	}

	// Special handling for for loops
	if tagName == TagNameFor {
		return p.parseFor(attrs, pos)
	}

	// Special handling for switch/case
	if tagName == TagNameSwitch {
		return p.parseSwitch(attrs, pos)
	}

	// Special handling for block (template inheritance)
	if tagName == TagNameBlock {
		return p.parseBlock(attrs, pos)
	}

	// Parse children
	children, err := p.parseNodes()
	if err != nil {
		return nil, err
	}

	// Expect block close
	if !p.isBlockClose() {
		return nil, p.newMismatchedTagError(tagName, "")
	}

	// Consume BLOCK_CLOSE
	p.advance()

	// Get closing tag name
	closeNameTok := p.current()
	if closeNameTok.Type != TokenTypeTagName {
		return nil, p.newExpectedTokenError(TokenTypeTagName, closeNameTok)
	}
	closeName := closeNameTok.Value
	p.advance() // consume TAG_NAME

	// Verify matching
	if closeName != tagName {
		return nil, p.newMismatchedTagError(tagName, closeName)
	}

	// Consume CLOSE_TAG
	closeTok := p.current()
	if closeTok.Type != TokenTypeCloseTag {
		return nil, p.newExpectedTokenError(TokenTypeCloseTag, closeTok)
	}
	p.advance()

	tag := NewBlockTag(tagName, attrs, children, pos)
	// Capture raw source for keepRaw error strategy (full block from open to close)
	endOffset := closeTok.Position.Offset + len(p.config.CloseDelim)
	tag.RawSource = p.extractRawSource(pos.Offset, endOffset)
	return tag, nil
}

// parseConditional parses an if/elseif/else conditional block
func (p *Parser) parseConditional(ifAttrs Attributes, pos Position) (*ConditionalNode, error) {
	var branches []ConditionalBranch

	// Get the condition from the if tag
	condition, ok := ifAttrs.Get(AttrEval)
	if !ok {
		return nil, p.newConditionError(ErrMsgCondMissingEval, pos)
	}

	// Parse the first branch (if)
	children, nextTag, nextAttrs, nextPos, err := p.parseConditionalBranch()
	if err != nil {
		return nil, err
	}

	branches = append(branches, NewConditionalBranch(condition, children, false, pos))

	// Process subsequent branches (elseif, else)
	for nextTag != "" {
		switch nextTag {
		case TagNameElseIf:
			// elseif needs an eval attribute
			condition, ok := nextAttrs.Get(AttrEval)
			if !ok {
				return nil, p.newConditionError(ErrMsgCondMissingEval, nextPos)
			}

			children, nextTag, nextAttrs, nextPos, err = p.parseConditionalBranch()
			if err != nil {
				return nil, err
			}

			branches = append(branches, NewConditionalBranch(condition, children, false, nextPos))

		case TagNameElse:
			// else cannot have an eval attribute
			if nextAttrs.Has(AttrEval) {
				return nil, p.newConditionError(ErrMsgCondInvalidElse, nextPos)
			}

			children, nextTag, nextAttrs, nextPos, err = p.parseConditionalBranch()
			if err != nil {
				return nil, err
			}

			// else must be the last branch
			if nextTag != "" {
				return nil, p.newConditionError(ErrMsgCondElseNotLast, nextPos)
			}

			branches = append(branches, NewConditionalBranch("", children, true, nextPos))

		default:
			// Unexpected tag inside conditional
			return nil, p.newConditionError(ErrMsgCondUnexpectedTag, nextPos)
		}
	}

	return NewConditionalNode(branches, pos), nil
}

// parseConditionalBranch parses nodes until we hit elseif, else, or the closing if tag
// Returns: children nodes, next tag name (empty if closing), next tag attrs, next tag position, error
func (p *Parser) parseConditionalBranch() ([]Node, string, Attributes, Position, error) {
	var children []Node

	for !p.isAtEnd() {
		tok := p.current()

		// Check for block close (could be elseif, else, or /if)
		if tok.Type == TokenTypeBlockClose {
			// This is {~/ - check if it's the closing /if
			if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == TokenTypeTagName {
				nextName := p.tokens[p.pos+1].Value
				if nextName == TagNameIf {
					// This is the closing {~/exons.if~}
					p.advance() // consume BLOCK_CLOSE

					closeNameTok := p.current()
					p.advance() // consume TAG_NAME

					closeTok := p.current()
					if closeTok.Type != TokenTypeCloseTag {
						return nil, "", nil, Position{}, p.newExpectedTokenError(TokenTypeCloseTag, closeTok)
					}
					p.advance() // consume CLOSE_TAG

					return children, "", nil, closeNameTok.Position, nil
				}
			}
		}

		// Check for open tag (could be elseif or else, or a normal nested tag)
		if tok.Type == TokenTypeOpenTag {
			// Peek at the tag name
			if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == TokenTypeTagName {
				nextName := p.tokens[p.pos+1].Value
				if nextName == TagNameElseIf || nextName == TagNameElse {
					// This is a branch boundary
					openPos := tok.Position
					p.advance() // consume OPEN_TAG

					_ = p.current()
					p.advance() // consume TAG_NAME

					// Parse attributes
					attrs, err := p.parseAttributes()
					if err != nil {
						return nil, "", nil, Position{}, err
					}

					// Consume CLOSE_TAG
					closeTok := p.current()
					if closeTok.Type != TokenTypeCloseTag {
						return nil, "", nil, Position{}, p.newExpectedTokenError(TokenTypeCloseTag, closeTok)
					}
					p.advance()

					return children, nextName, attrs, openPos, nil
				}
			}
		}

		// Parse a normal node
		node, err := p.parseNode()
		if err != nil {
			return nil, "", nil, Position{}, err
		}
		if node != nil {
			children = append(children, node)
		}
	}

	// Reached EOF without finding closing tag
	return nil, "", nil, Position{}, p.newConditionError(ErrMsgCondNotClosed, Position{})
}

// newConditionError creates a conditional-specific error
func (p *Parser) newConditionError(message string, pos Position) error {
	return &ParserError{
		Message:  message,
		Position: pos,
	}
}

// parseFor parses a for loop block
func (p *Parser) parseFor(attrs Attributes, pos Position) (*ForNode, error) {
	// Get required 'item' attribute
	itemVar, ok := attrs.Get(AttrItem)
	if !ok || itemVar == "" {
		return nil, p.newForError(ErrMsgForMissingItem, pos)
	}

	// Get required 'in' attribute
	source, ok := attrs.Get(AttrIn)
	if !ok || source == "" {
		return nil, p.newForError(ErrMsgForMissingIn, pos)
	}

	// Get optional 'index' attribute
	indexVar, _ := attrs.Get(AttrIndex)

	// Get optional 'limit' attribute
	limit := 0
	if limitStr, hasLimit := attrs.Get(AttrLimit); hasLimit {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit < 0 {
			return nil, p.newForError(ErrMsgForInvalidLimit, pos)
		}
		limit = parsedLimit
	}

	// Parse loop body until {~/exons.for~}
	children, err := p.parseForBody()
	if err != nil {
		return nil, err
	}

	return NewForNode(itemVar, indexVar, source, limit, children, pos), nil
}

// parseForBody parses the body of a for loop until the closing tag
func (p *Parser) parseForBody() ([]Node, error) {
	var children []Node

	for !p.isAtEnd() {
		tok := p.current()

		// Check for closing {~/exons.for~}
		if tok.Type == TokenTypeBlockClose {
			if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == TokenTypeTagName {
				nextName := p.tokens[p.pos+1].Value
				if nextName == TagNameFor {
					// Consume closing sequence
					p.advance() // BLOCK_CLOSE

					_ = p.current() // TAG_NAME (already verified)
					p.advance()     // consume TAG_NAME

					closeTok := p.current()
					if closeTok.Type != TokenTypeCloseTag {
						return nil, p.newExpectedTokenError(TokenTypeCloseTag, closeTok)
					}
					p.advance() // CLOSE_TAG

					return children, nil
				}
			}
		}

		// Parse a normal node
		node, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		if node != nil {
			children = append(children, node)
		}
	}

	// Reached EOF without finding closing tag
	return nil, p.newForError(ErrMsgForNotClosed, Position{})
}

// newForError creates a for-loop specific error
func (p *Parser) newForError(message string, pos Position) error {
	return &ParserError{
		Message:  message,
		Position: pos,
	}
}

// parseSwitch parses a switch/case block
func (p *Parser) parseSwitch(attrs Attributes, pos Position) (*SwitchNode, error) {
	// Get required 'eval' attribute for the switch expression
	expression, ok := attrs.Get(AttrEval)
	if !ok || expression == "" {
		return nil, p.newSwitchError(ErrMsgSwitchMissingEval, pos)
	}

	var cases []SwitchCase
	var defaultCase *SwitchCase

	// Parse cases until we hit the closing switch tag
	for !p.isAtEnd() {
		tok := p.current()

		// Check for closing {~/exons.switch~}
		if tok.Type == TokenTypeBlockClose {
			if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == TokenTypeTagName {
				nextName := p.tokens[p.pos+1].Value
				if nextName == TagNameSwitch {
					// Consume closing sequence
					p.advance() // BLOCK_CLOSE
					p.advance() // TAG_NAME (exons.switch)

					closeTok := p.current()
					if closeTok.Type != TokenTypeCloseTag {
						return nil, p.newExpectedTokenError(TokenTypeCloseTag, closeTok)
					}
					p.advance() // CLOSE_TAG

					return NewSwitchNode(expression, cases, defaultCase, pos), nil
				}
			}
		}

		// Skip whitespace text nodes between cases
		if tok.Type == TokenTypeText {
			// Only allow whitespace text between cases
			p.advance()
			continue
		}

		// Expect an open tag for case or casedefault
		if tok.Type != TokenTypeOpenTag {
			return nil, p.newSwitchError(ErrMsgSwitchInvalidCaseTag, tok.Position)
		}

		// Parse the case/casedefault tag
		caseNode, isDefault, err := p.parseSwitchCase()
		if err != nil {
			return nil, err
		}

		if isDefault {
			// Check for duplicate default
			if defaultCase != nil {
				return nil, p.newSwitchError(ErrMsgSwitchDuplicateDefault, caseNode.Pos)
			}
			defaultCase = &caseNode
		} else {
			// Check that default wasn't already defined (must be last)
			if defaultCase != nil {
				return nil, p.newSwitchError(ErrMsgSwitchDefaultNotLast, caseNode.Pos)
			}
			cases = append(cases, caseNode)
		}
	}

	// Reached EOF without finding closing switch tag
	return nil, p.newSwitchError(ErrMsgSwitchNotClosed, pos)
}

// parseSwitchCase parses a single case or casedefault within a switch block
// Returns: case node, isDefault flag, error
func (p *Parser) parseSwitchCase() (SwitchCase, bool, error) {
	openTok := p.advance() // consume OPEN_TAG
	casePos := openTok.Position

	// Get tag name
	nameTok := p.current()
	if nameTok.Type != TokenTypeTagName {
		return SwitchCase{}, false, p.newExpectedTokenError(TokenTypeTagName, nameTok)
	}
	tagName := nameTok.Value
	p.advance() // consume TAG_NAME

	// Validate it's a case or casedefault tag
	if tagName != TagNameCase && tagName != TagNameCaseDefault {
		return SwitchCase{}, false, p.newSwitchError(ErrMsgSwitchInvalidCaseTag, casePos)
	}

	isDefault := tagName == TagNameCaseDefault

	// Parse attributes
	attrs, err := p.parseAttributes()
	if err != nil {
		return SwitchCase{}, false, err
	}

	// Validate attributes based on case type
	var value, eval string
	if !isDefault {
		// Regular case needs either value or eval
		value, _ = attrs.Get(AttrValue)
		eval, _ = attrs.Get(AttrEval)
		if value == "" && eval == "" {
			return SwitchCase{}, false, p.newSwitchError(ErrMsgSwitchMissingValue, casePos)
		}
	}

	// Consume the closing tag of the case opener
	closeTok := p.current()
	if closeTok.Type != TokenTypeCloseTag {
		return SwitchCase{}, false, p.newExpectedTokenError(TokenTypeCloseTag, closeTok)
	}
	p.advance() // consume CLOSE_TAG

	// Parse the case body until the closing tag
	children, err := p.parseSwitchCaseBody(tagName)
	if err != nil {
		return SwitchCase{}, false, err
	}

	return NewSwitchCase(value, eval, children, isDefault, casePos), isDefault, nil
}

// parseSwitchCaseBody parses the body of a case until its closing tag
func (p *Parser) parseSwitchCaseBody(closingTag string) ([]Node, error) {
	var children []Node

	for !p.isAtEnd() {
		tok := p.current()

		// Check for closing {~/exons.case~} or {~/exons.casedefault~}
		if tok.Type == TokenTypeBlockClose {
			if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == TokenTypeTagName {
				nextName := p.tokens[p.pos+1].Value
				if nextName == closingTag {
					// Consume closing sequence
					p.advance() // BLOCK_CLOSE
					p.advance() // TAG_NAME

					closeTok := p.current()
					if closeTok.Type != TokenTypeCloseTag {
						return nil, p.newExpectedTokenError(TokenTypeCloseTag, closeTok)
					}
					p.advance() // CLOSE_TAG

					return children, nil
				}
				// If we hit a different closing tag (like {~/exons.switch~}), the case was never closed
				if nextName == TagNameSwitch {
					return nil, p.newSwitchError(ErrMsgSwitchCaseNotClosed, tok.Position)
				}
			}
		}

		// Parse a normal node
		node, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		if node != nil {
			children = append(children, node)
		}
	}

	// Reached EOF without finding closing tag
	return nil, p.newSwitchError(ErrMsgSwitchCaseNotClosed, Position{})
}

// newSwitchError creates a switch-specific error
func (p *Parser) newSwitchError(message string, pos Position) error {
	return &ParserError{
		Message:  message,
		Position: pos,
	}
}

// parseRawBlock parses a raw block. The lexer scans raw bodies verbatim and
// emits at most one TEXT token followed by the canonical close triplet, so
// this is a trivial consumer; content round-trips byte-for-byte.
func (p *Parser) parseRawBlock(pos Position) (*TagNode, error) {
	var rawContent string
	if p.current().Type == TokenTypeText {
		rawContent = p.current().Value
		p.advance()
	}

	closeTok, err := p.consumeVerbatimClose(TagNameRaw)
	if err != nil {
		return nil, err
	}

	tag := NewRawBlockTag(rawContent, pos)
	// Capture raw source for keepRaw error strategy
	endOffset := closeTok.Position.Offset + len(p.config.CloseDelim)
	tag.RawSource = p.extractRawSource(pos.Offset, endOffset)
	return tag, nil
}

// parseCommentBlock parses a comment block - content is discarded
func (p *Parser) parseCommentBlock() (Node, error) {
	if p.current().Type == TokenTypeText {
		p.advance() // discard body
	}

	if _, err := p.consumeVerbatimClose(TagNameComment); err != nil {
		return nil, err
	}

	// Return nil - comment nodes produce no output
	return nil, nil
}

// consumeVerbatimClose consumes the close triplet of a verbatim block
// (BLOCK_CLOSE, TAG_NAME, CLOSE_TAG) and returns the CLOSE_TAG token.
func (p *Parser) consumeVerbatimClose(tagName string) (Token, error) {
	if !p.isBlockClose() {
		return Token{}, p.newMismatchedTagError(tagName, StringValueEmpty)
	}
	p.advance() // BLOCK_CLOSE

	closeNameTok := p.current()
	if closeNameTok.Type != TokenTypeTagName || closeNameTok.Value != tagName {
		return Token{}, p.newMismatchedTagError(tagName, closeNameTok.Value)
	}
	p.advance() // TAG_NAME

	closeTok := p.current()
	if closeTok.Type != TokenTypeCloseTag {
		return Token{}, p.newExpectedTokenError(TokenTypeCloseTag, closeTok)
	}
	p.advance() // CLOSE_TAG

	return closeTok, nil
}

// parseAttributes parses tag attributes until we hit a closing token
func (p *Parser) parseAttributes() (Attributes, error) {
	attrs := make(Attributes)

	for !p.isAtEnd() {
		tok := p.current()

		// Stop at closing tokens
		if tok.Type == TokenTypeSelfClose || tok.Type == TokenTypeCloseTag {
			break
		}

		// Expect attribute name
		if tok.Type != TokenTypeAttrName {
			return nil, p.newUnexpectedTokenError(tok)
		}
		attrName := tok.Value
		p.advance()

		// Expect equals
		if p.current().Type != TokenTypeEquals {
			return nil, p.newExpectedTokenError(TokenTypeEquals, p.current())
		}
		p.advance()

		// Expect value
		if p.current().Type != TokenTypeAttrValue {
			return nil, p.newExpectedTokenError(TokenTypeAttrValue, p.current())
		}
		attrValue := p.current().Value
		p.advance()

		attrs[attrName] = attrValue
	}

	return attrs, nil
}

// Helper methods

// current returns the current token
func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenTypeEOF}
	}
	return p.tokens[p.pos]
}

// advance consumes and returns the current token
func (p *Parser) advance() Token {
	tok := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

// isAtEnd returns true if we've reached EOF
func (p *Parser) isAtEnd() bool {
	return p.current().Type == TokenTypeEOF
}

// isBlockClose returns true if current token is BLOCK_CLOSE
func (p *Parser) isBlockClose() bool {
	return p.current().Type == TokenTypeBlockClose
}

// Error helpers

func (p *Parser) newUnexpectedTokenError(tok Token) error {
	return &ParserError{
		Message:  ErrMsgUnexpectedToken,
		Position: tok.Position,
		Token:    tok,
	}
}

func (p *Parser) newExpectedTokenError(expected TokenType, actual Token) error {
	return &ParserError{
		Message:  ErrMsgExpectedToken,
		Position: actual.Position,
		Expected: expected,
		Token:    actual,
	}
}

func (p *Parser) newMismatchedTagError(expected, actual string) error {
	return &ParserError{
		Message:     ErrMsgMismatchedTag,
		Position:    p.current().Position,
		ExpectedTag: expected,
		ActualTag:   actual,
	}
}

// ParserError represents a parser error with context
type ParserError struct {
	Message     string
	Position    Position
	Token       Token
	Expected    TokenType
	ExpectedTag string
	ActualTag   string
}

func (e *ParserError) Error() string {
	return fmt.Sprintf(ErrFmtWithPosition, e.Message, e.Position.String())
}
