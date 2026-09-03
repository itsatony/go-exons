package exons

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// exons.input — declared inputs stop being inert
// =============================================================================

// inputDoc builds a document with the given frontmatter inputs block and body.
func inputDoc(inputsYAML, body string) string {
	return "---\nname: t\ndescription: d\ntype: prompt\ninputs:\n" + inputsYAML + "---\n" + body
}

func TestTemplate_Inputs_Injection(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("a declared default renders with nothing bound", func(t *testing.T) {
		src := inputDoc("  tone:\n    type: text\n    default: neutral\n",
			`Write in a {~exons.input name="tone" /~} voice.`)
		out, err := engine.Execute(ctx, src, nil)
		require.NoError(t, err)
		assert.Equal(t, "Write in a neutral voice.", strings.TrimSpace(out))
	})

	t.Run("a bound value beats the declared default", func(t *testing.T) {
		src := inputDoc("  tone:\n    type: text\n    default: neutral\n",
			`{~exons.input name="tone" /~}`)
		out, err := engine.Execute(ctx, src, map[string]any{
			ContextKeyInput: map[string]any{"tone": "urgent"},
		})
		require.NoError(t, err)
		assert.Equal(t, "urgent", strings.TrimSpace(out))
	})

	t.Run("a bound empty string falls back to the default", func(t *testing.T) {
		// An untouched field and a cleared field are the same gesture from a form user,
		// so the default applies to both. This mirrors how exons.env treats an empty var.
		src := inputDoc("  tone:\n    type: text\n    default: neutral\n",
			`{~exons.input name="tone" /~}`)
		out, err := engine.Execute(ctx, src, map[string]any{
			ContextKeyInput: map[string]any{"tone": ""},
		})
		require.NoError(t, err)
		assert.Equal(t, "neutral", strings.TrimSpace(out))
	})

	t.Run("a falsy default is honored, not treated as absent", func(t *testing.T) {
		src := inputDoc("  verbose:\n    type: boolean\n    default: false\n  count:\n    type: number\n    default: 0\n",
			`{~exons.input name="verbose" /~}/{~exons.input name="count" /~}`)
		out, err := engine.Execute(ctx, src, nil)
		require.NoError(t, err)
		assert.Equal(t, "false/0", strings.TrimSpace(out))
	})

	t.Run("adding an input does not unbind the caller's other values", func(t *testing.T) {
		// The merge is per key. A whole-value replace would mean that declaring a new input
		// silently discarded everything an existing caller already binds.
		src := inputDoc("  a:\n    type: text\n    default: da\n  b:\n    type: text\n",
			`{~exons.input name="a" /~}|{~exons.input name="b" /~}`)
		out, err := engine.Execute(ctx, src, map[string]any{
			ContextKeyInput: map[string]any{"b": "bound-b"},
		})
		require.NoError(t, err)
		assert.Equal(t, "da|bound-b", strings.TrimSpace(out))
	})

	t.Run("a document declaring no inputs is untouched", func(t *testing.T) {
		src := "---\nname: t\ndescription: d\ntype: prompt\n---\n{~exons.var name=\"x\" default=\"d\" /~}"
		out, err := engine.Execute(ctx, src, nil)
		require.NoError(t, err)
		assert.Equal(t, "d", strings.TrimSpace(out))
	})
}

// TestTemplate_Inputs_NonMapGuard pins the rule that keeps this feature from breaking the
// single most idiomatic key name in a prompt library.
//
// data["input"] = "the user's question" is ordinary, and an include copies its non-reserved
// attributes into the child data as STRINGS — so {~exons.include input="y" /~} lets a
// DOCUMENT put a string under the reserved root. Neither may be turned into an error by a
// feature the author never opted into.
func TestTemplate_Inputs_NonMapGuard(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("a string under the reserved root renders exactly as before", func(t *testing.T) {
		src := inputDoc("  tone:\n    type: text\n    default: neutral\n",
			`{~exons.var name="input" /~}`)
		out, err := engine.Execute(ctx, src, map[string]any{ContextKeyInput: "the user's question"})
		require.NoError(t, err)
		assert.Equal(t, "the user's question", strings.TrimSpace(out))
	})

	t.Run("a map[string]string binding is accepted and normalized", func(t *testing.T) {
		src := inputDoc("  tone:\n    type: text\n    default: neutral\n  extra:\n    type: text\n",
			`{~exons.input name="tone" /~}|{~exons.input name="extra" /~}`)
		out, err := engine.Execute(ctx, src, map[string]any{
			ContextKeyInput: map[string]string{"extra": "e"},
		})
		require.NoError(t, err)
		assert.Equal(t, "neutral|e", strings.TrimSpace(out))
	})
}

