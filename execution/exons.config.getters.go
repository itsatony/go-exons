package execution

// GetProvider returns the provider or empty string.
func (e *Config) GetProvider() string {
	if e == nil {
		return ""
	}
	return e.Provider
}

// GetModel returns the model name or empty string.
func (e *Config) GetModel() string {
	if e == nil {
		return ""
	}
	return e.Model
}

// GetTemperature returns the temperature and whether it was set.
func (e *Config) GetTemperature() (float64, bool) {
	if e == nil || e.Temperature == nil {
		return 0, false
	}
	return *e.Temperature, true
}

// GetMaxTokens returns max_tokens and whether it was set.
func (e *Config) GetMaxTokens() (int, bool) {
	if e == nil || e.MaxTokens == nil {
		return 0, false
	}
	return *e.MaxTokens, true
}

// GetTopP returns top_p and whether it was set.
func (e *Config) GetTopP() (float64, bool) {
	if e == nil || e.TopP == nil {
		return 0, false
	}
	return *e.TopP, true
}

// GetTopK returns top_k and whether it was set.
func (e *Config) GetTopK() (int, bool) {
	if e == nil || e.TopK == nil {
		return 0, false
	}
	return *e.TopK, true
}

// GetStopSequences returns stop sequences or nil.
func (e *Config) GetStopSequences() []string {
	if e == nil {
		return nil
	}
	return e.StopSequences
}

// GetThinking returns the thinking config or nil.
func (e *Config) GetThinking() *ThinkingConfig {
	if e == nil {
		return nil
	}
	return e.Thinking
}

// GetResponseFormat returns the response format or nil.
func (e *Config) GetResponseFormat() *ResponseFormat {
	if e == nil {
		return nil
	}
	return e.ResponseFormat
}

// GetGuidedDecoding returns the guided decoding config or nil.
func (e *Config) GetGuidedDecoding() *GuidedDecoding {
	if e == nil {
		return nil
	}
	return e.GuidedDecoding
}

// HasThinking returns true if thinking is configured and enabled.
func (e *Config) HasThinking() bool {
	return e != nil && e.Thinking != nil && e.Thinking.Enabled
}

// HasResponseFormat returns true if response format is configured.
func (e *Config) HasResponseFormat() bool {
	return e != nil && e.ResponseFormat != nil
}

// HasGuidedDecoding returns true if guided decoding is configured.
func (e *Config) HasGuidedDecoding() bool {
	return e != nil && e.GuidedDecoding != nil
}

// GetMinP returns min_p and whether it was set.
func (e *Config) GetMinP() (float64, bool) {
	if e == nil || e.MinP == nil {
		return 0, false
	}
	return *e.MinP, true
}

// HasMinP returns true if min_p is configured.
func (e *Config) HasMinP() bool {
	return e != nil && e.MinP != nil
}

// GetRepetitionPenalty returns repetition_penalty and whether it was set.
func (e *Config) GetRepetitionPenalty() (float64, bool) {
	if e == nil || e.RepetitionPenalty == nil {
		return 0, false
	}
	return *e.RepetitionPenalty, true
}

// HasRepetitionPenalty returns true if repetition_penalty is configured.
func (e *Config) HasRepetitionPenalty() bool {
	return e != nil && e.RepetitionPenalty != nil
}

// GetSeed returns seed and whether it was set.
func (e *Config) GetSeed() (int, bool) {
	if e == nil || e.Seed == nil {
		return 0, false
	}
	return *e.Seed, true
}

// HasSeed returns true if seed is configured.
func (e *Config) HasSeed() bool {
	return e != nil && e.Seed != nil
}

// GetLogprobs returns logprobs and whether it was set.
func (e *Config) GetLogprobs() (int, bool) {
	if e == nil || e.Logprobs == nil {
		return 0, false
	}
	return *e.Logprobs, true
}

// HasLogprobs returns true if logprobs is configured.
func (e *Config) HasLogprobs() bool {
	return e != nil && e.Logprobs != nil
}

// GetStopTokenIDs returns stop_token_ids or nil.
func (e *Config) GetStopTokenIDs() []int {
	if e == nil {
		return nil
	}
	return e.StopTokenIDs
}

// HasStopTokenIDs returns true if stop_token_ids is configured.
func (e *Config) HasStopTokenIDs() bool {
	return e != nil && len(e.StopTokenIDs) > 0
}

// GetLogitBias returns logit_bias or nil.
func (e *Config) GetLogitBias() map[string]float64 {
	if e == nil {
		return nil
	}
	return e.LogitBias
}

// HasLogitBias returns true if logit_bias is configured.
func (e *Config) HasLogitBias() bool {
	return e != nil && len(e.LogitBias) > 0
}

// GetModality returns the modality string or empty.
func (e *Config) GetModality() string {
	if e == nil {
		return ""
	}
	return e.Modality
}

// HasModality returns true if modality is configured.
func (e *Config) HasModality() bool {
	return e != nil && e.Modality != ""
}

// GetImage returns the image config or nil.
func (e *Config) GetImage() *ImageConfig {
	if e == nil {
		return nil
	}
	return e.Image
}

// HasImage returns true if image config is configured.
func (e *Config) HasImage() bool {
	return e != nil && e.Image != nil
}

// GetAudio returns the audio config or nil.
func (e *Config) GetAudio() *AudioConfig {
	if e == nil {
		return nil
	}
	return e.Audio
}

