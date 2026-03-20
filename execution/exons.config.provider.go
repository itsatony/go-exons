package execution

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

// ToMap converts execution config to a parameter map for LLM clients.
// Only includes parameters that were explicitly set.
func (e *Config) ToMap() map[string]any {
	if e == nil {
		return nil
	}

	result := make(map[string]any)

	if e.Temperature != nil {
		result[ParamKeyTemperature] = *e.Temperature
	}
	if e.MaxTokens != nil {
		result[ParamKeyMaxTokens] = *e.MaxTokens
	}
	if e.TopP != nil {
		result[ParamKeyTopP] = *e.TopP
	}
	if len(e.StopSequences) > 0 {
		result[ParamKeyStop] = e.StopSequences
	}
	if e.MinP != nil {
		result[ParamKeyMinP] = *e.MinP
	}
	if e.RepetitionPenalty != nil {
		result[ParamKeyRepetitionPenalty] = *e.RepetitionPenalty
	}
	if e.Seed != nil {
		result[ParamKeySeed] = *e.Seed
	}
	if e.Logprobs != nil {
		result[ParamKeyLogprobs] = *e.Logprobs
	}
	if len(e.StopTokenIDs) > 0 {
		result[ParamKeyStopTokenIDs] = e.StopTokenIDs
	}
	if len(e.LogitBias) > 0 {
		result[ParamKeyLogitBias] = e.LogitBias
	}

	// LLM parameter alignment fields
	if e.FrequencyPenalty != nil {
		result[ParamKeyFrequencyPenalty] = *e.FrequencyPenalty
	}
	if e.PresencePenalty != nil {
		result[ParamKeyPresencePenalty] = *e.PresencePenalty
	}
	if e.N != nil {
		result[ParamKeyN] = *e.N
	}
	if e.MaxCompletionTokens != nil {
		result[ParamKeyMaxCompletionTokens] = *e.MaxCompletionTokens
	}
	if e.ReasoningEffort != "" {
		result[ParamKeyReasoningEffort] = e.ReasoningEffort
	}
	if e.TopA != nil {
		result[ParamKeyTopA] = *e.TopA
	}
	if e.User != "" {
		result[ParamKeyUser] = e.User
	}
	if e.ServiceTier != "" {
		result[ParamKeyServiceTier] = e.ServiceTier
	}
	if e.Store != nil {
		result[ParamKeyStore] = *e.Store
	}

	// Media fields
	if e.Modality != "" {
		result[ParamKeyModality] = e.Modality
	}
	if e.Image != nil {
		result[ParamKeyImage] = e.Image.ToMap()
	}
	if e.Audio != nil {
		result[ParamKeyAudio] = e.Audio.ToMap()
	}
	if e.Embedding != nil {
		result[ParamKeyEmbedding] = e.Embedding.ToMap()
	}
	if e.Streaming != nil {
		result[ParamKeyStreaming] = e.Streaming.ToMap()
	}
	if e.Async != nil {
		result[ParamKeyAsync] = e.Async.ToMap()
	}

	return result
}