// TestTemplate_Inputs_PresentAsNil pins the choice that makes control flow work with no
// executor change at all: a declared input with neither a bound value nor a default is
// PRESENT and nil, and every consumer already does the right thing with nil.
func TestTemplate_Inputs_PresentAsNil(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("an unbound optional list iterates zero times rather than failing", func(t *testing.T) {
		// This is the case that would otherwise be fatal: a missing for-loop collection path
		// errors unconditionally, bypassing the onerror escape hatch entirely.
		src := inputDoc("  sources:\n    type: multiselect\n",
			`[{~exons.for item="s" in="input.sources"~}{~exons.var name="s" /~};{~/exons.for~}]`)
		out, err := engine.Execute(ctx, src, nil)
		require.NoError(t, err)
		assert.Equal(t, "[]", strings.TrimSpace(out))
	})

	t.Run("a bound list iterates", func(t *testing.T) {
		src := inputDoc("  sources:\n    type: multiselect\n",
			`{~exons.for item="s" in="input.sources"~}{~exons.var name="s" /~};{~/exons.for~}`)
		out, err := engine.Execute(ctx, src, map[string]any{
			ContextKeyInput: map[string]any{"sources": []any{"a", "b"}},
		})
		require.NoError(t, err)
		assert.Equal(t, "a;b;", strings.TrimSpace(out))
	})

	t.Run("an unbound boolean evaluates falsy without erroring", func(t *testing.T) {
		src := inputDoc("  verbose:\n    type: boolean\n",
			`{~exons.if eval="input.verbose"~}LOUD{~exons.else~}quiet{~/exons.if~}`)
		out, err := engine.Execute(ctx, src, nil)
		require.NoError(t, err)
		assert.Equal(t, "quiet", strings.TrimSpace(out))
	})

	t.Run("a bound boolean branches", func(t *testing.T) {
		src := inputDoc("  verbose:\n    type: boolean\n",
			`{~exons.if eval="input.verbose"~}LOUD{~exons.else~}quiet{~/exons.if~}`)
		out, err := engine.Execute(ctx, src, map[string]any{
			ContextKeyInput: map[string]any{"verbose": true},
		})
		require.NoError(t, err)
		assert.Equal(t, "LOUD", strings.TrimSpace(out))
	})

	t.Run("an unbound input with no default renders empty, not an error", func(t *testing.T) {
		src := inputDoc("  tone:\n    type: text\n", `[{~exons.input name="tone" /~}]`)
		out, err := engine.Execute(ctx, src, nil)
		require.NoError(t, err)
		assert.Equal(t, "[]", strings.TrimSpace(out))
	})

	t.Run("a tag-level default covers an unbound input with no declared default", func(t *testing.T) {
		src := inputDoc("  tone:\n    type: text\n", `{~exons.input name="tone" default="fallback" /~}`)
		out, err := engine.Execute(ctx, src, nil)
		require.NoError(t, err)
		assert.Equal(t, "fallback", strings.TrimSpace(out))
	})
}

