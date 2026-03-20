package exons

import (
	"fmt"
	"strconv"

	"github.com/itsatony/go-cuserr"
)

// Error code constants for categorization
const (
	ErrCodeParse      = "EXONS_PARSE"
	ErrCodeExec       = "EXONS_EXEC"
	ErrCodeValidation = "EXONS_VALIDATION"
	ErrCodeRegistry   = "EXONS_REGISTRY"
	ErrCodeTemplate   = "EXONS_TEMPLATE"
	ErrCodeFunc       = "EXONS_FUNC"
	ErrCodeConfig     = "EXONS_CONFIG"
	ErrCodeEnv        = "EXONS_ENV"
	ErrCodeLabel      = "EXONS_LABEL"
	ErrCodeStatus     = "EXONS_STATUS"
	ErrCodeSchema     = "EXONS_SCHEMA"
	ErrCodeSpec       = "EXONS_SPEC"       // Spec validation errors
	ErrCodeRef        = "EXONS_REF"        // Reference resolution errors
	ErrCodeCompile    = "EXONS_COMPILE"    // Compilation errors
	ErrCodeExecution  = "EXONS_EXECUTION"  // Execution errors
	ErrCodeStorage    = "EXONS_STORAGE"    // Storage errors
	ErrCodeCredential = "EXONS_CREDENTIAL" // Credential validation errors
	ErrCodeManifest   = "EXONS_MANIFEST"   // Execution manifest errors
	ErrCodeA2A        = "EXONS_A2A"        // A2A Agent Card errors
	ErrCodeVersioning = "EXONS_VERSIONING" // Versioning operation errors
	ErrCodeGenSpec    = "EXONS_GENSPEC"    // GenSpec errors
	ErrCodeAgent      = "EXONS_AGENT"      // Agent errors
	ErrCodeCatalog    = "EXONS_CATALOG"    // Catalog errors
	ErrCodeSerialize  = "EXONS_SERIALIZE"  // Serialization errors
)

