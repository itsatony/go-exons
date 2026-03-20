package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock types for InheritanceResolver tests ---

// mockTemplateSourceResolver implements TemplateSourceResolver for testing.
type mockTemplateSourceResolver struct {
	sources map[string]string
}

func newMockTemplateSourceResolver(sources map[string]string) *mockTemplateSourceResolver {
	if sources == nil {
		sources = make(map[string]string)
	}
	return &mockTemplateSourceResolver{sources: sources}
}

func (m *mockTemplateSourceResolver) GetTemplateSource(name string) (string, bool) {
	source, ok := m.sources[name]
	return source, ok
}

// --- NewInheritanceResolver Tests ---

func TestNewInheritanceResolver(t *testing.T) {
	t.Run("positive max depth is preserved", func(t *testing.T) {
		resolver := NewInheritanceResolver(nil, nil, 5)
		assert.Equal(t, 5, resolver.maxDepth)
	})

	t.Run("zero max depth uses default", func(t *testing.T) {
		resolver := NewInheritanceResolver(nil, nil, 0)
		assert.Equal(t, DefaultMaxInheritanceDepth, resolver.maxDepth)
	})

	t.Run("negative max depth uses default", func(t *testing.T) {
		resolver := NewInheritanceResolver(nil, nil, -1)
		assert.Equal(t, DefaultMaxInheritanceDepth, resolver.maxDepth)
	})

	t.Run("inheritance chain starts empty", func(t *testing.T) {
		resolver := NewInheritanceResolver(nil, nil, 5)
		assert.NotNil(t, resolver.inheritanceChain)
		assert.Empty(t, resolver.inheritanceChain)
	})

	t.Run("engine and template resolver stored", func(t *testing.T) {
		engine := newMockTemplateExecutor()
		tsr := newMockTemplateSourceResolver(nil)
		resolver := NewInheritanceResolver(engine, tsr, 3)
		assert.Equal(t, engine, resolver.engine)
		assert.Equal(t, tsr, resolver.templateResolver)
	})
}

// --- ResolveInheritance Tests ---

func TestResolveInheritance_MaxDepthExceeded(t *testing.T) {
	tsr := newMockTemplateSourceResolver(nil)
	resolver := NewInheritanceResolver(nil, tsr, 2)
	pos := Position{Line: 1, Column: 1, Offset: 0}

	childRoot := &RootNode{Children: []Node{NewTextNode("child", pos)}}
	childInfo := NewInheritanceInfo("parent", pos)

	_, err := resolver.ResolveInheritance(context.Background(), childRoot, childInfo, 3)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgInheritanceDepthExceeded)
}

