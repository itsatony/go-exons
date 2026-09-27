package exons

import (
	"log/slog"
)

// Option is a functional option for configuring the Engine.
type Option func(*engineConfig)

// engineConfig holds the internal configuration for an Engine.
type engineConfig struct {
	openDelim      string
	closeDelim     string
	errorStrategy  ErrorStrategy
	maxDepth       int
	maxOutputSize  int
	logger         *slog.Logger
	envAllowlist   []string // glob patterns; if set, only matching env vars allowed
	envDenylist    []string // glob patterns; matching env vars are blocked
	envEnabled     bool     // {~exons.env~} opted in for every name (WithEnvEnabled); off by default
	envDisabled    bool     // {~exons.env~} refused even if opted in (WithEnvDisabled wins)
	markdownFences bool     // markdown code fences are inert regions
	refVerbatim    bool     // {~exons.ref~} splices the referenced body as TEXT instead of rendering it
	strictAttrs    bool     // Parse/ParseBody also refuse the error-severity issues Validate reports
}

// defaultEngineConfig returns the default engine configuration.
func defaultEngineConfig() *engineConfig {
	return &engineConfig{
		openDelim:     DefaultOpenDelim,
		closeDelim:    DefaultCloseDelim,
		errorStrategy: ErrorStrategyThrow,
		maxDepth:      DefaultMaxDepth,
		maxOutputSize: DefaultMaxOutputSize,
		logger:        slog.Default(),
		envDenylist:   DefaultEnvDenyPatterns(),
	}
}

// WithDelimiters sets custom delimiters for template tags.
// Default: "{~" and "~}"
func WithDelimiters(open, close string) Option {
	return func(c *engineConfig) {
		if open != "" {
			c.openDelim = open
		}
		if close != "" {
			c.closeDelim = close
		}
	}
}

// WithErrorStrategy sets the error handling strategy.
// Default: ErrorStrategyThrow
func WithErrorStrategy(strategy ErrorStrategy) Option {
	return func(c *engineConfig) {
		c.errorStrategy = strategy
	}
}

// WithMaxDepth sets the maximum nesting depth for templates.
// Use 0 for unlimited depth.
// Default: 10
func WithMaxDepth(depth int) Option {
	return func(c *engineConfig) {
		c.maxDepth = depth
	}
}

// WithMaxOutputSize sets the maximum rendered output size in bytes.
// Use 0 for unlimited output.
// Default: 10MB (DefaultMaxOutputSize)
func WithMaxOutputSize(size int) Option {
	return func(c *engineConfig) {
		c.maxOutputSize = size
	}
}

// WithLogger sets the logger for the engine.
// Default: slog.Default()
func WithLogger(logger *slog.Logger) Option {
	return func(c *engineConfig) {
		if logger != nil {
			c.logger = logger
		}
	}
}

// WithEnvEnabled opts in to the {~exons.env~} tag for every variable name the denylist does not
// block (DefaultEnvDenyPatterns unless replaced with WithEnvDenylist).
//
// ⛔ Since v0.35.0 {~exons.env~} is OFF by default. Every known consumer renders templates it did
// not write — third-party agent specs, referenced fragments, caller uploads — and for all of them a
// denylist-guarded env read is a disclosure primitive: an ordinary variable name (DATABASE_URL,
// OPENAI_BASE_URL, an internal hostname) passes a suffix denylist. Enable it only when every
// template the engine renders is trusted; otherwise prefer WithEnvAllowlist with explicit names.
func WithEnvEnabled() Option {
	return func(c *engineConfig) {
		c.envEnabled = true
	}
}

// WithEnvAllowlist opts in to {~exons.env~} for ONLY the environment variables matching the given
// glob patterns (case-insensitive, filepath.Match syntax); every other name is refused.
//
// A non-empty allowlist implies opt-in — WithEnvEnabled is not needed alongside it. The denylist
// is still checked first, so a listed name that also matches a deny pattern stays blocked.
// Pass nil (or an empty slice) to clear a previously set allowlist; the tag then reverts to
// disabled unless WithEnvEnabled was also given.
func WithEnvAllowlist(patterns []string) Option {
	return func(c *engineConfig) {
		c.envAllowlist = patterns
	}
}

// WithEnvDenylist sets glob patterns for environment variable names that are blocked from access
// via {~exons.env~} (case-insensitive, filepath.Match syntax). The denylist is checked before the
// allowlist and takes priority over it.
// Default: DefaultEnvDenyPatterns (blocks *_KEY, *_SECRET, *_TOKEN, etc.)
// Pass nil to remove deny filtering.
//
// ⚠ The denylist does NOT enable the tag. It only narrows an engine that opted in through
// WithEnvEnabled or WithEnvAllowlist; on an engine that did not opt in, every {~exons.env~} is
// refused regardless of the denylist.
func WithEnvDenylist(patterns []string) Option {
	return func(c *engineConfig) {
		c.envDenylist = patterns
	}
}

