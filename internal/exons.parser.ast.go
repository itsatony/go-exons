package internal

import (
	"fmt"
	"sort"
	"strings"
)

// Node is the interface all AST nodes implement
type Node interface {
	// Type returns the node type identifier
	Type() NodeType
	// Position returns the source position of this node
	Pos() Position
	// String returns a human-readable representation
	String() string
}

// RootNode is the top-level container for an AST
type RootNode struct {
	Children []Node
}

// Type returns NodeTypeRoot
func (n *RootNode) Type() NodeType {
	return NodeTypeRoot
}

// Pos returns a zero position (root has no specific position)
func (n *RootNode) Pos() Position {
	return Position{Offset: 0, Line: 1, Column: 1}
}

// String returns a string representation of the root node
func (n *RootNode) String() string {
	var sb strings.Builder
	sb.WriteString("RootNode{\n")
	for i, child := range n.Children {
		sb.WriteString(fmt.Sprintf("  [%d] %s\n", i, child.String()))
	}
	sb.WriteString("}")
	return sb.String()
}

// TextNode represents literal text content
type TextNode struct {
	pos     Position
	Content string
}

// Type returns NodeTypeText
func (n *TextNode) Type() NodeType {
	return NodeTypeText
}

// Pos returns the source position
func (n *TextNode) Pos() Position {
	return n.pos
}

// String returns a string representation
func (n *TextNode) String() string {
	content := n.Content
	if len(content) > MaxStringDisplayLength {
		content = content[:TruncatedStringLength] + TruncationSuffix
	}
	return fmt.Sprintf("TextNode{%q @ %s}", content, n.pos)
}

// NewTextNode creates a new text node
func NewTextNode(content string, pos Position) *TextNode {
	return &TextNode{
		pos:     pos,
		Content: content,
	}
}

// TagNode represents a tag (self-closing or block)
type TagNode struct {
	pos        Position
	Name       string     // Tag name (e.g., "exons.var", "UserProfile")
	Attributes Attributes // Tag attributes
	Children   []Node     // Child nodes (nil for self-closing)
	SelfClose  bool       // True for self-closing tags
	RawContent string     // For raw blocks, the unparsed content
	RawSource  string     // Original tag source for keepRaw error strategy
}

// Type returns NodeTypeTag or NodeTypeRaw depending on the tag
func (n *TagNode) Type() NodeType {
	if n.Name == TagNameRaw {
		return NodeTypeRaw
	}
	return NodeTypeTag
}

// Pos returns the source position
func (n *TagNode) Pos() Position {
	return n.pos
}

// String returns a string representation
func (n *TagNode) String() string {
	var sb strings.Builder
	if n.SelfClose {
		sb.WriteString(fmt.Sprintf("TagNode{%s, self-close, attrs=%v @ %s}", n.Name, n.Attributes, n.pos))
	} else {
		sb.WriteString(fmt.Sprintf("TagNode{%s, block, attrs=%v, children=%d @ %s}", n.Name, n.Attributes, len(n.Children), n.pos))
	}
	return sb.String()
}

// IsBuiltin returns true if this is a built-in exons tag
func (n *TagNode) IsBuiltin() bool {
	return strings.HasPrefix(n.Name, TagPrefix)
}

// IsRaw returns true if this is a raw block tag
func (n *TagNode) IsRaw() bool {
	return n.Name == TagNameRaw
}

// NewSelfClosingTag creates a new self-closing tag node
func NewSelfClosingTag(name string, attrs Attributes, pos Position) *TagNode {
	return &TagNode{
		pos:        pos,
		Name:       name,
		Attributes: attrs,
		SelfClose:  true,
	}
}

// NewBlockTag creates a new block tag node
func NewBlockTag(name string, attrs Attributes, children []Node, pos Position) *TagNode {
	return &TagNode{
		pos:        pos,
		Name:       name,
		Attributes: attrs,
		Children:   children,
		SelfClose:  false,
	}
}

// NewRawBlockTag creates a new raw block tag node with raw content
func NewRawBlockTag(rawContent string, pos Position) *TagNode {
	return &TagNode{
		pos:        pos,
		Name:       TagNameRaw,
		Attributes: make(Attributes),
		SelfClose:  false,
		RawContent: rawContent,
	}
}

// Attributes is a map of tag attribute key-value pairs
type Attributes map[string]string

// Get retrieves an attribute value, returning ok=false if not found
func (a Attributes) Get(key string) (string, bool) {
	if a == nil {
		return "", false
	}
	val, ok := a[key]
	return val, ok
}

// GetDefault retrieves an attribute value with a default fallback
func (a Attributes) GetDefault(key, defaultVal string) string {
	if a == nil {
		return defaultVal
	}
	if val, ok := a[key]; ok {
		return val
	}
	return defaultVal
}

// Has checks if an attribute exists
func (a Attributes) Has(key string) bool {
	if a == nil {
		return false
	}
	_, ok := a[key]
	return ok
}

