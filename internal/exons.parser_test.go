package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_ParsePlainText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple text",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "text with newlines",
			input:    "Line 1\nLine 2\nLine 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   \t\n   ",
			expected: "   \t\n   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, nil)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)

			parser := NewParser(tokens, nil)
			ast, err := parser.Parse()
			require.NoError(t, err)
			require.NotNil(t, ast)

			if tt.expected == "" {
				assert.Empty(t, ast.Children)
			} else {
				require.Len(t, ast.Children, 1)
				textNode, ok := ast.Children[0].(*TextNode)
				require.True(t, ok, "expected TextNode")
				assert.Equal(t, tt.expected, textNode.Content)
			}
		})
	}
}

func TestParser_ParseSelfClosingTag(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedTag   string
		expectedAttrs map[string]string
	}{
		{
			name:        "simple self-closing tag",
			input:       `{~exons.var name="user" /~}`,
			expectedTag: TagNameVar,
			expectedAttrs: map[string]string{
				"name": "user",
			},
		},
		{
			name:        "tag with multiple attributes",
			input:       `{~exons.var name="user" default="Guest" /~}`,
			expectedTag: TagNameVar,
			expectedAttrs: map[string]string{
				"name":    "user",
				"default": "Guest",
			},
		},
		{
			name:          "tag without attributes",
			input:         `{~MyTag /~}`,
			expectedTag:   "MyTag",
			expectedAttrs: map[string]string{},
		},
		{
			name:        "tag with hyphenated name",
			input:       `{~my-custom-tag attr="value" /~}`,
			expectedTag: "my-custom-tag",
			expectedAttrs: map[string]string{
				"attr": "value",
			},
		},
		{
			name:        "tag with dotted name",
			input:       `{~my.namespaced.tag attr="value" /~}`,
			expectedTag: "my.namespaced.tag",
			expectedAttrs: map[string]string{
				"attr": "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, nil)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)

			parser := NewParser(tokens, nil)
			ast, err := parser.Parse()
			require.NoError(t, err)
			require.NotNil(t, ast)

			require.Len(t, ast.Children, 1)
			tagNode, ok := ast.Children[0].(*TagNode)
			require.True(t, ok, "expected TagNode")

			assert.Equal(t, tt.expectedTag, tagNode.Name)
			assert.True(t, tagNode.SelfClose)
			assert.Nil(t, tagNode.Children)

			for key, expectedVal := range tt.expectedAttrs {
				actualVal, exists := tagNode.Attributes.Get(key)
				assert.True(t, exists, "attribute %s should exist", key)
				assert.Equal(t, expectedVal, actualVal)
			}
		})
	}
}

func TestParser_ParseBlockTag(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedTag      string
		expectedChildren int
	}{
		{
			name:             "simple block tag",
			input:            `{~section~}Content{~/section~}`,
			expectedTag:      "section",
			expectedChildren: 1, // TextNode
		},
		{
			name:             "empty block tag",
			input:            `{~section~}{~/section~}`,
			expectedTag:      "section",
			expectedChildren: 0,
		},
		{
			name:             "block tag with multiline content",
			input:            "{~section~}\nLine 1\nLine 2\n{~/section~}",
			expectedTag:      "section",
			expectedChildren: 1, // TextNode with newlines
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, nil)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)

			parser := NewParser(tokens, nil)
			ast, err := parser.Parse()
			require.NoError(t, err)
			require.NotNil(t, ast)

			require.Len(t, ast.Children, 1)
			tagNode, ok := ast.Children[0].(*TagNode)
			require.True(t, ok, "expected TagNode")

			assert.Equal(t, tt.expectedTag, tagNode.Name)
			assert.False(t, tagNode.SelfClose)
			assert.Len(t, tagNode.Children, tt.expectedChildren)
		})
	}
}

