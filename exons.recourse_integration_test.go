package exons

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DC13-entail, workstream B. Two defects, one funnel.
//
// Until v0.24.0 `exons.if` / `exons.for` / `exons.switch` failed UNCONDITIONALLY: their parse
// read the keys they cared about and let the attribute map fall out of scope, so `onerror=` and
// `default=` — parsed, and handed to the very function that dropped them — were structurally
// unreachable on a block tag. And `Resolver.Validate` was never invoked by the executor, so a
// refusal expressed only there was dead code and the opt-in `Engine.Validate` lint was STRICTER
// than the renderer.
//
// Every test in this file was verified to FAIL before the fix. A regression test that passes
// without the fix is not a guard.

// TestBlockTagOnErrorIsReachable pins the whole of defect 5: each of the six executor sites
// that returned "" , err without consulting the error strategy now routes through
// handleTagError, so an author's onerror= is honoured on a block tag exactly as it is on a
// self-closing one.
func TestBlockTagOnErrorIsReachable(t *testing.T) {
	cases := []struct {
		name   string
		source string
		data   map[string]any
		want   string
	}{
		{
			name:   "for: collection path not found",
			source: `A{~exons.for item="x" in="missing" onerror="remove"~}{~exons.var name="x" /~}{~/exons.for~}B`,
			data:   map[string]any{},
			want:   "AB",
		},
		{
			name:   "for: collection path not found, with default",
			source: `{~exons.for item="x" in="missing" onerror="default" default="(none)"~}!{~/exons.for~}`,
			data:   map[string]any{},
			want:   "(none)",
		},
		{
			name:   "for: value not iterable",
			source: `{~exons.for item="x" in="scalar" onerror="default" default="(not a list)"~}!{~/exons.for~}`,
			data:   map[string]any{"scalar": 42},
			want:   "(not a list)",
		},
		{
			name:   "if: condition expression fails",
			source: `{~exons.if eval="nope(" onerror="default" default="(bad cond)"~}yes{~/exons.if~}`,
			data:   map[string]any{},
			want:   "(bad cond)",
		},
		{
			name:   "switch: dispatch expression fails",
			source: `{~exons.switch eval="nope(" onerror="remove"~}{~exons.case value="a"~}A{~/exons.case~}{~/exons.switch~}`,
			data:   map[string]any{},
			want:   "",
		},
		{
			name:   "case: eval expression fails, strategy inherited from the switch",
			source: `{~exons.switch eval="k" onerror="default" default="(bad case)"~}{~exons.case eval="nope("~}A{~/exons.case~}{~/exons.switch~}`,
			data:   map[string]any{"k": "z"},
			want:   "(bad case)",
		},
	}

	engine := MustNew()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := engine.Execute(context.Background(), c.source, c.data)
			require.NoError(t, err, "onerror= should have suppressed the failure")
			assert.Equal(t, c.want, out)
		})
	}
}

// TestBlockTagWithoutOnErrorStillThrows is the other half of the contract, and the reason the
// case above is a fix rather than a weakening: absent an explicit strategy, a block-tag failure
// is still fatal. Nothing became quietly permissive.
func TestBlockTagWithoutOnErrorStillThrows(t *testing.T) {
	engine := MustNew()
	for _, source := range []string{
		`{~exons.for item="x" in="missing"~}!{~/exons.for~}`,
		`{~exons.if eval="nope("~}yes{~/exons.if~}`,
		`{~exons.switch eval="nope("~}{~exons.case value="a"~}A{~/exons.case~}{~/exons.switch~}`,
	} {
		_, err := engine.Execute(context.Background(), source, map[string]any{})
		assert.Error(t, err, "source=%q", source)
	}
}

