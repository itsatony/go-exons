package exons

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mediaDoc is the issue's own example (go-exons#20), verbatim in shape.
const mediaDoc = `---
name: media-coworker
description: an agent with default media engines
type: agent
speech:
  provider: openai
  model: gpt-4o-mini-tts
  voice: sage
media:
  image: { provider: openai, model: gpt-image-2 }
  video: { provider: google, model: veo-3 }
  audio: { provider: elevenlabs, model: music-v1 }
realtime: { provider: openai, model: gpt-realtime-2.1, voice: marin }
---
body
`

// TestMediaAndRealtimeDecodeIntoTypedFieldsAndNotIntoExtensions: both halves are
// load-bearing — the value proves the tag, the absence proves the key MOVED out of
// Extensions (which is what changes for every consumer at this bump).
func TestMediaAndRealtimeDecodeIntoTypedFieldsAndNotIntoExtensions(t *testing.T) {
	spec, err := Parse([]byte(mediaDoc))
	require.NoError(t, err)

	require.NotNil(t, spec.Media)
	assert.Equal(t, &MediaModelRef{Provider: "openai", Model: "gpt-image-2"}, spec.Media.Image)
	assert.Equal(t, &MediaModelRef{Provider: "google", Model: "veo-3"}, spec.Media.Video)
	assert.Equal(t, &MediaModelRef{Provider: "elevenlabs", Model: "music-v1"}, spec.Media.Audio)
	assert.Equal(t, &RealtimeConfig{Provider: "openai", Model: "gpt-realtime-2.1", Voice: "marin"}, spec.Realtime)
	require.NotNil(t, spec.Speech, "speech stays the TTS default beside the new blocks")
	assert.Equal(t, "sage", spec.Speech.Voice)

	for _, k := range []string{SpecFieldMedia, SpecFieldRealtime} {
		_, still := spec.Extensions[k]
		assert.False(t, still, "a typed field consumes its key: %s must not also sit in Extensions", k)
	}
	require.NoError(t, spec.ValidateStrict())
}

// TestFullExportRoundTripsMediaAndRealtimeByteStable goes through ExportFull (never a
// direct yaml.Marshal, whose struct tags would hide a missing buildSerializeMap arm)
// and asserts the second export is byte-identical to the first.
func TestFullExportRoundTripsMediaAndRealtimeByteStable(t *testing.T) {
	spec, err := Parse([]byte(mediaDoc))
	require.NoError(t, err)

	first, err := spec.ExportFull()
	require.NoError(t, err)
	reparsed, err := Parse(first)
	require.NoError(t, err)
	assert.Equal(t, spec.Media, reparsed.Media, "media must survive export → re-import")
	assert.Equal(t, spec.Realtime, reparsed.Realtime, "realtime must survive export → re-import")
	assert.Equal(t, spec.Speech, reparsed.Speech)

	second, err := reparsed.ExportFull()
	require.NoError(t, err)
	assert.Equal(t, string(first), string(second), "serialization must be byte-stable")
	assert.Contains(t, string(first), "media:\n    image:\n        provider: openai\n        model: gpt-image-2\n    video:", "sub-blocks serialize in declaration order")
}

// TestPartialMediaBlockRoundTrips: an absent sub-block stays absent (no `video: {}`
// appears out of nothing), and voice is optional everywhere.
func TestPartialMediaBlockRoundTrips(t *testing.T) {
	spec := &Spec{
		Name: "partial", Description: "d", Type: DocumentTypeAgent, Body: "b",
		Media:    &MediaConfig{Image: &MediaModelRef{Provider: "openai", Model: "gpt-image-2"}},
		Realtime: &RealtimeConfig{Provider: "openai", Model: "gpt-realtime-2.1"},
	}
	out, err := spec.ExportFull()
	require.NoError(t, err)
	assert.NotContains(t, string(out), "video")
	assert.NotContains(t, string(out), "voice")

	reparsed, err := Parse(out)
	require.NoError(t, err)
	assert.Equal(t, spec.Media, reparsed.Media)
	assert.Equal(t, spec.Realtime, reparsed.Realtime)
}