func TestParser_ParseNestedTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "nested block tags",
			input: `{~outer~}{~inner~}Content{~/inner~}{~/outer~}`,
		},
		{
			name:  "deeply nested tags",
			input: `{~a~}{~b~}{~c~}Deep{~/c~}{~/b~}{~/a~}`,
		},
		{
			name:  "nested with text between",
			input: `{~outer~}Before{~inner~}Inside{~/inner~}After{~/outer~}`,
		},
		{
			name:  "self-closing inside block",
			input: `{~outer~}{~exons.var name="x" /~}{~/outer~}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, nil)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)

			parser := NewParser(tokens, nil)
			ast, err := parser.Parse()
			require.NoError(t, err)
			require.NotNil(t, ast)

			// Just verify it parses without error
			assert.NotEmpty(t, ast.Children)
		})
	}
}

func TestParser_ParseRawBlock(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedContent string
	}{
		{
			name:            "raw block with plain text",
			input:           `{~exons.raw~}This is raw content{~/exons.raw~}`,
			expectedContent: "This is raw content",
		},
		{
			name:            "raw block preserves inner tags",
			input:           `{~exons.raw~}{~exons.var name="x" /~}{~/exons.raw~}`,
			expectedContent: `{~exons.var name="x" /~}`,
		},
		{
			name:            "raw block with newlines",
			input:           "{~exons.raw~}\nLine 1\nLine 2\n{~/exons.raw~}",
			expectedContent: "\nLine 1\nLine 2\n",
		},
		{
			name:            "empty raw block",
			input:           `{~exons.raw~}{~/exons.raw~}`,
			expectedContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, nil)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)

			parser := NewParser(tokens, nil)
			ast, err := parser.Parse()
			require.NoError(t, err)
			require.NotNil(t, ast)

			require.Len(t, ast.Children, 1)
			tagNode, ok := ast.Children[0].(*TagNode)
			require.True(t, ok, "expected TagNode")

			assert.Equal(t, TagNameRaw, tagNode.Name)
			assert.True(t, tagNode.IsRaw())
			assert.Equal(t, tt.expectedContent, tagNode.RawContent)
		})
	}
}

func TestParser_ParseMixedContent(t *testing.T) {
	input := `Hello {~exons.var name="user" /~}, welcome to {~section~}the app{~/section~}!`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Should have: Text, Tag, Text, Tag, Text
	require.Len(t, ast.Children, 5)

	// First: "Hello "
	text0, ok := ast.Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Hello ", text0.Content)

	// Second: exons.var tag
	tag1, ok := ast.Children[1].(*TagNode)
	require.True(t, ok)
	assert.Equal(t, TagNameVar, tag1.Name)
	assert.True(t, tag1.SelfClose)

	// Third: ", welcome to "
	text2, ok := ast.Children[2].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, ", welcome to ", text2.Content)

	// Fourth: section block tag
	tag3, ok := ast.Children[3].(*TagNode)
	require.True(t, ok)
	assert.Equal(t, "section", tag3.Name)
	assert.False(t, tag3.SelfClose)
	require.Len(t, tag3.Children, 1)

	// Fifth: "!"
	text4, ok := ast.Children[4].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "!", text4.Content)
}

func TestParser_ParseEscapedDelimiter(t *testing.T) {
	input := `Use \{~ for literal delimiters`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Should produce: "Use " + "{~" + " for literal delimiters"
	// But the lexer handles this, so parser just sees text nodes
	require.NotEmpty(t, ast.Children)
}

func TestParser_Errors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		errContains string
	}{
		{
			name:        "mismatched closing tag",
			input:       `{~outer~}Content{~/inner~}`,
			errContains: ErrMsgMismatchedTag,
		},
		{
			name:        "unclosed block tag",
			input:       `{~outer~}Content`,
			errContains: ErrMsgMismatchedTag,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, nil)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)

			parser := NewParser(tokens, nil)
			_, err = parser.Parse()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestParser_PositionTracking(t *testing.T) {
	input := "Text\n{~tag /~}"

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)
	require.NotNil(t, ast)

	require.Len(t, ast.Children, 2)

	// Text node at line 1
	textNode := ast.Children[0].(*TextNode)
	assert.Equal(t, 1, textNode.Pos().Line)
	assert.Equal(t, 1, textNode.Pos().Column)

	// Tag node at line 2
	tagNode := ast.Children[1].(*TagNode)
	assert.Equal(t, 2, tagNode.Pos().Line)
	assert.Equal(t, 1, tagNode.Pos().Column)
}

func TestParser_AttributeTypes(t *testing.T) {
	input := `{~tag name="value" count="42" flag="true" /~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)
	require.NotNil(t, ast)

	require.Len(t, ast.Children, 1)
	tagNode := ast.Children[0].(*TagNode)

	// All attributes stored as strings
	name, ok := tagNode.Attributes.Get("name")
	assert.True(t, ok)
	assert.Equal(t, "value", name)

	count, ok := tagNode.Attributes.Get("count")
	assert.True(t, ok)
	assert.Equal(t, "42", count)

	flag, ok := tagNode.Attributes.Get("flag")
	assert.True(t, ok)
	assert.Equal(t, "true", flag)
}

func TestParser_AttributeMethods(t *testing.T) {
	input := `{~tag name="value" /~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	tagNode := ast.Children[0].(*TagNode)
	attrs := tagNode.Attributes

	// Test Get
	val, ok := attrs.Get("name")
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	// Test Get with non-existent key
	_, ok = attrs.Get("nonexistent")
	assert.False(t, ok)

	// Test GetDefault
	assert.Equal(t, "value", attrs.GetDefault("name", "default"))
	assert.Equal(t, "default", attrs.GetDefault("nonexistent", "default"))

	// Test Has
	assert.True(t, attrs.Has("name"))
	assert.False(t, attrs.Has("nonexistent"))

	// Test Keys
	keys := attrs.Keys()
	assert.Contains(t, keys, "name")
}

func TestParser_ComplexTemplate(t *testing.T) {
	input := `{~exons.raw~}
<system>You are a helpful assistant.</system>
{~/exons.raw~}

<user>Hello, {~exons.var name="user.name" default="Guest" /~}!</user>

{~section role="assistant"~}
I can help you with {~exons.var name="task" /~}.
{~/section~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Should parse without error - complex templates work
	assert.NotEmpty(t, ast.Children)
}

func TestParser_RootNodeMethods(t *testing.T) {
	input := "Hello"

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	assert.Equal(t, NodeTypeRoot, ast.Type())
	assert.Equal(t, 1, ast.Pos().Line)
	assert.Contains(t, ast.String(), "RootNode")
}

func TestParser_TagNodeMethods(t *testing.T) {
	input := `{~exons.var name="x" /~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	tagNode := ast.Children[0].(*TagNode)

	assert.Equal(t, NodeTypeTag, tagNode.Type())
	assert.True(t, tagNode.IsBuiltin())
	assert.False(t, tagNode.IsRaw())
	assert.Contains(t, tagNode.String(), TagNameVar)
}

func TestParser_TextNodeMethods(t *testing.T) {
	input := "Hello, World!"

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	textNode := ast.Children[0].(*TextNode)

	assert.Equal(t, NodeTypeText, textNode.Type())
	assert.Contains(t, textNode.String(), "Hello")
}

func TestParser_LongTextTruncation(t *testing.T) {
	// Test that TextNode.String() truncates long content
	longContent := "This is a very long string that should be truncated when displayed in the String() method because it exceeds fifty characters"

	node := NewTextNode(longContent, Position{Line: 1, Column: 1})
	str := node.String()

	// Should contain "..." for truncation
	assert.Contains(t, str, "...")
	// Should not contain the full string
	assert.Less(t, len(str), len(longContent)+50) // Allow for formatting overhead
}

func TestParser_NilAttributesMethods(t *testing.T) {
	var attrs Attributes = nil

	// Get returns empty and false
	val, ok := attrs.Get("key")
	assert.Equal(t, "", val)
	assert.False(t, ok)

	// GetDefault returns default
	assert.Equal(t, "default", attrs.GetDefault("key", "default"))

	// Has returns false
	assert.False(t, attrs.Has("key"))

	// Keys returns nil
	assert.Nil(t, attrs.Keys())

	// Map returns empty map
	m := attrs.Map()
	assert.NotNil(t, m)
	assert.Empty(t, m)

	// String returns empty braces
	assert.Equal(t, "{}", attrs.String())
}

func TestParser_AttributesMap(t *testing.T) {
	attrs := Attributes{
		"name":  "value",
		"count": "42",
	}

	m := attrs.Map()
	assert.Equal(t, "value", m["name"])
	assert.Equal(t, "42", m["count"])

	// Modifying the copy shouldn't affect original
	m["name"] = "modified"
	origVal, _ := attrs.Get("name")
	assert.Equal(t, "value", origVal)
}

func TestParser_ConsecutiveTags(t *testing.T) {
	input := `{~a /~}{~b /~}{~c /~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	require.Len(t, ast.Children, 3)

	names := []string{"a", "b", "c"}
	for i, name := range names {
		tagNode, ok := ast.Children[i].(*TagNode)
		require.True(t, ok)
		assert.Equal(t, name, tagNode.Name)
	}
}

func TestParser_BlockTagWithAttributes(t *testing.T) {
	input := `{~section role="system" priority="high"~}Content here{~/section~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	require.Len(t, ast.Children, 1)
	tagNode := ast.Children[0].(*TagNode)

	assert.Equal(t, "section", tagNode.Name)
	assert.False(t, tagNode.SelfClose)

	role, ok := tagNode.Attributes.Get("role")
	assert.True(t, ok)
	assert.Equal(t, RoleSystem, role)

	priority, ok := tagNode.Attributes.Get("priority")
	assert.True(t, ok)
	assert.Equal(t, "high", priority)
}

func TestParser_SingleQuoteAttributes(t *testing.T) {
	input := `{~tag name='single quoted' /~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	tagNode := ast.Children[0].(*TagNode)
	name, ok := tagNode.Attributes.Get("name")
	assert.True(t, ok)
	assert.Equal(t, "single quoted", name)
}

func TestParser_EscapedQuotesInAttributes(t *testing.T) {
	input := `{~tag name="value with \"quotes\"" /~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	tagNode := ast.Children[0].(*TagNode)
	name, ok := tagNode.Attributes.Get("name")
	assert.True(t, ok)
	assert.Equal(t, `value with "quotes"`, name)
}

func TestParser_NestedRawBlockError(t *testing.T) {
	// Nested raw blocks are disallowed - this tests that error path
	// However, since the lexer tokenizes everything, and the parser tracks inRawBlock,
	// we need to construct a scenario where a raw block is detected inside another.
	// Currently, the raw block parser collects tokens until {~/exons.raw~},
	// so inner {~exons.raw~} would be collected as text.
	// The inRawBlock flag protects against recursive parseRawBlock calls.

	// This test verifies the raw block parsing logic works correctly
	input := `{~exons.raw~}Outer content{~/exons.raw~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)
	require.NotNil(t, ast)

	tagNode := ast.Children[0].(*TagNode)
	assert.Equal(t, TagNameRaw, tagNode.Name)
	assert.Equal(t, "Outer content", tagNode.RawContent)
}

func TestParser_RawBlockWithNestedTags(t *testing.T) {
	// Raw block preserves inner tag syntax as literal text
	input := `{~exons.raw~}{~inner~}content{~/inner~}{~/exons.raw~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)
	require.NotNil(t, ast)

	tagNode := ast.Children[0].(*TagNode)
	assert.Equal(t, TagNameRaw, tagNode.Name)
	// The inner tags are preserved as literal text
	assert.Contains(t, tagNode.RawContent, "{~inner~}")
	assert.Contains(t, tagNode.RawContent, "{~/inner~}")
}

func TestParser_RawBlockWithSelfClosingInside(t *testing.T) {
	// Raw block preserves self-closing tags as text
	input := `{~exons.raw~}Before {~tag attr="val" /~} After{~/exons.raw~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	tagNode := ast.Children[0].(*TagNode)
	assert.Contains(t, tagNode.RawContent, "{~tag")
	assert.Contains(t, tagNode.RawContent, "/~}")
}

func TestParser_EmptyAttributes(t *testing.T) {
	attrs := Attributes{}

	// Empty attributes
	assert.Equal(t, "{}", attrs.String())
	keys := attrs.Keys()
	assert.NotNil(t, keys)
	assert.Empty(t, keys)
}

func TestParser_AttributesKeysOrder(t *testing.T) {
	attrs := Attributes{
		"zebra":  "z",
		"apple":  "a",
		"middle": "m",
	}

	keys := attrs.Keys()
	// Keys should be sorted alphabetically
	assert.Equal(t, []string{"apple", "middle", "zebra"}, keys)
}

func TestParser_AttributesStringFormat(t *testing.T) {
	attrs := Attributes{
		"name": "value",
	}

	str := attrs.String()
	assert.Contains(t, str, "name=")
	assert.Contains(t, str, "\"value\"")
}

func TestParser_TagNodeTypeRaw(t *testing.T) {
	input := `{~exons.raw~}content{~/exons.raw~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	tagNode := ast.Children[0].(*TagNode)
	// Raw blocks should return NodeTypeRaw
	assert.Equal(t, NodeTypeRaw, tagNode.Type())
}

func TestParser_TagNodeStringBlockForm(t *testing.T) {
	input := `{~section~}content{~/section~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	tagNode := ast.Children[0].(*TagNode)
	str := tagNode.String()
	assert.Contains(t, str, "block")
	assert.Contains(t, str, "children=1")
}

func TestParser_ParserErrorString(t *testing.T) {
	err := &ParserError{
		Message: ErrMsgMismatchedTag,
		Position: Position{
			Line:   5,
			Column: 10,
		},
	}

	errStr := err.Error()
	assert.Contains(t, errStr, ErrMsgMismatchedTag)
	assert.Contains(t, errStr, "5")
}

// TestParser_ParseConditional tests the parseConditional function
func TestParser_ParseConditional(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		errMsg   string
		checkAST func(*testing.T, *RootNode)
	}{
		{
			name:    "valid if...else structure",
			input:   `{~exons.if eval="user.isAdmin"~}Admin{~exons.else~}Guest{~/exons.if~}`,
			wantErr: false,
			checkAST: func(t *testing.T, ast *RootNode) {
				require.Len(t, ast.Children, 1)
				condNode, ok := ast.Children[0].(*ConditionalNode)
				require.True(t, ok)
				assert.Len(t, condNode.Branches, 2)
				assert.Equal(t, "user.isAdmin", condNode.Branches[0].Condition)
				assert.False(t, condNode.Branches[0].IsElse)
				assert.True(t, condNode.Branches[1].IsElse)
			},
		},
		{
			name:    "multiple elseif branches",
			input:   `{~exons.if eval="x > 10"~}High{~exons.elseif eval="x > 5"~}Medium{~exons.else~}Low{~/exons.if~}`,
			wantErr: false,
			checkAST: func(t *testing.T, ast *RootNode) {
				condNode := ast.Children[0].(*ConditionalNode)
				assert.Len(t, condNode.Branches, 3)
				assert.Equal(t, "x > 10", condNode.Branches[0].Condition)
				assert.Equal(t, "x > 5", condNode.Branches[1].Condition)
				assert.False(t, condNode.Branches[1].IsElse)
				assert.True(t, condNode.Branches[2].IsElse)
			},
		},
		{
			name:    "missing eval attribute on if",
			input:   `{~exons.if~}Content{~/exons.if~}`,
			wantErr: true,
			errMsg:  ErrMsgCondMissingEval,
		},
		{
			name:    "missing eval attribute on elseif",
			input:   `{~exons.if eval="true"~}A{~exons.elseif~}B{~/exons.if~}`,
			wantErr: true,
			errMsg:  ErrMsgCondMissingEval,
		},
		{
			name:    "else with eval attribute",
			input:   `{~exons.if eval="true"~}A{~exons.else eval="false"~}B{~/exons.if~}`,
			wantErr: true,
			errMsg:  ErrMsgCondInvalidElse,
		},
		{
			name:    "else not as final branch",
			input:   `{~exons.if eval="a"~}A{~exons.else~}B{~exons.elseif eval="c"~}C{~/exons.if~}`,
			wantErr: true,
			errMsg:  ErrMsgCondElseNotLast,
		},
		{
			name:    "nested conditionals",
			input:   `{~exons.if eval="outer"~}{~exons.if eval="inner"~}Nested{~/exons.if~}{~/exons.if~}`,
			wantErr: false,
			checkAST: func(t *testing.T, ast *RootNode) {
				condNode := ast.Children[0].(*ConditionalNode)
				assert.Len(t, condNode.Branches, 1)
				// Inner conditional should be in children
				assert.Len(t, condNode.Branches[0].Children, 1)
				innerCond, ok := condNode.Branches[0].Children[0].(*ConditionalNode)
				assert.True(t, ok)
				assert.Equal(t, "inner", innerCond.Branches[0].Condition)
			},
		},
		{
			name:    "unclosed conditional block",
			input:   `{~exons.if eval="true"~}Content`,
			wantErr: true,
			errMsg:  ErrMsgCondNotClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, nil)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)

			parser := NewParser(tokens, nil)
			ast, err := parser.Parse()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, ast)
				if tt.checkAST != nil {
					tt.checkAST(t, ast)
				}
			}
		})
	}
}

