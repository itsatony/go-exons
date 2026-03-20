package internal

import (
	"context"
	"fmt"
	"strconv"
)

// ContextAccessor is the interface for accessing context data.
// This mirrors exons.Context to avoid import cycles.
type ContextAccessor interface {
	Get(path string) (any, bool)
	GetString(path string) string
	GetStringDefault(path, defaultVal string) string
	Has(path string) bool
}

// TemplateContextAccessor extends ContextAccessor with template execution capabilities.
// Resolvers that need to execute nested templates should check if their context
// implements this interface.
type TemplateContextAccessor interface {
	ContextAccessor
	// Engine returns the template executor for nested template resolution.
	// Returns nil if no engine is available.
	Engine() interface{}
	// Depth returns the current nesting depth for template includes.
	Depth() int
}

// ErrorStrategyAccessor extends ContextAccessor with error strategy support.
// The executor uses this interface to determine how to handle resolver failures.
type ErrorStrategyAccessor interface {
	ContextAccessor
	// ErrorStrategy returns the current error handling strategy.
	// Returns an int to avoid import cycles (maps to ErrorStrategy constants).
	ErrorStrategy() int
}

// ChildContextCreator extends ContextAccessor with the ability to create child contexts.
// This is used by the executor for loop iterations to create scoped contexts.
type ChildContextCreator interface {
	ContextAccessor
	// Child creates a child context with additional data.
	// The child inherits from the parent and can override values.
	// Returns interface{} to avoid import cycle issues - caller should type assert.
	Child(data map[string]any) interface{}
}

// VarResolver handles the exons.var built-in tag.
// It retrieves variable values from the execution context.
type VarResolver struct{}

// NewVarResolver creates a new VarResolver.
func NewVarResolver() *VarResolver {
	return &VarResolver{}
}

// TagName returns the tag name for this resolver.
func (r *VarResolver) TagName() string {
	return TagNameVar
}

// Resolve retrieves the variable value from the context.
func (r *VarResolver) Resolve(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error) {
	accessor, ok := execCtx.(ContextAccessor)
	if !ok {
		return "", NewBuiltinError(ErrMsgInvalidContext, TagNameVar)
	}

	// Get required 'name' attribute
	name, ok := attrs.Get(AttrName)
	if !ok {
		return "", NewBuiltinError(ErrMsgMissingNameAttr, TagNameVar)
	}

	// Try to get the value
	val, found := accessor.Get(name)
	if !found {
		// Check for default attribute
		if defaultVal, hasDefault := attrs.Get(AttrDefault); hasDefault {
			return defaultVal, nil
		}

		// Try to provide helpful error messages with suggestions or available keys
		var suggestions []string
		var availableKeys []string
		if keyLister, ok := accessor.(KeyLister); ok {
			availableKeys = keyLister.Keys()
			pathPrefix := ExtractPathPrefix(name)
			suggestions = FindSimilarStrings(pathPrefix, availableKeys, 3)
		}
		showHint := ShouldShowHint(attrs)
		if len(suggestions) > 0 {
			return "", NewVariableNotFoundWithSuggestionsError(name, suggestions, showHint)
		}
		if len(availableKeys) > 0 {
			return "", NewVariableNotFoundWithAvailableKeysError(name, availableKeys, showHint)
		}
		return "", NewVariableNotFoundBuiltinError(name, showHint)
	}

	// Convert value to string
	return valueToString(val), nil
}

// Validate checks that the required attributes are present.
func (r *VarResolver) Validate(attrs Attributes) error {
	if !attrs.Has(AttrName) {
		return NewBuiltinError(ErrMsgMissingNameAttr, TagNameVar)
	}
	return nil
}

