package execution

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// =============================================================================
// ToMap
// =============================================================================

func TestConfig_ToMap_Nil(t *testing.T) {
	var c *Config
	assert.Nil(t, c.ToMap())
}

func TestConfig_ToMap_Empty(t *testing.T) {
	c := &Config{}
	m := c.ToMap()
	assert.Empty(t, m)
}

func TestConfig_ToMap_AllFields(t *testing.T) {
	c := &Config{
		Temperature:         ptrFloat64(0.7),
		MaxTokens:           ptrInt(1000),
		TopP:                ptrFloat64(0.9),
		StopSequences:       []string{"stop"},
		MinP:                ptrFloat64(0.1),
		RepetitionPenalty:   ptrFloat64(1.2),
		Seed:                ptrInt(42),
		Logprobs:            ptrInt(5),
		StopTokenIDs:        []int{100},
		LogitBias:           map[string]float64{"0": 1.0},
		FrequencyPenalty:    ptrFloat64(0.5),
		PresencePenalty:     ptrFloat64(0.3),
		N:                   ptrInt(2),
		MaxCompletionTokens: ptrInt(4096),
		ReasoningEffort:     ReasoningEffortHigh,
		TopA:                ptrFloat64(0.5),
		User:                "user-123",
		ServiceTier:         ServiceTierAuto,
		Store:               ptrBool(true),
		Modality:            ModalityText,
		Image:               &ImageConfig{Width: ptrInt(1024)},
		Audio:               &AudioConfig{Voice: "alloy"},
		Embedding:           &EmbeddingConfig{Dimensions: ptrInt(768)},
		Streaming:           &StreamingConfig{Enabled: true},
		Async:               &AsyncConfig{Enabled: true},
	}

	m := c.ToMap()

	assert.Equal(t, 0.7, m[ParamKeyTemperature])
	assert.Equal(t, 1000, m[ParamKeyMaxTokens])
	assert.Equal(t, 0.9, m[ParamKeyTopP])
	assert.Equal(t, []string{"stop"}, m[ParamKeyStop])
	assert.Equal(t, 0.1, m[ParamKeyMinP])
	assert.Equal(t, 1.2, m[ParamKeyRepetitionPenalty])
	assert.Equal(t, 42, m[ParamKeySeed])
	assert.Equal(t, 5, m[ParamKeyLogprobs])
	assert.Equal(t, []int{100}, m[ParamKeyStopTokenIDs])
	assert.Equal(t, map[string]float64{"0": 1.0}, m[ParamKeyLogitBias])
	assert.Equal(t, 0.5, m[ParamKeyFrequencyPenalty])
	assert.Equal(t, 0.3, m[ParamKeyPresencePenalty])
	assert.Equal(t, 2, m[ParamKeyN])
	assert.Equal(t, 4096, m[ParamKeyMaxCompletionTokens])
	assert.Equal(t, ReasoningEffortHigh, m[ParamKeyReasoningEffort])
	assert.Equal(t, 0.5, m[ParamKeyTopA])
	assert.Equal(t, "user-123", m[ParamKeyUser])
	assert.Equal(t, ServiceTierAuto, m[ParamKeyServiceTier])
	assert.Equal(t, true, m[ParamKeyStore])
	assert.Equal(t, ModalityText, m[ParamKeyModality])
	assert.NotNil(t, m[ParamKeyImage])
	assert.NotNil(t, m[ParamKeyAudio])
	assert.NotNil(t, m[ParamKeyEmbedding])
	assert.NotNil(t, m[ParamKeyStreaming])
	assert.NotNil(t, m[ParamKeyAsync])
}

func TestConfig_ToMap_OnlySetFields(t *testing.T) {
	c := &Config{Temperature: ptrFloat64(0.5)}
	m := c.ToMap()
	assert.Len(t, m, 1)
	assert.Equal(t, 0.5, m[ParamKeyTemperature])
}

// =============================================================================
// ToOpenAI
// =============================================================================

func TestConfig_ToOpenAI_Nil(t *testing.T) {
	var c *Config
	assert.Nil(t, c.ToOpenAI())
}