// ToOpenAI converts the execution config to OpenAI API format.
func (e *Config) ToOpenAI() map[string]any {
	if e == nil {
		return nil
	}

	result := make(map[string]any)

	if e.Model != "" {
		result[ParamKeyModel] = e.Model
	}
	if e.Temperature != nil {
		result[ParamKeyTemperature] = *e.Temperature
	}
	if e.MaxTokens != nil {
		result[ParamKeyMaxTokens] = *e.MaxTokens
	}
	if e.TopP != nil {
		result[ParamKeyTopP] = *e.TopP
	}
	if len(e.StopSequences) > 0 {
		result[ParamKeyStop] = e.StopSequences
	}

	if e.Seed != nil {
		result[ParamKeySeed] = *e.Seed
	}
	if e.Logprobs != nil {
		result[ParamKeyLogprobs] = true
		result[ParamKeyTopLogprobs] = *e.Logprobs
	}
	if len(e.LogitBias) > 0 {
		result[ParamKeyLogitBias] = e.LogitBias
	}

	// OpenAI LLM parameter alignment
	if e.FrequencyPenalty != nil {
		result[ParamKeyFrequencyPenalty] = *e.FrequencyPenalty
	}
	if e.PresencePenalty != nil {
		result[ParamKeyPresencePenalty] = *e.PresencePenalty
	}
	if e.N != nil {
		result[ParamKeyN] = *e.N
	}
	if e.MaxCompletionTokens != nil {
		result[ParamKeyMaxCompletionTokens] = *e.MaxCompletionTokens
	}
	if e.ReasoningEffort != "" {
		result[ParamKeyReasoningEffort] = e.ReasoningEffort
	}
	if e.User != "" {
		result[ParamKeyUser] = e.User
	}
	if e.ServiceTier != "" {
		result[ParamKeyServiceTier] = e.ServiceTier
	}
	if e.Store != nil {
		result[ParamKeyStore] = *e.Store
	}

	if e.ResponseFormat != nil {
		result[ParamKeyResponseFormat] = e.ResponseFormat.ToOpenAI()
	}

	// OpenAI media params
	e.openAIImageParams(result)
	e.openAIAudioParams(result)
	e.openAIEmbeddingParams(result)
	if e.Streaming != nil && e.Streaming.Enabled {
		result[ParamKeyStream] = true
	}

	// Merge provider options
	for k, v := range e.ProviderOptions {
		result[k] = v
	}

	return result
}

// openAIImageParams adds OpenAI image generation params to the result map.
func (e *Config) openAIImageParams(result map[string]any) {
	if e.Image == nil {
		return
	}
	size := e.Image.EffectiveSize()
	if size != "" {
		result[ParamKeyImageSize] = size
	}
	if e.Image.Quality != "" {
		result[ParamKeyImageQuality] = e.Image.Quality
	}
	if e.Image.Style != "" {
		result[ParamKeyImageStyle] = e.Image.Style
	}
	if e.Image.NumImages != nil {
		result[ParamKeyImageN] = *e.Image.NumImages
	}
}

// openAIAudioParams adds OpenAI audio/TTS params to the result map.
func (e *Config) openAIAudioParams(result map[string]any) {
	if e.Audio == nil {
		return
	}
	if e.Audio.Voice != "" {
		result[ParamKeyVoice] = e.Audio.Voice
	}
	if e.Audio.Speed != nil {
		result[ParamKeySpeed] = *e.Audio.Speed
	}
	if e.Audio.OutputFormat != "" {
		// Only set response_format for audio if structured output response_format is not already set.
		// These target different OpenAI endpoints (TTS vs chat completions) but share the same key.
		if _, hasRF := result[ParamKeyResponseFormat]; !hasRF {
			result[ParamKeyResponseFormat] = e.Audio.OutputFormat
		}
	}
}

// openAIEmbeddingParams adds OpenAI embedding params to the result map.
func (e *Config) openAIEmbeddingParams(result map[string]any) {
	if e.Embedding == nil {
		return
	}
	if e.Embedding.Dimensions != nil {
		result[ParamKeyDimensions] = *e.Embedding.Dimensions
	}
	if e.Embedding.Format != "" {
		result[ParamKeyEncodingFormat] = e.Embedding.Format
	}
}