// TestTemplate_Inputs_UndeclaredIsAnError pins the payoff of present-⇔-declared: because
// injection makes every declared input present, an ABSENT name can only be undeclared, which
// is an author typo and the most valuable diagnostic this verb offers.
func TestTemplate_Inputs_UndeclaredIsAnError(t *testing.T) {
	ctx := context.Background()

	t.Run("throws under the default strategy and suggests the intended name", func(t *testing.T) {
		engine := MustNew()
		src := inputDoc("  max_bullets:\n    type: number\n    default: 5\n",
			`{~exons.input name="max_bullet" /~}`)
		_, err := engine.Execute(ctx, src, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not declared")
		assert.Contains(t, err.Error(), "max_bullets", "the did-you-mean must name the declared input")
	})

	t.Run("degrades to empty under the remove strategy like any other tag error", func(t *testing.T) {
		engine, err := New(WithErrorStrategy(ErrorStrategyRemove))
		require.NoError(t, err)
		src := inputDoc("  a:\n    type: text\n", `[{~exons.input name="typo" /~}]`)
		out, xerr := engine.Execute(ctx, src, nil)
		require.NoError(t, xerr)
		assert.Equal(t, "[]", strings.TrimSpace(out))
	})

	t.Run("a document declaring no inputs at all still reports the name", func(t *testing.T) {
		engine := MustNew()
		src := "---\nname: t\ndescription: d\ntype: prompt\n---\n{~exons.input name=\"tone\" /~}"
		_, err := engine.Execute(ctx, src, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not declared")
	})
}

// TestTemplate_Inputs_RequiredAttrIsRefused pins that `required=` is refused at BOTH entry
// points. The executor never calls a resolver's Validate, so a refusal expressed only there
// would never fire — the check lives in one helper both call.
func TestTemplate_Inputs_RequiredAttrIsRefused(t *testing.T) {
	engine := MustNew()
	src := inputDoc("  tone:\n    type: text\n    default: d\n",
		`{~exons.input name="tone" required="true" /~}`)
	_, err := engine.Execute(context.Background(), src, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "'required' attribute is not accepted")
}

// TestTemplate_Inputs_DefaultsAreNotSharedAcrossRenders is the concurrency guarantee.
//
// A *Template is built to be executed many times, concurrently. InputDef.Default holds
// YAML-decoded values, so aliasing a map or slice default into a live context would share it
// across renders. Run with -race.
func TestTemplate_Inputs_DefaultsAreNotSharedAcrossRenders(t *testing.T) {
	engine := MustNew()
	src := inputDoc("  cfg:\n    type: text\n    default:\n      k: v\n",
		`{~exons.var name="input.cfg.k" /~}`)
	tmpl, err := engine.Parse(src)
	require.NoError(t, err)

	const renders = 32
	var wg sync.WaitGroup
	outputs := make([]string, renders)
	for i := 0; i < renders; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			out, err := tmpl.Execute(context.Background(), nil)
			assert.NoError(t, err)
			outputs[i] = strings.TrimSpace(out)
		}(i)
	}
	wg.Wait()

	for i, out := range outputs {
		assert.Equal(t, "v", out, "render %d saw a mutated default", i)
	}
}

// TestTemplate_Inputs_ExplainMatchesExecute guards the funnel bypass.
//
// Explain calls the executor directly rather than going through ExecuteWithContext, so
// injection has to be applied there too. Without it a document declaring inputs would EXPLAIN
// differently than it RENDERS — the worst possible failure mode for a debugging tool, because
// the discrepancy looks like the bug being investigated.
func TestTemplate_Inputs_ExplainMatchesExecute(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()
	src := inputDoc("  tone:\n    type: text\n    default: neutral\n",
		`Write in a {~exons.input name="tone" /~} voice.`)

	tmpl, err := engine.Parse(src)
	require.NoError(t, err)

	executed, err := tmpl.Execute(ctx, nil)
	require.NoError(t, err)

	explained := tmpl.Explain(ctx, nil)
	require.NoError(t, explained.Error)
	assert.Equal(t, executed, explained.Output)
}

// =============================================================================
// exons.input — value rendering, the manifest, and the binary guarantee
// =============================================================================

func TestTemplate_Inputs_Rendering(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("a list reads as prose with the default separator", func(t *testing.T) {
		src := inputDoc("  tags:\n    type: multiselect\n", `{~exons.input name="tags" /~}`)
		out, err := engine.Execute(ctx, src, map[string]any{
			ContextKeyInput: map[string]any{"tags": []any{"a", "b", "c"}},
		})
		require.NoError(t, err)
		assert.Equal(t, "a, b, c", strings.TrimSpace(out))
	})

	t.Run("join names the separator", func(t *testing.T) {
		src := inputDoc("  tags:\n    type: multiselect\n", `{~exons.input name="tags" join=" | " /~}`)
		out, err := engine.Execute(ctx, src, map[string]any{
			ContextKeyInput: map[string]any{"tags": []any{"a", "b"}},
		})
		require.NoError(t, err)
		assert.Equal(t, "a | b", strings.TrimSpace(out))
	})

	t.Run("a file list renders as a filename manifest", func(t *testing.T) {
		src := inputDoc("  sources:\n    type: file-upload\n", `{~exons.input name="sources" /~}`)
		out, err := engine.Execute(ctx, src, map[string]any{
			ContextKeyInput: map[string]any{"sources": []any{
				map[string]any{"name": "q3.pdf", "mime_type": "application/pdf", "size_bytes": 1048576},
				map[string]any{"name": "notes.txt"},
			}},
		})
		require.NoError(t, err)
		assert.Equal(t, "- q3.pdf (application/pdf, 1.0 MB)\n- notes.txt", strings.TrimSpace(out))
	})

	t.Run("a heterogeneous list is not mistaken for a manifest", func(t *testing.T) {
		src := inputDoc("  things:\n    type: multiselect\n", `{~exons.input name="things" /~}`)
		out, err := engine.Execute(ctx, src, map[string]any{
			ContextKeyInput: map[string]any{"things": []any{
				map[string]any{"name": "a"}, "plain",
			}},
		})
		require.NoError(t, err)
		assert.NotContains(t, out, "- a", "one named map must not turn an ordinary list into a manifest")
	})
}

