// Package execution provides LLM execution configuration with multi-provider
// serialization support (OpenAI, Anthropic, Gemini, vLLM, Mistral, Cohere).
package execution

// LLM Provider names for structured output handling.
// Re-declared locally to avoid circular dependency with root exons package.
const (
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
	ProviderGoogle    = "google"
	ProviderGemini    = "gemini"
	ProviderVertex    = "vertex"
	ProviderVLLM      = "vllm"
	ProviderAzure     = "azure"
	ProviderMistral   = "mistral"
	ProviderCohere    = "cohere"
)

// Model parameter map keys (for ToMap conversion).
const (
	ParamKeyTemperature       = "temperature"
	ParamKeyMaxTokens         = "max_tokens"
	ParamKeyTopP              = "top_p"
	ParamKeyTopK              = "top_k"
	ParamKeyStop              = "stop"
	ParamKeyStopSequences     = "stop_sequences"
	ParamKeySeed              = "seed"
	ParamKeyMinP              = "min_p"
	ParamKeyRepetitionPenalty = "repetition_penalty"
	ParamKeyLogprobs          = "logprobs"
	ParamKeyTopLogprobs       = "top_logprobs"
	ParamKeyStopTokenIDs      = "stop_token_ids"
	ParamKeyLogitBias         = "logit_bias"
	ParamKeyFrequencyPenalty  = "frequency_penalty"
	ParamKeyPresencePenalty   = "presence_penalty"
	ParamKeyModel             = "model"

	// LLM parameter alignment keys
	ParamKeyN                   = "n"
	ParamKeyMaxCompletionTokens = "max_completion_tokens"
	ParamKeyReasoningEffort     = "reasoning_effort"
	ParamKeyTopA                = "top_a"
	ParamKeyUser                = "user"
	ParamKeyServiceTier         = "service_tier"
	ParamKeyStore               = "store"
	ParamKeyParallelToolCalls   = "parallel_tool_calls"
)

// Anthropic-specific parameter keys
const (
	ParamKeyAnthropicThinking     = "thinking"
	ParamKeyAnthropicOutputFormat = "output_format"
	ParamKeyThinkingType          = "type"
	ParamKeyThinkingTypeEnabled   = "enabled"
	ParamKeyBudgetTokens          = "budget_tokens"
)

// Gemini-specific parameter keys
const (
	ParamKeyGenerationConfig     = "generationConfig"
	ParamKeyGeminiMaxTokens      = "maxOutputTokens"
	ParamKeyGeminiTopP           = "topP"
	ParamKeyGeminiTopK           = "topK"
	ParamKeyGeminiStopSeqs       = "stopSequences"
	ParamKeyGeminiResponseMime   = "responseMimeType"
	ParamKeyGeminiResponseSchema = "responseSchema"
	GeminiResponseMimeJSON       = "application/json"
	ParamKeyGeminiNumImages      = "numberOfImages"
)

// vLLM guided decoding parameter keys
const (
	GuidedKeyDecodingBackend   = "guided_decoding_backend"
	GuidedKeyJSON              = "guided_json"
	GuidedKeyRegex             = "guided_regex"
	GuidedKeyChoice            = "guided_choice"
	GuidedKeyGrammar           = "guided_grammar"
	GuidedKeyWhitespacePattern = "guided_whitespace_pattern"
)

// Media parameter map keys (for serialization)
const (
	ParamKeyModality        = "modality"
	ParamKeyImage           = "image"
	ParamKeyAudio           = "audio"
	ParamKeyEmbedding       = "embedding"
	ParamKeyStreaming       = "streaming"
	ParamKeyAsync           = "async"
	ParamKeyStream          = "stream"
	ParamKeyImageSize       = "size"
	ParamKeyImageQuality    = "quality"
	ParamKeyImageStyle      = "style"
	ParamKeyImageN          = "n"
	ParamKeyVoice           = "voice"
	ParamKeySpeed           = "speed"
	ParamKeyDimensions      = "dimensions"
	ParamKeyEncodingFormat  = "encoding_format"
	ParamKeyAspectRatio     = "aspect_ratio"
	ParamKeyNegativePrompt  = "negative_prompt"
	ParamKeyNumImages       = "num_images"
	ParamKeyGuidanceScale   = "guidance_scale"
	ParamKeySteps           = "steps"
	ParamKeyStrength        = "strength"
	ParamKeyVoiceID         = "voice_id"
	ParamKeyOutputFormat    = "output_format"
	ParamKeyDuration        = "duration"
	ParamKeyLanguage        = "language"
	ParamKeyPollInterval    = "poll_interval_seconds"
	ParamKeyPollTimeout     = "poll_timeout_seconds"
	ParamKeyStreamMethod    = "method"
	ParamKeyWidth           = "width"
	ParamKeyHeight          = "height"
	ParamKeyEnabled         = "enabled"
	ParamKeyResponseFormat  = "response_format"
)

