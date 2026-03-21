package exons

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithEnvDisabled(t *testing.T) {
	engine, err := New(WithEnvDisabled())
	require.NoError(t, err)

	source := `{~exons.env name="HOME" /~}`
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	_, err = tmpl.Execute(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestWithEnvDenylist(t *testing.T) {
	t.Run("custom denylist blocks matching vars", func(t *testing.T) {
		engine, err := New(WithEnvDenylist([]string{"HOME"}))
		require.NoError(t, err)

		source := `{~exons.env name="HOME" /~}`
		tmpl, err := engine.Parse(source)
		require.NoError(t, err)

		_, err = tmpl.Execute(context.Background(), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "denied")
	})

	t.Run("nil denylist allows all", func(t *testing.T) {
		engine, err := New(WithEnvDenylist(nil))
		require.NoError(t, err)

		// PATH is always set and would normally be blocked by default patterns
		t.Setenv("EXONS_TEST_SECRET_KEY", "hunter2")
		source := `{~exons.env name="EXONS_TEST_SECRET_KEY" /~}`
		tmpl, err := engine.Parse(source)
		require.NoError(t, err)

		result, err := tmpl.Execute(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, "hunter2", result)
	})
}

func TestWithEnvAllowlist(t *testing.T) {
	engine, err := New(
		WithEnvDenylist(nil), // clear default deny
		WithEnvAllowlist([]string{"EXONS_TEST_*"}),
	)
	require.NoError(t, err)

	t.Run("matching var allowed", func(t *testing.T) {
		t.Setenv("EXONS_TEST_VALUE", "hello")
		source := `{~exons.env name="EXONS_TEST_VALUE" /~}`
		tmpl, err := engine.Parse(source)
		require.NoError(t, err)

		result, err := tmpl.Execute(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, "hello", result)
	})

	t.Run("non-matching var blocked", func(t *testing.T) {
		t.Setenv("OTHER_VALUE", "secret")
		source := `{~exons.env name="OTHER_VALUE" /~}`
		tmpl, err := engine.Parse(source)
		require.NoError(t, err)

		_, err = tmpl.Execute(context.Background(), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "allowlist")
	})
}

func TestWithMaxOutputSize(t *testing.T) {
	t.Run("output exceeding limit returns error", func(t *testing.T) {
		engine, err := New(WithMaxOutputSize(50))
		require.NoError(t, err)

		// Create template that produces > 50 bytes
		source := strings.Repeat("x", 60)
		tmpl, err := engine.Parse(source)
		require.NoError(t, err)

		_, err = tmpl.Execute(context.Background(), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maximum size")
	})

	t.Run("output under limit succeeds", func(t *testing.T) {
		engine, err := New(WithMaxOutputSize(100))
		require.NoError(t, err)

		source := strings.Repeat("x", 50)
		tmpl, err := engine.Parse(source)
		require.NoError(t, err)

		result, err := tmpl.Execute(context.Background(), nil)
		require.NoError(t, err)
		assert.Len(t, result, 50)
	})

	t.Run("zero means unlimited", func(t *testing.T) {
		engine, err := New(WithMaxOutputSize(0))
		require.NoError(t, err)

		source := strings.Repeat("x", 10000)
		tmpl, err := engine.Parse(source)
		require.NoError(t, err)

		result, err := tmpl.Execute(context.Background(), nil)
		require.NoError(t, err)
		assert.Len(t, result, 10000)
	})
}

func TestDefaultEnvDenyPatterns_Immutability(t *testing.T) {
	first := DefaultEnvDenyPatterns()
	second := DefaultEnvDenyPatterns()

	// Mutating one copy should not affect the other
	first[0] = "MUTATED"
	assert.NotEqual(t, first[0], second[0])
	assert.Equal(t, "*_KEY", second[0])
}

func TestDefaultEnvDenyPatterns_Content(t *testing.T) {
	patterns := DefaultEnvDenyPatterns()
	assert.True(t, len(patterns) > 0)

	// Verify key patterns are present
	patternSet := make(map[string]bool)
	for _, p := range patterns {
		patternSet[p] = true
	}
	assert.True(t, patternSet["*_KEY"])
	assert.True(t, patternSet["*_SECRET"])
	assert.True(t, patternSet["*_TOKEN"])
	assert.True(t, patternSet["*_PASSWORD"])
	assert.True(t, patternSet["*_PASSPHRASE"])
	assert.True(t, patternSet["*_DSN"])
}