// WithEnvDisabled refuses the {~exons.env~} tag outright, and wins over WithEnvEnabled and
// WithEnvAllowlist in any order.
//
// Since v0.35.0 this is the default, so the option is redundant on its own. It is kept, working,
// so that a consumer which pinned the refusal explicitly stays refused, and as a hard override for
// a caller that composes option lists it does not fully control.
func WithEnvDisabled() Option {
	return func(c *engineConfig) {
		c.envDisabled = true
	}
}

// envAccessEnabled reports whether {~exons.env~} may read the environment at all: an explicit
// WithEnvDisabled refuses, otherwise WithEnvEnabled or a non-empty allowlist opts in.
func (c *engineConfig) envAccessEnabled() bool {
	if c.envDisabled {
		return false
	}
	return c.envEnabled || len(c.envAllowlist) > 0
}

// WithMarkdownFences makes markdown code fences inert: exons tags, escapes,
// and verbatim fences inside a fenced code block (``` or ~~~, per a
// CommonMark subset) pass through as literal text instead of rendering.
// A fence opts back into live rendering when the first word of its info
// string is "exons" (e.g. ```exons).
//
// Intended for markdown-format templates (SKILL.md-style bodies; see
// Spec.ContentFormat). Off by default: plain templates commonly interpolate
// variables inside code fences.
func WithMarkdownFences() Option {
	return func(c *engineConfig) {
		c.markdownFences = true
	}
}

// WithRefVerbatim makes {~exons.ref~} splice the referenced document's body into the output as
// TEXT, without parsing or executing it — the only behaviour go-exons had before v0.34.0.
//
// ⛔ This is a migration path, not a configuration preference. It exists for one shape of
// SpecResolver: one whose ResolveSpec returns text that is ALREADY fully rendered, and which
// therefore performs its own reference recursion, cycle detection and budget. Rendering such a
// body a second time is a no-op right up until it contains a literal {~…~} — a quoted example, a
// fragment about the syntax itself, a user's own prose — and then it is an unknown-tag failure
// where there used to be inert text.
//
// ⚠ With it set, RefMaxDepth and the circular-reference check DO NOT APPLY, because nothing
// pushes a reference frame for them to read. A resolver that opts in owns those bounds itself.
// The fix is to return the raw body and let go-exons resolve the chain; see RenderSpecRef.
//
// ⛔ A body spliced through this option is a plain string, and since v0.34.1 every message
// marker in it is STRIPPED — the string carries no proof it was rendered, and an unproven NUL
// is a forged message. A resolver whose bodies are go-exons renders with messages that must
// survive implements RenderedSpecResolver instead (go-exons#9).
func WithRefVerbatim() Option {
	return func(c *engineConfig) {
		c.refVerbatim = true
	}
}

// WithStrictAttributes makes Parse and ParseBody refuse a template whose tags their own resolvers
// refuse: `{~exons.include ref="x" /~}` with no template=, `{~exons.message~}` with no role=, an
// onerror= that names no strategy, a for-loop with no item=. By default Parse judges GRAMMAR only
// and those documents parse, then fail at every render (go-exons#11).
//
// With it set, Parse walks the parsed AST with the same traversal Engine.Validate uses — every
// tag, its children, and every branch of if / for / switch — and refuses exactly the issues
// Validate would report with SeverityError. Warnings (an unknown tag, a missing include target)
// are never refused: a resolver or template can still be registered after the parse. The refusal
// matches errors.Is(err, ErrStrictAttributes), and errors.As with a *StrictAttributeError yields
// every issue with its tag name and position.
//
// ⛔ It is OPT-IN on purpose. A stricter default would newly refuse STORED documents in every
// consumer at once — registries, runtimes and caches that parse what they persisted months ago —
// on nothing more than a library bump. A consumer turns it on at its intake or publish gate, where
// a refusal reaches the author who can fix it.
//
// It judges the DOCUMENT, not a render, so no error strategy lifts it — neither the engine's nor a
// tag's own onerror=. A tag that would fail at render but be removed by onerror="remove" is still
// refused, exactly as Validate reports it.
//
// Frontmatter (decided, go-exons#11): Parse renders `{~…~}` tags inside the YAML frontmatter
// through the same engine before decoding it, and that render parses through Parse, so on a strict
// engine the frontmatter's tags are checked too. Under the default throw strategy that render
// already refuses such a tag; under a lenient strategy it would be kept or removed silently, and
// strict mode is what refuses it. Such a refusal arrives wrapped as the frontmatter error (ErrCodeConfig)
// with positions relative to the frontmatter text; errors.Is(err, ErrStrictAttributes) still
// matches. Templates registered with RegisterTemplate are parsed through Parse and checked as well,
// and a body {~exons.ref~} resolves at execute time goes through ParseBody and is checked there.
func WithStrictAttributes() Option {
	return func(c *engineConfig) {
		c.strictAttrs = true
	}
}