func TestResolveInheritance_ExactlyAtMaxDepth(t *testing.T) {
	tsr := newMockTemplateSourceResolver(nil)
	resolver := NewInheritanceResolver(nil, tsr, 2)
	pos := Position{Line: 1, Column: 1, Offset: 0}

	childRoot := &RootNode{Children: []Node{NewTextNode("child", pos)}}
	childInfo := NewInheritanceInfo("parent", pos)

	// Depth equal to maxDepth should still be within bounds (not exceeded)
	// The check is currentDepth > r.maxDepth
	tsr.sources["parent"] = "Parent content"

	result, err := resolver.ResolveInheritance(context.Background(), childRoot, childInfo, 2)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestResolveInheritance_CircularInheritance(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	t.Run("direct circular reference", func(t *testing.T) {
		tsr := newMockTemplateSourceResolver(map[string]string{
			// Parent extends itself (but we detect it through chain tracking)
			"parent": "Parent content",
		})
		resolver := NewInheritanceResolver(nil, tsr, 10)
		// Pre-load the chain to simulate that "parent" is already in the ancestry
		resolver.inheritanceChain = append(resolver.inheritanceChain, "parent")

		childRoot := &RootNode{Children: []Node{NewTextNode("child", pos)}}
		childInfo := NewInheritanceInfo("parent", pos)

		_, err := resolver.ResolveInheritance(context.Background(), childRoot, childInfo, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgCircularInheritance)
	})

	t.Run("indirect circular reference a -> b -> a", func(t *testing.T) {
		// Template B extends A, and Template A extends B
		tsr := newMockTemplateSourceResolver(map[string]string{
			"base": `{~exons.extends template="child-template" /~}{~exons.block name="content"~}base{~/exons.block~}`,
		})
		resolver := NewInheritanceResolver(nil, tsr, 10)
		resolver.inheritanceChain = append(resolver.inheritanceChain, "child-template")

		childRoot := &RootNode{Children: []Node{NewTextNode("child", pos)}}
		childInfo := NewInheritanceInfo("base", pos)

		// When resolving base, it tries to parse base, which extends "child-template"
		// But "child-template" is already in the chain, so circular is detected during
		// recursive resolution, or the direct match catches "base" in chain.
		// In this case, "base" is not in chain, so it goes through.
		// But "base" extends "child-template" which IS in the chain.
		// The resolution will detect circular on the recursive call.
		_, err := resolver.ResolveInheritance(context.Background(), childRoot, childInfo, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgCircularInheritance)
	})

	t.Run("chain resets after resolution", func(t *testing.T) {
		tsr := newMockTemplateSourceResolver(map[string]string{
			"parent": "Hello from parent",
		})
		resolver := NewInheritanceResolver(nil, tsr, 10)

		childRoot := &RootNode{Children: []Node{NewTextNode("child", pos)}}
		childInfo := NewInheritanceInfo("parent", pos)

		_, err := resolver.ResolveInheritance(context.Background(), childRoot, childInfo, 0)
		require.NoError(t, err)

		// After resolution, the chain should be clean (defer pops the entry)
		assert.Empty(t, resolver.inheritanceChain)
	})
}

func TestResolveInheritance_NilTemplateResolver(t *testing.T) {
	resolver := NewInheritanceResolver(nil, nil, 10)
	pos := Position{Line: 1, Column: 1, Offset: 0}

	childRoot := &RootNode{Children: []Node{NewTextNode("child", pos)}}
	childInfo := NewInheritanceInfo("parent", pos)

	_, err := resolver.ResolveInheritance(context.Background(), childRoot, childInfo, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgEngineNotAvailable)
}

func TestResolveInheritance_ParentTemplateNotFound(t *testing.T) {
	tsr := newMockTemplateSourceResolver(nil) // empty sources
	resolver := NewInheritanceResolver(nil, tsr, 10)
	pos := Position{Line: 1, Column: 1, Offset: 0}

	childRoot := &RootNode{Children: []Node{NewTextNode("child", pos)}}
	childInfo := NewInheritanceInfo("nonexistent-parent", pos)

	_, err := resolver.ResolveInheritance(context.Background(), childRoot, childInfo, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgTemplateNotFound)
}

func TestResolveInheritance_SimpleBlockOverride(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	// Parent template has a block with default content
	parentSource := `{~exons.block name="content"~}Default Content{~/exons.block~}`

	tsr := newMockTemplateSourceResolver(map[string]string{
		"parent": parentSource,
	})
	resolver := NewInheritanceResolver(nil, tsr, 10)

	// Child overrides the "content" block
	childBlock := NewBlockNode("content", []Node{NewTextNode("Overridden Content", pos)}, pos)
	childInfo := NewInheritanceInfo("parent", pos)
	err := childInfo.AddBlock(childBlock)
	require.NoError(t, err)

	childRoot := &RootNode{Children: []Node{
		NewSelfClosingTag(TagNameExtends, Attributes{AttrTemplate: "parent"}, pos),
		childBlock,
	}}

	result, err := resolver.ResolveInheritance(context.Background(), childRoot, childInfo, 0)
	require.NoError(t, err)
	require.NotNil(t, result)

	// The result should contain the overridden block
	require.Len(t, result.Children, 1)
	block, ok := result.Children[0].(*BlockNode)
	require.True(t, ok, "expected BlockNode, got %T", result.Children[0])
	assert.Equal(t, "content", block.Name)
	require.Len(t, block.Children, 1)

	textNode, ok := block.Children[0].(*TextNode)
	require.True(t, ok, "expected TextNode, got %T", block.Children[0])
	assert.Equal(t, "Overridden Content", textNode.Content)
}

func TestResolveInheritance_BlockNotOverridden(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	// Parent template has a block with default content
	parentSource := `{~exons.block name="header"~}Default Header{~/exons.block~}`

	tsr := newMockTemplateSourceResolver(map[string]string{
		"parent": parentSource,
	})
	resolver := NewInheritanceResolver(nil, tsr, 10)

	// Child provides no block overrides
	childInfo := NewInheritanceInfo("parent", pos)

	childRoot := &RootNode{Children: []Node{
		NewSelfClosingTag(TagNameExtends, Attributes{AttrTemplate: "parent"}, pos),
	}}

	result, err := resolver.ResolveInheritance(context.Background(), childRoot, childInfo, 0)
	require.NoError(t, err)
	require.NotNil(t, result)

	// The parent's default block content should be kept
	require.Len(t, result.Children, 1)
	block, ok := result.Children[0].(*BlockNode)
	require.True(t, ok, "expected BlockNode, got %T", result.Children[0])
	assert.Equal(t, "header", block.Name)
	require.Len(t, block.Children, 1)

	textNode, ok := block.Children[0].(*TextNode)
	require.True(t, ok, "expected TextNode, got %T", block.Children[0])
	assert.Equal(t, "Default Header", textNode.Content)
}

func TestResolveInheritance_MultipleBlocks(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	// Parent with two blocks and text between them
	parentSource := `{~exons.block name="header"~}Default Header{~/exons.block~}Middle{~exons.block name="footer"~}Default Footer{~/exons.block~}`

	tsr := newMockTemplateSourceResolver(map[string]string{
		"parent": parentSource,
	})
	resolver := NewInheritanceResolver(nil, tsr, 10)

	// Child overrides only the footer block
	footerBlock := NewBlockNode("footer", []Node{NewTextNode("Custom Footer", pos)}, pos)
	childInfo := NewInheritanceInfo("parent", pos)
	err := childInfo.AddBlock(footerBlock)
	require.NoError(t, err)

	childRoot := &RootNode{Children: []Node{
		NewSelfClosingTag(TagNameExtends, Attributes{AttrTemplate: "parent"}, pos),
		footerBlock,
	}}

	result, err := resolver.ResolveInheritance(context.Background(), childRoot, childInfo, 0)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have 3 children: header block (default), text, footer block (overridden)
	require.Len(t, result.Children, 3)

	// First: default header block
	headerBlock, ok := result.Children[0].(*BlockNode)
	require.True(t, ok, "expected BlockNode for header")
	assert.Equal(t, "header", headerBlock.Name)
	headerText, ok := headerBlock.Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Default Header", headerText.Content)

	// Second: text node "Middle"
	middleText, ok := result.Children[1].(*TextNode)
	require.True(t, ok, "expected TextNode for middle")
	assert.Equal(t, "Middle", middleText.Content)

	// Third: overridden footer block
	footerResult, ok := result.Children[2].(*BlockNode)
	require.True(t, ok, "expected BlockNode for footer")
	assert.Equal(t, "footer", footerResult.Name)
	require.Len(t, footerResult.Children, 1)
	footerText, ok := footerResult.Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Custom Footer", footerText.Content)
}

func TestResolveInheritance_ParentContentInsertion(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	// Parent template with a block
	parentSource := `{~exons.block name="content"~}Parent Default{~/exons.block~}`

	tsr := newMockTemplateSourceResolver(map[string]string{
		"parent": parentSource,
	})
	resolver := NewInheritanceResolver(nil, tsr, 10)

	// Child block uses exons.parent to include parent content
	parentTag := NewSelfClosingTag(TagNameParent, make(Attributes), pos)
	childBlock := NewBlockNode("content", []Node{
		NewTextNode("Before Parent - ", pos),
		parentTag,
		NewTextNode(" - After Parent", pos),
	}, pos)

	childInfo := NewInheritanceInfo("parent", pos)
	err := childInfo.AddBlock(childBlock)
	require.NoError(t, err)

	childRoot := &RootNode{Children: []Node{
		NewSelfClosingTag(TagNameExtends, Attributes{AttrTemplate: "parent"}, pos),
		childBlock,
	}}

	result, err := resolver.ResolveInheritance(context.Background(), childRoot, childInfo, 0)
	require.NoError(t, err)
	require.NotNil(t, result)

	// The result should have the merged block with parent content inserted
	require.Len(t, result.Children, 1)
	block, ok := result.Children[0].(*BlockNode)
	require.True(t, ok, "expected BlockNode")
	assert.Equal(t, "content", block.Name)

	// Children should be: "Before Parent - ", parent's children, " - After Parent"
	// Parent's block has one child: TextNode("Parent Default")
	require.Len(t, block.Children, 3)

	text1, ok := block.Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Before Parent - ", text1.Content)

	parentText, ok := block.Children[1].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Parent Default", parentText.Content)

	text3, ok := block.Children[2].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, " - After Parent", text3.Content)
}

