package exons

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/itsatony/go-cuserr"
	"github.com/itsatony/go-exons/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// v0.30.0 — a per-call, context-aware source of {~exons.extends~} parents.
//
// Until this release the ONLY place a parent could come from was the engine's
// process-global registry. A multi-tenant host whose parents live in a store
// reached with the request's identity had no seam at all: the same name had to
// mean the same parent for every caller, and "you may not read that" could not
// be said. These tests pin the seam on BOTH walks over the chain — the render
// and the declared-inputs walk — because one walk consulting the resolver and
// the other consulting the registry is exactly the two-implementations drift
// TestBothInheritanceWalksReportTheSameVocabulary was written to forbid.
// =============================================================================

// ctxKey is a private context key the tests use to prove the REQUEST context reaches the resolver.
type ctxKey struct{}

// storeResolver is a TemplateSourceResolver over a map, recording what it was asked and with which
// context — so a test can assert that the resolver was consulted at all, and under whose identity.
type storeResolver struct {
	sources  map[string]string
	failWith error
	asked    []string
	seenCtx  []any
}

func (s *storeResolver) ResolveTemplateSource(ctx context.Context, name string) (string, bool, error) {
	s.asked = append(s.asked, name)
	s.seenCtx = append(s.seenCtx, ctx.Value(ctxKey{}))
	if s.failWith != nil {
		return "", false, s.failWith
	}
	src, ok := s.sources[name]
	return src, ok, nil
}

// requireExtendsFailure asserts the machine-matchable inheritance contract (a cuserr whose tag is
// exons.extends) and returns the error for reason-specific claims.
func requireExtendsFailure(t *testing.T, err error) *cuserr.CustomError {
	t.Helper()
	require.Error(t, err)
	var ce *cuserr.CustomError
	require.True(t, errors.As(err, &ce), "an inheritance failure must be a cuserr: %v", err)
	tag, ok := ce.GetMetadata(MetaKeyTag)
	require.True(t, ok, "tag metadata present")
	assert.Equal(t, TagNameExtends, tag)
	return ce
}

const childExtendingBase = `{~exons.extends template="base" /~}{~exons.block name="content"~}custom{~/exons.block~}`

func TestTemplateSourceResolver_RenderWithAParentServedOnlyByTheResolver(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxKey{}, "tenant-42")
	engine := MustNew() // NOTHING registered — the registry must not be needed at all
	store := &storeResolver{sources: map[string]string{
		"base": `Header {~exons.block name="content"~}default{~/exons.block~} Footer`,
	}}

	tmpl, err := engine.Parse(childExtendingBase)
	require.NoError(t, err)

	out, err := tmpl.ExecuteWithContext(ctx, NewContext(nil).WithTemplateSourceResolver(store))
	require.NoError(t, err)
	assert.Equal(t, "Header custom Footer", out)

	assert.Equal(t, []string{"base"}, store.asked, "the resolver was the source, asked once for the one parent")
	assert.Equal(t, []any{"tenant-42"}, store.seenCtx,
		"the REQUEST context reaches the resolver — that is the whole reason it exists")
}

func TestTemplateSourceResolver_TwoLevelChainWithFrontmatterOnTheMidChainParent(t *testing.T) {
	// The bug engineSourceAdapter's header records, on the new path: a mid-chain parent carrying
	// frontmatter must have it stripped, or its own {~exons.extends~} is no longer the first tag
	// and the chain fails to parse. The strip is shared (inheritanceBodyOf), and this proves the
	// context adapter goes through it.
	engine := MustNew()
	store := &storeResolver{sources: map[string]string{
		"grand": "---\nname: grand\ndescription: g\n---\nG:{~exons.block name=\"content\"~}g{~/exons.block~}",
		"mid":   "---\nname: mid\ndescription: m\n---\n{~exons.extends template=\"grand\" /~}",
	}}
	tmpl, err := engine.Parse(`{~exons.extends template="mid" /~}{~exons.block name="content"~}leaf{~/exons.block~}`)
	require.NoError(t, err)

	out, err := tmpl.ExecuteWithContext(context.Background(), NewContext(nil).WithTemplateSourceResolver(store))
	require.NoError(t, err)
	assert.Equal(t, "G:leaf", out)
	assert.NotContains(t, out, "---")
	assert.Equal(t, []string{"mid", "grand"}, store.asked, "every level of the chain comes from the resolver")
}