// TestParser_ParseFor tests the parseFor and parseForBody functions
func TestParser_ParseFor(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		errMsg   string
		checkAST func(*testing.T, *RootNode)
	}{
		{
			name:    "valid for loop with item and in",
			input:   `{~exons.for item="user" in="users"~}Name: {~exons.var name="user.name" /~}{~/exons.for~}`,
			wantErr: false,
			checkAST: func(t *testing.T, ast *RootNode) {
				require.Len(t, ast.Children, 1)
				forNode, ok := ast.Children[0].(*ForNode)
				require.True(t, ok)
				assert.Equal(t, "user", forNode.ItemVar)
				assert.Equal(t, "users", forNode.Source)
				assert.Equal(t, "", forNode.IndexVar)
				assert.Equal(t, 0, forNode.Limit)
			},
		},
		{
			name:    "for loop with index variable",
			input:   `{~exons.for item="x" index="i" in="items"~}Index: {~exons.var name="i" /~}{~/exons.for~}`,
			wantErr: false,
			checkAST: func(t *testing.T, ast *RootNode) {
				forNode := ast.Children[0].(*ForNode)
				assert.Equal(t, "x", forNode.ItemVar)
				assert.Equal(t, "i", forNode.IndexVar)
				assert.Equal(t, "items", forNode.Source)
			},
		},
		{
			name:    "for loop with limit attribute",
			input:   `{~exons.for item="x" in="items" limit="10"~}Item{~/exons.for~}`,
			wantErr: false,
			checkAST: func(t *testing.T, ast *RootNode) {
				forNode := ast.Children[0].(*ForNode)
				assert.Equal(t, 10, forNode.Limit)
			},
		},
		{
			name:    "missing item attribute",
			input:   `{~exons.for in="items"~}Content{~/exons.for~}`,
			wantErr: true,
			errMsg:  ErrMsgForMissingItem,
		},
		{
			name:    "missing in attribute",
			input:   `{~exons.for item="x"~}Content{~/exons.for~}`,
			wantErr: true,
			errMsg:  ErrMsgForMissingIn,
		},
		{
			name:    "invalid limit - negative",
			input:   `{~exons.for item="x" in="items" limit="-1"~}Content{~/exons.for~}`,
			wantErr: true,
			errMsg:  ErrMsgForInvalidLimit,
		},
		{
			name:    "invalid limit - non-numeric",
			input:   `{~exons.for item="x" in="items" limit="abc"~}Content{~/exons.for~}`,
			wantErr: true,
			errMsg:  ErrMsgForInvalidLimit,
		},
		{
			name:    "nested for loops",
			input:   `{~exons.for item="row" in="rows"~}{~exons.for item="col" in="row.columns"~}Cell{~/exons.for~}{~/exons.for~}`,
			wantErr: false,
			checkAST: func(t *testing.T, ast *RootNode) {
				forNode := ast.Children[0].(*ForNode)
				assert.Len(t, forNode.Children, 1)
				innerFor, ok := forNode.Children[0].(*ForNode)
				assert.True(t, ok)
				assert.Equal(t, "col", innerFor.ItemVar)
				assert.Equal(t, "row.columns", innerFor.Source)
			},
		},
		{
			name:    "unclosed for block",
			input:   `{~exons.for item="x" in="items"~}Content`,
			wantErr: true,
			errMsg:  ErrMsgForNotClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, nil)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)

			parser := NewParser(tokens, nil)
			ast, err := parser.Parse()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, ast)
				if tt.checkAST != nil {
					tt.checkAST(t, ast)
				}
			}
		})
	}
}

