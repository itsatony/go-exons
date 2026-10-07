package exons

// SpeechConfig declares how a document should SOUND when it is read aloud.
//
// WHY IT IS TOP-LEVEL AND NOT `execution.audio`. `execution.*` parameterises the
// call that produces the document's OUTPUT — `execution.audio` is for an agent
// whose output IS audio (modality `audio_speech`). This block answers a different
// question: given text this document produced by any means, in what voice should a
// consumer read it back? An agent that is a coworker you talk to needs the second
// answer, and `ExecutionConfig` is `additionalProperties: false` around exactly one
// provider/model pair — the LLM's — so a voice named there has no engine that
// defines it.
//
// ⚠ `execution.audio.voice` STILL EXISTS AND STILL MEANS WHAT IT MEANT. Nothing
// here supersedes it and there is deliberately NO fallback between the two: a
// consumer reading a document aloud reads THIS block, and a consumer generating
// audio as an execution result reads that one. A silent fallback would make one
// key's meaning depend on which consumer happened to read it.
//
// DECLARATION-ONLY: go-exons stores declarations; a consumer resolves them. The
// bounds that are not checked in Go (speed 0.25-4.0, the output_format vocabulary)
// are stated in schema/exons.schema.json, the published contract for editors and CI.
//
// ⚠ WHERE THE NON-EMPTY RULE LIVES (v0.43.0, vAudience/atlas#849). A present block
// must name a provider AND a model — without a model a voice has no namespace.
// SpeechConfig.Validate refuses it, and only Spec.ValidateStrict (the writer's gate,
// which PatchSource runs on its RESULT) calls it; Parse and Spec.Validate do NOT.
// Before v0.27.0 a `speech:` block landed inertly in Spec.Extensions and until
// v0.43.0 nothing checked it, so refusing a half-filled one on READ would lose a
// stored document on nothing more than a library bump. Same rule as the media and
// realtime blocks (v0.42.0) and the tool allow-lists (v0.40.0): Parse tolerant,
// writers strict. Voice, voice_id and everything else stay optional.
type SpeechConfig struct {
	// Provider is the TTS vendor (e.g. "openai", "elevenlabs"). It is NOT drawn
	// from the ExecutionConfig provider vocabulary — that set names LLM providers
	// and carries no TTS-only vendor, so reusing it would refuse a working value.
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty"`

	// Model is the TTS model (e.g. "gpt-4o-mini-tts"). Without it, Voice has no
	// namespace: "sage" means nothing until something says which engine defines it.
	Model string `yaml:"model,omitempty" json:"model,omitempty"`

	// Voice is the voice name in the provider's namespace.
	Voice string `yaml:"voice,omitempty" json:"voice,omitempty"`

	// VoiceID is a provider-specific opaque identifier, for vendors whose voice
	// names are not stable addresses.
	VoiceID string `yaml:"voice_id,omitempty" json:"voice_id,omitempty"`

	// Instructions is free-text delivery steering ("warm, unhurried, never
	// chipper"), honoured by steerable TTS models and ignored by the rest. It is
	// the whole reason to prefer such a model, and it is the one field here that
	// cannot be expressed as a parameter.
	Instructions string `yaml:"instructions,omitempty" json:"instructions,omitempty"`

	// Speed is playback speed. The 0.25-4.0 bound lives in the schema; see the
	// type comment for why it is not enforced here.
	Speed float64 `yaml:"speed,omitempty" json:"speed,omitempty"`

	// OutputFormat is the container/codec (mp3, opus, aac, flac, wav, pcm).
	OutputFormat string `yaml:"output_format,omitempty" json:"output_format,omitempty"`

	// Language is a language code (e.g. "de").
	Language string `yaml:"language,omitempty" json:"language,omitempty"`

	// Region is the jurisdiction or provider region the audio may be produced in.
	//
	// ⚠ FREE-FORM ON PURPOSE. Downstream, this value is provider-interpreted:
	// "eu" and "us" for one vendor, "eu-central-1" or "europe-west4" for another.
	// An enum would refuse values that work today, and this field exists precisely
	// so a document can state a residency constraint it must be able to state.
	Region string `yaml:"region,omitempty" json:"region,omitempty"`
}

// Clone creates a deep copy of the SpeechConfig. Every field is a scalar, so the
// struct copy IS the deep copy — but the method exists so Spec.Clone has the same
// nil-safe shape here as for every other optional block, and so a field added
// later has an obvious home.
func (sc *SpeechConfig) Clone() *SpeechConfig {
	if sc == nil {
		return nil
	}
	clone := *sc
	return &clone
}

// Validate refuses a present speech block that does not name both a provider and
// a model (non-blank). A nil SpeechConfig is valid. Called by Spec.ValidateStrict,
// never by Parse — see the type comment. v0.43.0.
func (sc *SpeechConfig) Validate() error {
	if sc == nil {
		return nil
	}
	return validateEngineRef(SpecFieldSpeech, sc.Provider, sc.Model)
}
