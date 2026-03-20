package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractYAMLFrontmatter_NoFrontmatter(t *testing.T) {
	source := `Hello {~exons.var name="user" /~}!`

	result, err := ExtractYAMLFrontmatter(source)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.HasFrontmatter)
	assert.Empty(t, result.FrontmatterYAML)
	assert.Equal(t, source, result.TemplateBody)
}

func TestExtractYAMLFrontmatter_BasicFrontmatter(t *testing.T) {
	source := `---
name: test
---
Hello World!`

	result, err := ExtractYAMLFrontmatter(source)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.HasFrontmatter)
	assert.Equal(t, "name: test", result.FrontmatterYAML)
	assert.Equal(t, "Hello World!", result.TemplateBody)
}

func TestExtractYAMLFrontmatter_WithLeadingWhitespace(t *testing.T) {
	source := `  ---
name: test
---
Template body`

	result, err := ExtractYAMLFrontmatter(source)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.HasFrontmatter)
	assert.Equal(t, "name: test", result.FrontmatterYAML)
	assert.Equal(t, "Template body", result.TemplateBody)
}

func TestExtractYAMLFrontmatter_ComplexYAML(t *testing.T) {
	source := `---
name: customer-support
description: A support agent
model:
  api: chat
  provider: openai
  name: gpt-4
  parameters:
    temperature: 0.7
    max_tokens: 2048
inputs:
  query:
    type: string
    required: true
---
Hello {~exons.var name="user" /~}!`

	result, err := ExtractYAMLFrontmatter(source)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.HasFrontmatter)
	assert.Contains(t, result.FrontmatterYAML, "name: customer-support")
	assert.Contains(t, result.FrontmatterYAML, "temperature: 0.7")
	assert.Equal(t, `Hello {~exons.var name="user" /~}!`, result.TemplateBody)
}

func TestExtractYAMLFrontmatter_UnclosedError(t *testing.T) {
	source := `---
name: test
Missing close delimiter
Hello World!`

	result, err := ExtractYAMLFrontmatter(source)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), ErrMsgFrontmatterUnclosed)
}

func TestExtractYAMLFrontmatter_EmptyFrontmatter(t *testing.T) {
	source := `---
---
Template`

	result, err := ExtractYAMLFrontmatter(source)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.HasFrontmatter)
	assert.Empty(t, result.FrontmatterYAML)
	assert.Equal(t, "Template", result.TemplateBody)
}

func TestExtractYAMLFrontmatter_NoTemplateBody(t *testing.T) {
	source := `---
name: test
---`

	result, err := ExtractYAMLFrontmatter(source)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.HasFrontmatter)
	assert.Equal(t, "name: test", result.FrontmatterYAML)
	assert.Empty(t, result.TemplateBody)
}

func TestExtractYAMLFrontmatter_FrontmatterInMiddle(t *testing.T) {
	// Frontmatter that appears after template content should NOT be extracted
	source := `Template start
---
name: test
---
Template end`

	result, err := ExtractYAMLFrontmatter(source)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Frontmatter in middle is not extracted
	assert.False(t, result.HasFrontmatter)
	assert.Empty(t, result.FrontmatterYAML)
	assert.Equal(t, source, result.TemplateBody)
}

func TestExtractYAMLFrontmatter_EmptySource(t *testing.T) {
	result, err := ExtractYAMLFrontmatter("")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.HasFrontmatter)
	assert.Empty(t, result.FrontmatterYAML)
	assert.Empty(t, result.TemplateBody)
}

func TestExtractYAMLFrontmatter_OnlyWhitespace(t *testing.T) {
	result, err := ExtractYAMLFrontmatter("   \n\t  ")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.HasFrontmatter)
}

func TestExtractYAMLFrontmatter_WithTemplateVarsInYAML(t *testing.T) {
	// This tests that exons tags inside the YAML are preserved
	source := `---
model:
  name: "{~exons.env name=\"MODEL_NAME\" default=\"gpt-4\" /~}"
---
Template`

	result, err := ExtractYAMLFrontmatter(source)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.HasFrontmatter)
	// YAML preserves the escaped quotes
	assert.Contains(t, result.FrontmatterYAML, `exons.env`)
	assert.Contains(t, result.FrontmatterYAML, `MODEL_NAME`)
}

func TestExtractYAMLFrontmatter_Position(t *testing.T) {
	source := `---
name: test
---
Body`

	result, err := ExtractYAMLFrontmatter(source)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Frontmatter starts at line 1, column 1
	assert.Equal(t, 1, result.FrontmatterPosition.Line)
	assert.Equal(t, 1, result.FrontmatterPosition.Column)
	assert.Equal(t, 0, result.FrontmatterPosition.Offset)
}