// ToAnthropic converts the execution config to Anthropic API format.
func (e *Config) ToAnthropic() map[string]any {
	if e == nil {
		return nil
	}

	result := make(map[string]any)

	if e.Model != "" {
		result[ParamKeyModel] = e.Model
	}
	if e.Temperature != nil {
		result[ParamKeyTemperature] = *e.Temperature
	}
	if e.MaxTokens != nil {
		result[ParamKeyMaxTokens] = *e.MaxTokens
	}
	if e.TopP != nil {
		result[ParamKeyTopP] = *e.TopP
	}
	if e.TopK != nil {
		result[ParamKeyTopK] = *e.TopK
	}
	if len(e.StopSequences) > 0 {
		result[ParamKeyStopSequences] = e.StopSequences
	}
	if e.Seed != nil {
		result[ParamKeySeed] = *e.Seed
	}

	// Anthropic LLM parameter alignment
	if e.ReasoningEffort != "" {
		result[ParamKeyReasoningEffort] = e.ReasoningEffort
	}
	if e.User != "" {
		result[ParamKeyUser] = e.User
	}

	// Handle extended thinking
	if e.Thinking != nil && e.Thinking.Enabled {
		thinking := map[string]any{
			ParamKeyThinkingType: ParamKeyThinkingTypeEnabled,
		}
		if e.Thinking.BudgetTokens != nil {
			thinking[ParamKeyBudgetTokens] = *e.Thinking.BudgetTokens
		}
		result[ParamKeyAnthropicThinking] = thinking
	}

	// Handle response format for Anthropic
	if e.ResponseFormat != nil {
		result[ParamKeyAnthropicOutputFormat] = e.ResponseFormat.ToAnthropic()
	}

	// Streaming only — no media generation params for Anthropic
	if e.Streaming != nil && e.Streaming.Enabled {
		result[ParamKeyStream] = true
	}

	// Merge provider options
	for k, v := range e.ProviderOptions {
		result[k] = v
	}

	return result
}

// ToGemini converts the execution config to Gemini/Vertex AI API format.
// Supports embedding parameters: output_dimensionality (from Dimensions) and task_type
// (from InputType via GeminiTaskType mapping). Also supports image params (aspectRatio,
// numberOfImages) and streaming.
func (e *Config) ToGemini() map[string]any {
	if e == nil {
		return nil
	}

	result := make(map[string]any)

	if e.Model != "" {
		result[ParamKeyModel] = e.Model
	}

	// Gemini uses generationConfig for parameters
	genConfig := make(map[string]any)
	if e.Temperature != nil {
		genConfig[ParamKeyTemperature] = *e.Temperature
	}
	if e.MaxTokens != nil {
		genConfig[ParamKeyGeminiMaxTokens] = *e.MaxTokens
	}
	if e.TopP != nil {
		genConfig[ParamKeyGeminiTopP] = *e.TopP
	}
	if e.TopK != nil {
		genConfig[ParamKeyGeminiTopK] = *e.TopK
	}
	if len(e.StopSequences) > 0 {
		genConfig[ParamKeyGeminiStopSeqs] = e.StopSequences
	}

	if e.ResponseFormat != nil {
		genConfig[ParamKeyGeminiResponseMime] = GeminiResponseMimeJSON
		genConfig[ParamKeyGeminiResponseSchema] = e.ResponseFormat.ToGemini()
	}

	// Gemini image params in generationConfig
	if e.Image != nil {
		if e.Image.AspectRatio != "" {
			genConfig[ParamKeyAspectRatio] = e.Image.AspectRatio
		}
		if e.Image.NumImages != nil {
			genConfig[ParamKeyGeminiNumImages] = *e.Image.NumImages
		}
	}

	// Gemini embedding params in generationConfig
	if e.Embedding != nil {
		if e.Embedding.Dimensions != nil {
			genConfig[ParamKeyOutputDimensionality] = *e.Embedding.Dimensions
		}
		if e.Embedding.InputType != "" {
			if taskType, err := GeminiTaskType(e.Embedding.InputType); err == nil {
				genConfig[ParamKeyTaskType] = taskType
			}
		}
	}

	if len(genConfig) > 0 {
		result[ParamKeyGenerationConfig] = genConfig
	}

	// Streaming
	if e.Streaming != nil && e.Streaming.Enabled {
		result[ParamKeyStream] = true
	}

	// Merge provider options
	for k, v := range e.ProviderOptions {
		result[k] = v
	}

	return result
}

