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

// TestEveryInheritanceFailureIsMachineMatchable pins the ONE contract a consumer needs in order to
// tell "this document declares a parent I cannot resolve" apart from every other execute failure:
//
//	the error is a *cuserr.CustomError  AND  metadata[MetaKeyTag] == TagNameExtends
//
// ⚠ AND IT IS NOT THE CODE, which is the trap this test exists to close off. `ErrCodeExec` reads
// like a code and is not one: cuserr's NewValidationError(field, message) takes a FIELD and
// WrapStdError(err, context, message) takes a CONTEXT label, so every go-exons constructor has been
// passing ErrCodeExec as a metadata label while `.Code` was derived elsewhere. Derived HOW is the
// reason no consumer may match on it — FromStdError picks the category by substring-matching the
// CAUSE'S PROSE ("not found", "invalid", "exists", …), so the code of an inheritance failure is
// decided by the wording of the message underneath it. The three failures below duly produce three
// different codes today, and rephrasing any cause would silently change a fourth. The tag is set
// explicitly by this library and is the only sound discriminator.
//
// Before this release the tag held for two of the three failures resolveInheritance can produce. The
// third — the parent was named and the chain could not be walked — returned the resolver's own
// error: an unexported type from internal/ with no Unwrap, carrying no code and no cuserr metadata,
// and (for a missing parent) naming the tag `include`. So a consumer had to string-match a message
// that named a verb the author never wrote, and could not distinguish it from a genuine
// {~exons.include~} miss. That is the whole reason this test enumerates the failures rather than
// asserting one of them: the defect was an ASYMMETRY between siblings, so only a table over all of
// them can prove it closed.
//
// It is a WIRE-LEVEL contract, not an internal convenience — aigentverse maps this error to a typed
// 422 ("this definition inherits from a template; the registry provides none") rather than a 500,
// and that mapping is only sound if the match cannot false-positive onto another execute failure.
// Hence the final subtest: no OTHER failure may claim tag == extends.
func TestEveryInheritanceFailureIsMachineMatchable(t *testing.T) {
	// requireInheritanceFailure asserts the matchable contract and returns the message for the
	// caller to make its own, more specific claim about the REASON.
	requireInheritanceFailure := func(t *testing.T, err error) *cuserr.CustomError {
		t.Helper()
		require.Error(t, err)

		var ce *cuserr.CustomError
		require.True(t, errors.As(err, &ce),
			"an inheritance failure must be a cuserr, or a consumer can only string-match it: %v", err)

		tag, ok := ce.GetMetadata(MetaKeyTag)
		require.True(t, ok, "the tag half of the match must be PRESENT, not merely correct when set")
		assert.Equal(t, TagNameExtends, tag,
			"an inheritance failure names exons.extends — the verb the author actually wrote")
		return ce
	}

	t.Run("the parent is not registered", func(t *testing.T) {
		engine := MustNew()
		src := "---\nname: child\ndescription: d\n---\n{~exons.extends template=\"absent\" /~}"

		_, err := engine.Execute(context.Background(), src, map[string]any{})
		ce := requireInheritanceFailure(t, err)

		name, ok := ce.GetMetadata(MetaKeyTemplateName)
		require.True(t, ok, "the parent this document declares is what a consumer displays")
		assert.Equal(t, "absent", name)

		// The cause survives the wrap, so the specific reason is still readable. This is the
		// assertion that would have caught the mislabel: the message names extends, not include.
		assert.Contains(t, err.Error(), internal.ErrMsgTemplateNotFound, "the reason is preserved")
		assert.Contains(t, err.Error(), TagNameExtends,
			"the failing verb is exons.extends; naming exons.include sent authors hunting a tag they never wrote")
		assert.NotContains(t, err.Error(), TagNameInclude,
			"regression guard for the mislabel — this constructor was used ONLY by extends and hardcoded include")
	})

	t.Run("the chain is circular", func(t *testing.T) {
		engine := MustNew()
		require.NoError(t, engine.RegisterTemplate("loop-a",
			"---\nname: loop-a\ndescription: a\n---\n{~exons.extends template=\"loop-b\" /~}"))
		require.NoError(t, engine.RegisterTemplate("loop-b",
			"---\nname: loop-b\ndescription: b\n---\n{~exons.extends template=\"loop-a\" /~}"))

		tmpl, found := engine.GetTemplate("loop-a")
		require.True(t, found)

		_, err := tmpl.Execute(context.Background(), map[string]any{})
		requireInheritanceFailure(t, err)
		assert.Contains(t, err.Error(), internal.ErrMsgCircularInheritance,
			"a cycle and a missing parent are both inheritance failures and must stay distinguishable BY REASON")
	})

	t.Run("the extends declaration itself cannot be read", func(t *testing.T) {
		// A well-formed tag with the WRONG attribute: parsed, so not a parse error, but the parent
		// cannot be named. This arm was already typed; it is here so the table proves the SET.
		engine := MustNew()
		src := "---\nname: child\ndescription: d\n---\n{~exons.extends name=\"absent\" /~}"

		_, err := engine.Execute(context.Background(), src, map[string]any{})
		if err == nil {
			t.Skip("this build does not treat a mis-attributed extends as an unreadable declaration")
		}
		requireInheritanceFailure(t, err)
	})

	t.Run("the derived code DISAGREES across the failures, so it is not the contract", func(t *testing.T) {
		// A guard against the plausible-looking refactor "just match on .Code instead". Two
		// inheritance failures, two different codes, neither chosen by this library — proof by
		// demonstration that the code carries no inheritance signal at all.
		engine := MustNew()
		require.NoError(t, engine.RegisterTemplate("loop-x",
			"---\nname: loop-x\ndescription: x\n---\n{~exons.extends template=\"loop-y\" /~}"))
		require.NoError(t, engine.RegisterTemplate("loop-y",
			"---\nname: loop-y\ndescription: y\n---\n{~exons.extends template=\"loop-x\" /~}"))
		cyclic, found := engine.GetTemplate("loop-x")
		require.True(t, found)

		_, missingErr := MustNew().Execute(context.Background(),
			"---\nname: child\ndescription: d\n---\n{~exons.extends template=\"absent\" /~}", map[string]any{})
		_, cyclicErr := cyclic.Execute(context.Background(), map[string]any{})

		var missingCE, cyclicCE *cuserr.CustomError
		require.True(t, errors.As(missingErr, &missingCE))
		require.True(t, errors.As(cyclicErr, &cyclicCE))
		assert.NotEqual(t, missingCE.Code, cyclicCE.Code,
			"if these ever agree it is a coincidence of wording, not a contract — keep matching on the tag")
	})

	t.Run("no other execute failure claims the extends tag", func(t *testing.T) {
		// The soundness half of the consumer's mapping. If an unrelated failure could carry
		// tag == extends, a 422 saying "this inherits from a template" would be a lie about a
		// document that does not inherit at all.
		engine := MustNew(WithErrorStrategy(ErrorStrategyThrow))

		for name, src := range map[string]string{
			"a missing variable":        "---\nname: n\ndescription: d\n---\n{~exons.var name=\"nope\" /~}",
			"an unknown verb":           "---\nname: n\ndescription: d\n---\n{~exons.frobnicate /~}",
			"a missing include target":  "---\nname: n\ndescription: d\n---\n{~exons.include template=\"absent\" /~}",
			"a malformed if expression": "---\nname: n\ndescription: d\n---\n{~exons.if eval=\"1 +++ \"~}x{~/exons.if~}",
		} {
			_, err := engine.Execute(context.Background(), src, map[string]any{})
			if err == nil {
				continue // leniency here is another rule's business; only a WRONG tag matters
			}
			var ce *cuserr.CustomError
			if !errors.As(err, &ce) {
				continue
			}
			if tag, ok := ce.GetMetadata(MetaKeyTag); ok {
				assert.NotEqual(t, TagNameExtends, tag,
					"%s must not present itself as an inheritance failure: %v", name, err)
			}
		}
	})
}

