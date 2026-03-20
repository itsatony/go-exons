// Package internal contains the template engine internals: lexer, parser,
// executor, expression evaluator, and function registry.
//
// This package is not part of the public API.
package internal

// TokenType represents the type of a lexical token
type TokenType string

// Token type constants
const (
	TokenTypeText       TokenType = "TEXT"
	TokenTypeOpenTag    TokenType = "OPEN_TAG"
	TokenTypeCloseTag   TokenType = "CLOSE_TAG"
	TokenTypeSelfClose  TokenType = "SELF_CLOSE"
	TokenTypeBlockClose TokenType = "BLOCK_CLOSE"
	TokenTypeTagName    TokenType = "TAG_NAME"
	TokenTypeAttrName   TokenType = "ATTR_NAME"
	TokenTypeAttrValue  TokenType = "ATTR_VALUE"
	TokenTypeEquals     TokenType = "EQUALS"
	TokenTypeEOF        TokenType = "EOF"
)

// NodeType identifies AST node types
type NodeType int

// Node type constants
const (
	NodeTypeRoot NodeType = iota
	NodeTypeText
	NodeTypeTag
	NodeTypeRaw
	NodeTypeConditional
	NodeTypeFor    // Loop node
	NodeTypeSwitch // Switch/case node
)

// Node type string names for debugging
const (
	NodeTypeNameRoot        = "ROOT"
	NodeTypeNameText        = "TEXT"
	NodeTypeNameTag         = "TAG"
	NodeTypeNameRaw         = "RAW"
	NodeTypeNameConditional = "CONDITIONAL"
	NodeTypeNameFor         = "FOR"
	NodeTypeNameSwitch      = "SWITCH"
)

// String returns the string representation of the node type
func (n NodeType) String() string {
	switch n {
	case NodeTypeRoot:
		return NodeTypeNameRoot
	case NodeTypeText:
		return NodeTypeNameText
	case NodeTypeTag:
		return NodeTypeNameTag
	case NodeTypeRaw:
		return NodeTypeNameRaw
	case NodeTypeConditional:
		return NodeTypeNameConditional
	case NodeTypeFor:
		return NodeTypeNameFor
	case NodeTypeSwitch:
		return NodeTypeNameSwitch
	default:
		return NodeTypeNameRoot
	}
}

// Lexer state constants
const (
	LexStateText  = "TEXT"
	LexStateTag   = "TAG"
	LexStateAttr  = "ATTR"
	LexStateValue = "VALUE"
)

// Character constants
const (
	CharEquals      = '='
	CharDoubleQuote = '"'
	CharSingleQuote = '\''
	CharBackslash   = '\\'
	CharSlash       = '/'
	CharNewline     = '\n'
	CharSpace       = ' '
	CharTab         = '\t'
	CharCarriageRet = '\r'
	CharNullByte    = "\x00" // String for use with strings.ReplaceAll (security: marker sanitization)
)

// String constants for delimiter matching
const (
	StrOpenDelim  = "{~"
	StrCloseDelim = "~}"
	StrSelfClose  = "/~}"
	StrBlockClose = "{~/"
	StrEscapeOpen = "\\{~"
)

// Delimiter lengths
const (
	LenOpenDelim  = 2 // {~
	LenCloseDelim = 2 // ~}
	LenSelfClose  = 3 // /~}
	LenBlockClose = 3 // {~/
	LenEscapeOpen = 3 // \{~
)

// Log message constants
const (
	LogMsgLexerCreated       = "lexer created"
	LogMsgTokenizerStart     = "starting tokenization"
	LogMsgTokenizerEnd       = "tokenization complete"
	LogMsgParserCreated      = "parser created"
	LogMsgParserStart        = "starting parse"
	LogMsgParserEnd          = "parse complete"
	LogMsgExecutorCreated    = "executor created"
	LogMsgExecutorStart      = "starting execution"
	LogMsgExecutorEnd        = "execution complete"
	LogMsgResolverInvoked    = "resolver invoked"
	LogMsgResolverComplete   = "resolver complete"
	LogMsgRegistryCreated    = "registry created"
	LogMsgResolverRegistered = "resolver registered"
	LogMsgResolverCollision  = "resolver registration collision - first-come-wins"
)

