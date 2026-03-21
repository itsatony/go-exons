package internal

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// ExecutorConfig holds executor configuration options.
type ExecutorConfig struct {
	MaxDepth      int // Maximum nesting depth (0 = unlimited)
	MaxOutputSize int // Maximum rendered output size in bytes (0 = unlimited)
}

// DefaultExecutorConfig returns the default executor configuration.
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		MaxDepth:      DefaultMaxDepth,
		MaxOutputSize: DefaultMaxOutputSize,
	}
}

// Executor traverses an AST and produces output by resolving tags.
type Executor struct {
	registry *Registry
	config   ExecutorConfig
	logger   *slog.Logger
	funcs    *FuncRegistry // Function registry for expression evaluation
}

// NewExecutor creates a new executor with the given registry and configuration.
func NewExecutor(registry *Registry, config ExecutorConfig, logger *slog.Logger) *Executor {
	if logger == nil {
		logger = slog.Default()
	}
	logger.Debug(LogMsgExecutorCreated)

	// Create function registry with built-in functions
	funcs := NewFuncRegistry()
	RegisterBuiltinFuncs(funcs)

	return &Executor{
		registry: registry,
		config:   config,
		logger:   logger,
		funcs:    funcs,
	}
}

// RegisterFunc registers a custom function for use in expressions.
func (e *Executor) RegisterFunc(f *Func) error {
	return e.funcs.Register(f)
}

// MustRegisterFunc registers a custom function and panics on error.
func (e *Executor) MustRegisterFunc(f *Func) {
	e.funcs.MustRegister(f)
}

// HasFunc checks if a function is registered with the given name.
func (e *Executor) HasFunc(name string) bool {
	return e.funcs.Has(name)
}

// ListFuncs returns all registered function names.
func (e *Executor) ListFuncs() []string {
	return e.funcs.List()
}

// FuncCount returns the number of registered functions.
func (e *Executor) FuncCount() int {
	return e.funcs.Count()
}

// Execute processes the AST and returns the rendered output.
func (e *Executor) Execute(ctx context.Context, root *RootNode, execCtx ContextAccessor) (string, error) {
	e.logger.Debug(LogMsgExecutorStart)

	result, err := e.executeNodes(ctx, root.Children, execCtx, 0)
	if err != nil {
		return "", err
	}

	e.logger.Debug(LogMsgExecutorEnd)
	return result, nil
}

// executeNodes processes a slice of nodes and concatenates their output.
func (e *Executor) executeNodes(ctx context.Context, nodes []Node, execCtx ContextAccessor, depth int) (string, error) {
	// Check depth limit
	if e.config.MaxDepth > 0 && depth > e.config.MaxDepth {
		return "", NewExecutorError(ErrMsgMaxDepthExceeded, "", Position{})
	}

	var sb strings.Builder

	for _, node := range nodes {
		output, err := e.executeNode(ctx, node, execCtx, depth)
		if err != nil {
			return "", err
		}
		if e.config.MaxOutputSize > 0 && sb.Len()+len(output) > e.config.MaxOutputSize {
			return "", NewExecutorError(ErrMsgOutputSizeExceeded, "", Position{})
		}
		sb.WriteString(output)
	}

	return sb.String(), nil
}

// executeNode processes a single node and returns its output.
func (e *Executor) executeNode(ctx context.Context, node Node, execCtx ContextAccessor, depth int) (string, error) {
	switch n := node.(type) {
	case *TextNode:
		return n.Content, nil

	case *TagNode:
		return e.executeTag(ctx, n, execCtx, depth)

	case *ConditionalNode:
		return e.executeConditional(ctx, n, execCtx, depth)

	case *ForNode:
		return e.executeFor(ctx, n, execCtx, depth)

	case *SwitchNode:
		return e.executeSwitch(ctx, n, execCtx, depth)

	case *BlockNode:
		return e.executeBlockNode(ctx, n, execCtx, depth)

	default:
		return "", NewExecutorError(ErrMsgUnknownNodeType, "", node.Pos())
	}
}

// executeBlockNode processes a block node (template inheritance)
func (e *Executor) executeBlockNode(ctx context.Context, block *BlockNode, execCtx ContextAccessor, depth int) (string, error) {
	// Execute the block's children - at this point inheritance should be resolved
	return e.executeNodes(ctx, block.Children, execCtx, depth+1)
}