// TestTemplate_Inputs_BinaryIsWithheld is a GUARANTEE, not a formatting preference.
//
// renderValue matches a slice on its element KIND and returns string(rv.Bytes()) for uint8 —
// deliberately, so json.RawMessage renders as text. That arm is correct for exons.var and
// catastrophic for an uploaded file: a caller binding the bytes would paste the whole file
// body into the prompt, by accident rather than by misuse.
func TestTemplate_Inputs_BinaryIsWithheld(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()
	const secret = "SENSITIVE FILE BODY"

	var digest [len(secret)]byte
	copy(digest[:], secret)

	cases := map[string]any{
		"a bare byte slice":         []byte(secret),
		"a byte slice in a list":    []any{[]byte(secret)},
		"a byte slice in a map":     map[string]any{"body": []byte(secret)},
		"a byte slice in a nested":  []any{map[string]any{"name": "f", "body": []byte(secret)}},
		"a named byte-slice type":   testBlob(secret),
		"a byte slice under a file": []any{map[string]any{"name": "f.pdf", "content": []byte(secret)}},
		// The four below are the shapes the FIRST version of the sweep did not reach. The
		// struct is the important one: renderValue has an explicit reflect.Struct arm that
		// walks exported fields, so the whole file body was rendered as text — through the
		// one function written to stop exactly that.
		"an exported struct field":  testUpload{Name: "q3.pdf", Body: []byte(secret)},
		"a pointer to a struct":     &testUpload{Name: "q3.pdf", Body: []byte(secret)},
		"a struct inside a list":    []any{testUpload{Name: "q3.pdf", Body: []byte(secret)}},
		"a byte ARRAY, not a slice": digest,
	}

	// The numeric rendering of the first two bytes. An untraversed byte ARRAY renders as
	// `83, 69, 78, …`, which contains no secret SUBSTRING — so asserting only that the string
	// is absent would pass while every byte leaked.
	numeric := strconv.Itoa(int(secret[0])) + ", " + strconv.Itoa(int(secret[1]))

	for name, bound := range cases {
		t.Run(name+" never reaches the output", func(t *testing.T) {
			src := inputDoc("  doc:\n    type: file-upload\n", `{~exons.input name="doc" /~}`)
			out, err := engine.Execute(ctx, src, map[string]any{
				ContextKeyInput: map[string]any{"doc": bound},
			})
			require.NoError(t, err)
			assert.NotContains(t, out, secret,
				"a declared input rendered raw file bytes into the prompt as text")
			assert.NotContains(t, out, numeric,
				"a declared input rendered raw file bytes into the prompt as numbers")
		})
	}

	t.Run("exons.var is deliberately unchanged", func(t *testing.T) {
		// The withholding is a property of exons.input, not of the renderer. exons.var
		// renders a byte slice as text on purpose, so json.RawMessage reads as JSON.
		src := "---\nname: t\ndescription: d\ntype: prompt\n---\n{~exons.var name=\"b\" /~}"
		out, err := engine.Execute(ctx, src, map[string]any{"b": []byte(secret)})
		require.NoError(t, err)
		assert.Contains(t, out, secret)
	})
}

// testBlob is a named byte-slice type, standing in for json.RawMessage — the withholding
// matches on element kind so a named type is caught too, not only the literal []byte.
type testBlob []byte

// testUpload is the shape a Go runtime naturally reaches for when it binds an uploaded file:
// a struct with an exported byte-slice field. It is what the sweep originally missed.
type testUpload struct {
	Name string
	Body []byte
}

