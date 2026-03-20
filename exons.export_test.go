package exons

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportDirectory_Basic(t *testing.T) {
	spec := &Spec{
		Name:        "export-skill",
		Description: "A skill for export",
		Type:        DocumentTypeSkill,
		Body:        "Skill body content",
	}

	zipData, err := ExportDirectory(spec, nil)
	require.NoError(t, err)
	require.NotNil(t, zipData)

	// Read back the zip
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	require.NoError(t, err)
	require.Len(t, reader.File, 1)
	assert.Equal(t, DocumentFilenameSkill, reader.File[0].Name)
}

func TestExportDirectory_WithResources(t *testing.T) {
	spec := &Spec{
		Name:        "res-export",
		Description: "Export with resources",
		Type:        DocumentTypeSkill,
		Body:        "Body",
	}
	resources := map[string][]byte{
		"config.json": []byte(`{"key":"val"}`),
		"data.txt":    []byte("some data"),
	}

	zipData, err := ExportDirectory(spec, resources)
	require.NoError(t, err)
	require.NotNil(t, zipData)

	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	require.NoError(t, err)
	// 1 document + 2 resources
	assert.Len(t, reader.File, 3)

	// Verify all expected files are present
	names := make(map[string]bool)
	for _, f := range reader.File {
		names[f.Name] = true
	}
	assert.True(t, names[DocumentFilenameSkill])
	assert.True(t, names["config.json"])
	assert.True(t, names["data.txt"])
}

func TestExportDirectory_NilSpec(t *testing.T) {
	zipData, err := ExportDirectory(nil, nil)
	assert.Error(t, err)
	assert.Nil(t, zipData)
	assert.Contains(t, err.Error(), ErrMsgExportFailed)
}

func TestExportDirectory_AgentType(t *testing.T) {
	spec := &Spec{
		Name:        "agent-export",
		Description: "An agent for export",
		Type:        DocumentTypeAgent,
		Body:        "Agent content",
	}

	zipData, err := ExportDirectory(spec, nil)
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	require.NoError(t, err)
	require.Len(t, reader.File, 1)
	assert.Equal(t, DocumentFilenameAgent, reader.File[0].Name)
}

func TestExport_Import_Roundtrip(t *testing.T) {
	original := &Spec{
		Name:        "roundtrip-skill",
		Description: "Roundtrip test skill",
		Type:        DocumentTypeSkill,
		Body:        "Roundtrip body content",
		Inputs: map[string]*InputDef{
			"query": {Type: SchemaTypeString, Required: true},
		},
	}
	resources := map[string][]byte{
		"extra.txt": []byte("extra file data"),
	}

	// Export
	zipData, err := ExportDirectory(original, resources)
	require.NoError(t, err)

	// Import
	result, err := ImportDirectory(zipData)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Spec)

	assert.Equal(t, original.Name, result.Spec.Name)
	assert.Equal(t, original.Description, result.Spec.Description)
	assert.Equal(t, original.Type, result.Spec.Type)
	assert.Equal(t, original.Body, result.Spec.Body)
	require.Contains(t, result.Spec.Inputs, "query")
	assert.True(t, result.Spec.Inputs["query"].Required)

	// Verify resources survived
	assert.Len(t, result.Resources, 1)
	assert.Equal(t, []byte("extra file data"), result.Resources["extra.txt"])
}

func TestDocumentFilename(t *testing.T) {
	tests := []struct {
		docType  DocumentType
		expected string
	}{
		{DocumentTypeAgent, DocumentFilenameAgent},
		{DocumentTypePrompt, DocumentFilenamePrompt},
		{DocumentTypeSkill, DocumentFilenameSkill},
		{"", DocumentFilenameSkill},                     // default
		{DocumentType("custom"), DocumentFilenameSkill}, // unknown defaults to skill
	}

	for _, tc := range tests {
		t.Run(string(tc.docType), func(t *testing.T) {
			assert.Equal(t, tc.expected, documentFilename(tc.docType))
		})
	}
}

func TestExportDirectory_PromptType(t *testing.T) {
	spec := &Spec{
		Name:        "prompt-export",
		Description: "A prompt for export",
		Type:        DocumentTypePrompt,
		Body:        "Prompt content",
	}

	zipData, err := ExportDirectory(spec, nil)
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	require.NoError(t, err)
	require.Len(t, reader.File, 1)
	assert.Equal(t, DocumentFilenamePrompt, reader.File[0].Name)

	// Also verify content can be read back
	rc, err := reader.File[0].Open()
	require.NoError(t, err)
	defer rc.Close()
	content, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Prompt content")
}