// executeConditional processes a conditional node and returns its output.
func (e *Executor) executeConditional(ctx context.Context, cond *ConditionalNode, execCtx ContextAccessor, depth int) (string, error) {
	e.logger.Debug(LogMsgConditionEval, slog.Int(LogFieldBranches, len(cond.Branches)))

	for i, branch := range cond.Branches {
		// else branch - always execute if we reach it
		if branch.IsElse {
			e.logger.Debug(LogMsgBranchSelected, slog.Int(LogFieldBranch, i), slog.Bool(LogFieldIsElse, true))
			return e.executeNodes(ctx, branch.Children, execCtx, depth+1)
		}

		// Evaluate the condition expression
		result, err := e.evaluateCondition(ctx, branch.Condition, execCtx)
		if err != nil {
			return "", NewExecutorErrorWithCause(ErrMsgCondExprFailed, TagNameIf, branch.Pos, err)
		}

		if result {
			e.logger.Debug(LogMsgBranchSelected, slog.Int(LogFieldBranch, i), slog.String(LogFieldCondition, branch.Condition))
			return e.executeNodes(ctx, branch.Children, execCtx, depth+1)
		}
	}

	// No branch matched - return empty string
	return "", nil
}

// evaluateCondition parses and evaluates a condition expression.
// The context.Context from the executor is passed through for timeout/cancellation support.
func (e *Executor) evaluateCondition(ctx context.Context, expr string, execCtx ContextAccessor) (bool, error) {
	return EvaluateExpressionBoolWithContext(ctx, expr, e.funcs, execCtx)
}

// executeFor processes a for loop node and returns its output.
func (e *Executor) executeFor(ctx context.Context, forNode *ForNode, execCtx ContextAccessor, depth int) (string, error) {
	e.logger.Debug(LogMsgForStart,
		slog.String(LogFieldItemVar, forNode.ItemVar),
		slog.String(LogFieldIndexVar, forNode.IndexVar),
		slog.String(LogFieldCollection, forNode.Source))

	// Get the collection to iterate over
	collection, found := execCtx.Get(forNode.Source)
	if !found {
		return "", NewExecutorError(ErrMsgForCollectionPath, TagNameFor, forNode.Pos())
	}

	// Convert to slice
	items, err := toIterableSlice(collection)
	if err != nil {
		return "", NewExecutorErrorWithCause(ErrMsgForNotIterable, TagNameFor, forNode.Pos(), err)
	}

	// Determine iteration limit
	limit := forNode.Limit
	if limit <= 0 {
		limit = DefaultMaxLoopIterations
	}
	if len(items) > limit {
		e.logger.Debug(LogMsgForLimitApplied,
			slog.Int(LogFieldCollection, len(items)),
			slog.Int(AttrLimit, limit))
		items = items[:limit]
	}

	// Check if context supports child creation
	childCreator, canCreateChild := execCtx.(ChildContextCreator)
	if !canCreateChild {
		return "", NewExecutorError(ErrMsgForContextNoChild, TagNameFor, forNode.Pos())
	}

	// Iterate and execute children for each item
	var sb strings.Builder
	for i, item := range items {
		e.logger.Debug(LogMsgForIteration,
			slog.Int(LogFieldIteration, i))

		// Build child context data with loop variables
		childData := make(map[string]any)
		childData[forNode.ItemVar] = item
		if forNode.IndexVar != "" {
			childData[forNode.IndexVar] = i
		}

		// Create child context
		childCtxInterface := childCreator.Child(childData)
		childCtx, ok := childCtxInterface.(ContextAccessor)
		if !ok {
			return "", NewExecutorError(ErrMsgForContextNoChild, TagNameFor, forNode.Pos())
		}

		// Execute children with child context
		result, err := e.executeNodes(ctx, forNode.Children, childCtx, depth+1)
		if err != nil {
			return "", err
		}
		sb.WriteString(result)
	}

	e.logger.Debug(LogMsgForEnd, slog.Int(LogFieldIteration, len(items)))
	return sb.String(), nil
}