// TestTemplate_Inputs_NilRootStillApplies pins the one case rule 1 must NOT claim.
//
// data["input"] = nil says "nothing is bound", not "input is my own variable", but the guard
// tested only for "not a map" — so a nil root took the no-op exit and the document's declared
// defaults silently did not apply, which is the single outcome this feature exists to prevent.
func TestTemplate_Inputs_NilRootStillApplies(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	src := inputDoc("  tone:\n    type: text\n    default: neutral\n", `[{~exons.input name="tone" /~}]`)
	out, err := engine.Execute(ctx, src, map[string]any{ContextKeyInput: nil})
	require.NoError(t, err)
	assert.Equal(t, "[neutral]", strings.TrimSpace(out),
		"a nil input root must apply the declared defaults, not skip injection")

	// The actual rule-1 case is unaffected: a STRING under the reserved key is a document's own
	// variable and still renders as it did before this feature existed.
	strSrc := inputDoc("  tone:\n    type: text\n    default: neutral\n", `[{~exons.var name="input" /~}]`)
	strOut, err := engine.Execute(ctx, strSrc, map[string]any{ContextKeyInput: "the user's question"})
	require.NoError(t, err)
	assert.Equal(t, "[the user's question]", strings.TrimSpace(strOut))
}

// TestTemplate_Inputs_CallerBindingsAreNotAliased is the sibling of
// TestTemplate_Inputs_DefaultsAreNotSharedAcrossRenders.
//
// Context.Get returns the LIVE value rather than a copy, and the map injection replaces came
// from Context.Data(), which deep-copies. So merging caller bindings shallowly did not merely
// miss an optimisation — it WEAKENED a thread-safety property the context documents, for the
// one code path built to be executed concurrently.
func TestTemplate_Inputs_CallerBindingsAreNotAliased(t *testing.T) {
	engine := MustNew()

	tmpl, err := engine.Parse(inputDoc("  cfg:\n    type: text\n", `{~exons.input name="cfg" /~}`))
	require.NoError(t, err)

	// The caller keeps its own reference — exactly what a concurrent caller reusing a data map
	// across renders does.
	nested := map[string]any{"depth": "original"}
	injected := tmpl.contextWithInputs(context.Background(), NewContext(map[string]any{
		ContextKeyInput: map[string]any{"cfg": nested},
	}))

	nested["depth"] = "mutated by the caller after injection"

	got, found := injected.Get(ContextKeyInput + PathSeparator + "cfg" + PathSeparator + "depth")
	require.True(t, found)
	assert.Equal(t, "original", got,
		"the caller's nested map was ALIASED into the render context — a mutation on the caller's "+
			"side reached a context the render had already built")
}

// =============================================================================
// Spec.BindInputs / Spec.ValidateInputBinding — the pre-render contract
// =============================================================================

func TestSpec_BindInputs(t *testing.T) {
	spec, err := ParseYAMLSpec("name: t\ninputs:\n  a:\n    type: text\n    default: da\n  b:\n    type: text\n")
	require.NoError(t, err)

	t.Run("nil values yields the declared defaults alone", func(t *testing.T) {
		bound := spec.BindInputs(nil)
		assert.Equal(t, "da", bound["a"])
		assert.Nil(t, bound["b"])
	})

	t.Run("supplied values win and undeclared keys are preserved", func(t *testing.T) {
		bound := spec.BindInputs(map[string]any{"a": "mine", "extra": 1})
		assert.Equal(t, "mine", bound["a"])
		assert.Equal(t, 1, bound["extra"], "silently dropping a caller's value is worse than reporting it")
	})

	t.Run("the result is independent of the spec", func(t *testing.T) {
		mapSpec, err := ParseYAMLSpec("name: t\ninputs:\n  cfg:\n    type: text\n    default:\n      k: v\n")
		require.NoError(t, err)
		bound := mapSpec.BindInputs(nil)
		bound["cfg"].(map[string]any)["k"] = "mutated"
		assert.Equal(t, "v", mapSpec.Inputs["cfg"].Default.(map[string]any)["k"])
	})
}

