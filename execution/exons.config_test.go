package execution

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Helpers
// =============================================================================

func ptrFloat64(v float64) *float64 { return &v }
func ptrInt(v int) *int             { return &v }
func ptrBool(v bool) *bool          { return &v }

// =============================================================================
// Config.Validate — Temperature
// =============================================================================

func TestConfig_Validate_NilConfig(t *testing.T) {
	var c *Config
	assert.NoError(t, c.Validate())
}

func TestConfig_Validate_EmptyConfig(t *testing.T) {
	c := &Config{}
	assert.NoError(t, c.Validate())
}

func TestConfig_Validate_Temperature(t *testing.T) {
	tests := []struct {
		name    string
		temp    *float64
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"zero ok", ptrFloat64(0.0), false},
		{"max ok", ptrFloat64(2.0), false},
		{"mid ok", ptrFloat64(0.7), false},
		{"below min fail", ptrFloat64(-0.1), true},
		{"above max fail", ptrFloat64(2.1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Temperature: tt.temp}
			err := c.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgTemperatureOutOfRange)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Config.Validate — TopP
// =============================================================================

func TestConfig_Validate_TopP(t *testing.T) {
	tests := []struct {
		name    string
		val     *float64
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"zero ok", ptrFloat64(0.0), false},
		{"one ok", ptrFloat64(1.0), false},
		{"below min fail", ptrFloat64(-0.1), true},
		{"above max fail", ptrFloat64(1.1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{TopP: tt.val}
			err := c.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgTopPOutOfRange)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Config.Validate — MaxTokens
// =============================================================================

func TestConfig_Validate_MaxTokens(t *testing.T) {
	tests := []struct {
		name    string
		val     *int
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"1 ok", ptrInt(1), false},
		{"1000 ok", ptrInt(1000), false},
		{"0 fail", ptrInt(0), true},
		{"-1 fail", ptrInt(-1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{MaxTokens: tt.val}
			err := c.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgMaxTokensInvalid)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Config.Validate — TopK
// =============================================================================

func TestConfig_Validate_TopK(t *testing.T) {
	tests := []struct {
		name    string
		val     *int
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"0 ok", ptrInt(0), false},
		{"10 ok", ptrInt(10), false},
		{"-1 fail", ptrInt(-1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{TopK: tt.val}
			err := c.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgTopKInvalid)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Config.Validate — MinP
// =============================================================================

func TestConfig_Validate_MinP(t *testing.T) {
	tests := []struct {
		name    string
		val     *float64
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"zero ok", ptrFloat64(0.0), false},
		{"one ok", ptrFloat64(1.0), false},
		{"below min fail", ptrFloat64(-0.1), true},
		{"above max fail", ptrFloat64(1.1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{MinP: tt.val}
			err := c.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgMinPOutOfRange)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Config.Validate — RepetitionPenalty
// =============================================================================

func TestConfig_Validate_RepetitionPenalty(t *testing.T) {
	tests := []struct {
		name    string
		val     *float64
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"0.1 ok", ptrFloat64(0.1), false},
		{"2.0 ok", ptrFloat64(2.0), false},
		{"0.0 fail", ptrFloat64(0.0), true},
		{"-1.0 fail", ptrFloat64(-1.0), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{RepetitionPenalty: tt.val}
			err := c.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgRepetitionPenaltyOutOfRange)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Config.Validate — Logprobs
// =============================================================================

func TestConfig_Validate_Logprobs(t *testing.T) {
	tests := []struct {
		name    string
		val     *int
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"0 ok", ptrInt(0), false},
		{"20 ok", ptrInt(20), false},
		{"-1 fail", ptrInt(-1), true},
		{"21 fail", ptrInt(21), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Logprobs: tt.val}
			err := c.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgLogprobsOutOfRange)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Config.Validate — StopTokenIDs
// =============================================================================

func TestConfig_Validate_StopTokenIDs(t *testing.T) {
	tests := []struct {
		name    string
		val     []int
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"valid ok", []int{0, 1, 100}, false},
		{"negative fail", []int{-1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{StopTokenIDs: tt.val}
			err := c.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgStopTokenIDNegative)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Config.Validate — LogitBias
// =============================================================================

func TestConfig_Validate_LogitBias(t *testing.T) {
	tests := []struct {
		name    string
		val     map[string]float64
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"valid ok", map[string]float64{"0": 50.0}, false},
		{"boundary ok", map[string]float64{"0": -100.0, "1": 100.0}, false},
		{"above max fail", map[string]float64{"0": 101.0}, true},
		{"below min fail", map[string]float64{"0": -101.0}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{LogitBias: tt.val}
			err := c.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgLogitBiasValueOutOfRange)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Config.Validate — FrequencyPenalty
// =============================================================================

func TestConfig_Validate_FrequencyPenalty(t *testing.T) {
	tests := []struct {
		name    string
		val     *float64
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"-2.0 ok", ptrFloat64(-2.0), false},
		{"2.0 ok", ptrFloat64(2.0), false},
		{"0.0 ok", ptrFloat64(0.0), false},
		{"-2.1 fail", ptrFloat64(-2.1), true},
		{"2.1 fail", ptrFloat64(2.1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{FrequencyPenalty: tt.val}
			err := c.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgFrequencyPenaltyOutOfRange)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Config.Validate — PresencePenalty
// =============================================================================

func TestConfig_Validate_PresencePenalty(t *testing.T) {
	tests := []struct {
		name    string
		val     *float64
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"-2.0 ok", ptrFloat64(-2.0), false},
		{"2.0 ok", ptrFloat64(2.0), false},
		{"-2.1 fail", ptrFloat64(-2.1), true},
		{"2.1 fail", ptrFloat64(2.1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{PresencePenalty: tt.val}
			err := c.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgPresencePenaltyOutOfRange)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Config.Validate — N
// =============================================================================

func TestConfig_Validate_N(t *testing.T) {
	tests := []struct {
		name    string
		val     *int
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"1 ok", ptrInt(1), false},
		{"128 ok", ptrInt(128), false},
		{"0 fail", ptrInt(0), true},
		{"129 fail", ptrInt(129), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{N: tt.val}
			err := c.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgNOutOfRange)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Config.Validate — MaxCompletionTokens
// =============================================================================

func TestConfig_Validate_MaxCompletionTokens(t *testing.T) {
	tests := []struct {
		name    string
		val     *int
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"1 ok", ptrInt(1), false},
		{"0 fail", ptrInt(0), true},
		{"-1 fail", ptrInt(-1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{MaxCompletionTokens: tt.val}
			err := c.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgMaxCompletionTokensInvalid)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Config.Validate — ReasoningEffort
// =============================================================================

func TestConfig_Validate_ReasoningEffort(t *testing.T) {
	tests := []struct {
		name    string
		val     string
		wantErr bool
	}{
		{"empty ok", "", false},
		{"low ok", ReasoningEffortLow, false},
		{"medium ok", ReasoningEffortMedium, false},
		{"high ok", ReasoningEffortHigh, false},
		{"max ok", ReasoningEffortMax, false},
		{"invalid fail", "invalid", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{ReasoningEffort: tt.val}
			err := c.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgReasoningEffortInvalid)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Config.Validate — TopA
// =============================================================================

func TestConfig_Validate_TopA(t *testing.T) {
	tests := []struct {
		name    string
		val     *float64
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"0.0 ok", ptrFloat64(0.0), false},
		{"1.0 ok", ptrFloat64(1.0), false},
		{"-0.1 fail", ptrFloat64(-0.1), true},
		{"1.1 fail", ptrFloat64(1.1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{TopA: tt.val}
			err := c.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgTopAOutOfRange)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Config.Validate — ServiceTier
// =============================================================================

func TestConfig_Validate_ServiceTier(t *testing.T) {
	tests := []struct {
		name    string
		val     string
		wantErr bool
	}{
		{"empty ok", "", false},
		{"auto ok", ServiceTierAuto, false},
		{"default ok", ServiceTierDefault, false},
		{"invalid fail", "invalid", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{ServiceTier: tt.val}
			err := c.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgServiceTierInvalid)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Config.Validate — Thinking
// =============================================================================

func TestConfig_Validate_Thinking(t *testing.T) {
	t.Run("nil ok", func(t *testing.T) {
		c := &Config{}
		assert.NoError(t, c.Validate())
	})

	t.Run("enabled with valid budget ok", func(t *testing.T) {
		c := &Config{Thinking: &ThinkingConfig{Enabled: true, BudgetTokens: ptrInt(1000)}}
		assert.NoError(t, c.Validate())
	})

	t.Run("enabled with nil budget ok", func(t *testing.T) {
		c := &Config{Thinking: &ThinkingConfig{Enabled: true}}
		assert.NoError(t, c.Validate())
	})

	t.Run("disabled with invalid budget ok (not checked)", func(t *testing.T) {
		c := &Config{Thinking: &ThinkingConfig{Enabled: false, BudgetTokens: ptrInt(-1)}}
		assert.NoError(t, c.Validate())
	})

	t.Run("enabled with zero budget fail", func(t *testing.T) {
		c := &Config{Thinking: &ThinkingConfig{Enabled: true, BudgetTokens: ptrInt(0)}}
		err := c.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgThinkingBudgetInvalid)
	})

	t.Run("enabled with negative budget fail", func(t *testing.T) {
		c := &Config{Thinking: &ThinkingConfig{Enabled: true, BudgetTokens: ptrInt(-1)}}
		err := c.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgThinkingBudgetInvalid)
	})
}

// =============================================================================
// Config.Validate — Modality
// =============================================================================

func TestConfig_Validate_Modality(t *testing.T) {
	tests := []struct {
		name    string
		val     string
		wantErr bool
	}{
		{"empty ok", "", false},
		{"text ok", ModalityText, false},
		{"image ok", ModalityImage, false},
		{"embedding ok", ModalityEmbedding, false},
		{"video ok", ModalityVideo, false},
		{"image_edit ok", ModalityImageEdit, false},
		{"invalid fail", "invalid", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Modality: tt.val}
			err := c.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgInvalidModality)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Config.Validate — delegates to sub-configs
// =============================================================================

func TestConfig_Validate_Image(t *testing.T) {
	t.Run("nil ok", func(t *testing.T) {
		c := &Config{}
		assert.NoError(t, c.Validate())
	})
	t.Run("invalid image fails", func(t *testing.T) {
		c := &Config{Image: &ImageConfig{Width: ptrInt(0)}}
		assert.Error(t, c.Validate())
	})
	t.Run("valid image ok", func(t *testing.T) {
		c := &Config{Image: &ImageConfig{Width: ptrInt(1024)}}
		assert.NoError(t, c.Validate())
	})
}

func TestConfig_Validate_Audio(t *testing.T) {
	t.Run("nil ok", func(t *testing.T) {
		c := &Config{}
		assert.NoError(t, c.Validate())
	})
	t.Run("invalid audio fails", func(t *testing.T) {
		c := &Config{Audio: &AudioConfig{Speed: ptrFloat64(0.1)}}
		assert.Error(t, c.Validate())
	})
}

func TestConfig_Validate_Embedding(t *testing.T) {
	t.Run("nil ok", func(t *testing.T) {
		c := &Config{}
		assert.NoError(t, c.Validate())
	})
	t.Run("invalid embedding fails", func(t *testing.T) {
		c := &Config{Embedding: &EmbeddingConfig{Dimensions: ptrInt(0)}}
		assert.Error(t, c.Validate())
	})
}

func TestConfig_Validate_Streaming(t *testing.T) {
	t.Run("nil ok", func(t *testing.T) {
		c := &Config{}
		assert.NoError(t, c.Validate())
	})
	t.Run("invalid streaming fails", func(t *testing.T) {
		c := &Config{Streaming: &StreamingConfig{Enabled: true, Method: "invalid"}}
		assert.Error(t, c.Validate())
	})
	t.Run("disabled with invalid method ok", func(t *testing.T) {
		c := &Config{Streaming: &StreamingConfig{Enabled: false, Method: "invalid"}}
		assert.NoError(t, c.Validate())
	})
}

func TestConfig_Validate_Async(t *testing.T) {
	t.Run("nil ok", func(t *testing.T) {
		c := &Config{}
		assert.NoError(t, c.Validate())
	})
	t.Run("invalid async fails", func(t *testing.T) {
		c := &Config{Async: &AsyncConfig{PollIntervalSeconds: ptrFloat64(0)}}
		assert.Error(t, c.Validate())
	})
}

// =============================================================================
// Sub-type Validate — ImageConfig
// =============================================================================

func TestImageConfig_Validate(t *testing.T) {
	t.Run("nil ok", func(t *testing.T) {
		var c *ImageConfig
		assert.NoError(t, c.Validate())
	})
	t.Run("empty ok", func(t *testing.T) {
		c := &ImageConfig{}
		assert.NoError(t, c.Validate())
	})
	t.Run("width valid range", func(t *testing.T) {
		assert.NoError(t, (&ImageConfig{Width: ptrInt(1)}).Validate())
		assert.NoError(t, (&ImageConfig{Width: ptrInt(ImageMaxWidth)}).Validate())
		assert.Error(t, (&ImageConfig{Width: ptrInt(0)}).Validate())
		assert.Error(t, (&ImageConfig{Width: ptrInt(ImageMaxWidth + 1)}).Validate())
	})
	t.Run("height valid range", func(t *testing.T) {
		assert.NoError(t, (&ImageConfig{Height: ptrInt(1)}).Validate())
		assert.NoError(t, (&ImageConfig{Height: ptrInt(ImageMaxHeight)}).Validate())
		assert.Error(t, (&ImageConfig{Height: ptrInt(0)}).Validate())
		assert.Error(t, (&ImageConfig{Height: ptrInt(ImageMaxHeight + 1)}).Validate())
	})
	t.Run("quality enum", func(t *testing.T) {
		assert.NoError(t, (&ImageConfig{Quality: ImageQualityStandard}).Validate())
		assert.NoError(t, (&ImageConfig{Quality: ImageQualityHD}).Validate())
		assert.NoError(t, (&ImageConfig{Quality: ImageQualityHigh}).Validate())
		assert.Error(t, (&ImageConfig{Quality: "ultra"}).Validate())
	})
	t.Run("style enum", func(t *testing.T) {
		assert.NoError(t, (&ImageConfig{Style: ImageStyleNatural}).Validate())
		assert.NoError(t, (&ImageConfig{Style: ImageStyleVivid}).Validate())
		assert.Error(t, (&ImageConfig{Style: "abstract"}).Validate())
	})
	t.Run("num_images range", func(t *testing.T) {
		assert.NoError(t, (&ImageConfig{NumImages: ptrInt(1)}).Validate())
		assert.NoError(t, (&ImageConfig{NumImages: ptrInt(ImageMaxNumImages)}).Validate())
		assert.Error(t, (&ImageConfig{NumImages: ptrInt(0)}).Validate())
		assert.Error(t, (&ImageConfig{NumImages: ptrInt(ImageMaxNumImages + 1)}).Validate())
	})
	t.Run("guidance_scale range", func(t *testing.T) {
		assert.NoError(t, (&ImageConfig{GuidanceScale: ptrFloat64(0.0)}).Validate())
		assert.NoError(t, (&ImageConfig{GuidanceScale: ptrFloat64(ImageMaxGuidanceScale)}).Validate())
		assert.Error(t, (&ImageConfig{GuidanceScale: ptrFloat64(-1.0)}).Validate())
		assert.Error(t, (&ImageConfig{GuidanceScale: ptrFloat64(ImageMaxGuidanceScale + 1)}).Validate())
	})
	t.Run("steps range", func(t *testing.T) {
		assert.NoError(t, (&ImageConfig{Steps: ptrInt(1)}).Validate())
		assert.NoError(t, (&ImageConfig{Steps: ptrInt(ImageMaxSteps)}).Validate())
		assert.Error(t, (&ImageConfig{Steps: ptrInt(0)}).Validate())
		assert.Error(t, (&ImageConfig{Steps: ptrInt(ImageMaxSteps + 1)}).Validate())
	})
	t.Run("strength range", func(t *testing.T) {
		assert.NoError(t, (&ImageConfig{Strength: ptrFloat64(0.0)}).Validate())
		assert.NoError(t, (&ImageConfig{Strength: ptrFloat64(1.0)}).Validate())
		assert.Error(t, (&ImageConfig{Strength: ptrFloat64(-0.1)}).Validate())
		assert.Error(t, (&ImageConfig{Strength: ptrFloat64(1.1)}).Validate())
	})
}

// =============================================================================
// Sub-type Validate — AudioConfig
// =============================================================================

func TestAudioConfig_Validate(t *testing.T) {
	t.Run("nil ok", func(t *testing.T) {
		var c *AudioConfig
		assert.NoError(t, c.Validate())
	})
	t.Run("speed range", func(t *testing.T) {
		assert.NoError(t, (&AudioConfig{Speed: ptrFloat64(AudioMinSpeed)}).Validate())
		assert.NoError(t, (&AudioConfig{Speed: ptrFloat64(AudioMaxSpeed)}).Validate())
		assert.Error(t, (&AudioConfig{Speed: ptrFloat64(AudioMinSpeed - 0.01)}).Validate())
		assert.Error(t, (&AudioConfig{Speed: ptrFloat64(AudioMaxSpeed + 0.01)}).Validate())
	})
	t.Run("output format enum", func(t *testing.T) {
		assert.NoError(t, (&AudioConfig{OutputFormat: AudioFormatMP3}).Validate())
		assert.NoError(t, (&AudioConfig{OutputFormat: AudioFormatOpus}).Validate())
		assert.NoError(t, (&AudioConfig{OutputFormat: AudioFormatPCM}).Validate())
		assert.Error(t, (&AudioConfig{OutputFormat: "ogg"}).Validate())
	})
	t.Run("duration range", func(t *testing.T) {
		assert.NoError(t, (&AudioConfig{Duration: ptrFloat64(1.0)}).Validate())
		assert.NoError(t, (&AudioConfig{Duration: ptrFloat64(AudioMaxDuration)}).Validate())
		assert.Error(t, (&AudioConfig{Duration: ptrFloat64(0.0)}).Validate())
		assert.Error(t, (&AudioConfig{Duration: ptrFloat64(AudioMaxDuration + 1)}).Validate())
	})
}

// =============================================================================
// Sub-type Validate — EmbeddingConfig
// =============================================================================

func TestEmbeddingConfig_Validate(t *testing.T) {
	t.Run("nil ok", func(t *testing.T) {
		var c *EmbeddingConfig
		assert.NoError(t, c.Validate())
	})
	t.Run("dimensions range", func(t *testing.T) {
		assert.NoError(t, (&EmbeddingConfig{Dimensions: ptrInt(1)}).Validate())
		assert.NoError(t, (&EmbeddingConfig{Dimensions: ptrInt(EmbeddingMaxDimensions)}).Validate())
		assert.Error(t, (&EmbeddingConfig{Dimensions: ptrInt(0)}).Validate())
		assert.Error(t, (&EmbeddingConfig{Dimensions: ptrInt(EmbeddingMaxDimensions + 1)}).Validate())
	})
	t.Run("format enum", func(t *testing.T) {
		assert.NoError(t, (&EmbeddingConfig{Format: EmbeddingFormatFloat}).Validate())
		assert.NoError(t, (&EmbeddingConfig{Format: EmbeddingFormatBase64}).Validate())
		assert.Error(t, (&EmbeddingConfig{Format: "raw"}).Validate())
	})
	t.Run("input_type enum", func(t *testing.T) {
		assert.NoError(t, (&EmbeddingConfig{InputType: EmbeddingInputTypeSearchQuery}).Validate())
		assert.NoError(t, (&EmbeddingConfig{InputType: EmbeddingInputTypeClustering}).Validate())
		assert.Error(t, (&EmbeddingConfig{InputType: "unknown"}).Validate())
	})
	t.Run("output_dtype enum", func(t *testing.T) {
		assert.NoError(t, (&EmbeddingConfig{OutputDtype: EmbeddingDtypeFloat32}).Validate())
		assert.NoError(t, (&EmbeddingConfig{OutputDtype: EmbeddingDtypeBinary}).Validate())
		assert.Error(t, (&EmbeddingConfig{OutputDtype: "float16"}).Validate())
	})
	t.Run("truncation enum", func(t *testing.T) {
		assert.NoError(t, (&EmbeddingConfig{Truncation: EmbeddingTruncationNone}).Validate())
		assert.NoError(t, (&EmbeddingConfig{Truncation: EmbeddingTruncationEnd}).Validate())
		assert.Error(t, (&EmbeddingConfig{Truncation: "middle"}).Validate())
	})
	t.Run("pooling_type enum", func(t *testing.T) {
		assert.NoError(t, (&EmbeddingConfig{PoolingType: EmbeddingPoolingMean}).Validate())
		assert.NoError(t, (&EmbeddingConfig{PoolingType: EmbeddingPoolingCLS}).Validate())
		assert.Error(t, (&EmbeddingConfig{PoolingType: "max"}).Validate())
	})
}

// =============================================================================
// Sub-type Validate — StreamingConfig
// =============================================================================

func TestStreamingConfig_Validate(t *testing.T) {
	t.Run("nil ok", func(t *testing.T) {
		var c *StreamingConfig
		assert.NoError(t, c.Validate())
	})
	t.Run("enabled with valid method ok", func(t *testing.T) {
		assert.NoError(t, (&StreamingConfig{Enabled: true, Method: StreamMethodSSE}).Validate())
		assert.NoError(t, (&StreamingConfig{Enabled: true, Method: StreamMethodWebSocket}).Validate())
	})
	t.Run("enabled with empty method ok", func(t *testing.T) {
		assert.NoError(t, (&StreamingConfig{Enabled: true}).Validate())
	})
	t.Run("enabled with invalid method fail", func(t *testing.T) {
		assert.Error(t, (&StreamingConfig{Enabled: true, Method: "grpc"}).Validate())
	})
	t.Run("disabled with invalid method ok", func(t *testing.T) {
		assert.NoError(t, (&StreamingConfig{Enabled: false, Method: "grpc"}).Validate())
	})
}

// =============================================================================
// Sub-type Validate — AsyncConfig
// =============================================================================

func TestAsyncConfig_Validate(t *testing.T) {
	t.Run("nil ok", func(t *testing.T) {
		var c *AsyncConfig
		assert.NoError(t, c.Validate())
	})
	t.Run("poll_interval must be positive", func(t *testing.T) {
		assert.Error(t, (&AsyncConfig{PollIntervalSeconds: ptrFloat64(0)}).Validate())
		assert.Error(t, (&AsyncConfig{PollIntervalSeconds: ptrFloat64(-1)}).Validate())
		assert.NoError(t, (&AsyncConfig{PollIntervalSeconds: ptrFloat64(1.0)}).Validate())
	})
	t.Run("poll_timeout must be positive", func(t *testing.T) {
		assert.Error(t, (&AsyncConfig{PollTimeoutSeconds: ptrFloat64(0)}).Validate())
		assert.NoError(t, (&AsyncConfig{PollTimeoutSeconds: ptrFloat64(1.0)}).Validate())
	})
	t.Run("poll_timeout must be >= poll_interval", func(t *testing.T) {
		assert.Error(t, (&AsyncConfig{
			PollIntervalSeconds: ptrFloat64(10.0),
			PollTimeoutSeconds:  ptrFloat64(5.0),
		}).Validate())
		assert.NoError(t, (&AsyncConfig{
			PollIntervalSeconds: ptrFloat64(5.0),
			PollTimeoutSeconds:  ptrFloat64(5.0),
		}).Validate())
		assert.NoError(t, (&AsyncConfig{
			PollIntervalSeconds: ptrFloat64(5.0),
			PollTimeoutSeconds:  ptrFloat64(10.0),
		}).Validate())
	})
}

// =============================================================================
// Config.Clone
// =============================================================================

func TestConfig_Clone_Nil(t *testing.T) {
	var c *Config
	assert.Nil(t, c.Clone())
}

func TestConfig_Clone_ScalarFields(t *testing.T) {
	c := &Config{
		Provider:        "openai",
		Model:           "gpt-4",
		ReasoningEffort: ReasoningEffortHigh,
		User:            "user-123",
		ServiceTier:     ServiceTierAuto,
		Modality:        ModalityText,
	}
	clone := c.Clone()
	assert.Equal(t, c.Provider, clone.Provider)
	assert.Equal(t, c.Model, clone.Model)
	assert.Equal(t, c.ReasoningEffort, clone.ReasoningEffort)
	assert.Equal(t, c.User, clone.User)
	assert.Equal(t, c.ServiceTier, clone.ServiceTier)
	assert.Equal(t, c.Modality, clone.Modality)
}

func TestConfig_Clone_PointerFieldsIndependent(t *testing.T) {
	c := &Config{
		Temperature:         ptrFloat64(0.7),
		MaxTokens:           ptrInt(1000),
		TopP:                ptrFloat64(0.9),
		TopK:                ptrInt(40),
		MinP:                ptrFloat64(0.1),
		RepetitionPenalty:   ptrFloat64(1.2),
		Seed:                ptrInt(42),
		Logprobs:            ptrInt(5),
		FrequencyPenalty:    ptrFloat64(0.5),
		PresencePenalty:     ptrFloat64(0.3),
		N:                   ptrInt(2),
		MaxCompletionTokens: ptrInt(4096),
		TopA:                ptrFloat64(0.5),
		Store:               ptrBool(true),
	}
	clone := c.Clone()

	// Modify original
	*c.Temperature = 1.0
	*c.MaxTokens = 2000
	*c.TopP = 0.1
	*c.MinP = 0.5

	// Clone should be unaffected
	assert.Equal(t, 0.7, *clone.Temperature)
	assert.Equal(t, 1000, *clone.MaxTokens)
	assert.Equal(t, 0.9, *clone.TopP)
	assert.Equal(t, 0.1, *clone.MinP)
}

func TestConfig_Clone_SliceFieldsIndependent(t *testing.T) {
	c := &Config{
		StopSequences: []string{"stop1", "stop2"},
		StopTokenIDs:  []int{100, 200},
	}
	clone := c.Clone()

	c.StopSequences[0] = "changed"
	c.StopTokenIDs[0] = 999

	assert.Equal(t, "stop1", clone.StopSequences[0])
	assert.Equal(t, 100, clone.StopTokenIDs[0])
}

func TestConfig_Clone_MapFieldsIndependent(t *testing.T) {
	c := &Config{
		LogitBias:       map[string]float64{"0": 1.0, "1": -1.0},
		ProviderOptions: map[string]any{"key": "value"},
	}
	clone := c.Clone()

	c.LogitBias["0"] = 99.0
	c.ProviderOptions["key"] = "changed"

	assert.Equal(t, 1.0, clone.LogitBias["0"])
	assert.Equal(t, "value", clone.ProviderOptions["key"])
}

func TestConfig_Clone_NestedConfigs(t *testing.T) {
	c := &Config{
		Thinking: &ThinkingConfig{Enabled: true, BudgetTokens: ptrInt(1000)},
		ResponseFormat: &ResponseFormat{
			Type: ResponseFormatJSONSchema,
			JSONSchema: &JSONSchemaSpec{
				Name:   "test",
				Schema: map[string]any{"type": "object"},
			},
		},
		GuidedDecoding: &GuidedDecoding{
			Backend: GuidedBackendXGrammar,
			JSON:    map[string]any{"type": "string"},
		},
		Image:     &ImageConfig{Width: ptrInt(1024), Quality: ImageQualityHD},
		Audio:     &AudioConfig{Voice: "alloy", Speed: ptrFloat64(1.0)},
		Embedding: &EmbeddingConfig{Dimensions: ptrInt(768), Format: EmbeddingFormatFloat},
		Streaming: &StreamingConfig{Enabled: true, Method: StreamMethodSSE},
		Async:     &AsyncConfig{Enabled: true, PollIntervalSeconds: ptrFloat64(5.0)},
	}
	clone := c.Clone()

	// Modify originals
	c.Thinking.Enabled = false
	c.ResponseFormat.Type = "changed"
	c.GuidedDecoding.Backend = "changed"
	*c.Image.Width = 512
	c.Audio.Voice = "changed"
	*c.Embedding.Dimensions = 256
	c.Streaming.Enabled = false
	c.Async.Enabled = false

	// Clone should be unaffected
	assert.True(t, clone.Thinking.Enabled)
	assert.Equal(t, ResponseFormatJSONSchema, clone.ResponseFormat.Type)
	assert.Equal(t, GuidedBackendXGrammar, clone.GuidedDecoding.Backend)
	assert.Equal(t, 1024, *clone.Image.Width)
	assert.Equal(t, "alloy", clone.Audio.Voice)
	assert.Equal(t, 768, *clone.Embedding.Dimensions)
	assert.True(t, clone.Streaming.Enabled)
	assert.True(t, clone.Async.Enabled)
}

// =============================================================================
// Config.Merge
// =============================================================================

func TestConfig_Merge_NilNil(t *testing.T) {
	var a, b *Config
	assert.Nil(t, a.Merge(b))
}

func TestConfig_Merge_NilOther(t *testing.T) {
	var a *Config
	b := &Config{Provider: "openai", Temperature: ptrFloat64(0.7)}
	result := a.Merge(b)
	require.NotNil(t, result)
	assert.Equal(t, "openai", result.Provider)
	assert.Equal(t, 0.7, *result.Temperature)
	// Verify it is a clone (independent)
	*b.Temperature = 1.0
	assert.Equal(t, 0.7, *result.Temperature)
}

func TestConfig_Merge_BaseNil(t *testing.T) {
	a := &Config{Provider: "openai"}
	var b *Config
	result := a.Merge(b)
	require.NotNil(t, result)
	assert.Equal(t, "openai", result.Provider)
}

func TestConfig_Merge_StringOverrides(t *testing.T) {
	base := &Config{Provider: "openai", Model: "gpt-4", ReasoningEffort: ReasoningEffortLow, User: "user1", ServiceTier: ServiceTierAuto}
	other := &Config{Provider: "anthropic", Model: "claude-3", ReasoningEffort: ReasoningEffortHigh, User: "user2", ServiceTier: ServiceTierDefault}
	result := base.Merge(other)
	assert.Equal(t, "anthropic", result.Provider)
	assert.Equal(t, "claude-3", result.Model)
	assert.Equal(t, ReasoningEffortHigh, result.ReasoningEffort)
	assert.Equal(t, "user2", result.User)
	assert.Equal(t, ServiceTierDefault, result.ServiceTier)
}

func TestConfig_Merge_PointerCoalescing(t *testing.T) {
	base := &Config{Temperature: ptrFloat64(0.5), MaxTokens: ptrInt(1000)}
	other := &Config{Temperature: ptrFloat64(0.9)}
	result := base.Merge(other)
	assert.Equal(t, 0.9, *result.Temperature)
	assert.Equal(t, 1000, *result.MaxTokens) // base retained
}

func TestConfig_Merge_SliceReplacement(t *testing.T) {
	base := &Config{StopSequences: []string{"a"}, StopTokenIDs: []int{1}}
	other := &Config{StopSequences: []string{"b", "c"}, StopTokenIDs: []int{2, 3}}
	result := base.Merge(other)
	assert.Equal(t, []string{"b", "c"}, result.StopSequences)
	assert.Equal(t, []int{2, 3}, result.StopTokenIDs)
}

func TestConfig_Merge_MapReplacement(t *testing.T) {
	base := &Config{LogitBias: map[string]float64{"0": 1.0}}
	other := &Config{LogitBias: map[string]float64{"1": -1.0}}
	result := base.Merge(other)
	assert.Equal(t, map[string]float64{"1": -1.0}, result.LogitBias)
}

func TestConfig_Merge_NestedConfigReplacement(t *testing.T) {
	base := &Config{
		Thinking: &ThinkingConfig{Enabled: true, BudgetTokens: ptrInt(1000)},
		Image:    &ImageConfig{Width: ptrInt(512)},
	}
	other := &Config{
		Thinking: &ThinkingConfig{Enabled: false},
		Image:    &ImageConfig{Width: ptrInt(1024)},
	}
	result := base.Merge(other)
	assert.False(t, result.Thinking.Enabled)
	assert.Equal(t, 1024, *result.Image.Width)
}

func TestConfig_Merge_ProviderOptionsMerge(t *testing.T) {
	base := &Config{ProviderOptions: map[string]any{"k1": "v1"}}
	other := &Config{ProviderOptions: map[string]any{"k1": "override", "k2": "v2"}}
	result := base.Merge(other)
	assert.Equal(t, "override", result.ProviderOptions["k1"])
	assert.Equal(t, "v2", result.ProviderOptions["k2"])
}

func TestConfig_Merge_OriginalUnmodified(t *testing.T) {
	base := &Config{Temperature: ptrFloat64(0.5)}
	other := &Config{Temperature: ptrFloat64(0.9)}
	_ = base.Merge(other)
	assert.Equal(t, 0.5, *base.Temperature)
	assert.Equal(t, 0.9, *other.Temperature)
}

// =============================================================================
// GetEffectiveProvider
// =============================================================================

func TestConfig_GetEffectiveProvider(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		var c *Config
		assert.Equal(t, "", c.GetEffectiveProvider())
	})

	t.Run("explicit provider", func(t *testing.T) {
		c := &Config{Provider: "custom"}
		assert.Equal(t, "custom", c.GetEffectiveProvider())
	})

	t.Run("guided decoding hints vllm", func(t *testing.T) {
		c := &Config{GuidedDecoding: &GuidedDecoding{Backend: GuidedBackendXGrammar}}
		assert.Equal(t, ProviderVLLM, c.GetEffectiveProvider())
	})

	t.Run("min_p hints vllm", func(t *testing.T) {
		c := &Config{MinP: ptrFloat64(0.1)}
		assert.Equal(t, ProviderVLLM, c.GetEffectiveProvider())
	})

	t.Run("repetition_penalty hints vllm", func(t *testing.T) {
		c := &Config{RepetitionPenalty: ptrFloat64(1.2)}
		assert.Equal(t, ProviderVLLM, c.GetEffectiveProvider())
	})

	t.Run("stop_token_ids hints vllm", func(t *testing.T) {
		c := &Config{StopTokenIDs: []int{100}}
		assert.Equal(t, ProviderVLLM, c.GetEffectiveProvider())
	})

	t.Run("thinking enabled hints anthropic", func(t *testing.T) {
		c := &Config{Thinking: &ThinkingConfig{Enabled: true}}
		assert.Equal(t, ProviderAnthropic, c.GetEffectiveProvider())
	})

	t.Run("model gpt-4 hints openai", func(t *testing.T) {
		c := &Config{Model: "gpt-4"}
		assert.Equal(t, ProviderOpenAI, c.GetEffectiveProvider())
	})

	t.Run("model claude-3 hints anthropic", func(t *testing.T) {
		c := &Config{Model: "claude-3-opus"}
		assert.Equal(t, ProviderAnthropic, c.GetEffectiveProvider())
	})

	t.Run("model gemini-pro hints gemini", func(t *testing.T) {
		c := &Config{Model: "gemini-pro"}
		assert.Equal(t, ProviderGemini, c.GetEffectiveProvider())
	})

	t.Run("model mistral-large hints mistral", func(t *testing.T) {
		c := &Config{Model: "mistral-large-latest"}
		assert.Equal(t, ProviderMistral, c.GetEffectiveProvider())
	})

	t.Run("model command-r hints cohere", func(t *testing.T) {
		c := &Config{Model: "command-r-plus"}
		assert.Equal(t, ProviderCohere, c.GetEffectiveProvider())
	})

	t.Run("no config returns empty", func(t *testing.T) {
		c := &Config{}
		assert.Equal(t, "", c.GetEffectiveProvider())
	})
}

// =============================================================================
// GetEffectiveProvider tests (the only non-trivial getter — kept)
// =============================================================================

// =============================================================================
// GeminiTaskType
// =============================================================================

func TestGeminiTaskType(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{EmbeddingInputTypeSearchQuery, GeminiTaskRetrievalQuery, false},
		{EmbeddingInputTypeSearchDocument, GeminiTaskRetrievalDocument, false},
		{EmbeddingInputTypeSemanticSimilarity, GeminiTaskSemanticSimilarity, false},
		{EmbeddingInputTypeClassification, GeminiTaskClassification, false},
		{EmbeddingInputTypeClustering, GeminiTaskClustering, false},
		{"invalid", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := GeminiTaskType(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// =============================================================================
// CohereUpperCase
// =============================================================================

func TestCohereUpperCase(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{EmbeddingTruncationNone, CohereTruncateNone, false},
		{EmbeddingTruncationStart, CohereTruncateStart, false},
		{EmbeddingTruncationEnd, CohereTruncateEnd, false},
		{"invalid", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := CohereUpperCase(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// =============================================================================
// Model detection helpers
// =============================================================================

func TestModelDetection(t *testing.T) {
	t.Run("OpenAI models", func(t *testing.T) {
		assert.True(t, isOpenAIModel("gpt-4"))
		assert.True(t, isOpenAIModel("o1-preview"))
		assert.True(t, isOpenAIModel("text-embedding"))
		assert.False(t, isOpenAIModel("claude-3"))
		assert.False(t, isOpenAIModel(""))
	})
	t.Run("Anthropic models", func(t *testing.T) {
		assert.True(t, isAnthropicModel("claude-3"))
		assert.True(t, isAnthropicModel("claude"))
		assert.False(t, isAnthropicModel("gpt-4"))
	})
	t.Run("Gemini models", func(t *testing.T) {
		assert.True(t, isGeminiModel("gemini-pro"))
		assert.True(t, isGeminiModel("gemini"))
		assert.False(t, isGeminiModel("gpt-4"))
	})
	t.Run("Mistral models", func(t *testing.T) {
		assert.True(t, isMistralModel("mistral-large"))
		assert.True(t, isMistralModel("codestral-latest"))
		assert.True(t, isMistralModel("pixtral-12b"))
		assert.True(t, isMistralModel("ministral-8b"))
		assert.True(t, isMistralModel("open-mistral-nemo"))
		assert.True(t, isMistralModel("open-mixtral-8x7b"))
		assert.False(t, isMistralModel("gpt-4"))
	})
	t.Run("Cohere models", func(t *testing.T) {
		assert.True(t, isCohereModel("command-r"))
		assert.True(t, isCohereModel("embed-v4"))
		assert.True(t, isCohereModel("rerank-v3"))
		assert.True(t, isCohereModel("c4ai-aya"))
		assert.False(t, isCohereModel("gpt-4"))
	})
}

// =============================================================================
// ImageConfig.EffectiveSize
// =============================================================================

func TestImageConfig_EffectiveSize(t *testing.T) {
	t.Run("nil returns empty", func(t *testing.T) {
		var c *ImageConfig
		assert.Equal(t, "", c.EffectiveSize())
	})
	t.Run("explicit size wins", func(t *testing.T) {
		c := &ImageConfig{Size: "1024x1024", Width: ptrInt(512), Height: ptrInt(512)}
		assert.Equal(t, "1024x1024", c.EffectiveSize())
	})
	t.Run("derived from width and height", func(t *testing.T) {
		c := &ImageConfig{Width: ptrInt(512), Height: ptrInt(768)}
		assert.Equal(t, "512x768", c.EffectiveSize())
	})
	t.Run("only width returns empty", func(t *testing.T) {
		c := &ImageConfig{Width: ptrInt(512)}
		assert.Equal(t, "", c.EffectiveSize())
	})
	t.Run("neither returns empty", func(t *testing.T) {
		c := &ImageConfig{}
		assert.Equal(t, "", c.EffectiveSize())
	})
}

// =============================================================================
// Sub-config Clone tests
// =============================================================================

func TestImageConfig_Clone(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var c *ImageConfig
		assert.Nil(t, c.Clone())
	})
	t.Run("deep copy", func(t *testing.T) {
		c := &ImageConfig{Width: ptrInt(1024), Height: ptrInt(768), Quality: ImageQualityHD}
		clone := c.Clone()
		*c.Width = 512
		assert.Equal(t, 1024, *clone.Width)
	})
}

func TestAudioConfig_Clone(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var c *AudioConfig
		assert.Nil(t, c.Clone())
	})
	t.Run("deep copy", func(t *testing.T) {
		c := &AudioConfig{Voice: "alloy", Speed: ptrFloat64(1.5)}
		clone := c.Clone()
		*c.Speed = 2.0
		assert.Equal(t, 1.5, *clone.Speed)
	})
}

func TestEmbeddingConfig_Clone(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var c *EmbeddingConfig
		assert.Nil(t, c.Clone())
	})
	t.Run("deep copy", func(t *testing.T) {
		c := &EmbeddingConfig{Dimensions: ptrInt(768), Normalize: ptrBool(true)}
		clone := c.Clone()
		*c.Dimensions = 256
		*c.Normalize = false
		assert.Equal(t, 768, *clone.Dimensions)
		assert.True(t, *clone.Normalize)
	})
}

func TestStreamingConfig_Clone(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var c *StreamingConfig
		assert.Nil(t, c.Clone())
	})
	t.Run("copy", func(t *testing.T) {
		c := &StreamingConfig{Enabled: true, Method: StreamMethodSSE}
		clone := c.Clone()
		c.Enabled = false
		assert.True(t, clone.Enabled)
	})
}

func TestAsyncConfig_Clone(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var c *AsyncConfig
		assert.Nil(t, c.Clone())
	})
	t.Run("deep copy", func(t *testing.T) {
		c := &AsyncConfig{Enabled: true, PollIntervalSeconds: ptrFloat64(5.0), PollTimeoutSeconds: ptrFloat64(60.0)}
		clone := c.Clone()
		*c.PollIntervalSeconds = 10.0
		assert.Equal(t, 5.0, *clone.PollIntervalSeconds)
	})
}

// =============================================================================
// Sub-config ToMap tests
// =============================================================================

func TestImageConfig_ToMap(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		var c *ImageConfig
		assert.Nil(t, c.ToMap())
	})
	t.Run("populated", func(t *testing.T) {
		c := &ImageConfig{Width: ptrInt(1024), Quality: ImageQualityHD, NumImages: ptrInt(2)}
		m := c.ToMap()
		assert.Equal(t, 1024, m[ParamKeyWidth])
		assert.Equal(t, ImageQualityHD, m[ParamKeyImageQuality])
		assert.Equal(t, 2, m[ParamKeyNumImages])
	})
}

func TestAudioConfig_ToMap(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		var c *AudioConfig
		assert.Nil(t, c.ToMap())
	})
	t.Run("populated", func(t *testing.T) {
		c := &AudioConfig{Voice: "alloy", Speed: ptrFloat64(1.5), OutputFormat: AudioFormatMP3}
		m := c.ToMap()
		assert.Equal(t, "alloy", m[ParamKeyVoice])
		assert.Equal(t, 1.5, m[ParamKeySpeed])
		assert.Equal(t, AudioFormatMP3, m[ParamKeyOutputFormat])
	})
}

func TestEmbeddingConfig_ToMap(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		var c *EmbeddingConfig
		assert.Nil(t, c.ToMap())
	})
	t.Run("populated", func(t *testing.T) {
		c := &EmbeddingConfig{
			Dimensions:  ptrInt(768),
			Format:      EmbeddingFormatFloat,
			InputType:   EmbeddingInputTypeSearchQuery,
			OutputDtype: EmbeddingDtypeFloat32,
			Truncation:  EmbeddingTruncationEnd,
			Normalize:   ptrBool(true),
			PoolingType: EmbeddingPoolingMean,
		}
		m := c.ToMap()
		assert.Equal(t, 768, m[ParamKeyDimensions])
		assert.Equal(t, EmbeddingFormatFloat, m[ParamKeyEncodingFormat])
		assert.Equal(t, EmbeddingInputTypeSearchQuery, m[ParamKeyInputType])
		assert.Equal(t, EmbeddingDtypeFloat32, m[ParamKeyOutputDtype])
		assert.Equal(t, EmbeddingTruncationEnd, m[ParamKeyTruncation])
		assert.Equal(t, true, m[ParamKeyNormalize])
		assert.Equal(t, EmbeddingPoolingMean, m[ParamKeyPoolingType])
	})
}

func TestStreamingConfig_ToMap(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		var c *StreamingConfig
		assert.Nil(t, c.ToMap())
	})
	t.Run("populated", func(t *testing.T) {
		c := &StreamingConfig{Enabled: true, Method: StreamMethodSSE}
		m := c.ToMap()
		assert.Equal(t, true, m[ParamKeyEnabled])
		assert.Equal(t, StreamMethodSSE, m[ParamKeyStreamMethod])
	})
}

func TestAsyncConfig_ToMap(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		var c *AsyncConfig
		assert.Nil(t, c.ToMap())
	})
	t.Run("populated", func(t *testing.T) {
		c := &AsyncConfig{Enabled: true, PollIntervalSeconds: ptrFloat64(5.0), PollTimeoutSeconds: ptrFloat64(60.0)}
		m := c.ToMap()
		assert.Equal(t, true, m[ParamKeyEnabled])
		assert.Equal(t, 5.0, m[ParamKeyPollInterval])
		assert.Equal(t, 60.0, m[ParamKeyPollTimeout])
	})
}
