package exons

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"sync"

	"github.com/itsatony/go-exons/internal"
)

// Engine is the main entry point for the exons templating system.
// It manages parsing, execution, resolver registration, and template storage.
//
// Thread safety: Engine is safe for concurrent access. Template registration,
// resolver configuration, and execution can all be called from multiple goroutines.
type Engine struct {
	registry     *internal.Registry
	templates    map[string]*Template // Named templates for inclusion
	tmplMu       sync.RWMutex         // Protects templates map
	config       *engineConfig
	executor     *internal.Executor
	logger       *slog.Logger
	specResolver SpecResolver         // Spec resolver for reference resolution
	specAdapter  *SpecResolverAdapter // Cached adapter (avoids per-call allocation)
	specResMu    sync.RWMutex         // Protects specResolver and specAdapter
}

// New creates a new exons Engine with the given options.
func New(opts ...Option) (*Engine, error) {
	config := defaultEngineConfig()
	for _, opt := range opts {
		opt(config)
	}

	logger := config.logger
	if logger == nil {
		logger = slog.Default()
	}

	registry := internal.NewRegistry(logger)
	internal.RegisterBuiltins(registry)

	executorConfig := internal.ExecutorConfig{
		MaxDepth: config.maxDepth,
	}
	executor := internal.NewExecutor(registry, executorConfig, logger)

	return &Engine{
		registry:  registry,
		templates: make(map[string]*Template),
		config:    config,
		executor:  executor,
		logger:    logger,
	}, nil
}

// MustNew creates a new Engine and panics if there's an error.
func MustNew(opts ...Option) *Engine {
	engine, err := New(opts...)
	if err != nil {
		panic(err)
	}
	return engine
}

// SetSpecResolver configures the spec resolver for reference resolution.
// When set, the resolver is automatically injected into execution contexts,
// enabling {~exons.ref slug="..." /~} tag functionality.
// Safe for concurrent use; however, changing the resolver while executions
// are in-flight means in-flight calls may see either the old or new resolver.
func (e *Engine) SetSpecResolver(r SpecResolver) {
	e.specResMu.Lock()
	defer e.specResMu.Unlock()
	e.specResolver = r
	if r != nil {
		e.specAdapter = NewSpecResolverAdapter(r)
	} else {
		e.specAdapter = nil
	}
}

// GetSpecResolver returns the configured spec resolver, or nil if none is set.
func (e *Engine) GetSpecResolver() SpecResolver {
	e.specResMu.RLock()
	defer e.specResMu.RUnlock()
	return e.specResolver
}

// getSpecAdapter returns the cached SpecResolverAdapter, or nil if no resolver is set.
func (e *Engine) getSpecAdapter() *SpecResolverAdapter {
	e.specResMu.RLock()
	defer e.specResMu.RUnlock()
	return e.specAdapter
}

// getSpecResolverForCatalog returns the current SpecResolver for catalog generation.
func (e *Engine) getSpecResolverForCatalog() SpecResolver {
	e.specResMu.RLock()
	defer e.specResMu.RUnlock()
	return e.specResolver
}