func TestExtractYAMLFrontmatter_CarriageReturnNewline(t *testing.T) {
	source := "---\r\nname: test\r\n---\r\nTemplate"

	result, err := ExtractYAMLFrontmatter(source)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.HasFrontmatter)
	assert.Equal(t, "name: test", result.FrontmatterYAML)
	assert.Equal(t, "Template", result.TemplateBody)
}

func TestExtractYAMLFrontmatter_WithBOM(t *testing.T) {
	// UTF-8 BOM followed by frontmatter
	source := "\xef\xbb\xbf---\nname: test\n---\nTemplate"

	result, err := ExtractYAMLFrontmatter(source)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.HasFrontmatter)
	assert.Equal(t, "name: test", result.FrontmatterYAML)
	assert.Equal(t, "Template", result.TemplateBody)
}

func TestExtractYAMLFrontmatter_LegacyJSONConfigError(t *testing.T) {
	// Legacy JSON config block should return an error with migration hint
	source := `{~exons.config~}
{"name": "test"}
{~/exons.config~}
Template`

	result, err := ExtractYAMLFrontmatter(source)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), ErrMsgLegacyJSONConfigDetected)
}

func TestExtractYAMLFrontmatter_DashesInContent(t *testing.T) {
	// Markdown horizontal rule inside content should not be treated as frontmatter
	source := `---
name: test
---
Some content

---

More content with dashes`

	result, err := ExtractYAMLFrontmatter(source)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.HasFrontmatter)
	// The dashes in the middle should be part of template body
	assert.Contains(t, result.TemplateBody, "More content with dashes")
}

func TestExtractYAMLFrontmatter_NoNewlineAfterOpeningDelimiter(t *testing.T) {
	// "---" without newline should not be treated as frontmatter
	source := `---foo
name: test
---
Template`

	result, err := ExtractYAMLFrontmatter(source)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Not frontmatter because --- is not followed by newline
	assert.False(t, result.HasFrontmatter)
	assert.Equal(t, source, result.TemplateBody)
}

func TestExtractYAMLFrontmatter_MultilineValues(t *testing.T) {
	source := `---
name: test
description: |
  This is a
  multiline description
---
Template`

	result, err := ExtractYAMLFrontmatter(source)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.HasFrontmatter)
	assert.Contains(t, result.FrontmatterYAML, "multiline description")
	assert.Equal(t, "Template", result.TemplateBody)
}

func TestExtractYAMLFrontmatter_WithComments(t *testing.T) {
	source := `---
# This is a comment
name: test
# Another comment
model:
  name: gpt-4  # inline comment
---
Template`

	result, err := ExtractYAMLFrontmatter(source)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.HasFrontmatter)
	assert.Contains(t, result.FrontmatterYAML, "# This is a comment")
	assert.Contains(t, result.FrontmatterYAML, "name: test")
}

// Backward compatibility test with deprecated ExtractConfigBlock function
func TestExtractConfigBlock_Deprecated(t *testing.T) {
	source := `---
name: test
---
Template`

	result, err := ExtractConfigBlock(source, DefaultLexerConfig())
	require.NoError(t, err)
	require.NotNil(t, result)

	// Deprecated method accessors should still work
	assert.True(t, result.HasConfig())
	assert.Equal(t, "name: test", result.ConfigJSON())
	assert.Equal(t, "Template", result.TemplateBody)
}

func TestCalculatePosition(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		want   Position
	}{
		{
			name:   "empty",
			prefix: "",
			want:   Position{Offset: 0, Line: 1, Column: 1},
		},
		{
			name:   "single line",
			prefix: "hello",
			want:   Position{Offset: 5, Line: 1, Column: 6},
		},
		{
			name:   "two lines",
			prefix: "hello\nworld",
			want:   Position{Offset: 11, Line: 2, Column: 6},
		},
		{
			name:   "three lines",
			prefix: "a\nb\nc",
			want:   Position{Offset: 5, Line: 3, Column: 2},
		},
		{
			name:   "trailing newline",
			prefix: "hello\n",
			want:   Position{Offset: 6, Line: 2, Column: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculatePosition(tt.prefix)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfigError_Error(t *testing.T) {
	err := NewConfigError(ErrMsgFrontmatterUnclosed, Position{Line: 5, Column: 10}, nil)
	assert.Contains(t, err.Error(), ErrMsgFrontmatterUnclosed)
	assert.Contains(t, err.Error(), "line 5")
	assert.Contains(t, err.Error(), "column 10")
}

func TestConfigError_Unwrap(t *testing.T) {
	cause := &ConfigError{Message: "inner error"}
	err := NewConfigError("outer", Position{}, cause)
	assert.Equal(t, cause, err.Unwrap())
}
