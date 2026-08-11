package exons

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// StripMessageMarkers is the inverse of the {~exons.message~} framing for a caller that has
// one string to fill rather than a message array. The framing contains NUL bytes, and a
// consumer that carries them onward hits things that refuse them — a postgres `jsonb` column
// refuses  outright (SQLSTATE 22P05), which is how this was found: a prompt template
// using the message tag failed a whole chat send with an opaque database error.
//
// ⚠ EVERY CASE HERE ASSERTS THE ABSENCE OF A NUL EXPLICITLY, not merely the expected prose.
// A future marker format that changed shape would still leave the prose looking right while
// re-introducing the byte that actually breaks the consumer.

func TestStripMessageMarkers_RealRenderedOutput(t *testing.T) {
	// Rendered live from a real template (atlas: @org-org-admin-tester/atlas-input-test).
	in := "Repeat what the input is.\n\n\x00MSG_START:user:false:\n\nWhat you should do: draw\n\nQuery?\n\x00MSG_END\x00"
	out := StripMessageMarkers(in)

	assert.NotContains(t, out, "\x00", "the NUL is the byte that breaks the consumer")
	assert.NotContains(t, out, "MSG_START")
	assert.NotContains(t, out, "MSG_END")
	// ⚠ BOTH halves survive: the prose BEFORE the marker (which
	// ExtractMessagesFromOutput would discard) and the message content itself.
	assert.Contains(t, out, "Repeat what the input is.")
	assert.Contains(t, out, "What you should do: draw")
	assert.Contains(t, out, "Query?")
}

func TestStripMessageMarkers_NoMarkersIsUnchanged(t *testing.T) {
	// The overwhelmingly common case. Byte-identical, so adding this call to a render path
	// cannot alter what a marker-free template produces.
	for _, in := range []string{"", "plain prose", "a colon: and a MSG_START word in prose"} {
		assert.Equal(t, in, StripMessageMarkers(in))
	}
}

func TestStripMessageMarkers_MultipleMessagesFlattenAndKeepOrder(t *testing.T) {
	in := "\x00MSG_START:system:true:be brief\x00MSG_END\x00between\x00MSG_START:user:false:hello\x00MSG_END\x00"
	out := StripMessageMarkers(in)

	assert.NotContains(t, out, "\x00")
	assert.Equal(t, "be briefbetweenhello", out, "flattened in source order, roles dropped")
	// ⚠ The flattening is the documented limitation, asserted so it is a decision rather
	// than a surprise: a caller that needs the roles must use ExtractMessagesFromOutput.
	assert.NotContains(t, out, "system")
	assert.NotContains(t, out, "true")
}

func TestStripMessageMarkers_MalformedFramingStillYieldsNoNUL(t *testing.T) {
	// A truncated header, a lone end marker, and a start marker with no end. None of these
	// can be produced by the resolver, and all three are the shapes where a naive
	// "cut between the markers" implementation leaks the byte it exists to remove.
	for name, in := range map[string]string{
		"truncated header": "\x00MSG_START:user",
		"lone end marker":  "prose\x00MSG_END\x00",
		"start, no end":    "\x00MSG_START:user:false:dangling content",
		"nul in prose":     "prose with a stray \x00 byte",
	} {
		out := StripMessageMarkers(in)
		if name == "nul in prose" {
			// ⚠ HONESTLY NOT COVERED, AND PINNED AS SUCH. A bare NUL outside any marker is
			// not framing; the resolver sanitizes content, so this can only come from the
			// caller's own data, and silently deleting bytes from it would be the wrong
			// liberty for this function to take.
			assert.Contains(t, out, "\x00", "a NUL that is not framing is left alone")
			continue
		}
		assert.NotContainsf(t, out, "\x00", "%s must not leak a NUL", name)
		assert.NotContainsf(t, out, "MSG_", "%s must not leak marker text", name)
	}
}

// TestStripMessageMarkers_ThroughARealTemplate is the seam test: it runs an ACTUAL message
// tag through the engine rather than asserting against a hand-written marker string.
//
// ⚠ A HAND-WRITTEN FIXTURE IS EXACTLY HOW THIS FUNCTION WOULD SURVIVE THE MARKER FORMAT
// CHANGING — it would keep passing against a shape the resolver no longer emits.
func TestStripMessageMarkers_ThroughARealTemplate(t *testing.T) {
	eng, err := New()
	require.NoError(t, err)
	tmpl, err := eng.Parse(`before {~exons.message role="user"~}hello {~exons.var name="who" /~}{~/exons.message~} after`)
	require.NoError(t, err)

	raw, err := tmpl.Execute(context.Background(), map[string]any{"who": "world"})
	require.NoError(t, err)
	require.True(t, strings.Contains(raw, "\x00"), "the fixture is only meaningful if the engine really framed it")

	out := StripMessageMarkers(raw)
	assert.NotContains(t, out, "\x00")
	assert.Contains(t, out, "hello world")
	assert.Contains(t, out, "before")
	assert.Contains(t, out, "after")
}
