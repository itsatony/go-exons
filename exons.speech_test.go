package exons

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const speechDoc = `---
name: read-me-aloud
description: an agent with a voice
type: agent
speech:
  provider: openai
  model: gpt-4o-mini-tts
  voice: sage
  instructions: "Warm, unhurried, faintly amused. Never chipper."
  region: eu
  speed: 1.0
  output_format: opus
---
body
`

// TestSpeechDecodesIntoTheTypedFieldAndNotIntoExtensions is the positive half of
// the v0.27.0 change: `speech:` stops being an unknown key.
//
// ⚠ BOTH ASSERTIONS ARE LOAD-BEARING AND THEY FAIL FOR DIFFERENT REASONS. Without
// the typed field, Speech is nil; without the `yaml:"speech"` tag actually
// matching, the key would ALSO still sit in Extensions while some other field
// happened to be populated. Asserting the value alone would not prove the key
// moved, which is the thing that changes for every consumer.
func TestSpeechDecodesIntoTheTypedFieldAndNotIntoExtensions(t *testing.T) {
	spec, err := Parse([]byte(speechDoc))
	require.NoError(t, err)
	require.NotNil(t, spec.Speech, "speech: must decode into the typed field")

	assert.Equal(t, "openai", spec.Speech.Provider)
	assert.Equal(t, "gpt-4o-mini-tts", spec.Speech.Model)
	assert.Equal(t, "sage", spec.Speech.Voice)
	assert.Equal(t, "Warm, unhurried, faintly amused. Never chipper.", spec.Speech.Instructions)
	assert.Equal(t, "eu", spec.Speech.Region)
	assert.InDelta(t, 1.0, spec.Speech.Speed, 0.0001)
	assert.Equal(t, "opus", spec.Speech.OutputFormat)

	_, stillInExtensions := spec.Extensions[SpecFieldSpeech]
	assert.False(t, stillInExtensions,
		"a typed field consumes its key: speech must NOT also appear in Extensions")
}

// TestTranscriptionStaysInExtensions pins the deliberate asymmetry with `speech:`.
//
// `transcription:` is a live downstream convention read as
// spec.Extensions["transcription"] (go-vaibstract's ParseExonsTranscriptionSpec).
// Typing it here would consume the key exactly as the test above proves `speech:`
// is now consumed — and every transcription-configured document would silently
// lose its provider, model and region at the consumer's next pin bump, with
// nothing failing. The schema declares the block (so editors and CI can check it);
// the Go struct deliberately does not.
//
// ⚠ THE REFLECTION ARM IS THE ONE THAT CATCHES THE MISTAKE. The map assertion
// alone would keep passing if someone added a `Transcription` field under a
// different yaml tag; the field check refuses the field itself, which is the thing
// the reason forbids.
func TestTranscriptionStaysInExtensions(t *testing.T) {
	const doc = `---
name: listen-to-me
description: an agent that listens
type: agent
transcription:
  provider: speechmatics
  model: ursa-2
  region: eu
---
body
`
	spec, err := Parse([]byte(doc))
	require.NoError(t, err)

	raw, ok := spec.Extensions["transcription"]
	require.True(t, ok, "transcription: must remain an Extensions key")
	block, ok := raw.(map[string]any)
	require.True(t, ok, "the transcription block must decode as a map")
	assert.Equal(t, "speechmatics", block["provider"])
	assert.Equal(t, "ursa-2", block["model"])

	_, typed := reflect.TypeOf(Spec{}).FieldByName("Transcription")
	assert.False(t, typed,
		"Spec must NOT gain a Transcription field — it would empty Extensions[\"transcription\"] "+
			"for every consumer that reads it there; see the Speech field's comment")
}

