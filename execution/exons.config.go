package execution

// Config holds LLM execution parameters parsed from .exons frontmatter.
// Config is safe for concurrent reads. Callers must not modify the config after sharing it;
// use Clone() to create an independent copy if mutation is needed.
type Config struct {
	// Provider identifier (e.g., "openai", "anthropic", "gemini", "vllm")
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty"`
	// Model identifier (e.g., "gpt-4", "claude-sonnet-4-5")
	Model string `yaml:"model,omitempty" json:"model,omitempty"`

	// Common inference parameters
	Temperature   *float64 `yaml:"temperature,omitempty" json:"temperature,omitempty"`
	MaxTokens     *int     `yaml:"max_tokens,omitempty" json:"max_tokens,omitempty"`
	TopP          *float64 `yaml:"top_p,omitempty" json:"top_p,omitempty"`
	TopK          *int     `yaml:"top_k,omitempty" json:"top_k,omitempty"`
	StopSequences []string `yaml:"stop_sequences,omitempty" json:"stop_sequences,omitempty"`

	// Extended inference parameters
	MinP              *float64           `yaml:"min_p,omitempty" json:"min_p,omitempty"`
	RepetitionPenalty *float64           `yaml:"repetition_penalty,omitempty" json:"repetition_penalty,omitempty"`
	Seed              *int               `yaml:"seed,omitempty" json:"seed,omitempty"`
	Logprobs          *int               `yaml:"logprobs,omitempty" json:"logprobs,omitempty"`
	StopTokenIDs      []int              `yaml:"stop_token_ids,omitempty" json:"stop_token_ids,omitempty"`
	LogitBias         map[string]float64 `yaml:"logit_bias,omitempty" json:"logit_bias,omitempty"`

	// LLM parameter alignment (LiteLLM/OpenRouter)
	FrequencyPenalty    *float64 `yaml:"frequency_penalty,omitempty" json:"frequency_penalty,omitempty"`
	PresencePenalty     *float64 `yaml:"presence_penalty,omitempty" json:"presence_penalty,omitempty"`
	N                   *int     `yaml:"n,omitempty" json:"n,omitempty"`
	MaxCompletionTokens *int     `yaml:"max_completion_tokens,omitempty" json:"max_completion_tokens,omitempty"`
	ReasoningEffort     string   `yaml:"reasoning_effort,omitempty" json:"reasoning_effort,omitempty"`
	TopA                *float64 `yaml:"top_a,omitempty" json:"top_a,omitempty"`
	User                string   `yaml:"user,omitempty" json:"user,omitempty"`
	ServiceTier         string   `yaml:"service_tier,omitempty" json:"service_tier,omitempty"`
	Store               *bool    `yaml:"store,omitempty" json:"store,omitempty"`

	// Extended thinking configuration (Anthropic)
	Thinking *ThinkingConfig `yaml:"thinking,omitempty" json:"thinking,omitempty"`

	// Structured output configuration
	ResponseFormat *ResponseFormat `yaml:"response_format,omitempty" json:"response_format,omitempty"`
	GuidedDecoding *GuidedDecoding `yaml:"guided_decoding,omitempty" json:"guided_decoding,omitempty"`

	// Modality — execution intent signal (e.g., "text", "image", "audio_speech", "embedding")
	Modality string `yaml:"modality,omitempty" json:"modality,omitempty"`

	// Media generation configs
	Image     *ImageConfig     `yaml:"image,omitempty" json:"image,omitempty"`
	Audio     *AudioConfig     `yaml:"audio,omitempty" json:"audio,omitempty"`
	Embedding *EmbeddingConfig `yaml:"embedding,omitempty" json:"embedding,omitempty"`

	// Execution mode configs
	Streaming *StreamingConfig `yaml:"streaming,omitempty" json:"streaming,omitempty"`
	Async     *AsyncConfig     `yaml:"async,omitempty" json:"async,omitempty"`

	// Provider-specific options (passthrough)
	ProviderOptions map[string]any `yaml:"provider_options,omitempty" json:"provider_options,omitempty"`
}