// Log field names
const (
	LogFieldSource       = "source_length"
	LogFieldTokens       = "token_count"
	LogFieldNodes        = "node_count"
	LogFieldTag          = "tag"
	LogFieldResolver     = "resolver"
	LogFieldDuration     = "duration"
	LogFieldLine         = "line"
	LogFieldColumn       = "column"
	LogFieldTemplateName = "template_name"
	LogFieldDepth        = "depth"
	LogFieldBranch       = "branch"
	LogFieldBranches     = "branches"
	LogFieldIsElse       = "is_else"
	LogFieldCondition    = "condition"
	LogFieldExpression   = "expression"
	LogFieldResult       = "result"
)

// Tag namespace prefix
const TagPrefix = "exons."

// Built-in tag names
const (
	TagNameVar           = "exons.var"
	TagNameRaw           = "exons.raw"
	TagNameInclude       = "exons.include"
	TagNameIf            = "exons.if"
	TagNameElseIf        = "exons.elseif"
	TagNameElse          = "exons.else"
	TagNameComment       = "exons.comment"
	TagNameFor           = "exons.for"
	TagNameSwitch        = "exons.switch"
	TagNameCase          = "exons.case"
	TagNameCaseDefault   = "exons.casedefault"
	TagNameEnv           = "exons.env"
	TagNameConfig        = "exons.config" // Legacy configuration block (JSON)
	TagNameExtends       = "exons.extends"
	TagNameBlock         = "exons.block"
	TagNameParent        = "exons.parent"
	TagNameRef           = "exons.ref"
	TagNameSkillsCatalog = "exons.skills_catalog"
	TagNameToolsCatalog  = "exons.tools_catalog"
	// TagNameMessage is defined separately in the message tag constants section
)

// Attribute name constants
const (
	AttrName     = "name"
	AttrDefault  = "default"
	AttrTemplate = "template"
	AttrWith     = "with"
	AttrIsolate  = "isolate"
	AttrEval     = "eval"
	AttrOnError  = "onerror"
	AttrItem     = "item"
	AttrIndex    = "index"
	AttrIn       = "in"
	AttrLimit    = "limit"
	AttrValue    = "value"
	AttrRequired = "required"
	AttrSlug     = "slug"
	AttrVersion  = "version"
)

// Boolean attribute values
const (
	AttrValueTrue  = "true"
	AttrValueFalse = "false"
)

// Error message constants for include resolver
const (
	ErrMsgMissingTemplateAttr = "missing required 'template' attribute"
	ErrMsgEngineNotAvailable  = "engine not available in context"
	ErrMsgDepthExceeded       = "maximum template inclusion depth exceeded"
)

// Meta key constants for internal data passing and error metadata
const (
	MetaKeyParentDepth  = "_parentDepth"
	MetaKeyValue        = "_value"
	MetaKeyPath         = "path"
	MetaKeyTemplateName = "template_name"
	MetaKeyFromType     = "from_type"
	MetaKeyToType       = "to_type"
	MetaKeyIterableType = "iterable_type"
)

// Log messages for template operations
const (
	LogMsgTemplateRegistered = "template registered"
	LogMsgTemplateIncluded   = "template included"
	LogMsgIncludeDepthCheck  = "checking include depth"
)

// Error messages for conditional resolver
const (
	ErrMsgCondMissingEval   = "missing required 'eval' attribute"
	ErrMsgCondInvalidElse   = "else tag cannot have eval attribute"
	ErrMsgCondUnexpectedTag = "unexpected conditional tag"
	ErrMsgCondNotClosed     = "conditional block not closed"
	ErrMsgCondElseNotLast   = "else must be last in conditional chain"
	ErrMsgCondExprFailed    = "condition expression evaluation failed"
)

// Log messages for conditional operations
const (
	LogMsgConditionEval  = "evaluating condition"
	LogMsgBranchSelected = "branch selected"
)

// Error format string constants (for Error() methods)
const (
	ErrFmtWithPosition       = "%s at %s"
	ErrFmtWithTagAndPosition = "%s [%s] at %s"
	ErrFmtWithCause          = "%s: %v"
	ErrFmtTagMessage         = "%s: %s"
)

// String format constants for AST String() methods
const (
	FmtOpenBrace   = "{"
	FmtCloseBrace  = "}"
	FmtCommaSep    = ", "
	FmtKeyValueSep = "="
	FmtEmptyBraces = "{}"
)

// String display constants for truncation
const (
	MaxStringDisplayLength = 50
	TruncatedStringLength  = 47
	TruncationSuffix       = "..."
)

// ErrorStrategy mirrors exons.ErrorStrategy for internal use
type ErrorStrategy int

// Error strategy constants
const (
	ErrorStrategyThrow ErrorStrategy = iota
	ErrorStrategyDefault
	ErrorStrategyRemove
	ErrorStrategyKeepRaw
	ErrorStrategyLog
)