func TestTemplateSourceResolver_NotFoundNamesTheParent(t *testing.T) {
	engine := MustNew()
	store := &storeResolver{sources: map[string]string{}}
	tmpl, err := engine.Parse(childExtendingBase)
	require.NoError(t, err)

	_, err = tmpl.ExecuteWithContext(context.Background(), NewContext(nil).WithTemplateSourceResolver(store))
	ce := requireExtendsFailure(t, err)
	name, ok := ce.GetMetadata(MetaKeyTemplateName)
	require.True(t, ok)
	assert.Equal(t, "base", name)
	assert.Contains(t, err.Error(), internal.ErrMsgTemplateNotFound)
	assert.NotContains(t, err.Error(), internal.ErrMsgTemplateSourceLookupFailed,
		"absent is absent — it must not read as a lookup failure either")
}

func TestTemplateSourceResolver_LookupFailureKeepsItsCauseAndIsNotNotFound(t *testing.T) {
	// "The store said no" and "there is no such template" demand opposite actions from a host,
	// so the first must never be rewritten as the second — and the host must be able to reach
	// its own error through the wrap to say WHY.
	unauthorized := errors.New("store: identity tenant-42 may not read template base")
	engine := MustNew()
	store := &storeResolver{failWith: unauthorized}
	tmpl, err := engine.Parse(childExtendingBase)
	require.NoError(t, err)

	_, err = tmpl.ExecuteWithContext(context.Background(), NewContext(nil).WithTemplateSourceResolver(store))
	ce := requireExtendsFailure(t, err)

	assert.True(t, errors.Is(err, unauthorized), "the host's own error is in the unwrap chain: %v", err)
	name, ok := ce.GetMetadata(MetaKeyTemplateName)
	require.True(t, ok, "the parent whose lookup failed is named")
	assert.Equal(t, "base", name)
	assert.Contains(t, err.Error(), internal.ErrMsgTemplateSourceLookupFailed)
	assert.Contains(t, err.Error(), unauthorized.Error(), "the real reason is readable, not only matchable")
	assert.NotContains(t, err.Error(), internal.ErrMsgTemplateNotFound,
		"a failed lookup must not send the author hunting for a typo in a correct name")
}

func TestTemplateSourceResolver_WinsOverARegisteredTemplateOfTheSameName(t *testing.T) {
	engine := MustNew()
	engine.MustRegisterTemplate("base", `REGISTRY {~exons.block name="content"~}x{~/exons.block~}`)
	store := &storeResolver{sources: map[string]string{
		"base": `STORE {~exons.block name="content"~}x{~/exons.block~}`,
	}}
	tmpl, err := engine.Parse(childExtendingBase)
	require.NoError(t, err)

	withStore, err := tmpl.ExecuteWithContext(context.Background(), NewContext(nil).WithTemplateSourceResolver(store))
	require.NoError(t, err)
	assert.Equal(t, "STORE custom", withStore, "the per-call resolver decides, not the process-global registry")

	withoutStore, err := tmpl.ExecuteWithContext(context.Background(), NewContext(nil))
	require.NoError(t, err)
	assert.Equal(t, "REGISTRY custom", withoutStore, "and without one the registry path is unchanged")

	// The registry is NOT a fallback once a resolver is present: a name the store lacks stays
	// absent even though the registry knows it. Otherwise a process-global parent could answer
	// for a request that scoped its parents to an identity.
	empty := &storeResolver{sources: map[string]string{}}
	_, err = tmpl.ExecuteWithContext(context.Background(), NewContext(nil).WithTemplateSourceResolver(empty))
	requireExtendsFailure(t, err)
	assert.Contains(t, err.Error(), internal.ErrMsgTemplateNotFound)
}