func TestResolveInheritance_MultiLevelInheritance(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	// Grandparent: base template
	grandparentSource := `{~exons.block name="title"~}Base Title{~/exons.block~}`

	// Parent: extends grandparent, overrides title
	parentSource := `{~exons.extends template="grandparent" /~}{~exons.block name="title"~}Parent Title{~/exons.block~}`

	tsr := newMockTemplateSourceResolver(map[string]string{
		"grandparent": grandparentSource,
		"parent":      parentSource,
	})
	resolver := NewInheritanceResolver(nil, tsr, 10)

	// Child extends parent, overrides title again
	childBlock := NewBlockNode("title", []Node{NewTextNode("Child Title", pos)}, pos)
	childInfo := NewInheritanceInfo("parent", pos)
	err := childInfo.AddBlock(childBlock)
	require.NoError(t, err)

	childRoot := &RootNode{Children: []Node{
		NewSelfClosingTag(TagNameExtends, Attributes{AttrTemplate: "parent"}, pos),
		childBlock,
	}}

	result, err := resolver.ResolveInheritance(context.Background(), childRoot, childInfo, 0)
	require.NoError(t, err)
	require.NotNil(t, result)

	// The final result should have the child's title
	require.Len(t, result.Children, 1)
	block, ok := result.Children[0].(*BlockNode)
	require.True(t, ok, "expected BlockNode")
	assert.Equal(t, "title", block.Name)

	require.Len(t, block.Children, 1)
	textNode, ok := block.Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Child Title", textNode.Content)
}

func TestResolveInheritance_ExtendsTagsRemovedFromOutput(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	// Parent has an extends tag (which should be stripped) and other tags
	parentSource := `Before{~exons.block name="content"~}Default{~/exons.block~}After`

	tsr := newMockTemplateSourceResolver(map[string]string{
		"parent": parentSource,
	})
	resolver := NewInheritanceResolver(nil, tsr, 10)

	childInfo := NewInheritanceInfo("parent", pos)
	childRoot := &RootNode{Children: []Node{
		NewSelfClosingTag(TagNameExtends, Attributes{AttrTemplate: "parent"}, pos),
	}}

	result, err := resolver.ResolveInheritance(context.Background(), childRoot, childInfo, 0)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify the extends tag is not in the output
	for _, child := range result.Children {
		if tag, ok := child.(*TagNode); ok {
			assert.NotEqual(t, TagNameExtends, tag.Name, "extends tag should be removed from output")
		}
	}
}

func TestResolveInheritance_InvalidParentSource(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	// Parent source with broken syntax
	tsr := newMockTemplateSourceResolver(map[string]string{
		"parent": `{~exons.if~}unclosed`,
	})
	resolver := NewInheritanceResolver(nil, tsr, 10)

	childInfo := NewInheritanceInfo("parent", pos)
	childRoot := &RootNode{Children: []Node{
		NewSelfClosingTag(TagNameExtends, Attributes{AttrTemplate: "parent"}, pos),
	}}

	_, err := resolver.ResolveInheritance(context.Background(), childRoot, childInfo, 0)
	require.Error(t, err)
}

// --- mergeBlocks Tests ---

func TestMergeBlocks_EmptyChildBlocks(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	parentBlock := NewBlockNode("header", []Node{NewTextNode("Default", pos)}, pos)
	parentRoot := &RootNode{Children: []Node{parentBlock}}

	result := resolver.mergeBlocks(parentRoot, map[string]*BlockNode{})

	require.Len(t, result.Children, 1)
	block, ok := result.Children[0].(*BlockNode)
	require.True(t, ok)
	assert.Equal(t, "header", block.Name)
}

func TestMergeBlocks_NilChildBlocks(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	parentBlock := NewBlockNode("header", []Node{NewTextNode("Default", pos)}, pos)
	parentRoot := &RootNode{Children: []Node{parentBlock}}

	result := resolver.mergeBlocks(parentRoot, nil)

	require.Len(t, result.Children, 1)
	block, ok := result.Children[0].(*BlockNode)
	require.True(t, ok)
	assert.Equal(t, "header", block.Name)
}

func TestMergeBlocks_EmptyParent(t *testing.T) {
	resolver := NewInheritanceResolver(nil, nil, 10)

	parentRoot := &RootNode{Children: []Node{}}

	result := resolver.mergeBlocks(parentRoot, map[string]*BlockNode{})

	assert.Empty(t, result.Children)
}

// --- mergeBlocksInNodes Tests (through mergeBlocks) ---

func TestMergeBlocksInNodes_ConditionalNode(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	// Parent has a conditional with a block inside a branch
	innerBlock := NewBlockNode("inner", []Node{NewTextNode("Default Inner", pos)}, pos)
	branch := NewConditionalBranch("true", []Node{innerBlock}, false, pos)
	conditional := NewConditionalNode([]ConditionalBranch{branch}, pos)

	parentRoot := &RootNode{Children: []Node{conditional}}

	// Child overrides the inner block
	childBlock := NewBlockNode("inner", []Node{NewTextNode("Custom Inner", pos)}, pos)
	childBlocks := map[string]*BlockNode{
		"inner": childBlock,
	}

	result := resolver.mergeBlocks(parentRoot, childBlocks)

	require.Len(t, result.Children, 1)
	condResult, ok := result.Children[0].(*ConditionalNode)
	require.True(t, ok)
	require.Len(t, condResult.Branches, 1)
	require.Len(t, condResult.Branches[0].Children, 1)

	mergedBlock, ok := condResult.Branches[0].Children[0].(*BlockNode)
	require.True(t, ok)
	assert.Equal(t, "inner", mergedBlock.Name)
	require.Len(t, mergedBlock.Children, 1)
	textNode, ok := mergedBlock.Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Custom Inner", textNode.Content)
}

func TestMergeBlocksInNodes_ForNode(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	// Parent has a for loop with a block inside
	innerBlock := NewBlockNode("item-block", []Node{NewTextNode("Default Item", pos)}, pos)
	forNode := NewForNode("item", "i", "items", 0, []Node{innerBlock}, pos)

	parentRoot := &RootNode{Children: []Node{forNode}}

	// Child overrides the item-block
	childBlock := NewBlockNode("item-block", []Node{NewTextNode("Custom Item", pos)}, pos)
	childBlocks := map[string]*BlockNode{
		"item-block": childBlock,
	}

	result := resolver.mergeBlocks(parentRoot, childBlocks)

	require.Len(t, result.Children, 1)
	forResult, ok := result.Children[0].(*ForNode)
	require.True(t, ok)
	require.Len(t, forResult.Children, 1)

	mergedBlock, ok := forResult.Children[0].(*BlockNode)
	require.True(t, ok)
	assert.Equal(t, "item-block", mergedBlock.Name)
	require.Len(t, mergedBlock.Children, 1)
	textNode, ok := mergedBlock.Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Custom Item", textNode.Content)
}

