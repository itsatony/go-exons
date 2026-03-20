package execution

import "fmt"

// ThinkingConfig configures extended thinking mode (Anthropic Claude).
type ThinkingConfig struct {
	Enabled      bool `yaml:"enabled" json:"enabled"`
	BudgetTokens *int `yaml:"budget_tokens,omitempty" json:"budget_tokens,omitempty"`
}

// ResponseFormat configures structured output enforcement.
type ResponseFormat struct {
	// Type: "text", "json_object", "json_schema", or "enum"
	Type string `yaml:"type" json:"type"`
	// JSONSchema for structured output validation (when type is "json_schema")
	JSONSchema *JSONSchemaSpec `yaml:"json_schema,omitempty" json:"json_schema,omitempty"`
	// Enum constraint for choice-based outputs (when type is "enum")
	Enum *EnumConstraint `yaml:"enum,omitempty" json:"enum,omitempty"`
}

// JSONSchemaSpec defines a JSON schema for structured outputs.
type JSONSchemaSpec struct {
	// Name of the schema (required for API calls)
	Name string `yaml:"name" json:"name"`
	// Description of what the schema represents
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	// Schema is the JSON schema definition
	Schema map[string]any `yaml:"schema" json:"schema"`
	// Strict enables strict schema validation
	Strict bool `yaml:"strict,omitempty" json:"strict,omitempty"`
	// AdditionalProperties controls whether extra properties are allowed
	AdditionalProperties *bool `yaml:"additionalProperties,omitempty" json:"additionalProperties,omitempty"`
	// PropertyOrdering specifies the order of properties in output (Gemini 2.5+ only)
	PropertyOrdering []string `yaml:"propertyOrdering,omitempty" json:"propertyOrdering,omitempty"`
}