// TestFullExportRoundTripsSpeechAndRequirements is the guard for the class of bug
// this cycle both introduced-and-avoided and found already present: a typed field
// that no export emits is a value that survives Parse and dies at Serialize,
// because the typed field consumed the key the Extensions passthrough would
// otherwise have carried.
//
// ⚠ IT MUST GO THROUGH ExportFull, NOT yaml.Marshal(spec) — the struct tag does
// the work in a direct marshal, so a direct-marshal test passes against a
// buildSerializeMap that emits nothing. That is exactly how the requirements gap
// survived until v0.27.0.
func TestFullExportRoundTripsSpeechAndRequirements(t *testing.T) {
	original := &Spec{
		Name:        "round-trip",
		Description: "a document that must survive a round trip",
		Type:        DocumentTypeAgent,
		Speech:      &SpeechConfig{Provider: "openai", Model: "gpt-4o-mini-tts", Voice: "sage", Region: "eu"},
		Requirements: &SpecRequirements{
			MCP: []MCPRequirement{{Capability: "dns-management", CredentialRef: "dns_api_key", Scope: "org"}},
		},
		Body: "body",
	}

	out, err := original.ExportFull()
	require.NoError(t, err)

	reparsed, err := Parse(out)
	require.NoError(t, err)

	require.NotNil(t, reparsed.Speech, "speech must survive an export → re-import round trip")
	assert.Equal(t, "sage", reparsed.Speech.Voice)
	assert.Equal(t, "eu", reparsed.Speech.Region)

	require.NotNil(t, reparsed.Requirements, "requirements must survive an export → re-import round trip")
	require.Len(t, reparsed.Requirements.MCP, 1)
	assert.Equal(t, "dns-management", reparsed.Requirements.MCP[0].Capability)
	assert.Equal(t, "dns_api_key", reparsed.Requirements.MCP[0].CredentialRef)
}

// TestAgentSkillExportCarriesNeitherSpeechNorRequirements is the negative half.
// The Agent-Skills card is a closed portable vocabulary; an extra key there is a
// conformance failure for the consumers that validate it. Both blocks sit under
// IncludeMetadata, which AgentSkillsExportOptions sets false.
//
// ⚠ PAIRED WITH A POSITIVE ASSERTION ON PURPOSE: an absence check alone passes
// vacuously against an export that produced nothing at all.
func TestAgentSkillExportCarriesNeitherSpeechNorRequirements(t *testing.T) {
	spec := &Spec{
		Name:         "card",
		Description:  "a card",
		Type:         DocumentTypeAgent,
		Speech:       &SpeechConfig{Voice: "sage"},
		Requirements: &SpecRequirements{MCP: []MCPRequirement{{Capability: "dns-management"}}},
		Body:         "body",
	}

	out, err := spec.ExportAgentSkill()
	require.NoError(t, err)
	card := string(out)

	assert.Contains(t, card, "card", "the card must actually carry the definition")
	assert.NotContains(t, card, SpecFieldSpeech)
	assert.NotContains(t, card, SpecFieldRequirements)
}

// TestShippedExampleDocumentParses guards the one artifact this repo publishes as
// "the full frontmatter surface" (README: examples/dns-specialist.exons).
//
// ⚠ NOTHING CHECKED IT BEFORE v0.27.0. It is a reference document readers copy
// from, it is edited by hand whenever the surface grows — this cycle added a
// `speech:` block to it — and a typo in it fails nowhere: no test parsed it, and
// the `examples/` tree is excluded from `fmt-check` for its own reasons. The
// cheapest honest guard is to parse it.
func TestShippedExampleDocumentParses(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("examples", "dns-specialist.exons"))
	require.NoError(t, err, "the reference document must exist where the README says it does")

	spec, err := Parse(raw)
	require.NoError(t, err, "the reference document must parse")

	// Asserting a value, not just the absence of an error: a document that parsed
	// but decoded its blocks somewhere unexpected would still satisfy NoError.
	require.NotNil(t, spec.Speech, "the reference document declares a speech block")
	assert.Equal(t, "sage", spec.Speech.Voice)
	assert.NotNil(t, spec.Safety, "…and still declares the blocks it declared before")
}