// TestBranchLevelOnErrorOverridesTheConstruct pins the attribute-precedence rule: a branch or a
// case supplies its own attributes first and the enclosing construct's second.
//
// This is also the test that would catch the parseConditional threading trap. That function
// returns the position of the tag TERMINATING each branch, so a branch's own attributes must be
// captured BEFORE the recursive call that overwrites them — the same discipline v0.23.0 had to
// apply to the position field, one field over. Read the elseif's attributes after the call and
// this test reports the ELSE's strategy for the elseif's failure.
func TestBranchLevelOnErrorOverridesTheConstruct(t *testing.T) {
	engine := MustNew()

	t.Run("elseif declares its own strategy", func(t *testing.T) {
		source := `{~exons.if eval="a" onerror="default" default="(from if)"~}A` +
			`{~exons.elseif eval="nope(" onerror="default" default="(from elseif)"~}B` +
			`{~exons.else~}C{~/exons.if~}`
		out, err := engine.Execute(context.Background(), source, map[string]any{"a": false})
		require.NoError(t, err)
		assert.Equal(t, "(from elseif)", out,
			"the failing elseif's OWN default must win over the exons.if's")
	})

	t.Run("elseif inherits the construct's strategy when it declares none", func(t *testing.T) {
		source := `{~exons.if eval="a" onerror="default" default="(from if)"~}A` +
			`{~exons.elseif eval="nope("~}B{~/exons.if~}`
		out, err := engine.Execute(context.Background(), source, map[string]any{"a": false})
		require.NoError(t, err)
		assert.Equal(t, "(from if)", out)
	})

	t.Run("case declares its own strategy", func(t *testing.T) {
		source := `{~exons.switch eval="k" onerror="default" default="(from switch)"~}` +
			`{~exons.case eval="nope(" onerror="default" default="(from case)"~}A{~/exons.case~}` +
			`{~/exons.switch~}`
		out, err := engine.Execute(context.Background(), source, map[string]any{"k": "z"})
		require.NoError(t, err)
		assert.Equal(t, "(from case)", out)
	})
}

// TestBlockTagKeepRawEmitsTheWholeConstruct pins design decision B-1. keepraw needs RawSource,
// and no block construct captured one — the three of them consume their closing sequence inside
// a body helper. Letting keepraw fall back to "" would have been a strategy that lies about
// what it did, so the source is captured through the closing tag instead.
func TestBlockTagKeepRawEmitsTheWholeConstruct(t *testing.T) {
	engine := MustNew()

	for _, c := range []struct{ name, source string }{
		{"for", `{~exons.for item="x" in="missing" onerror="keepraw"~}body{~/exons.for~}`},
		{"if", `{~exons.if eval="nope(" onerror="keepraw"~}body{~/exons.if~}`},
		{"switch", `{~exons.switch eval="nope(" onerror="keepraw"~}{~exons.case value="a"~}A{~/exons.case~}{~/exons.switch~}`},
	} {
		t.Run(c.name, func(t *testing.T) {
			out, err := engine.Execute(context.Background(), c.source, map[string]any{})
			require.NoError(t, err)
			assert.Equal(t, c.source, out,
				"keepraw must emit the construct from its opening tag through its closing tag")
		})
	}
}

// TestResolverValidateIsInvokedByTheExecutor pins defect 6.
//
// exons.message is the ONLY built-in whose Validate refusal was dead code — var, include, env
// and ref all duplicate their required-attribute check into Resolve. MessageResolver.Resolve
// does `role, _ := attrs.Get(AttrRole)` and never calls isValidRole, so a missing or
// unrecognised role rendered a MALFORMED MESSAGE MARKER, which ExecuteAndExtractMessages then
// parses into a MessageInfo that a runtime hands to an LLM chat API. Engine.Validate rejected
// both spellings; the renderer did not. The lint was stricter than the renderer.
func TestResolverValidateIsInvokedByTheExecutor(t *testing.T) {
	engine := MustNew()

	t.Run("missing role is refused at render", func(t *testing.T) {
		_, err := engine.Execute(context.Background(), `{~exons.message~}hi{~/exons.message~}`, map[string]any{})
		require.Error(t, err)
	})

	t.Run("unrecognised role is refused at render", func(t *testing.T) {
		_, err := engine.Execute(context.Background(),
			`{~exons.message role="bogus"~}hi{~/exons.message~}`, map[string]any{})
		require.Error(t, err)
	})

	t.Run("no marker leaks into the output on refusal", func(t *testing.T) {
		out, err := engine.Execute(context.Background(),
			`{~exons.message role="bogus" onerror="remove"~}hi{~/exons.message~}`, map[string]any{})
		require.NoError(t, err)
		// The marker constant is internal; its literal prefix is the contract that matters
		// here — this is exactly the byte sequence ExecuteAndExtractMessages scans for.
		assert.NotContains(t, out, "\x00MSG_START:",
			"a refused message must not emit a start marker for ExecuteAndExtractMessages to parse")
	})

	t.Run("a valid role still renders", func(t *testing.T) {
		out, err := engine.Execute(context.Background(),
			`{~exons.message role="system"~}hi{~/exons.message~}`, map[string]any{})
		require.NoError(t, err)
		assert.True(t, strings.Contains(out, RoleSystem), "got %q", out)
	})
}

