package exons

// Error code constants for cuserr integration.
const (
	ErrCodeParse      = "EXONS_PARSE"
	ErrCodeCompile    = "EXONS_COMPILE"
	ErrCodeExecution  = "EXONS_EXECUTION"
	ErrCodeValidation = "EXONS_VALIDATION"
	ErrCodeStorage    = "EXONS_STORAGE"
	ErrCodeCredential = "EXONS_CREDENTIAL"
	ErrCodeManifest   = "EXONS_MANIFEST"
	ErrCodeA2A        = "EXONS_A2A"
	ErrCodeVersioning = "EXONS_VERSIONING"
	ErrCodeGenSpec    = "EXONS_GENSPEC"
)

// Validation error messages.
const (
	ErrMsgNameRequired         = "name is required"
	ErrMsgNameInvalidSlug      = "name must match slug pattern"
	ErrMsgDescriptionRequired  = "description is required"
	ErrMsgTypeInvalid          = "type must be: prompt, skill, or agent"
	ErrMsgMissingAttribute     = "required attribute missing"
	ErrMsgMissingExecution     = "execution configuration is required"
	ErrMsgTemperatureOutOfRange = "temperature must be between 0.0 and 2.0"
	ErrMsgTopPOutOfRange       = "top_p must be between 0.0 and 1.0"
	ErrMsgMaxTokensInvalid     = "max_tokens must be greater than 0"
)

// GenSpec error messages.
const (
	ErrMsgPromptNoMemory     = "prompt type does not support memory configuration"
	ErrMsgPromptNoDispatch   = "prompt type does not support dispatch rules"
	ErrMsgPromptNoRegistry   = "prompt type does not support registry metadata"
	ErrMsgSkillNoDispatch    = "skill type does not support dispatch rules"
	ErrMsgMemoryInvalidScope = "memory scope must match slug pattern"
	ErrMsgDispatchMaxTurns   = "dispatch max_turns must be between 1 and 1000"
	ErrMsgDispatchCostLimit  = "dispatch cost_limit_usd must be between 0 and 1000"
	ErrMsgVerifyNameRequired = "verification name is required"
	ErrMsgVerifyNameInvalid  = "verification name must match slug pattern"
	ErrMsgVerifyNoAssertions = "verification expect must have at least one assertion"
	ErrMsgVerifyTimeout      = "verification timeout_seconds must be between 1 and 600"
	ErrMsgRegistryNamespace  = "registry namespace must match slug pattern"
	ErrMsgRegistryOrigin     = "registry origin must be: internal, external, or unknown"
	ErrMsgSafetyGuardrails   = "safety guardrails must be: enabled or disabled"
	ErrMsgTimeoutInvalid     = "timeout_seconds must be between 1 and 3600"
	ErrMsgMaxToolCallsInvalid = "max_tool_calls must be between 1 and 10000"
)