// Embedding parameter keys
const (
	ParamKeyInputType           = "input_type"
	ParamKeyOutputDtype         = "output_dtype"
	ParamKeyTruncation          = "truncation"
	ParamKeyNormalize           = "normalize"
	ParamKeyPoolingType         = "pooling_type"
	ParamKeyOutputDimension     = "output_dimension"
	ParamKeyOutputDimensionality = "output_dimensionality"
	ParamKeyTaskType            = "task_type"
	ParamKeyEmbeddingTypes      = "embedding_types"
	ParamKeyTruncate            = "truncate"
)

// Cohere-specific parameter keys
const (
	ParamKeyCohereTopP          = "p"
	ParamKeyCohereTopK          = "k"
	ParamKeyCohereStopSequences = "stop_sequences"
)

// JSON Schema property keys
const (
	SchemaKeyType                 = "type"
	SchemaKeyProperties           = "properties"
	SchemaKeyRequired             = "required"
	SchemaKeyAdditionalProperties = "additionalProperties"
	SchemaKeyEnum                 = "enum"
	SchemaKeyItems                = "items"
	SchemaKeyPropertyOrdering     = "propertyOrdering"
	SchemaKeyDescription          = "description"
	SchemaKeySchema               = "schema"
	SchemaKeyStrict               = "strict"
	SchemaKeyFormat               = "format"
	SchemaKeyJSONSchema           = "json_schema"
)

// Tool format keys
const (
	ToolKeyFunction    = "function"
	ToolKeyParameters  = "parameters"
	ToolKeyInputSchema = "input_schema"
)

// Response format types for structured outputs
const (
	ResponseFormatText       = "text"
	ResponseFormatJSONObject = "json_object"
	ResponseFormatJSONSchema = "json_schema"
	ResponseFormatEnum       = "enum"
)

// vLLM guided decoding backends
const (
	GuidedBackendXGrammar         = "xgrammar"
	GuidedBackendOutlines         = "outlines"
	GuidedBackendLMFormatEnforcer = "lm_format_enforcer"
	GuidedBackendAuto             = "auto"
)

// Schema types
const (
	SchemaTypeString  = "string"
	SchemaTypeNumber  = "number"
	SchemaTypeBoolean = "boolean"
	SchemaTypeArray   = "array"
	SchemaTypeObject  = "object"
)

// Modality constants — execution intent signal
const (
	ModalityText               = "text"
	ModalityImage              = "image"
	ModalityAudioSpeech        = "audio_speech"
	ModalityAudioTranscription = "audio_transcription"
	ModalityMusic              = "music"
	ModalitySoundEffects       = "sound_effects"
	ModalityEmbedding          = "embedding"
	ModalityVideo              = "video"
	ModalityImageEdit          = "image_edit"
)

// Streaming method constants
const (
	StreamMethodSSE       = "sse"
	StreamMethodWebSocket = "websocket"
)

// Image quality constants
const (
	ImageQualityStandard = "standard"
	ImageQualityHD       = "hd"
	ImageQualityLow      = "low"
	ImageQualityMedium   = "medium"
	ImageQualityHigh     = "high"
)

// Image style constants
const (
	ImageStyleNatural = "natural"
	ImageStyleVivid   = "vivid"
)

// Audio format constants
const (
	AudioFormatMP3  = "mp3"
	AudioFormatOpus = "opus"
	AudioFormatAAC  = "aac"
	AudioFormatFLAC = "flac"
	AudioFormatWAV  = "wav"
	AudioFormatPCM  = "pcm"
)

// Embedding format constants (wire encoding)
const (
	EmbeddingFormatFloat  = "float"
	EmbeddingFormatBase64 = "base64"
)

// Embedding input type constants
const (
	EmbeddingInputTypeSearchQuery        = "search_query"
	EmbeddingInputTypeSearchDocument     = "search_document"
	EmbeddingInputTypeClassification     = "classification"
	EmbeddingInputTypeClustering         = "clustering"
	EmbeddingInputTypeSemanticSimilarity = "semantic_similarity"
)

// Embedding output dtype constants (quantization data type)
const (
	EmbeddingDtypeFloat32 = "float32"
	EmbeddingDtypeInt8    = "int8"
	EmbeddingDtypeUint8   = "uint8"
	EmbeddingDtypeBinary  = "binary"
	EmbeddingDtypeUbinary = "ubinary"
)

// Embedding truncation strategy constants
const (
	EmbeddingTruncationNone  = "none"
	EmbeddingTruncationStart = "start"
	EmbeddingTruncationEnd   = "end"
)

// Embedding pooling type constants (vLLM)
const (
	EmbeddingPoolingMean = "mean"
	EmbeddingPoolingCLS  = "cls"
	EmbeddingPoolingLast = "last"
)

// Gemini task type mappings (UPPER_CASE values for Gemini API)
const (
	GeminiTaskRetrievalQuery     = "RETRIEVAL_QUERY"
	GeminiTaskRetrievalDocument  = "RETRIEVAL_DOCUMENT"
	GeminiTaskSemanticSimilarity = "SEMANTIC_SIMILARITY"
	GeminiTaskClassification     = "CLASSIFICATION"
	GeminiTaskClustering         = "CLUSTERING"
)

// Cohere truncation UPPER_CASE constants
const (
	CohereTruncateNone  = "NONE"
	CohereTruncateStart = "START"
	CohereTruncateEnd   = "END"
)

