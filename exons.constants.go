package exons

import "time"

// Version is the current library version. Loaded from versions.yaml at build time.
const Version = "0.5.0"

// File extension for exons specification files.
const FileExtensionExons = ".exons"

// Delimiter constants — the {~ ~} syntax chosen for minimal collision with prompt content
const (
	DefaultOpenDelim  = "{~"
	DefaultCloseDelim = "~}"
	DefaultSelfClose  = "/~}"
	DefaultBlockClose = "{~/"
)

// Built-in tag names — all use exons. namespace prefix
const (
	TagNameVar         = "exons.var"
	TagNameRaw         = "exons.raw"
	TagNameInclude     = "exons.include"     // Nested template inclusion
	TagNameIf          = "exons.if"          // Conditional
	TagNameElseIf      = "exons.elseif"      // Conditional branch
	TagNameElse        = "exons.else"        // Conditional fallback
	TagNameFor         = "exons.for"         // Loop
	TagNameComment     = "exons.comment"     // Comment (removed from output)
	TagNameDefault     = "exons.default"     // Default value
	TagNameSwitch      = "exons.switch"      // Switch/case
	TagNameCase        = "exons.case"        // Switch case
	TagNameCaseDefault = "exons.casedefault" // Default case in switch
	TagNameEnv         = "exons.env"         // Environment variable resolver
	TagNameConfig      = "exons.config"      // Legacy configuration block (JSON)
	TagNameExtends     = "exons.extends"     // Template inheritance — extends parent
	TagNameBlock       = "exons.block"       // Template inheritance — overridable block
	TagNameParent      = "exons.parent"      // Template inheritance — call parent block content
	TagNameMessage     = "exons.message"     // Conversation message for chat API
	TagNameRef         = "exons.ref"         // Spec reference resolver
)

// YAML frontmatter constants
const (
	// YAMLFrontmatterDelimiter is the standard YAML frontmatter delimiter
	YAMLFrontmatterDelimiter = "---"
)

// Message role constants for exons.message tag
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// Message attribute constants
const (
	AttrRole  = "role"
	AttrCache = "cache"
)

// Reserved namespace prefix for built-in tags
const (
	TagNamespacePrefix = "exons."
)

// Attribute name constants
const (
	AttrName     = "name"
	AttrDefault  = "default"
	AttrEval     = "eval"
	AttrOnError  = "onerror"
	AttrFormat   = "format"
	AttrEscape   = "escape"
	AttrItem     = "item"
	AttrIndex    = "index"
	AttrIn       = "in"
	AttrLimit    = "limit"
	AttrValue    = "value"
	AttrText     = "text"
	AttrTemplate = "template" // Template name for include
	AttrWith     = "with"     // Context path for include
	AttrIsolate  = "isolate"  // Isolated context flag for include
	AttrRequired = "required" // Required flag for env resolver
	AttrSlug     = "slug"     // Spec slug for reference
	AttrVersion  = "version"  // Spec version for reference
)

// Boolean attribute values
const (
	AttrValueTrue  = "true"
	AttrValueFalse = "false"
)

// ErrorStrategy defines how to handle errors during execution
type ErrorStrategy int

const (
	// ErrorStrategyThrow stops execution and returns the error
	ErrorStrategyThrow ErrorStrategy = iota
	// ErrorStrategyDefault replaces failed content with a default value
	ErrorStrategyDefault
	// ErrorStrategyRemove removes the tag entirely from output
	ErrorStrategyRemove
	// ErrorStrategyKeepRaw keeps the original tag text in output
	ErrorStrategyKeepRaw
	// ErrorStrategyLog logs the error and continues with empty string
	ErrorStrategyLog
)

// ErrorStrategyNotSet is a sentinel value indicating no strategy override
const ErrorStrategyNotSet ErrorStrategy = -1