func TestMergeBlocksInNodes_SwitchNode(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	// Parent has a switch node with a block inside a case
	innerBlock := NewBlockNode("case-block", []Node{NewTextNode("Default Case", pos)}, pos)
	switchCase := NewSwitchCase("option1", "", []Node{innerBlock}, false, pos)
	switchNode := NewSwitchNode("expr", []SwitchCase{switchCase}, nil, pos)

	parentRoot := &RootNode{Children: []Node{switchNode}}

	// Child overrides the case-block
	childBlock := NewBlockNode("case-block", []Node{NewTextNode("Custom Case", pos)}, pos)
	childBlocks := map[string]*BlockNode{
		"case-block": childBlock,
	}

	result := resolver.mergeBlocks(parentRoot, childBlocks)

	require.Len(t, result.Children, 1)
	switchResult, ok := result.Children[0].(*SwitchNode)
	require.True(t, ok)
	require.Len(t, switchResult.Cases, 1)
	require.Len(t, switchResult.Cases[0].Children, 1)

	mergedBlock, ok := switchResult.Cases[0].Children[0].(*BlockNode)
	require.True(t, ok)
	assert.Equal(t, "case-block", mergedBlock.Name)
	require.Len(t, mergedBlock.Children, 1)
	textNode, ok := mergedBlock.Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Custom Case", textNode.Content)
}

func TestMergeBlocksInNodes_NestedTagNode(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	// Parent has a tag node (non-extends) with a block inside its children
	innerBlock := NewBlockNode("nested", []Node{NewTextNode("Default Nested", pos)}, pos)
	tagNode := NewBlockTag("custom.tag", make(Attributes), []Node{innerBlock}, pos)

	parentRoot := &RootNode{Children: []Node{tagNode}}

	// Child overrides the nested block
	childBlock := NewBlockNode("nested", []Node{NewTextNode("Custom Nested", pos)}, pos)
	childBlocks := map[string]*BlockNode{
		"nested": childBlock,
	}

	result := resolver.mergeBlocks(parentRoot, childBlocks)

	require.Len(t, result.Children, 1)
	tagResult, ok := result.Children[0].(*TagNode)
	require.True(t, ok)
	require.Len(t, tagResult.Children, 1)

	mergedBlock, ok := tagResult.Children[0].(*BlockNode)
	require.True(t, ok)
	assert.Equal(t, "nested", mergedBlock.Name)
	require.Len(t, mergedBlock.Children, 1)
	textNode, ok := mergedBlock.Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Custom Nested", textNode.Content)
}

func TestMergeBlocksInNodes_ExtendsTagSkipped(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	// Parent AST has an extends tag (shouldn't normally happen, but test the skip)
	extendsTag := NewSelfClosingTag(TagNameExtends, Attributes{AttrTemplate: "other"}, pos)
	textNode := NewTextNode("Content", pos)

	parentRoot := &RootNode{Children: []Node{extendsTag, textNode}}

	result := resolver.mergeBlocks(parentRoot, map[string]*BlockNode{})

	// Extends tag should be skipped, only text node remains
	require.Len(t, result.Children, 1)
	text, ok := result.Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Content", text.Content)
}

func TestMergeBlocksInNodes_TextNodePassthrough(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	parentRoot := &RootNode{Children: []Node{
		NewTextNode("Hello", pos),
		NewTextNode(" World", pos),
	}}

	result := resolver.mergeBlocks(parentRoot, map[string]*BlockNode{})

	require.Len(t, result.Children, 2)
	text1, ok := result.Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Hello", text1.Content)
	text2, ok := result.Children[1].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, " World", text2.Content)
}

func TestMergeBlocksInNodes_SelfClosingTagPassthrough(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	// Self-closing tags (non-extends) should pass through
	varTag := NewSelfClosingTag(TagNameVar, Attributes{AttrName: "user"}, pos)
	parentRoot := &RootNode{Children: []Node{varTag}}

	result := resolver.mergeBlocks(parentRoot, map[string]*BlockNode{})

	require.Len(t, result.Children, 1)
	tag, ok := result.Children[0].(*TagNode)
	require.True(t, ok)
	assert.Equal(t, TagNameVar, tag.Name)
}

// --- resolveParentCalls Tests ---

func TestResolveParentCalls_NoParentTag(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	childBlock := NewBlockNode("content", []Node{
		NewTextNode("Just Child Content", pos),
	}, pos)
	parentBlock := NewBlockNode("content", []Node{
		NewTextNode("Parent Content", pos),
	}, pos)

	result := resolver.resolveParentCalls(childBlock, parentBlock)

	require.Len(t, result.Children, 1)
	textNode, ok := result.Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Just Child Content", textNode.Content)
}

func TestResolveParentCalls_SingleParentTag(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	parentTag := NewSelfClosingTag(TagNameParent, make(Attributes), pos)
	childBlock := NewBlockNode("content", []Node{
		NewTextNode("Before ", pos),
		parentTag,
		NewTextNode(" After", pos),
	}, pos)
	parentBlock := NewBlockNode("content", []Node{
		NewTextNode("PARENT", pos),
	}, pos)

	result := resolver.resolveParentCalls(childBlock, parentBlock)

	require.Len(t, result.Children, 3)
	assert.Equal(t, "Before ", result.Children[0].(*TextNode).Content)
	assert.Equal(t, "PARENT", result.Children[1].(*TextNode).Content)
	assert.Equal(t, " After", result.Children[2].(*TextNode).Content)
}

func TestResolveParentCalls_MultipleParentTags(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	parentTag1 := NewSelfClosingTag(TagNameParent, make(Attributes), pos)
	parentTag2 := NewSelfClosingTag(TagNameParent, make(Attributes), pos)
	childBlock := NewBlockNode("content", []Node{
		parentTag1,
		NewTextNode(" middle ", pos),
		parentTag2,
	}, pos)
	parentBlock := NewBlockNode("content", []Node{
		NewTextNode("P", pos),
	}, pos)

	result := resolver.resolveParentCalls(childBlock, parentBlock)

	// Should have: P, " middle ", P
	require.Len(t, result.Children, 3)
	assert.Equal(t, "P", result.Children[0].(*TextNode).Content)
	assert.Equal(t, " middle ", result.Children[1].(*TextNode).Content)
	assert.Equal(t, "P", result.Children[2].(*TextNode).Content)
}

func TestResolveParentCalls_ParentWithMultipleChildren(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	parentTag := NewSelfClosingTag(TagNameParent, make(Attributes), pos)
	childBlock := NewBlockNode("content", []Node{parentTag}, pos)
	parentBlock := NewBlockNode("content", []Node{
		NewTextNode("Part1", pos),
		NewTextNode("Part2", pos),
		NewTextNode("Part3", pos),
	}, pos)

	result := resolver.resolveParentCalls(childBlock, parentBlock)

	// All parent children should be inserted
	require.Len(t, result.Children, 3)
	assert.Equal(t, "Part1", result.Children[0].(*TextNode).Content)
	assert.Equal(t, "Part2", result.Children[1].(*TextNode).Content)
	assert.Equal(t, "Part3", result.Children[2].(*TextNode).Content)
}

