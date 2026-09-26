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
	envDisabled    bool     // completely disable {~exons.env~}
	markdownFences bool     // markdown code fences are inert regions
	refVerbatim    bool     // {~exons.ref~} splices the referenced body as TEXT instead of rendering it
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

// WithEnvAllowlist restricts {~exons.env~} to only allow environment variables
// matching the given glob patterns (case-insensitive, filepath.Match syntax).
// If set, only matching variables are accessible; all others are blocked.
// The denylist is still checked first.
// Pass nil to clear any previously set allowlist.
func WithEnvAllowlist(patterns []string) Option {
	return func(c *engineConfig) {
		c.envAllowlist = patterns
	}
}

// WithEnvDenylist sets glob patterns for environment variable names that are
// blocked from access via {~exons.env~} (case-insensitive, filepath.Match syntax).
// Default: DefaultEnvDenyPatterns (blocks *_KEY, *_SECRET, *_TOKEN, etc.)
// Pass nil to allow all env vars (no deny filtering).
func WithEnvDenylist(patterns []string) Option {
	return func(c *engineConfig) {
		c.envDenylist = patterns
	}
}

// WithEnvDisabled completely disables the {~exons.env~} tag.
// Any use will return an error.
func WithEnvDisabled() Option {
	return func(c *engineConfig) {
		c.envDisabled = true
	}
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
func WithRefVerbatim() Option {
	return func(c *engineConfig) {
		c.refVerbatim = true
	}
}
