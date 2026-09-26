package exons

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Since v0.35.0 {~exons.env~} is OPT-IN (go-exons#7). Every known consumer renders templates it did
// not write, and a suffix denylist lets an ordinary name through — so the default engine must not
// read the environment from ANY route a template can take into a body: the root, a message, a
// referenced fragment, an included template.

// envOptinVar is an ordinary name — it matches no DefaultEnvDenyPatterns entry, which is exactly
// the name the old default handed to any template that asked.
const (
	envOptinVar   = "EXONS_OPTIN_PROBE_URL"
	envOptinValue = "leaked-value-7f3a"
	envOptinTag   = `{~exons.env name="` + envOptinVar + `" /~}`
)

// envDisabledOptionHint is what an author reads when the tag refuses: the option that enables it.
// Asserted on the error so a refusal can never regress into an unexplained empty string.
const envDisabledOptionHint = "WithEnvAllowlist"

func TestEnvOptIn_DefaultRefusesEveryRoute(t *testing.T) {
	t.Setenv(envOptinVar, envOptinValue)
	ctx := context.Background()

	engineWithBodies := func(t *testing.T) *Engine {
		t.Helper()
		engine := refEngine(t, map[string]string{"frag": "frag:" + envOptinTag})
		engine.MustRegisterTemplate("inc", "inc:"+envOptinTag)
		return engine
	}

	routes := map[string]string{
		"root template":     envOptinTag,
		"inside a message":  `{~exons.message role="system"~}sys ` + envOptinTag + `{~/exons.message~}`,
		"inside a ref body": `{~exons.ref slug="frag" /~}`,
		"inside an include": `{~exons.include template="inc" /~}`,
		"ref in a message":  `{~exons.message role="user"~}{~exons.ref slug="frag" /~}{~/exons.message~}`,
	}
	for name, src := range routes {
		t.Run(name, func(t *testing.T) {
			out, err := engineWithBodies(t).Execute(ctx, src, nil)
			require.Error(t, err, "the default engine must refuse exons.env")
			assert.Contains(t, err.Error(), envDisabledOptionHint, "the refusal must name the option that enables it")
			assert.Contains(t, err.Error(), "WithEnvEnabled")
			assert.NotContains(t, err.Error(), envOptinValue)
			assert.NotContains(t, out, envOptinValue)
		})
	}

	t.Run("a default attribute does not turn the refusal into a silent value under Throw", func(t *testing.T) {
		_, err := MustNew().Execute(ctx, `{~exons.env name="`+envOptinVar+`" default="d" /~}`, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), envDisabledOptionHint)
	})
}

func TestEnvOptIn_OtherStrategiesApplyNormally(t *testing.T) {
	t.Setenv(envOptinVar, envOptinValue)
	ctx := context.Background()

	cases := map[string]struct {
		opts []Option
		src  string
		want string
	}{
		"engine-wide default strategy uses the default attribute": {
			opts: []Option{WithErrorStrategy(ErrorStrategyDefault)},
			src:  `[{~exons.env name="` + envOptinVar + `" default="fallback" /~}]`,
			want: "[fallback]",
		},
		"engine-wide remove strategy removes the tag": {
			opts: []Option{WithErrorStrategy(ErrorStrategyRemove)},
			src:  "[" + envOptinTag + "]",
			want: "[]",
		},
		"per-tag onerror=default": {
			src:  `[{~exons.env name="` + envOptinVar + `" default="fb" onerror="default" /~}]`,
			want: "[fb]",
		},
		"per-tag onerror=keepraw keeps the source, not the value": {
			src:  "[" + `{~exons.env name="` + envOptinVar + `" onerror="keepraw" /~}` + "]",
			want: "[" + `{~exons.env name="` + envOptinVar + `" onerror="keepraw" /~}` + "]",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			out, err := MustNew(tc.opts...).Execute(ctx, tc.src, nil)
			require.NoError(t, err)
			assert.Equal(t, tc.want, out)
			assert.NotContains(t, out, envOptinValue)
		})
	}
}