// TestResolverValidateRefusalIsGovernable is the reason invoking Validate is a fix rather than
// a new unconditional stop: it goes through the same funnel, so onerror= applies. This is the
// concern NowResolver.Validate's comment raised about parse-time validation, answered.
func TestResolverValidateRefusalIsGovernable(t *testing.T) {
	engine := MustNew()
	out, err := engine.Execute(context.Background(),
		`A{~exons.message role="bogus" onerror="default" default="(bad role)"~}hi{~/exons.message~}B`,
		map[string]any{})
	require.NoError(t, err)
	assert.Contains(t, out, "(bad role)")
}

// TestLintCatchesAnInvalidOnErrorOnEveryShapeThatHonoursIt closes the half of defect 5 that only
// became a defect once defect 5 was fixed.
//
// Before v0.24.0 an onerror= on a block construct was inert, so Engine.Validate checking it only on
// TagNode was correct. Now every one of these shapes HONOURS the attribute, and the failure mode is
// silent and inverted: getErrorStrategy returns as soon as the key is present, and ParseErrorStrategy
// maps an unrecognised value to THROW. So a typo does not degrade to the renderer's configured
// leniency — it hard-fails under a renderer configured never to hard-fail.
//
// The lint is the only place that can say so, and a lint that covers some of the shapes teaches an
// author to read its silence as approval.
func TestLintCatchesAnInvalidOnErrorOnEveryShapeThatHonoursIt(t *testing.T) {
	engine := MustNew()

	cases := []struct {
		name   string
		source string
	}{
		{"plain tag", `{~exons.var name="x" onerror="remov" /~}`},
		{"exons.if", `{~exons.if eval="x" onerror="remov"~}a{~/exons.if~}`},
		{"an individual elseif", `{~exons.if eval="x"~}a{~exons.elseif eval="y" onerror="remov"~}b{~/exons.if~}`},
		{"exons.for", `{~exons.for item="i" in="xs" onerror="remov"~}a{~/exons.for~}`},
		{"exons.switch", `{~exons.switch eval="x" onerror="remov"~}{~exons.case value="a"~}a{~/exons.case~}{~/exons.switch~}`},
		{"an individual case", `{~exons.switch eval="x"~}{~exons.case value="a" onerror="remov"~}a{~/exons.case~}{~/exons.switch~}`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result, err := engine.Validate(c.source)
			require.NoError(t, err)
			var found bool
			for _, issue := range result.Issues() {
				if issue.Message == ErrMsgInvalidOnErrorAttr {
					found = true
					break
				}
			}
			assert.True(t, found, "an unrecognised onerror= must be reported on every shape that honours it")
		})
	}

	t.Run("a VALID onerror is never reported, on any shape", func(t *testing.T) {
		// The companion direction: a lint that fires on correct code is the reason authors turn
		// lints off, and this one is a SeverityError.
		valid := []string{
			`{~exons.if eval="x" onerror="remove"~}a{~/exons.if~}`,
			`{~exons.for item="i" in="xs" onerror="keepraw"~}a{~/exons.for~}`,
			`{~exons.switch eval="x" onerror="throw"~}{~exons.case value="a"~}a{~/exons.case~}{~/exons.switch~}`,
			`{~exons.if eval="x"~}a{~exons.elseif eval="y" onerror="default" default="d"~}b{~/exons.if~}`,
		}
		for _, src := range valid {
			result, err := engine.Validate(src)
			require.NoError(t, err)
			for _, issue := range result.Issues() {
				assert.NotEqual(t, ErrMsgInvalidOnErrorAttr, issue.Message, "source: %s", src)
			}
		}
	})
}
