package exons

import (
	"context"
	"fmt"
	"strings"

	"github.com/itsatony/go-exons/execution"
)

// CompiledMessage represents a message in a compiled agent output.
// Unlike Message (which is for template output extraction), CompiledMessage
// is the result of agent compilation — fully resolved and ready for LLM APIs.
type CompiledMessage struct {
	// Role is the message role: "system", "user", "assistant", or "tool".
	Role string
	// Content is the fully resolved message content.
	Content string
	// Cache indicates whether caching is hinted for this message.
	Cache bool
}

// CompiledSpec is the result of compiling an agent spec.
// It contains all information needed to make an LLM API call.
type CompiledSpec struct {
	// Messages is the compiled message array ready for LLM APIs.
	Messages []CompiledMessage
	// Execution is the merged execution config for the LLM call.
	Execution *execution.Config
	// Tools is the cloned tools config for function calling.
	Tools *ToolsConfig
	// Constraints is the cloned constraints config.
	Constraints *ConstraintsConfig
}

// CompileOption is a functional option for configuring compilation.
type CompileOption func(*CompileOptions)

// CompileOptions holds configuration for agent compilation.
type CompileOptions struct {
	// Resolver provides spec lookup for skill resolution during compilation.
	Resolver SpecResolver
	// Engine is an optional pre-configured engine for template execution.
	// If nil, a fresh engine is created via MustNew().
	Engine *Engine
	// SkillsCatalogFormat controls the format for skills catalog generation.
	SkillsCatalogFormat CatalogFormat
	// ToolsCatalogFormat controls the format for tools catalog generation.
	ToolsCatalogFormat CatalogFormat
}