// TestAgentSkillExportCarriesNeitherMediaNorRealtime: the Agent-Skills card is a
// closed vocabulary. Paired with a positive assertion so it cannot pass vacuously.
func TestAgentSkillExportCarriesNeitherMediaNorRealtime(t *testing.T) {
	spec, err := Parse([]byte(mediaDoc))
	require.NoError(t, err)
	out, err := spec.ExportAgentSkill()
	require.NoError(t, err)
	assert.Contains(t, string(out), "media-coworker")
	assert.NotContains(t, string(out), "media:")
	assert.NotContains(t, string(out), "realtime")
	assert.NotContains(t, string(out), "gpt-image-2")
}

// TestEngineRefsAreCheckedByValidateStrictNotParse pins WHERE the non-empty rule
// lives. Parse must keep loading a half-filled block (before v0.42.0 it was an inert
// extension, so refusing it on read would lose a stored agent on a library bump);
// ValidateStrict — the writer's gate — refuses it.
func TestEngineRefsAreCheckedByValidateStrictNotParse(t *testing.T) {
	head := "---\nname: a\ndescription: d\ntype: agent\n"
	cases := map[string]struct {
		fm  string
		msg string
	}{
		"image without model":     {"media:\n  image: {provider: openai}\n", ErrMsgEngineModelRequired},
		"video without provider":  {"media:\n  video: {model: veo-3}\n", ErrMsgEngineProviderRequired},
		"audio blank provider":    {"media:\n  audio: {provider: \"  \", model: m}\n", ErrMsgEngineProviderRequired},
		"audio empty mapping":     {"media:\n  audio: {}\n", ErrMsgEngineProviderRequired},
		"realtime without model":  {"realtime: {provider: openai, voice: marin}\n", ErrMsgEngineModelRequired},
		"realtime only a voice":   {"realtime: {voice: marin}\n", ErrMsgEngineProviderRequired},
		"realtime blank model":    {"realtime: {provider: openai, model: \"\\t\"}\n", ErrMsgEngineModelRequired},
		"realtime empty (no key)": {"realtime: {}\n", ErrMsgEngineProviderRequired},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			spec, err := Parse([]byte(head + tc.fm + "---\nbody\n"))
			require.NoError(t, err, "Parse stays tolerant")
			err = spec.ValidateStrict()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.msg)
		})
	}

	t.Run("empty media mapping and absent blocks are valid", func(t *testing.T) {
		spec, err := Parse([]byte(head + "media: {}\n---\nbody\n"))
		require.NoError(t, err)
		require.NoError(t, spec.ValidateStrict())
		assert.True(t, spec.Media.IsEmpty())
		assert.True(t, (*MediaConfig)(nil).IsEmpty())
		assert.NoError(t, (*MediaConfig)(nil).Validate())
		assert.NoError(t, (*RealtimeConfig)(nil).Validate())
	})
}

// TestMediaNotAMappingIsRefusedByParse states the one narrowing: before v0.42.0 a
// scalar `media:` sat in Extensions; now it must decode into the typed field.
func TestMediaNotAMappingIsRefusedByParse(t *testing.T) {
	_, err := Parse([]byte("---\nname: a\ndescription: d\ntype: agent\nmedia: openai\n---\nbody\n"))
	require.Error(t, err)
}

// TestMediaUnknownKeysAreIgnoredByParse: nested unknown keys follow the repo-wide
// rule — the parser ignores them, the schema refuses them (agreement corpus).
func TestMediaUnknownKeysAreIgnoredByParse(t *testing.T) {
	spec, err := Parse([]byte("---\nname: a\ndescription: d\ntype: agent\nmedia:\n  hologram: {provider: x, model: y}\n  image: {provider: openai, model: gpt-image-2, dpi: 300}\n---\nbody\n"))
	require.NoError(t, err)
	assert.Equal(t, &MediaModelRef{Provider: "openai", Model: "gpt-image-2"}, spec.Media.Image)
	require.NoError(t, spec.ValidateStrict())
}

