package exons

import (
	"context"
	"errors"
	"testing"

	"github.com/itsatony/go-cuserr"
	"github.com/itsatony/go-exons/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// DC13-entail · Workstream A — lineage
//
// The interaction the ~100 existing inheritance tests never covered: inheritance
// meeting frontmatter. Every one of those tests gives its parent a bare body, so
// two whole classes of defect sat in the open.
//
// Defect 2 — the resolver lexed the parent's RAW source, and the lexer has no
// frontmatter awareness (stripping lives only in ExtractYAMLFrontmatter, called
// from Engine.Parse). A parent carrying frontmatter therefore spliced its own
// ---…--- into the child's output as literal text.
//
// Defect 1 — a *Template owns exactly one *Spec, its own, so a parent's inputs:
// were invisible to the child that extends it. The parent's body then resolved
// against the CHILD's declarations and {~exons.input~} failed outright.
// =============================================================================

// -----------------------------------------------------------------------------
// Defect 2 — the parent's frontmatter must not reach the output
// -----------------------------------------------------------------------------

func TestInheritance_ParentFrontmatterIsNotSplicedIntoOutput(t *testing.T) {
	ctx := context.Background()

	t.Run("parent with frontmatter emits no delimiters", func(t *testing.T) {
		engine := MustNew()
		engine.MustRegisterTemplate("base", "---\nname: base-layout\ndescription: a parent that declares things\n---\nHeader {~exons.block name=\"content\"~}default{~/exons.block~} Footer")

		result, err := engine.Execute(ctx, `{~exons.extends template="base" /~}{~exons.block name="content"~}custom{~/exons.block~}`, nil)
		require.NoError(t, err)

		assert.NotContains(t, result, "---", "the parent's frontmatter delimiters must not reach the output")
		assert.NotContains(t, result, "name: base-layout", "the parent's frontmatter body must not reach the output")
		assert.NotContains(t, result, "description:", "the parent's frontmatter body must not reach the output")

		// The body itself must still be intact.
		assert.Contains(t, result, "Header")
		assert.Contains(t, result, "custom")
		assert.Contains(t, result, "Footer")
	})

	t.Run("grandparent frontmatter is stripped too", func(t *testing.T) {
		engine := MustNew()
		engine.MustRegisterTemplate("grand", "---\nname: grand\ndescription: g\n---\nG:{~exons.block name=\"content\"~}g{~/exons.block~}")
		engine.MustRegisterTemplate("mid", "---\nname: mid\ndescription: m\n---\n{~exons.extends template=\"grand\" /~}")

		result, err := engine.Execute(ctx, `{~exons.extends template="mid" /~}{~exons.block name="content"~}leaf{~/exons.block~}`, nil)
		require.NoError(t, err)

		assert.NotContains(t, result, "---")
		assert.NotContains(t, result, "name: grand")
		assert.NotContains(t, result, "name: mid")
		assert.Contains(t, result, "G:")
		assert.Contains(t, result, "leaf")
	})

	t.Run("a parent without frontmatter is unchanged", func(t *testing.T) {
		engine := MustNew()
		engine.MustRegisterTemplate("base", `Header {~exons.block name="content"~}default{~/exons.block~} Footer`)

		result, err := engine.Execute(ctx, `{~exons.extends template="base" /~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, "Header default Footer", result)
	})

	t.Run("a body that legitimately contains --- is preserved", func(t *testing.T) {
		// The strip must be the SAME statement of what frontmatter is that Engine.Parse
		// uses, not a looser one that eats any horizontal rule in the body.
		engine := MustNew()
		engine.MustRegisterTemplate("base", "---\nname: base\ndescription: b\n---\nAbove\n\n---\n\nBelow")

		result, err := engine.Execute(ctx, `{~exons.extends template="base" /~}`, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "Above")
		assert.Contains(t, result, "Below")
		assert.Contains(t, result, "---", "a horizontal rule inside the parent's BODY is content, not frontmatter")
		assert.NotContains(t, result, "name: base")
	})
}

// -----------------------------------------------------------------------------
// Defect 1 — the declaration chain merges, composing document authoritative
// -----------------------------------------------------------------------------

const inheritedParentWithInputs = `---
name: base-with-inputs
description: a parent that declares its own inputs
inputs:
  tone:
    type: text
    default: formal
  audience:
    type: text
    default: engineers
---
Tone={~exons.input name="tone" /~} Audience={~exons.input name="audience" /~} {~exons.block name="content"~}default{~/exons.block~}`

func TestInheritance_ParentDeclaredInputsAreInScope(t *testing.T) {
	ctx := context.Background()

	t.Run("a parent body renders against the parent's own declarations", func(t *testing.T) {
		// Before this release the spliced parent body executed against the CHILD's spec, and
		// exons.input reports an undeclared name as a hard error rather than a blank — so this
		// did not merely render wrong, it did not render at all.
		engine := MustNew()
		engine.MustRegisterTemplate("base", inheritedParentWithInputs)

		result, err := engine.Execute(ctx, `{~exons.extends template="base" /~}{~exons.block name="content"~}child{~/exons.block~}`, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "Tone=formal")
		assert.Contains(t, result, "Audience=engineers")
		assert.Contains(t, result, "child")
	})

	t.Run("a caller binding overrides an inherited default", func(t *testing.T) {
		engine := MustNew()
		engine.MustRegisterTemplate("base", inheritedParentWithInputs)

		result, err := engine.Execute(ctx, `{~exons.extends template="base" /~}`, map[string]any{
			ContextKeyInput: map[string]any{"tone": "casual"},
		})
		require.NoError(t, err)
		assert.Contains(t, result, "Tone=casual")
		assert.Contains(t, result, "Audience=engineers", "an unbound inherited input still takes its declared default")
	})

	t.Run("the composing document is the authority", func(t *testing.T) {
		// The rule, stated once and mirroring aiv ADR-141: the child's declaration of the same
		// name wins, and the parent's supplies the fallback for names the child does not restate.
		engine := MustNew()
		engine.MustRegisterTemplate("base", inheritedParentWithInputs)

		child := "---\nname: child\ndescription: overrides tone\ninputs:\n  tone:\n    type: text\n    default: playful\n---\n{~exons.extends template=\"base\" /~}"
		result, err := engine.Execute(ctx, child, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "Tone=playful", "the child's declaration of tone must win")
		assert.Contains(t, result, "Audience=engineers", "a name only the parent declares still resolves")
	})

	t.Run("a grandparent's declarations reach the leaf, nearer ancestor winning", func(t *testing.T) {
		engine := MustNew()
		engine.MustRegisterTemplate("grand",
			"---\nname: grand\ndescription: g\ninputs:\n  tone:\n    type: text\n    default: grand-tone\n  origin:\n    type: text\n    default: grand-origin\n---\n"+
				"Tone={~exons.input name=\"tone\" /~} Origin={~exons.input name=\"origin\" /~}")
		engine.MustRegisterTemplate("mid",
			"---\nname: mid\ndescription: m\ninputs:\n  tone:\n    type: text\n    default: mid-tone\n---\n{~exons.extends template=\"grand\" /~}")

		result, err := engine.Execute(ctx, `{~exons.extends template="mid" /~}`, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "Tone=mid-tone", "the nearer ancestor overrides the further one")
		assert.Contains(t, result, "Origin=grand-origin")
	})

	t.Run("DryRun reports an inherited input as declared", func(t *testing.T) {
		// This is the assertion that catches a PARTIAL fix. Binding the input in the executor
		// while DryRun still answers Declared: false is one rule with two implementations.
		engine := MustNew()
		engine.MustRegisterTemplate("base", inheritedParentWithInputs)

		tmpl, err := engine.Parse(`{~exons.extends template="base" /~}`)
		require.NoError(t, err)

		dr := tmpl.DryRun(ctx, nil)
		require.NotEmpty(t, dr.Inputs, "the walk must reach the spliced parent body")
		for _, ref := range dr.Inputs {
			assert.True(t, ref.Declared, "input %q is declared by the parent and must report as declared", ref.Name)
		}
	})

	t.Run("a template declaring nothing itself still injects its parent's inputs", func(t *testing.T) {
		// The old guard was `t.spec == nil` — a child with NO frontmatter at all short-circuited
		// before the chain was ever consulted.
		engine := MustNew()
		engine.MustRegisterTemplate("base", inheritedParentWithInputs)

		tmpl, err := engine.Parse(`{~exons.extends template="base" /~}`)
		require.NoError(t, err)
		require.Nil(t, tmpl.Spec(), "the child deliberately has no frontmatter of its own")

		out, err := tmpl.Execute(ctx, nil)
		require.NoError(t, err)
		assert.Contains(t, out, "Tone=formal")
	})

	t.Run("a circular chain degrades rather than hanging", func(t *testing.T) {
		// mergedInputs applies the resolver's own circular rule. Execution still refuses the
		// chain — that refusal is where the reason belongs — but the declaration walk must
		// terminate on its own rather than rely on it.
		engine := MustNew()
		engine.MustRegisterTemplate("a", "---\nname: a\ndescription: a\ninputs:\n  x:\n    type: text\n    default: ax\n---\n{~exons.extends template=\"b\" /~}")
		engine.MustRegisterTemplate("b", "---\nname: b\ndescription: b\ninputs:\n  y:\n    type: text\n    default: by\n---\n{~exons.extends template=\"a\" /~}")

		tmpl, err := engine.Parse(`{~exons.extends template="a" /~}`)
		require.NoError(t, err)

		assert.True(t, tmpl.declaresInput("x"))
		assert.True(t, tmpl.declaresInput("y"))
		assert.False(t, tmpl.declaresInput("z"))

		_, execErr := tmpl.Execute(ctx, nil)
		assert.Error(t, execErr, "the circular chain itself is still refused at execute")
	})
}

// -----------------------------------------------------------------------------
// Site 4 — the DryRun preview honours declared defaults
// -----------------------------------------------------------------------------

func TestDryRun_PreviewAppliesDeclaredInputDefaults(t *testing.T) {
	ctx := context.Background()

	t.Run("preview matches the render for a declared default", func(t *testing.T) {
		engine := MustNew()
		src := "---\nname: p\ndescription: d\ninputs:\n  tone:\n    type: text\n    default: friendly\n---\nTone is {~exons.input name=\"tone\" /~}."
		tmpl, err := engine.Parse(src)
		require.NoError(t, err)

		rendered, err := tmpl.Execute(ctx, nil)
		require.NoError(t, err)
		require.Equal(t, "Tone is friendly.", rendered)

		dr := tmpl.DryRun(ctx, map[string]any{})
		assert.Equal(t, rendered, dr.Output, "the preview must not disagree with the render about a declared default")
	})

	t.Run("a declared input does not become an unused variable", func(t *testing.T) {
		// The reason the defaults go into the PREVIEW map and not the ANALYSIS map.
		engine := MustNew()
		src := "---\nname: p\ndescription: d\ninputs:\n  tone:\n    type: text\n    default: friendly\n---\nTone is {~exons.input name=\"tone\" /~}."
		tmpl, err := engine.Parse(src)
		require.NoError(t, err)

		dr := tmpl.DryRun(ctx, map[string]any{})
		assert.NotContains(t, dr.UnusedVariables, ContextKeyInput)
	})

	t.Run("a caller binding still wins in the preview", func(t *testing.T) {
		engine := MustNew()
		src := "---\nname: p\ndescription: d\ninputs:\n  tone:\n    type: text\n    default: friendly\n---\nTone is {~exons.input name=\"tone\" /~}."
		tmpl, err := engine.Parse(src)
		require.NoError(t, err)

		dr := tmpl.DryRun(ctx, map[string]any{ContextKeyInput: map[string]any{"tone": "brusque"}})
		assert.Equal(t, "Tone is brusque.", dr.Output)
	})

	t.Run("a document using input as an ordinary variable is left alone", func(t *testing.T) {
		// Rule 1 — the same exit contextWithInputs takes.
		engine := MustNew()
		src := "---\nname: p\ndescription: d\ninputs:\n  tone:\n    type: text\n    default: friendly\n---\nGot {~exons.var name=\"input\" /~}."
		tmpl, err := engine.Parse(src)
		require.NoError(t, err)

		dr := tmpl.DryRun(ctx, map[string]any{ContextKeyInput: "the user's question"})
		assert.Equal(t, "Got the user's question.", dr.Output)
	})
}

// -----------------------------------------------------------------------------
// The merged contract is reachable from OUTSIDE the package
//
// Template.Spec() is the parse result and reports the document's own frontmatter
// alone. Without an accessor for the merged set, a consumer building a form or
// projecting a wire contract silently omits every field an extending document
// inherits — which is how examples/09-typed-inputs found this gap.
// -----------------------------------------------------------------------------

func TestDeclaredInputs_ExposesTheMergedContract(t *testing.T) {
	engine := MustNew()
	engine.MustRegisterTemplate("base", inheritedParentWithInputs)

	child := "---\nname: child\ndescription: overrides tone, adds severity\ninputs:\n  tone:\n    type: select\n    default: playful\n  severity:\n    type: number\n    default: 3\n---\n{~exons.extends template=\"base\" /~}"
	tmpl, err := engine.Parse(child)
	require.NoError(t, err)

	t.Run("Spec reports only what the document itself declares", func(t *testing.T) {
		assert.NotContains(t, tmpl.Spec().Inputs, "audience",
			"Spec() is the parse result — widening it would report a field absent from its source")
	})

	t.Run("DeclaredInputs merges the chain, child authoritative", func(t *testing.T) {
		declared, err := tmpl.DeclaredInputs()
		require.NoError(t, err, "a chain walked to its root reports no error")
		require.Contains(t, declared, "audience", "the parent's declaration must be reachable")
		assert.Equal(t, "engineers", declared["audience"].Default)
		assert.Equal(t, "playful", declared["tone"].Default, "the child's declaration of tone wins")
		assert.Equal(t, InputTypeSelect, declared["tone"].Type)
	})

	t.Run("the returned map is independent of the template", func(t *testing.T) {
		declared, err := tmpl.DeclaredInputs()
		require.NoError(t, err)
		delete(declared, "audience")
		again, err := tmpl.DeclaredInputs()
		require.NoError(t, err)
		assert.Contains(t, again, "audience", "mutating the result must not reach the spec")
	})

	t.Run("the returned InputDefs are copies, including of a registered PARENT's spec", func(t *testing.T) {
		// The aliasing this guards against reaches through the child into a template the caller
		// never named — corrupting every future and concurrent render of the parent.
		declared, err := tmpl.DeclaredInputs()
		require.NoError(t, err)
		declared["audience"].Default = "mutated"
		declared["audience"].Type = "wrecked"

		parent, found := engine.GetTemplate("base")
		require.True(t, found)
		assert.Equal(t, "engineers", parent.Spec().Inputs["audience"].Default,
			"the parent's parsed spec must be untouched")

		again, err := tmpl.DeclaredInputs()
		require.NoError(t, err)
		assert.Equal(t, "engineers", again["audience"].Default)
	})

	t.Run("keys lead with the document's own author order", func(t *testing.T) {
		keys, err := tmpl.DeclaredInputKeys()
		require.NoError(t, err)
		assert.Equal(t, []string{"tone", "severity", "audience"}, keys,
			"own declarations in author order, then inherited names nearest-parent first")
	})

	t.Run("a template with no inheritance answers with its own set", func(t *testing.T) {
		plain, err := engine.Parse("---\nname: plain\ndescription: p\ninputs:\n  a:\n    type: text\n  b:\n    type: text\n---\nbody")
		require.NoError(t, err)
		keys, err := plain.DeclaredInputKeys()
		require.NoError(t, err)
		assert.Equal(t, []string{"a", "b"}, keys)
		declared, err := plain.DeclaredInputs()
		require.NoError(t, err)
		assert.Len(t, declared, 2)
	})

	t.Run("a template declaring nothing answers nil, not an empty map", func(t *testing.T) {
		bare, err := engine.Parse("just text")
		require.NoError(t, err)
		declared, err := bare.DeclaredInputs()
		require.NoError(t, err)
		assert.Nil(t, declared)
		keys, err := bare.DeclaredInputKeys()
		require.NoError(t, err)
		assert.Nil(t, keys)
	})

	t.Run("keys and map always agree", func(t *testing.T) {
		// The two are separate orderings of ONE traversal; a name in either and not the other
		// would mean the traversal ran twice and disagreed with itself.
		keys, err := tmpl.DeclaredInputKeys()
		require.NoError(t, err)
		declared, err := tmpl.DeclaredInputs()
		require.NoError(t, err)
		assert.Len(t, keys, len(declared))
		for _, k := range keys {
			assert.Contains(t, declared, k)
		}
	})
}

// TestDeclaredInputsReportsAnIncompleteChain pins the rule that makes the accessor honest:
//
//	DeclaredInputs errors in exactly the cases where ExecuteWithContext refuses to render.
//
// Without it the accessor is a THIRD view of one template, agreeing with neither Execute nor
// DryRun — it hands back a well-formed, complete-LOOKING contract for a document that cannot run.
// The consumer this was written for projects that contract onto a wire, where a plausible lie is
// worse than a gap.
func TestDeclaredInputsReportsAnIncompleteChain(t *testing.T) {
	newEngine := func() *Engine {
		e := MustNew()
		require.NoError(t, e.RegisterTemplate("cycle-a",
			"---\nname: cycle-a\ndescription: a\ninputs:\n  from-a:\n    type: text\n---\n{~exons.extends template=\"cycle-b\" /~}"))
		require.NoError(t, e.RegisterTemplate("cycle-b",
			"---\nname: cycle-b\ndescription: b\ninputs:\n  from-b:\n    type: text\n---\n{~exons.extends template=\"cycle-a\" /~}"))
		return e
	}

	t.Run("a circular chain errors, and Execute refuses the same document", func(t *testing.T) {
		engine := newEngine()
		tmpl, found := engine.GetTemplate("cycle-a")
		require.True(t, found)

		declared, err := tmpl.DeclaredInputs()
		require.Error(t, err, "a cycle must not be reported as a complete contract")
		assert.Contains(t, err.Error(), internal.ErrMsgCircularInheritance,
			"the same reason, in the same words, ResolveInheritance gives")
		assert.Contains(t, declared, "from-a",
			"what the walk did reach is still returned — the caller may display it, not publish it")

		// The equivalence the rule asserts: the executor refuses this exact document.
		_, execErr := tmpl.Execute(context.Background(), map[string]any{})
		require.Error(t, execErr)

		keys, keysErr := tmpl.DeclaredInputKeys()
		require.Error(t, keysErr, "both accessors report, or the two disagree")
		assert.NotEmpty(t, keys)
	})

	t.Run("a missing parent errors and names the ancestor it stopped at", func(t *testing.T) {
		engine := MustNew()
		tmpl, err := engine.Parse(
			"---\nname: orphan\ndescription: o\ninputs:\n  own:\n    type: text\n---\n{~exons.extends template=\"nowhere\" /~}")
		require.NoError(t, err)

		declared, err := tmpl.DeclaredInputs()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgTemplateNotFound)
		assert.Contains(t, declared, "own")

		// The ancestor is carried as METADATA, not spliced into the message: a caller that wants
		// to point at the broken link needs the name as a value, not as prose to re-parse.
		var custom *cuserr.CustomError
		require.True(t, errors.As(err, &custom), "the chain error must be a *cuserr.CustomError")
		stoppedAt, ok := custom.GetMetadata(MetaKeyTemplateName)
		require.True(t, ok, "the ancestor the walk stopped at must be named")
		assert.Equal(t, "nowhere", stoppedAt)

		_, execErr := tmpl.Execute(context.Background(), map[string]any{})
		require.Error(t, execErr, "Execute must refuse what DeclaredInputs called incomplete")
	})

	t.Run("an engine-less extending template errors", func(t *testing.T) {
		engine := MustNew()
		tmpl, err := engine.Parse(
			"---\nname: detached\ndescription: d\ninputs:\n  own:\n    type: text\n---\n{~exons.extends template=\"base\" /~}")
		require.NoError(t, err)
		tmpl.engine = nil

		_, err = tmpl.DeclaredInputs()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgInheritanceNoEngine)
	})

	t.Run("a non-extending template never errors even with no engine", func(t *testing.T) {
		engine := MustNew()
		tmpl, err := engine.Parse("---\nname: solo\ndescription: s\ninputs:\n  own:\n    type: text\n---\nbody")
		require.NoError(t, err)
		tmpl.engine = nil

		declared, err := tmpl.DeclaredInputs()
		require.NoError(t, err, "no chain means nothing to fail to walk")
		assert.Contains(t, declared, "own")
	})

	// The two subtests below are the ones that MATTER: each pins a boundary where the equivalence
	// was actually broken. The three above all passed while the rule was false in both directions.

	t.Run("an unreadable extends declaration errors, and does not read as a clean contract", func(t *testing.T) {
		// The fail-OPEN direction, and the dangerous one. newTemplateWithConfig nils
		// inheritanceInfo whenever extraction failed, so a guard that tests the INFO before the
		// ERROR reports "complete" — with a nil error — for a document Execute refuses outright.
		// The order of two ifs is the whole property.
		engine := MustNew()
		tmpl, err := engine.Parse(
			"---\nname: broken\ndescription: b\ninputs:\n  own:\n    type: text\n---\n{~exons.parent /~}")
		require.NoError(t, err, "extraction failure is non-fatal at PARSE — that is the trap")

		declared, err := tmpl.DeclaredInputs()
		require.Error(t, err, "a contract whose parent cannot be identified is not complete")
		assert.Contains(t, err.Error(), ErrMsgInheritanceUnreadable)
		assert.Contains(t, declared, "own")

		_, execErr := tmpl.Execute(context.Background(), map[string]any{})
		require.Error(t, execErr, "Execute refuses it, so DeclaredInputs must not bless it")
	})

	t.Run("a chain ending exactly at maxDepth is complete, not exceeded", func(t *testing.T) {
		// The fail-CLOSED direction. ResolveInheritance refuses only when a parent is DEMANDED
		// beyond the bound, so it resolves maxDepth+1 parents. A walk that bounds its own loop
		// instead reports DepthExceeded for the last chain the executor renders happily — one
		// rule, two implementations, disagreeing precisely at the boundary.
		engine := MustNew(WithMaxDepth(1))
		engine.MustRegisterTemplate("p2",
			"---\nname: p2\ndescription: root\ninputs:\n  root-input:\n    type: text\n    default: from-root\n---\nRoot={~exons.input name=\"root-input\" /~}")
		engine.MustRegisterTemplate("p1",
			"---\nname: p1\ndescription: mid\n---\n{~exons.extends template=\"p2\" /~}")

		child := "---\nname: leaf\ndescription: l\n---\n{~exons.extends template=\"p1\" /~}"
		tmpl, err := engine.Parse(child)
		require.NoError(t, err)

		// The executor's verdict is the reference: two parents at maxDepth=1 resolve.
		out, execErr := tmpl.Execute(context.Background(), map[string]any{})
		require.NoError(t, execErr, "the executor resolves maxDepth+1 parents")
		assert.Contains(t, out, "Root=from-root")

		declared, err := tmpl.DeclaredInputs()
		require.NoError(t, err, "the walk must agree with the executor at the boundary")
		assert.Contains(t, declared, "root-input", "the root ancestor's declarations are reachable")
	})
}

// -----------------------------------------------------------------------------
// Defects 3 + 4 — one resolution helper, three callers
//
// ExecuteWithContext, dryRunAST and Explain used to each decide both WHAT the
// resolution outcome was and HOW to report it, and they disagreed in three ways.
// They now share resolveInheritance and disagree only in vocabulary.
// -----------------------------------------------------------------------------

func TestInheritance_ExplainResolvesLikeExecute(t *testing.T) {
	ctx := context.Background()

	t.Run("a document explains what it renders", func(t *testing.T) {
		// Defect 3. Explain walked t.ast, so an extending template explained its own block
		// definitions rather than the spliced document — the worst failure mode for a debugging
		// tool, because the discrepancy looks like the bug being investigated.
		engine := MustNew()
		engine.MustRegisterTemplate("base", `Header {~exons.block name="content"~}default{~/exons.block~} Footer`)

		tmpl, err := engine.Parse(`{~exons.extends template="base" /~}{~exons.block name="content"~}custom{~/exons.block~}`)
		require.NoError(t, err)

		rendered, err := tmpl.Execute(ctx, nil)
		require.NoError(t, err)

		explained := tmpl.Explain(ctx, nil)
		require.NoError(t, explained.Error)
		assert.Equal(t, rendered, explained.Output, "Explain must describe the document that runs")
		assert.Contains(t, explained.AST, "Header", "the formatted AST must be the resolved one too")
	})

	t.Run("Explain surfaces a resolution failure rather than inventing output", func(t *testing.T) {
		engine := MustNew()
		tmpl, err := engine.Parse(`{~exons.extends template="does-not-exist" /~}body`)
		require.NoError(t, err)

		explained := tmpl.Explain(ctx, nil)
		require.Error(t, explained.Error, "an unresolvable parent must reach the caller")

		_, execErr := tmpl.Execute(ctx, nil)
		assert.Error(t, execErr, "and Execute must agree")
	})
}

func TestInheritance_SwallowedFailuresNowReachTheCaller(t *testing.T) {
	ctx := context.Background()

	t.Run("an unreadable extends declaration is fatal at execute", func(t *testing.T) {
		// Defect 4a. Two extends tags: ExtractInheritanceInfo fails, inheritanceInfo is nil, and
		// ExecuteWithContext used to render the bare child body and return nil — a document
		// nobody wrote, reported as success.
		engine := MustNew()
		tmpl, err := engine.Parse(`{~exons.extends template="a" /~}{~exons.extends template="b" /~}body`)
		require.NoError(t, err, "Parse stays non-fatal on purpose — the template can still be inspected")

		_, execErr := tmpl.Execute(ctx, nil)
		require.Error(t, execErr)
		assert.Contains(t, execErr.Error(), ErrMsgInheritanceUnreadable)

		dr := tmpl.DryRun(ctx, nil)
		assert.False(t, dr.Valid, "DryRun's verdict must agree with what Execute does")
		assert.False(t, dr.AnalysisComplete())
	})

	t.Run("an engine-less extends is fatal at execute", func(t *testing.T) {
		// Defect 4b, design decision A-2 — a deliberate behaviour change.
		engine := MustNew()
		engine.MustRegisterTemplate("base-a2", `parent body`)
		tmpl, err := engine.Parse(`{~exons.extends template="base-a2" /~}body`)
		require.NoError(t, err)

		// Same package, so the unexported field is reachable.
		tmpl.engine = nil

		_, execErr := tmpl.Execute(ctx, nil)
		require.Error(t, execErr)
		assert.Contains(t, execErr.Error(), ErrMsgInheritanceNoEngine)
	})

	t.Run("a template that extends nothing is untouched", func(t *testing.T) {
		// The control: none of the above may make an ordinary document fail.
		engine := MustNew()
		tmpl, err := engine.Parse(`plain {~exons.var name="x" default="d" /~}`)
		require.NoError(t, err)

		out, err := tmpl.Execute(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, "plain d", out)

		dr := tmpl.DryRun(ctx, nil)
		assert.True(t, dr.Valid)
		assert.True(t, dr.AnalysisComplete())
	})
}