func TestConfig_ToOpenAI_BasicParams(t *testing.T) {
	c := &Config{
		Model:         "gpt-4",
		Temperature:   ptrFloat64(0.7),
		MaxTokens:     ptrInt(1000),
		TopP:          ptrFloat64(0.9),
		StopSequences: []string{"stop"},
	}
	m := c.ToOpenAI()
	assert.Equal(t, "gpt-4", m[ParamKeyModel])
	assert.Equal(t, 0.7, m[ParamKeyTemperature])
	assert.Equal(t, 1000, m[ParamKeyMaxTokens])
	assert.Equal(t, 0.9, m[ParamKeyTopP])
	assert.Equal(t, []string{"stop"}, m[ParamKeyStop])
}

func TestConfig_ToOpenAI_SeedAndLogprobs(t *testing.T) {
	c := &Config{
		Seed:     ptrInt(42),
		Logprobs: ptrInt(5),
	}
	m := c.ToOpenAI()
	assert.Equal(t, 42, m[ParamKeySeed])
	// OpenAI uses dual-field logprobs: logprobs=true + top_logprobs=N
	assert.Equal(t, true, m[ParamKeyLogprobs])
	assert.Equal(t, 5, m[ParamKeyTopLogprobs])
}

func TestConfig_ToOpenAI_LogitBias(t *testing.T) {
	c := &Config{LogitBias: map[string]float64{"100": 5.0}}
	m := c.ToOpenAI()
	assert.Equal(t, map[string]float64{"100": 5.0}, m[ParamKeyLogitBias])
}

func TestConfig_ToOpenAI_V29Params(t *testing.T) {
	c := &Config{
		FrequencyPenalty:    ptrFloat64(0.5),
		PresencePenalty:     ptrFloat64(0.3),
		N:                   ptrInt(2),
		MaxCompletionTokens: ptrInt(4096),
		ReasoningEffort:     ReasoningEffortHigh,
		User:                "user-123",
		ServiceTier:         ServiceTierAuto,
		Store:               ptrBool(true),
	}
	m := c.ToOpenAI()
	assert.Equal(t, 0.5, m[ParamKeyFrequencyPenalty])
	assert.Equal(t, 0.3, m[ParamKeyPresencePenalty])
	assert.Equal(t, 2, m[ParamKeyN])
	assert.Equal(t, 4096, m[ParamKeyMaxCompletionTokens])
	assert.Equal(t, ReasoningEffortHigh, m[ParamKeyReasoningEffort])
	assert.Equal(t, "user-123", m[ParamKeyUser])
	assert.Equal(t, ServiceTierAuto, m[ParamKeyServiceTier])
	assert.Equal(t, true, m[ParamKeyStore])
}

func TestConfig_ToOpenAI_ResponseFormat(t *testing.T) {
	c := &Config{
		ResponseFormat: &ResponseFormat{Type: ResponseFormatJSONSchema},
	}
	m := c.ToOpenAI()
	assert.NotNil(t, m[ParamKeyResponseFormat])
}

func TestConfig_ToOpenAI_ImageParams(t *testing.T) {
	c := &Config{
		Image: &ImageConfig{
			Size:      "1024x1024",
			Quality:   ImageQualityHD,
			Style:     ImageStyleVivid,
			NumImages: ptrInt(2),
		},
	}
	m := c.ToOpenAI()
	assert.Equal(t, "1024x1024", m[ParamKeyImageSize])
	assert.Equal(t, ImageQualityHD, m[ParamKeyImageQuality])
	assert.Equal(t, ImageStyleVivid, m[ParamKeyImageStyle])
	assert.Equal(t, 2, m[ParamKeyImageN])
}

func TestConfig_ToOpenAI_AudioParams(t *testing.T) {
	c := &Config{
		Audio: &AudioConfig{
			Voice:        "alloy",
			Speed:        ptrFloat64(1.5),
			OutputFormat: AudioFormatMP3,
		},
	}
	m := c.ToOpenAI()
	assert.Equal(t, "alloy", m[ParamKeyVoice])
	assert.Equal(t, 1.5, m[ParamKeySpeed])
	assert.Equal(t, AudioFormatMP3, m[ParamKeyResponseFormat])
}

