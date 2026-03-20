package exons

import (
	"context"

	"github.com/itsatony/go-exons/internal"
)

// Template represents a parsed template that can be executed multiple times.
type Template struct {
	source          string
	templateBody    string // Template body without config block
	ast             *internal.RootNode
	executor        *internal.Executor
	config          *engineConfig
	engine          TemplateExecutor          // Engine reference for nested template execution
	spec            *Spec                     // Parsed spec configuration from frontmatter
	inheritanceInfo *internal.InheritanceInfo // Inheritance info (nil if no extends)
}

// newTemplateWithConfig creates a new template with spec configuration (internal use).
func newTemplateWithConfig(source, templateBody string, ast *internal.RootNode, executor *internal.Executor, config *engineConfig, engine TemplateExecutor, spec *Spec) *Template {
	// Extract inheritance info from AST (non-fatal — nil on error preserves fail-safe behavior)
	inheritanceInfo, err := internal.ExtractInheritanceInfo(ast)
	if err != nil {
		inheritanceInfo = nil
	}

	return &Template{
		source:          source,
		templateBody:    templateBody,
		ast:             ast,
		executor:        executor,
		config:          config,
		engine:          engine,
		spec:            spec,
		inheritanceInfo: inheritanceInfo,
	}
}

// Execute renders the template with the given data.
// This is a convenience method that creates a Context from the data map.
func (t *Template) Execute(ctx context.Context, data map[string]any) (string, error) {
	execCtx := NewContextWithStrategy(data, t.config.errorStrategy)
	return t.ExecuteWithContext(ctx, execCtx)
}

// ExecuteWithContext renders the template with the given execution context.
// Use this when you need more control over the context (e.g., parent scoping).
// The engine reference is injected into the context for nested template support.
// If the template uses extends (template inheritance), inheritance is resolved before execution.
func (t *Template) ExecuteWithContext(ctx context.Context, execCtx *Context) (string, error) {
	// Inject engine reference into context for nested template resolution
	if t.engine != nil && execCtx.Engine() == nil {
		execCtx = execCtx.WithEngine(t.engine)
	}

	// Resolve inheritance if the template extends another template
	astToExecute := t.ast
	if t.inheritanceInfo != nil && t.engine != nil {
		// Create an adapter that wraps the engine for TemplateSourceResolver interface
		sourceResolver := &engineSourceAdapter{engine: t.engine}
		resolver := internal.NewInheritanceResolver(nil, sourceResolver, t.config.maxDepth)
		resolvedAST, err := resolver.ResolveInheritance(ctx, t.ast, t.inheritanceInfo, 0)
		if err != nil {
			return "", err
		}
		astToExecute = resolvedAST
	}

	return t.executor.Execute(ctx, astToExecute, execCtx)
}

// engineSourceAdapter adapts TemplateExecutor to internal.TemplateSourceResolver.
type engineSourceAdapter struct {
	engine TemplateExecutor
}

func (a *engineSourceAdapter) GetTemplateSource(name string) (string, bool) {
	return a.engine.GetTemplateSource(name)
}

// Source returns the original template source string (including config block if present).
func (t *Template) Source() string {
	return t.source
}

// TemplateBody returns the template body without the config block.
// This is the portion of the template that is actually executed.
func (t *Template) TemplateBody() string {
	return t.templateBody
}

// Spec returns the spec configuration from the frontmatter.
// Returns nil if the template has no frontmatter.
func (t *Template) Spec() *Spec {
	return t.spec
}

// HasSpec returns true if the template has a spec configuration.
func (t *Template) HasSpec() bool {
	return t.spec != nil
}

// ExecuteAndExtractMessages executes the template and extracts structured messages from the output.
// This is useful for chat/conversation templates that use {~exons.message~} tags.
// Returns the messages array and any error from execution.
func (t *Template) ExecuteAndExtractMessages(ctx context.Context, data map[string]any) ([]Message, error) {
	output, err := t.Execute(ctx, data)
	if err != nil {
		return nil, err
	}
	return ExtractMessagesFromOutput(output), nil
}

// Message represents a structured message extracted from template output.
// Messages are produced by the exons.message tag resolver.
type Message struct {
	// Role is the message role: "system", "user", "assistant", or "tool".
	Role string
	// Content is the message content with leading/trailing whitespace trimmed.
	Content string
	// Cache indicates whether caching is hinted for this message.
	Cache bool
}

// ExtractMessagesFromOutput parses executed template output and extracts structured messages.
// Messages are marked by special markers inserted by the exons.message tag resolver.
// This is a standalone function for when you already have the executed output.
func ExtractMessagesFromOutput(output string) []Message {
	internalMessages := internal.ExtractMessages(output)
	if internalMessages == nil {
		return nil
	}

	messages := make([]Message, len(internalMessages))
	for i, m := range internalMessages {
		messages[i] = Message{
			Role:    m.Role,
			Content: m.Content,
			Cache:   m.Cache,
		}
	}
	return messages
}

// internalAttributesAdapter wraps internal.Attributes to implement the public Attributes interface.
type internalAttributesAdapter struct {
	attrs internal.Attributes
}

func (a *internalAttributesAdapter) Get(key string) (string, bool) {
	return a.attrs.Get(key)
}

func (a *internalAttributesAdapter) GetDefault(key, defaultVal string) string {
	return a.attrs.GetDefault(key, defaultVal)
}

func (a *internalAttributesAdapter) Has(key string) bool {
	return a.attrs.Has(key)
}

func (a *internalAttributesAdapter) Keys() []string {
	return a.attrs.Keys()
}

func (a *internalAttributesAdapter) Map() map[string]string {
	return a.attrs.Map()
}

// Position.String() is defined in exons.errors.go.