func TestEnvOptIn_WithEnvEnabledReads(t *testing.T) {
	t.Setenv(envOptinVar, envOptinValue)
	ctx := context.Background()

	engine := refEngine(t, map[string]string{"frag": envOptinTag}, WithEnvEnabled())
	out, err := engine.Execute(ctx, envOptinTag+"|"+`{~exons.ref slug="frag" /~}`, nil)
	require.NoError(t, err)
	assert.Equal(t, envOptinValue+"|"+envOptinValue, out)

	t.Run("the default denylist still applies once enabled", func(t *testing.T) {
		t.Setenv("EXONS_OPTIN_PROBE_KEY", "k")
		_, err := MustNew(WithEnvEnabled()).Execute(ctx, `{~exons.env name="EXONS_OPTIN_PROBE_KEY" /~}`, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "denied")
	})

	t.Run("a custom denylist applies with WithEnvEnabled", func(t *testing.T) {
		_, err := MustNew(WithEnvEnabled(), WithEnvDenylist([]string{"EXONS_OPTIN_*"})).Execute(ctx, envOptinTag, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "denied")
	})

	t.Run("a denylist alone does not enable the tag", func(t *testing.T) {
		_, err := MustNew(WithEnvDenylist(nil)).Execute(ctx, envOptinTag, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), envDisabledOptionHint)
	})
}

func TestEnvOptIn_AllowlistAdmitsOnlyListed(t *testing.T) {
	t.Setenv(envOptinVar, envOptinValue)
	t.Setenv("EXONS_OPTIN_OTHER", "other-value")
	ctx := context.Background()

	engine := MustNew(WithEnvAllowlist([]string{envOptinVar}))

	out, err := engine.Execute(ctx, envOptinTag, nil)
	require.NoError(t, err, "a non-empty allowlist implies opt-in")
	assert.Equal(t, envOptinValue, out)

	out, err = engine.Execute(ctx, `{~exons.env name="EXONS_OPTIN_OTHER" /~}`, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "allowlist")
	assert.NotContains(t, out, "other-value")

	t.Run("a listed name the denylist blocks stays blocked", func(t *testing.T) {
		t.Setenv("EXONS_OPTIN_PROBE_TOKEN", "t")
		_, err := MustNew(WithEnvAllowlist([]string{"EXONS_OPTIN_*"})).
			Execute(ctx, `{~exons.env name="EXONS_OPTIN_PROBE_TOKEN" /~}`, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "denied")
	})

	t.Run("clearing the allowlist reverts to disabled", func(t *testing.T) {
		_, err := MustNew(WithEnvAllowlist([]string{envOptinVar}), WithEnvAllowlist(nil)).Execute(ctx, envOptinTag, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), envDisabledOptionHint)
	})
}

func TestEnvOptIn_WithEnvDisabledWinsInAnyOrder(t *testing.T) {
	t.Setenv(envOptinVar, envOptinValue)
	ctx := context.Background()

	orders := map[string][]Option{
		"disabled alone":             {WithEnvDisabled()},
		"disabled after enabled":     {WithEnvEnabled(), WithEnvDisabled()},
		"disabled before enabled":    {WithEnvDisabled(), WithEnvEnabled()},
		"disabled before allowlist":  {WithEnvDisabled(), WithEnvAllowlist([]string{envOptinVar})},
		"allowlist before disabled":  {WithEnvAllowlist([]string{envOptinVar}), WithEnvDisabled()},
		"denylist cleared, disabled": {WithEnvDenylist(nil), WithEnvEnabled(), WithEnvDisabled()},
	}
	for name, opts := range orders {
		t.Run(name, func(t *testing.T) {
			out, err := MustNew(opts...).Execute(ctx, envOptinTag, nil)
			require.Error(t, err)
			assert.NotContains(t, out, envOptinValue)
		})
	}
}