// Parse parses a template source string and returns a Template.
// The returned Template can be executed multiple times with different data.
//
// If the source contains YAML frontmatter (delimited by --- on separate lines),
// it is extracted and parsed as a Spec configuration. The frontmatter must appear
// at the start of the source (after optional whitespace/BOM).
func (e *Engine) Parse(source string) (*Template, error) {
	// Create lexer config
	lexerConfig := internal.LexerConfig{
		OpenDelim:  e.config.openDelim,
		CloseDelim: e.config.closeDelim,
	}

	// Extract config block if present
	configResult, err := internal.ExtractConfigBlock(source, lexerConfig)
	if err != nil {
		pos := Position{}
		if configErr, ok := err.(*internal.ConfigError); ok {
			pos = Position{
				Offset: configErr.Position.Offset,
				Line:   configErr.Position.Line,
				Column: configErr.Position.Column,
			}
		}
		return nil, NewConfigBlockError(ErrMsgConfigBlockExtract, pos, err)
	}

	// Parse frontmatter if present
	var spec *Spec

	if configResult.HasFrontmatter && configResult.FrontmatterYAML != "" {
		// Resolve environment variables in YAML before parsing
		resolvedYAML, resolveErr := e.resolveConfigEnvVars(configResult.FrontmatterYAML)
		if resolveErr != nil {
			return nil, resolveErr
		}

		// Parse as Spec
		spec, err = ParseYAMLSpec(resolvedYAML)
		if err != nil {
			return nil, err
		}

		// Validate if we have a spec with required fields
		if spec != nil {
			if validErr := spec.ValidateOptional(); validErr != nil {
				return nil, validErr
			}
		}
	}

	// Use template body (without config block) for parsing
	templateBody := configResult.TemplateBody

	// Sync spec.Body with the extracted template body so that
	// Compile/CompileAgent can use it without a deferred mutation
	// (which would be a data race on concurrent calls).
	if spec != nil && spec.Body == "" && templateBody != "" {
		spec.Body = templateBody
	}

	// Create lexer with configured delimiters
	lexer := internal.NewLexerWithConfig(templateBody, lexerConfig, e.logger)

	// Tokenize
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, NewParseError(ErrMsgParseFailed, Position{}, err)
	}

	// Parse with source for raw text extraction (keepRaw strategy)
	parser := internal.NewParserWithSource(tokens, templateBody, e.logger)
	ast, err := parser.Parse()
	if err != nil {
		return nil, NewParseError(ErrMsgParseFailed, Position{}, err)
	}

	return newTemplateWithConfig(source, templateBody, ast, e.executor, e.config, e, spec), nil
}

// resolveConfigEnvVars resolves {~exons.env~} tags in the YAML frontmatter.
// This allows environment variables to be used in frontmatter configuration.
// NOTE: When using exons tags in YAML strings, use single quotes to avoid
// escaping issues. YAML double quotes require backslash escaping (e.g., \")
// which conflicts with exons tag attribute parsing.
func (e *Engine) resolveConfigEnvVars(yamlContent string) (string, error) {
	// If there are no template tags, return as-is
	if !strings.Contains(yamlContent, e.config.openDelim) {
		return yamlContent, nil
	}

	// Execute the YAML content as a template to resolve env vars
	// We use an empty context since env vars don't need external data
	ctx := context.Background()
	result, err := e.Execute(ctx, yamlContent, nil)
	if err != nil {
		return "", NewFrontmatterError(ErrMsgFrontmatterParse, Position{}, err)
	}

	return result, nil
}

// Execute is a convenience method that parses and executes a template in one step.
//
// If a SpecResolver is configured via SetSpecResolver, it is automatically injected
// into the execution context, enabling {~exons.ref slug="..." /~} tag resolution.
//
// PERFORMANCE WARNING: This method parses the template on every call.
// For production workloads or repeated execution, use Parse() instead:
//
//	tmpl, err := engine.Parse(source)
//	if err != nil { return err }
//	result, err := tmpl.Execute(ctx, data)  // Reuse tmpl for multiple executions
//
// Parsing is typically 2-3x more expensive than execution alone.
func (e *Engine) Execute(ctx context.Context, source string, data map[string]any) (string, error) {
	tmpl, err := e.Parse(source)
	if err != nil {
		return "", err
	}
	return e.executeTemplate(ctx, tmpl, data)
}

// executeTemplate executes a template with automatic SpecResolver injection.
func (e *Engine) executeTemplate(ctx context.Context, tmpl *Template, data map[string]any) (string, error) {
	adapter := e.getSpecAdapter()
	if adapter == nil {
		return tmpl.Execute(ctx, data)
	}
	// Create context with spec resolver injected
	execCtx := NewContextWithStrategy(data, e.config.errorStrategy)
	execCtx = execCtx.WithSpecResolver(adapter)
	return tmpl.ExecuteWithContext(ctx, execCtx)
}

