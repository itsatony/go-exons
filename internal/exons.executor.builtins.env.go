package internal

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// EnvConfig controls which environment variables the EnvResolver can access.
type EnvConfig struct {
	// Disabled completely disables the {~exons.env~} tag.
	Disabled bool
	// Allowlist restricts access to only variables matching these glob patterns.
	// If non-empty, only matching names are permitted (after denylist check).
	// Uses filepath.Match glob syntax (*, ?).
	Allowlist []string
	// Denylist blocks access to variables matching these glob patterns.
	// Checked before allowlist. Uses filepath.Match glob syntax (*, ?).
	Denylist []string
}

// EnvResolver handles the exons.env built-in tag.
// It retrieves environment variable values from the system,
// subject to allowlist/denylist filtering configured via EnvConfig.
//
// Usage:
//
//	{~exons.env name="API_KEY" /~}                   -> os.Getenv("API_KEY")
//	{~exons.env name="API_KEY" default="none" /~}   -> os.Getenv or "none"
//	{~exons.env name="MISSING" required="true" /~}  -> error if not set
type EnvResolver struct {
	config EnvConfig
}

// NewEnvResolver creates a new EnvResolver with the given configuration.
func NewEnvResolver(config EnvConfig) *EnvResolver {
	return &EnvResolver{config: config}
}

// TagName returns the tag name for this resolver.
func (r *EnvResolver) TagName() string {
	return TagNameEnv
}

// Resolve retrieves the environment variable value.
func (r *EnvResolver) Resolve(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error) {
	// Check if env access is disabled
	if r.config.Disabled {
		return "", NewBuiltinError(ErrMsgEnvDisabled, TagNameEnv)
	}

	// Get required 'name' attribute
	name, ok := attrs.Get(AttrName)
	if !ok {
		return "", NewBuiltinError(ErrMsgMissingNameAttr, TagNameEnv)
	}

	// Check access control
	if err := r.checkAccess(name); err != nil {
		return "", err
	}

	// Check if required flag is set
	requiredStr, hasRequired := attrs.Get(AttrRequired)
	isRequired := hasRequired && requiredStr == AttrValueTrue

	// Try to get the environment variable
	val := os.Getenv(name)

	// If empty, check default or return error
	if val == "" {
		// Check for default attribute
		if defaultVal, hasDefault := attrs.Get(AttrDefault); hasDefault {
			return defaultVal, nil
		}

		// If required and not set, return error
		if isRequired {
			return "", NewEnvVarRequiredError(name)
		}

		// Return empty string if not required and no default
		return "", nil
	}

	return val, nil
}

// checkAccess verifies the variable name against denylist and allowlist.
func (r *EnvResolver) checkAccess(name string) error {
	upperName := strings.ToUpper(name)

	// Check denylist first (takes priority)
	for _, pattern := range r.config.Denylist {
		upperPattern := strings.ToUpper(pattern)
		matched, err := filepath.Match(upperPattern, upperName)
		if err != nil {
			return NewEnvVarInvalidPatternError(pattern)
		}
		if matched {
			return NewEnvVarDeniedError(name)
		}
	}

	// If allowlist is set, name must match at least one pattern
	if len(r.config.Allowlist) > 0 {
		for _, pattern := range r.config.Allowlist {
			upperPattern := strings.ToUpper(pattern)
			matched, err := filepath.Match(upperPattern, upperName)
			if err != nil {
				return NewEnvVarInvalidPatternError(pattern)
			}
			if matched {
				return nil
			}
		}
		return NewEnvVarNotAllowedError(name)
	}

	return nil
}

// Validate checks that the required attributes are present.
func (r *EnvResolver) Validate(attrs Attributes) error {
	if !attrs.Has(AttrName) {
		return NewBuiltinError(ErrMsgMissingNameAttr, TagNameEnv)
	}
	return nil
}

// NewEnvVarNotFoundError creates an error for environment variable not found.
func NewEnvVarNotFoundError(varName string) *BuiltinError {
	return NewBuiltinError(ErrMsgEnvVarNotFound, TagNameEnv).
		WithMetadata(MetaKeyEnvVar, varName)
}

// NewEnvVarRequiredError creates an error for required environment variable not set.
func NewEnvVarRequiredError(varName string) *BuiltinError {
	return NewBuiltinError(ErrMsgEnvVarRequired, TagNameEnv).
		WithMetadata(MetaKeyEnvVar, varName)
}

// NewEnvVarDeniedError creates an error for denied environment variable access.
func NewEnvVarDeniedError(varName string) *BuiltinError {
	return NewBuiltinError(ErrMsgEnvVarDenied, TagNameEnv).
		WithMetadata(MetaKeyEnvVar, varName)
}

// NewEnvVarNotAllowedError creates an error for variable not in allowlist.
func NewEnvVarNotAllowedError(varName string) *BuiltinError {
	return NewBuiltinError(ErrMsgEnvVarNotInList, TagNameEnv).
		WithMetadata(MetaKeyEnvVar, varName)
}

// NewEnvVarInvalidPatternError creates an error for a malformed glob pattern.
func NewEnvVarInvalidPatternError(pattern string) *BuiltinError {
	return NewBuiltinError(ErrMsgEnvInvalidPattern, TagNameEnv).
		WithMetadata(MetaKeyEnvVar, pattern)
}