// ErrorStrategyNotSet is a sentinel value indicating no strategy override
const ErrorStrategyNotSet ErrorStrategy = -1

// Error strategy name constants for parsing
const (
	ErrorStrategyNameThrow   = "throw"
	ErrorStrategyNameDefault = "default"
	ErrorStrategyNameRemove  = "remove"
	ErrorStrategyNameKeepRaw = "keepraw"
	ErrorStrategyNameLog     = "log"
)

// ParseErrorStrategy parses a string into an ErrorStrategy.
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

// String returns the string representation of the error strategy.
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

// Log messages for error strategy handling
const (
	LogMsgErrorStrategyApplied = "error strategy applied"
	LogMsgErrorLogged          = "error logged and execution continued"
	LogFieldStrategy           = "strategy"
	LogFieldErrorMsg           = "error_message"
)

// Error messages for for loop
const (
	ErrMsgForMissingItem    = "missing required 'item' attribute"
	ErrMsgForMissingIn      = "missing required 'in' attribute"
	ErrMsgForInvalidLimit   = "invalid 'limit' attribute value"
	ErrMsgForCollectionPath = "collection path not found"
	ErrMsgForNotIterable    = "value is not iterable"
	ErrMsgForLimitExceeded  = "loop iteration limit exceeded"
	ErrMsgForNotClosed      = "for block not closed"
	ErrMsgForContextNoChild = "context does not support child creation"
)

// Log messages for for loop operations
const (
	LogMsgForStart        = "starting for loop"
	LogMsgForIteration    = "for loop iteration"
	LogMsgForEnd          = "for loop complete"
	LogMsgForLimitApplied = "loop limit applied"
)

// Log field names for for loop
const (
	LogFieldIteration  = "iteration"
	LogFieldCollection = "collection"
	LogFieldItemVar    = "item_var"
	LogFieldIndexVar   = "index_var"
)

// Default values for for loop
const (
	DefaultMaxLoopIterations = 10000
)

// Map iteration field names
// When iterating over a map, each item is a map with these keys.
const (
	ForMapKeyField   = "key"
	ForMapValueField = "value"
)

// Error messages for switch/case
const (
	ErrMsgSwitchMissingEval      = "missing required 'eval' attribute for switch"
	ErrMsgSwitchMissingValue     = "case requires 'value' or 'eval' attribute"
	ErrMsgSwitchNotClosed        = "switch block not closed"
	ErrMsgSwitchCaseNotClosed    = "case block not closed"
	ErrMsgSwitchDefaultNotLast   = "default case must be last in switch"
	ErrMsgSwitchDuplicateDefault = "only one default case allowed in switch"
	ErrMsgSwitchInvalidCaseTag   = "unexpected tag inside switch block"
)

// Log messages for switch/case operations
const (
	LogMsgSwitchEval    = "evaluating switch expression"
	LogMsgSwitchCase    = "evaluating switch case"
	LogMsgCaseMatch     = "switch case matched"
	LogMsgCaseDefault   = "switch default case selected"
	LogMsgSwitchNoMatch = "no switch case matched"
)

// Log field names for switch/case
const (
	LogFieldCaseValue = "case_value"
	LogFieldCaseEval  = "case_eval"
)

// Error messages for env resolver
const (
	ErrMsgEnvVarNotFound = "environment variable not found"
	ErrMsgEnvVarRequired = "required environment variable not set"
	ErrMsgEnvVarEmpty    = "environment variable is empty"
)

// Log messages for env resolver operations
const (
	LogMsgEnvResolve = "resolving environment variable"
)

// Log field names for env resolver
const (
	LogFieldEnvVar = "env_var"
)

// Meta key constants for env resolver
const (
	MetaKeyEnvVar = "env_var"
)

// Error messages for config block (legacy JSON - kept for migration hints)
const (
	ErrMsgConfigBlockExtract  = "failed to extract config block"
	ErrMsgConfigBlockParse    = "failed to parse config block JSON"
	ErrMsgConfigBlockInvalid  = "invalid config block format"
	ErrMsgConfigBlockUnclosed = "config block not properly closed"
)

// YAML frontmatter constants
const (
	YAMLFrontmatterDelimiter            = "---"
	YAMLFrontmatterDelimiterWithNewline = "---\n"
	YAMLFrontmatterDelimiterWithCRLF    = "---\r\n"
)