// Error message constants — ALL error messages must be constants (NO MAGIC STRINGS)
const (
	// Parse errors
	ErrMsgParseFailed     = "template parsing failed"
	ErrMsgInvalidSyntax   = "invalid template syntax"
	ErrMsgUnexpectedChar  = "unexpected character"
	ErrMsgUnterminatedTag = "unterminated tag"
	ErrMsgUnterminatedStr = "unterminated string literal"
	ErrMsgInvalidEscape   = "invalid escape sequence"
	ErrMsgUnexpectedEOF   = "unexpected end of input"
	ErrMsgMismatchedTag   = "mismatched closing tag"
	ErrMsgInvalidTagName  = "invalid tag name"
	ErrMsgEmptyTagName    = "tag name cannot be empty"
	ErrMsgNestedRawBlock  = "nested raw blocks are not allowed"

	// Execution errors
	ErrMsgUnknownTag       = "unknown tag"
	ErrMsgUnknownResolver  = "no resolver registered for tag"
	ErrMsgResolverFailed   = "resolver execution failed"
	ErrMsgVariableNotFound = "variable not found"
	ErrMsgInvalidPath      = "invalid variable path"
	ErrMsgEmptyPath        = "variable path cannot be empty"
	ErrMsgExecutionFailed  = "template execution failed"

	// Validation errors
	ErrMsgMissingAttribute = "required attribute missing"
	ErrMsgInvalidAttribute = "invalid attribute value"

	// Registry errors
	ErrMsgResolverExists = "resolver already registered"

	// Type conversion errors
	ErrMsgTypeConversion = "type conversion failed"

	// Template errors (nested template inclusion)
	ErrMsgTemplateNotFound      = "template not found"
	ErrMsgTemplateAlreadyExists = "template already registered"
	ErrMsgTemplateDepthExceeded = "template inclusion depth exceeded"
	ErrMsgInvalidTemplateName   = "invalid template name"
	ErrMsgEmptyTemplateName     = "template name cannot be empty"
	ErrMsgMissingTemplateAttr   = "missing required 'template' attribute"
	ErrMsgEngineNotAvailable    = "engine not available for nested template resolution"
	ErrMsgReservedTemplateName  = "template name uses reserved exons.* namespace"

	// Error strategy messages
	ErrMsgInvalidErrorStrategy = "invalid error strategy"
	ErrMsgErrorHandledByStrat  = "error handled by strategy"

	// Validation messages
	ErrMsgValidationFailed     = "template validation failed"
	ErrMsgUnknownTagInTemplate = "unknown tag in template"
	ErrMsgInvalidOnErrorAttr   = "invalid onerror attribute value"
	ErrMsgMissingIncludeTarget = "included template not found"

	// For loop messages
	ErrMsgForMissingItem    = "missing required 'item' attribute"
	ErrMsgForMissingIn      = "missing required 'in' attribute"
	ErrMsgForInvalidLimit   = "invalid 'limit' attribute value"
	ErrMsgForCollectionPath = "collection path not found"
	ErrMsgForNotIterable    = "value is not iterable"
	ErrMsgForLimitExceeded  = "loop iteration limit exceeded"
	ErrMsgForNotClosed      = "for block not closed"

	// Switch/case messages
	ErrMsgSwitchMissingEval      = "missing required 'eval' attribute for switch"
	ErrMsgSwitchMissingValue     = "case requires 'value' or 'eval' attribute"
	ErrMsgSwitchNotClosed        = "switch block not closed"
	ErrMsgSwitchCaseNotClosed    = "case block not closed"
	ErrMsgSwitchDefaultNotLast   = "default case must be last in switch"
	ErrMsgSwitchDuplicateDefault = "only one default case allowed in switch"
	ErrMsgSwitchInvalidCaseTag   = "unexpected tag inside switch block"

	// Custom function messages
	ErrMsgFuncNilFunc       = "function cannot be nil"
	ErrMsgFuncEmptyName     = "function name cannot be empty"
	ErrMsgFuncAlreadyExists = "function already registered"

	// Context messages
	ErrMsgInvalidContextType = "invalid context type"

	// Environment variable messages
	ErrMsgEnvVarNotFound = "environment variable not found"
	ErrMsgEnvVarRequired = "required environment variable not set"

	// Config block messages (legacy JSON — kept for backward compatibility)
	ErrMsgConfigBlockExtract    = "failed to extract config block"
	ErrMsgConfigBlockParse      = "failed to parse config block JSON"
	ErrMsgConfigBlockInvalid    = "invalid config block format"
	ErrMsgConfigBlockUnclosed   = "config block not properly closed"
	ErrMsgInputValidationFailed = "input validation failed"
	ErrMsgRequiredInputMissing  = "required input missing"
	ErrMsgInputTypeMismatch     = "input type mismatch"

	// YAML frontmatter messages
	ErrMsgFrontmatterExtract       = "failed to extract YAML frontmatter"
	ErrMsgFrontmatterParse         = "failed to parse YAML frontmatter"
	ErrMsgFrontmatterInvalid       = "invalid YAML frontmatter format"
	ErrMsgFrontmatterUnclosed      = "YAML frontmatter not properly closed"
	ErrMsgLegacyJSONConfigDetected = "legacy JSON config block detected - please migrate to YAML frontmatter with --- delimiters"

	// Message tag messages
	ErrMsgMessageMissingRole      = "missing required 'role' attribute"
	ErrMsgMessageInvalidRole      = "invalid role - must be system, user, assistant, or tool"
	ErrMsgMessageNestedNotAllowed = "nested message tags are not allowed"

	// YAML frontmatter size limits
	ErrMsgFrontmatterTooLarge = "YAML frontmatter exceeds maximum size limit"

	// Deployment status messages
	ErrMsgInvalidDeploymentStatus = "invalid deployment status"
	ErrMsgStatusTransitionDenied  = "status transition not allowed"
	ErrMsgArchivedVersionReadOnly = "archived versions are read-only"

	// Label messages
	ErrMsgInvalidLabelName   = "invalid label name"
	ErrMsgLabelNotFound      = "label not found"
	ErrMsgLabelNameTooLong   = "label name exceeds maximum length"
	ErrMsgLabelNameEmpty     = "label name cannot be empty"
	ErrMsgLabelVersionError  = "label version mismatch"
	ErrMsgInvalidLabelFormat = "must start with lowercase letter and contain only lowercase letters, digits, underscores, or hyphens"

	// Schema validation messages
	ErrMsgSchemaValidationFailed     = "schema validation failed"
	ErrMsgSchemaInvalidType          = "schema has invalid type"
	ErrMsgSchemaMissingType          = "schema missing required 'type' field"
	ErrMsgSchemaMissingProperties    = "object schema missing 'properties' field"
	ErrMsgSchemaInvalidProperties    = "schema 'properties' field must be an object"
	ErrMsgSchemaInvalidRequired      = "schema 'required' field must be an array"
	ErrMsgSchemaInvalidEnum          = "enum values must be a non-empty array"
	ErrMsgSchemaUnsupportedProvider  = "unsupported provider for schema validation"
	ErrMsgSchemaAdditionalProperties = "strict mode requires additionalProperties: false"
	ErrMsgSchemaPropertyOrdering     = "propertyOrdering requires Gemini 2.5+ provider"
	ErrMsgEnumEmptyValues            = "enum constraint requires at least one value"
	ErrMsgGuidedDecodingConflict     = "only one guided decoding constraint allowed"

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

	// A2A Agent Card compilation messages
	ErrMsgA2ACardNilSpec     = "cannot compile agent card from nil spec"
	ErrMsgA2ACardMissingURL  = "agent card requires a service URL in options"
	ErrMsgA2ACardMissingName = "agent card requires a spec name"

	// Credential and manifest validation messages
	ErrMsgCredentialNotFound        = "credential label not found in credentials map"
	ErrMsgCredentialMissingProvider = "credential must specify a provider"
	ErrMsgInvalidProviderBinding    = "provider binding must be required, preferred, or any"
	ErrMsgEstimatedLatencyNegative  = "estimated latency must be non-negative"
	ErrMsgManifestNilSpec           = "cannot compile manifest from nil spec"

	// Spec validation messages
	ErrMsgSpecNameRequired        = "spec name is required"
	ErrMsgSpecNameTooLong         = "spec name exceeds maximum length"
	ErrMsgSpecNameInvalidFormat   = "spec name must be slug format (lowercase letters, digits, hyphens)"
	ErrMsgSpecDescriptionRequired = "spec description is required"
	ErrMsgSpecDescriptionTooLong  = "spec description exceeds maximum length"
	ErrMsgNameRequired            = "name is required"
	ErrMsgNameInvalidSlug         = "name must match slug pattern"
	ErrMsgDescriptionRequired     = "description is required"
	ErrMsgTypeInvalid             = "type must be: prompt, skill, or agent"
	ErrMsgMissingExecution        = "execution configuration is required"
	ErrMsgNameTooLong             = "name exceeds maximum length"
	ErrMsgDescriptionTooLong      = "description exceeds maximum length"
	ErrMsgSpecNil                 = "spec is nil"
	ErrMsgYAMLUnmarshalFailed     = "YAML unmarshal failed"
	ErrMsgCompileNotAvailable     = "compile not available in this version"

	// Reference resolution messages
	ErrMsgRefNotFound      = "referenced spec not found"
	ErrMsgRefCircular      = "circular reference detected"
	ErrMsgRefDepthExceeded = "reference resolution depth exceeded"
	ErrMsgRefMissingSlug   = "exons.ref requires slug attribute"
	ErrMsgRefInvalidSlug   = "invalid spec slug format"

	// Agent/compilation messages
	ErrMsgNotAnAgent              = "document is not an agent type"
	ErrMsgSkillNotFound           = "skill not found"
	ErrMsgSkillRefEmpty           = "skill reference slug is empty"
	ErrMsgSkillRefAmbiguous       = "skill reference is ambiguous"
	ErrMsgNoExecutionConfig       = "execution configuration is required"
	ErrMsgNoProvider              = "provider is required in execution config"
	ErrMsgNoModel                 = "model is required in execution config"
	ErrMsgCompilationFailed       = "agent compilation failed"
	ErrMsgInvalidDocumentType     = "invalid document type"
	ErrMsgPromptNoSkillsAllowed   = "prompt type does not support skills"
	ErrMsgPromptNoToolsAllowed    = "prompt type does not support tools"
	ErrMsgPromptNoConstraints     = "prompt type does not support constraints"
	ErrMsgAgentMessagesInvalid    = "agent messages must include system or user role"
	ErrMsgSkillRefInvalidVersion  = "invalid skill reference version"
	ErrMsgCatalogGenerationFailed = "catalog generation failed"
	ErrMsgSkillNoSkillsAllowed    = "skill type does not support nested skills"
	ErrMsgInvalidSkillInjection   = "invalid skill injection mode"
	ErrMsgMCPServerNameEmpty      = "MCP server name is empty"
	ErrMsgMCPServerURLEmpty       = "MCP server URL is empty"
	ErrMsgMessageTemplateNoRole   = "message template requires a role"
	ErrMsgMessageTemplateNoBody   = "message template requires content"
	ErrMsgInlineSkillNoSlug       = "inline skill requires a slug"
	ErrMsgInlineSkillNoBody       = "inline skill requires a body"
	ErrMsgAgentNoBodyOrMessages   = "agent requires body or messages"
	ErrMsgUnsupportedMsgProvider  = "unsupported provider for message serialization"
	ErrMsgNoDocumentResolver      = "no document resolver configured"

	// GenSpec error messages
	ErrMsgPromptNoMemory      = "prompt type does not support memory configuration"
	ErrMsgPromptNoDispatch    = "prompt type does not support dispatch rules"
	ErrMsgPromptNoRegistry    = "prompt type does not support registry metadata"
	ErrMsgSkillNoDispatch     = "skill type does not support dispatch rules"
	ErrMsgMemoryInvalidScope  = "memory scope must match slug pattern"
	ErrMsgDispatchMaxTurns    = "dispatch max_turns must be between 1 and 1000"
	ErrMsgDispatchCostLimit   = "dispatch cost_limit_usd must be between 0 and 1000"
	ErrMsgVerifyNameRequired  = "verification name is required"
	ErrMsgVerifyNameInvalid   = "verification name must match slug pattern"
	ErrMsgVerifyNoAssertions  = "verification expect must have at least one assertion"
	ErrMsgVerifyTimeout       = "verification timeout_seconds must be between 1 and 600"
	ErrMsgRegistryNamespace   = "registry namespace must match slug pattern"
	ErrMsgRegistryOrigin      = "registry origin must be: internal, external, or unknown"
	ErrMsgSafetyGuardrails    = "safety guardrails must be: enabled or disabled"
	ErrMsgTimeoutInvalid      = "timeout_seconds must be between 1 and 3600"
	ErrMsgMaxToolCallsInvalid = "max_tool_calls must be between 1 and 10000"

	// Additional validation messages
	ErrMsgInvalidSkopeSlug      = "invalid slug format"
	ErrMsgInvalidVisibility     = "invalid visibility value"
	ErrMsgVersionNumberNegative = "version number must be non-negative"

	// Import/export error messages
	ErrMsgImportFailed     = "import failed"
	ErrMsgImportZipFailed  = "zip import failed"
	ErrMsgImportNoDocument = "no document found in archive"
	ErrMsgImportReadFailed = "failed to read import file"
	ErrMsgExportFailed     = "export failed"
	ErrMsgExportZipFailed  = "zip export failed"

	// SKILL.md format error messages
	ErrMsgSkillMDInvalidFormat = "invalid SKILL.md format"
	ErrMsgSkillMDMissingFM     = "SKILL.md missing frontmatter"
	ErrMsgSkillMDParseFailed   = "SKILL.md parse failed"

	// Storage error messages
	ErrMsgCryptoRandFailure     = "cryptographic random number generator failure"
	ErrMsgPathTraversalDetected = "invalid template name: path traversal characters detected"
)

