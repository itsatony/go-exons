package exons

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNowTag_RendersSeededReferenceTime verifies the {~exons.now~} OUTPUT tag
// renders through the public Engine, seeded via the reserved data key so the
// assertion is deterministic. This is the contract aiv depends on: seed
// ContextKeyReferenceTime, and every now-tag in the render agrees on that instant.
func TestNowTag_RendersSeededReferenceTime(t *testing.T) {
	engine := MustNew()
	ref := time.Date(2026, time.July, 21, 14, 30, 0, 0, time.UTC)
	data := map[string]any{ContextKeyReferenceTime: ref}

	cases := []struct {
		source string
		want   string
	}{
		{`{~exons.now /~}`, "2026-07-21T14:30:00Z"},
		{`Today is {~exons.now format="date" /~}.`, "Today is 2026-07-21."},
		{`{~exons.now format="date-de" /~}`, "21.07.2026"},
		{`{~exons.now format="year" /~}`, "2026"},
		{`{~exons.now format="weekday" /~}`, "Tuesday"},
	}
	for _, c := range cases {
		out, err := engine.Execute(context.Background(), c.source, data)
		require.NoError(t, err, "source=%q", c.source)
		assert.Equal(t, c.want, out, "source=%q", c.source)
	}
}

// TestNowTag_AgreesAcrossMultipleTags verifies two now-tags in one render resolve
// to the same seeded instant (no per-tag time.Now() drift).
func TestNowTag_AgreesAcrossMultipleTags(t *testing.T) {
	engine := MustNew()
	ref := time.Date(2026, time.July, 21, 14, 30, 0, 0, time.UTC)
	out, err := engine.Execute(context.Background(),
		`{~exons.now format="unix" /~}-{~exons.now format="unix" /~}`,
		map[string]any{ContextKeyReferenceTime: ref})
	require.NoError(t, err)
	assert.Equal(t, "1784644200-1784644200", out)
}

// TestNowTag_UnknownFormatHonorsErrorStrategy verifies an unknown format is a
// Resolve error subject to the engine's error strategy: ErrorStrategyRemove (aiv's
// render/resolve setting) drops the tag rather than hard-failing the template.
func TestNowTag_UnknownFormatHonorsErrorStrategy(t *testing.T) {
	engine, err := New(WithErrorStrategy(ErrorStrategyRemove))
	require.NoError(t, err)
	out, xerr := engine.Execute(context.Background(),
		`[{~exons.now format="bogus" /~}]`,
		map[string]any{ContextKeyReferenceTime: time.Now()})
	require.NoError(t, xerr)
	assert.Equal(t, "[]", out)
}