func TestResolveParentCalls_EmptyChildBlock(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	childBlock := NewBlockNode("content", []Node{}, pos)
	parentBlock := NewBlockNode("content", []Node{
		NewTextNode("Parent Content", pos),
	}, pos)

	result := resolver.resolveParentCalls(childBlock, parentBlock)

	assert.Empty(t, result.Children)
	assert.Equal(t, "content", result.Name)
}

func TestResolveParentCalls_EmptyParentBlock(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	parentTag := NewSelfClosingTag(TagNameParent, make(Attributes), pos)
	childBlock := NewBlockNode("content", []Node{
		NewTextNode("Before", pos),
		parentTag,
		NewTextNode("After", pos),
	}, pos)
	parentBlock := NewBlockNode("content", []Node{}, pos)

	result := resolver.resolveParentCalls(childBlock, parentBlock)

	// Parent has no children, so exons.parent replaces with nothing
	require.Len(t, result.Children, 2)
	assert.Equal(t, "Before", result.Children[0].(*TextNode).Content)
	assert.Equal(t, "After", result.Children[1].(*TextNode).Content)
}

func TestResolveParentCalls_NestedInTagNode(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	// Parent tag nested inside another tag node
	parentTag := NewSelfClosingTag(TagNameParent, make(Attributes), pos)
	innerTag := NewBlockTag("custom.wrapper", make(Attributes), []Node{parentTag}, pos)

	childBlock := NewBlockNode("content", []Node{innerTag}, pos)
	parentBlock := NewBlockNode("content", []Node{
		NewTextNode("Parent Here", pos),
	}, pos)

	result := resolver.resolveParentCalls(childBlock, parentBlock)

	require.Len(t, result.Children, 1)
	wrapper, ok := result.Children[0].(*TagNode)
	require.True(t, ok)
	require.Len(t, wrapper.Children, 1)
	assert.Equal(t, "Parent Here", wrapper.Children[0].(*TextNode).Content)
}

func TestResolveParentCalls_NestedInBlockNode(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	// Parent tag inside a nested block
	parentTag := NewSelfClosingTag(TagNameParent, make(Attributes), pos)
	nestedBlock := NewBlockNode("inner", []Node{parentTag}, pos)

	childBlock := NewBlockNode("content", []Node{nestedBlock}, pos)
	parentBlock := NewBlockNode("content", []Node{
		NewTextNode("Injected", pos),
	}, pos)

	result := resolver.resolveParentCalls(childBlock, parentBlock)

	require.Len(t, result.Children, 1)
	inner, ok := result.Children[0].(*BlockNode)
	require.True(t, ok)
	require.Len(t, inner.Children, 1)
	assert.Equal(t, "Injected", inner.Children[0].(*TextNode).Content)
}

func TestResolveParentCalls_NestedInConditionalNode(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	// Parent tag inside a conditional branch
	parentTag := NewSelfClosingTag(TagNameParent, make(Attributes), pos)
	branch := NewConditionalBranch("true", []Node{parentTag}, false, pos)
	conditional := NewConditionalNode([]ConditionalBranch{branch}, pos)

	childBlock := NewBlockNode("content", []Node{conditional}, pos)
	parentBlock := NewBlockNode("content", []Node{
		NewTextNode("Conditional Parent", pos),
	}, pos)

	result := resolver.resolveParentCalls(childBlock, parentBlock)

	require.Len(t, result.Children, 1)
	cond, ok := result.Children[0].(*ConditionalNode)
	require.True(t, ok)
	require.Len(t, cond.Branches, 1)
	require.Len(t, cond.Branches[0].Children, 1)
	assert.Equal(t, "Conditional Parent", cond.Branches[0].Children[0].(*TextNode).Content)
}

func TestResolveParentCalls_NestedInForNode(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	// Parent tag inside a for loop
	parentTag := NewSelfClosingTag(TagNameParent, make(Attributes), pos)
	forNode := NewForNode("item", "i", "items", 0, []Node{parentTag}, pos)

	childBlock := NewBlockNode("content", []Node{forNode}, pos)
	parentBlock := NewBlockNode("content", []Node{
		NewTextNode("Loop Parent", pos),
	}, pos)

	result := resolver.resolveParentCalls(childBlock, parentBlock)

	require.Len(t, result.Children, 1)
	loop, ok := result.Children[0].(*ForNode)
	require.True(t, ok)
	require.Len(t, loop.Children, 1)
	assert.Equal(t, "Loop Parent", loop.Children[0].(*TextNode).Content)
}

func TestResolveParentCalls_NestedInSwitchNode(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	resolver := NewInheritanceResolver(nil, nil, 10)

	// Parent tag inside a switch case
	parentTag := NewSelfClosingTag(TagNameParent, make(Attributes), pos)
	switchCase := NewSwitchCase("val", "", []Node{parentTag}, false, pos)
	switchNode := NewSwitchNode("expr", []SwitchCase{switchCase}, nil, pos)

	childBlock := NewBlockNode("content", []Node{switchNode}, pos)
	parentBlock := NewBlockNode("content", []Node{
		NewTextNode("Switch Parent", pos),
	}, pos)

	result := resolver.resolveParentCalls(childBlock, parentBlock)

	require.Len(t, result.Children, 1)
	sw, ok := result.Children[0].(*SwitchNode)
	require.True(t, ok)
	require.Len(t, sw.Cases, 1)
	require.Len(t, sw.Cases[0].Children, 1)
	assert.Equal(t, "Switch Parent", sw.Cases[0].Children[0].(*TextNode).Content)
}

func TestResolveParentCalls_PreservesBlockMetadata(t *testing.T) {
	pos := Position{Line: 5, Column: 10, Offset: 42}
	resolver := NewInheritanceResolver(nil, nil, 10)

	childBlock := &BlockNode{
		pos:       pos,
		Name:      "test-block",
		Children:  []Node{NewTextNode("content", pos)},
		RawSource: "original raw source",
	}
	parentBlock := NewBlockNode("test-block", nil, pos)

	result := resolver.resolveParentCalls(childBlock, parentBlock)

	assert.Equal(t, pos, result.Pos())
	assert.Equal(t, "test-block", result.Name)
	assert.Equal(t, "original raw source", result.RawSource)
}

// --- parseTemplateWithInheritance Tests ---

func TestParseTemplateWithInheritance_SimpleTemplate(t *testing.T) {
	resolver := NewInheritanceResolver(nil, nil, 10)

	root, info, err := resolver.parseTemplateWithInheritance("Hello World")
	require.NoError(t, err)
	require.NotNil(t, root)
	assert.Nil(t, info, "no inheritance info expected for simple template")
	require.Len(t, root.Children, 1)
}