// Error strategy string values for attribute parsing
const (
	ErrorStrategyNameThrow   = "throw"
	ErrorStrategyNameDefault = "default"
	ErrorStrategyNameRemove  = "remove"
	ErrorStrategyNameKeepRaw = "keepraw"
	ErrorStrategyNameLog     = "log"
)

// String returns the string representation of the error strategy
func (s ErrorStrategy) String() string {
	switch s {
	case ErrorStrategyThrow:
		return ErrorStrategyNameThrow
	case ErrorStrategyDefault:
		return ErrorStrategyNameDefault
	case ErrorStrategyRemove:
		return ErrorStrategyNameRemove
	case ErrorStrategyKeepRaw:
		return ErrorStrategyNameKeepRaw
	case ErrorStrategyLog:
		return ErrorStrategyNameLog
	default:
		return ErrorStrategyNameThrow
	}
}

// ParseErrorStrategy parses a string into an ErrorStrategy.
// Returns ErrorStrategyThrow for unknown values.
func ParseErrorStrategy(s string) ErrorStrategy {
	switch s {
	case ErrorStrategyNameDefault:
		return ErrorStrategyDefault
	case ErrorStrategyNameRemove:
		return ErrorStrategyRemove
	case ErrorStrategyNameKeepRaw:
		return ErrorStrategyKeepRaw
	case ErrorStrategyNameLog:
		return ErrorStrategyLog
	case ErrorStrategyNameThrow:
		return ErrorStrategyThrow
	default:
		return ErrorStrategyThrow
	}
}

// IsValidErrorStrategy checks if a string is a valid error strategy name.
func IsValidErrorStrategy(s string) bool {
	switch s {
	case ErrorStrategyNameThrow, ErrorStrategyNameDefault,
		ErrorStrategyNameRemove, ErrorStrategyNameKeepRaw, ErrorStrategyNameLog:
		return true
	default:
		return false
	}
}

// ValidationSeverity indicates the severity of a validation issue.
type ValidationSeverity int

const (
	// SeverityError indicates a critical issue that prevents execution
	SeverityError ValidationSeverity = iota
	// SeverityWarning indicates a potential issue that may cause problems
	SeverityWarning
	// SeverityInfo indicates informational feedback
	SeverityInfo
)

// Validation severity string names
const (
	SeverityNameError   = "error"
	SeverityNameWarning = "warning"
	SeverityNameInfo    = "info"
)

// String returns the string representation of the validation severity
func (s ValidationSeverity) String() string {
	switch s {
	case SeverityError:
		return SeverityNameError
	case SeverityWarning:
		return SeverityNameWarning
	case SeverityInfo:
		return SeverityNameInfo
	default:
		return SeverityNameError
	}
}

// Path separator for dot-notation
const PathSeparator = "."

// Default configuration values
const (
	DefaultExecutionTimeout   = 30 * time.Second
	DefaultResolverTimeout    = 5 * time.Second
	DefaultFunctionTimeout    = 1 * time.Second
	DefaultMaxLoopIterations  = 10000
	DefaultMaxDepth           = 10
	DefaultMaxOutputSize      = 10 * 1024 * 1024 // 10MB
	DefaultMaxFrontmatterSize = 64 * 1024        // 64KB — DoS protection for YAML frontmatter
)