// ToVLLM converts the execution config to vLLM API format.
// Supports embedding parameters: normalize and pooling_type. Also supports guided decoding,
// extended inference params (min_p, repetition_penalty, logprobs, etc.), and streaming.
func (e *Config) ToVLLM() map[string]any {
	if e == nil {
		return nil
	}

	result := make(map[string]any)

	if e.Model != "" {
		result[ParamKeyModel] = e.Model
	}
	if e.Temperature != nil {
		result[ParamKeyTemperature] = *e.Temperature
	}
	if e.MaxTokens != nil {
		result[ParamKeyMaxTokens] = *e.MaxTokens
	}
	if e.TopP != nil {
		result[ParamKeyTopP] = *e.TopP
	}
	if e.TopK != nil {
		result[ParamKeyTopK] = *e.TopK
	}
	if len(e.StopSequences) > 0 {
		result[ParamKeyStop] = e.StopSequences
	}
	if e.MinP != nil {
		result[ParamKeyMinP] = *e.MinP
	}
	if e.RepetitionPenalty != nil {
		result[ParamKeyRepetitionPenalty] = *e.RepetitionPenalty
	}
	if e.Seed != nil {
		result[ParamKeySeed] = *e.Seed
	}
	if e.Logprobs != nil {
		result[ParamKeyLogprobs] = *e.Logprobs
	}
	if len(e.StopTokenIDs) > 0 {
		result[ParamKeyStopTokenIDs] = e.StopTokenIDs
	}
	if len(e.LogitBias) > 0 {
		result[ParamKeyLogitBias] = e.LogitBias
	}

	// vLLM LLM parameter alignment
	if e.FrequencyPenalty != nil {
		result[ParamKeyFrequencyPenalty] = *e.FrequencyPenalty
	}
	if e.PresencePenalty != nil {
		result[ParamKeyPresencePenalty] = *e.PresencePenalty
	}
	if e.N != nil {
		result[ParamKeyN] = *e.N
	}

	// Add guided decoding parameters
	if e.GuidedDecoding != nil {
		gdParams := e.GuidedDecoding.ToVLLM()
		for k, v := range gdParams {
			result[k] = v
		}
	}

	// vLLM embedding params
	if e.Embedding != nil {
		if e.Embedding.Normalize != nil {
			result[ParamKeyNormalize] = *e.Embedding.Normalize
		}
		if e.Embedding.PoolingType != "" {
			result[ParamKeyPoolingType] = e.Embedding.PoolingType
		}
	}

	// Streaming only — no media params for vLLM (text inference only)
	if e.Streaming != nil && e.Streaming.Enabled {
		result[ParamKeyStream] = true
	}

	// Merge provider options
	for k, v := range e.ProviderOptions {
		result[k] = v
	}

	return result
}

// ToMistral converts the execution config to Mistral AI API format.
// Mistral uses an OpenAI-compatible structure with provider-specific embedding params:
// output_dimension (from Dimensions), encoding_format (from Format), and output_dtype.
// Supports response_format and streaming.
func (e *Config) ToMistral() map[string]any {
	if e == nil {
		return nil
	}

	result := make(map[string]any)

	if e.Model != "" {
		result[ParamKeyModel] = e.Model
	}
	if e.Temperature != nil {
		result[ParamKeyTemperature] = *e.Temperature
	}
	if e.MaxTokens != nil {
		result[ParamKeyMaxTokens] = *e.MaxTokens
	}
	if e.TopP != nil {
		result[ParamKeyTopP] = *e.TopP
	}
	if len(e.StopSequences) > 0 {
		result[ParamKeyStop] = e.StopSequences
	}
	if e.Seed != nil {
		result[ParamKeySeed] = *e.Seed
	}

	// Mistral LLM parameter alignment
	if e.FrequencyPenalty != nil {
		result[ParamKeyFrequencyPenalty] = *e.FrequencyPenalty
	}
	if e.PresencePenalty != nil {
		result[ParamKeyPresencePenalty] = *e.PresencePenalty
	}

	if e.ResponseFormat != nil {
		result[ParamKeyResponseFormat] = e.ResponseFormat.ToOpenAI()
	}

	// Mistral embedding params
	if e.Embedding != nil {
		if e.Embedding.Dimensions != nil {
			result[ParamKeyOutputDimension] = *e.Embedding.Dimensions
		}
		if e.Embedding.Format != "" {
			result[ParamKeyEncodingFormat] = e.Embedding.Format
		}
		if e.Embedding.OutputDtype != "" {
			result[ParamKeyOutputDtype] = e.Embedding.OutputDtype
		}
	}

	if e.Streaming != nil && e.Streaming.Enabled {
		result[ParamKeyStream] = true
	}

	// Merge provider options
	for k, v := range e.ProviderOptions {
		result[k] = v
	}

	return result
}