func TestParseTemplateWithInheritance_WithExtends(t *testing.T) {
	resolver := NewInheritanceResolver(nil, nil, 10)

	source := `{~exons.extends template="base" /~}{~exons.block name="content"~}Hello{~/exons.block~}`
	root, info, err := resolver.parseTemplateWithInheritance(source)
	require.NoError(t, err)
	require.NotNil(t, root)
	require.NotNil(t, info)
	assert.Equal(t, "base", info.ParentTemplate)
	assert.True(t, info.HasBlock("content"))
}

func TestParseTemplateWithInheritance_InvalidSyntax(t *testing.T) {
	resolver := NewInheritanceResolver(nil, nil, 10)

	// Malformed template
	_, _, err := resolver.parseTemplateWithInheritance(`{~exons.if~}unclosed`)
	require.Error(t, err)
}

func TestParseTemplateWithInheritance_EmptyTemplate(t *testing.T) {
	resolver := NewInheritanceResolver(nil, nil, 10)

	root, info, err := resolver.parseTemplateWithInheritance("")
	require.NoError(t, err)
	require.NotNil(t, root)
	assert.Nil(t, info)
}

// --- ExtractInheritanceInfo Additional Tests ---

func TestExtractInheritanceInfo_NilRoot(t *testing.T) {
	info, err := ExtractInheritanceInfo(nil)
	require.NoError(t, err)
	assert.Nil(t, info)
}

func TestExtractInheritanceInfo_EmptyRoot(t *testing.T) {
	root := &RootNode{Children: []Node{}}
	info, err := ExtractInheritanceInfo(root)
	require.NoError(t, err)
	assert.Nil(t, info)
}

func TestExtractInheritanceInfo_ParentOutsideBlock(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	// Parent tag at top-level without extends context
	parentTag := NewSelfClosingTag(TagNameParent, make(Attributes), pos)
	root := &RootNode{Children: []Node{parentTag}}

	_, err := ExtractInheritanceInfo(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgParentOutsideBlock)
}

func TestExtractInheritanceInfo_ExtendsAfterNonWhitespace(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	// Non-whitespace text before extends
	textNode := NewTextNode("Hello", pos)
	extendsTag := NewSelfClosingTag(TagNameExtends, Attributes{AttrTemplate: "base"}, pos)
	root := &RootNode{Children: []Node{textNode, extendsTag}}

	_, err := ExtractInheritanceInfo(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgExtendsNotFirst)
}

func TestExtractInheritanceInfo_ExtendsAfterWhitespace(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	// Whitespace-only text before extends (allowed)
	whitespace := NewTextNode("  \n  ", pos)
	extendsTag := NewSelfClosingTag(TagNameExtends, Attributes{AttrTemplate: "base"}, pos)
	blockNode := NewBlockNode("content", []Node{NewTextNode("Hello", pos)}, pos)
	root := &RootNode{Children: []Node{whitespace, extendsTag, blockNode}}

	info, err := ExtractInheritanceInfo(root)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "base", info.ParentTemplate)
}

func TestExtractInheritanceInfo_ExtendsMissingTemplate(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	extendsTag := NewSelfClosingTag(TagNameExtends, make(Attributes), pos)
	root := &RootNode{Children: []Node{extendsTag}}

	_, err := ExtractInheritanceInfo(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgExtendsMissingTemplate)
}

func TestExtractInheritanceInfo_MultipleExtends(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	extends1 := NewSelfClosingTag(TagNameExtends, Attributes{AttrTemplate: "base1"}, pos)
	extends2 := NewSelfClosingTag(TagNameExtends, Attributes{AttrTemplate: "base2"}, pos)
	root := &RootNode{Children: []Node{extends1, extends2}}

	_, err := ExtractInheritanceInfo(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgExtendsMultiple)
}

func TestExtractInheritanceInfo_DuplicateBlockNames(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	extendsTag := NewSelfClosingTag(TagNameExtends, Attributes{AttrTemplate: "base"}, pos)
	block1 := NewBlockNode("content", []Node{NewTextNode("One", pos)}, pos)
	block2 := NewBlockNode("content", []Node{NewTextNode("Two", pos)}, pos)
	root := &RootNode{Children: []Node{extendsTag, block1, block2}}

	_, err := ExtractInheritanceInfo(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgBlockDuplicateName)
}

func TestExtractInheritanceInfo_BlocksCollected(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	extendsTag := NewSelfClosingTag(TagNameExtends, Attributes{AttrTemplate: "base"}, pos)
	headerBlock := NewBlockNode("header", []Node{NewTextNode("H", pos)}, pos)
	contentBlock := NewBlockNode("content", []Node{NewTextNode("C", pos)}, pos)
	footerBlock := NewBlockNode("footer", []Node{NewTextNode("F", pos)}, pos)
	root := &RootNode{Children: []Node{extendsTag, headerBlock, contentBlock, footerBlock}}

	info, err := ExtractInheritanceInfo(root)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, 3, len(info.Blocks))
	assert.True(t, info.HasBlock("header"))
	assert.True(t, info.HasBlock("content"))
	assert.True(t, info.HasBlock("footer"))
}

func TestExtractInheritanceInfo_BlocksWithoutExtends(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	// Blocks without extends should not produce inheritance info
	block := NewBlockNode("standalone", []Node{NewTextNode("content", pos)}, pos)
	root := &RootNode{Children: []Node{block}}

	info, err := ExtractInheritanceInfo(root)
	require.NoError(t, err)
	assert.Nil(t, info, "blocks without extends should not produce inheritance info")
}

func TestExtractInheritanceInfo_NonTagNonBlockNodesIgnored(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	// Other node types (conditionals, for loops) should be ignored
	extendsTag := NewSelfClosingTag(TagNameExtends, Attributes{AttrTemplate: "base"}, pos)
	conditional := NewConditionalNode([]ConditionalBranch{
		NewConditionalBranch("true", []Node{NewTextNode("yes", pos)}, false, pos),
	}, pos)
	root := &RootNode{Children: []Node{extendsTag, conditional}}

	info, err := ExtractInheritanceInfo(root)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "base", info.ParentTemplate)
	assert.Empty(t, info.Blocks)
}

// --- CollectBlocks Additional Tests ---

func TestCollectBlocks_EmptyRoot(t *testing.T) {
	root := &RootNode{Children: []Node{}}
	blocks := CollectBlocks(root)
	assert.Empty(t, blocks)
}

func TestCollectBlocks_NestedInTagNode(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	innerBlock := NewBlockNode("inner", []Node{NewTextNode("nested content", pos)}, pos)
	wrapper := NewBlockTag("custom.wrapper", make(Attributes), []Node{innerBlock}, pos)
	root := &RootNode{Children: []Node{wrapper}}

	blocks := CollectBlocks(root)
	assert.Len(t, blocks, 1)
	assert.Contains(t, blocks, "inner")
}

