package internal

import (
	"context"
)

// InheritanceResolver handles template inheritance resolution.
// It resolves parent templates and merges block content.
type InheritanceResolver struct {
	engine           TemplateExecutor
	maxDepth         int
	templateResolver TemplateSourceResolver
	inheritanceChain []string // Track templates to detect circular inheritance
}

// TemplateSourceResolver provides access to raw template sources
type TemplateSourceResolver interface {
	GetTemplateSource(name string) (string, bool)
}

// NewInheritanceResolver creates a new inheritance resolver
func NewInheritanceResolver(engine TemplateExecutor, templateResolver TemplateSourceResolver, maxDepth int) *InheritanceResolver {
	if maxDepth <= 0 {
		maxDepth = DefaultMaxInheritanceDepth
	}
	return &InheritanceResolver{
		engine:           engine,
		maxDepth:         maxDepth,
		templateResolver: templateResolver,
		inheritanceChain: make([]string, 0),
	}
}

// ResolveInheritance resolves template inheritance by merging child blocks into parent.
// Returns the final AST with all inheritance resolved.
func (r *InheritanceResolver) ResolveInheritance(
	ctx context.Context,
	childRoot *RootNode,
	childInfo *InheritanceInfo,
	currentDepth int,
) (*RootNode, error) {
	if currentDepth > r.maxDepth {
		return nil, NewBuiltinError(ErrMsgInheritanceDepthExceeded, TagNameExtends)
	}

	// Check for circular inheritance
	parentName := childInfo.ParentTemplate
	for _, ancestor := range r.inheritanceChain {
		if ancestor == parentName {
			return nil, NewBuiltinError(ErrMsgCircularInheritance, TagNameExtends)
		}
	}
	r.inheritanceChain = append(r.inheritanceChain, parentName)
	defer func() {
		r.inheritanceChain = r.inheritanceChain[:len(r.inheritanceChain)-1]
	}()

	// Get parent template source
	if r.templateResolver == nil {
		return nil, NewBuiltinError(ErrMsgEngineNotAvailable, TagNameExtends)
	}

	parentSource, exists := r.templateResolver.GetTemplateSource(parentName)
	if !exists {
		return nil, NewTemplateNotFoundBuiltinError(parentName)
	}

	// Parse parent template
	parentRoot, parentInfo, err := r.parseTemplateWithInheritance(parentSource)
	if err != nil {
		return nil, err
	}

	// If parent also extends another template, resolve recursively
	if parentInfo != nil {
		parentRoot, err = r.ResolveInheritance(ctx, parentRoot, parentInfo, currentDepth+1)
		if err != nil {
			return nil, err
		}
	}

	// Merge child blocks into parent
	mergedRoot := r.mergeBlocks(parentRoot, childInfo.Blocks)
	return mergedRoot, nil
}

// parseTemplateWithInheritance parses a template and extracts inheritance info
func (r *InheritanceResolver) parseTemplateWithInheritance(source string) (*RootNode, *InheritanceInfo, error) {
	// Create a lexer and parser for the parent template
	lexer := NewLexer(source, nil)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, nil, err
	}

	parser := NewParserWithSource(tokens, source, nil)
	root, err := parser.Parse()
	if err != nil {
		return nil, nil, err
	}

	// Extract inheritance info
	info, err := ExtractInheritanceInfo(root)
	if err != nil {
		return nil, nil, err
	}

	return root, info, nil
}

// mergeBlocks merges child block definitions into parent AST
func (r *InheritanceResolver) mergeBlocks(parentRoot *RootNode, childBlocks map[string]*BlockNode) *RootNode {
	// Clone the parent root and replace matching blocks
	newChildren := r.mergeBlocksInNodes(parentRoot.Children, childBlocks)
	return &RootNode{Children: newChildren}
}

// mergeBlocksInNodes recursively merges blocks in a node slice
func (r *InheritanceResolver) mergeBlocksInNodes(nodes []Node, childBlocks map[string]*BlockNode) []Node {
	result := make([]Node, 0, len(nodes))

	for _, node := range nodes {
		switch n := node.(type) {
		case *BlockNode:
			// Check if child provides an override
			if childBlock, ok := childBlocks[n.Name]; ok {
				// Replace with child's block content
				// Note: We need to handle exons.parent calls within child block
				mergedBlock := r.resolveParentCalls(childBlock, n)
				result = append(result, mergedBlock)
			} else {
				// Keep parent's default block content
				result = append(result, n)
			}

		case *TagNode:
			// Skip extends tags in output
			if n.Name == TagNameExtends {
				continue
			}
			// Recursively process children
			if n.Children != nil {
				n.Children = r.mergeBlocksInNodes(n.Children, childBlocks)
			}
			result = append(result, n)

		case *ConditionalNode:
			// Process conditional branches
			for i := range n.Branches {
				n.Branches[i].Children = r.mergeBlocksInNodes(n.Branches[i].Children, childBlocks)
			}
			result = append(result, n)

		case *ForNode:
			// Process for loop body
			n.Children = r.mergeBlocksInNodes(n.Children, childBlocks)
			result = append(result, n)

		case *SwitchNode:
			// Process switch cases
			for i := range n.Cases {
				n.Cases[i].Children = r.mergeBlocksInNodes(n.Cases[i].Children, childBlocks)
			}
			result = append(result, n)

		default:
			result = append(result, node)
		}
	}

	return result
}

// resolveParentCalls resolves exons.parent tags within a child block
// by inserting the parent block's content
func (r *InheritanceResolver) resolveParentCalls(childBlock, parentBlock *BlockNode) *BlockNode {
	newChildren := r.resolveParentCallsInNodes(childBlock.Children, parentBlock.Children)
	return &BlockNode{
		pos:       childBlock.pos,
		Name:      childBlock.Name,
		Children:  newChildren,
		RawSource: childBlock.RawSource,
	}
}

// resolveParentCallsInNodes replaces exons.parent tags with parent content
func (r *InheritanceResolver) resolveParentCallsInNodes(nodes []Node, parentContent []Node) []Node {
	result := make([]Node, 0, len(nodes))

	for _, node := range nodes {
		switch n := node.(type) {
		case *TagNode:
			if n.Name == TagNameParent {
				// Replace with parent content
				result = append(result, parentContent...)
			} else {
				// Recursively process children
				if n.Children != nil {
					n.Children = r.resolveParentCallsInNodes(n.Children, parentContent)
				}
				result = append(result, n)
			}

		case *BlockNode:
			// Recursively process nested blocks
			n.Children = r.resolveParentCallsInNodes(n.Children, parentContent)
			result = append(result, n)

		case *ConditionalNode:
			for i := range n.Branches {
				n.Branches[i].Children = r.resolveParentCallsInNodes(n.Branches[i].Children, parentContent)
			}
			result = append(result, n)

		case *ForNode:
			n.Children = r.resolveParentCallsInNodes(n.Children, parentContent)
			result = append(result, n)

		case *SwitchNode:
			for i := range n.Cases {
				n.Cases[i].Children = r.resolveParentCallsInNodes(n.Cases[i].Children, parentContent)
			}
			result = append(result, n)

		default:
			result = append(result, node)
		}
	}

	return result
}