func TestConfig_ToOpenAI_AudioResponseFormatCollisionGuard(t *testing.T) {
	c := &Config{
		ResponseFormat: &ResponseFormat{Type: ResponseFormatJSONSchema},
		Audio:          &AudioConfig{OutputFormat: AudioFormatMP3},
	}
	m := c.ToOpenAI()
	// response_format should be the structured output format, not the audio format
	rf, ok := m[ParamKeyResponseFormat].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, ResponseFormatJSONSchema, rf[SchemaKeyType])
}

func TestConfig_ToOpenAI_EmbeddingParams(t *testing.T) {
	c := &Config{
		Embedding: &EmbeddingConfig{
			Dimensions: ptrInt(768),
			Format:     EmbeddingFormatFloat,
		},
	}
	m := c.ToOpenAI()
	assert.Equal(t, 768, m[ParamKeyDimensions])
	assert.Equal(t, EmbeddingFormatFloat, m[ParamKeyEncodingFormat])
}

func TestConfig_ToOpenAI_Streaming(t *testing.T) {
	c := &Config{Streaming: &StreamingConfig{Enabled: true}}
	m := c.ToOpenAI()
	assert.Equal(t, true, m[ParamKeyStream])
}

func TestConfig_ToOpenAI_ProviderOptions(t *testing.T) {
	c := &Config{ProviderOptions: map[string]any{"custom_key": "value"}}
	m := c.ToOpenAI()
	assert.Equal(t, "value", m["custom_key"])
}

func TestConfig_ToOpenAI_NotPresent(t *testing.T) {
	// MinP, RepetitionPenalty, StopTokenIDs, TopK should NOT be in OpenAI output
	c := &Config{
		MinP:              ptrFloat64(0.1),
		RepetitionPenalty: ptrFloat64(1.2),
		StopTokenIDs:      []int{100},
		TopK:              ptrInt(40),
	}
	m := c.ToOpenAI()
	assert.NotContains(t, m, ParamKeyMinP)
	assert.NotContains(t, m, ParamKeyRepetitionPenalty)
	assert.NotContains(t, m, ParamKeyStopTokenIDs)
	assert.NotContains(t, m, ParamKeyTopK)
}

// =============================================================================
// ToAnthropic
// =============================================================================

func TestConfig_ToAnthropic_Nil(t *testing.T) {
	var c *Config
	assert.Nil(t, c.ToAnthropic())
}

func TestConfig_ToAnthropic_BasicParams(t *testing.T) {
	c := &Config{
		Model:         "claude-3-opus",
		Temperature:   ptrFloat64(0.7),
		MaxTokens:     ptrInt(1000),
		TopP:          ptrFloat64(0.9),
		TopK:          ptrInt(40),
		StopSequences: []string{"stop"},
		Seed:          ptrInt(42),
	}
	m := c.ToAnthropic()
	assert.Equal(t, "claude-3-opus", m[ParamKeyModel])
	assert.Equal(t, 0.7, m[ParamKeyTemperature])
	assert.Equal(t, 1000, m[ParamKeyMaxTokens])
	assert.Equal(t, 0.9, m[ParamKeyTopP])
	assert.Equal(t, 40, m[ParamKeyTopK])
	assert.Equal(t, []string{"stop"}, m[ParamKeyStopSequences])
	assert.Equal(t, 42, m[ParamKeySeed])
}

func TestConfig_ToAnthropic_ReasoningAndUser(t *testing.T) {
	c := &Config{ReasoningEffort: ReasoningEffortHigh, User: "user-123"}
	m := c.ToAnthropic()
	assert.Equal(t, ReasoningEffortHigh, m[ParamKeyReasoningEffort])
	assert.Equal(t, "user-123", m[ParamKeyUser])
}

func TestConfig_ToAnthropic_Thinking(t *testing.T) {
	c := &Config{
		Thinking: &ThinkingConfig{Enabled: true, BudgetTokens: ptrInt(1000)},
	}
	m := c.ToAnthropic()
	thinking, ok := m[ParamKeyAnthropicThinking].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, ParamKeyThinkingTypeEnabled, thinking[ParamKeyThinkingType])
	assert.Equal(t, 1000, thinking[ParamKeyBudgetTokens])
}

