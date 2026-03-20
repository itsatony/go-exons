package exons

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportFromSkillMD_Basic(t *testing.T) {
	content := "---\nname: test-skill\ndescription: A test skill\n---\n"
	spec, err := ImportFromSkillMD(content)
	require.NoError(t, err)
	require.NotNil(t, spec)
	assert.Equal(t, "test-skill", spec.Name)
	assert.Equal(t, "A test skill", spec.Description)
}

func TestImportFromSkillMD_Empty(t *testing.T) {
	spec, err := ImportFromSkillMD("")
	assert.Error(t, err)
	assert.Nil(t, spec)
	assert.Contains(t, err.Error(), ErrMsgSkillMDInvalidFormat)
}

func TestImportFromSkillMD_MissingFrontmatter(t *testing.T) {
	content := "Just some text without frontmatter"
	spec, err := ImportFromSkillMD(content)
	assert.Error(t, err)
	assert.Nil(t, spec)
	assert.Contains(t, err.Error(), ErrMsgSkillMDMissingFM)
}

func TestImportFromSkillMD_UnclosedFrontmatter(t *testing.T) {
	content := "---\nname: test\ndescription: no closing delimiter"
	spec, err := ImportFromSkillMD(content)
	assert.Error(t, err)
	assert.Nil(t, spec)
	assert.Contains(t, err.Error(), ErrMsgSkillMDInvalidFormat)
}

func TestImportFromSkillMD_WithBody(t *testing.T) {
	content := "---\nname: body-skill\ndescription: Skill with body\n---\nThis is the body content.\nSecond line.\n"
	spec, err := ImportFromSkillMD(content)
	require.NoError(t, err)
	require.NotNil(t, spec)
	assert.Equal(t, "body-skill", spec.Name)
	assert.Contains(t, spec.Body, "This is the body content.")
	assert.Contains(t, spec.Body, "Second line.")
}

func TestImportFromSkillMD_WithInputs(t *testing.T) {
	content := "---\nname: input-skill\ndescription: Skill with inputs\ninputs:\n  query:\n    type: string\n    required: true\n---\nBody here\n"
	spec, err := ImportFromSkillMD(content)
	require.NoError(t, err)
	require.NotNil(t, spec)
	require.Contains(t, spec.Inputs, "query")
	assert.Equal(t, SchemaTypeString, spec.Inputs["query"].Type)
	assert.True(t, spec.Inputs["query"].Required)
}

func TestExportToSkillMD_Basic(t *testing.T) {
	spec := &Spec{
		Name:        "export-skill",
		Description: "A skill for export",
		Body:        "Body content here",
	}

	data, err := spec.ExportToSkillMD()
	require.NoError(t, err)
	require.NotNil(t, data)

	content := string(data)
	assert.Contains(t, content, YAMLFrontmatterDelimiter)
	assert.Contains(t, content, "export-skill")
	assert.Contains(t, content, "Body content here")
}

func TestExportToSkillMD_NilSpec(t *testing.T) {
	var spec *Spec
	data, err := spec.ExportToSkillMD()
	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestExportToSkillMD_StripsExecution(t *testing.T) {
	spec := &Spec{
		Name:        "stripped-skill",
		Description: "Should not include execution",
		Body:        "Body",
		Extensions:  map[string]any{"custom_key": "custom_value"},
	}

	data, err := spec.ExportToSkillMD()
	require.NoError(t, err)
	content := string(data)

	// Extensions should NOT appear in Agent Skills export
	assert.NotContains(t, content, "custom_key")
}

func TestExportToSkillMD_Import_Roundtrip(t *testing.T) {
	original := &Spec{
		Name:        "roundtrip-skill",
		Description: "Roundtrip test",
		Body:        "Template body content",
		Inputs: map[string]*InputDef{
			"name": {Type: SchemaTypeString, Required: true, Description: "User name"},
		},
		Outputs: map[string]*OutputDef{
			"greeting": {Type: SchemaTypeString, Description: "Greeting message"},
		},
	}

	// Export
	data, err := original.ExportToSkillMD()
	require.NoError(t, err)

	// Import
	imported, err := ImportFromSkillMD(string(data))
	require.NoError(t, err)
	require.NotNil(t, imported)

	assert.Equal(t, original.Name, imported.Name)
	assert.Equal(t, original.Description, imported.Description)
	assert.Equal(t, original.Body, imported.Body)
	require.Contains(t, imported.Inputs, "name")
	assert.True(t, imported.Inputs["name"].Required)
	assert.Equal(t, "User name", imported.Inputs["name"].Description)
}
