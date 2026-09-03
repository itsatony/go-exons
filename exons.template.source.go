package exons

import (
	"context"
	"sync"

	"github.com/itsatony/go-exons/internal"
)

// TemplateSourceResolver resolves the parent named by {~exons.extends template="…" /~} to its
// SOURCE, per call and with the request's context in hand.
//
// It exists because the engine's registry (Engine.RegisterTemplate) is process-global: one name,
// one parent, for every caller. A multi-tenant host whose parents live in a remote store reached
// with the REQUEST's identity cannot use it — the parent a document may extend depends on who is
// asking, and the answer may be "you are not allowed to see that", which a registry cannot say.
// Attach an implementation to the execution context with Context.WithTemplateSourceResolver and
// BOTH walks over the extends chain — the execute-time render and the declaration-time inputs walk
// (DeclaredInputsWithContext, ValidateInputBindingWithContext) — consult it, so the two keep
// reporting one vocabulary for one chain.
//
// The source may carry YAML frontmatter; the library strips it exactly as Engine.Parse does, so a
// host hands back whatever it stores and never re-implements the delimiter rules.
//
// The answer is THREE-VALUED and the distinction is the contract:
//
//   - found=false, err=nil — no such template. Reported as a template-not-found inheritance
//     failure naming the parent.
//   - err != nil — the LOOKUP itself failed (unauthorized, unreachable, timed out). Reported as an
//     inheritance failure whose unwrap chain contains err, so the host can name the real reason.
//     It is never rewritten as "not found": an author whose parent name is correct must not be
//     sent hunting for a typo because the store was down.
//
// When a resolver is present on the context it is the ONLY source of parents for that render —
// the engine registry is not consulted as a fallback, because a fallback is precisely how a
// process-global template would leak into a request that scoped its parents to an identity.
type TemplateSourceResolver interface {
	ResolveTemplateSource(ctx context.Context, name string) (source string, found bool, err error)
}

// TemplateSourceFunc adapts a plain function to TemplateSourceResolver.
type TemplateSourceFunc func(ctx context.Context, name string) (string, bool, error)

// ResolveTemplateSource implements TemplateSourceResolver.
func (f TemplateSourceFunc) ResolveTemplateSource(ctx context.Context, name string) (string, bool, error) {
	return f(ctx, name)
}

// Extends reports the parent this template declares with {~exons.extends template="…" /~}.
//
// ok is false when the template extends nothing — and ALSO when the extends declaration could not
// be read (the tag is present but its parent cannot be named). A host that needs to tell those two
// apart should render or DryRun the template, which report the unreadable declaration as an error;
// this accessor answers only "is there a parent I could go and fetch".
func (t *Template) Extends() (parent string, ok bool) {
	if t.inheritanceInfo == nil || t.inheritanceInfo.ParentTemplate == "" {
		return "", false
	}
	return t.inheritanceInfo.ParentTemplate, true
}

// contextSourceAdapter adapts a public TemplateSourceResolver to the executor's
// internal.ContextTemplateSourceResolver, handing the inheritance resolver the parent's BODY rather
// than its raw source. Same reason as engineSourceAdapter, same helper — see
// inheritanceBodyOf.
type contextSourceAdapter struct {
	resolver TemplateSourceResolver
}

func (a *contextSourceAdapter) ResolveTemplateSource(ctx context.Context, name string) (string, bool, error) {
	source, found, err := a.resolver.ResolveTemplateSource(ctx, name)
	if err != nil {
		return "", false, err
	}
	if !found {
		return "", false, nil
	}
	return inheritanceBodyOf(source), true, nil
}

// inheritanceBodyOf strips a parent's YAML frontmatter so the inheritance resolver — whose lexer
// has no frontmatter awareness — is never handed a ---…--- block as content.
//
// It is the ONE statement of that strip for every source-shaped parent (a third-party
// TemplateExecutor's registry, a context resolver's store), and it goes through the same helper
// Engine.Parse uses rather than a second definition of the delimiters. A parent whose frontmatter
// is malformed falls back to its raw source rather than to "not found": the parse that follows
// reports the real reason, and a bool cannot carry it.
func inheritanceBodyOf(source string) string {
	result, err := internal.ExtractYAMLFrontmatter(source)
	if err != nil || result == nil {
		return source
	}
	return result.TemplateBody
}

// memoizedTemplateSource answers a repeated question within ONE render exactly once.
//
// ExecuteWithContext walks the extends chain twice — once to collect declared inputs
// (contextWithInputs) and once to splice the parent bodies in — and against the registry that
// costs nothing. Against a per-call resolver every parent would be fetched twice per render, a
// second round trip to a remote store for no new information. Worse than the cost: a store whose
// answer CHANGED between the two walks would hand the render a parent the inputs walk never saw.
// Caching every outcome, failures included, makes the two walks describe one chain by
// construction.
//
// It is built per ExecuteWithContext call and never outlives it, so nothing is cached across
// requests or identities.
type memoizedTemplateSource struct {
	inner TemplateSourceResolver
	mu    sync.Mutex
	seen  map[string]templateSourceAnswer
}

type templateSourceAnswer struct {
	source string
	found  bool
	err    error
}

func memoizeTemplateSource(inner TemplateSourceResolver) *memoizedTemplateSource {
	return &memoizedTemplateSource{inner: inner, seen: make(map[string]templateSourceAnswer)}
}

func (m *memoizedTemplateSource) ResolveTemplateSource(ctx context.Context, name string) (string, bool, error) {
	m.mu.Lock()
	answer, ok := m.seen[name]
	m.mu.Unlock()
	if ok {
		return answer.source, answer.found, answer.err
	}
	source, found, err := m.inner.ResolveTemplateSource(ctx, name)
	m.mu.Lock()
	m.seen[name] = templateSourceAnswer{source: source, found: found, err: err}
	m.mu.Unlock()
	return source, found, err
}