func TestConfig_ToAnthropic_ResponseFormat(t *testing.T) {
	c := &Config{
		ResponseFormat: &ResponseFormat{
			Type: ResponseFormatJSONSchema,
			JSONSchema: &JSONSchemaSpec{
				Name:   "test",
				Schema: map[string]any{"type": "object"},
			},
		},
	}
	m := c.ToAnthropic()
	assert.NotNil(t, m[ParamKeyAnthropicOutputFormat])
}

func TestConfig_ToAnthropic_Streaming(t *testing.T) {
	c := &Config{Streaming: &StreamingConfig{Enabled: true}}
	m := c.ToAnthropic()
	assert.Equal(t, true, m[ParamKeyStream])
}

func TestConfig_ToAnthropic_NotPresent(t *testing.T) {
	c := &Config{
		MinP:              ptrFloat64(0.1),
		RepetitionPenalty: ptrFloat64(1.2),
		Logprobs:          ptrInt(5),
		FrequencyPenalty:  ptrFloat64(0.5),
		PresencePenalty:   ptrFloat64(0.3),
		N:                 ptrInt(2),
		Image:             &ImageConfig{Width: ptrInt(1024)},
		Audio:             &AudioConfig{Voice: "alloy"},
		Embedding:         &EmbeddingConfig{Dimensions: ptrInt(768)},
	}
	m := c.ToAnthropic()
	assert.NotContains(t, m, ParamKeyMinP)
	assert.NotContains(t, m, ParamKeyRepetitionPenalty)
	assert.NotContains(t, m, ParamKeyLogprobs)
	assert.NotContains(t, m, ParamKeyFrequencyPenalty)
	assert.NotContains(t, m, ParamKeyPresencePenalty)
	assert.NotContains(t, m, ParamKeyN)
	assert.NotContains(t, m, ParamKeyImage)
	assert.NotContains(t, m, ParamKeyAudio)
	assert.NotContains(t, m, ParamKeyEmbedding)
}

// =============================================================================
// ToGemini
// =============================================================================

func TestConfig_ToGemini_Nil(t *testing.T) {
	var c *Config
	assert.Nil(t, c.ToGemini())
}

func TestConfig_ToGemini_BasicParams(t *testing.T) {
	c := &Config{
		Model:         "gemini-pro",
		Temperature:   ptrFloat64(0.7),
		MaxTokens:     ptrInt(1000),
		TopP:          ptrFloat64(0.9),
		TopK:          ptrInt(40),
		StopSequences: []string{"stop"},
	}
	m := c.ToGemini()
	assert.Equal(t, "gemini-pro", m[ParamKeyModel])

	gc, ok := m[ParamKeyGenerationConfig].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 0.7, gc[ParamKeyTemperature])
	assert.Equal(t, 1000, gc[ParamKeyGeminiMaxTokens])
	assert.Equal(t, 0.9, gc[ParamKeyGeminiTopP])
	assert.Equal(t, 40, gc[ParamKeyGeminiTopK])
	assert.Equal(t, []string{"stop"}, gc[ParamKeyGeminiStopSeqs])
}

func TestConfig_ToGemini_ResponseFormat(t *testing.T) {
	c := &Config{
		ResponseFormat: &ResponseFormat{Type: ResponseFormatJSONSchema},
	}
	m := c.ToGemini()
	gc, ok := m[ParamKeyGenerationConfig].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, GeminiResponseMimeJSON, gc[ParamKeyGeminiResponseMime])
	assert.NotNil(t, gc[ParamKeyGeminiResponseSchema])
}

func TestConfig_ToGemini_ImageParams(t *testing.T) {
	c := &Config{
		Image: &ImageConfig{
			AspectRatio: "16:9",
			NumImages:   ptrInt(3),
		},
	}
	m := c.ToGemini()
	gc, ok := m[ParamKeyGenerationConfig].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "16:9", gc[ParamKeyAspectRatio])
	assert.Equal(t, 3, gc[ParamKeyGeminiNumImages])
}

func TestConfig_ToGemini_EmbeddingParams(t *testing.T) {
	c := &Config{
		Embedding: &EmbeddingConfig{
			Dimensions: ptrInt(768),
			InputType:  EmbeddingInputTypeSearchQuery,
		},
	}
	m := c.ToGemini()
	gc, ok := m[ParamKeyGenerationConfig].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 768, gc[ParamKeyOutputDimensionality])
	assert.Equal(t, GeminiTaskRetrievalQuery, gc[ParamKeyTaskType])
}

