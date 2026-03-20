package exons

// Version is the current library version. Loaded from versions.yaml at build time.
const Version = "0.1.0"

// File extension for exons specification files.
const FileExtensionExons = ".exons"

// Document types determine validation rules and compilation behavior.
const (
	DocumentTypePrompt = "prompt"
	DocumentTypeSkill  = "skill"
	DocumentTypeAgent  = "agent"
)

// Tag namespace prefix for built-in tags.
const TagNamespacePrefix = "exons."

// Built-in tag names.
const (
	TagNameVar     = "exons.var"
	TagNameIf      = "exons.if"
	TagNameElseIf  = "exons.elseif"
	TagNameElse    = "exons.else"
	TagNameFor     = "exons.for"
	TagNameRaw     = "exons.raw"
	TagNameComment = "exons.comment"
	TagNameInclude = "exons.include"
	TagNameMessage = "exons.message"
	TagNameRef     = "exons.ref"
)

// Catalog tag names.
const (
	TagNameSkillsCatalog = "exons.skills_catalog"
	TagNameToolsCatalog  = "exons.tools_catalog"
)

// Message roles.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// Provider identifiers.
const (
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
	ProviderGemini    = "gemini"
	ProviderVLLM      = "vllm"
	ProviderMistral   = "mistral"
	ProviderCohere    = "cohere"
	ProviderAzure     = "azure"
)

// GenSpec version.
const GenSpecVersion = "1"

// Default execution limits.
const (
	DefaultResolverTimeout  = 5  // seconds
	DefaultFunctionTimeout  = 1  // seconds
	DefaultExecutionTimeout = 30 // seconds
	DefaultMaxLoopIter      = 10000
	DefaultMaxOutputSize    = 10 * 1024 * 1024 // 10MB
	DefaultMaxDepth         = 10
)
