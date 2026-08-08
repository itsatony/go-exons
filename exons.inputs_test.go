package exons

import (
	"context"
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

	cases := map[string]any{
		"a bare byte slice":         []byte(secret),
		"a byte slice in a list":    []any{[]byte(secret)},
		"a byte slice in a map":     map[string]any{"body": []byte(secret)},
		"a byte slice in a nested":  []any{map[string]any{"name": "f", "body": []byte(secret)}},
		"a named byte-slice type":   testBlob(secret),
		"a byte slice under a file": []any{map[string]any{"name": "f.pdf", "content": []byte(secret)}},
	}

	for name, bound := range cases {
		t.Run(name+" never reaches the output", func(t *testing.T) {
			src := inputDoc("  doc:\n    type: file-upload\n", `{~exons.input name="doc" /~}`)
			out, err := engine.Execute(ctx, src, map[string]any{
				ContextKeyInput: map[string]any{"doc": bound},
			})
			require.NoError(t, err)
			assert.NotContains(t, out, secret,
				"a declared input rendered raw file bytes into the prompt")
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