func TestConfig_ToGemini_Streaming(t *testing.T) {
	c := &Config{Streaming: &StreamingConfig{Enabled: true}}
	m := c.ToGemini()
	assert.Equal(t, true, m[ParamKeyStream])
}

func TestConfig_ToGemini_NotPresent(t *testing.T) {
	c := &Config{
		MinP:              ptrFloat64(0.1),
		RepetitionPenalty: ptrFloat64(1.2),
		Logprobs:          ptrInt(5),
	}
	m := c.ToGemini()
	// These should not appear anywhere in the output
	gc, _ := m[ParamKeyGenerationConfig].(map[string]any)
	if gc != nil {
		assert.NotContains(t, gc, ParamKeyMinP)
		assert.NotContains(t, gc, ParamKeyRepetitionPenalty)
		assert.NotContains(t, gc, ParamKeyLogprobs)
	}
}

// =============================================================================
// ToVLLM
// =============================================================================

func TestConfig_ToVLLM_Nil(t *testing.T) {
	var c *Config
	assert.Nil(t, c.ToVLLM())
}

func TestConfig_ToVLLM_AllParams(t *testing.T) {
	c := &Config{
		Model:             "meta-llama/Llama-2-7b",
		Temperature:       ptrFloat64(0.8),
		MaxTokens:         ptrInt(1000),
		TopP:              ptrFloat64(0.9),
		TopK:              ptrInt(40),
		StopSequences:     []string{"stop"},
		MinP:              ptrFloat64(0.1),
		RepetitionPenalty: ptrFloat64(1.2),
		Seed:              ptrInt(42),
		Logprobs:          ptrInt(5),
		StopTokenIDs:      []int{100},
		LogitBias:         map[string]float64{"0": 1.0},
		FrequencyPenalty:  ptrFloat64(0.5),
		PresencePenalty:   ptrFloat64(0.3),
		N:                 ptrInt(2),
	}
	m := c.ToVLLM()
	assert.Equal(t, "meta-llama/Llama-2-7b", m[ParamKeyModel])
	assert.Equal(t, 0.8, m[ParamKeyTemperature])
	assert.Equal(t, 1000, m[ParamKeyMaxTokens])
	assert.Equal(t, 0.9, m[ParamKeyTopP])
	assert.Equal(t, 40, m[ParamKeyTopK])
	assert.Equal(t, 0.1, m[ParamKeyMinP])
	assert.Equal(t, 1.2, m[ParamKeyRepetitionPenalty])
	assert.Equal(t, 42, m[ParamKeySeed])
	assert.Equal(t, 5, m[ParamKeyLogprobs])
	assert.Equal(t, []int{100}, m[ParamKeyStopTokenIDs])
	assert.Equal(t, map[string]float64{"0": 1.0}, m[ParamKeyLogitBias])
	assert.Equal(t, 0.5, m[ParamKeyFrequencyPenalty])
	assert.Equal(t, 0.3, m[ParamKeyPresencePenalty])
	assert.Equal(t, 2, m[ParamKeyN])
}

func TestConfig_ToVLLM_GuidedDecoding(t *testing.T) {
	c := &Config{
		GuidedDecoding: &GuidedDecoding{
			Backend: GuidedBackendXGrammar,
			JSON:    map[string]any{"type": "object"},
		},
	}
	m := c.ToVLLM()
	assert.Equal(t, GuidedBackendXGrammar, m[GuidedKeyDecodingBackend])
	assert.NotNil(t, m[GuidedKeyJSON])
}

func TestConfig_ToVLLM_EmbeddingParams(t *testing.T) {
	c := &Config{
		Embedding: &EmbeddingConfig{
			Normalize:   ptrBool(true),
			PoolingType: EmbeddingPoolingMean,
		},
	}
	m := c.ToVLLM()
	assert.Equal(t, true, m[ParamKeyNormalize])
	assert.Equal(t, EmbeddingPoolingMean, m[ParamKeyPoolingType])
}

func TestConfig_ToVLLM_Streaming(t *testing.T) {
	c := &Config{Streaming: &StreamingConfig{Enabled: true}}
	m := c.ToVLLM()
	assert.Equal(t, true, m[ParamKeyStream])
}

