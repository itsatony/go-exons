package exons

import "strings"

// MediaConfig declares the document's DEFAULT media-generation models — the model a
// consumer uses when a media call (image, video, audio/music) names none itself.
//
// WHY IT IS TOP-LEVEL AND NOT `execution.image` / `execution.audio`. Those blocks
// parameterise the call that produces the document's OUTPUT (size, quality, speed,
// format) and carry no model: `ExecutionConfig` is `additionalProperties: false`
// around exactly one provider/model pair — the LLM's. This block answers a
// different question: when an agent's tools generate an image, a video or a piece
// of audio, which engine should they reach for by default? (go-exons#20,
// vAudience/atlas#849.)
//
// DECLARATION-ONLY. go-exons stores the declaration; a consumer resolves it, and a
// default is never exempt from the consumer's own residency or governance checks.
//
// ⚠ WHERE THE NON-EMPTY RULE LIVES. A present sub-block must name a provider AND a
// model — refused by MediaConfig.Validate, which Spec.ValidateStrict (the writer's gate)
// runs; never by Parse or Spec.Validate.
// Before v0.42.0 a `media:` block landed inertly in Spec.Extensions, so refusing a
// half-filled one in Parse would be a NARROWING shipped in a minor release: a
// reader would lose an agent that loads today. Same rule as v0.40.0's allow-lists
// (Parse tolerant, writers strict); the schema states the rule for editors and CI.
type MediaConfig struct {
	// Image is the default image-generation model.
	Image *MediaModelRef `yaml:"image,omitempty" json:"image,omitempty"`
	// Video is the default video-generation model.
	Video *MediaModelRef `yaml:"video,omitempty" json:"video,omitempty"`
	// Audio is the default audio/music-generation model. Not text-to-speech:
	// reading text aloud is `speech:`.
	Audio *MediaModelRef `yaml:"audio,omitempty" json:"audio,omitempty"`
}

// MediaModelRef names one engine: a provider and a model in that provider's
// namespace. Neither is drawn from the ExecutionConfig provider vocabulary — that
// set names LLM providers and carries no media-only vendor (elevenlabs, runway…),
// so reusing it would refuse a working value. Free-form on purpose, like
// SpeechConfig.Provider.
type MediaModelRef struct {
	// Provider is the vendor (e.g. "openai", "google", "elevenlabs").
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty"`
	// Model is the model in the provider's namespace (e.g. "gpt-image-2").
	Model string `yaml:"model,omitempty" json:"model,omitempty"`
	// Voice is an optional voice name in the provider's namespace, for an engine
	// that takes one (e.g. an audio model that speaks or sings).
	Voice string `yaml:"voice,omitempty" json:"voice,omitempty"`
}

// RealtimeConfig declares the document's default REALTIME (live voice call)
// engine: a speech-to-speech model and the voice it should answer in.
//
// It is not `speech:` — that block voices text the document already produced
// (text-to-speech); this one is the model you TALK to, whose voice lives in the
// realtime engine's own namespace. There is deliberately no fallback between the
// two: "marin" in one engine and "sage" in another are different namespaces, and a
// silent fallback would hand one engine a voice the other defined.
//
// Same non-empty rule and same placement as MediaConfig: a present block names a
// provider and a model; Validate refuses it otherwise, Parse does not.
type RealtimeConfig struct {
	// Provider is the realtime vendor (e.g. "openai").
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty"`
	// Model is the realtime model (e.g. "gpt-realtime-2.1").
	Model string `yaml:"model,omitempty" json:"model,omitempty"`
	// Voice is the voice name in the provider's namespace (e.g. "marin"). Optional:
	// absent means the consumer's default voice for that model.
	Voice string `yaml:"voice,omitempty" json:"voice,omitempty"`
}

// Validate refuses a present sub-block that does not name both a provider and a
// model. A nil MediaConfig, or an empty one (`media: {}`), is valid.
func (mc *MediaConfig) Validate() error {
	if mc == nil {
		return nil
	}
	if err := mc.Image.validate(MediaFieldImage); err != nil {
		return err
	}
	if err := mc.Video.validate(MediaFieldVideo); err != nil {
		return err
	}
	return mc.Audio.validate(MediaFieldAudio)
}

// IsEmpty reports whether no sub-block is set.
func (mc *MediaConfig) IsEmpty() bool {
	return mc == nil || (mc.Image == nil && mc.Video == nil && mc.Audio == nil)
}

// validate checks one sub-block; kind names it (image/video/audio) in the error.
func (r *MediaModelRef) validate(kind string) error {
	if r == nil {
		return nil
	}
	return validateEngineRef(SpecFieldMedia+"."+kind, r.Provider, r.Model)
}

// Validate refuses a present realtime block that does not name both a provider
// and a model.
func (rc *RealtimeConfig) Validate() error {
	if rc == nil {
		return nil
	}
	return validateEngineRef(SpecFieldRealtime, rc.Provider, rc.Model)
}

// validateEngineRef is the one rule both blocks share: provider and model are both
// non-blank. path is the dotted frontmatter key, carried as the error's value.
func validateEngineRef(path, provider, model string) error {
	if strings.TrimSpace(provider) == "" {
		return NewMetadataValidationError(ErrMsgEngineProviderRequired, path)
	}
	if strings.TrimSpace(model) == "" {
		return NewMetadataValidationError(ErrMsgEngineModelRequired, path)
	}
	return nil
}

// Clone creates a deep copy of the MediaConfig.
func (mc *MediaConfig) Clone() *MediaConfig {
	if mc == nil {
		return nil
	}
	return &MediaConfig{
		Image: mc.Image.Clone(),
		Video: mc.Video.Clone(),
		Audio: mc.Audio.Clone(),
	}
}

// Clone creates a deep copy of the MediaModelRef (every field is a scalar).
func (r *MediaModelRef) Clone() *MediaModelRef {
	if r == nil {
		return nil
	}
	clone := *r
	return &clone
}

// Clone creates a deep copy of the RealtimeConfig (every field is a scalar).
func (rc *RealtimeConfig) Clone() *RealtimeConfig {
	if rc == nil {
		return nil
	}
	clone := *rc
	return &clone
}
