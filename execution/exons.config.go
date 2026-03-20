// Package execution provides LLM execution configuration with multi-provider
// serialization support (OpenAI, Anthropic, Gemini, vLLM, Mistral, Cohere).
package execution

// Config holds LLM execution parameters parsed from .exons frontmatter.
type Config struct {
	Provider    string   `yaml:"provider,omitempty" json:"provider,omitempty"`
	Model       string   `yaml:"model,omitempty" json:"model,omitempty"`
	Temperature *float64 `yaml:"temperature,omitempty" json:"temperature,omitempty"`
	TopP        *float64 `yaml:"top_p,omitempty" json:"top_p,omitempty"`
	MaxTokens   *int     `yaml:"max_tokens,omitempty" json:"max_tokens,omitempty"`
	TopK        *int     `yaml:"top_k,omitempty" json:"top_k,omitempty"`
	Stop        []string `yaml:"stop,omitempty" json:"stop,omitempty"`
	Modality    string   `yaml:"modality,omitempty" json:"modality,omitempty"`
}