// =============================================================================
// ToMistral
// =============================================================================

func TestConfig_ToMistral_Nil(t *testing.T) {
	var c *Config
	assert.Nil(t, c.ToMistral())
}

func TestConfig_ToMistral_BasicParams(t *testing.T) {
	c := &Config{
		Model:            "mistral-large",
		Temperature:      ptrFloat64(0.7),
		MaxTokens:        ptrInt(1000),
		TopP:             ptrFloat64(0.9),
		StopSequences:    []string{"stop"},
		Seed:             ptrInt(42),
		FrequencyPenalty: ptrFloat64(0.5),
		PresencePenalty:  ptrFloat64(0.3),
	}
	m := c.ToMistral()
	assert.Equal(t, "mistral-large", m[ParamKeyModel])
	assert.Equal(t, 0.7, m[ParamKeyTemperature])
	assert.Equal(t, 1000, m[ParamKeyMaxTokens])
	assert.Equal(t, 0.9, m[ParamKeyTopP])
	assert.Equal(t, 42, m[ParamKeySeed])
	assert.Equal(t, 0.5, m[ParamKeyFrequencyPenalty])
	assert.Equal(t, 0.3, m[ParamKeyPresencePenalty])
}

func TestConfig_ToMistral_ResponseFormat(t *testing.T) {
	c := &Config{
		ResponseFormat: &ResponseFormat{Type: ResponseFormatJSONSchema},
	}
	m := c.ToMistral()
	assert.NotNil(t, m[ParamKeyResponseFormat])
}

func TestConfig_ToMistral_EmbeddingParams(t *testing.T) {
	c := &Config{
		Embedding: &EmbeddingConfig{
			Dimensions:  ptrInt(768),
			Format:      EmbeddingFormatFloat,
			OutputDtype: EmbeddingDtypeFloat32,
		},
	}
	m := c.ToMistral()
	assert.Equal(t, 768, m[ParamKeyOutputDimension])
	assert.Equal(t, EmbeddingFormatFloat, m[ParamKeyEncodingFormat])
	assert.Equal(t, EmbeddingDtypeFloat32, m[ParamKeyOutputDtype])
}

func TestConfig_ToMistral_Streaming(t *testing.T) {
	c := &Config{Streaming: &StreamingConfig{Enabled: true}}
	m := c.ToMistral()
	assert.Equal(t, true, m[ParamKeyStream])
}

// =============================================================================
// ToCohere
// =============================================================================

func TestConfig_ToCohere_Nil(t *testing.T) {
	var c *Config
	assert.Nil(t, c.ToCohere())
}

func TestConfig_ToCohere_BasicParams(t *testing.T) {
	c := &Config{
		Model:         "command-r-plus",
		Temperature:   ptrFloat64(0.7),
		MaxTokens:     ptrInt(1000),
		TopP:          ptrFloat64(0.9),
		TopK:          ptrInt(40),
		StopSequences: []string{"stop"},
		Seed:          ptrInt(42),
	}
	m := c.ToCohere()
	assert.Equal(t, "command-r-plus", m[ParamKeyModel])
	assert.Equal(t, 0.7, m[ParamKeyTemperature])
	assert.Equal(t, 1000, m[ParamKeyMaxTokens])
	assert.Equal(t, 0.9, m[ParamKeyCohereTopP])
	assert.Equal(t, 40, m[ParamKeyCohereTopK])
	assert.Equal(t, []string{"stop"}, m[ParamKeyCohereStopSequences])
	assert.Equal(t, 42, m[ParamKeySeed])
}

func TestConfig_ToCohere_EmbeddingParams(t *testing.T) {
	c := &Config{
		Embedding: &EmbeddingConfig{
			Dimensions:  ptrInt(1024),
			InputType:   EmbeddingInputTypeSearchDocument,
			OutputDtype: EmbeddingDtypeInt8,
			Truncation:  EmbeddingTruncationEnd,
		},
	}
	m := c.ToCohere()
	assert.Equal(t, 1024, m[ParamKeyOutputDimension])
	assert.Equal(t, EmbeddingInputTypeSearchDocument, m[ParamKeyInputType])
	assert.Equal(t, []string{EmbeddingDtypeInt8}, m[ParamKeyEmbeddingTypes])
	assert.Equal(t, CohereTruncateEnd, m[ParamKeyTruncate])
}

