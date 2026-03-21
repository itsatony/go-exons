package internal

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvResolver_TagName(t *testing.T) {
	resolver := NewEnvResolver(EnvConfig{})
	assert.Equal(t, TagNameEnv, resolver.TagName())
}

func TestEnvResolver_Validate(t *testing.T) {
	resolver := NewEnvResolver(EnvConfig{})

	tests := []struct {
		name    string
		attrs   Attributes
		wantErr bool
	}{
		{
			name:    "valid with name attribute",
			attrs:   Attributes{AttrName: "TEST_VAR"},
			wantErr: false,
		},
		{
			name:    "valid with name and default",
			attrs:   Attributes{AttrName: "TEST_VAR", AttrDefault: "fallback"},
			wantErr: false,
		},
		{
			name:    "missing name attribute",
			attrs:   Attributes{},
			wantErr: true,
		},
		{
			name:    "nil attributes",
			attrs:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := resolver.Validate(tt.attrs)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEnvResolver_Resolve(t *testing.T) {
	resolver := NewEnvResolver(EnvConfig{})
	ctx := context.Background()

	// Set up test environment variables
	testEnvVar := "EXONS_TEST_ENV_VAR"
	testEnvValue := "test_value_12345"
	t.Setenv(testEnvVar, testEnvValue)

	emptyEnvVar := "EXONS_TEST_EMPTY_VAR"
	t.Setenv(emptyEnvVar, "")

	tests := []struct {
		name       string
		attrs      Attributes
		want       string
		wantErr    bool
		errContain string
	}{
		{
			name:  "resolve existing env var",
			attrs: Attributes{AttrName: testEnvVar},
			want:  testEnvValue,
		},
		{
			name:  "resolve non-existent env var returns empty",
			attrs: Attributes{AttrName: "EXONS_NON_EXISTENT_VAR_12345"},
			want:  "",
		},
		{
			name:  "resolve non-existent env var with default",
			attrs: Attributes{AttrName: "EXONS_NON_EXISTENT_VAR_12345", AttrDefault: "default_val"},
			want:  "default_val",
		},
		{
			name:       "resolve non-existent required env var",
			attrs:      Attributes{AttrName: "EXONS_NON_EXISTENT_VAR_12345", AttrRequired: AttrValueTrue},
			wantErr:    true,
			errContain: ErrMsgEnvVarRequired,
		},
		{
			name:  "resolve empty env var returns empty",
			attrs: Attributes{AttrName: emptyEnvVar},
			want:  "",
		},
		{
			name:  "resolve empty env var with default uses default",
			attrs: Attributes{AttrName: emptyEnvVar, AttrDefault: "fallback"},
			want:  "fallback",
		},
		{
			name:       "missing name attribute",
			attrs:      Attributes{},
			wantErr:    true,
			errContain: ErrMsgMissingNameAttr,
		},
		{
			name:  "required=false does not error on missing",
			attrs: Attributes{AttrName: "EXONS_NON_EXISTENT_VAR_12345", AttrRequired: AttrValueFalse},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.Resolve(ctx, nil, tt.attrs)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestEnvResolver_Integration(t *testing.T) {
	// Test that EnvResolver is registered in builtins
	registry := NewRegistry(nil)
	RegisterBuiltins(registry, BuiltinConfig{})

	assert.True(t, registry.Has(TagNameEnv), "EnvResolver should be registered")

	resolver, found := registry.Get(TagNameEnv)
	require.True(t, found)
	assert.NotNil(t, resolver)
}

func TestEnvResolver_ResolveWithSystemEnvVars(t *testing.T) {
	resolver := NewEnvResolver(EnvConfig{})
	ctx := context.Background()

	// Test with PATH which should exist on all systems
	pathVal := os.Getenv("PATH")
	if pathVal != "" {
		attrs := Attributes{AttrName: "PATH"}
		got, err := resolver.Resolve(ctx, nil, attrs)
		require.NoError(t, err)
		assert.Equal(t, pathVal, got)
	}

	// Test with HOME which should exist on Unix systems
	homeVal := os.Getenv("HOME")
	if homeVal != "" {
		attrs := Attributes{AttrName: "HOME"}
		got, err := resolver.Resolve(ctx, nil, attrs)
		require.NoError(t, err)
		assert.Equal(t, homeVal, got)
	}
}

func TestNewEnvVarNotFoundError(t *testing.T) {
	err := NewEnvVarNotFoundError("MY_VAR")
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), ErrMsgEnvVarNotFound)
	assert.Contains(t, err.Error(), "MY_VAR")
}

func TestNewEnvVarRequiredError(t *testing.T) {
	err := NewEnvVarRequiredError("REQUIRED_VAR")
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), ErrMsgEnvVarRequired)
	assert.Contains(t, err.Error(), "REQUIRED_VAR")
}

// --- Tests for EnvConfig: Disabled, Denylist, Allowlist ---

func TestEnvResolver_Disabled(t *testing.T) {
	resolver := NewEnvResolver(EnvConfig{Disabled: true})
	ctx := context.Background()

	t.Setenv("EXONS_TEST_DISABLED_VAR", "should_not_see_this")

	_, err := resolver.Resolve(ctx, nil, Attributes{AttrName: "EXONS_TEST_DISABLED_VAR"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgEnvDisabled)
}

func TestEnvResolver_Disabled_AnyVar(t *testing.T) {
	resolver := NewEnvResolver(EnvConfig{Disabled: true})
	ctx := context.Background()

	// Even a non-existent var should return the disabled error, not a missing-var error
	_, err := resolver.Resolve(ctx, nil, Attributes{AttrName: "NONEXISTENT_DISABLED_TEST"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgEnvDisabled)
}

func TestEnvResolver_Denylist(t *testing.T) {
	ctx := context.Background()

	t.Run("var matching denylist pattern is denied", func(t *testing.T) {
		resolver := NewEnvResolver(EnvConfig{
			Denylist: []string{"*_KEY"},
		})
		t.Setenv("MY_SECRET_KEY", "secret123")

		_, err := resolver.Resolve(ctx, nil, Attributes{AttrName: "MY_SECRET_KEY"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgEnvVarDenied)
	})

	t.Run("var not matching denylist pattern is allowed", func(t *testing.T) {
		resolver := NewEnvResolver(EnvConfig{
			Denylist: []string{"*_KEY"},
		})
		t.Setenv("EXONS_TEST_SAFE_VAR", "safe_value")

		got, err := resolver.Resolve(ctx, nil, Attributes{AttrName: "EXONS_TEST_SAFE_VAR"})
		require.NoError(t, err)
		assert.Equal(t, "safe_value", got)
	})

	t.Run("case-insensitive denylist matching", func(t *testing.T) {
		resolver := NewEnvResolver(EnvConfig{
			Denylist: []string{"*_KEY"},
		})
		t.Setenv("my_key", "lower_case_secret")

		_, err := resolver.Resolve(ctx, nil, Attributes{AttrName: "my_key"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgEnvVarDenied)
	})

	t.Run("multiple denylist patterns first match wins", func(t *testing.T) {
		resolver := NewEnvResolver(EnvConfig{
			Denylist: []string{"*_TOKEN", "*_SECRET", "*_KEY"},
		})
		t.Setenv("API_TOKEN", "tok_value")
		t.Setenv("DB_SECRET", "secret_value")

		_, err := resolver.Resolve(ctx, nil, Attributes{AttrName: "API_TOKEN"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgEnvVarDenied)

		_, err = resolver.Resolve(ctx, nil, Attributes{AttrName: "DB_SECRET"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgEnvVarDenied)
	})
}

func TestEnvResolver_Allowlist(t *testing.T) {
	ctx := context.Background()

	t.Run("matching var is allowed", func(t *testing.T) {
		resolver := NewEnvResolver(EnvConfig{
			Allowlist: []string{"APP_*"},
		})
		t.Setenv("APP_NAME", "myapp")

		got, err := resolver.Resolve(ctx, nil, Attributes{AttrName: "APP_NAME"})
		require.NoError(t, err)
		assert.Equal(t, "myapp", got)
	})

	t.Run("non-matching var is denied", func(t *testing.T) {
		resolver := NewEnvResolver(EnvConfig{
			Allowlist: []string{"APP_*"},
		})
		t.Setenv("DB_HOST", "localhost")

		_, err := resolver.Resolve(ctx, nil, Attributes{AttrName: "DB_HOST"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgEnvVarNotInList)
	})

	t.Run("case-insensitive allowlist matching", func(t *testing.T) {
		resolver := NewEnvResolver(EnvConfig{
			Allowlist: []string{"APP_*"},
		})
		t.Setenv("app_config", "some_config")

		got, err := resolver.Resolve(ctx, nil, Attributes{AttrName: "app_config"})
		require.NoError(t, err)
		assert.Equal(t, "some_config", got)
	})

	t.Run("multiple allowlist patterns", func(t *testing.T) {
		resolver := NewEnvResolver(EnvConfig{
			Allowlist: []string{"APP_*", "SVC_*"},
		})
		t.Setenv("SVC_PORT", "8080")

		got, err := resolver.Resolve(ctx, nil, Attributes{AttrName: "SVC_PORT"})
		require.NoError(t, err)
		assert.Equal(t, "8080", got)
	})
}

func TestEnvResolver_DenylistPriorityOverAllowlist(t *testing.T) {
	ctx := context.Background()

	resolver := NewEnvResolver(EnvConfig{
		Denylist:  []string{"*_SECRET"},
		Allowlist: []string{"APP_*"},
	})
	t.Setenv("APP_SECRET", "should_be_denied")

	// APP_SECRET matches allowlist (APP_*) but also matches denylist (*_SECRET)
	// Denylist takes priority
	_, err := resolver.Resolve(ctx, nil, Attributes{AttrName: "APP_SECRET"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgEnvVarDenied)
}

func TestEnvResolver_DenylistAllowlistCombined_AllowedThrough(t *testing.T) {
	ctx := context.Background()

	resolver := NewEnvResolver(EnvConfig{
		Denylist:  []string{"*_SECRET"},
		Allowlist: []string{"APP_*"},
	})
	t.Setenv("APP_NAME", "myapp")

	// APP_NAME matches allowlist and does NOT match denylist — should succeed
	got, err := resolver.Resolve(ctx, nil, Attributes{AttrName: "APP_NAME"})
	require.NoError(t, err)
	assert.Equal(t, "myapp", got)
}

func TestEnvResolver_InvalidGlobPattern(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid denylist pattern", func(t *testing.T) {
		resolver := NewEnvResolver(EnvConfig{
			Denylist: []string{"[invalid"},
		})
		t.Setenv("ANYTHING", "val")

		_, err := resolver.Resolve(ctx, nil, Attributes{AttrName: "ANYTHING"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgEnvInvalidPattern)
	})

	t.Run("invalid allowlist pattern", func(t *testing.T) {
		resolver := NewEnvResolver(EnvConfig{
			Allowlist: []string{"[invalid"},
		})
		t.Setenv("ANYTHING", "val")

		_, err := resolver.Resolve(ctx, nil, Attributes{AttrName: "ANYTHING"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgEnvInvalidPattern)
	})
}

func TestEnvResolver_EmptyDenylistAllowsAll(t *testing.T) {
	ctx := context.Background()

	resolver := NewEnvResolver(EnvConfig{
		Denylist:  []string{},
		Allowlist: []string{},
	})
	t.Setenv("EXONS_TEST_OPEN_VAR", "open_value")

	got, err := resolver.Resolve(ctx, nil, Attributes{AttrName: "EXONS_TEST_OPEN_VAR"})
	require.NoError(t, err)
	assert.Equal(t, "open_value", got)
}

func TestEnvResolver_ErrorConstructors(t *testing.T) {
	t.Run("NewEnvVarDeniedError", func(t *testing.T) {
		err := NewEnvVarDeniedError("SECRET_KEY")
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), ErrMsgEnvVarDenied)
		assert.Contains(t, err.Error(), "SECRET_KEY")
	})

	t.Run("NewEnvVarNotAllowedError", func(t *testing.T) {
		err := NewEnvVarNotAllowedError("FORBIDDEN_VAR")
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), ErrMsgEnvVarNotInList)
		assert.Contains(t, err.Error(), "FORBIDDEN_VAR")
	})

	t.Run("NewEnvVarInvalidPatternError", func(t *testing.T) {
		err := NewEnvVarInvalidPatternError("[bad")
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), ErrMsgEnvInvalidPattern)
		assert.Contains(t, err.Error(), "[bad")
	})
}

func TestEnvResolver_IntegrationWithBuiltinConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("default config with no deny blocks nothing", func(t *testing.T) {
		// BuiltinConfig{} uses zero-value EnvConfig — no deny/allow patterns
		registry := NewRegistry(nil)
		RegisterBuiltins(registry, BuiltinConfig{})

		resolver, found := registry.Get(TagNameEnv)
		require.True(t, found)

		t.Setenv("EXONS_INT_TEST_ACCESS_TOKEN", "tok123")

		got, err := resolver.Resolve(ctx, nil, Attributes{AttrName: "EXONS_INT_TEST_ACCESS_TOKEN"})
		require.NoError(t, err)
		assert.Equal(t, "tok123", got)
	})

	t.Run("builtin config with denylist blocks matching vars", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry, BuiltinConfig{
			Env: EnvConfig{
				Denylist: []string{"*_TOKEN", "*_KEY", "*_SECRET"},
			},
		})

		resolver, found := registry.Get(TagNameEnv)
		require.True(t, found)

		t.Setenv("MY_API_TOKEN", "should_be_blocked")

		_, err := resolver.Resolve(ctx, nil, Attributes{AttrName: "MY_API_TOKEN"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgEnvVarDenied)
	})

	t.Run("builtin config with disabled blocks everything", func(t *testing.T) {
		registry := NewRegistry(nil)
		RegisterBuiltins(registry, BuiltinConfig{
			Env: EnvConfig{Disabled: true},
		})

		resolver, found := registry.Get(TagNameEnv)
		require.True(t, found)

		t.Setenv("HARMLESS_VAR", "val")

		_, err := resolver.Resolve(ctx, nil, Attributes{AttrName: "HARMLESS_VAR"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgEnvDisabled)
	})
}