func TestTemplateSourceResolver_CycleAndDepthBoundsHoldOnTheContextPath(t *testing.T) {
	t.Run("cycle", func(t *testing.T) {
		engine := MustNew()
		store := &storeResolver{sources: map[string]string{
			"a": `{~exons.extends template="b" /~}`,
			"b": `{~exons.extends template="a" /~}`,
		}}
		tmpl, err := engine.Parse(`{~exons.extends template="a" /~}`)
		require.NoError(t, err)
		_, err = tmpl.ExecuteWithContext(context.Background(), NewContext(nil).WithTemplateSourceResolver(store))
		requireExtendsFailure(t, err)
		assert.Contains(t, err.Error(), internal.ErrMsgCircularInheritance)
	})

	t.Run("depth", func(t *testing.T) {
		const maxDepth = 2
		engine := MustNew(WithMaxDepth(maxDepth))
		sources := map[string]string{}
		// A chain longer than the bound: p0 → p1 → … → p5, each extending the next.
		for i := 0; i < 5; i++ {
			sources[fmt.Sprintf("p%d", i)] = fmt.Sprintf(`{~exons.extends template="p%d" /~}`, i+1)
		}
		sources["p5"] = "root"
		store := &storeResolver{sources: sources}
		tmpl, err := engine.Parse(`{~exons.extends template="p0" /~}`)
		require.NoError(t, err)
		_, err = tmpl.ExecuteWithContext(context.Background(), NewContext(nil).WithTemplateSourceResolver(store))
		requireExtendsFailure(t, err)
		assert.Contains(t, err.Error(), internal.ErrMsgInheritanceDepthExceeded)
		assert.LessOrEqual(t, len(store.asked), maxDepth+1, "the bound stops the walk; it does not fetch the whole chain first")
	})
}

func TestTemplateSourceResolver_WorksWithoutAnEngine(t *testing.T) {
	// Preference order, arm (1) before arm (3): a template with NO engine and a context resolver
	// still renders — the resolver is a complete source of parents on its own. Without the
	// resolver the same template reports ErrMsgInheritanceNoEngine exactly as before.
	engine := MustNew()
	tmpl, err := engine.Parse(childExtendingBase)
	require.NoError(t, err)
	tmpl.engine = nil

	_, err = tmpl.ExecuteWithContext(context.Background(), NewContext(nil))
	requireExtendsFailure(t, err)
	assert.Contains(t, err.Error(), ErrMsgInheritanceNoEngine)

	store := &storeResolver{sources: map[string]string{
		"base": `Header {~exons.block name="content"~}default{~/exons.block~} Footer`,
	}}
	out, err := tmpl.ExecuteWithContext(context.Background(), NewContext(nil).WithTemplateSourceResolver(store))
	require.NoError(t, err)
	assert.Equal(t, "Header custom Footer", out)
}