func TestCollectBlocks_NestedInConditional(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	innerBlock := NewBlockNode("cond-block", []Node{NewTextNode("in conditional", pos)}, pos)
	branch := NewConditionalBranch("true", []Node{innerBlock}, false, pos)
	conditional := NewConditionalNode([]ConditionalBranch{branch}, pos)
	root := &RootNode{Children: []Node{conditional}}

	blocks := CollectBlocks(root)
	assert.Len(t, blocks, 1)
	assert.Contains(t, blocks, "cond-block")
}

func TestCollectBlocks_NestedInForLoop(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	innerBlock := NewBlockNode("loop-block", []Node{NewTextNode("in loop", pos)}, pos)
	forNode := NewForNode("item", "", "items", 0, []Node{innerBlock}, pos)
	root := &RootNode{Children: []Node{forNode}}

	blocks := CollectBlocks(root)
	assert.Len(t, blocks, 1)
	assert.Contains(t, blocks, "loop-block")
}

func TestCollectBlocks_NestedInSwitchCase(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	innerBlock := NewBlockNode("switch-block", []Node{NewTextNode("in switch", pos)}, pos)
	switchCase := NewSwitchCase("val", "", []Node{innerBlock}, false, pos)
	switchNode := NewSwitchNode("expr", []SwitchCase{switchCase}, nil, pos)
	root := &RootNode{Children: []Node{switchNode}}

	blocks := CollectBlocks(root)
	assert.Len(t, blocks, 1)
	assert.Contains(t, blocks, "switch-block")
}

func TestCollectBlocks_DeeplyNested(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	// Block inside a block inside a conditional
	innerBlock := NewBlockNode("deep", []Node{NewTextNode("deep content", pos)}, pos)
	outerBlock := NewBlockNode("outer", []Node{innerBlock}, pos)
	branch := NewConditionalBranch("true", []Node{outerBlock}, false, pos)
	conditional := NewConditionalNode([]ConditionalBranch{branch}, pos)
	root := &RootNode{Children: []Node{conditional}}

	blocks := CollectBlocks(root)
	assert.Len(t, blocks, 2)
	assert.Contains(t, blocks, "outer")
	assert.Contains(t, blocks, "deep")
}

func TestCollectBlocks_DuplicateNameLastWins(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	// Two blocks with same name at different levels
	block1 := NewBlockNode("dup", []Node{NewTextNode("first", pos)}, pos)
	block2 := NewBlockNode("dup", []Node{NewTextNode("second", pos)}, pos)
	root := &RootNode{Children: []Node{block1, block2}}

	blocks := CollectBlocks(root)
	assert.Len(t, blocks, 1)
	// Last one wins (map overwrite)
	textNode, ok := blocks["dup"].Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "second", textNode.Content)
}

// --- InheritanceInfo Methods Tests ---

func TestInheritanceInfo_HasBlock(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	info := NewInheritanceInfo("parent", pos)

	assert.False(t, info.HasBlock("content"))

	block := NewBlockNode("content", nil, pos)
	err := info.AddBlock(block)
	require.NoError(t, err)

	assert.True(t, info.HasBlock("content"))
	assert.False(t, info.HasBlock("other"))
}

func TestInheritanceInfo_GetBlock(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	info := NewInheritanceInfo("parent", pos)

	_, ok := info.GetBlock("content")
	assert.False(t, ok)

	block := NewBlockNode("content", []Node{NewTextNode("test", pos)}, pos)
	err := info.AddBlock(block)
	require.NoError(t, err)

	retrieved, ok := info.GetBlock("content")
	assert.True(t, ok)
	assert.Equal(t, block, retrieved)
}

func TestInheritanceInfo_AddBlockDuplicate(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	info := NewInheritanceInfo("parent", pos)

	block1 := NewBlockNode("content", nil, pos)
	err := info.AddBlock(block1)
	require.NoError(t, err)

	block2 := NewBlockNode("content", nil, pos)
	err = info.AddBlock(block2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgBlockDuplicateName)
}

func TestInheritanceInfo_AddMultipleBlocks(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	info := NewInheritanceInfo("parent", pos)

	blocks := []string{"header", "content", "footer", "sidebar"}
	for _, name := range blocks {
		err := info.AddBlock(NewBlockNode(name, nil, pos))
		require.NoError(t, err)
	}

	for _, name := range blocks {
		assert.True(t, info.HasBlock(name))
	}
	assert.Equal(t, len(blocks), len(info.Blocks))
}

// --- BlockNode Tests ---

func TestBlockNode_Type(t *testing.T) {
	block := NewBlockNode("test", nil, Position{})
	assert.Equal(t, NodeTypeBlock, block.Type())
}

func TestBlockNode_Pos(t *testing.T) {
	pos := Position{Line: 3, Column: 7, Offset: 42}
	block := NewBlockNode("test", nil, pos)
	assert.Equal(t, pos, block.Pos())
}

func TestBlockNode_String(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	children := []Node{NewTextNode("a", pos), NewTextNode("b", pos)}
	block := NewBlockNode("myblock", children, pos)

	str := block.String()
	assert.Contains(t, str, "myblock")
	assert.Contains(t, str, "children=2")
}

// --- isFirstSignificantNode Tests ---

func TestIsFirstSignificantNode_NilSlice(t *testing.T) {
	assert.True(t, isFirstSignificantNode(nil))
}

func TestIsFirstSignificantNode_EmptySlice(t *testing.T) {
	assert.True(t, isFirstSignificantNode([]Node{}))
}

func TestIsFirstSignificantNode_WhitespaceOnly(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	nodes := []Node{
		NewTextNode("  ", pos),
		NewTextNode("\n\t", pos),
	}
	assert.True(t, isFirstSignificantNode(nodes))
}

func TestIsFirstSignificantNode_NonWhitespaceText(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	nodes := []Node{NewTextNode("Hello", pos)}
	assert.False(t, isFirstSignificantNode(nodes))
}

func TestIsFirstSignificantNode_MixedWhitespaceAndText(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	nodes := []Node{
		NewTextNode("  ", pos),
		NewTextNode("Hello", pos),
	}
	assert.False(t, isFirstSignificantNode(nodes))
}

func TestIsFirstSignificantNode_NonTextNode(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	nodes := []Node{
		NewSelfClosingTag(TagNameVar, make(Attributes), pos),
	}
	assert.False(t, isFirstSignificantNode(nodes))
}

func TestIsFirstSignificantNode_WhitespaceBeforeTag(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	nodes := []Node{
		NewTextNode("  \n  ", pos),
		NewSelfClosingTag(TagNameVar, make(Attributes), pos),
	}
	assert.False(t, isFirstSignificantNode(nodes))
}

// --- Integration-style tests combining parsing and inheritance resolution ---

