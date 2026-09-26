package internal

import (
	"context"
	"regexp"
	"strconv"
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
			WithMetadata(LogFieldDepth, strconv.Itoa(depth))
	}

	// Check for circular reference
	chain := getRefChain(execCtx)
	for _, refSlug := range chain {
		if refSlug == slug {
			// Appended into a FRESH slice. `chain` is the context's live one, and Context.fromState
			// and Context.Child alias it rather than copying, so appending in place is safe only
			// while every producer happens to hand back a slice with no spare capacity. That is an
			// invariant nobody is enforcing, and the failure it buys is two sibling frames writing
			// over each other's chain.
			reported := make([]string, 0, len(chain)+1)
			reported = append(reported, chain...)
			reported = append(reported, slug)
			return "", NewRefCircularError(slug, reported)
		}
	}

	// Resolve the spec — and, when the resolver can, RENDER it.
	//
	// Until v0.34.0 this returned the body and executeTag spliced it in as text, so a referenced
	// document's own {~exons.ref~}, {~exons.var~} and {~exons.now~} reached the output as literal
	// tags. ⭐ The two guards directly above are the evidence that this was never the intent: the
	// depth limit and the circular-chain check could not fire, because nothing pushed a frame —
	// they are dead code describing the recursion the port never wired (vAudience/atlas#696).
	//
	// A renderer implementation pushes that frame (depth+1, chain+slug) and executes the body in
	// a context derived from this one, which is what makes both guards live for the first time.
	if renderer, isRenderer := resolver.(SpecRefRenderer); isRenderer {
		out, lookupFailed, err := renderer.RenderSpecRef(ctx, execCtx, slug, version)
		if err == nil {
			return out, nil
		}
		// ⛔ Two conditions, two messages. "Not found" sends an author to check the slug;
		// "could not be rendered" sends them into the referenced document. Collapsing them
		// would be this codebase's most-repeated defect committed on purpose.
		if lookupFailed {
			return "", NewBuiltinError(AppendHint(ErrMsgRefNotFound, HintRefNotFound), TagNameRef).
				WithMetadata(LogFieldSpecSlug, slug).
				WithMetadata(LogFieldSpecVersion, version).
				WithCause(err)
		}
		return "", NewBuiltinError(ErrMsgRefRenderFailed, TagNameRef).
			WithMetadata(LogFieldSpecSlug, slug).
			WithMetadata(LogFieldSpecVersion, version).
			WithCause(err)
	}

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

// SpecRefRenderer is the richer contract a spec resolver MAY implement: resolve the reference
// and return its RENDERED output, executed in a context derived from execCtx with the reference
// frame pushed.
//
// It is optional on purpose. A resolver that implements only SpecBodyResolver keeps the pre-
// v0.34.0 verbatim splice, which is the honest behaviour for one that already returns rendered
// text — re-executing rendered text is a no-op right up until the rendered text contains a
// literal {~…~}, and then it is an unknown-tag failure where there used to be inert prose.
//
// ⛔ The classification travels BESIDE the error, never inside it. A sentinel wrapped into the
// error was the obvious spelling and was wrong: BuiltinError and ExecutorError both expose their
// cause through Unwrap, so an INNER reference's lookup sentinel stayed reachable by errors.Is
// from every outer frame — and a four-deep chain reported "referenced spec not found" four times,
// each naming a slug that resolves perfectly. ⭐ A sentinel that travels answers a question about
// the whole chain when it was asked about one link.
type SpecRefRenderer interface {
	// RenderSpecRef returns the rendered output, or an error. lookupFailed is true only when it
	// was THIS reference's own lookup that failed — it is decided by the only code that knows.
	RenderSpecRef(
		ctx context.Context,
		execCtx interface{},
		slug string,
		version string,
	) (out string, lookupFailed bool, err error)
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

// isValidSpecSlug validates the spec slug format. Two forms are accepted:
//   - a BARE slug — start with a lowercase letter, then lowercase letters, digits, or
//     hyphens (e.g. "greeting"); the spec resolver supplies the namespace.
//   - a NAMESPACE-QUALIFIED slug of the form "@org/name" (e.g. "@aigentverse/source-scout"),
//     the portable cross-namespace reference contract used by registries that address specs
//     by "@org/name". Each segment is lowercase alphanumeric/hyphen. A trailing "@version" is
//     already stripped by the caller before validation, so the only "@" reaching a qualified
//     slug is the leading namespace marker.
var (
	specSlugRegex          = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	qualifiedSpecSlugRegex = regexp.MustCompile(`^@[a-z0-9][a-z0-9-]*/[a-z0-9][a-z0-9-]*$`)
)

func isValidSpecSlug(slug string) bool {
	if slug == "" {
		return false
	}
	return specSlugRegex.MatchString(slug) || qualifiedSpecSlugRegex.MatchString(slug)
}

// NewRefCircularError creates an error for circular reference detection.
func NewRefCircularError(slug string, chain []string) *BuiltinError {
	chainStr := strings.Join(chain, " -> ")
	return NewBuiltinError(ErrMsgRefCircular, TagNameRef).
		WithMetadata(LogFieldSpecSlug, slug).
		WithMetadata(LogFieldRefChain, chainStr)
}