// ReasoningEffort enum constants
const (
	ReasoningEffortLow    = "low"
	ReasoningEffortMedium = "medium"
	ReasoningEffortHigh   = "high"
	ReasoningEffortMax    = "max"
)

// ServiceTier enum constants
const (
	ServiceTierAuto    = "auto"
	ServiceTierDefault = "default"
)

// NMax is the maximum number of completions
const NMax = 128

// Media validation limits
const (
	ImageMaxWidth          = 8192
	ImageMaxHeight         = 8192
	ImageMaxNumImages      = 10
	ImageMaxGuidanceScale  = 30.0
	ImageMaxSteps          = 200
	AudioMinSpeed          = 0.25
	AudioMaxSpeed          = 4.0
	AudioMaxDuration       = 600.0
	EmbeddingMaxDimensions = 65536
)

// Error code for execution config validation
const ErrCodeExecution = "EXONS_EXECUTION"

// Metadata keys
const (
	MetaKeyProvider = "provider"
)

// Error message constants — ALL error messages must be constants (NO MAGIC STRINGS)
const (
	// Core execution parameter validation messages
	ErrMsgTemperatureOutOfRange = "temperature must be between 0.0 and 2.0"
	ErrMsgTopPOutOfRange        = "top_p must be between 0.0 and 1.0"
	ErrMsgMaxTokensInvalid      = "max_tokens must be positive"
	ErrMsgTopKInvalid           = "top_k must be non-negative"
	ErrMsgThinkingBudgetInvalid = "thinking.budget_tokens must be positive"

	// Inference parameter validation messages
	ErrMsgMinPOutOfRange              = "min_p must be between 0.0 and 1.0"
	ErrMsgRepetitionPenaltyOutOfRange = "repetition_penalty must be greater than 0.0"
	ErrMsgLogprobsOutOfRange          = "logprobs must be between 0 and 20"
	ErrMsgStopTokenIDNegative         = "stop_token_ids values must be non-negative"
	ErrMsgLogitBiasValueOutOfRange    = "logit_bias values must be between -100.0 and 100.0"

	// LLM parameter alignment validation messages
	ErrMsgFrequencyPenaltyOutOfRange = "frequency_penalty must be between -2.0 and 2.0"
	ErrMsgPresencePenaltyOutOfRange  = "presence_penalty must be between -2.0 and 2.0"
	ErrMsgNOutOfRange                = "n must be between 1 and 128"
	ErrMsgMaxCompletionTokensInvalid = "max_completion_tokens must be positive"
	ErrMsgReasoningEffortInvalid     = "reasoning_effort must be low, medium, high, or max"
	ErrMsgTopAOutOfRange             = "top_a must be between 0.0 and 1.0"
	ErrMsgServiceTierInvalid         = "service_tier must be auto or default"

	// Media generation validation messages
	ErrMsgInvalidModality               = "invalid modality value"
	ErrMsgImageWidthOutOfRange          = "image width must be between 1 and 8192"
	ErrMsgImageHeightOutOfRange         = "image height must be between 1 and 8192"
	ErrMsgImageNumImagesOutOfRange      = "num_images must be between 1 and 10"
	ErrMsgImageGuidanceScaleOutOfRange  = "guidance_scale must be between 0.0 and 30.0"
	ErrMsgImageStepsOutOfRange          = "steps must be between 1 and 200"
	ErrMsgImageStrengthOutOfRange       = "strength must be between 0.0 and 1.0"
	ErrMsgImageInvalidQuality           = "invalid image quality value"
	ErrMsgImageInvalidStyle             = "invalid image style value"
	ErrMsgAudioSpeedOutOfRange          = "audio speed must be between 0.25 and 4.0"
	ErrMsgAudioInvalidFormat            = "invalid audio output format"
	ErrMsgAudioDurationOutOfRange       = "audio duration must be between 0.0 and 600.0"
	ErrMsgEmbeddingDimensionsOutOfRange = "embedding dimensions must be between 1 and 65536"
	ErrMsgEmbeddingInvalidFormat        = "invalid embedding format"
	ErrMsgEmbeddingInvalidInputType     = "invalid embedding input type"
	ErrMsgEmbeddingInvalidOutputDtype   = "invalid embedding output dtype"
	ErrMsgEmbeddingInvalidTruncation    = "invalid embedding truncation strategy"
	ErrMsgEmbeddingInvalidPoolingType   = "invalid embedding pooling type"
	ErrMsgStreamInvalidMethod           = "invalid streaming method"
	ErrMsgAsyncPollIntervalInvalid      = "async poll interval must be positive"
	ErrMsgAsyncPollTimeoutInvalid       = "async poll timeout must be positive"
	ErrMsgAsyncPollTimeoutTooSmall      = "async poll timeout must be greater than or equal to poll interval"

	// Schema/provider messages
	ErrMsgSchemaUnsupportedProvider = "unsupported provider for schema validation"
)

// Attribute name constants used by ResponseFormat.ToOpenAI
const AttrName = "name"

// JSON formatting constants
const JSONIndentDefault = "  "
