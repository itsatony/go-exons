package internal

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvResolver_TagName(t *testing.T) {
	resolver := NewEnvResolver()
	assert.Equal(t, TagNameEnv, resolver.TagName())
}

func TestEnvResolver_Validate(t *testing.T) {
	resolver := NewEnvResolver()

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
	resolver := NewEnvResolver()
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
	RegisterBuiltins(registry)

	assert.True(t, registry.Has(TagNameEnv), "EnvResolver should be registered")

	resolver, found := registry.Get(TagNameEnv)
	require.True(t, found)
	assert.NotNil(t, resolver)
}

func TestEnvResolver_ResolveWithSystemEnvVars(t *testing.T) {
	resolver := NewEnvResolver()
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