func TestConfig_ToCohere_Streaming(t *testing.T) {
	c := &Config{Streaming: &StreamingConfig{Enabled: true}}
	m := c.ToCohere()
	assert.Equal(t, true, m[ParamKeyStream])
}

// =============================================================================
// ProviderFormat
// =============================================================================

func TestConfig_ProviderFormat_NilConfig(t *testing.T) {
	var c *Config
	result, err := c.ProviderFormat(ProviderOpenAI)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestConfig_ProviderFormat_OpenAI(t *testing.T) {
	c := &Config{ResponseFormat: &ResponseFormat{Type: ResponseFormatJSONSchema}}
	result, err := c.ProviderFormat(ProviderOpenAI)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestConfig_ProviderFormat_Azure(t *testing.T) {
	c := &Config{ResponseFormat: &ResponseFormat{Type: ResponseFormatJSONSchema}}
	result, err := c.ProviderFormat(ProviderAzure)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestConfig_ProviderFormat_Anthropic(t *testing.T) {
	c := &Config{ResponseFormat: &ResponseFormat{
		Type: ResponseFormatJSONSchema,
		JSONSchema: &JSONSchemaSpec{
			Name:   "test",
			Schema: map[string]any{"type": "object"},
		},
	}}
	result, err := c.ProviderFormat(ProviderAnthropic)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestConfig_ProviderFormat_Gemini(t *testing.T) {
	c := &Config{ResponseFormat: &ResponseFormat{Type: ResponseFormatJSONSchema}}
	for _, prov := range []string{ProviderGoogle, ProviderGemini, ProviderVertex} {
		result, err := c.ProviderFormat(prov)
		require.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestConfig_ProviderFormat_VLLM(t *testing.T) {
	c := &Config{GuidedDecoding: &GuidedDecoding{Backend: GuidedBackendXGrammar}}
	result, err := c.ProviderFormat(ProviderVLLM)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestConfig_ProviderFormat_Mistral(t *testing.T) {
	c := &Config{ResponseFormat: &ResponseFormat{Type: ResponseFormatJSONSchema}}
	result, err := c.ProviderFormat(ProviderMistral)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestConfig_ProviderFormat_Cohere(t *testing.T) {
	c := &Config{}
	result, err := c.ProviderFormat(ProviderCohere)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestConfig_ProviderFormat_Unknown(t *testing.T) {
	c := &Config{}
	_, err := c.ProviderFormat("unknown_provider")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgSchemaUnsupportedProvider)
}

func TestConfig_ProviderFormat_NilResponseFormat(t *testing.T) {
	c := &Config{}
	result, err := c.ProviderFormat(ProviderOpenAI)
	require.NoError(t, err)
	assert.Nil(t, result)
}

// =============================================================================
// JSON / YAML round-trip
// =============================================================================

func TestConfig_JSON_Nil(t *testing.T) {
	var c *Config
	s, err := c.JSON()
	require.NoError(t, err)
	assert.Equal(t, "", s)
}

func TestConfig_JSON_RoundTrip(t *testing.T) {
	c := &Config{
		Provider:    "openai",
		Model:       "gpt-4",
		Temperature: ptrFloat64(0.7),
		MaxTokens:   ptrInt(1000),
	}
	jsonStr, err := c.JSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonStr)

	var parsed Config
	err = json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)
	assert.Equal(t, c.Provider, parsed.Provider)
	assert.Equal(t, c.Model, parsed.Model)
	assert.Equal(t, *c.Temperature, *parsed.Temperature)
}

func TestConfig_YAML_Nil(t *testing.T) {
	var c *Config
	s, err := c.YAML()
	require.NoError(t, err)
	assert.Equal(t, "", s)
}

func TestConfig_YAML_RoundTrip(t *testing.T) {
	c := &Config{
		Provider:    "openai",
		Model:       "gpt-4",
		Temperature: ptrFloat64(0.7),
	}
	yamlStr, err := c.YAML()
	require.NoError(t, err)
	assert.NotEmpty(t, yamlStr)

	var parsed Config
	err = yaml.Unmarshal([]byte(yamlStr), &parsed)
	require.NoError(t, err)
	assert.Equal(t, c.Provider, parsed.Provider)
	assert.Equal(t, *c.Temperature, *parsed.Temperature)
}

// =============================================================================
// ResponseFormat serialization
// =============================================================================

func TestResponseFormat_ToOpenAI(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		var rf *ResponseFormat
		assert.Nil(t, rf.ToOpenAI())
	})
	t.Run("json_schema with strict", func(t *testing.T) {
		rf := &ResponseFormat{
			Type: ResponseFormatJSONSchema,
			JSONSchema: &JSONSchemaSpec{
				Name:   "test",
				Strict: true,
				Schema: map[string]any{
					"type":       "object",
					"properties": map[string]any{"name": map[string]any{"type": "string"}},
				},
			},
		}
		m := rf.ToOpenAI()
		assert.Equal(t, ResponseFormatJSONSchema, m[SchemaKeyType])
		js, ok := m[SchemaKeyJSONSchema].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "test", js[AttrName])
		assert.Equal(t, true, js[SchemaKeyStrict])
	})
	t.Run("enum", func(t *testing.T) {
		rf := &ResponseFormat{
			Type: ResponseFormatEnum,
			Enum: &EnumConstraint{Values: []string{"a", "b"}},
		}
		m := rf.ToOpenAI()
		assert.Equal(t, []string{"a", "b"}, m[SchemaKeyEnum])
	})
}

func TestResponseFormat_ToAnthropic(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		var rf *ResponseFormat
		assert.Nil(t, rf.ToAnthropic())
	})
	t.Run("nil json_schema returns nil", func(t *testing.T) {
		rf := &ResponseFormat{Type: ResponseFormatText}
		assert.Nil(t, rf.ToAnthropic())
	})
	t.Run("with json_schema", func(t *testing.T) {
		rf := &ResponseFormat{
			Type: ResponseFormatJSONSchema,
			JSONSchema: &JSONSchemaSpec{
				Schema: map[string]any{"type": "object"},
			},
		}
		m := rf.ToAnthropic()
		require.NotNil(t, m)
		assert.NotNil(t, m[SchemaKeyFormat])
	})
}

func TestResponseFormat_ToGemini(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		var rf *ResponseFormat
		assert.Nil(t, rf.ToGemini())
	})
	t.Run("with property ordering", func(t *testing.T) {
		rf := &ResponseFormat{
			Type: ResponseFormatJSONSchema,
			JSONSchema: &JSONSchemaSpec{
				Name:             "test",
				Schema:           map[string]any{"type": "object"},
				PropertyOrdering: []string{"a", "b"},
			},
		}
		m := rf.ToGemini()
		js, ok := m[SchemaKeyJSONSchema].(map[string]any)
		require.True(t, ok)
		schema, ok := js[SchemaKeySchema].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, []string{"a", "b"}, schema[SchemaKeyPropertyOrdering])
	})
}