// Metadata keys for cuserr.WithMetadata
const (
	MetaKeyLine         = "line"
	MetaKeyColumn       = "column"
	MetaKeyOffset       = "offset"
	MetaKeyTag          = "tag"
	MetaKeyResolver     = "resolver"
	MetaKeyVariable     = "variable"
	MetaKeyAttribute    = "attribute"
	MetaKeyExpected     = "expected"
	MetaKeyActual       = "actual"
	MetaKeyPath         = "path"
	MetaKeyValue        = "value"
	MetaKeyTemplateName = "template_name"
	MetaKeyCurrentDepth = "current_depth"
	MetaKeyMaxDepth     = "max_depth"
	MetaKeyFuncName     = "func_name"
	MetaKeyReason       = "reason"
	MetaKeyFromType     = "from_type"
	MetaKeyToType       = "to_type"
	MetaKeyEnvVar       = "env_var"
	MetaKeyInputName    = "input_name"
	MetaKeySpecSlug     = "spec_slug"       // Spec slug for reference errors
	MetaKeySpecName     = "spec_name"       // Spec name for validation errors
	MetaKeyRefChain     = "reference_chain" // Reference chain for circular detection
	MetaKeyLabel        = "label"           // Label name for label operations
	MetaKeyFromStatus   = "from_status"     // Source status in transitions
	MetaKeyToStatus     = "to_status"       // Target status in transitions
	MetaKeyVersion      = "version"         // Version number
	MetaKeyStatus       = "status"          // Deployment status value
	MetaKeyProvider     = "provider"        // LLM provider name
)

// Escape sequence constants
const (
	EscapeOpenDelim  = "\\{~"
	LiteralOpenDelim = "{~"
)

// Internal meta keys for nested template data passing.
// These are used internally and prefixed with underscore to avoid collision.
const (
	MetaKeyParentDepth = "_parentDepth" // Used to pass depth between nested template executions
	MetaKeyValueData   = "_value"       // Used to pass non-map values in with attribute
	MetaKeyRawSource   = "_rawSource"   // Original tag source for keepRaw strategy
	MetaKeyStrategy    = "strategy"     // Applied error strategy for logging
)

// Input/Output schema types
const (
	SchemaTypeString  = "string"
	SchemaTypeNumber  = "number"
	SchemaTypeBoolean = "boolean"
	SchemaTypeArray   = "array"
	SchemaTypeObject  = "object"
)

// Model parameter map keys (for ToMap conversion)
const (
	ParamKeyTemperature       = "temperature"
	ParamKeyMaxTokens         = "max_tokens"
	ParamKeyTopP              = "top_p"
	ParamKeyFrequencyPenalty  = "frequency_penalty"
	ParamKeyPresencePenalty   = "presence_penalty"
	ParamKeyStop              = "stop"
	ParamKeySeed              = "seed"
	ParamKeyMinP              = "min_p"
	ParamKeyRepetitionPenalty = "repetition_penalty"
	ParamKeyLogprobs          = "logprobs"
	ParamKeyTopLogprobs       = "top_logprobs"
	ParamKeyStopTokenIDs      = "stop_token_ids"
	ParamKeyLogitBias         = "logit_bias"
	ParamKeyModel             = "model"
	ParamKeyTopK              = "top_k"
	ParamKeyStopSequences     = "stop_sequences"

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
)

// Error format strings for type validation
const (
	ErrFmtTypeMismatch = "expected %s, got %s"
)

// Error format string constants
const (
	FmtPosition       = "line %d, column %d"
	RefChainSeparator = " -> "
)

// LLM Provider names for structured output handling
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

// vLLM guided decoding parameter keys
const (
	GuidedKeyDecodingBackend   = "guided_decoding_backend"
	GuidedKeyJSON              = "guided_json"
	GuidedKeyRegex             = "guided_regex"
	GuidedKeyChoice            = "guided_choice"
	GuidedKeyGrammar           = "guided_grammar"
	GuidedKeyWhitespacePattern = "guided_whitespace_pattern"
)

// Spec validation constraints
const (
	// SpecNameMaxLength is the maximum length for spec names (slug format)
	SpecNameMaxLength = 64
	// SpecDescriptionMaxLength is the maximum length for spec descriptions
	SpecDescriptionMaxLength = 1024
	// SpecSlugPattern is the regex pattern for valid spec names/slugs.
	// Must start with lowercase letter, followed by lowercase letters, digits, or hyphens.
	SpecSlugPattern = `^[a-z][a-z0-9-]*$`
)