// Position represents a location in the source template
type Position struct {
	Offset int // Byte offset from start
	Line   int // 1-indexed line number
	Column int // 1-indexed column number
}

// String returns a human-readable position string
func (p Position) String() string {
	return fmt.Sprintf(FmtPosition, p.Line, p.Column)
}

// NewParseError creates a parse error with position context
func NewParseError(msg string, pos Position, cause error) error {
	var err *cuserr.CustomError
	if cause != nil {
		err = cuserr.WrapStdError(cause, ErrCodeParse, msg)
	} else {
		err = cuserr.NewValidationError(ErrCodeParse, msg)
	}
	return err.
		WithMetadata(MetaKeyLine, strconv.Itoa(pos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(pos.Column)).
		WithMetadata(MetaKeyOffset, strconv.Itoa(pos.Offset))
}

// NewUnterminatedTagError creates an error for unterminated tags
func NewUnterminatedTagError(pos Position) error {
	return cuserr.NewValidationError(ErrCodeParse, ErrMsgUnterminatedTag).
		WithMetadata(MetaKeyLine, strconv.Itoa(pos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(pos.Column)).
		WithMetadata(MetaKeyOffset, strconv.Itoa(pos.Offset))
}

// NewUnterminatedStrError creates an error for unterminated string literals
func NewUnterminatedStrError(pos Position) error {
	return cuserr.NewValidationError(ErrCodeParse, ErrMsgUnterminatedStr).
		WithMetadata(MetaKeyLine, strconv.Itoa(pos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(pos.Column)).
		WithMetadata(MetaKeyOffset, strconv.Itoa(pos.Offset))
}

// NewMismatchedTagError creates an error for mismatched closing tags
func NewMismatchedTagError(expected, actual string, pos Position) error {
	return cuserr.NewValidationError(ErrCodeParse, ErrMsgMismatchedTag).
		WithMetadata(MetaKeyLine, strconv.Itoa(pos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(pos.Column)).
		WithMetadata(MetaKeyExpected, expected).
		WithMetadata(MetaKeyActual, actual)
}

// NewNestedRawBlockError creates an error for nested raw blocks
func NewNestedRawBlockError(pos Position) error {
	return cuserr.NewValidationError(ErrCodeParse, ErrMsgNestedRawBlock).
		WithMetadata(MetaKeyLine, strconv.Itoa(pos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(pos.Column)).
		WithMetadata(MetaKeyOffset, strconv.Itoa(pos.Offset))
}

// NewExecutionError creates an execution error with tag context
func NewExecutionError(msg string, tagName string, pos Position, cause error) error {
	var err *cuserr.CustomError
	if cause != nil {
		err = cuserr.WrapStdError(cause, ErrCodeExec, msg)
	} else {
		err = cuserr.NewValidationError(ErrCodeExec, msg)
	}
	return err.
		WithMetadata(MetaKeyTag, tagName).
		WithMetadata(MetaKeyLine, strconv.Itoa(pos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(pos.Column))
}

// NewVariableNotFoundError creates a variable not found error
func NewVariableNotFoundError(path string) error {
	return cuserr.NewNotFoundError(MetaKeyVariable, ErrMsgVariableNotFound).
		WithMetadata(MetaKeyPath, path)
}

// NewUnknownTagError creates an unknown tag error
func NewUnknownTagError(tagName string, pos Position) error {
	return cuserr.NewNotFoundError(MetaKeyResolver, ErrMsgUnknownResolver).
		WithMetadata(MetaKeyTag, tagName).
		WithMetadata(MetaKeyLine, strconv.Itoa(pos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(pos.Column))
}

// NewResolverExistsError creates a resolver collision error
func NewResolverExistsError(tagName string) error {
	return cuserr.NewValidationError(ErrCodeRegistry, ErrMsgResolverExists).
		WithMetadata(MetaKeyTag, tagName)
}

// NewMissingAttributeError creates a missing required attribute error
func NewMissingAttributeError(attrName string, tagName string) error {
	return cuserr.NewValidationError(ErrCodeValidation, ErrMsgMissingAttribute).
		WithMetadata(MetaKeyAttribute, attrName).
		WithMetadata(MetaKeyTag, tagName)
}

// NewInvalidAttributeError creates an invalid attribute value error
func NewInvalidAttributeError(attrName string, value string, reason string) error {
	return cuserr.NewValidationError(ErrCodeValidation, ErrMsgInvalidAttribute).
		WithMetadata(MetaKeyAttribute, attrName).
		WithMetadata(MetaKeyValue, value).
		WithMetadata(MetaKeyReason, reason)
}

// NewResolverError creates an error for resolver failures
func NewResolverError(resolverName string, cause error) error {
	return cuserr.WrapStdError(cause, ErrCodeExec, ErrMsgResolverFailed).
		WithMetadata(MetaKeyResolver, resolverName)
}

// NewTypeConversionError creates a type conversion error
func NewTypeConversionError(fromType, toType string, value any) error {
	return cuserr.NewValidationError(ErrCodeExec, ErrMsgTypeConversion).
		WithMetadata(MetaKeyFromType, fromType).
		WithMetadata(MetaKeyToType, toType).
		WithMetadata(MetaKeyValue, fmt.Sprintf("%v", value))
}

// NewTemplateNotFoundError creates an error for missing templates
func NewTemplateNotFoundError(name string) error {
	return cuserr.NewNotFoundError(MetaKeyTemplateName, ErrMsgTemplateNotFound).
		WithMetadata(MetaKeyTemplateName, name)
}

// NewTemplateExistsError creates an error for duplicate template registration
func NewTemplateExistsError(name string) error {
	return cuserr.NewValidationError(ErrCodeTemplate, ErrMsgTemplateAlreadyExists).
		WithMetadata(MetaKeyTemplateName, name)
}

// NewTemplateDepthError creates an error when max inclusion depth is exceeded
func NewTemplateDepthError(depth, max int) error {
	return cuserr.NewValidationError(ErrCodeTemplate, ErrMsgTemplateDepthExceeded).
		WithMetadata(MetaKeyCurrentDepth, strconv.Itoa(depth)).
		WithMetadata(MetaKeyMaxDepth, strconv.Itoa(max))
}

// NewReservedTemplateNameError creates an error for reserved namespace usage
func NewReservedTemplateNameError(name string) error {
	return cuserr.NewValidationError(ErrCodeTemplate, ErrMsgReservedTemplateName).
		WithMetadata(MetaKeyTemplateName, name)
}

// NewEmptyTemplateNameError creates an error for empty template names
func NewEmptyTemplateNameError() error {
	return cuserr.NewValidationError(ErrCodeTemplate, ErrMsgEmptyTemplateName)
}

// NewEngineNotAvailableError creates an error when engine is not in context
func NewEngineNotAvailableError() error {
	return cuserr.NewInternalError(ErrCodeTemplate, nil).
		WithMetadata(MetaKeyTag, TagNameInclude)
}

// NewFuncRegistrationError creates an error for function registration failures
func NewFuncRegistrationError(msg, funcName string) error {
	err := cuserr.NewValidationError(ErrCodeFunc, msg)
	if funcName != "" {
		err = err.WithMetadata(MetaKeyFuncName, funcName)
	}
	return err
}

// NewEnvVarNotFoundError creates an error for environment variable not found
func NewEnvVarNotFoundError(varName string) error {
	return cuserr.NewNotFoundError(ErrCodeEnv, ErrMsgEnvVarNotFound).
		WithMetadata(MetaKeyEnvVar, varName)
}

// NewEnvVarRequiredError creates an error for required environment variable not set
func NewEnvVarRequiredError(varName string) error {
	return cuserr.NewValidationError(ErrCodeEnv, ErrMsgEnvVarRequired).
		WithMetadata(MetaKeyEnvVar, varName)
}

// NewConfigBlockError creates an error for config block parsing failures
func NewConfigBlockError(msg string, pos Position, cause error) error {
	var err *cuserr.CustomError
	if cause != nil {
		err = cuserr.WrapStdError(cause, ErrCodeConfig, msg)
	} else {
		err = cuserr.NewValidationError(ErrCodeConfig, msg)
	}
	return err.
		WithMetadata(MetaKeyLine, strconv.Itoa(pos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(pos.Column)).
		WithMetadata(MetaKeyOffset, strconv.Itoa(pos.Offset))
}

// NewConfigBlockParseError creates an error for config block JSON parsing failures
func NewConfigBlockParseError(cause error) error {
	return cuserr.WrapStdError(cause, ErrCodeConfig, ErrMsgConfigBlockParse)
}

// NewInputValidationError creates an error for input validation failures
func NewInputValidationError(inputName, reason string) error {
	return cuserr.NewValidationError(ErrCodeConfig, ErrMsgInputValidationFailed).
		WithMetadata(MetaKeyInputName, inputName).
		WithMetadata(MetaKeyReason, reason)
}

// NewRequiredInputMissingError creates an error for missing required input
func NewRequiredInputMissingError(inputName string) error {
	return cuserr.NewValidationError(ErrCodeConfig, ErrMsgRequiredInputMissing).
		WithMetadata(MetaKeyInputName, inputName)
}

// NewFrontmatterError creates an error for YAML frontmatter extraction failures
func NewFrontmatterError(msg string, pos Position, cause error) error {
	var err *cuserr.CustomError
	if cause != nil {
		err = cuserr.WrapStdError(cause, ErrCodeConfig, msg)
	} else {
		err = cuserr.NewValidationError(ErrCodeConfig, msg)
	}
	return err.
		WithMetadata(MetaKeyLine, strconv.Itoa(pos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(pos.Column)).
		WithMetadata(MetaKeyOffset, strconv.Itoa(pos.Offset))
}

// NewFrontmatterParseError creates an error for YAML frontmatter parsing failures
func NewFrontmatterParseError(cause error) error {
	return cuserr.WrapStdError(cause, ErrCodeConfig, ErrMsgFrontmatterParse)
}

// NewMessageTagError creates an error for message tag validation failures
func NewMessageTagError(msg string, tagPos Position) error {
	return cuserr.NewValidationError(ErrCodeValidation, msg).
		WithMetadata(MetaKeyTag, TagNameMessage).
		WithMetadata(MetaKeyLine, strconv.Itoa(tagPos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(tagPos.Column))
}

// NewLabelNotFoundError creates an error for label not found.
func NewLabelNotFoundError(templateName, label string) error {
	return cuserr.NewNotFoundError(ErrCodeLabel, ErrMsgLabelNotFound).
		WithMetadata(MetaKeyTemplateName, templateName).
		WithMetadata(MetaKeyLabel, label)
}

// NewInvalidLabelNameError creates an error for invalid label name.
func NewInvalidLabelNameError(label, reason string) error {
	return cuserr.NewValidationError(ErrCodeLabel, ErrMsgInvalidLabelName).
		WithMetadata(MetaKeyLabel, label).
		WithMetadata(MetaKeyReason, reason)
}

// NewInvalidStatusTransitionError creates an error for invalid status transition.
func NewInvalidStatusTransitionError(from, to DeploymentStatus) error {
	return cuserr.NewValidationError(ErrCodeStatus, ErrMsgStatusTransitionDenied).
		WithMetadata(MetaKeyFromStatus, string(from)).
		WithMetadata(MetaKeyToStatus, string(to))
}

// NewArchivedVersionError creates an error for operations on archived versions.
func NewArchivedVersionError(templateName string, version int) error {
	return cuserr.NewValidationError(ErrCodeStatus, ErrMsgArchivedVersionReadOnly).
		WithMetadata(MetaKeyTemplateName, templateName).
		WithMetadata(MetaKeyVersion, strconv.Itoa(version))
}

// NewInvalidDeploymentStatusError creates an error for invalid deployment status value.
func NewInvalidDeploymentStatusError(status string) error {
	return cuserr.NewValidationError(ErrCodeStatus, ErrMsgInvalidDeploymentStatus).
		WithMetadata(MetaKeyStatus, status)
}

// NewSchemaValidationError creates an error for schema validation failures.
func NewSchemaValidationError(msg, path string) error {
	return cuserr.NewValidationError(ErrCodeSchema, msg).
		WithMetadata(MetaKeyPath, path)
}

// NewSchemaProviderError creates an error for provider-specific schema issues.
func NewSchemaProviderError(msg, provider string) error {
	return cuserr.NewValidationError(ErrCodeSchema, msg).
		WithMetadata(MetaKeyProvider, provider)
}

// NewSpecValidationError creates an error for spec validation failures.
func NewSpecValidationError(msg, specName string) error {
	return cuserr.NewValidationError(ErrCodeSpec, msg).
		WithMetadata(MetaKeySpecName, specName)
}

// NewSpecNameRequiredError creates an error for missing spec name.
func NewSpecNameRequiredError() error {
	return cuserr.NewValidationError(ErrCodeSpec, ErrMsgSpecNameRequired)
}

// NewSpecNameTooLongError creates an error for spec name exceeding max length.
func NewSpecNameTooLongError(name string, maxLen int) error {
	return cuserr.NewValidationError(ErrCodeSpec, ErrMsgSpecNameTooLong).
		WithMetadata(MetaKeySpecName, name).
		WithMetadata(MetaKeyMaxDepth, strconv.Itoa(maxLen))
}

// NewSpecNameInvalidFormatError creates an error for invalid spec name format.
func NewSpecNameInvalidFormatError(name string) error {
	return cuserr.NewValidationError(ErrCodeSpec, ErrMsgSpecNameInvalidFormat).
		WithMetadata(MetaKeySpecName, name)
}

// NewSpecDescriptionRequiredError creates an error for missing spec description.
func NewSpecDescriptionRequiredError() error {
	return cuserr.NewValidationError(ErrCodeSpec, ErrMsgSpecDescriptionRequired)
}

// NewSpecDescriptionTooLongError creates an error for spec description exceeding max length.
func NewSpecDescriptionTooLongError(maxLen int) error {
	return cuserr.NewValidationError(ErrCodeSpec, ErrMsgSpecDescriptionTooLong).
		WithMetadata(MetaKeyMaxDepth, strconv.Itoa(maxLen))
}

// NewRefNotFoundError creates an error for referenced spec not found.
func NewRefNotFoundError(slug, version string) error {
	return cuserr.NewNotFoundError(ErrCodeRef, ErrMsgRefNotFound).
		WithMetadata(MetaKeySpecSlug, slug).
		WithMetadata(AttrVersion, version)
}

// NewRefCircularError creates an error for circular reference detection.
func NewRefCircularError(slug string, chain []string) error {
	chainStr := ""
	for i, s := range chain {
		if i > 0 {
			chainStr += RefChainSeparator
		}
		chainStr += s
	}
	return cuserr.NewValidationError(ErrCodeRef, ErrMsgRefCircular).
		WithMetadata(MetaKeySpecSlug, slug).
		WithMetadata(MetaKeyRefChain, chainStr)
}

// NewRefDepthExceededError creates an error for reference resolution depth exceeded.
func NewRefDepthExceededError(depth, maxDepth int) error {
	return cuserr.NewValidationError(ErrCodeRef, ErrMsgRefDepthExceeded).
		WithMetadata(MetaKeyCurrentDepth, strconv.Itoa(depth)).
		WithMetadata(MetaKeyMaxDepth, strconv.Itoa(maxDepth))
}

// NewRefMissingSlugError creates an error for missing slug attribute in exons.ref.
func NewRefMissingSlugError() error {
	return cuserr.NewValidationError(ErrCodeRef, ErrMsgRefMissingSlug).
		WithMetadata(MetaKeyTag, TagNameRef)
}

// NewRefInvalidSlugError creates an error for invalid slug format in exons.ref.
func NewRefInvalidSlugError(slug string) error {
	return cuserr.NewValidationError(ErrCodeRef, ErrMsgRefInvalidSlug).
		WithMetadata(MetaKeySpecSlug, slug)
}

// NewCompileNotAvailableError creates an error for compile methods that are not yet available.
func NewCompileNotAvailableError() error {
	return cuserr.NewValidationError(ErrCodeCompile, ErrMsgCompileNotAvailable)
}

// NewSerializeError creates an error for serialization failures.
func NewSerializeError(msg string, cause error) error {
	if cause != nil {
		return cuserr.WrapStdError(cause, ErrCodeSerialize, msg)
	}
	return cuserr.NewValidationError(ErrCodeSerialize, msg)
}

// NewCatalogError creates an error for catalog generation failures.
func NewCatalogError(msg string, cause error) error {
	if cause != nil {
		return cuserr.WrapStdError(cause, ErrCodeCatalog, msg)
	}
	return cuserr.NewValidationError(ErrCodeCatalog, msg)
}

// NewCatalogFormatError creates an error for unsupported catalog format.
func NewCatalogFormatError(msg string, format string) error {
	return cuserr.NewValidationError(ErrCodeCatalog, msg).
		WithMetadata(MetaKeyFormat, format)
}

// MetaKeyFormat is the metadata key for format values in catalog errors.
const MetaKeyFormat = "format"

// NewExportError creates an error for document export failures.
func NewExportError(msg string, cause error) error {
	if cause != nil {
		return cuserr.WrapStdError(cause, ErrCodeSerialize, msg)
	}
	return cuserr.NewValidationError(ErrCodeSerialize, msg)
}

// NewImportError creates an error for document import failures.
func NewImportError(msg string, cause error) error {
	if cause != nil {
		return cuserr.WrapStdError(cause, ErrCodeSpec, msg)
	}
	return cuserr.NewValidationError(ErrCodeSpec, msg)
}

// NewAgentValidationError creates an error for agent-specific validation failures.
func NewAgentValidationError(msg string, specName string) error {
	return cuserr.NewValidationError(ErrCodeAgent, msg).
		WithMetadata(MetaKeySpecName, specName)
}