func TestSpec_ValidateInputBinding(t *testing.T) {
	src := "name: t\ninputs:\n" +
		"  who:\n    type: text\n    required: true\n" +
		"  tone:\n    type: select\n    options:\n      - value: warm\n      - value: cool\n" +
		"  files:\n    type: file-upload\n    max_files: 2\n"
	spec, err := ParseYAMLSpec(src)
	require.NoError(t, err)

	t.Run("a required input with no value and no default is reported", func(t *testing.T) {
		errs := spec.ValidateInputBinding(nil)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "who")
	})

	t.Run("a value off the declared option list is reported", func(t *testing.T) {
		errs := spec.ValidateInputBinding(map[string]any{"who": "me", "tone": "spicy"})
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "spicy")
	})

	t.Run("max_files is enforced", func(t *testing.T) {
		errs := spec.ValidateInputBinding(map[string]any{
			"who": "me", "files": []any{1, 2, 3},
		})
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "at most 2")
	})

	t.Run("an acceptable binding reports nothing", func(t *testing.T) {
		assert.Empty(t, spec.ValidateInputBinding(map[string]any{"who": "me", "tone": "warm"}))
	})

	t.Run("an input declaring no options constrains nothing", func(t *testing.T) {
		open, err := ParseYAMLSpec("name: t\ninputs:\n  pick:\n    type: select\n")
		require.NoError(t, err)
		assert.Empty(t, open.ValidateInputBinding(map[string]any{"pick": "anything"}))
	})

	t.Run("a declared default satisfies requiredness", func(t *testing.T) {
		withDefault, err := ParseYAMLSpec("name: t\ninputs:\n  r:\n    type: text\n    required: true\n    default: d\n")
		require.NoError(t, err)
		assert.Empty(t, withDefault.ValidateInputBinding(nil),
			"the value is never absent at render, so refusing here makes an unsubmittable form")
	})
}

// =============================================================================
// DryRun — the sound analysis surface
// =============================================================================

func TestTemplate_DryRun_Inputs(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("records each reference and whether it is declared", func(t *testing.T) {
		src := inputDoc("  tone:\n    type: text\n",
			`{~exons.input name="tone" /~} and {~exons.input name="ghost" default="d" /~}`)
		tmpl, err := engine.Parse(src)
		require.NoError(t, err)

		result := tmpl.DryRun(ctx, nil)
		require.Len(t, result.Inputs, 2)

		assert.Equal(t, "tone", result.Inputs[0].Name)
		assert.True(t, result.Inputs[0].Declared)
		assert.False(t, result.Inputs[0].HasDefault)

		assert.Equal(t, "ghost", result.Inputs[1].Name)
		assert.False(t, result.Inputs[1].Declared)
		assert.True(t, result.Inputs[1].HasDefault)
		assert.Equal(t, "d", result.Inputs[1].Default)
	})

	t.Run("an input reference is not also reported as a variable", func(t *testing.T) {
		// Folding the two together would throw away the distinction the verb exists for.
		src := inputDoc("  tone:\n    type: text\n", `{~exons.input name="tone" /~}`)
		tmpl, err := engine.Parse(src)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		assert.Empty(t, result.Variables)
		assert.Len(t, result.Inputs, 1)
	})

	t.Run("a reference inside an inert span is structurally invisible", func(t *testing.T) {
		// exons.raw, exons.comment and {~~ ~~} are consumed at the LEXER, so no tag node is
		// ever built for their contents. This is a stronger guarantee than a rule a source
		// scanner has to remember to apply.
		for name, body := range map[string]string{
			"raw block":     `{~exons.raw~}{~exons.input name="tone" /~}{~/exons.raw~}`,
			"comment block": `{~exons.comment~}{~exons.input name="tone" /~}{~/exons.comment~}`,
		} {
			src := inputDoc("  tone:\n    type: text\n", body)
			tmpl, err := engine.Parse(src)
			require.NoError(t, err, name)
			assert.Empty(t, tmpl.DryRun(ctx, nil).Inputs, name)
		}
	})

	t.Run("a dotted reference is judged by its first segment", func(t *testing.T) {
		src := inputDoc("  user:\n    type: text\n", `{~exons.input name="user.email" /~}`)
		tmpl, err := engine.Parse(src)
		require.NoError(t, err)
		result := tmpl.DryRun(ctx, nil)
		require.Len(t, result.Inputs, 1)
		assert.True(t, result.Inputs[0].Declared)
	})
}

// TestTemplate_DryRun_RegisteredIsAsked pins the fix to a field that used to lie.
//
// Registered was hardcoded true with the comment "assume registered since it parsed" — but
// the parser never consults the resolver registry, so an unregistered verb was reported as
// registered and the one field a caller would use to catch a typo always said everything was
// fine.
func TestTemplate_DryRun_RegisteredIsAsked(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	tmpl, err := engine.Parse(`{~exons.now /~}{~exons.frobnicate /~}`)
	require.NoError(t, err)

	result := tmpl.DryRun(ctx, nil)
	require.Len(t, result.Resolvers, 2)

	byName := map[string]bool{}
	for _, r := range result.Resolvers {
		byName[r.TagName] = r.Registered
	}
	assert.True(t, byName[TagNameNow], "a built-in must report as registered")
	assert.False(t, byName["exons.frobnicate"], "an unregistered verb must not report as registered")
}