// ToCohere converts the execution config to Cohere API format.
// Cohere uses different parameter names than OpenAI: "p" for top_p, "k" for top_k,
// "stop_sequences" for stop. Embedding params: output_dimension, input_type,
// embedding_types (OutputDtype as []string), and truncate (truncation in UPPER_CASE via
// CohereUpperCase). Supports streaming.
func (e *Config) ToCohere() map[string]any {
	if e == nil {
		return nil
	}

	result := make(map[string]any)

	if e.Model != "" {
		result[ParamKeyModel] = e.Model
	}
	if e.Temperature != nil {
		result[ParamKeyTemperature] = *e.Temperature
	}
	if e.MaxTokens != nil {
		result[ParamKeyMaxTokens] = *e.MaxTokens
	}
	if e.TopP != nil {
		result[ParamKeyCohereTopP] = *e.TopP
	}
	if e.TopK != nil {
		result[ParamKeyCohereTopK] = *e.TopK
	}
	if len(e.StopSequences) > 0 {
		result[ParamKeyCohereStopSequences] = e.StopSequences
	}
	if e.Seed != nil {
		result[ParamKeySeed] = *e.Seed
	}

	// Cohere embedding params
	if e.Embedding != nil {
		if e.Embedding.Dimensions != nil {
			result[ParamKeyOutputDimension] = *e.Embedding.Dimensions
		}
		if e.Embedding.InputType != "" {
			result[ParamKeyInputType] = e.Embedding.InputType
		}
		if e.Embedding.OutputDtype != "" {
			result[ParamKeyEmbeddingTypes] = []string{e.Embedding.OutputDtype}
		}
		if e.Embedding.Truncation != "" {
			if upper, err := CohereUpperCase(e.Embedding.Truncation); err == nil {
				result[ParamKeyTruncate] = upper
			}
		}
	}

	if e.Streaming != nil && e.Streaming.Enabled {
		result[ParamKeyStream] = true
	}

	// Merge provider options
	for k, v := range e.ProviderOptions {
		result[k] = v
	}

	return result
}

// ProviderFormat returns the response format for a specific provider.
func (e *Config) ProviderFormat(provider string) (map[string]any, error) {
	if e == nil {
		return nil, nil
	}

	switch provider {
	case ProviderOpenAI, ProviderAzure:
		if e.ResponseFormat != nil {
			return e.ResponseFormat.ToOpenAI(), nil
		}
		return nil, nil

	case ProviderAnthropic:
		if e.ResponseFormat != nil {
			return e.ResponseFormat.ToAnthropic(), nil
		}
		return nil, nil

	case ProviderGoogle, ProviderGemini, ProviderVertex:
		if e.ResponseFormat != nil {
			return e.ResponseFormat.ToGemini(), nil
		}
		return nil, nil

	case ProviderVLLM:
		if e.GuidedDecoding != nil {
			return e.GuidedDecoding.ToVLLM(), nil
		}
		return nil, nil

	case ProviderMistral:
		// Mistral uses OpenAI-compatible response_format
		if e.ResponseFormat != nil {
			return e.ResponseFormat.ToOpenAI(), nil
		}
		return nil, nil

	case ProviderCohere:
		// Cohere does not use response_format
		return nil, nil

	default:
		return nil, NewProviderError(ErrMsgSchemaUnsupportedProvider, provider)
	}
}

// JSON returns the JSON representation of the execution config.
func (e *Config) JSON() (string, error) {
	if e == nil {
		return "", nil
	}
	data, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// YAML returns the YAML representation of the execution config.
func (e *Config) YAML() (string, error) {
	if e == nil {
		return "", nil
	}
	data, err := yaml.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(data), nil
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