// Error messages for YAML frontmatter
const (
	ErrMsgFrontmatterExtract       = "failed to extract YAML frontmatter"
	ErrMsgFrontmatterParse         = "failed to parse YAML frontmatter"
	ErrMsgFrontmatterInvalid       = "invalid YAML frontmatter format"
	ErrMsgFrontmatterUnclosed      = "YAML frontmatter not properly closed"
	ErrMsgFrontmatterNotAtStart    = "YAML frontmatter must be at start of template"
	ErrMsgLegacyJSONConfigDetected = "legacy JSON config block detected - please migrate to YAML frontmatter with --- delimiters"
)

// Message tag constants
const (
	TagNameMessage = "exons.message"
)

// Message role constants
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

// Error messages for message tag
const (
	ErrMsgMessageMissingRole      = "missing required 'role' attribute"
	ErrMsgMessageInvalidRole      = "invalid role - must be system, user, assistant, or tool"
	ErrMsgMessageNestedNotAllowed = "nested message tags are not allowed"
)

// Log messages for message tag
const (
	LogMsgMessageExtracted = "message extracted"
	LogMsgMessageRole      = "message_role"
)

// Error messages for template inheritance
const (
	ErrMsgExtendsNotFirst          = "extends must be first tag in template"
	ErrMsgExtendsMultiple          = "only one extends allowed per template"
	ErrMsgExtendsMissingTemplate   = "missing required 'template' attribute for extends"
	ErrMsgBlockMissingName         = "missing required 'name' attribute for block"
	ErrMsgBlockDuplicateName       = "duplicate block name"
	ErrMsgBlockNotClosed           = "block not properly closed"
	ErrMsgParentOutsideBlock       = "parent can only be used inside a block"
	ErrMsgCircularInheritance      = "circular template inheritance detected"
	ErrMsgInheritanceDepthExceeded = "template inheritance depth exceeded"
)

// Log messages for template inheritance
const (
	LogMsgExtendsFound        = "template extends found"
	LogMsgBlockDefined        = "block defined"
	LogMsgBlockOverride       = "block override"
	LogMsgParentBlockInserted = "parent block content inserted"
	LogMsgInheritanceResolved = "template inheritance resolved"
)

// Log field names for template inheritance
const (
	LogFieldParentTemplate   = "parent_template"
	LogFieldBlockName        = "block_name"
	LogFieldInheritanceDepth = "inheritance_depth"
)

// Default values for template inheritance
const (
	DefaultMaxInheritanceDepth = 10
)

// Node type for inheritance
const (
	NodeTypeBlock NodeType = iota + 100
)

// Reference resolver constants
const (
	RefMaxDepth      = 10
	RefVersionLatest = "latest"
)

// Special template name for self-reference in compilation
const (
	TemplateNameSelf = "self"
)

// Context keys used during agent compilation
const (
	ContextKeySelfBody = "_selfBody"
	ContextKeySkills   = "skills"
	ContextKeyTools    = "tools"
)

// Error messages for reference resolver
const (
	ErrMsgRefMissingSlug   = "missing required 'slug' attribute"
	ErrMsgRefNotFound      = "referenced spec not found"
	ErrMsgRefCircular      = "circular reference detected"
	ErrMsgRefDepthExceeded = "reference resolution depth exceeded"
	ErrMsgRefNoResolver    = "spec resolver not available in context"
	ErrMsgRefInvalidSlug   = "invalid spec slug format"
)

// Log messages for reference resolver
const (
	LogMsgRefResolve  = "resolving spec reference"
	LogMsgRefResolved = "spec reference resolved"
)

// Log field names for reference resolver
const (
	LogFieldSpecSlug    = "spec_slug"
	LogFieldSpecVersion = "spec_version"
	LogFieldRefChain    = "ref_chain"
)

// Meta key constants for reference resolver
const (
	MetaKeySpecSlug     = "spec_slug"
	MetaKeyRefChain     = "ref_chain"
	MetaKeyRefDepth     = "_refDepth"
	MetaKeyRefChainList = "_refChainList"
)

// String value constants
const (
	StringValueEmpty = ""
)

// Parser error message constants
const (
	ErrMsgUnexpectedToken = "unexpected token"
	ErrMsgExpectedToken   = "expected token"
	ErrMsgMismatchedTag   = "mismatched closing tag"
	ErrMsgNestedRawBlock  = "nested raw blocks are not allowed"
)

// Error message constants for lexer
const (
	ErrMsgUnterminatedTag = "unterminated tag"
	ErrMsgUnterminatedStr = "unterminated string literal"
	ErrMsgInvalidTagName  = "invalid tag name"
	ErrMsgUnexpectedChar  = "unexpected character"
)