// ExecuteWithCatalogs executes a template with auto-generated skill and tool catalogs
// injected into the context data. This makes {~exons.var name="skills" /~} and
// {~exons.var name="tools" /~} available without manual context plumbing.
//
// The spec is used to generate catalog strings: skills from spec.Skills via the
// configured SpecResolver, tools from spec.Tools. The generated catalogs are
// injected into a copy of the data map under the "skills" and "tools" keys.
// The caller's original data map is NOT modified.
//
// If format is empty (CatalogFormatDefault), the default markdown format is used.
func (e *Engine) ExecuteWithCatalogs(ctx context.Context, source string, data map[string]any, spec *Spec, format CatalogFormat) (string, error) {
	// Create a shallow copy of data to avoid mutating the caller's map
	dataCopy := make(map[string]any, len(data)+2)
	for k, v := range data {
		dataCopy[k] = v
	}

	// Generate skills catalog if spec has skills
	if spec != nil && len(spec.Skills) > 0 {
		resolver := e.getSpecResolverForCatalog()
		skillsCatalog, err := GenerateSkillsCatalog(ctx, spec.Skills, resolver, format)
		if err != nil {
			return "", err
		}
		if skillsCatalog != "" {
			dataCopy[ContextKeySkills] = skillsCatalog
		}
	}

	// Generate tools catalog if spec has tools
	if spec != nil && spec.Tools != nil && spec.Tools.HasTools() {
		toolsCatalog, err := GenerateToolsCatalog(spec.Tools, format)
		if err != nil {
			return "", err
		}
		if toolsCatalog != "" {
			dataCopy[ContextKeyTools] = toolsCatalog
		}
	}

	// Execute with SpecResolver injection
	tmpl, err := e.Parse(source)
	if err != nil {
		return "", err
	}
	return e.executeTemplate(ctx, tmpl, dataCopy)
}

// Register adds a custom resolver to the engine.
// Returns an error if a resolver for the same tag name is already registered.
func (e *Engine) Register(r Resolver) error {
	adapter := &resolverAdapter{resolver: r}
	return e.registry.Register(adapter)
}

// RegisterResolver adds a custom resolver to the engine.
// This is an alias for Register that satisfies the TemplateRunner interface.
func (e *Engine) RegisterResolver(r Resolver) error {
	return e.Register(r)
}

// MustRegister adds a custom resolver and panics if registration fails.
func (e *Engine) MustRegister(r Resolver) {
	if err := e.Register(r); err != nil {
		panic(err)
	}
}

// HasResolver checks if a resolver is registered for the given tag name.
func (e *Engine) HasResolver(tagName string) bool {
	return e.registry.Has(tagName)
}

// ListResolvers returns all registered resolver tag names in sorted order.
func (e *Engine) ListResolvers() []string {
	return e.registry.List()
}

// ResolverCount returns the number of registered resolvers.
func (e *Engine) ResolverCount() int {
	return e.registry.Count()
}

// RegisterTemplate registers a named template for later inclusion via exons.include.
// Template names cannot be empty or use the reserved "exons." namespace prefix.
// Returns an error if a template with the same name already exists.
func (e *Engine) RegisterTemplate(name string, source string) error {
	// Validate template name
	if name == "" {
		return NewEmptyTemplateNameError()
	}
	if strings.HasPrefix(name, TagNamespacePrefix) {
		return NewReservedTemplateNameError(name)
	}

	// Check for existing template
	e.tmplMu.Lock()
	defer e.tmplMu.Unlock()

	if _, exists := e.templates[name]; exists {
		return NewTemplateExistsError(name)
	}

	// Parse the template
	tmpl, err := e.Parse(source)
	if err != nil {
		return err
	}

	e.templates[name] = tmpl
	return nil
}

// MustRegisterTemplate registers a template and panics on error.
func (e *Engine) MustRegisterTemplate(name string, source string) {
	if err := e.RegisterTemplate(name, source); err != nil {
		panic(err)
	}
}

// UnregisterTemplate removes a registered template by name.
// Returns true if the template existed and was removed, false otherwise.
func (e *Engine) UnregisterTemplate(name string) bool {
	e.tmplMu.Lock()
	defer e.tmplMu.Unlock()

	if _, exists := e.templates[name]; exists {
		delete(e.templates, name)
		return true
	}
	return false
}