// EnumConstraint defines enum/choice constraints for outputs.
type EnumConstraint struct {
	// Values contains the allowed enum values
	Values []string `yaml:"values" json:"values"`
	// Description explains the enum choices
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// GuidedDecoding configures vLLM's structured output constraints.
type GuidedDecoding struct {
	// Backend specifies the guided decoding engine: "xgrammar", "outlines", "lm_format_enforcer", "auto"
	Backend string `yaml:"backend,omitempty" json:"backend,omitempty"`
	// JSON is a JSON schema for structured output
	JSON map[string]any `yaml:"json,omitempty" json:"json,omitempty"`
	// Regex is a regex pattern constraint
	Regex string `yaml:"regex,omitempty" json:"regex,omitempty"`
	// Choice is a list of allowed output choices
	Choice []string `yaml:"choice,omitempty" json:"choice,omitempty"`
	// Grammar is a context-free grammar constraint
	Grammar string `yaml:"grammar,omitempty" json:"grammar,omitempty"`
	// WhitespacePattern controls whitespace handling
	WhitespacePattern string `yaml:"whitespace_pattern,omitempty" json:"whitespace_pattern,omitempty"`
}

// ImageConfig configures image generation parameters.
// ImageConfig is safe for concurrent reads. Callers must not modify the config after passing
// it to a Config; use Clone() to create an independent copy if mutation is needed.
type ImageConfig struct {
	// Width of the generated image in pixels (1-8192)
	Width *int `yaml:"width,omitempty" json:"width,omitempty"`
	// Height of the generated image in pixels (1-8192)
	Height *int `yaml:"height,omitempty" json:"height,omitempty"`
	// Size is a provider-specific size string (e.g., "1024x1024")
	Size string `yaml:"size,omitempty" json:"size,omitempty"`
	// Quality of the generated image: "standard", "hd", "low", "medium", "high"
	Quality string `yaml:"quality,omitempty" json:"quality,omitempty"`
	// Style of the generated image: "natural", "vivid"
	Style string `yaml:"style,omitempty" json:"style,omitempty"`
	// AspectRatio for the generated image (e.g., "16:9", "1:1")
	AspectRatio string `yaml:"aspect_ratio,omitempty" json:"aspect_ratio,omitempty"`
	// NegativePrompt describes content to avoid in generation
	NegativePrompt string `yaml:"negative_prompt,omitempty" json:"negative_prompt,omitempty"`
	// NumImages is the number of images to generate (1-10)
	NumImages *int `yaml:"num_images,omitempty" json:"num_images,omitempty"`
	// GuidanceScale controls adherence to the prompt (0.0-30.0)
	GuidanceScale *float64 `yaml:"guidance_scale,omitempty" json:"guidance_scale,omitempty"`
	// Steps is the number of diffusion steps (1-200)
	Steps *int `yaml:"steps,omitempty" json:"steps,omitempty"`
	// Strength controls how much to transform a reference image (0.0-1.0)
	Strength *float64 `yaml:"strength,omitempty" json:"strength,omitempty"`
}

// Validate checks the image config for consistency.
func (c *ImageConfig) Validate() error {
	if c == nil {
		return nil
	}

	if c.Width != nil {
		if *c.Width < 1 || *c.Width > ImageMaxWidth {
			return NewConfigValidationError(ErrMsgImageWidthOutOfRange)
		}
	}

	if c.Height != nil {
		if *c.Height < 1 || *c.Height > ImageMaxHeight {
			return NewConfigValidationError(ErrMsgImageHeightOutOfRange)
		}
	}

	if c.Quality != "" && !isValidImageQuality(c.Quality) {
		return NewConfigValidationError(ErrMsgImageInvalidQuality)
	}

	if c.Style != "" && !isValidImageStyle(c.Style) {
		return NewConfigValidationError(ErrMsgImageInvalidStyle)
	}

	if c.NumImages != nil {
		if *c.NumImages < 1 || *c.NumImages > ImageMaxNumImages {
			return NewConfigValidationError(ErrMsgImageNumImagesOutOfRange)
		}
	}

	if c.GuidanceScale != nil {
		if *c.GuidanceScale < 0.0 || *c.GuidanceScale > ImageMaxGuidanceScale {
			return NewConfigValidationError(ErrMsgImageGuidanceScaleOutOfRange)
		}
	}

	if c.Steps != nil {
		if *c.Steps < 1 || *c.Steps > ImageMaxSteps {
			return NewConfigValidationError(ErrMsgImageStepsOutOfRange)
		}
	}

	if c.Strength != nil {
		if *c.Strength < 0.0 || *c.Strength > 1.0 {
			return NewConfigValidationError(ErrMsgImageStrengthOutOfRange)
		}
	}

	return nil
}

// Clone creates a deep copy of the image config.
func (c *ImageConfig) Clone() *ImageConfig {
	if c == nil {
		return nil
	}

	clone := &ImageConfig{
		Size:           c.Size,
		Quality:        c.Quality,
		Style:          c.Style,
		AspectRatio:    c.AspectRatio,
		NegativePrompt: c.NegativePrompt,
	}

	if c.Width != nil {
		v := *c.Width
		clone.Width = &v
	}
	if c.Height != nil {
		v := *c.Height
		clone.Height = &v
	}
	if c.NumImages != nil {
		v := *c.NumImages
		clone.NumImages = &v
	}
	if c.GuidanceScale != nil {
		v := *c.GuidanceScale
		clone.GuidanceScale = &v
	}
	if c.Steps != nil {
		v := *c.Steps
		clone.Steps = &v
	}
	if c.Strength != nil {
		v := *c.Strength
		clone.Strength = &v
	}

	return clone
}

// ToMap converts the image config to a parameter map.
func (c *ImageConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := make(map[string]any)

	if c.Width != nil {
		result[ParamKeyWidth] = *c.Width
	}
	if c.Height != nil {
		result[ParamKeyHeight] = *c.Height
	}
	if c.Size != "" {
		result[ParamKeyImageSize] = c.Size
	}
	if c.Quality != "" {
		result[ParamKeyImageQuality] = c.Quality
	}
	if c.Style != "" {
		result[ParamKeyImageStyle] = c.Style
	}
	if c.AspectRatio != "" {
		result[ParamKeyAspectRatio] = c.AspectRatio
	}
	if c.NegativePrompt != "" {
		result[ParamKeyNegativePrompt] = c.NegativePrompt
	}
	if c.NumImages != nil {
		result[ParamKeyNumImages] = *c.NumImages
	}
	if c.GuidanceScale != nil {
		result[ParamKeyGuidanceScale] = *c.GuidanceScale
	}
	if c.Steps != nil {
		result[ParamKeySteps] = *c.Steps
	}
	if c.Strength != nil {
		result[ParamKeyStrength] = *c.Strength
	}

	return result
}

// EffectiveSize returns the image size string. If Size is set, it is returned directly.
// Otherwise, if both Width and Height are set, a "WxH" string is derived.
func (c *ImageConfig) EffectiveSize() string {
	if c == nil {
		return ""
	}
	if c.Size != "" {
		return c.Size
	}
	if c.Width != nil && c.Height != nil {
		return fmt.Sprintf("%dx%d", *c.Width, *c.Height)
	}
	return ""
}

// AudioConfig configures audio generation (TTS/transcription) parameters.
// AudioConfig is safe for concurrent reads. Callers must not modify the config after passing
// it to a Config; use Clone() to create an independent copy if mutation is needed.
type AudioConfig struct {
	// Voice is the voice name (e.g., "alloy", "echo", "nova")
	Voice string `yaml:"voice,omitempty" json:"voice,omitempty"`
	// VoiceID is a provider-specific voice identifier
	VoiceID string `yaml:"voice_id,omitempty" json:"voice_id,omitempty"`
	// Speed controls the playback speed (0.25-4.0)
	Speed *float64 `yaml:"speed,omitempty" json:"speed,omitempty"`
	// OutputFormat is the audio output format: "mp3", "opus", "aac", "flac", "wav", "pcm"
	OutputFormat string `yaml:"output_format,omitempty" json:"output_format,omitempty"`
	// Duration is the maximum duration in seconds (0-600)
	Duration *float64 `yaml:"duration,omitempty" json:"duration,omitempty"`
	// Language is the language code (e.g., "en", "es")
	Language string `yaml:"language,omitempty" json:"language,omitempty"`
}

// Validate checks the audio config for consistency.
func (c *AudioConfig) Validate() error {
	if c == nil {
		return nil
	}

	if c.Speed != nil {
		if *c.Speed < AudioMinSpeed || *c.Speed > AudioMaxSpeed {
			return NewConfigValidationError(ErrMsgAudioSpeedOutOfRange)
		}
	}

	if c.OutputFormat != "" && !isValidAudioFormat(c.OutputFormat) {
		return NewConfigValidationError(ErrMsgAudioInvalidFormat)
	}

	if c.Duration != nil {
		if *c.Duration <= 0.0 || *c.Duration > AudioMaxDuration {
			return NewConfigValidationError(ErrMsgAudioDurationOutOfRange)
		}
	}

	return nil
}

// Clone creates a deep copy of the audio config.
func (c *AudioConfig) Clone() *AudioConfig {
	if c == nil {
		return nil
	}

	clone := &AudioConfig{
		Voice:        c.Voice,
		VoiceID:      c.VoiceID,
		OutputFormat: c.OutputFormat,
		Language:     c.Language,
	}

	if c.Speed != nil {
		v := *c.Speed
		clone.Speed = &v
	}
	if c.Duration != nil {
		v := *c.Duration
		clone.Duration = &v
	}

	return clone
}

// ToMap converts the audio config to a parameter map.
func (c *AudioConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := make(map[string]any)

	if c.Voice != "" {
		result[ParamKeyVoice] = c.Voice
	}
	if c.VoiceID != "" {
		result[ParamKeyVoiceID] = c.VoiceID
	}
	if c.Speed != nil {
		result[ParamKeySpeed] = *c.Speed
	}
	if c.OutputFormat != "" {
		result[ParamKeyOutputFormat] = c.OutputFormat
	}
	if c.Duration != nil {
		result[ParamKeyDuration] = *c.Duration
	}
	if c.Language != "" {
		result[ParamKeyLanguage] = c.Language
	}

	return result
}

// EmbeddingConfig configures embedding generation parameters.
// EmbeddingConfig is safe for concurrent reads. The Normalize *bool pointer must not be
// modified after the config is shared; use Clone() to create an independent copy if mutation
// is needed.
type EmbeddingConfig struct {
	// Dimensions is the number of embedding dimensions (1-65536)
	Dimensions *int `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
	// Format is the embedding wire encoding format: "float" or "base64" (OpenAI encoding_format)
	Format string `yaml:"format,omitempty" json:"format,omitempty"`
	// InputType classifies the input for retrieval/search optimization
	InputType string `yaml:"input_type,omitempty" json:"input_type,omitempty"`
	// OutputDtype is the quantization data type: "float32", "int8", "uint8", "binary", "ubinary"
	OutputDtype string `yaml:"output_dtype,omitempty" json:"output_dtype,omitempty"`
	// Truncation controls how inputs exceeding the model's max length are handled: "none", "start", "end"
	Truncation string `yaml:"truncation,omitempty" json:"truncation,omitempty"`
	// Normalize controls whether embeddings are L2-normalized (vLLM)
	Normalize *bool `yaml:"normalize,omitempty" json:"normalize,omitempty"`
	// PoolingType selects the pooling strategy: "mean", "cls", "last" (vLLM)
	PoolingType string `yaml:"pooling_type,omitempty" json:"pooling_type,omitempty"`
}

// Validate checks the embedding config for consistency.
func (c *EmbeddingConfig) Validate() error {
	if c == nil {
		return nil
	}

	if c.Dimensions != nil {
		if *c.Dimensions < 1 || *c.Dimensions > EmbeddingMaxDimensions {
			return NewConfigValidationError(ErrMsgEmbeddingDimensionsOutOfRange)
		}
	}

	if c.Format != "" && !isValidEmbeddingFormat(c.Format) {
		return NewConfigValidationError(ErrMsgEmbeddingInvalidFormat)
	}

	if c.InputType != "" && !isValidEmbeddingInputType(c.InputType) {
		return NewConfigValidationError(ErrMsgEmbeddingInvalidInputType)
	}

	if c.OutputDtype != "" && !isValidEmbeddingOutputDtype(c.OutputDtype) {
		return NewConfigValidationError(ErrMsgEmbeddingInvalidOutputDtype)
	}

	if c.Truncation != "" && !isValidEmbeddingTruncation(c.Truncation) {
		return NewConfigValidationError(ErrMsgEmbeddingInvalidTruncation)
	}

	if c.PoolingType != "" && !isValidEmbeddingPoolingType(c.PoolingType) {
		return NewConfigValidationError(ErrMsgEmbeddingInvalidPoolingType)
	}

	return nil
}

// Clone creates a deep copy of the embedding config.
func (c *EmbeddingConfig) Clone() *EmbeddingConfig {
	if c == nil {
		return nil
	}

	clone := &EmbeddingConfig{
		Format:      c.Format,
		InputType:   c.InputType,
		OutputDtype: c.OutputDtype,
		Truncation:  c.Truncation,
		PoolingType: c.PoolingType,
	}

	if c.Dimensions != nil {
		v := *c.Dimensions
		clone.Dimensions = &v
	}

	if c.Normalize != nil {
		v := *c.Normalize
		clone.Normalize = &v
	}

	return clone
}

// ToMap converts the embedding config to a parameter map.
func (c *EmbeddingConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := make(map[string]any)

	if c.Dimensions != nil {
		result[ParamKeyDimensions] = *c.Dimensions
	}
	if c.Format != "" {
		result[ParamKeyEncodingFormat] = c.Format
	}
	if c.InputType != "" {
		result[ParamKeyInputType] = c.InputType
	}
	if c.OutputDtype != "" {
		result[ParamKeyOutputDtype] = c.OutputDtype
	}
	if c.Truncation != "" {
		result[ParamKeyTruncation] = c.Truncation
	}
	if c.Normalize != nil {
		result[ParamKeyNormalize] = *c.Normalize
	}
	if c.PoolingType != "" {
		result[ParamKeyPoolingType] = c.PoolingType
	}

	return result
}

// StreamingConfig configures streaming execution behavior.
// StreamingConfig is safe for concurrent reads. Use Clone() for mutation.
type StreamingConfig struct {
	// Enabled indicates whether streaming is enabled
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Method is the streaming transport method: "sse" or "websocket"
	Method string `yaml:"method,omitempty" json:"method,omitempty"`
}

// Validate checks the streaming config for consistency.
func (c *StreamingConfig) Validate() error {
	if c == nil {
		return nil
	}
	if c.Enabled && c.Method != "" && !isValidStreamMethod(c.Method) {
		return NewConfigValidationError(ErrMsgStreamInvalidMethod)
	}
	return nil
}

// Clone creates a deep copy of the streaming config.
func (c *StreamingConfig) Clone() *StreamingConfig {
	if c == nil {
		return nil
	}
	return &StreamingConfig{
		Enabled: c.Enabled,
		Method:  c.Method,
	}
}

// ToMap converts the streaming config to a parameter map.
func (c *StreamingConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}
	result := map[string]any{
		ParamKeyEnabled: c.Enabled,
	}
	if c.Method != "" {
		result[ParamKeyStreamMethod] = c.Method
	}
	return result
}

// AsyncConfig configures asynchronous execution behavior.
// AsyncConfig is safe for concurrent reads. Use Clone() for mutation.
type AsyncConfig struct {
	// Enabled indicates whether async execution is enabled
	Enabled bool `yaml:"enabled" json:"enabled"`
	// PollIntervalSeconds is the polling interval in seconds (must be > 0)
	PollIntervalSeconds *float64 `yaml:"poll_interval_seconds,omitempty" json:"poll_interval_seconds,omitempty"`
	// PollTimeoutSeconds is the maximum polling duration in seconds (must be > 0, >= PollInterval)
	PollTimeoutSeconds *float64 `yaml:"poll_timeout_seconds,omitempty" json:"poll_timeout_seconds,omitempty"`
}

// Validate checks the async config for consistency.
func (c *AsyncConfig) Validate() error {
	if c == nil {
		return nil
	}

	if c.PollIntervalSeconds != nil {
		if *c.PollIntervalSeconds <= 0 {
			return NewConfigValidationError(ErrMsgAsyncPollIntervalInvalid)
		}
	}

	if c.PollTimeoutSeconds != nil {
		if *c.PollTimeoutSeconds <= 0 {
			return NewConfigValidationError(ErrMsgAsyncPollTimeoutInvalid)
		}
	}

	if c.PollIntervalSeconds != nil && c.PollTimeoutSeconds != nil {
		if *c.PollTimeoutSeconds < *c.PollIntervalSeconds {
			return NewConfigValidationError(ErrMsgAsyncPollTimeoutTooSmall)
		}
	}

	return nil
}

// Clone creates a deep copy of the async config.
func (c *AsyncConfig) Clone() *AsyncConfig {
	if c == nil {
		return nil
	}

	clone := &AsyncConfig{
		Enabled: c.Enabled,
	}

	if c.PollIntervalSeconds != nil {
		v := *c.PollIntervalSeconds
		clone.PollIntervalSeconds = &v
	}
	if c.PollTimeoutSeconds != nil {
		v := *c.PollTimeoutSeconds
		clone.PollTimeoutSeconds = &v
	}

	return clone
}

// ToMap converts the async config to a parameter map.
func (c *AsyncConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := make(map[string]any)

	result[ParamKeyEnabled] = c.Enabled
	if c.PollIntervalSeconds != nil {
		result[ParamKeyPollInterval] = *c.PollIntervalSeconds
	}
	if c.PollTimeoutSeconds != nil {
		result[ParamKeyPollTimeout] = *c.PollTimeoutSeconds
	}

	return result
}

// ResponseFormat serialization methods

// ToOpenAI converts the response format to OpenAI API format.
// Returns nil if the response format is not configured.
func (rf *ResponseFormat) ToOpenAI() map[string]any {
	if rf == nil {
		return nil
	}

	result := map[string]any{
		SchemaKeyType: rf.Type,
	}

	if rf.JSONSchema != nil {
		jsonSchema := map[string]any{
			AttrName: rf.JSONSchema.Name,
		}

		if rf.JSONSchema.Description != "" {
			jsonSchema[SchemaKeyDescription] = rf.JSONSchema.Description
		}

		if rf.JSONSchema.Strict {
			jsonSchema[SchemaKeyStrict] = true
		}

		if rf.JSONSchema.Schema != nil {
			// Ensure additionalProperties: false for strict mode
			schema := copySchema(rf.JSONSchema.Schema)
			if rf.JSONSchema.Strict {
				ensureAdditionalPropertiesFalse(schema)
			}
			jsonSchema[SchemaKeySchema] = schema
		}

		result[SchemaKeyJSONSchema] = jsonSchema
	}

	if rf.Enum != nil && len(rf.Enum.Values) > 0 {
		result[SchemaKeyEnum] = rf.Enum.Values
	}

	return result
}

// ToAnthropic converts to Anthropic output_format structure.
// Returns nil if the response format is not configured.
func (rf *ResponseFormat) ToAnthropic() map[string]any {
	if rf == nil || rf.JSONSchema == nil {
		return nil
	}

	schema := copySchema(rf.JSONSchema.Schema)
	ensureAdditionalPropertiesFalse(schema)

	return map[string]any{
		SchemaKeyFormat: map[string]any{
			SchemaKeyType:   ResponseFormatJSONSchema,
			SchemaKeySchema: schema,
		},
	}
}

// ToGemini converts to Google Gemini/Vertex AI format.
// Returns nil if the response format is not configured.
func (rf *ResponseFormat) ToGemini() map[string]any {
	if rf == nil {
		return nil
	}

	result := map[string]any{
		SchemaKeyType: rf.Type,
	}

	if rf.JSONSchema != nil {
		schema := copySchema(rf.JSONSchema.Schema)
		ensureAdditionalPropertiesFalse(schema)

		// Add propertyOrdering for Gemini 2.5+
		if len(rf.JSONSchema.PropertyOrdering) > 0 {
			schema[SchemaKeyPropertyOrdering] = rf.JSONSchema.PropertyOrdering
		}

		jsonSchema := map[string]any{
			AttrName:        rf.JSONSchema.Name,
			SchemaKeySchema: schema,
		}

		if rf.JSONSchema.Description != "" {
			jsonSchema[SchemaKeyDescription] = rf.JSONSchema.Description
		}

		result[SchemaKeyJSONSchema] = jsonSchema
	}

	return result
}

// GuidedDecoding serialization methods

// ToVLLM converts to vLLM guided decoding format.
// Returns nil if guided decoding is not configured.
func (gd *GuidedDecoding) ToVLLM() map[string]any {
	if gd == nil {
		return nil
	}

	result := make(map[string]any)

	if gd.Backend != "" {
		result[GuidedKeyDecodingBackend] = gd.Backend
	}

	if gd.JSON != nil {
		schema := copySchema(gd.JSON)
		ensureAdditionalPropertiesFalse(schema)
		result[GuidedKeyJSON] = schema
	}

	if gd.Regex != "" {
		result[GuidedKeyRegex] = gd.Regex
	}

	if len(gd.Choice) > 0 {
		result[GuidedKeyChoice] = gd.Choice
	}

	if gd.Grammar != "" {
		result[GuidedKeyGrammar] = gd.Grammar
	}

	if gd.WhitespacePattern != "" {
		result[GuidedKeyWhitespacePattern] = gd.WhitespacePattern
	}

	return result
}

// Schema helpers

// copySchema creates a deep copy of a schema map.
func copySchema(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}

	dst := make(map[string]any, len(src))
	for k, v := range src {
		switch val := v.(type) {
		case map[string]any:
			dst[k] = copySchema(val)
		case []any:
			dst[k] = copySlice(val)
		default:
			dst[k] = v
		}
	}
	return dst
}

// copySlice creates a deep copy of a slice.
func copySlice(src []any) []any {
	if src == nil {
		return nil
	}

	dst := make([]any, len(src))
	for i, v := range src {
		switch val := v.(type) {
		case map[string]any:
			dst[i] = copySchema(val)
		case []any:
			dst[i] = copySlice(val)
		default:
			dst[i] = v
		}
	}
	return dst
}

// ensureAdditionalPropertiesFalse recursively ensures all objects have additionalProperties: false.
func ensureAdditionalPropertiesFalse(schema map[string]any) {
	if schema == nil {
		return
	}

	// Check if this is an object type
	if typeVal, ok := schema[SchemaKeyType]; ok && typeVal == SchemaTypeObject {
		// Set additionalProperties: false if not already set
		if _, exists := schema[SchemaKeyAdditionalProperties]; !exists {
			schema[SchemaKeyAdditionalProperties] = false
		}
	}

	// Recursively process properties
	if props, ok := schema[SchemaKeyProperties].(map[string]any); ok {
		for _, propVal := range props {
			if propSchema, ok := propVal.(map[string]any); ok {
				ensureAdditionalPropertiesFalse(propSchema)
			}
		}
	}

	// Recursively process array items
	if items, ok := schema[SchemaKeyItems].(map[string]any); ok {
		ensureAdditionalPropertiesFalse(items)
	}
}

// Validation helpers

// isValidModality checks if the given string is a valid modality value.
func isValidModality(m string) bool {
	switch m {
	case ModalityText, ModalityImage, ModalityAudioSpeech,
		ModalityAudioTranscription, ModalityMusic,
		ModalitySoundEffects, ModalityEmbedding,
		ModalityVideo, ModalityImageEdit:
		return true
	default:
		return false
	}
}

// isValidImageQuality checks if the given string is a valid image quality.
func isValidImageQuality(q string) bool {
	switch q {
	case ImageQualityStandard, ImageQualityHD,
		ImageQualityLow, ImageQualityMedium, ImageQualityHigh:
		return true
	default:
		return false
	}
}

// isValidImageStyle checks if the given string is a valid image style.
func isValidImageStyle(s string) bool {
	switch s {
	case ImageStyleNatural, ImageStyleVivid:
		return true
	default:
		return false
	}
}

// isValidAudioFormat checks if the given string is a valid audio format.
func isValidAudioFormat(f string) bool {
	switch f {
	case AudioFormatMP3, AudioFormatOpus, AudioFormatAAC,
		AudioFormatFLAC, AudioFormatWAV, AudioFormatPCM:
		return true
	default:
		return false
	}
}

// isValidEmbeddingFormat checks if the given string is a valid embedding format.
func isValidEmbeddingFormat(f string) bool {
	switch f {
	case EmbeddingFormatFloat, EmbeddingFormatBase64:
		return true
	default:
		return false
	}
}

// isValidEmbeddingInputType checks if the given string is a valid embedding input type.
func isValidEmbeddingInputType(t string) bool {
	switch t {
	case EmbeddingInputTypeSearchQuery, EmbeddingInputTypeSearchDocument,
		EmbeddingInputTypeClassification, EmbeddingInputTypeClustering,
		EmbeddingInputTypeSemanticSimilarity:
		return true
	default:
		return false
	}
}

// isValidEmbeddingOutputDtype checks if the given string is a valid embedding output dtype.
func isValidEmbeddingOutputDtype(d string) bool {
	switch d {
	case EmbeddingDtypeFloat32, EmbeddingDtypeInt8, EmbeddingDtypeUint8,
		EmbeddingDtypeBinary, EmbeddingDtypeUbinary:
		return true
	default:
		return false
	}
}

// isValidEmbeddingTruncation checks if the given string is a valid embedding truncation strategy.
func isValidEmbeddingTruncation(t string) bool {
	switch t {
	case EmbeddingTruncationNone, EmbeddingTruncationStart, EmbeddingTruncationEnd:
		return true
	default:
		return false
	}
}

// isValidEmbeddingPoolingType checks if the given string is a valid embedding pooling type.
func isValidEmbeddingPoolingType(p string) bool {
	switch p {
	case EmbeddingPoolingMean, EmbeddingPoolingCLS, EmbeddingPoolingLast:
		return true
	default:
		return false
	}
}

// isValidStreamMethod checks if the given string is a valid streaming method.
func isValidStreamMethod(m string) bool {
	switch m {
	case StreamMethodSSE, StreamMethodWebSocket:
		return true
	default:
		return false
	}
}

// isValidReasoningEffort checks if a reasoning effort string is valid.
func isValidReasoningEffort(effort string) bool {
	switch effort {
	case ReasoningEffortLow, ReasoningEffortMedium, ReasoningEffortHigh, ReasoningEffortMax:
		return true
	default:
		return false
	}
}

// isValidServiceTier checks if a service tier string is valid.
func isValidServiceTier(tier string) bool {
	switch tier {
	case ServiceTierAuto, ServiceTierDefault:
		return true
	default:
		return false
	}
}

// Model detection helpers

// isOpenAIModel checks if the model name suggests OpenAI.
func isOpenAIModel(name string) bool {
	prefixes := []string{"gpt-", "o1-", "o3-", "text-", "davinci", "curie", "babbage", "ada"}
	for _, prefix := range prefixes {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// isAnthropicModel checks if the model name suggests Anthropic.
func isAnthropicModel(name string) bool {
	prefixes := []string{"claude-", "claude"}
	for _, prefix := range prefixes {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// isGeminiModel checks if the model name suggests Google Gemini.
func isGeminiModel(name string) bool {
	prefixes := []string{"gemini-", "gemini"}
	for _, prefix := range prefixes {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// isMistralModel checks if the model name suggests Mistral AI.
func isMistralModel(name string) bool {
	prefixes := []string{"mistral-", "codestral-", "pixtral-", "ministral-", "open-mistral-", "open-mixtral-"}
	for _, prefix := range prefixes {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// isCohereModel checks if the model name suggests Cohere.
func isCohereModel(name string) bool {
	prefixes := []string{"command-", "embed-", "rerank-", "c4ai-"}
	for _, prefix := range prefixes {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// GeminiTaskType converts an embedding input type to Gemini's UPPER_CASE task_type format.
//
// Mapping:
//   - "search_query"        -> "RETRIEVAL_QUERY"
//   - "search_document"     -> "RETRIEVAL_DOCUMENT"
//   - "semantic_similarity" -> "SEMANTIC_SIMILARITY"
//   - "classification"      -> "CLASSIFICATION"
//   - "clustering"          -> "CLUSTERING"
func GeminiTaskType(inputType string) (string, error) {
	switch inputType {
	case EmbeddingInputTypeSearchQuery:
		return GeminiTaskRetrievalQuery, nil
	case EmbeddingInputTypeSearchDocument:
		return GeminiTaskRetrievalDocument, nil
	case EmbeddingInputTypeSemanticSimilarity:
		return GeminiTaskSemanticSimilarity, nil
	case EmbeddingInputTypeClassification:
		return GeminiTaskClassification, nil
	case EmbeddingInputTypeClustering:
		return GeminiTaskClustering, nil
	default:
		return "", NewConfigValidationError(ErrMsgEmbeddingInvalidInputType)
	}
}

// CohereUpperCase converts an embedding truncation strategy to Cohere's UPPER_CASE format.
//
// Mapping:
//   - "none"  -> "NONE"
//   - "start" -> "START"
//   - "end"   -> "END"
func CohereUpperCase(truncation string) (string, error) {
	switch truncation {
	case EmbeddingTruncationNone:
		return CohereTruncateNone, nil
	case EmbeddingTruncationStart:
		return CohereTruncateStart, nil
	case EmbeddingTruncationEnd:
		return CohereTruncateEnd, nil
	default:
		return "", NewConfigValidationError(ErrMsgEmbeddingInvalidTruncation)
	}
}
