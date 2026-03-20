package exons

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Parse
// =============================================================================

func TestParse_EmptyData(t *testing.T) {
	_, err := Parse([]byte{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgFrontmatterInvalid)
}

func TestParse_NoFrontmatter_BodyOnly(t *testing.T) {
	data := []byte("Just some body content without frontmatter.")
	spec, err := Parse(data)
	require.NoError(t, err)
	require.NotNil(t, spec)
	assert.Equal(t, DocumentTypeSkill, spec.Type) // default
	assert.Equal(t, "Just some body content without frontmatter.", spec.Body)
	assert.Equal(t, "", spec.Name) // no frontmatter, no name
}

func TestParse_ValidFrontmatterAndBody(t *testing.T) {
	data := []byte("---\nname: my-spec\ndescription: A test spec\ntype: skill\n---\nHello body!")
	spec, err := Parse(data)
	require.NoError(t, err)
	require.NotNil(t, spec)
	assert.Equal(t, "my-spec", spec.Name)
	assert.Equal(t, "A test spec", spec.Description)
	assert.Equal(t, DocumentTypeSkill, spec.Type)
	assert.Equal(t, "Hello body!", spec.Body)
}

func TestParse_BOMPrefix(t *testing.T) {
	data := []byte("\xef\xbb\xbf---\nname: bom-spec\ndescription: BOM test\n---\nBody")
	spec, err := Parse(data)
	require.NoError(t, err)
	require.NotNil(t, spec)
	assert.Equal(t, "bom-spec", spec.Name)
}

func TestParse_CRLFLineEndings(t *testing.T) {
	data := []byte("---\r\nname: crlf-spec\r\ndescription: CRLF test\r\n---\r\nBody content")
	spec, err := Parse(data)
	require.NoError(t, err)
	require.NotNil(t, spec)
	assert.Equal(t, "crlf-spec", spec.Name)
	assert.Equal(t, "Body content", spec.Body)
}

func TestParse_MissingClosingDelimiter(t *testing.T) {
	data := []byte("---\nname: test\ndescription: test\n")
	_, err := Parse(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgFrontmatterUnclosed)
}

func TestParse_SizeLimitExceeded(t *testing.T) {
	large := "---\nname: " + strings.Repeat("a", int(DefaultMaxFrontmatterSize)+1) + "\n---\n"
	_, err := Parse([]byte(large))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgFrontmatterTooLarge)
}

func TestParse_InvalidYAML(t *testing.T) {
	data := []byte("---\ninvalid: [yaml: content\n---\n")
	_, err := Parse(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgFrontmatterParse)
}

func TestParse_ValidationFailure(t *testing.T) {
	// Valid YAML but invalid spec (invalid type)
	data := []byte("---\nname: test-spec\ndescription: desc\ntype: invalid_type\n---\n")
	_, err := Parse(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgInvalidDocumentType)
}

func TestParse_DefaultTypeSkill(t *testing.T) {
	data := []byte("---\nname: test-spec\ndescription: desc\n---\n")
	spec, err := Parse(data)
	require.NoError(t, err)
	assert.Equal(t, DocumentTypeSkill, spec.Type)
}

func TestParse_EmptyBody(t *testing.T) {
	data := []byte("---\nname: test-spec\ndescription: desc\n---\n")
	spec, err := Parse(data)
	require.NoError(t, err)
	assert.Equal(t, "", spec.Body)
}

// =============================================================================
// ParseFile
// =============================================================================

func TestParseFile_Valid(t *testing.T) {
	// Create a temp file
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.exons")
	content := "---\nname: file-spec\ndescription: File test\ntype: prompt\n---\nFile body"
	err := os.WriteFile(fp, []byte(content), 0644)
	require.NoError(t, err)

	spec, err := ParseFile(fp)
	require.NoError(t, err)
	require.NotNil(t, spec)
	assert.Equal(t, "file-spec", spec.Name)
	assert.Equal(t, "File body", spec.Body)
}

func TestParseFile_NonExistentFile(t *testing.T) {
	_, err := ParseFile("/nonexistent/path/to/file.exons")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgFrontmatterExtract)
}

// =============================================================================
// MustParse
// =============================================================================

func TestMustParse_Valid(t *testing.T) {
	data := []byte("---\nname: must-parse\ndescription: must parse test\n---\n")
	assert.NotPanics(t, func() {
		spec := MustParse(data)
		assert.Equal(t, "must-parse", spec.Name)
	})
}

func TestMustParse_Invalid(t *testing.T) {
	data := []byte{} // empty = error
	assert.Panics(t, func() {
		MustParse(data)
	})
}
