package internal

import (
	"context"
	"regexp"
	"strings"
)

// RefResolver handles the exons.ref built-in tag.
// It resolves references to other specs by slug and version.
type RefResolver struct{}

// NewRefResolver creates a new RefResolver.
func NewRefResolver() *RefResolver {
	return &RefResolver{}
}

// TagName returns the tag name for this resolver.
func (r *RefResolver) TagName() string {
	return TagNameRef
}

// Resolve resolves the spec reference and returns the spec template body.
func (r *RefResolver) Resolve(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error) {
	// Get slug attribute (required)
	slug, ok := attrs.Get(AttrSlug)
	if !ok || slug == "" {
		return "", NewBuiltinError(ErrMsgRefMissingSlug, TagNameRef)
	}

	// Parse slug@version syntax if present (do this BEFORE validation)
	version := RefVersionLatest
	if atIdx := strings.LastIndex(slug, "@"); atIdx > 0 {
		version = slug[atIdx+1:]
		slug = slug[:atIdx]
	}

	// Validate slug format (after extracting version)
	if !isValidSpecSlug(slug) {
		return "", NewBuiltinError(ErrMsgRefInvalidSlug, TagNameRef).
			WithMetadata(LogFieldSpecSlug, slug)
	}

	// Version attribute overrides @version syntax
	if v, hasVersion := attrs.Get(AttrVersion); hasVersion && v != "" {
		version = v
	}

	// Get spec resolver from context
	resolver, ok := getSpecResolver(execCtx)
	if !ok {
		return "", NewBuiltinError(AppendHint(ErrMsgRefNoResolver, HintRefNoResolver), TagNameRef)
	}

	// Check reference depth
	depth := getRefDepth(execCtx)
	if depth >= RefMaxDepth {
		return "", NewBuiltinError(ErrMsgRefDepthExceeded, TagNameRef).
			WithMetadata(LogFieldSpecSlug, slug).
			WithMetadata("depth", string(rune(depth+'0')))
	}

	// Check for circular reference
	chain := getRefChain(execCtx)
	for _, refSlug := range chain {
		if refSlug == slug {
			return "", NewRefCircularError(slug, append(chain, slug))
		}
	}

	// Resolve the spec
	body, err := resolver.ResolveSpecBody(ctx, slug, version)
	if err != nil {
		return "", NewBuiltinError(AppendHint(ErrMsgRefNotFound, HintRefNotFound), TagNameRef).
			WithMetadata(LogFieldSpecSlug, slug).
			WithMetadata(LogFieldSpecVersion, version)
	}

	return body, nil
}

// Validate checks that the required attributes are present.
func (r *RefResolver) Validate(attrs Attributes) error {
	if !attrs.Has(AttrSlug) {
		return NewBuiltinError(ErrMsgRefMissingSlug, TagNameRef)
	}
	return nil
}

// SpecBodyResolver provides spec body lookup for reference resolution.
// This is the internal interface used by the ref resolver.
type SpecBodyResolver interface {
	// ResolveSpecBody looks up a spec by slug and version and returns its template body.
	ResolveSpecBody(ctx context.Context, slug string, version string) (string, error)
}

// SpecResolverAccessor provides access to a spec resolver from context.
// The returned interface{} should implement SpecBodyResolver.
type SpecResolverAccessor interface {
	// SpecResolver returns the spec resolver for reference resolution.
	// Returns interface{} to avoid import cycles - the returned value should
	// implement SpecBodyResolver or have a matching ResolveSpecBody method.
	SpecResolver() interface{}
}

// RefDepthAccessor provides access to the current reference depth.
type RefDepthAccessor interface {
	// RefDepth returns the current reference resolution depth.
	RefDepth() int
}

// RefChainAccessor provides access to the current reference chain.
type RefChainAccessor interface {
	// RefChain returns the current chain of referenced spec slugs.
	RefChain() []string
}

// getSpecResolver extracts the spec resolver from context.
func getSpecResolver(execCtx interface{}) (SpecBodyResolver, bool) {
	if accessor, ok := execCtx.(SpecResolverAccessor); ok {
		resolverVal := accessor.SpecResolver()
		if resolverVal == nil {
			return nil, false
		}
		// Type assert to SpecBodyResolver
		if resolver, ok := resolverVal.(SpecBodyResolver); ok {
			return resolver, true
		}
	}
	return nil, false
}

// getRefDepth extracts the reference depth from context.
func getRefDepth(execCtx interface{}) int {
	if accessor, ok := execCtx.(RefDepthAccessor); ok {
		return accessor.RefDepth()
	}
	return 0
}

// getRefChain extracts the reference chain from context.
func getRefChain(execCtx interface{}) []string {
	if accessor, ok := execCtx.(RefChainAccessor); ok {
		return accessor.RefChain()
	}
	return nil
}

// isValidSpecSlug validates the spec slug format.
// Must start with lowercase letter, followed by lowercase letters, digits, or hyphens.
var specSlugRegex = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func isValidSpecSlug(slug string) bool {
	if slug == "" {
		return false
	}
	return specSlugRegex.MatchString(slug)
}

// NewRefCircularError creates an error for circular reference detection.
func NewRefCircularError(slug string, chain []string) *BuiltinError {
	chainStr := strings.Join(chain, " -> ")
	return NewBuiltinError(ErrMsgRefCircular, TagNameRef).
		WithMetadata(LogFieldSpecSlug, slug).
		WithMetadata(LogFieldRefChain, chainStr)
}