// Spec field name constants for YAML/JSON serialization keys.
// Used in buildSerializeMap, GetStandardFields, GetExonsFields, and extension key filtering.
const (
	// Standard fields
	SpecFieldName          = "name"
	SpecFieldDescription   = "description"
	SpecFieldLicense       = "license"
	SpecFieldCompatibility = "compatibility"
	SpecFieldAllowedTools  = "allowed_tools"
	SpecFieldMetadata      = "metadata"
	SpecFieldInputs        = "inputs"
	SpecFieldOutputs       = "outputs"
	SpecFieldSample        = "sample"

	// go-exons extension fields
	SpecFieldType        = "type"
	SpecFieldExecution   = "execution"
	SpecFieldExtensions  = "extensions"
	SpecFieldSkills      = "skills"
	SpecFieldTools       = "tools"
	SpecFieldContext     = "context"
	SpecFieldConstraints = "constraints"
	SpecFieldMessages    = "messages"

	// Credential/requirements fields
	SpecFieldCredentials  = "credentials"
	SpecFieldCredential   = "credential"
	SpecFieldRequirements = "requirements"

	// GenSpec field
	SpecFieldGenSpec = "genspec"
)

// DocumentType identifies the kind of document (prompt, skill, agent).
type DocumentType string

const (
	// DocumentTypePrompt is a simple prompt template (no skills/tools/constraints)
	DocumentTypePrompt DocumentType = "prompt"
	// DocumentTypeSkill is a reusable skill document (default type)
	DocumentTypeSkill DocumentType = "skill"
	// DocumentTypeAgent is a full agent definition with skills, tools, and constraints
	DocumentTypeAgent DocumentType = "agent"
)

// CatalogFormat defines the output format for catalog generation.
type CatalogFormat string

const (
	// CatalogFormatDefault uses markdown format
	CatalogFormatDefault CatalogFormat = ""
	// CatalogFormatDetailed includes full descriptions and parameters
	CatalogFormatDetailed CatalogFormat = "detailed"
	// CatalogFormatCompact uses minimal single-line format
	CatalogFormatCompact CatalogFormat = "compact"
	// CatalogFormatFunctionCalling generates JSON schema for function calling
	CatalogFormatFunctionCalling CatalogFormat = "function_calling"
)

// Catalog resolver tag names
const (
	TagNameSkillsCatalog = "exons.skills_catalog"
	TagNameToolsCatalog  = "exons.tools_catalog"
)

// SkillInjection defines how a skill is injected into an agent's context.
type SkillInjection string

const (
	// SkillInjectionNone does not inject skill content into the agent
	SkillInjectionNone SkillInjection = "none"
	// SkillInjectionSystemPrompt appends skill content to the system prompt
	SkillInjectionSystemPrompt SkillInjection = "system_prompt"
	// SkillInjectionUserContext injects skill content into user context
	SkillInjectionUserContext SkillInjection = "user_context"
)

// DeploymentStatus represents the lifecycle status of a template version.
type DeploymentStatus string

// Deployment status values — lifecycle of a template version.
const (
	// DeploymentStatusDraft is the initial status for new versions not yet ready for use.
	DeploymentStatusDraft DeploymentStatus = "draft"
	// DeploymentStatusActive indicates the version is ready for production use.
	DeploymentStatusActive DeploymentStatus = "active"
	// DeploymentStatusDeprecated marks the version as still functional but discouraged.
	DeploymentStatusDeprecated DeploymentStatus = "deprecated"
	// DeploymentStatusArchived is a terminal state — version is read-only and preserved for history.
	DeploymentStatusArchived DeploymentStatus = "archived"
)

// Reserved label names — commonly used deployment targets.
const (
	LabelProduction = "production"
	LabelStaging    = "staging"
	LabelCanary     = "canary"
)

