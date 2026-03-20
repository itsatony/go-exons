package exons

import (
	"context"
)

// AgentExecutor provides a high-level API for agent compilation.
// It combines parsing, validation, and compilation into single method calls,
// holding configuration (resolver, engine, catalog formats) so callers don't
// need to construct CompileOptions on every call.
//
// Thread safety: AgentExecutor itself is not safe for concurrent mutation of
// its fields. However, multiple goroutines may call Execute/ExecuteFile/ExecuteSpec
// concurrently as long as the underlying resolver and engine are thread-safe.
type AgentExecutor struct {
	resolver            SpecResolver
	engine              *Engine
	skillsCatalogFormat CatalogFormat
	toolsCatalogFormat  CatalogFormat
}

// AgentExecutorOption is a functional option for configuring AgentExecutor.
type AgentExecutorOption func(*AgentExecutor)

// WithAgentResolver sets the SpecResolver for skill resolution.
func WithAgentResolver(r SpecResolver) AgentExecutorOption {
	return func(ae *AgentExecutor) {
		ae.resolver = r
	}
}

// WithAgentEngine sets a pre-configured engine for template execution.
// If not set, a fresh engine is created on each compilation call.
func WithAgentEngine(e *Engine) AgentExecutorOption {
	return func(ae *AgentExecutor) {
		ae.engine = e
	}
}

// WithAgentSkillsCatalogFormat sets the format for skills catalog generation.
func WithAgentSkillsCatalogFormat(f CatalogFormat) AgentExecutorOption {
	return func(ae *AgentExecutor) {
		ae.skillsCatalogFormat = f
	}
}

// WithAgentToolsCatalogFormat sets the format for tools catalog generation.
func WithAgentToolsCatalogFormat(f CatalogFormat) AgentExecutorOption {
	return func(ae *AgentExecutor) {
		ae.toolsCatalogFormat = f
	}
}

// NewAgentExecutor creates a new AgentExecutor with the given options.
func NewAgentExecutor(opts ...AgentExecutorOption) *AgentExecutor {
	ae := &AgentExecutor{}
	for _, opt := range opts {
		opt(ae)
	}
	return ae
}

// Execute parses a source string and compiles it as an agent.
// Returns error if parsing or compilation fails.
func (ae *AgentExecutor) Execute(ctx context.Context, source string, input map[string]any) (*CompiledSpec, error) {
	spec, err := Parse([]byte(source))
	if err != nil {
		return nil, NewCompilationError(ErrMsgAgentExecParseFailed, err)
	}
	return spec.CompileAgent(ctx, input, ae.compileOptions())
}

// ExecuteFile reads a file and compiles it as an agent.
// Uses ParseFile from exons.parse.go to read and parse the file.
func (ae *AgentExecutor) ExecuteFile(ctx context.Context, path string, input map[string]any) (*CompiledSpec, error) {
	spec, err := ParseFile(path)
	if err != nil {
		return nil, NewCompilationError(ErrMsgAgentExecReadFile, err)
	}
	return spec.CompileAgent(ctx, input, ae.compileOptions())
}

// ExecuteSpec compiles an existing Spec as an agent.
// Returns an error if the spec is nil.
func (ae *AgentExecutor) ExecuteSpec(ctx context.Context, spec *Spec, input map[string]any) (*CompiledSpec, error) {
	if spec == nil {
		return nil, NewCompilationError(ErrMsgAgentExecNilSpec, nil)
	}
	return spec.CompileAgent(ctx, input, ae.compileOptions())
}

// ActivateSkill parses a source string and activates a specific skill.
// The skill is resolved via the configured SpecResolver and injected based
// on the skill's injection mode.
func (ae *AgentExecutor) ActivateSkill(ctx context.Context, source string, skillSlug string, input map[string]any) (*CompiledSpec, error) {
	spec, err := Parse([]byte(source))
	if err != nil {
		return nil, NewCompilationError(ErrMsgAgentExecParseFailed, err)
	}
	return spec.ActivateSkill(ctx, skillSlug, input, ae.compileOptions())
}

// compileOptions builds CompileOptions from executor configuration.
func (ae *AgentExecutor) compileOptions() *CompileOptions {
	return &CompileOptions{
		Resolver:            ae.resolver,
		Engine:              ae.engine,
		SkillsCatalogFormat: ae.skillsCatalogFormat,
		ToolsCatalogFormat:  ae.toolsCatalogFormat,
	}
}