func TestMediaAndRealtimeCloneIsDeep(t *testing.T) {
	spec, err := Parse([]byte(mediaDoc))
	require.NoError(t, err)
	clone := spec.Clone()
	require.Equal(t, spec.Media, clone.Media)
	require.Equal(t, spec.Realtime, clone.Realtime)

	clone.Media.Image.Model = "changed"
	clone.Realtime.Voice = "changed"
	assert.Equal(t, "gpt-image-2", spec.Media.Image.Model, "clone must not alias the original")
	assert.Equal(t, "marin", spec.Realtime.Voice, "clone must not alias the original")
	assert.Nil(t, (*MediaConfig)(nil).Clone())
	assert.Nil(t, (*MediaModelRef)(nil).Clone())
	assert.Nil(t, (*RealtimeConfig)(nil).Clone())
}

// TestPatchSource_MediaAndRealtime: a consumer that edits the source (atlas's agent
// setup) can set, replace and remove each block without re-serializing the document.
func TestPatchSource_MediaAndRealtime(t *testing.T) {
	base := "---\nname: a\ndescription: d\ntype: agent\n# keep me\nexecution:\n  model: m\n---\nbody\n"

	got := mustPatch(t, base,
		SetMediaModel(MediaFieldImage, &MediaModelRef{Provider: "openai", Model: "gpt-image-2"}),
		SetMediaModel(MediaFieldVideo, &MediaModelRef{Provider: "google", Model: "veo-3"}),
		SetRealtime(&RealtimeConfig{Provider: "openai", Model: "gpt-realtime-2.1", Voice: "marin"}),
	)
	assert.Equal(t, "---\nname: a\ndescription: d\ntype: agent\n# keep me\nexecution:\n  model: m\n"+
		"media:\n  image:\n    provider: openai\n    model: gpt-image-2\n  video:\n    provider: google\n    model: veo-3\n"+
		"realtime:\n  provider: openai\n  model: gpt-realtime-2.1\n  voice: marin\n---\nbody\n", got)

	spec, err := Parse([]byte(got))
	require.NoError(t, err)
	assert.Equal(t, "veo-3", spec.Media.Video.Model)
	assert.Equal(t, "marin", spec.Realtime.Voice)

	// Replace one sub-block, leave its sibling alone.
	got2 := mustPatch(t, got, SetMediaModel(MediaFieldImage, &MediaModelRef{Provider: "google", Model: "imagen-4"}))
	spec, err = Parse([]byte(got2))
	require.NoError(t, err)
	assert.Equal(t, "imagen-4", spec.Media.Image.Model)
	assert.Equal(t, "veo-3", spec.Media.Video.Model)

	// Remove everything: media is pruned once emptied, realtime removed whole.
	got3 := mustPatch(t, got,
		SetMediaModel(MediaFieldImage, nil),
		SetMediaModel(MediaFieldVideo, nil),
		SetRealtime(nil),
	)
	assert.Equal(t, base, got3)

	// Removing an absent block is a no-op.
	assert.Equal(t, base, mustPatch(t, base, SetRealtime(nil), SetMediaModel(MediaFieldAudio, nil)))
}

func TestPatchSource_MediaRefusals(t *testing.T) {
	agentDoc := "---\nname: a\ndescription: d\ntype: agent\n---\nbody\n"
	cases := map[string]struct {
		edit   SourceEdit
		reason error
	}{
		"unknown media kind":        {SetMediaModel("hologram", &MediaModelRef{Provider: "x", Model: "y"}), ErrPatchEditInvalid},
		"media ref without model":   {SetMediaModel(MediaFieldImage, &MediaModelRef{Provider: "openai"}), ErrPatchResultInvalid},
		"realtime without provider": {SetRealtime(&RealtimeConfig{Model: "gpt-realtime-2.1"}), ErrPatchResultInvalid},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			out, err := PatchSource([]byte(agentDoc), tc.edit)
			require.Error(t, err)
			assert.Nil(t, out)
			assert.True(t, errors.Is(err, ErrPatchRefused))
			assert.True(t, errors.Is(err, tc.reason), "want %v: %v", tc.reason, err)
		})
	}
}