// executeSwitch processes a switch/case node and returns its output.
func (e *Executor) executeSwitch(ctx context.Context, switchNode *SwitchNode, execCtx ContextAccessor, depth int) (string, error) {
	e.logger.Debug(LogMsgSwitchEval,
		slog.String(LogFieldExpression, switchNode.Expression))

	// Evaluate the switch expression to get the value to compare against
	switchValue, err := EvaluateExpressionWithContext(ctx, switchNode.Expression, e.funcs, execCtx)
	if err != nil {
		return "", NewExecutorErrorWithCause(ErrMsgCondExprFailed, TagNameSwitch, switchNode.Pos(), err)
	}

	// Convert switch value to string for comparison
	switchValueStr := toSwitchString(switchValue)

	// Try each case in order
	for _, caseNode := range switchNode.Cases {
		e.logger.Debug(LogMsgSwitchCase,
			slog.String(LogFieldCaseValue, caseNode.Value),
			slog.String(LogFieldCaseEval, caseNode.Eval))

		matched := false

		if caseNode.Value != "" {
			// Value comparison - compare switch value string to case value
			matched = switchValueStr == caseNode.Value
		} else if caseNode.Eval != "" {
			// Boolean expression evaluation
			result, evalErr := e.evaluateCondition(ctx, caseNode.Eval, execCtx)
			if evalErr != nil {
				return "", NewExecutorErrorWithCause(ErrMsgCondExprFailed, TagNameCase, caseNode.Pos, evalErr)
			}
			matched = result
		}

		if matched {
			e.logger.Debug(LogMsgCaseMatch,
				slog.String(LogFieldCaseValue, caseNode.Value),
				slog.String(LogFieldCaseEval, caseNode.Eval))
			return e.executeNodes(ctx, caseNode.Children, execCtx, depth+1)
		}
	}

	// No case matched - check for default
	if switchNode.Default != nil {
		e.logger.Debug(LogMsgCaseDefault)
		return e.executeNodes(ctx, switchNode.Default.Children, execCtx, depth+1)
	}

	// No match and no default - return empty string
	e.logger.Debug(LogMsgSwitchNoMatch,
		slog.String(LogFieldExpression, switchNode.Expression))
	return "", nil
}