// TestParser_ParseSwitch tests the parseSwitch and parseSwitchCase functions
func TestParser_ParseSwitch(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		errMsg   string
		checkAST func(*testing.T, *RootNode)
	}{
		{
			name:    "valid switch with single case",
			input:   `{~exons.switch eval="status"~}{~exons.case value="active"~}Active{~/exons.case~}{~/exons.switch~}`,
			wantErr: false,
			checkAST: func(t *testing.T, ast *RootNode) {
				require.Len(t, ast.Children, 1)
				switchNode, ok := ast.Children[0].(*SwitchNode)
				require.True(t, ok)
				assert.Equal(t, "status", switchNode.Expression)
				assert.Len(t, switchNode.Cases, 1)
				assert.Equal(t, "active", switchNode.Cases[0].Value)
				assert.Nil(t, switchNode.Default)
			},
		},
		{
			name:    "switch with multiple cases",
			input:   `{~exons.switch eval="role"~}{~exons.case value="admin"~}A{~/exons.case~}{~exons.case value="user"~}U{~/exons.case~}{~/exons.switch~}`,
			wantErr: false,
			checkAST: func(t *testing.T, ast *RootNode) {
				switchNode := ast.Children[0].(*SwitchNode)
				assert.Len(t, switchNode.Cases, 2)
				assert.Equal(t, "admin", switchNode.Cases[0].Value)
				assert.Equal(t, "user", switchNode.Cases[1].Value)
			},
		},
		{
			name:    "switch with default case",
			input:   `{~exons.switch eval="x"~}{~exons.case value="1"~}One{~/exons.case~}{~exons.casedefault~}Other{~/exons.casedefault~}{~/exons.switch~}`,
			wantErr: false,
			checkAST: func(t *testing.T, ast *RootNode) {
				switchNode := ast.Children[0].(*SwitchNode)
				assert.Len(t, switchNode.Cases, 1)
				assert.NotNil(t, switchNode.Default)
				assert.True(t, switchNode.Default.IsDefault)
			},
		},
		{
			name:    "switch with default not as last case",
			input:   `{~exons.switch eval="x"~}{~exons.casedefault~}Default{~/exons.casedefault~}{~exons.case value="1"~}One{~/exons.case~}{~/exons.switch~}`,
			wantErr: true,
			errMsg:  ErrMsgSwitchDefaultNotLast,
		},
		{
			name:    "switch missing eval attribute",
			input:   `{~exons.switch~}{~exons.case value="1"~}One{~/exons.case~}{~/exons.switch~}`,
			wantErr: true,
			errMsg:  ErrMsgSwitchMissingEval,
		},
		{
			name:    "case with value attribute",
			input:   `{~exons.switch eval="color"~}{~exons.case value="red"~}Red{~/exons.case~}{~/exons.switch~}`,
			wantErr: false,
			checkAST: func(t *testing.T, ast *RootNode) {
				switchNode := ast.Children[0].(*SwitchNode)
				assert.Equal(t, "red", switchNode.Cases[0].Value)
				assert.Equal(t, "", switchNode.Cases[0].Eval)
			},
		},
		{
			name:    "case with eval attribute",
			input:   `{~exons.switch eval="age"~}{~exons.case eval="age > 18"~}Adult{~/exons.case~}{~/exons.switch~}`,
			wantErr: false,
			checkAST: func(t *testing.T, ast *RootNode) {
				switchNode := ast.Children[0].(*SwitchNode)
				assert.Equal(t, "", switchNode.Cases[0].Value)
				assert.Equal(t, "age > 18", switchNode.Cases[0].Eval)
			},
		},
		{
			name:    "case missing both value and eval",
			input:   `{~exons.switch eval="x"~}{~exons.case~}Content{~/exons.case~}{~/exons.switch~}`,
			wantErr: true,
			errMsg:  ErrMsgSwitchMissingValue,
		},
		{
			name:    "unclosed switch block",
			input:   `{~exons.switch eval="x"~}{~exons.case value="1"~}One{~/exons.case~}`,
			wantErr: true,
			errMsg:  ErrMsgSwitchNotClosed,
		},
		{
			name:    "nested switch statements",
			input:   `{~exons.switch eval="outer"~}{~exons.case value="1"~}{~exons.switch eval="inner"~}{~exons.case value="a"~}Nested{~/exons.case~}{~/exons.switch~}{~/exons.case~}{~/exons.switch~}`,
			wantErr: false,
			checkAST: func(t *testing.T, ast *RootNode) {
				switchNode := ast.Children[0].(*SwitchNode)
				assert.Len(t, switchNode.Cases, 1)
				assert.Len(t, switchNode.Cases[0].Children, 1)
				innerSwitch, ok := switchNode.Cases[0].Children[0].(*SwitchNode)
				assert.True(t, ok)
				assert.Equal(t, "inner", innerSwitch.Expression)
			},
		},
		{
			name:    "duplicate default case",
			input:   `{~exons.switch eval="x"~}{~exons.casedefault~}First{~/exons.casedefault~}{~exons.casedefault~}Second{~/exons.casedefault~}{~/exons.switch~}`,
			wantErr: true,
			errMsg:  ErrMsgSwitchDuplicateDefault,
		},
		{
			name:    "unclosed case block",
			input:   `{~exons.switch eval="x"~}{~exons.case value="1"~}Content{~/exons.switch~}`,
			wantErr: true,
			errMsg:  ErrMsgSwitchCaseNotClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, nil)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)

			parser := NewParser(tokens, nil)
			ast, err := parser.Parse()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, ast)
				if tt.checkAST != nil {
					tt.checkAST(t, ast)
				}
			}
		})
	}
}

