package exons

import (
	"testing"

	"github.com/itsatony/go-exons/execution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// GetExtension / SetExtension / RemoveExtension
// =============================================================================

func TestGetExtension_NilSpec(t *testing.T) {
	var s *Spec
	val, ok := s.GetExtension("key")
	assert.Nil(t, val)
	assert.False(t, ok)
}

func TestGetExtension_NilExtensions(t *testing.T) {
	s := &Spec{}
	val, ok := s.GetExtension("key")
	assert.Nil(t, val)
	assert.False(t, ok)
}

func TestGetExtension_MissingKey(t *testing.T) {
	s := &Spec{Extensions: map[string]any{"other": "val"}}
	val, ok := s.GetExtension("key")
	assert.Nil(t, val)
	assert.False(t, ok)
}

func TestSetExtension_CreatesMapIfNil(t *testing.T) {
	s := &Spec{}
	assert.Nil(t, s.Extensions)
	s.SetExtension("key", "value")
	assert.NotNil(t, s.Extensions)
	assert.Equal(t, "value", s.Extensions["key"])
}

func TestSetExtension_NilSpec(t *testing.T) {
	var s *Spec
	// Should not panic
	assert.NotPanics(t, func() {
		s.SetExtension("key", "value")
	})
}

func TestSetExtension_AndGet_Roundtrip(t *testing.T) {
	s := &Spec{}
	s.SetExtension("mykey", 42)
	val, ok := s.GetExtension("mykey")
	assert.True(t, ok)
	assert.Equal(t, 42, val)
}

func TestRemoveExtension(t *testing.T) {
	s := &Spec{Extensions: map[string]any{"a": 1, "b": 2}}
	s.RemoveExtension("a")
	_, ok := s.GetExtension("a")
	assert.False(t, ok)
	val, ok := s.GetExtension("b")
	assert.True(t, ok)
	assert.Equal(t, 2, val)
}

func TestRemoveExtension_NilSpec(t *testing.T) {
	var s *Spec
	assert.NotPanics(t, func() {
		s.RemoveExtension("key")
	})
}

func TestRemoveExtension_NilExtensions(t *testing.T) {
	s := &Spec{}
	assert.NotPanics(t, func() {
		s.RemoveExtension("key")
	})
}

// =============================================================================
// GetExtensions
// =============================================================================

func TestGetExtensions_NilSpec(t *testing.T) {
	var s *Spec
	assert.Nil(t, s.GetExtensions())
}

func TestGetExtensions_Empty(t *testing.T) {
	s := &Spec{}
	assert.Nil(t, s.GetExtensions())
}

func TestGetExtensions_Populated(t *testing.T) {
	exts := map[string]any{"k1": "v1", "k2": "v2"}
	s := &Spec{Extensions: exts}
	result := s.GetExtensions()
	assert.Equal(t, exts, result)
}

// =============================================================================
// HasExtension
// =============================================================================

func TestHasExtension_NilSpec(t *testing.T) {
	var s *Spec
	assert.False(t, s.HasExtension("key"))
}

func TestHasExtension_NilExtensions(t *testing.T) {
	s := &Spec{}
	assert.False(t, s.HasExtension("key"))
}

func TestHasExtension_Present(t *testing.T) {
	s := &Spec{Extensions: map[string]any{"key": "val"}}
	assert.True(t, s.HasExtension("key"))
}

func TestHasExtension_NotPresent(t *testing.T) {
	s := &Spec{Extensions: map[string]any{"other": "val"}}
	assert.False(t, s.HasExtension("key"))
}

// =============================================================================
// GetExtensionAs
// =============================================================================

type testExtStruct struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestGetExtensionAs_Success(t *testing.T) {
	s := &Spec{
		Extensions: map[string]any{
			"config": map[string]any{"name": "test", "count": float64(5)},
		},
	}
	result, err := GetExtensionAs[testExtStruct](s, "config")
	require.NoError(t, err)
	assert.Equal(t, "test", result.Name)
	assert.Equal(t, 5, result.Count)
}

func TestGetExtensionAs_MissingKey(t *testing.T) {
	s := &Spec{Extensions: map[string]any{}}
	_, err := GetExtensionAs[testExtStruct](s, "missing")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgExtensionNotFound)
}