// Label validation constraints.
const (
	// LabelMaxLength is the maximum length of a label name.
	LabelMaxLength = 64
	// LabelNamePattern is the regex pattern for valid label names.
	// Must start with lowercase letter, followed by lowercase letters, digits, underscores, or hyphens.
	LabelNamePattern = `^[a-z][a-z0-9_-]*$`
)

// Metadata keys for deployment audit trail.
const (
	// MetaKeyLabelPrefix prefixes label-related metadata entries.
	MetaKeyLabelPrefix = "label:"
	// MetaKeyStatusChangedAt records when status was last changed.
	MetaKeyStatusChangedAt = "status_changed_at"
	// MetaKeyStatusChangedBy records who changed the status.
	MetaKeyStatusChangedBy = "status_changed_by"
	// MetaKeyLabelAssignedAt records when a label was assigned.
	MetaKeyLabelAssignedAt = "label_assigned_at"
	// MetaKeyLabelAssignedBy records who assigned a label.
	MetaKeyLabelAssignedBy = "label_assigned_by"
	// MetaKeyRollbackFromVersion records the version that was rolled back to.
	MetaKeyRollbackFromVersion = "rollback_from_version"
	// MetaKeyClonedFrom records the template name that was cloned from.
	MetaKeyClonedFrom = "cloned_from"
	// MetaKeyClonedFromVersion records the version that was cloned from.
	MetaKeyClonedFromVersion = "cloned_from_version"
)

// Reference resolution constants
const (
	// RefMaxDepth is the maximum depth for nested spec references
	RefMaxDepth = 10
	// RefVersionLatest is the default version when not specified
	RefVersionLatest = "latest"
)

// Special template name for self-reference
const (
	TemplateNameSelf = "self"
)

// Context keys used during agent compilation
const (
	ContextKeyInput       = "input"
	ContextKeyMeta        = "meta"
	ContextKeyContext     = "context"
	ContextKeyConstraints = "constraints"
	ContextKeySkills      = "skills"
	ContextKeyTools       = "tools"
	ContextKeySelfBody    = "_selfBody"
)

// Skill injection markers
const (
	SkillInjectionMarkerStart = "<!-- SKILL_START:"
	SkillInjectionMarkerEnd   = "<!-- SKILL_END:"
	SkillInjectionMarkerClose = " -->"
)

// Field constraints
const (
	MaxLicenseLength       = 100
	MaxCompatibilityLength = 500
)

// Storage ID prefixes
const (
	TemplateIDPrefix = "tmpl_"
)

// Model API types
const (
	ModelAPIChat       = "chat"
	ModelAPICompletion = "completion"
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
	ParamKeyGeminiNumImages = "numberOfImages"

	// Embedding parameter keys
	ParamKeyInputType   = "input_type"
	ParamKeyOutputDtype = "output_dtype"
	ParamKeyTruncation  = "truncation"
	ParamKeyNormalize   = "normalize"
	ParamKeyPoolingType = "pooling_type"

	// Provider-specific embedding parameter keys
	ParamKeyOutputDimension      = "output_dimension"
	ParamKeyOutputDimensionality = "output_dimensionality"
	ParamKeyTaskType             = "task_type"
	ParamKeyEmbeddingTypes       = "embedding_types"
	ParamKeyTruncate             = "truncate"

	// Cohere-specific parameter keys
	ParamKeyCohereTopP          = "p"
	ParamKeyCohereTopK          = "k"
	ParamKeyCohereStopSequences = "stop_sequences"
)

// Cohere truncation UPPER_CASE constants
const (
	CohereTruncateNone  = "NONE"
	CohereTruncateStart = "START"
	CohereTruncateEnd   = "END"
)

// Provider binding modes for ExecutionRequirements
const (
	ProviderBindingRequired  = "required"
	ProviderBindingPreferred = "preferred"
	ProviderBindingAny       = "any"
)

// Metadata keys for credential context
const (
	MetaKeyCredentialLabel = "credential_label"
)