func TestResolveInheritance_FullPipelineSimple(t *testing.T) {
	// Parent: header + content block + footer
	parentSource := `Header{~exons.block name="content"~}Default Content{~/exons.block~}Footer`

	// Child: extends parent, overrides content
	childSource := `{~exons.extends template="parent" /~}{~exons.block name="content"~}Custom Content{~/exons.block~}`

	tsr := newMockTemplateSourceResolver(map[string]string{
		"parent": parentSource,
	})
	resolver := NewInheritanceResolver(nil, tsr, 10)

	// Parse child
	lexer := NewLexer(childSource, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParserWithSource(tokens, childSource, nil)
	childRoot, err := parser.Parse()
	require.NoError(t, err)

	childInfo, err := ExtractInheritanceInfo(childRoot)
	require.NoError(t, err)
	require.NotNil(t, childInfo)
	assert.Equal(t, "parent", childInfo.ParentTemplate)
	assert.True(t, childInfo.HasBlock("content"))

	// Resolve inheritance
	result, err := resolver.ResolveInheritance(context.Background(), childRoot, childInfo, 0)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify structure: Header text, overridden block, Footer text
	require.Len(t, result.Children, 3)

	headerText, ok := result.Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Header", headerText.Content)

	block, ok := result.Children[1].(*BlockNode)
	require.True(t, ok)
	assert.Equal(t, "content", block.Name)
	require.Len(t, block.Children, 1)
	contentText, ok := block.Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Custom Content", contentText.Content)

	footerText, ok := result.Children[2].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Footer", footerText.Content)
}

func TestResolveInheritance_FullPipelineWithParentCall(t *testing.T) {
	// Parent with default content in block
	parentSource := `{~exons.block name="nav"~}Home | About{~/exons.block~}`

	// Child uses parent() to include default and add extra
	childSource := `{~exons.extends template="parent" /~}{~exons.block name="nav"~}{~exons.parent /~} | Contact{~/exons.block~}`

	tsr := newMockTemplateSourceResolver(map[string]string{
		"parent": parentSource,
	})
	resolver := NewInheritanceResolver(nil, tsr, 10)

	// Parse child
	lexer := NewLexer(childSource, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParserWithSource(tokens, childSource, nil)
	childRoot, err := parser.Parse()
	require.NoError(t, err)

	childInfo, err := ExtractInheritanceInfo(childRoot)
	require.NoError(t, err)
	require.NotNil(t, childInfo)

	// Resolve
	result, err := resolver.ResolveInheritance(context.Background(), childRoot, childInfo, 0)
	require.NoError(t, err)
	require.NotNil(t, result)

	// The block should contain: parent's content ("Home | About"), then " | Contact"
	require.Len(t, result.Children, 1)
	block, ok := result.Children[0].(*BlockNode)
	require.True(t, ok)
	assert.Equal(t, "nav", block.Name)

	// Block children: TextNode("Home | About"), TextNode(" | Contact")
	require.Len(t, block.Children, 2)
	parentContent, ok := block.Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Home | About", parentContent.Content)

	childContent, ok := block.Children[1].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, " | Contact", childContent.Content)
}

func TestResolveInheritance_ThreeLevelInheritance(t *testing.T) {
	// Grandparent
	grandparentSource := `{~exons.block name="title"~}Site{~/exons.block~} | {~exons.block name="subtitle"~}Welcome{~/exons.block~}`

	// Parent extends grandparent, overrides subtitle
	parentSource := `{~exons.extends template="grandparent" /~}{~exons.block name="subtitle"~}Parent Section{~/exons.block~}`

	// Child extends parent, overrides title
	childSource := `{~exons.extends template="parent" /~}{~exons.block name="title"~}My Page{~/exons.block~}`

	tsr := newMockTemplateSourceResolver(map[string]string{
		"grandparent": grandparentSource,
		"parent":      parentSource,
	})
	resolver := NewInheritanceResolver(nil, tsr, 10)

	// Parse child
	lexer := NewLexer(childSource, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParserWithSource(tokens, childSource, nil)
	childRoot, err := parser.Parse()
	require.NoError(t, err)

	childInfo, err := ExtractInheritanceInfo(childRoot)
	require.NoError(t, err)
	require.NotNil(t, childInfo)

	// Resolve
	result, err := resolver.ResolveInheritance(context.Background(), childRoot, childInfo, 0)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Expected: "My Page" (child override) | " | " | "Parent Section" (parent override)
	require.Len(t, result.Children, 3)

	titleBlock, ok := result.Children[0].(*BlockNode)
	require.True(t, ok)
	assert.Equal(t, "title", titleBlock.Name)
	titleText, ok := titleBlock.Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "My Page", titleText.Content)

	separator, ok := result.Children[1].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, " | ", separator.Content)

	subtitleBlock, ok := result.Children[2].(*BlockNode)
	require.True(t, ok)
	assert.Equal(t, "subtitle", subtitleBlock.Name)
	subtitleText, ok := subtitleBlock.Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Parent Section", subtitleText.Content)
}

// --- ParsedTemplateWithInheritance Tests ---

func TestParsedTemplateWithInheritance_Struct(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	root := &RootNode{Children: []Node{NewTextNode("test", pos)}}
	info := NewInheritanceInfo("parent", pos)

	parsed := &ParsedTemplateWithInheritance{
		Root:        root,
		Inheritance: info,
	}

	assert.NotNil(t, parsed.Root)
	assert.NotNil(t, parsed.Inheritance)
	assert.Equal(t, "parent", parsed.Inheritance.ParentTemplate)
}

func TestParsedTemplateWithInheritance_NilInheritance(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	root := &RootNode{Children: []Node{NewTextNode("test", pos)}}

	parsed := &ParsedTemplateWithInheritance{
		Root:        root,
		Inheritance: nil,
	}

	assert.NotNil(t, parsed.Root)
	assert.Nil(t, parsed.Inheritance)
}

// --- NewBlockNode Tests ---

func TestNewBlockNode(t *testing.T) {
	pos := Position{Line: 2, Column: 5, Offset: 15}
	children := []Node{NewTextNode("hello", pos)}

	block := NewBlockNode("myblock", children, pos)

	assert.Equal(t, "myblock", block.Name)
	assert.Equal(t, pos, block.Pos())
	assert.Len(t, block.Children, 1)
	assert.Empty(t, block.RawSource)
}

func TestNewBlockNode_NilChildren(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	block := NewBlockNode("empty", nil, pos)

	assert.Equal(t, "empty", block.Name)
	assert.Nil(t, block.Children)
}

// --- NewInheritanceInfo Tests ---

func TestNewInheritanceInfo(t *testing.T) {
	pos := Position{Line: 3, Column: 1, Offset: 20}

	info := NewInheritanceInfo("base-layout", pos)

	assert.Equal(t, "base-layout", info.ParentTemplate)
	assert.Equal(t, pos, info.ExtendsPos)
	assert.NotNil(t, info.Blocks)
	assert.Empty(t, info.Blocks)
}
