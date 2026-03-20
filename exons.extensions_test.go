package exons

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