// -----------------------------------------------------------------------------
// Template.ValidateInputBinding — the accessor a RUNTIME needs
//
// Spec.ValidateInputBinding walks the document's own frontmatter, so a document
// using `extends:` has every INHERITED required input skipped while
// contextWithInputs binds them regardless. A host asking "was this caller's form
// filled in" through the Spec therefore gets a clean answer over a binding that is
// missing a required value — one rule with two implementations, on the one path
// that decides whether to refuse a request.
//
// ⭐ THE HELD SIBLING IS WHAT MAKES THE FIRST CASE A MEASUREMENT: it asserts the
// Spec-level accessor's own ANSWER on the same binding, so the test states why both
// methods exist rather than merely exercising the new one.
// -----------------------------------------------------------------------------

func TestTemplate_ValidateInputBinding(t *testing.T) {
	engine := MustNew()
	// The parent declares `mandate`, required with no default — the exact shape a
	// child cannot see through its own Spec.
	engine.MustRegisterTemplate("vib-base", "---\nname: vib-base\ndescription: d\ninputs:\n"+
		"  mandate:\n    type: text\n    required: true\n"+
		"  tone:\n    type: select\n    options:\n      - value: warm\n      - value: cool\n"+
		"---\nbody")

	child := "---\nname: vib-child\ndescription: d\ninputs:\n  extra:\n    type: text\n---\n" +
		"{~exons.extends template=\"vib-base\" /~}"
	tmpl, err := engine.Parse(child)
	require.NoError(t, err)

	t.Run("an INHERITED required input with no value is reported", func(t *testing.T) {
		errs, chainErr := tmpl.ValidateInputBinding(map[string]any{"extra": "x"})
		require.NoError(t, chainErr)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "mandate")

		// ⭐ THE HELD SIBLING, AND IT IS THE ENTIRE REASON THIS METHOD EXISTS: the
		// spec-level accessor sees the child's frontmatter alone and reports the
		// binding CLEAN. A build that delegated to it would satisfy every other
		// assertion in this test.
		assert.Empty(t, tmpl.Spec().ValidateInputBinding(map[string]any{"extra": "x"}),
			"the Spec-level accessor is extends-blind; that is what makes it the wrong one here")
	})

	t.Run("an INHERITED select rejects a value off its option list", func(t *testing.T) {
		errs, chainErr := tmpl.ValidateInputBinding(map[string]any{"mandate": "m", "tone": "spicy"})
		require.NoError(t, chainErr)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "spicy")
	})

	t.Run("an acceptable binding reports nothing", func(t *testing.T) {
		errs, chainErr := tmpl.ValidateInputBinding(map[string]any{"mandate": "m", "tone": "warm"})
		require.NoError(t, chainErr)
		assert.Empty(t, errs)
	})

	t.Run("an empty string is UNBOUND, not a supplied value", func(t *testing.T) {
		// Same rule isUnboundInputValue applies at binding time. A form that submits
		// every declared key sends "" for a field the user left alone, and reading
		// that as "supplied" would make requiredness unenforceable from a browser.
		errs, chainErr := tmpl.ValidateInputBinding(map[string]any{"mandate": ""})
		require.NoError(t, chainErr)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "mandate")
	})

	t.Run("a document declaring nothing accepts anything", func(t *testing.T) {
		bare, err := engine.Parse("no frontmatter at all")
		require.NoError(t, err)
		errs, chainErr := bare.ValidateInputBinding(map[string]any{"whatever": "x"})
		require.NoError(t, chainErr)
		assert.Empty(t, errs)
	})

	t.Run("violations are ordered, so the same binding reports the same way twice", func(t *testing.T) {
		// A merged input set is a map; range order would make this correct and unstable.
		multi, err := engine.Parse("---\nname: m\ndescription: d\ninputs:\n" +
			"  zeta:\n    type: text\n    required: true\n" +
			"  alpha:\n    type: text\n    required: true\n---\nbody")
		require.NoError(t, err)
		for i := 0; i < 8; i++ {
			errs, chainErr := multi.ValidateInputBinding(nil)
			require.NoError(t, chainErr)
			require.Len(t, errs, 2)
			assert.Contains(t, errs[0].Error(), "alpha")
			assert.Contains(t, errs[1].Error(), "zeta")
		}
	})
}