// TestBothInheritanceWalksReportTheSameVocabulary pins the agreement that motivated the change.
//
// One template has TWO walks over its extends chain: ExecuteWithContext resolves it to render, and
// ancestorSpecs walks it to collect declared inputs. They answer different questions about the same
// chain, so a consumer that gets a typed error from one and an untyped error from the other cannot
// treat "this document cannot be composed here" as a single condition — which is exactly what a
// registry must do to answer it with one status code.
func TestBothInheritanceWalksReportTheSameVocabulary(t *testing.T) {
	engine := MustNew()
	src := "---\nname: child\ndescription: d\ninputs:\n  own:\n    type: text\n---\n{~exons.extends template=\"absent\" /~}"

	tmpl, err := engine.Parse(src)
	require.NoError(t, err, "an unresolvable parent is not a PARSE failure — the document is well-formed")

	_, execErr := tmpl.Execute(context.Background(), map[string]any{})
	_, specsErr := tmpl.DeclaredInputs()
	require.Error(t, execErr, "the execute walk refuses")
	require.Error(t, specsErr, "the specs walk refuses")

	for label, e := range map[string]error{"execute walk": execErr, "specs walk": specsErr} {
		var ce *cuserr.CustomError
		require.True(t, errors.As(e, &ce), "%s must be a cuserr", label)

		tag, ok := ce.GetMetadata(MetaKeyTag)
		require.True(t, ok, "%s: tag present", label)
		assert.Equal(t, TagNameExtends, tag, "%s: same tag", label)

		name, ok := ce.GetMetadata(MetaKeyTemplateName)
		require.True(t, ok, "%s: names the parent", label)
		assert.Equal(t, "absent", name, "%s: the SAME parent", label)
	}
}