// GetTemplate retrieves a registered template by name.
// Returns the template and true if found, or nil and false if not.
func (e *Engine) GetTemplate(name string) (*Template, bool) {
	e.tmplMu.RLock()
	defer e.tmplMu.RUnlock()

	tmpl, ok := e.templates[name]
	return tmpl, ok
}

// GetTemplateSource retrieves the source string of a registered template by name.
// This implements TemplateSourceResolver for template inheritance support.
// Returns the source and true if found, or empty string and false if not.
func (e *Engine) GetTemplateSource(name string) (string, bool) {
	e.tmplMu.RLock()
	defer e.tmplMu.RUnlock()

	tmpl, ok := e.templates[name]
	if !ok {
		return "", false
	}
	return tmpl.Source(), true
}

// HasTemplate checks if a template is registered with the given name.
func (e *Engine) HasTemplate(name string) bool {
	e.tmplMu.RLock()
	defer e.tmplMu.RUnlock()

	_, ok := e.templates[name]
	return ok
}

// ListTemplates returns all registered template names in sorted order.
func (e *Engine) ListTemplates() []string {
	e.tmplMu.RLock()
	defer e.tmplMu.RUnlock()

	names := make([]string, 0, len(e.templates))
	for name := range e.templates {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// TemplateCount returns the number of registered templates.
func (e *Engine) TemplateCount() int {
	e.tmplMu.RLock()
	defer e.tmplMu.RUnlock()

	return len(e.templates)
}

// ExecuteTemplate executes a registered template by name with the given data.
// This implements the TemplateExecutor interface for nested template support.
// It handles depth tracking for nested template inclusion.
// Note: This method creates a copy of the data map to avoid mutating caller's data.
func (e *Engine) ExecuteTemplate(ctx context.Context, name string, data map[string]any) (string, error) {
	tmpl, ok := e.GetTemplate(name)
	if !ok {
		return "", NewTemplateNotFoundError(name)
	}

	// Extract parent depth if provided and create clean data copy
	parentDepth := 0
	var cleanData map[string]any
	if data != nil {
		// Extract depth before copying
		if pd, ok := data[MetaKeyParentDepth]; ok {
			if depth, ok := pd.(int); ok {
				parentDepth = depth
			}
		}
		// Create a copy without the meta key to avoid mutating caller's data
		cleanData = make(map[string]any, len(data))
		for k, v := range data {
			if k != MetaKeyParentDepth {
				cleanData[k] = v
			}
		}
	}

	// Create context with incremented depth
	execCtx := NewContextWithStrategy(cleanData, e.config.errorStrategy)
	execCtx = execCtx.WithEngine(e).WithDepth(parentDepth + 1)

	// Inject spec resolver if configured
	adapter := e.getSpecAdapter()
	if adapter != nil {
		execCtx = execCtx.WithSpecResolver(adapter)
	}

	return tmpl.ExecuteWithContext(ctx, execCtx)
}

// MaxDepth returns the configured maximum nesting depth.
// This implements the TemplateExecutor interface for nested template support.
func (e *Engine) MaxDepth() int {
	return e.config.maxDepth
}

// resolverAdapter adapts the public Resolver interface to internal.InternalResolver.
type resolverAdapter struct {
	resolver Resolver
}

func (a *resolverAdapter) TagName() string {
	return a.resolver.TagName()
}

func (a *resolverAdapter) Resolve(ctx context.Context, execCtx interface{}, attrs internal.Attributes) (string, error) {
	// Convert execCtx to *Context
	exonsCtx, ok := execCtx.(*Context)
	if !ok {
		return "", NewExecutionError(ErrMsgInvalidContextType, a.TagName(), Position{}, nil)
	}

	// Wrap internal.Attributes to satisfy public Attributes interface
	wrappedAttrs := &internalAttributesAdapter{attrs: attrs}
	return a.resolver.Resolve(ctx, exonsCtx, wrappedAttrs)
}

func (a *resolverAdapter) Validate(attrs internal.Attributes) error {
	wrappedAttrs := &internalAttributesAdapter{attrs: attrs}
	return a.resolver.Validate(wrappedAttrs)
}