// TestParser_ParseCommentBlock tests the parseCommentBlock function
func TestParser_ParseCommentBlock(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		errMsg   string
		checkAST func(*testing.T, *RootNode)
	}{
		{
			name:    "comment with text content returns nil",
			input:   `{~exons.comment~}This is a comment{~/exons.comment~}`,
			wantErr: false,
			checkAST: func(t *testing.T, ast *RootNode) {
				// Comment nodes produce nil, so they're not in the AST
				assert.Empty(t, ast.Children)
			},
		},
		{
			name:    "comment with tags inside - all discarded",
			input:   `{~exons.comment~}{~exons.var name="x" /~}{~section~}ignored{~/section~}{~/exons.comment~}`,
			wantErr: false,
			checkAST: func(t *testing.T, ast *RootNode) {
				assert.Empty(t, ast.Children)
			},
		},
		{
			name:    "unclosed comment",
			input:   `{~exons.comment~}This comment never closes`,
			wantErr: true,
			errMsg:  ErrMsgMismatchedTag,
		},
		{
			name:    "multiple comments in sequence",
			input:   `{~exons.comment~}First{~/exons.comment~}Text{~exons.comment~}Second{~/exons.comment~}`,
			wantErr: false,
			checkAST: func(t *testing.T, ast *RootNode) {
				// Only the text between comments should remain
				require.Len(t, ast.Children, 1)
				textNode, ok := ast.Children[0].(*TextNode)
				require.True(t, ok)
				assert.Equal(t, "Text", textNode.Content)
			},
		},
		{
			name:    "empty comment block",
			input:   `{~exons.comment~}{~/exons.comment~}`,
			wantErr: false,
			checkAST: func(t *testing.T, ast *RootNode) {
				assert.Empty(t, ast.Children)
			},
		},
		{
			name:    "comment with multiline content",
			input:   "{~exons.comment~}\nLine 1\nLine 2\nLine 3\n{~/exons.comment~}",
			wantErr: false,
			checkAST: func(t *testing.T, ast *RootNode) {
				assert.Empty(t, ast.Children)
			},
		},
		{
			name:    "comment mixed with regular content",
			input:   `Before{~exons.comment~}Hidden{~/exons.comment~}After`,
			wantErr: false,
			checkAST: func(t *testing.T, ast *RootNode) {
				require.Len(t, ast.Children, 2)
				assert.Equal(t, "Before", ast.Children[0].(*TextNode).Content)
				assert.Equal(t, "After", ast.Children[1].(*TextNode).Content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, nil)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)

			parser := NewParser(tokens, nil)
			ast, err := parser.Parse()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, ast)
				if tt.checkAST != nil {
					tt.checkAST(t, ast)
				}
			}
		})
	}
}
