package internal

import (
	"fmt"
	"strings"
)

// BlockNode represents a named overridable block in template inheritance.
// Example: {~exons.block name="header"~}default content{~/exons.block~}
type BlockNode struct {
	pos       Position
	Name      string // Block name from 'name' attribute
	Children  []Node // Block content (child nodes)
	RawSource string // Original source for parent() calls
}

// Type returns NodeTypeBlock
func (n *BlockNode) Type() NodeType {
	return NodeTypeBlock
}

// Pos returns the source position
func (n *BlockNode) Pos() Position {
	return n.pos
}

// String returns a string representation
func (n *BlockNode) String() string {
	return fmt.Sprintf("BlockNode{name=%q, children=%d @ %s}", n.Name, len(n.Children), n.pos)
}

// NewBlockNode creates a new block node for template inheritance
func NewBlockNode(name string, children []Node, pos Position) *BlockNode {
	return &BlockNode{
		pos:      pos,
		Name:     name,
		Children: children,
	}
}

// InheritanceInfo stores information about template inheritance.
// This is attached to templates that use extends.
type InheritanceInfo struct {
	ParentTemplate string                // Name of the parent template
	Blocks         map[string]*BlockNode // Named blocks defined in this template
	ExtendsPos     Position              // Position of extends tag
}

// NewInheritanceInfo creates a new inheritance info
func NewInheritanceInfo(parentTemplate string, pos Position) *InheritanceInfo {
	return &InheritanceInfo{
		ParentTemplate: parentTemplate,
		Blocks:         make(map[string]*BlockNode),
		ExtendsPos:     pos,
	}
}

// HasBlock checks if a block with the given name exists
func (i *InheritanceInfo) HasBlock(name string) bool {
	_, ok := i.Blocks[name]
	return ok
}

// GetBlock retrieves a block by name
func (i *InheritanceInfo) GetBlock(name string) (*BlockNode, bool) {
	b, ok := i.Blocks[name]
	return b, ok
}

// AddBlock adds a block to the inheritance info
func (i *InheritanceInfo) AddBlock(block *BlockNode) error {
	if i.Blocks[block.Name] != nil {
		return &ParserError{
			Message:  ErrMsgBlockDuplicateName + ": " + block.Name,
			Position: block.pos,
		}
	}
	i.Blocks[block.Name] = block
	return nil
}

// ParsedTemplateWithInheritance wraps a parsed template with inheritance info
type ParsedTemplateWithInheritance struct {
	Root        *RootNode
	Inheritance *InheritanceInfo // nil if template doesn't use extends
}

// parseBlock parses a {~exons.block~}...{~/exons.block~} construct
func (p *Parser) parseBlock(attrs Attributes, pos Position) (Node, error) {
	// Get required 'name' attribute
	blockName, ok := attrs.Get(AttrName)
	if !ok {
		return nil, p.newBlockError(ErrMsgBlockMissingName, pos)
	}

	// Parse children until closing block tag
	var children []Node
	for !p.isAtEnd() {
		tok := p.current()

		// Check for closing tag {~/exons.block~}
		if tok.Type == TokenTypeBlockClose {
			if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == TokenTypeTagName {
				nextName := p.tokens[p.pos+1].Value
				if nextName == TagNameBlock {
					// Consume closing sequence
					p.advance() // BLOCK_CLOSE
					p.advance() // TAG_NAME (exons.block)
					if p.current().Type == TokenTypeCloseTag {
						p.advance() // CLOSE_TAG
					}
					return NewBlockNode(blockName, children, pos), nil
				}
			}
		}

		// Parse child node
		node, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		if node != nil {
			children = append(children, node)
		}
	}

	return nil, p.newBlockError(ErrMsgBlockNotClosed, pos)
}

// newBlockError creates an error specific to block parsing
func (p *Parser) newBlockError(message string, pos Position) error {
	return &ParserError{
		Message:  fmt.Sprintf("%s [%s]", message, TagNameBlock),
		Position: pos,
	}
}

// ExtractInheritanceInfo extracts inheritance information from a parsed AST.
// It scans the root node for extends and block tags.
// Returns nil if the template doesn't use inheritance.
func ExtractInheritanceInfo(root *RootNode) (*InheritanceInfo, error) {
	if root == nil || len(root.Children) == 0 {
		return nil, nil
	}

	var inheritanceInfo *InheritanceInfo
	foundExtends := false

	for i, child := range root.Children {
		switch node := child.(type) {
		case *TagNode:
			if node.Name == TagNameExtends {
				// Check if extends is first significant tag
				if foundExtends {
					return nil, &ParserError{Message: ErrMsgExtendsMultiple, Position: node.Pos()}
				}

				// Extends should be first or preceded only by text/whitespace
				if !isFirstSignificantNode(root.Children[:i]) {
					return nil, &ParserError{Message: ErrMsgExtendsNotFirst, Position: node.Pos()}
				}

				templateName, ok := node.Attributes.Get(AttrTemplate)
				if !ok {
					return nil, &ParserError{Message: ErrMsgExtendsMissingTemplate, Position: node.Pos()}
				}

				foundExtends = true
				inheritanceInfo = NewInheritanceInfo(templateName, node.Pos())
			} else if node.Name == TagNameParent && inheritanceInfo == nil {
				// Parent tag found outside of inheritance context
				return nil, &ParserError{Message: ErrMsgParentOutsideBlock, Position: node.Pos()}
			}

		case *BlockNode:
			// Collect blocks
			if inheritanceInfo != nil {
				if err := inheritanceInfo.AddBlock(node); err != nil {
					return nil, err
				}
			}
		}
	}

	return inheritanceInfo, nil
}

// isFirstSignificantNode checks if all preceding nodes are whitespace-only text
func isFirstSignificantNode(nodes []Node) bool {
	for _, node := range nodes {
		switch n := node.(type) {
		case *TextNode:
			if strings.TrimSpace(n.Content) != "" {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// CollectBlocks collects all block nodes from an AST
func CollectBlocks(root *RootNode) map[string]*BlockNode {
	blocks := make(map[string]*BlockNode)
	collectBlocksRecursive(root.Children, blocks)
	return blocks
}

// collectBlocksRecursive recursively collects blocks from nodes
func collectBlocksRecursive(nodes []Node, blocks map[string]*BlockNode) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *BlockNode:
			blocks[n.Name] = n
			// Also collect nested blocks
			collectBlocksRecursive(n.Children, blocks)
		case *TagNode:
			if n.Children != nil {
				collectBlocksRecursive(n.Children, blocks)
			}
		case *ConditionalNode:
			for _, branch := range n.Branches {
				collectBlocksRecursive(branch.Children, blocks)
			}
		case *ForNode:
			collectBlocksRecursive(n.Children, blocks)
		case *SwitchNode:
			for _, c := range n.Cases {
				collectBlocksRecursive(c.Children, blocks)
			}
		}
	}
}