// Keys returns all attribute keys in sorted order
func (a Attributes) Keys() []string {
	if a == nil {
		return nil
	}
	keys := make([]string, 0, len(a))
	for k := range a {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Map returns a copy of the underlying map
func (a Attributes) Map() map[string]string {
	if a == nil {
		return make(map[string]string)
	}
	result := make(map[string]string, len(a))
	for k, v := range a {
		result[k] = v
	}
	return result
}

// String returns a string representation of the attributes
func (a Attributes) String() string {
	if len(a) == 0 {
		return FmtEmptyBraces
	}
	keys := a.Keys()
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, k+FmtKeyValueSep+fmt.Sprintf("%q", a[k]))
	}
	return FmtOpenBrace + strings.Join(pairs, FmtCommaSep) + FmtCloseBrace
}

// ConditionalNode represents an if/elseif/else conditional block
type ConditionalNode struct {
	pos      Position
	Branches []ConditionalBranch
}

// ConditionalBranch represents a single branch in a conditional
type ConditionalBranch struct {
	Condition string // Expression string (empty for else)
	Children  []Node // Content to render if condition is true
	IsElse    bool   // True for the final else branch
	Pos       Position
}

// Type returns NodeTypeConditional
func (n *ConditionalNode) Type() NodeType {
	return NodeTypeConditional
}

// Pos returns the source position
func (n *ConditionalNode) Pos() Position {
	return n.pos
}

// String returns a string representation
func (n *ConditionalNode) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("ConditionalNode{branches=%d @ %s", len(n.Branches), n.pos))
	for i, branch := range n.Branches {
		if branch.IsElse {
			sb.WriteString(fmt.Sprintf(", [%d]else", i))
		} else {
			sb.WriteString(fmt.Sprintf(", [%d]if(%s)", i, branch.Condition))
		}
	}
	sb.WriteString("}")
	return sb.String()
}

// NewConditionalNode creates a new conditional node
func NewConditionalNode(branches []ConditionalBranch, pos Position) *ConditionalNode {
	return &ConditionalNode{
		pos:      pos,
		Branches: branches,
	}
}

// NewConditionalBranch creates a new conditional branch
func NewConditionalBranch(condition string, children []Node, isElse bool, pos Position) ConditionalBranch {
	return ConditionalBranch{
		Condition: condition,
		Children:  children,
		IsElse:    isElse,
		Pos:       pos,
	}
}

// ForNode represents a for loop block
type ForNode struct {
	pos      Position
	ItemVar  string // Variable name for current item (required)
	IndexVar string // Variable name for iteration index (optional)
	Source   string // Path to collection to iterate (required)
	Limit    int    // Max iterations (0 = use engine default)
	Children []Node // Loop body
}

// Type returns NodeTypeFor
func (n *ForNode) Type() NodeType {
	return NodeTypeFor
}

// Pos returns the source position
func (n *ForNode) Pos() Position {
	return n.pos
}

// String returns a string representation
func (n *ForNode) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("ForNode{item=%s", n.ItemVar))
	if n.IndexVar != "" {
		sb.WriteString(fmt.Sprintf(", index=%s", n.IndexVar))
	}
	sb.WriteString(fmt.Sprintf(", in=%s", n.Source))
	if n.Limit > 0 {
		sb.WriteString(fmt.Sprintf(", limit=%d", n.Limit))
	}
	sb.WriteString(fmt.Sprintf(", children=%d @ %s}", len(n.Children), n.pos))
	return sb.String()
}

// NewForNode creates a new for loop node
func NewForNode(itemVar, indexVar, source string, limit int, children []Node, pos Position) *ForNode {
	return &ForNode{
		pos:      pos,
		ItemVar:  itemVar,
		IndexVar: indexVar,
		Source:   source,
		Limit:    limit,
		Children: children,
	}
}

// SwitchNode represents a switch/case block
type SwitchNode struct {
	pos        Position
	Expression string       // The expression to switch on
	Cases      []SwitchCase // Case branches
	Default    *SwitchCase  // Optional default case (nil if none)
}

// SwitchCase represents a single case in a switch
type SwitchCase struct {
	Value     string   // For value comparison (mutually exclusive with Eval)
	Eval      string   // For expression evaluation (mutually exclusive with Value)
	Children  []Node   // Content to render if matched
	IsDefault bool     // True for the default case
	Pos       Position // Position of this case
}

// Type returns NodeTypeSwitch
func (n *SwitchNode) Type() NodeType {
	return NodeTypeSwitch
}

// Pos returns the source position
func (n *SwitchNode) Pos() Position {
	return n.pos
}

// String returns a string representation
func (n *SwitchNode) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("SwitchNode{expr=%s, cases=%d", n.Expression, len(n.Cases)))
	if n.Default != nil {
		sb.WriteString(", hasDefault=true")
	}
	sb.WriteString(fmt.Sprintf(" @ %s}", n.pos))
	return sb.String()
}

// NewSwitchNode creates a new switch node
func NewSwitchNode(expr string, cases []SwitchCase, defaultCase *SwitchCase, pos Position) *SwitchNode {
	return &SwitchNode{
		pos:        pos,
		Expression: expr,
		Cases:      cases,
		Default:    defaultCase,
	}
}

// NewSwitchCase creates a new switch case
func NewSwitchCase(value, eval string, children []Node, isDefault bool, pos Position) SwitchCase {
	return SwitchCase{
		Value:     value,
		Eval:      eval,
		Children:  children,
		IsDefault: isDefault,
		Pos:       pos,
	}
}