// TestValidateInputBinding_AnEmptyCollectionDoesNotAnswerARequiredInput pins the hole that
// made the required check a NO-OP for every collection kind.
//
// ⛔ isUnboundInputValue treats only nil and "" as unbound — right for BINDING, where an
// untouched field and a cleared field are one gesture — so reused for requiredness it let an
// explicit `[]` through as "present". Nothing else constrains an empty list (option membership
// and max_files are both vacuous on one), so the binding validated CLEAN while the rendered
// prompt still had a hole in it.
//
// ⭐ NOT A CORNER CASE: a required `file-upload` is the field type atlas#353 was reported
// against, and `{"papers": []}` is exactly what a stored routine or a curl sends.
func TestValidateInputBinding_AnEmptyCollectionDoesNotAnswerARequiredInput(t *testing.T) {
	src := "name: t\ninputs:\n" +
		"  papers:\n    type: file-upload\n    required: true\n" +
		"  tags:\n    type: multiselect\n    required: true\n    options:\n      - value: a\n      - value: b\n"
	spec, err := ParseYAMLSpec(src)
	require.NoError(t, err)

	for _, tc := range []struct {
		name   string
		values map[string]any
	}{
		{"an empty []any", map[string]any{"papers": []any{}, "tags": []any{}}},
		{"an empty []string", map[string]any{"papers": []string{}, "tags": []string{}}},
	} {
		t.Run(tc.name+" is refused", func(t *testing.T) {
			errs := spec.ValidateInputBinding(tc.values)
			require.Len(t, errs, 2, "an empty list is not an answer to a required field")
			assert.Contains(t, errs[0].Error(), "papers")
			assert.Contains(t, errs[1].Error(), "tags")
		})
	}

	t.Run("HELD SIBLING — a NON-empty list is accepted", func(t *testing.T) {
		// Without this, "an empty list is refused" is equally satisfied by a build that
		// refuses every collection-valued required input, i.e. an unsubmittable form.
		assert.Empty(t, spec.ValidateInputBinding(map[string]any{
			"papers": []any{map[string]any{"name": "a.txt"}},
			"tags":   []any{"a"},
		}))
	})

	t.Run("HELD SIBLING — an empty list on an OPTIONAL input is fine", func(t *testing.T) {
		opt, err := ParseYAMLSpec("name: t\ninputs:\n  maybe:\n    type: multiselect\n    options:\n      - value: a\n")
		require.NoError(t, err)
		assert.Empty(t, opt.ValidateInputBinding(map[string]any{"maybe": []any{}}))
	})

	t.Run("an empty map does not answer a required input either", func(t *testing.T) {
		m, err := ParseYAMLSpec("name: t\ninputs:\n  cfg:\n    type: text\n    required: true\n")
		require.NoError(t, err)
		require.Len(t, m.ValidateInputBinding(map[string]any{"cfg": map[string]any{}}), 1)
		assert.Empty(t, m.ValidateInputBinding(map[string]any{"cfg": map[string]any{"k": "v"}}))
	})

	t.Run("a scalar zero is NOT empty — 0 and false are real answers", func(t *testing.T) {
		// ⚠ The direction that would be much worse than the bug: a number field answered 0,
		// or a boolean answered false, is answered. Emptiness is about COLLECTIONS.
		z, err := ParseYAMLSpec("name: t\ninputs:\n" +
			"  n:\n    type: number\n    required: true\n" +
			"  b:\n    type: boolean\n    required: true\n")
		require.NoError(t, err)
		assert.Empty(t, z.ValidateInputBinding(map[string]any{"n": 0, "b": false}))
	})

	t.Run("the TEMPLATE accessor agrees — one rule, not two", func(t *testing.T) {
		// The two accessors differ only in WHICH input set they walk; a fix applied to one
		// and not the other is the drift this package's own comments warn about.
		engine := MustNew()
		tmpl, err := engine.Parse("---\nname: t\ndescription: d\ninputs:\n" +
			"  papers:\n    type: file-upload\n    required: true\n---\nbody")
		require.NoError(t, err)
		errs, chainErr := tmpl.ValidateInputBinding(map[string]any{"papers": []any{}})
		require.NoError(t, chainErr)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "papers")
	})
}