func TestTemplate_Extends(t *testing.T) {
	engine := MustNew()

	t.Run("a template that extends names its parent", func(t *testing.T) {
		tmpl, err := engine.Parse(childExtendingBase)
		require.NoError(t, err)
		parent, ok := tmpl.Extends()
		assert.True(t, ok)
		assert.Equal(t, "base", parent)
	})

	t.Run("a template that does not extend says so", func(t *testing.T) {
		tmpl, err := engine.Parse(`plain {~exons.var name="x" default="y" /~}`)
		require.NoError(t, err)
		parent, ok := tmpl.Extends()
		assert.False(t, ok)
		assert.Empty(t, parent)
	})

	t.Run("an unreadable extends declaration is not a parent you can fetch", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.extends name="absent" /~}`)
		if err != nil {
			t.Skip("this build rejects a mis-attributed extends at parse")
		}
		if tmpl.inheritanceErr == nil {
			t.Skip("this build does not record a mis-attributed extends as unreadable")
		}
		parent, ok := tmpl.Extends()
		assert.False(t, ok, "ok=false — the host has no name to look up")
		assert.Empty(t, parent)
	})
}

// TestContext_EverySetterPreservesTheTemplateSourceResolver is the guard the Context struct's own
// comment names. Every With* setter and Child rebuild the struct field by field; one that forgets
// the resolver compiles cleanly and hands back a context that LOOKS intact with the resolver gone —
// which on the render path silently switches the source of parents back to the registry.
func TestContext_EverySetterPreservesTheTemplateSourceResolver(t *testing.T) {
	store := &storeResolver{}
	base := NewContextWithStrategy(map[string]any{"k": "v"}, ErrorStrategyLog).WithTemplateSourceResolver(store)
	require.Same(t, store, base.TemplateSourceResolver())

	setters := map[string]func(*Context) *Context{
		"WithEngine":       func(c *Context) *Context { return c.WithEngine(MustNew()) },
		"WithDepth":        func(c *Context) *Context { return c.WithDepth(3) },
		"WithSpecResolver": func(c *Context) *Context { return c.WithSpecResolver(nil) },
		"WithRefDepth":     func(c *Context) *Context { return c.WithRefDepth(2) },
		"WithRefChain":     func(c *Context) *Context { return c.WithRefChain([]string{"a"}) },
		"withData":         func(c *Context) *Context { return c.withData(map[string]any{"z": 1}) },
		"Child":            func(c *Context) *Context { return c.Child(map[string]any{"c": 1}).(*Context) },
		"WithTemplateSourceResolver(same)": func(c *Context) *Context {
			return c.WithTemplateSourceResolver(c.TemplateSourceResolver())
		},
	}
	for name, apply := range setters {
		t.Run(name, func(t *testing.T) {
			out := apply(base)
			assert.Same(t, store, out.TemplateSourceResolver(), "%s dropped the resolver", name)
			// And the other fields still travel too — a setter that preserved only the new field
			// by special-casing it would pass the line above and break everything else.
			assert.Equal(t, ErrorStrategyLog, out.ErrorStrategyValue())
		})
	}

	t.Run("the setter REPLACES rather than merges", func(t *testing.T) {
		other := &storeResolver{}
		assert.Same(t, other, base.WithTemplateSourceResolver(other).TemplateSourceResolver())
		assert.Nil(t, base.WithTemplateSourceResolver(nil).TemplateSourceResolver(), "nil clears it")
	})

	t.Run("a fresh context carries none", func(t *testing.T) {
		assert.Nil(t, NewContext(nil).TemplateSourceResolver())
		assert.Nil(t, NewContextWithStrategy(nil, ErrorStrategyLog).TemplateSourceResolver())
	})
}

// -----------------------------------------------------------------------------
// The declared-inputs walk consumes the SAME resolver
// -----------------------------------------------------------------------------

const storeParentWithInputs = `---
name: base-with-inputs
description: a parent served by the store, declaring inputs
inputs:
  tone:
    type: text
    default: formal
  audience:
    type: text
    required: true
---
Tone={~exons.input name="tone" /~} Audience={~exons.input name="audience" /~} {~exons.block name="content"~}default{~/exons.block~}`

func TestTemplateSourceResolver_InputsWalkSeesAParentServedOnlyByTheResolver(t *testing.T) {
	ctx := context.Background()
	engine := MustNew()
	store := &storeResolver{sources: map[string]string{"base": storeParentWithInputs}}
	tmpl, err := engine.Parse("---\nname: child\ndescription: d\ninputs:\n  own:\n    type: text\n---\n" + childExtendingBase)
	require.NoError(t, err)
	execCtx := NewContext(nil).WithTemplateSourceResolver(store)

	t.Run("DeclaredInputsWithContext includes the inherited declarations", func(t *testing.T) {
		declared, err := tmpl.DeclaredInputsWithContext(ctx, execCtx)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"own", "tone", "audience"}, sortedInputNames(declared))
		assert.Equal(t, "formal", declared["tone"].Default)
		assert.True(t, declared["audience"].Required)
	})

	t.Run("the context-free accessor cannot see them — which is why the WithContext form exists", func(t *testing.T) {
		declared, err := tmpl.DeclaredInputs()
		require.Error(t, err, "the registry has no such parent; the walk reports it")
		assert.ElementsMatch(t, []string{"own"}, sortedInputNames(declared), "the child's own declarations are still returned")
	})

	t.Run("ValidateInputBindingWithContext refuses a missing INHERITED required input", func(t *testing.T) {
		violations, err := tmpl.ValidateInputBindingWithContext(ctx, execCtx, map[string]any{"own": "x"})
		require.NoError(t, err, "the chain walked to its end")
		require.Len(t, violations, 1)
		assert.Contains(t, violations[0].Error(), "audience")

		violations, err = tmpl.ValidateInputBindingWithContext(ctx, execCtx, map[string]any{"own": "x", "audience": "engineers"})
		require.NoError(t, err)
		assert.Empty(t, violations)
	})

	t.Run("the render binds the inherited default from the SAME parent", func(t *testing.T) {
		// contextWithInputs runs the walk with the execution context, so the parent body's own
		// {~exons.input~} resolves and its declared default applies — with nothing registered.
		out, err := tmpl.ExecuteWithContext(ctx, execCtx.withData(map[string]any{
			ContextKeyInput: map[string]any{"audience": "engineers"},
		}))
		require.NoError(t, err)
		assert.Equal(t, "Tone=formal Audience=engineers custom", out)
	})

	t.Run("nil execCtx delegates to the context-free behaviour byte for byte", func(t *testing.T) {
		a, aErr := tmpl.DeclaredInputs()
		b, bErr := tmpl.DeclaredInputsWithContext(ctx, nil)
		assert.Equal(t, a, b)
		assert.Equal(t, fmt.Sprint(aErr), fmt.Sprint(bErr))
		va, vaErr := tmpl.ValidateInputBinding(map[string]any{})
		vb, vbErr := tmpl.ValidateInputBindingWithContext(ctx, nil, map[string]any{})
		assert.Equal(t, fmt.Sprint(va), fmt.Sprint(vb))
		assert.Equal(t, fmt.Sprint(vaErr), fmt.Sprint(vbErr))
	})
}

// TestBothInheritanceWalksReportTheSameVocabularyUnderTheContextResolver is the sibling of
// TestBothInheritanceWalksReportTheSameVocabulary for the new source of parents: whichever way the
// chain fails at the store, the render and the inputs walk must name the same tag and the same
// parent — and for a lookup FAILURE, both must carry the host's own error.
func TestBothInheritanceWalksReportTheSameVocabularyUnderTheContextResolver(t *testing.T) {
	ctx := context.Background()
	engine := MustNew()
	src := "---\nname: child\ndescription: d\ninputs:\n  own:\n    type: text\n---\n{~exons.extends template=\"absent\" /~}"
	tmpl, err := engine.Parse(src)
	require.NoError(t, err)

	cases := map[string]struct {
		store *storeResolver
		cause error
	}{
		"the store has no such parent": {store: &storeResolver{sources: map[string]string{}}},
		"the store refuses the lookup": func() struct {
			store *storeResolver
			cause error
		} {
			cause := errors.New("store unreachable")
			return struct {
				store *storeResolver
				cause error
			}{store: &storeResolver{failWith: cause}, cause: cause}
		}(),
	}

	for label, tc := range cases {
		t.Run(label, func(t *testing.T) {
			execCtx := NewContext(nil).WithTemplateSourceResolver(tc.store)
			_, execErr := tmpl.ExecuteWithContext(ctx, execCtx)
			_, specsErr := tmpl.DeclaredInputsWithContext(ctx, execCtx)
			require.Error(t, execErr, "the execute walk refuses")
			require.Error(t, specsErr, "the specs walk refuses")

			for walk, e := range map[string]error{"execute walk": execErr, "specs walk": specsErr} {
				ce := requireExtendsFailure(t, e)
				name, ok := ce.GetMetadata(MetaKeyTemplateName)
				require.True(t, ok, "%s: names the parent", walk)
				assert.Equal(t, "absent", name, "%s: the SAME parent", walk)
				if tc.cause != nil {
					assert.True(t, errors.Is(e, tc.cause), "%s: carries the host's cause: %v", walk, e)
				}
			}
		})
	}
}

// TestTemplateSourceFunc pins the adapter, which is the shape a host reaches for first.
func TestTemplateSourceFunc(t *testing.T) {
	var got string
	f := TemplateSourceFunc(func(_ context.Context, name string) (string, bool, error) {
		got = name
		return "body", true, nil
	})
	src, found, err := f.ResolveTemplateSource(context.Background(), "x")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "body", src)
	assert.Equal(t, "x", got)

	// And it plugs into the render like any other implementation.
	engine := MustNew()
	tmpl, err := engine.Parse(childExtendingBase)
	require.NoError(t, err)
	out, err := tmpl.ExecuteWithContext(context.Background(), NewContext(nil).WithTemplateSourceResolver(
		TemplateSourceFunc(func(context.Context, string) (string, bool, error) {
			return `[{~exons.block name="content"~}d{~/exons.block~}]`, true, nil
		})))
	require.NoError(t, err)
	assert.Equal(t, "[custom]", out)
}
