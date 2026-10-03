package exons

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStrictRenderability_ValidateAgreesWithExecute is go-exons#13's contract: on a
// WithStrictRenderability engine, Validate(...).Errors() is non-empty EXACTLY when Execute refuses,
// for data-independent refusals. Every row RENDERS as well as validates, so the verdict cannot
// drift from what Execute does (the shape deepr pinned its interim copy with).
func TestStrictRenderability_ValidateAgreesWithExecute(t *testing.T) {
	t.Setenv("EXONS_STRICTRENDER_OK", "value")
	custom := NewResolverFunc("custom.tag", func(context.Context, *Context, Attributes) (string, error) {
		return "c", nil
	}, nil)

	cases := []struct {
		name       string
		opts       []Option
		source     string
		wantRefuse bool
	}{
		{"plain text", nil, "hello", false},
		{"env on an engine that never opted in", nil, `{~exons.env name="EXONS_STRICTRENDER_OK" /~}`, true},
		{"env with default= on an engine that never opted in", nil, `{~exons.env name="EXONS_STRICTRENDER_OK" default="x" /~}`, true},
		{"env opted in for every name", []Option{WithEnvEnabled()}, `{~exons.env name="EXONS_STRICTRENDER_OK" /~}`, false},
		{"env name on the allowlist", []Option{WithEnvAllowlist([]string{"EXONS_STRICTRENDER_*"})}, `{~exons.env name="EXONS_STRICTRENDER_OK" /~}`, false},
		{"env name outside the allowlist", []Option{WithEnvAllowlist([]string{"OTHER_*"})}, `{~exons.env name="EXONS_STRICTRENDER_OK" /~}`, true},
		{"env name the denylist blocks", []Option{WithEnvEnabled()}, `{~exons.env name="MY_API_KEY" /~}`, true},
		{"env disabled wins over enabled", []Option{WithEnvEnabled(), WithEnvDisabled()}, `{~exons.env name="EXONS_STRICTRENDER_OK" /~}`, true},
		{"env inside an if branch", nil, `{~exons.if eval="true"~}{~exons.env name="EXONS_STRICTRENDER_OK" /~}{~/exons.if~}`, true},
		{"unknown tag", nil, `{~nope.tag /~}`, true},
		{"unknown tag inside a message", nil, `{~exons.message role="user"~}{~nope.tag /~}{~/exons.message~}`, true},
		{"unknown tag inside raw is inert", nil, `{~exons.raw~}{~nope.tag /~}{~/exons.raw~}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := append([]Option{WithStrictRenderability()}, tc.opts...)
			e := MustNew(opts...)
			require.NoError(t, e.Register(custom))

			res, err := e.Validate(tc.source)
			require.NoError(t, err)
			_, execErr := e.Execute(context.Background(), tc.source, nil)

			assert.Equal(t, tc.wantRefuse, execErr != nil, "Execute: %v", execErr)
			assert.Equal(t, tc.wantRefuse, len(res.Errors()) > 0, "Validate errors: %v", res.Errors())
		})
	}

	t.Run("a registered custom tag renders and validates", func(t *testing.T) {
		e := MustNew(WithStrictRenderability())
		require.NoError(t, e.Register(custom))
		res, err := e.Validate(`{~custom.tag /~}`)
		require.NoError(t, err)
		assert.Empty(t, res.Errors())
	})
}

// TestStrictRenderability_DefaultUnchanged: without the option both constructs still pass Validate
// (an unknown tag stays a WARNING) — the option is opt-in, a stricter default would refuse stored
// documents fleet-wide.
func TestStrictRenderability_DefaultUnchanged(t *testing.T) {
	e := MustNew()
	res, err := e.Validate(`{~exons.env name="HOME" /~}{~nope.tag /~}`)
	require.NoError(t, err)
	assert.Empty(t, res.Errors())
	require.Len(t, res.Warnings(), 1)
	assert.Equal(t, ErrMsgUnknownTagInTemplate, res.Warnings()[0].Message)
}

// TestStrictRenderability_DoesNotChangeParse: it is separate from WithStrictAttributes, whose Parse
// must keep tolerating a resolver registered AFTER the parse (aigentverse).
func TestStrictRenderability_DoesNotChangeParse(t *testing.T) {
	e := MustNew(WithStrictAttributes(), WithStrictRenderability())
	_, err := e.Parse(`{~later.tag /~}`)
	require.NoError(t, err, "Parse must not refuse an unknown tag, even with both options")

	res, err := e.Validate(`{~later.tag /~}`)
	require.NoError(t, err)
	require.Len(t, res.Errors(), 1)
	assert.Equal(t, "later.tag", res.Errors()[0].TagName)
}

func TestStrictRenderability_EnvRefusalNamesTheOption(t *testing.T) {
	e := MustNew(WithStrictRenderability())
	res, err := e.Validate(`{~exons.env name="HOME" /~}`)
	require.NoError(t, err)
	require.Len(t, res.Errors(), 1)
	assert.True(t, strings.Contains(res.Errors()[0].Message, "WithEnvAllowlist"), res.Errors()[0].Message)
	assert.Equal(t, TagNameEnv, res.Errors()[0].TagName)
}

// TestStrictRenderability_FollowsExecutesStrategy (review M1): the finding's severity is Execute's
// verdict — an error only when the failure would stop the render (effective strategy throw), a
// warning when a tag's onerror= or the engine's strategy lets Execute carry on. Every row renders.
func TestStrictRenderability_FollowsExecutesStrategy(t *testing.T) {
	cases := []struct {
		name       string
		opts       []Option
		source     string
		wantRefuse bool
	}{
		{"unknown tag, onerror=remove", nil, `{~nope.tag onerror="remove" /~}`, false},
		{"unknown tag, onerror=default", nil, `{~nope.tag onerror="default" default="x" /~}`, false},
		{"unknown tag, onerror=keepraw", nil, `{~nope.tag onerror="keepraw" /~}`, false},
		{"unknown tag, onerror=log", nil, `{~nope.tag onerror="log" /~}`, false},
		{"unknown tag, onerror=throw", nil, `{~nope.tag onerror="throw" /~}`, true},
		{"unknown tag, misspelled onerror parses to throw", nil, `{~nope.tag onerror="remov" /~}`, true},
		{"env not opted in, onerror=default", nil, `{~exons.env name="HOME" onerror="default" default="x" /~}`, false},
		{"env not opted in, onerror=remove", nil, `{~exons.env name="HOME" onerror="remove" /~}`, false},
		{"env not opted in, default= alone does not mask it", nil, `{~exons.env name="HOME" default="x" /~}`, true},
		{"lenient engine, unknown tag", []Option{WithErrorStrategy(ErrorStrategyRemove)}, `{~nope.tag /~}`, false},
		{"lenient engine, env not opted in", []Option{WithErrorStrategy(ErrorStrategyDefault)}, `{~exons.env name="HOME" /~}`, false},
		{"lenient engine, tag insists on throw", []Option{WithErrorStrategy(ErrorStrategyRemove)}, `{~nope.tag onerror="throw" /~}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := MustNew(append([]Option{WithStrictRenderability()}, tc.opts...)...)
			res, err := e.Validate(tc.source)
			require.NoError(t, err)
			_, execErr := e.Execute(context.Background(), tc.source, nil)

			assert.Equal(t, tc.wantRefuse, execErr != nil, "Execute: %v", execErr)
			assert.Equal(t, tc.wantRefuse, len(res.Errors()) > 0, "Validate errors: %v", res.Errors())
			if !tc.wantRefuse {
				assert.NotEmpty(t, res.Warnings(), "a tolerated failure is still reported, as a warning")
			}
		})
	}
}

// TestStrictRenderability_BranchesAreJudgedAsIfReached pins the documented limit: Validate cannot
// know which branch the data takes, so a tag in a branch Execute never enters is still reported.
func TestStrictRenderability_BranchesAreJudgedAsIfReached(t *testing.T) {
	e := MustNew(WithStrictRenderability())
	for _, src := range []string{
		`{~exons.if eval="false"~}{~nope.tag /~}{~/exons.if~}`,
		`{~exons.for item="x" in="input.none"~}{~nope.tag /~}{~/exons.for~}`,
	} {
		res, err := e.Validate(src)
		require.NoError(t, err)
		assert.NotEmpty(t, res.Errors(), "reported as if reached: %s", src)
		_, execErr := e.Execute(context.Background(), src, map[string]any{"input": map[string]any{"none": []any{}}})
		assert.NoError(t, execErr, "the render never reaches it: %s", src)
	}
}