// =============================================================================
// GuidedDecoding serialization
// =============================================================================

func TestGuidedDecoding_ToVLLM(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		var gd *GuidedDecoding
		assert.Nil(t, gd.ToVLLM())
	})
	t.Run("all fields", func(t *testing.T) {
		gd := &GuidedDecoding{
			Backend:           GuidedBackendXGrammar,
			JSON:              map[string]any{"type": "string"},
			Regex:             "^[a-z]+$",
			Choice:            []string{"a", "b"},
			Grammar:           "grammar rule",
			WhitespacePattern: "\\s+",
		}
		m := gd.ToVLLM()
		assert.Equal(t, GuidedBackendXGrammar, m[GuidedKeyDecodingBackend])
		assert.NotNil(t, m[GuidedKeyJSON])
		assert.Equal(t, "^[a-z]+$", m[GuidedKeyRegex])
		assert.Equal(t, []string{"a", "b"}, m[GuidedKeyChoice])
		assert.Equal(t, "grammar rule", m[GuidedKeyGrammar])
		assert.Equal(t, "\\s+", m[GuidedKeyWhitespacePattern])
	})
}

// =============================================================================
// Error constructors
// =============================================================================

func TestNewConfigValidationError(t *testing.T) {
	err := NewConfigValidationError("test message")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test message")
}

func TestNewProviderError(t *testing.T) {
	err := NewProviderError("test message", "openai")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test message")
}
