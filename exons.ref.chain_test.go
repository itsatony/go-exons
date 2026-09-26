package exons

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// vAudience/atlas#696 — {~exons.ref~} resolves the whole chain
//
// Until v0.34.0 a referenced body was spliced as TEXT. Every test below fails against that
// build, which is the only reason any of them is worth having: the depth limit and the
// circular-chain check existed for four years and could not fire, because nothing pushed a
// frame for them to read.
// =============================================================================

// refEngine builds an engine with a map resolver holding bodies with no declared inputs.
func refEngine(t *testing.T, bodies map[string]string, opts ...Option) *Engine {
	t.Helper()
	engine, err := New(opts...)
	require.NoError(t, err)
	resolver := NewMapSpecResolver()
	for slug, body := range bodies {
		resolver.Add(slug, &Spec{Name: slug}, body)
	}
	engine.SetSpecResolver(resolver)
	return engine
}

func TestRefChain_NestedReferencesResolve(t *testing.T) {
	ctx := context.Background()

	t.Run("a three-deep chain renders through", func(t *testing.T) {
		engine := refEngine(t, map[string]string{
			"a": `A[{~exons.ref slug="b" /~}]`,
			"b": `B[{~exons.ref slug="c" /~}]`,
			"c": `C`,
		})

		result, err := engine.Execute(ctx, `{~exons.ref slug="a" /~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, "A[B[C]]", result)
	})

	t.Run("a referenced body's own tags execute", func(t *testing.T) {
		// The reported symptom: {~exons.now~} inside a fragment reached the reader as a literal
		// tag. Pinned on the TAG's absence rather than on a formatted instant, because the
		// instant is the clock's business and the literal tag is the defect.
		engine := refEngine(t, map[string]string{
			"stamp": `at {~exons.now /~}`,
		})

		result, err := engine.Execute(ctx, `{~exons.ref slug="stamp" /~}`, nil)
		require.NoError(t, err)
		assert.NotContains(t, result, "exons.now")
		assert.True(t, strings.HasPrefix(result, "at "), "got %q", result)
	})

	t.Run("a referenced body reads the CALLER's data", func(t *testing.T) {
		engine := refEngine(t, map[string]string{
			"greet": `Hallo {~exons.var name="who" /~}`,
		})

		result, err := engine.Execute(ctx,
			`{~exons.ref slug="greet" /~}`,
			map[string]any{"who": "Toni"})
		require.NoError(t, err)
		assert.Equal(t, "Hallo Toni", result)
	})

	t.Run("exons.include inside a referenced body works", func(t *testing.T) {
		engine := refEngine(t, map[string]string{
			"outer": `outer:{~exons.include template="frag" /~}`,
		})
		require.NoError(t, engine.RegisterTemplate("frag", "FRAG"))

		result, err := engine.Execute(ctx, `{~exons.ref slug="outer" /~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, "outer:FRAG", result)
	})
}

func TestRefChain_ReferencedInputDefaults(t *testing.T) {
	ctx := context.Background()

	newEngine := func(t *testing.T) *Engine {
		t.Helper()
		engine := MustNew()
		resolver := NewMapSpecResolver()
		resolver.Add("tone", &Spec{
			Name:   "tone",
			Inputs: map[string]*InputDef{"style": {Default: "förmlich"}},
		}, `Ton: {~exons.var name="input.style" /~}`)
		engine.SetSpecResolver(resolver)
		return engine
	}

	t.Run("the referenced document's default fills an unbound input", func(t *testing.T) {
		result, err := newEngine(t).Execute(ctx, `{~exons.ref slug="tone" /~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, "Ton: förmlich", result)
	})

	t.Run("the CALLER's binding wins over the reference's default", func(t *testing.T) {
		// The composing document is the authority; the composed one supplies the fallback —
		// the same direction as template inheritance, and deliberately the same function.
		result, err := newEngine(t).Execute(ctx,
			`{~exons.ref slug="tone" /~}`,
			map[string]any{ContextKeyInput: map[string]any{"style": "locker"}})
		require.NoError(t, err)
		assert.Equal(t, "Ton: locker", result)
	})

	t.Run("a reference's defaults do not leak back to the caller", func(t *testing.T) {
		// The frame is a CHILD context. A default that survived the return would make a document
		// mean different things depending on what it happened to reference earlier.
		result, err := newEngine(t).Execute(ctx,
			`{~exons.ref slug="tone" /~}|after={~exons.var name="input.style" default="—" /~}`,
			nil)
		require.NoError(t, err)
		assert.Equal(t, "Ton: förmlich|after=—", result)
	})
}

func TestRefChain_CyclesAreRefused(t *testing.T) {
	ctx := context.Background()

	t.Run("a two-step cycle refuses and names the chain", func(t *testing.T) {
		engine := refEngine(t, map[string]string{
			"a": `A{~exons.ref slug="b" /~}`,
			"b": `B{~exons.ref slug="a" /~}`,
		})

		_, err := engine.Execute(ctx, `{~exons.ref slug="a" /~}`, nil)
		require.Error(t, err)
		msg := err.Error()
		assert.Contains(t, msg, "circular")
		// The chain is the whole value of the message: "a cycle exists" sends an author looking
		// at every reference in the document; "a -> b -> a" sends them to one edge.
		assert.Contains(t, msg, "a"+RefChainSeparator+"b")
	})

	t.Run("a self-reference refuses", func(t *testing.T) {
		engine := refEngine(t, map[string]string{
			"loop": `L{~exons.ref slug="loop" /~}`,
		})

		_, err := engine.Execute(ctx, `{~exons.ref slug="loop" /~}`, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "circular")
	})

	t.Run("the same slug TWICE side by side is not a cycle", func(t *testing.T) {
		// A diamond is legal. Only an ancestor repeating is a cycle, and conflating the two
		// would refuse the most ordinary reason to have fragments at all.
		engine := refEngine(t, map[string]string{
			"leaf": `L`,
			"top":  `{~exons.ref slug="leaf" /~}+{~exons.ref slug="leaf" /~}`,
		})

		result, err := engine.Execute(ctx, `{~exons.ref slug="top" /~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, "L+L", result)
	})
}

func TestRefChain_DepthLimitApplies(t *testing.T) {
	ctx := context.Background()

	// A distinct slug per level, so the chain never repeats and ONLY the depth limit can stop it.
	// An unbounded chain of distinct slugs is what the circular check structurally cannot catch.
	bodies := make(map[string]string, 40)
	for i := 0; i < 40; i++ {
		bodies[fmt.Sprintf("s%d", i)] = fmt.Sprintf(`.{~exons.ref slug="s%d" /~}`, i+1)
	}
	bodies["s40"] = "END"

	engine := refEngine(t, bodies)
	_, err := engine.Execute(ctx, `{~exons.ref slug="s0" /~}`, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "depth")
}

func TestRefChain_VerbatimOptionKeepsTheOldSplice(t *testing.T) {
	ctx := context.Background()

	// The escape hatch for a resolver that returns ALREADY-RENDERED text. It is pinned because
	// its whole purpose is to be byte-identical to the pre-v0.34.0 behaviour — a guard that only
	// checked "it still renders" would pass against the option doing nothing.
	engine := refEngine(t, map[string]string{
		"frag": `literal {~exons.var name="who" /~}`,
	}, WithRefVerbatim())

	result, err := engine.Execute(ctx, `{~exons.ref slug="frag" /~}`, map[string]any{"who": "x"})
	require.NoError(t, err)
	assert.Equal(t, `literal {~exons.var name="who" /~}`, result)
}

func TestRefChain_LookupAndRenderFailuresAreDifferentMessages(t *testing.T) {
	ctx := context.Background()

	t.Run("an unknown slug is NOT FOUND", func(t *testing.T) {
		engine := refEngine(t, map[string]string{"known": "K"})
		_, err := engine.Execute(ctx, `{~exons.ref slug="missing" /~}`, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
		assert.NotContains(t, err.Error(), "could not be rendered")
	})

	t.Run("a broken referenced body is a RENDER failure", func(t *testing.T) {
		// ⛔ One code for two conditions is this stack's most-repeated defect. "Not found" sends
		// an author to check the slug; the slug here is perfect and the body is wrong.
		engine := refEngine(t, map[string]string{
			"broken": `{~exons.nosuchtag /~}`,
		})
		_, err := engine.Execute(ctx, `{~exons.ref slug="broken" /~}`, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not be rendered")
		assert.NotContains(t, err.Error(), "not found")
	})
}

func TestRefChain_BodyOpeningWithAMarkdownRuleIsContent(t *testing.T) {
	ctx := context.Background()

	// ⛔ Engine.Parse offers a leading `---` to a YAML scanner. A referenced body has already had
	// its frontmatter taken off by whoever produced it, so a `---` left in it is a horizontal
	// rule — content. Rendering refs through Parse would turn a fragment that works today into a
	// config-block error, which is a defect the reporter would have found before any test did.
	engine := refEngine(t, map[string]string{
		"rule": "---\n\nAbschnitt {~exons.var name=\"n\" /~}",
	})

	result, err := engine.Execute(ctx, `{~exons.ref slug="rule" /~}`, map[string]any{"n": "2"})
	require.NoError(t, err)
	assert.Equal(t, "---\n\nAbschnitt 2", result)
}

func TestRefChain_NestedFailureIsGovernedByOnerror(t *testing.T) {
	ctx := context.Background()

	// A refusal raised INSIDE a referenced body still travels the caller's error funnel. If it
	// did not, the fix would have made every reference an unconditional stop — a new fatal path
	// on documents that had chosen how to degrade.
	engine := refEngine(t, map[string]string{
		"a": `A{~exons.ref slug="b" /~}`,
		"b": `B{~exons.ref slug="a" /~}`,
	})

	result, err := engine.Execute(ctx,
		`ok:{~exons.ref slug="a" onerror="default" default="[kein Fragment]" /~}`, nil)
	require.NoError(t, err)
	assert.Equal(t, "ok:[kein Fragment]", result)
}

func TestRefChain_AHandBuiltAdapterRendersToo(t *testing.T) {
	ctx := context.Background()

	// ⛔ This is the wiring test, and it is the one most likely to have been missed.
	//
	// vaichat2 does not let Engine.SetSpecResolver install the adapter: it builds its own with
	// NewSpecResolverAdapter and injects it per execution (vchat.resolver.spec.go:150, :340), as
	// does aigentflow (aigentflow.plugin.exons.executor.go:128-129). An adapter that learned its
	// engine at CONSTRUCTION would have been engine-less on exactly that path, spliced verbatim,
	// and shipped this whole release inert for two consumers — with every other test still green.
	resolver := NewMapSpecResolver()
	resolver.Add("inner", &Spec{Name: "inner"}, "INNER")
	resolver.Add("outer", &Spec{Name: "outer"}, `O[{~exons.ref slug="inner" /~}]`)

	engine := MustNew()
	tmpl, err := engine.Parse(`{~exons.ref slug="outer" /~}`)
	require.NoError(t, err)

	// Note: SetSpecResolver is NEVER called. The resolver reaches the render only through a
	// hand-built adapter on the context, which is the shape under test.
	execCtx := NewContext(nil).WithSpecResolver(NewSpecResolverAdapter(resolver))

	result, err := tmpl.ExecuteWithContext(ctx, execCtx)
	require.NoError(t, err)
	assert.Equal(t, "O[INNER]", result)
}

// =============================================================================
// Findings from the 0.34.0 review — each reproduced against the first cut of this
// release, each fixed here. They are grouped because they share one shape: render
// state that was re-derived from the ENGINE instead of carried from the CALLER.
// =============================================================================

func TestRefChain_ANestedMissIsNotReportedAsTheOuterSlugMissing(t *testing.T) {
	ctx := context.Background()

	// ⛔ The first cut wrapped a lookup sentinel into the error, and BuiltinError/ExecutorError
	// both Unwrap to their cause — so an INNER miss stayed reachable by errors.Is from every
	// outer frame. A four-deep chain reported "not found" four times, each naming a slug that
	// resolves perfectly, which is the exact defect the message split exists to prevent.
	engine := refEngine(t, map[string]string{
		"a": `A{~exons.ref slug="missing" /~}`,
	})

	_, err := engine.Execute(ctx, `{~exons.ref slug="a" /~}`, nil)
	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "could not be rendered", "the OUTER frame resolved fine; it failed to render")
	assert.Contains(t, msg, "spec_slug=a")
	// The inner frame keeps the honest "not found" — the split is per level, not per chain.
	assert.Contains(t, msg, "not found")
	assert.Contains(t, msg, "spec_slug=missing")
}

func TestRefChain_IncludeDoesNotResetTheFrame(t *testing.T) {
	ctx := context.Background()

	t.Run("a cycle THROUGH an include is still a cycle", func(t *testing.T) {
		// Engine.ExecuteTemplate built a fresh context, putting refDepth back to 0 and the chain
		// back to empty on every hop. The recursion then ran until the INCLUDE depth ran out and
		// reported a depth message about the wrong construct — RefMaxDepth bypassed entirely.
		engine := refEngine(t, map[string]string{
			"a": `A{~exons.include template="t" /~}`,
		})
		require.NoError(t, engine.RegisterTemplate("t", `T{~exons.ref slug="a" /~}`))

		_, err := engine.Execute(ctx, `{~exons.ref slug="a" /~}`, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "circular")
		assert.NotContains(t, err.Error(), "maximum template inclusion depth")
	})

	t.Run("a context-supplied resolver survives an include", func(t *testing.T) {
		// ExecuteTemplate re-read the ENGINE's adapter, which is nil unless SetSpecResolver was
		// called — and vaichat2 and aigentflow deliberately do not call it. An included template
		// that referenced a spec failed with "spec resolver not available" on exactly their path.
		resolver := NewMapSpecResolver()
		resolver.Add("a", &Spec{Name: "a"}, "A")

		engine := MustNew() // ⛔ SetSpecResolver is never called.
		require.NoError(t, engine.RegisterTemplate("t", `T{~exons.ref slug="a" /~}`))
		tmpl, err := engine.Parse(`{~exons.include template="t" /~}`)
		require.NoError(t, err)

		out, err := tmpl.ExecuteWithContext(ctx,
			NewContext(nil).WithSpecResolver(NewSpecResolverAdapter(resolver)))
		require.NoError(t, err)
		assert.Equal(t, "TA", out)
	})
}

func TestRefChain_TemplateEntryPointsResolveRefsToo(t *testing.T) {
	ctx := context.Background()

	// Only Engine.Execute injected the adapter, so Template.Execute and — the one that matters —
	// ExecuteAndExtractMessages could not resolve a single reference, while the release notes
	// advertised chains. ⭐ A capability present on one entry point and absent from its sibling is
	// the same defect as one wired nowhere; it is merely harder to notice.
	engine := MustNew()
	resolver := NewMapSpecResolver()
	resolver.Add("m", &Spec{Name: "m"}, "FRAGMENT")
	engine.SetSpecResolver(resolver)

	t.Run("Template.Execute", func(t *testing.T) {
		tmpl, err := engine.Parse(`x{~exons.ref slug="m" /~}`)
		require.NoError(t, err)
		out, err := tmpl.Execute(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, "xFRAGMENT", out)
	})

	t.Run("ExecuteAndExtractMessages", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="system"~}{~exons.ref slug="m" /~}{~/exons.message~}`)
		require.NoError(t, err)
		msgs, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.Equal(t, "FRAGMENT", msgs[0].Content)
	})
}

func TestRefChain_MessageMarkersAreValidOnlyAtTopLevel(t *testing.T) {
	ctx := context.Background()

	engine := refEngine(t, map[string]string{
		"m": `{~exons.message role="system"~}hallo{~/exons.message~}`,
	})

	t.Run("a referenced body's messages work at top level", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.ref slug="m" /~}`)
		require.NoError(t, err)
		msgs, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.Equal(t, "system", msgs[0].Role)
		assert.Equal(t, "hallo", msgs[0].Content)
	})

	t.Run("NAMED RESIDUAL: nested inside a message, the inner markers flatten", func(t *testing.T) {
		// ⚠ Pinned as a STATED limitation, not as desired behaviour. The message tag strips NUL
		// from its children to stop marker injection, so an inner message's delimiters go and its
		// marker TEXT stays — the caller gets one message whose content carries `MSG_START:`.
		//
		// The nested-message behaviour predates this release; what v0.34.0 changes is that a
		// fragment author can now reach it without seeing it, because the body is executed. It is
		// recorded here rather than left to be discovered, and tracked as go-exons#5.
		tmpl, err := engine.Parse(
			`{~exons.message role="user"~}{~exons.ref slug="m" /~}{~/exons.message~}`)
		require.NoError(t, err)
		msgs, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.Equal(t, "user", msgs[0].Role)
		assert.Contains(t, msgs[0].Content, "MSG_START")
	})
}
