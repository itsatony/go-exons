package exons

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestZip builds an in-memory zip archive from a map of filename → content.
func createTestZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, data := range files {
		f, err := w.Create(name)
		require.NoError(t, err)
		_, err = f.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func TestImport_Markdown(t *testing.T) {
	doc := "---\nname: test-skill\ndescription: A test skill\ntype: skill\n---\nHello world\n"
	result, err := Import([]byte(doc), "test.md")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Spec)
	assert.Equal(t, "test-skill", result.Spec.Name)
	assert.Equal(t, "A test skill", result.Spec.Description)
	assert.Equal(t, DocumentTypeSkill, result.Spec.Type)
	assert.Contains(t, result.Spec.Body, "Hello world")
	assert.Empty(t, result.Resources)
}

func TestImport_Zip(t *testing.T) {
	doc := "---\nname: zip-skill\ndescription: A zipped skill\ntype: skill\n---\nBody from zip\n"
	zipData := createTestZip(t, map[string][]byte{
		DocumentFilenameSkill: []byte(doc),
	})

	result, err := Import(zipData, "bundle.zip")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Spec)
	assert.Equal(t, "zip-skill", result.Spec.Name)
	assert.Contains(t, result.Spec.Body, "Body from zip")
	assert.Empty(t, result.Resources)
}

func TestImport_ZipWithResources(t *testing.T) {
	doc := "---\nname: res-skill\ndescription: Skill with resources\ntype: skill\n---\nBody\n"
	zipData := createTestZip(t, map[string][]byte{
		DocumentFilenameSkill: []byte(doc),
		"data/config.json":    []byte(`{"key":"value"}`),
		"assets/logo.png":     []byte("fake-png-data"),
	})

	result, err := Import(zipData, "bundle.zip")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Spec)
	assert.Equal(t, "res-skill", result.Spec.Name)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, []byte(`{"key":"value"}`), result.Resources["data/config.json"])
	assert.Equal(t, []byte("fake-png-data"), result.Resources["assets/logo.png"])
}

func TestImport_ZipNoDocument(t *testing.T) {
	zipData := createTestZip(t, map[string][]byte{
		"readme.txt":       []byte("not a document"),
		"data/config.json": []byte(`{}`),
	})

	result, err := Import(zipData, "bundle.zip")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), ErrMsgImportNoDocument)
}

func TestImport_EmptyData(t *testing.T) {
	result, err := Import([]byte{}, "test.md")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), ErrMsgImportFailed)
}

func TestImport_DefaultExtension(t *testing.T) {
	doc := "---\nname: default-ext\ndescription: Default extension test\ntype: skill\n---\nBody text\n"
	result, err := Import([]byte(doc), "myfile.txt")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "default-ext", result.Spec.Name)
}

func TestImportDirectory_AgentMD(t *testing.T) {
	doc := "---\nname: test-agent\ndescription: An agent\ntype: agent\n---\nAgent body\n"
	zipData := createTestZip(t, map[string][]byte{
		DocumentFilenameAgent: []byte(doc),
	})

	result, err := ImportDirectory(zipData)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Spec)
	assert.Equal(t, "test-agent", result.Spec.Name)
	assert.Equal(t, DocumentTypeAgent, result.Spec.Type)
}

func TestImportDirectory_PromptMD(t *testing.T) {
	doc := "---\nname: test-prompt\ndescription: A prompt\ntype: prompt\n---\nPrompt body\n"
	zipData := createTestZip(t, map[string][]byte{
		DocumentFilenamePrompt: []byte(doc),
	})

	result, err := ImportDirectory(zipData)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Spec)
	assert.Equal(t, "test-prompt", result.Spec.Name)
	assert.Equal(t, DocumentTypePrompt, result.Spec.Type)
}