// A2A (Agent-to-Agent) protocol constants
const (
	// A2AProtocolVersionDefault is the default A2A protocol version
	A2AProtocolVersionDefault = "0.3.0"
	// A2AVersionDefault is the default agent version when not specified
	A2AVersionDefault = "1.0.0"
)

// A2A MIME type constants for input/output modes
const (
	A2AMIMETextPlain       = "text/plain"
	A2AMIMEApplicationJSON = "application/json"
	A2AMIMEImagePNG        = "image/png"
	A2AMIMEAudioMPEG       = "audio/mpeg"
	A2AMIMETextMarkdown    = "text/markdown"
)

// A2A extension key prefix
const (
	ExtensionPrefixA2A = "a2a."
)

// JSON formatting constants
const (
	JSONIndentDefault = "  "
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

// GenSpec version.
const GenSpecVersion = "1"

// Document export/import constants
const (
	DocumentFilenameAgent  = "AGENT.md"
	DocumentFilenamePrompt = "PROMPT.md"
	DocumentFilenameSkill  = "SKILL.md"
	FileExtensionMarkdown  = ".md"
	FileExtensionZip       = ".zip"
)

// Metadata keys for agent context
const (
	MetaKeyDocumentType  = "document_type"
	MetaKeySkillSlug     = "skill_slug"
	MetaKeyInjectionMode = "injection_mode"
	MetaKeySkillVersion  = "skill_version"
	MetaKeyMessageIndex  = "message_index"
	MetaKeyMessageRole   = "message_role"
	MetaKeyCompileStage  = "compile_stage"
)

// Versioning error messages
const (
	ErrMsgVersionGetFailed       = "failed to get version"
	ErrMsgVersionSaveRollback    = "failed to save rollback"
	ErrMsgVersionGetSource       = "failed to get source version"
	ErrMsgVersionTemplateExists  = "template already exists"
	ErrMsgVersionSaveClone       = "failed to save clone"
	ErrMsgVersionMinimumRequired = "must keep at least 1 version"
	ErrMsgVersionNoPrevious      = "no previous version for version 1"
)

// Catalog-specific error detail messages
const (
	ErrMsgCatalogFuncCallingSkills = "function_calling not supported for skills catalog"
	ErrMsgCatalogUnknownFormat     = "unknown catalog format"
)

// CatalogCompactDescriptionMaxLen is the max description length in compact format.
const CatalogCompactDescriptionMaxLen = 80

// Catalog formatting constants for markdown output generation.
const (
	CatalogHeaderSkills    = "## Skills\n\n"
	CatalogHeaderTools     = "## Tools\n\n"
	CatalogMDHeading3      = "### "
	CatalogMDListItem      = "- **"
	CatalogMDBoldClose     = "**"
	CatalogMDColonSep      = ": "
	CatalogMDDashSep       = " - "
	CatalogMDInjectionPfx  = "- Injection: "
	CatalogMDURLPfx        = "- URL: "
	CatalogMDMCPPfx        = "[MCP] "
	CatalogMDMCPListPfx    = "- **[MCP] "
	CatalogMDMCPDetailPfx  = "### [MCP] "
	CatalogMDCodeBlockOpen = "```json\n"
	CatalogMDCodeBlockEnd  = "\n```\n"
	CatalogMDParenOpen     = " ("
	CatalogMDParenClose    = ")"
	CatalogCompactSep      = "; "
	TruncationSuffix       = "..."
)

// Tool definition map key constants (for ToOpenAITool, ToGenericTool).
const (
	ToolKeyType        = "type"
	ToolKeyName        = "name"
	ToolKeyDescription = "description"
)

// Import resource size limit for zip bomb protection.
const MaxImportResourceSize = 50 * 1024 * 1024 // 50MB

// Import error for multiple document files in archive.
const ErrMsgImportMultipleDocuments = "multiple document files found in archive"