// HasAudio returns true if audio config is configured.
func (e *Config) HasAudio() bool {
	return e != nil && e.Audio != nil
}

// GetEmbedding returns the embedding config or nil.
func (e *Config) GetEmbedding() *EmbeddingConfig {
	if e == nil {
		return nil
	}
	return e.Embedding
}

// HasEmbedding returns true if embedding config is configured.
func (e *Config) HasEmbedding() bool {
	return e != nil && e.Embedding != nil
}

// GetStreaming returns the streaming config or nil.
func (e *Config) GetStreaming() *StreamingConfig {
	if e == nil {
		return nil
	}
	return e.Streaming
}

// HasStreaming returns true if streaming config is configured.
func (e *Config) HasStreaming() bool {
	return e != nil && e.Streaming != nil
}

// GetAsync returns the async config or nil.
func (e *Config) GetAsync() *AsyncConfig {
	if e == nil {
		return nil
	}
	return e.Async
}

// HasAsync returns true if async config is configured.
func (e *Config) HasAsync() bool {
	return e != nil && e.Async != nil
}

// GetFrequencyPenalty returns frequency_penalty and whether it was set.
func (e *Config) GetFrequencyPenalty() (float64, bool) {
	if e == nil || e.FrequencyPenalty == nil {
		return 0, false
	}
	return *e.FrequencyPenalty, true
}

// HasFrequencyPenalty returns true if frequency_penalty is configured.
func (e *Config) HasFrequencyPenalty() bool {
	return e != nil && e.FrequencyPenalty != nil
}

// GetPresencePenalty returns presence_penalty and whether it was set.
func (e *Config) GetPresencePenalty() (float64, bool) {
	if e == nil || e.PresencePenalty == nil {
		return 0, false
	}
	return *e.PresencePenalty, true
}

// HasPresencePenalty returns true if presence_penalty is configured.
func (e *Config) HasPresencePenalty() bool {
	return e != nil && e.PresencePenalty != nil
}

// GetN returns n and whether it was set.
func (e *Config) GetN() (int, bool) {
	if e == nil || e.N == nil {
		return 0, false
	}
	return *e.N, true
}

// HasN returns true if n is configured.
func (e *Config) HasN() bool {
	return e != nil && e.N != nil
}

// GetMaxCompletionTokens returns max_completion_tokens and whether it was set.
func (e *Config) GetMaxCompletionTokens() (int, bool) {
	if e == nil || e.MaxCompletionTokens == nil {
		return 0, false
	}
	return *e.MaxCompletionTokens, true
}

// HasMaxCompletionTokens returns true if max_completion_tokens is configured.
func (e *Config) HasMaxCompletionTokens() bool {
	return e != nil && e.MaxCompletionTokens != nil
}

// GetReasoningEffort returns the reasoning_effort string or empty.
func (e *Config) GetReasoningEffort() string {
	if e == nil {
		return ""
	}
	return e.ReasoningEffort
}

// HasReasoningEffort returns true if reasoning_effort is configured.
func (e *Config) HasReasoningEffort() bool {
	return e != nil && e.ReasoningEffort != ""
}

// GetTopA returns top_a and whether it was set.
func (e *Config) GetTopA() (float64, bool) {
	if e == nil || e.TopA == nil {
		return 0, false
	}
	return *e.TopA, true
}

// HasTopA returns true if top_a is configured.
func (e *Config) HasTopA() bool {
	return e != nil && e.TopA != nil
}

// GetUser returns the user string or empty.
func (e *Config) GetUser() string {
	if e == nil {
		return ""
	}
	return e.User
}

// HasUser returns true if user is configured.
func (e *Config) HasUser() bool {
	return e != nil && e.User != ""
}

// GetServiceTier returns the service_tier string or empty.
func (e *Config) GetServiceTier() string {
	if e == nil {
		return ""
	}
	return e.ServiceTier
}

// HasServiceTier returns true if service_tier is configured.
func (e *Config) HasServiceTier() bool {
	return e != nil && e.ServiceTier != ""
}

// GetStore returns store and whether it was set.
func (e *Config) GetStore() (bool, bool) {
	if e == nil || e.Store == nil {
		return false, false
	}
	return *e.Store, true
}

// HasStore returns true if store is configured.
func (e *Config) HasStore() bool {
	return e != nil && e.Store != nil
}

// GetEffectiveProvider detects the intended provider from configuration.
// Returns the explicit provider if set, otherwise infers from config shape or model name.
func (e *Config) GetEffectiveProvider() string {
	if e == nil {
		return ""
	}

	// Explicit provider takes precedence
	if e.Provider != "" {
		return e.Provider
	}

	// Infer from configuration shape
	if e.GuidedDecoding != nil {
		return ProviderVLLM
	}
	if e.MinP != nil || e.RepetitionPenalty != nil || len(e.StopTokenIDs) > 0 {
		return ProviderVLLM
	}
	if e.Thinking != nil && e.Thinking.Enabled {
		return ProviderAnthropic
	}

	// Try to infer from model name
	if e.Model != "" {
		if isOpenAIModel(e.Model) {
			return ProviderOpenAI
		}
		if isAnthropicModel(e.Model) {
			return ProviderAnthropic
		}
		if isGeminiModel(e.Model) {
			return ProviderGemini
		}
		if isMistralModel(e.Model) {
			return ProviderMistral
		}
		if isCohereModel(e.Model) {
			return ProviderCohere
		}
	}

	return ""
}
