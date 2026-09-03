package exons

import (
	"context"
	"strings"

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

	// inheritanceErr records why inheritanceInfo is nil when extraction FAILED, as opposed to
	// the ordinary case of a template that simply does not extend anything. The two are
	// indistinguishable from inheritanceInfo alone, and DryRun must distinguish them: "no
	// parent" and "a parent this build could not read" produce the same reference collections
	// but only one of them is complete. See dryRunAST.
	inheritanceErr error
}

// newTemplateWithConfig creates a new template with spec configuration (internal use).
func newTemplateWithConfig(source, templateBody string, ast *internal.RootNode, executor *internal.Executor, config *engineConfig, engine TemplateExecutor, spec *Spec) *Template {
	// Extraction failure is non-fatal for RENDERING — a template that cannot state its parent can
	// still render its own body, and nil preserves that fail-safe. It is not non-fatal for
	// ANALYSIS, so the reason is kept rather than discarded; DryRun reports it.
	inheritanceInfo, inheritanceErr := internal.ExtractInheritanceInfo(ast)
	if inheritanceErr != nil {
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
		inheritanceErr:  inheritanceErr,
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
	// A per-call parent resolver is asked each question ONCE for this render, however many walks
	// ask it — see memoizedTemplateSource. Done before anything reads it.
	if execCtx != nil {
		if r := execCtx.TemplateSourceResolver(); r != nil {
			execCtx = execCtx.WithTemplateSourceResolver(memoizeTemplateSource(r))
		}
	}

	// Declared inputs become readable BEFORE anything else runs — see contextWithInputs. The
	// execution context goes in because it may carry the per-call parent resolver, and an
	// inherited input's default must come from the SAME parent the render is about to splice in.
	execCtx = t.contextWithInputs(ctx, execCtx)

	// Inject engine reference into context for nested template resolution
	if t.engine != nil && execCtx.Engine() == nil {
		execCtx = execCtx.WithEngine(t.engine)
	}

	// Resolve inheritance if the template extends another template.
	//
	// Every non-nil outcome error is returned. Two of the three used to be swallowed here — an
	// unreadable extends declaration rendered the bare child, and an engine-less extends rendered
	// the bare child too — both reported as SUCCESS. See resolveInheritance.
	astToExecute, _, err := t.resolveInheritance(ctx, execCtx)
	if err != nil {
		return "", err
	}

	return t.executor.Execute(ctx, astToExecute, execCtx)
}

// inheritanceOutcome names WHY resolveInheritance returned the AST it did.
//
// The three callers — ExecuteWithContext, dryRunAST and Explain — need one resolution but report
// it in three different vocabularies: an error return, an analysis-completeness channel, and a
// result field. So the helper decides what happened and each caller decides how to say it. Before
// this release each caller decided both, and they disagreed in three separate ways.
type inheritanceOutcome int

const (
	// inheritanceNone — the template extends nothing. Its own AST is the whole document.
	inheritanceNone inheritanceOutcome = iota
	// inheritanceResolved — the parent chain was spliced in successfully.
	inheritanceResolved
	// inheritanceUnreadable — the {~exons.extends~} declaration itself could not be read.
	inheritanceUnreadable
	// inheritanceNoEngine — the template extends, but has no engine to resolve the parent through.
	inheritanceNoEngine
	// inheritanceFailed — the parent was identified and the chain could not be resolved.
	inheritanceFailed
)

// resolveInheritance returns the AST that represents this document: the inheritance-resolved one
// when the template extends another and resolution succeeded, and the template's own AST in every
// other case, alongside the reason.
//
// Where the parent comes from is decided by inheritanceResolverFor: a per-call resolver on execCtx when
// one is set, else the engine registry, else there is nowhere to look. execCtx may be nil — the
// analysis callers (dryRunAST, Explain) hold a data map rather than a context and get the
// registry path, byte-for-byte what they did before the resolver existed.
//
// The AST is ALWAYS non-nil, including on failure. A caller whose job is analysis rather than
// execution (dryRunAST) degrades to the raw AST and reports the reason; a caller whose job is to
// produce the document (ExecuteWithContext, Explain) treats a non-nil error as fatal.
//
// ⚠ That fatality is a deliberate BEHAVIOUR CHANGE for two of the three failures. Both used to
// render the child's bare block bodies and return nil. That is not a degraded render — it is a
// DIFFERENT DOCUMENT reported as success, and a caller has no way to tell. dryRunAST already
// conceded as much: its no-engine message says the damage is identical to an outright resolution
// failure, which ExecuteWithContext has always returned.
func (t *Template) resolveInheritance(ctx context.Context, execCtx *Context) (*internal.RootNode, inheritanceOutcome, error) {
	// The {~exons.extends~} declaration itself could not be read, so inheritanceInfo is nil for a
	// template that DOES extend. Parse keeps that non-fatal on purpose — a template that cannot
	// state its parent can still be inspected — but executing it produces a document nobody wrote.
	if t.inheritanceErr != nil {
		return t.ast, inheritanceUnreadable, NewInheritanceError(ErrMsgInheritanceUnreadable, t.inheritanceErr)
	}

	if t.inheritanceInfo == nil {
		return t.ast, inheritanceNone, nil
	}

	resolver := t.inheritanceResolverFor(execCtx)
	if resolver == nil {
		return t.ast, inheritanceNoEngine, NewInheritanceError(ErrMsgInheritanceNoEngine, nil)
	}

	resolvedAST, err := resolver.ResolveInheritance(ctx, t.ast, t.inheritanceInfo, 0)
	if err != nil {
		// Wrapped, not passed through. The resolver's own error is an unexported internal type with
		// no Unwrap, so a consumer holding it could match neither a code nor a tag and had to
		// string-match a message — while the two failures above it in this same function returned a
		// typed one. Three siblings of one resolution, two machine-matchable. The specs walk
		// (ancestorSpecs) has always reported all of its conditions as typed errors; this closes the
		// asymmetry between the two walks rather than inventing a vocabulary.
		return t.ast, inheritanceFailed, NewInheritanceResolutionError(t.inheritanceInfo.ParentTemplate, err)
	}
	return resolvedAST, inheritanceResolved, nil
}

// inheritanceResolverFor builds the executor-side resolver for ONE render, choosing the source of
// parents in this order and stopping at the first that applies:
//
//  1. the per-call TemplateSourceResolver on execCtx, if set — it WINS over the registry outright,
//     including for a name the registry also knows, because a host that scopes parents to a
//     request must not have a process-global template answer in its place;
//  2. the engine registry, when the template has an engine;
//  3. nothing — nil, which the caller reports as ErrMsgInheritanceNoEngine.
//
// Both adapters strip frontmatter through the same helper (inheritanceBodyOf / TemplateBody), so
// the resolver is never handed a ---…--- block as content whichever source the parent came from.
func (t *Template) inheritanceResolverFor(execCtx *Context) *internal.InheritanceResolver {
	lexerConfig := internal.LexerConfig{
		OpenDelim:      t.config.openDelim,
		CloseDelim:     t.config.closeDelim,
		MarkdownFences: t.config.markdownFences,
	}
	if execCtx != nil {
		if r := execCtx.TemplateSourceResolver(); r != nil {
			return internal.NewContextInheritanceResolver(&contextSourceAdapter{resolver: r}, t.config.maxDepth, lexerConfig)
		}
	}
	if t.engine == nil {
		return nil
	}
	return internal.NewInheritanceResolver(nil, &engineSourceAdapter{engine: t.engine}, t.config.maxDepth, lexerConfig)
}

// templateProvider is the optional, richer form of TemplateExecutor: it hands back the parsed
// *Template rather than a source string, so a caller can reach the parent's already-stripped body
// and its already-parsed *Spec without re-parsing either. *Engine satisfies it.
//
// It is deliberately NOT folded into TemplateExecutor. That interface is EXPORTED, so widening it
// breaks every external implementer — on a library whose whole v0.23.0 story was earning a clean
// release line. An optional interface asserted at the use site costs nobody anything.
type templateProvider interface {
	GetTemplate(name string) (*Template, bool)
}

// engineSourceAdapter adapts TemplateExecutor to internal.TemplateSourceResolver, handing the
// inheritance resolver the parent's BODY rather than its raw source.
//
// The resolver lexes whatever it is given and the lexer has no frontmatter awareness — stripping
// happens only in internal.ExtractYAMLFrontmatter, called from Engine.Parse. So handing over
// tmpl.Source() spliced a parent's own ---…--- into the child's output as literal text, and for a
// MID-CHAIN parent it was worse than cosmetic: the frontmatter counts as content, so the parent's
// own {~exons.extends~} was no longer the first tag and the chain failed to parse at all.
type engineSourceAdapter struct {
	engine TemplateExecutor
}

func (a *engineSourceAdapter) GetTemplateSource(name string) (string, bool) {
	// An *Engine already holds the stripped body from its own Parse. Prefer it, so there is ONE
	// statement of what frontmatter is rather than two that can disagree.
	if provider, ok := a.engine.(templateProvider); ok {
		if tmpl, found := provider.GetTemplate(name); found {
			return tmpl.TemplateBody(), true
		}
		return "", false
	}

	// A third-party TemplateExecutor exposes only raw source; strip it through the same helper
	// Engine.Parse uses, never a second definition of the delimiters — see inheritanceBodyOf.
	source, found := a.engine.GetTemplateSource(name)
	if !found {
		return "", false
	}
	return inheritanceBodyOf(source), true
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

// StripMessageMarkers removes the {~exons.message~} framing from executed output and
// returns the plain text — message contents and unmarked prose alike, in their original
// order.
//
// ⚠ IT EXISTS BECAUSE THE MARKERS CONTAIN NUL BYTES AND NOT EVERY CONSUMER IS A MESSAGE
// SPLITTER. The framing is `\x00MSG_START:<role>:<cache>:` … `\x00MSG_END\x00`, so a caller
// that renders a template and uses the result as text carries those NULs onward. Measured
// downstream (vaichat2 → thalamus → postgres): a `jsonb` column REFUSES ` `
// (SQLSTATE 22P05), so a prompt template using the message tag failed the whole send with an
// opaque database error. A text/plain body, a log line and a filename are the same hazard,
// one step less loudly.
//
// ⚠ IT IS NOT A SUBSTITUTE FOR ExtractMessagesFromOutput AND MUST NOT BE USED AS ONE. It
// FLATTENS roles: a template emitting a system and a user message yields one string with no
// boundary between them. A caller that can carry structured messages should extract them;
// this is for the caller that has exactly one string to fill.
//
// ⚠ AND IT KEEPS UNMARKED PROSE, WHICH IS THE OTHER HALF. ExtractMessagesFromOutput returns
// only what sits BETWEEN markers, so body text outside them is silently discarded — data loss
// for a single-string caller, and a template mixing plain body text with one message tag is
// an ordinary authoring shape.
//
// Output with no markers is returned unchanged, so this is safe to call unconditionally.
func StripMessageMarkers(output string) string {
	if !strings.Contains(output, internal.MessageStartMarker) {
		// The common case: no message tag was used. Nothing to unframe — but a lone end
		// marker is malformed output that still carries NULs, so it is removed either way.
		return strings.ReplaceAll(output, internal.MessageEndMarker, "")
	}
	var b strings.Builder
	b.Grow(len(output))
	rest := output
	for {
		start := strings.Index(rest, internal.MessageStartMarker)
		if start < 0 {
			b.WriteString(rest)
			break
		}
		// Everything before the marker is prose the author wrote outside any message.
		b.WriteString(rest[:start])
		rest = rest[start+len(internal.MessageStartMarker):]
		// The header is `<role>:<cache>:` — two field separators. A truncated header means
		// malformed output; drop what is left of it rather than emit a NUL.
		for i := 0; i < 2; i++ {
			sep := strings.Index(rest, internal.MessageFieldSep)
			if sep < 0 {
				rest = ""
				break
			}
			rest = rest[sep+len(internal.MessageFieldSep):]
		}
	}
	// The end markers carry the remaining NULs and delimit nothing the caller needs.
	return strings.ReplaceAll(b.String(), internal.MessageEndMarker, "")
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