// toSwitchString converts a value to its string representation for switch comparison.
func toSwitchString(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case bool:
		if v {
			return AttrValueTrue
		}
		return AttrValueFalse
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%g", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// toIterableSlice converts a value to a slice of any for iteration.
func toIterableSlice(val any) ([]any, error) {
	if val == nil {
		return []any{}, nil
	}

	switch v := val.(type) {
	case []any:
		return v, nil
	case []string:
		result := make([]any, len(v))
		for i, s := range v {
			result[i] = s
		}
		return result, nil
	case []int:
		result := make([]any, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result, nil
	case []int64:
		result := make([]any, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result, nil
	case []float64:
		result := make([]any, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result, nil
	case []bool:
		result := make([]any, len(v))
		for i, b := range v {
			result[i] = b
		}
		return result, nil
	case []map[string]any:
		result := make([]any, len(v))
		for i, m := range v {
			result[i] = m
		}
		return result, nil
	case map[string]any:
		// Iterate over map keys (alphabetically sorted for determinism)
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sortStrings(keys)
		result := make([]any, len(keys))
		for i, k := range keys {
			// Create a key-value pair map for each entry
			result[i] = map[string]any{
				ForMapKeyField:   k,
				ForMapValueField: v[k],
			}
		}
		return result, nil
	default:
		return nil, NewTypeNotIterableError(fmt.Sprintf("%T", val))
	}
}

// sortStrings sorts a slice of strings in place (simple insertion sort for small slices).
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// executeTag processes a tag node and returns its output.
func (e *Executor) executeTag(ctx context.Context, tag *TagNode, execCtx ContextAccessor, depth int) (string, error) {
	e.logger.Debug(LogMsgResolverInvoked, slog.String(LogFieldTag, tag.Name))

	// Handle raw blocks specially
	if tag.IsRaw() {
		return tag.RawContent, nil
	}

	// Look up resolver
	resolver, ok := e.registry.Get(tag.Name)
	if !ok {
		return e.handleTagError(tag, execCtx, NewExecutorError(ErrMsgUnknownTag, tag.Name, tag.Pos()))
	}

	// Execute resolver
	result, err := resolver.Resolve(ctx, execCtx, tag.Attributes)
	if err != nil {
		return e.handleTagError(tag, execCtx, NewExecutorErrorWithCause(ErrMsgResolverFailed, tag.Name, tag.Pos(), err))
	}

	// For block tags with children, process children
	if !tag.SelfClose && len(tag.Children) > 0 {
		childResult, err := e.executeNodes(ctx, tag.Children, execCtx, depth+1)
		if err != nil {
			return "", err
		}
		// Combine resolver result with children (resolver result comes first)
		// For message tags, sanitize content and add the end marker after children
		if tag.Name == TagNameMessage {
			// Sanitize content to prevent marker injection attacks
			// Strip null bytes which are used as marker delimiters
			childResult = strings.ReplaceAll(childResult, CharNullByte, "")
			return result + childResult + MessageEndMarker, nil
		}
		return result + childResult, nil
	}

	e.logger.Debug(LogMsgResolverComplete, slog.String(LogFieldTag, tag.Name))
	return result, nil
}

// handleTagError applies the appropriate error strategy for a tag execution failure.
func (e *Executor) handleTagError(tag *TagNode, execCtx ContextAccessor, err error) (string, error) {
	// Determine the error strategy to use
	strategy := e.getErrorStrategy(tag, execCtx)

	e.logger.Debug(LogMsgErrorStrategyApplied,
		slog.String(LogFieldTag, tag.Name),
		slog.String(LogFieldStrategy, ErrorStrategy(strategy).String()),
		slog.String(LogFieldErrorMsg, err.Error()))

	switch ErrorStrategy(strategy) {
	case ErrorStrategyThrow:
		// Default behavior - propagate the error
		return "", err

	case ErrorStrategyDefault:
		// Use the default attribute value if available
		if defaultVal, hasDefault := tag.Attributes.Get(AttrDefault); hasDefault {
			return defaultVal, nil
		}
		// No default specified - return empty string
		return "", nil

	case ErrorStrategyRemove:
		// Remove the tag entirely - return empty string
		return "", nil

	case ErrorStrategyKeepRaw:
		// Keep the original tag source
		if tag.RawSource != "" {
			return tag.RawSource, nil
		}
		// Fallback to empty if no raw source captured
		return "", nil

	case ErrorStrategyLog:
		// Log the error and continue with empty string
		e.logger.Warn(LogMsgErrorLogged,
			slog.String(LogFieldTag, tag.Name),
			slog.Any("error", err))
		return "", nil

	default:
		// Unknown strategy - fall back to throw
		return "", err
	}
}

// getErrorStrategy determines which error strategy to use for a tag.
// Priority: per-tag onerror attribute > context default > throw
func (e *Executor) getErrorStrategy(tag *TagNode, execCtx ContextAccessor) int {
	// Check for per-tag onerror attribute
	if onErrorStr, hasOnError := tag.Attributes.Get(AttrOnError); hasOnError {
		return int(ParseErrorStrategy(onErrorStr))
	}

	// Check if context provides error strategy
	if stratCtx, ok := execCtx.(ErrorStrategyAccessor); ok {
		return stratCtx.ErrorStrategy()
	}

	// Default to throw
	return int(ErrorStrategyThrow)
}

// ExecutorError represents an executor error with context.
type ExecutorError struct {
	Message  string
	TagName  string
	Position Position
	Cause    error
	Metadata map[string]string
}

// NewExecutorError creates a new executor error.
func NewExecutorError(message, tagName string, pos Position) *ExecutorError {
	return &ExecutorError{
		Message:  message,
		TagName:  tagName,
		Position: pos,
		Metadata: make(map[string]string),
	}
}

// NewExecutorErrorWithCause creates a new executor error with a cause.
func NewExecutorErrorWithCause(message, tagName string, pos Position, cause error) *ExecutorError {
	return &ExecutorError{
		Message:  message,
		TagName:  tagName,
		Position: pos,
		Cause:    cause,
		Metadata: make(map[string]string),
	}
}

// WithMetadata adds a metadata key-value pair and returns the error for chaining.
func (e *ExecutorError) WithMetadata(key, value string) *ExecutorError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}

// Error implements the error interface.
func (e *ExecutorError) Error() string {
	var result string
	if e.TagName != StringValueEmpty {
		result = fmt.Sprintf(ErrFmtWithTagAndPosition, e.Message, e.TagName, e.Position.String())
	} else {
		result = fmt.Sprintf(ErrFmtWithPosition, e.Message, e.Position.String())
	}
	if e.Cause != nil {
		result = fmt.Sprintf(ErrFmtWithCause, result, e.Cause)
	}
	if len(e.Metadata) > 0 {
		for k, v := range e.Metadata {
			result += fmt.Sprintf(" [%s=%s]", k, v)
		}
	}
	return result
}

// Unwrap returns the underlying cause error.
func (e *ExecutorError) Unwrap() error {
	return e.Cause
}

// NewTypeNotIterableError creates an error for non-iterable types in for loops.
func NewTypeNotIterableError(typeName string) *ExecutorError {
	return NewExecutorError(ErrMsgTypeNotIterable, TagNameFor, Position{}).
		WithMetadata(MetaKeyIterableType, typeName)
}

// Executor error message constants
const (
	ErrMsgMaxDepthExceeded   = "maximum nesting depth exceeded"
	ErrMsgUnknownNodeType    = "unknown node type"
	ErrMsgUnknownTag         = "unknown tag"
	ErrMsgResolverFailed     = "resolver failed"
	ErrMsgTypeNotIterable    = "type is not iterable"
	ErrMsgOutputSizeExceeded = "rendered output exceeds maximum size limit"
)

// Default configuration values for internal use (tests and fallback).
// Production values are set by the root package via Engine.New() options.
const (
	DefaultMaxDepth      = 100
	DefaultMaxOutputSize = 10 * 1024 * 1024 // 10MB — matches root exons.DefaultMaxOutputSize
)