// NewCompileOptions creates a CompileOptions from functional options.
func NewCompileOptions(opts ...CompileOption) *CompileOptions {
	o := &CompileOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// WithResolver sets the SpecResolver for skill resolution during compilation.
func WithResolver(r SpecResolver) CompileOption {
	return func(o *CompileOptions) {
		o.Resolver = r
	}
}

// WithCompileEngine sets a pre-configured engine for template execution.
func WithCompileEngine(e *Engine) CompileOption {
	return func(o *CompileOptions) {
		o.Engine = e
	}
}

// WithSkillsCatalogFormat sets the format for skills catalog generation.
func WithSkillsCatalogFormat(f CatalogFormat) CompileOption {
	return func(o *CompileOptions) {
		o.SkillsCatalogFormat = f
	}
}

// WithToolsCatalogFormat sets the format for tools catalog generation.
func WithToolsCatalogFormat(f CatalogFormat) CompileOption {
	return func(o *CompileOptions) {
		o.ToolsCatalogFormat = f
	}
}

// compileEngine returns the engine from options, or creates a fresh one.
func compileEngine(opts *CompileOptions) *Engine {
	if opts != nil && opts.Engine != nil {
		return opts.Engine
	}
	return MustNew()
}

// buildCompileContext constructs the template context map for agent compilation.
// It includes input, meta, context, constraints, and catalog strings.
func buildCompileContext(ctx context.Context, s *Spec, input map[string]any, opts *CompileOptions) map[string]any {
	data := make(map[string]any)

	// Flatten input at top level and under "input" key
	if input != nil {
		data[ContextKeyInput] = input
		for k, v := range input {
			data[k] = v
		}
	}

	// Meta information
	meta := map[string]any{
		MetaKeyDocumentType: string(s.EffectiveType()),
		MetaKeySpecName:     s.Name,
	}
	data[ContextKeyMeta] = meta

	// Context from spec
	if s.Context != nil {
		data[ContextKeyContext] = s.Context
		for k, v := range s.Context {
			if _, exists := data[k]; !exists {
				data[k] = v
			}
		}
	}

	// Constraints as context
	if s.Constraints != nil {
		constraintsMap := map[string]any{}
		if len(s.Constraints.Behavioral) > 0 {
			constraintsMap[ConstraintsKeyBehavioral] = s.Constraints.Behavioral
		}
		if len(s.Constraints.Safety) > 0 {
			constraintsMap[ConstraintsKeySafety] = s.Constraints.Safety
		}
		data[ContextKeyConstraints] = constraintsMap
	}

	// Skills catalog (non-fatal on error)
	var resolver SpecResolver
	var skillsFmt, toolsFmt CatalogFormat
	if opts != nil {
		resolver = opts.Resolver
		skillsFmt = opts.SkillsCatalogFormat
		toolsFmt = opts.ToolsCatalogFormat
	}
	if len(s.Skills) > 0 {
		catalog, err := GenerateSkillsCatalog(ctx, s.Skills, resolver, skillsFmt)
		if err == nil && catalog != "" {
			data[ContextKeySkills] = catalog
		}
	}

	// Tools catalog (non-fatal on error)
	if s.Tools != nil && s.Tools.HasTools() {
		catalog, err := GenerateToolsCatalog(s.Tools, toolsFmt)
		if err == nil && catalog != "" {
			data[ContextKeyTools] = catalog
		}
	}

	return data
}

// compileMessages executes each MessageTemplate through the engine and returns CompiledMessages.
func compileMessages(ctx context.Context, engine *Engine, templates []MessageTemplate, data map[string]any, compiledBody string) ([]CompiledMessage, error) {
	messages := make([]CompiledMessage, 0, len(templates))
	for i, mt := range templates {
		content := mt.Content

		// If message content contains template tags, execute through engine
		if strings.Contains(content, DefaultOpenDelim) {
			// Inject compiled body as _selfBody for self-reference
			msgData := make(map[string]any, len(data)+1)
			for k, v := range data {
				msgData[k] = v
			}
			if compiledBody != "" {
				msgData[ContextKeySelfBody] = compiledBody
			}

			result, err := engine.Execute(ctx, content, msgData)
			if err != nil {
				return nil, NewCompileMessageError(i, mt.Role, err)
			}
			content = result
		}

		content = strings.TrimSpace(content)
		if content == "" {
			continue
		}

		messages = append(messages, CompiledMessage{
			Role:    mt.Role,
			Content: content,
			Cache:   mt.Cache,
		})
	}
	return messages, nil
}

// buildDefaultMessages creates default messages when no messages are defined in the spec.
// If a body is present, it becomes a system message.
func buildDefaultMessages(compiledBody string, input map[string]any) []CompiledMessage {
	var messages []CompiledMessage

	if compiledBody != "" {
		messages = append(messages, CompiledMessage{
			Role:    RoleSystem,
			Content: compiledBody,
		})
	}

	// If input has a string "query" or "message", add as user message
	if input != nil {
		for _, key := range []string{DefaultInputKeyQuery, DefaultInputKeyMessage} {
			if val, ok := input[key]; ok {
				if s, ok := val.(string); ok && s != "" {
					messages = append(messages, CompiledMessage{
						Role:    RoleUser,
						Content: s,
					})
					break
				}
			}
		}
	}

	return messages
}

// injectSkillIntoSystemPrompt appends skill content to the first system message,
// wrapped in injection markers. Creates a system message if none exists.
func injectSkillIntoSystemPrompt(compiled *CompiledSpec, slug string, content string) {
	marker := fmt.Sprintf("\n\n%s%s%s\n%s\n%s%s%s",
		SkillInjectionMarkerStart, slug, SkillInjectionMarkerClose,
		content,
		SkillInjectionMarkerEnd, slug, SkillInjectionMarkerClose,
	)

	// Find existing system message
	for i, msg := range compiled.Messages {
		if msg.Role == RoleSystem {
			compiled.Messages[i].Content += marker
			return
		}
	}

	// No system message found — create one
	compiled.Messages = append([]CompiledMessage{{
		Role:    RoleSystem,
		Content: strings.TrimPrefix(marker, "\n\n"),
	}}, compiled.Messages...)
}

// injectSkillIntoUserContext appends skill content to the last user message,
// wrapped in injection markers. Creates a user message if none exists.
func injectSkillIntoUserContext(compiled *CompiledSpec, slug string, content string) {
	marker := fmt.Sprintf("\n\n%s%s%s\n%s\n%s%s%s",
		SkillInjectionMarkerStart, slug, SkillInjectionMarkerClose,
		content,
		SkillInjectionMarkerEnd, slug, SkillInjectionMarkerClose,
	)

	// Find last user message
	for i := len(compiled.Messages) - 1; i >= 0; i-- {
		if compiled.Messages[i].Role == RoleUser {
			compiled.Messages[i].Content += marker
			return
		}
	}

	// No user message found — create one
	compiled.Messages = append(compiled.Messages, CompiledMessage{
		Role:    RoleUser,
		Content: strings.TrimPrefix(marker, "\n\n"),
	})
}

// Compile compiles the spec by executing its body through an engine.
// Returns the compiled body string.
func (s *Spec) Compile(ctx context.Context, input map[string]any, opts *CompileOptions) (string, error) {
	if s == nil {
		return "", NewCompilationError(ErrMsgCompileNotAgent, nil)
	}

	if s.Body == "" {
		return "", nil
	}

	engine := compileEngine(opts)

	// Build context
	data := buildCompileContext(ctx, s, input, opts)

	// Execute body template
	result, err := engine.Execute(ctx, s.Body, data)
	if err != nil {
		return "", NewCompileBodyError(err)
	}

	return strings.TrimSpace(result), nil
}

// CompileAgent compiles the agent spec into a CompiledSpec ready for LLM API calls.
// The spec must be of agent type with execution config and either body or messages.
//
// Compilation steps:
//  1. Validate the spec is a valid agent
//  2. Build context with input, meta, catalogs
//  3. Compile body template (if present)
//  4. Compile messages or build default messages from body
//  5. Clone execution config, tools, and constraints
func (s *Spec) CompileAgent(ctx context.Context, input map[string]any, opts *CompileOptions) (*CompiledSpec, error) {
	if s == nil {
		return nil, NewCompilationError(ErrMsgCompileNotAgent, nil)
	}

	// Validate agent
	if err := s.ValidateAsAgent(); err != nil {
		return nil, err
	}

	engine := compileEngine(opts)

	// Set up resolver if provided
	if opts != nil && opts.Resolver != nil {
		engine.SetSpecResolver(opts.Resolver)
	}

	// Build context data
	data := buildCompileContext(ctx, s, input, opts)

	// Compile body
	var compiledBody string
	if s.Body != "" {
		// Register body as "self" template to allow self-reference
		if engine.HasTemplate(TemplateNameSelf) {
			engine.UnregisterTemplate(TemplateNameSelf)
		}
		if err := engine.RegisterTemplate(TemplateNameSelf, s.Body); err != nil {
			return nil, NewCompileBodyError(err)
		}

		result, err := engine.Execute(ctx, s.Body, data)
		if err != nil {
			return nil, NewCompileBodyError(err)
		}
		compiledBody = strings.TrimSpace(result)
	}

	// Build compiled spec
	compiled := &CompiledSpec{
		Execution:   s.Execution.Clone(),
		Tools:       s.Tools.Clone(),
		Constraints: s.Constraints.Clone(),
	}

	// Compile messages
	if len(s.Messages) > 0 {
		msgs, err := compileMessages(ctx, engine, s.Messages, data, compiledBody)
		if err != nil {
			return nil, err
		}
		compiled.Messages = msgs
	} else {
		compiled.Messages = buildDefaultMessages(compiledBody, input)
	}

	return compiled, nil
}

// ActivateSkill compiles an agent with a specific skill activated.
// The skill content is resolved via the SpecResolver and injected into the
// appropriate message based on the skill's injection mode.
//
// Injection modes:
//   - "system_prompt" (default): appends to system message
//   - "user_context": appends to user message
//   - "none": no injection (skill is still compiled)
func (s *Spec) ActivateSkill(ctx context.Context, skillSlug string, input map[string]any, opts *CompileOptions) (*CompiledSpec, error) {
	if s == nil {
		return nil, NewCompilationError(ErrMsgCompileNotAgent, nil)
	}

	// Find the skill reference
	var skillRef *SkillRef
	for i := range s.Skills {
		if s.Skills[i].Slug == skillSlug {
			skillRef = &s.Skills[i]
			break
		}
	}
	if skillRef == nil {
		return nil, NewSkillNotFoundError(skillSlug)
	}

	// First compile the agent normally
	compiled, err := s.CompileAgent(ctx, input, opts)
	if err != nil {
		return nil, err
	}

	// Resolve the skill body
	if opts == nil || opts.Resolver == nil {
		return compiled, nil
	}

	resolvedSpec, resolvedBody, err := opts.Resolver.ResolveSpec(ctx, skillSlug, RefVersionLatest)
	if err != nil {
		return nil, NewCompileSkillError(skillSlug, err)
	}

	// Execute the skill body through engine — ensure resolver is set
	// so {~exons.ref~} tags in the skill body can resolve
	engine := compileEngine(opts)
	if opts != nil && opts.Resolver != nil {
		engine.SetSpecResolver(opts.Resolver)
	}
	skillData := buildCompileContext(ctx, s, input, opts)
	if resolvedSpec != nil && resolvedSpec.Context != nil {
		for k, v := range resolvedSpec.Context {
			skillData[k] = v
		}
	}

	var skillContent string
	if resolvedBody != "" {
		if strings.Contains(resolvedBody, DefaultOpenDelim) {
			skillContent, err = engine.Execute(ctx, resolvedBody, skillData)
			if err != nil {
				return nil, NewCompileSkillError(skillSlug, err)
			}
			skillContent = strings.TrimSpace(skillContent)
		} else {
			skillContent = strings.TrimSpace(resolvedBody)
		}
	}

	if skillContent == "" {
		return compiled, nil
	}

	// Inject based on injection mode
	injection := skillRef.Injection
	if injection == "" {
		injection = string(SkillInjectionSystemPrompt)
	}

	switch SkillInjection(injection) {
	case SkillInjectionSystemPrompt:
		injectSkillIntoSystemPrompt(compiled, skillSlug, skillContent)
	case SkillInjectionUserContext:
		injectSkillIntoUserContext(compiled, skillSlug, skillContent)
	case SkillInjectionNone:
		// No injection
	}

	// Merge skill execution config (2-layer: agent + resolved skill)
	if resolvedSpec != nil && resolvedSpec.Execution != nil && compiled.Execution != nil {
		compiled.Execution = compiled.Execution.Merge(resolvedSpec.Execution)
	}

	return compiled, nil
}