func TestGetExtensionAs_NilSpec(t *testing.T) {
	var s *Spec
	_, err := GetExtensionAs[testExtStruct](s, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgExtensionNotFound)
}

func TestGetExtensionAs_NilExtensions(t *testing.T) {
	s := &Spec{}
	_, err := GetExtensionAs[testExtStruct](s, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgExtensionNotFound)
}

func TestGetExtensionAs_SimpleType(t *testing.T) {
	s := &Spec{Extensions: map[string]any{"count": float64(42)}}
	result, err := GetExtensionAs[float64](s, "count")
	require.NoError(t, err)
	assert.Equal(t, float64(42), result)
}

// =============================================================================
// GetStandardFields
// =============================================================================

func TestGetStandardFields_NilSpec(t *testing.T) {
	var s *Spec
	assert.Nil(t, s.GetStandardFields())
}

func TestGetStandardFields_Populated(t *testing.T) {
	s := &Spec{
		Name:        "test",
		Description: "desc",
		Inputs:      map[string]*InputDef{"q": {Type: "string"}},
		Outputs:     map[string]*OutputDef{"r": {Type: "string"}},
		Sample:      map[string]any{"q": "hello"},
	}
	m := s.GetStandardFields()
	assert.Equal(t, "test", m[SpecFieldName])
	assert.Equal(t, "desc", m[SpecFieldDescription])
	assert.NotNil(t, m[SpecFieldInputs])
	assert.NotNil(t, m[SpecFieldOutputs])
	assert.NotNil(t, m[SpecFieldSample])
}

func TestGetStandardFields_EmptySpec(t *testing.T) {
	s := &Spec{}
	m := s.GetStandardFields()
	assert.Empty(t, m)
}

// =============================================================================
// GetExonsFields
// =============================================================================

func TestGetExonsFields_NilSpec(t *testing.T) {
	var s *Spec
	assert.Nil(t, s.GetExonsFields())
}

func TestGetExonsFields_Populated(t *testing.T) {
	s := &Spec{
		Type:       DocumentTypeAgent,
		Execution:  &execution.Config{Provider: "openai"},
		Skills:     []SkillRef{{Slug: "s"}},
		Tools:      &ToolsConfig{},
		Context:    map[string]any{"k": "v"},
		Constraints: &ConstraintsConfig{},
		Messages:   []MessageTemplate{{Role: "user"}},
		Credentials: map[string]*CredentialRef{"main": {}},
		Credential: "main",
	}
	m := s.GetExonsFields()
	assert.Equal(t, string(DocumentTypeAgent), m[SpecFieldType])
	assert.NotNil(t, m[SpecFieldExecution])
	assert.NotNil(t, m[SpecFieldSkills])
	assert.NotNil(t, m[SpecFieldTools])
	assert.NotNil(t, m[SpecFieldContext])
	assert.NotNil(t, m[SpecFieldConstraints])
	assert.NotNil(t, m[SpecFieldMessages])
	assert.NotNil(t, m[SpecFieldCredentials])
	assert.Equal(t, "main", m[SpecFieldCredential])
}

func TestGetExonsFields_EmptySpec(t *testing.T) {
	s := &Spec{}
	m := s.GetExonsFields()
	assert.Empty(t, m)
}

// =============================================================================
// No overlap between standard and exons fields
// =============================================================================

func TestFieldSetsNoOverlap(t *testing.T) {
	s := &Spec{
		Name:        "test",
		Description: "desc",
		Type:        DocumentTypeAgent,
		Execution:   &execution.Config{Provider: "openai"},
		Inputs:      map[string]*InputDef{"q": {Type: "string"}},
		Outputs:     map[string]*OutputDef{"r": {Type: "string"}},
		Sample:      map[string]any{"q": "hello"},
		Skills:      []SkillRef{{Slug: "s"}},
		Tools:       &ToolsConfig{},
		Context:     map[string]any{"k": "v"},
		Constraints: &ConstraintsConfig{},
		Messages:    []MessageTemplate{{Role: "user"}},
		Credentials: map[string]*CredentialRef{"main": {}},
		Credential:  "main",
	}

	standard := s.GetStandardFields()
	exonsF := s.GetExonsFields()

	for key := range standard {
		_, exists := exonsF[key]
		assert.False(t, exists, "key %q should not be in both standard and exons fields", key)
	}
}