// valueToString converts any value to its string representation.
func valueToString(val any) string {
	switch v := val.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case nil:
		return ""
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// RawResolver handles the exons.raw built-in tag.
// This is a marker resolver - the executor handles raw blocks specially
// by preserving their content without parsing.
type RawResolver struct{}

// NewRawResolver creates a new RawResolver.
func NewRawResolver() *RawResolver {
	return &RawResolver{}
}

// TagName returns the tag name for this resolver.
func (r *RawResolver) TagName() string {
	return TagNameRaw
}

// Resolve returns an error because raw blocks should be handled by the executor.
// The raw block content is stored in TagNode.RawContent and should be
// returned directly by the executor without calling this resolver.
func (r *RawResolver) Resolve(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error) {
	// This should never be called - executor handles raw blocks specially
	return "", NewBuiltinError(ErrMsgRawResolverCalled, TagNameRaw)
}

// Validate always returns nil since raw blocks don't have required attributes.
func (r *RawResolver) Validate(attrs Attributes) error {
	return nil
}

// RegisterBuiltins registers all built-in resolvers with the registry.
func RegisterBuiltins(registry *Registry) {
	registry.MustRegister(NewVarResolver())
	registry.MustRegister(NewRawResolver())
	registry.MustRegister(NewIncludeResolver())
	registry.MustRegister(NewEnvResolver())
	registry.MustRegister(&MessageResolver{})
	registry.MustRegister(NewRefResolver())
	registry.MustRegister(NewSkillsCatalogResolver())
	registry.MustRegister(NewToolsCatalogResolver())
}

// BuiltinError represents an error from a built-in resolver.
type BuiltinError struct {
	Message  string
	TagName  string
	Metadata map[string]string
}

// NewBuiltinError creates a new builtin error.
func NewBuiltinError(message, tagName string) *BuiltinError {
	return &BuiltinError{
		Message:  message,
		TagName:  tagName,
		Metadata: make(map[string]string),
	}
}

// WithMetadata adds a metadata key-value pair and returns the error for chaining.
func (e *BuiltinError) WithMetadata(key, value string) *BuiltinError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}

// Error implements the error interface.
func (e *BuiltinError) Error() string {
	base := fmt.Sprintf(ErrFmtTagMessage, e.TagName, e.Message)
	if len(e.Metadata) > 0 {
		for k, v := range e.Metadata {
			base += fmt.Sprintf(" [%s=%s]", k, v)
		}
	}
	return base
}

// NewVariableNotFoundBuiltinError creates an error for variable not found.
// When showHint is true, an actionable hint is appended to the error message.
func NewVariableNotFoundBuiltinError(path string, showHint bool) *BuiltinError {
	message := ErrMsgVariableNotFound
	if showHint {
		message = AppendHint(message, HintVarNotFound)
	}
	return NewBuiltinError(message, TagNameVar).
		WithMetadata(MetaKeyPath, path)
}

// NewVariableNotFoundWithSuggestionsError creates an error for variable not found
// with "did you mean?" suggestions based on available keys in the context.
// When showHint is true, an actionable hint is appended to the error message.
func NewVariableNotFoundWithSuggestionsError(path string, suggestions []string, showHint bool) *BuiltinError {
	message := ErrMsgVariableNotFound
	if len(suggestions) > 0 {
		message += FormatSuggestions(suggestions)
	}
	if showHint {
		message = AppendHint(message, HintVarNotFound)
	}
	return NewBuiltinError(message, TagNameVar).
		WithMetadata(MetaKeyPath, path)
}

// NewVariableNotFoundWithAvailableKeysError creates an error for variable not found
// with a list of available keys when no similar suggestions are found.
// When showHint is true, an actionable hint is appended to the error message.
func NewVariableNotFoundWithAvailableKeysError(path string, availableKeys []string, showHint bool) *BuiltinError {
	message := ErrMsgVariableNotFound
	if len(availableKeys) > 0 {
		message += FormatAvailableKeys(availableKeys, 5)
	}
	if showHint {
		message = AppendHint(message, HintVarNotFound)
	}
	return NewBuiltinError(message, TagNameVar).
		WithMetadata(MetaKeyPath, path)
}

// NewTemplateNotFoundBuiltinError creates an error for template not found.
func NewTemplateNotFoundBuiltinError(name string) *BuiltinError {
	return NewBuiltinError(ErrMsgTemplateNotFound, TagNameInclude).
		WithMetadata(MetaKeyTemplateName, name)
}

// NewTemplateNotFoundWithHintError creates an error for template not found with an actionable hint.
func NewTemplateNotFoundWithHintError(name string) *BuiltinError {
	message := AppendHint(ErrMsgTemplateNotFound, HintTemplateNotFound)
	return NewBuiltinError(message, TagNameInclude).
		WithMetadata(MetaKeyTemplateName, name)
}

// Builtin error message constants
const (
	ErrMsgInvalidContext    = "invalid execution context type"
	ErrMsgMissingNameAttr   = "missing required 'name' attribute"
	ErrMsgVariableNotFound  = "variable not found"
	ErrMsgTemplateNotFound  = "template not found"
	ErrMsgRawResolverCalled = "raw resolver should not be called directly"
)
