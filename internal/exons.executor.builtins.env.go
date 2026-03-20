package internal

import (
	"context"
	"os"
)

// EnvResolver handles the exons.env built-in tag.
// It retrieves environment variable values from the system.
//
// Usage:
//
//	{~exons.env name="API_KEY" /~}                   -> os.Getenv("API_KEY")
//	{~exons.env name="API_KEY" default="none" /~}   -> os.Getenv or "none"
//	{~exons.env name="MISSING" required="true" /~}  -> error if not set
type EnvResolver struct{}

// NewEnvResolver creates a new EnvResolver.
func NewEnvResolver() *EnvResolver {
	return &EnvResolver{}
}

// TagName returns the tag name for this resolver.
func (r *EnvResolver) TagName() string {
	return TagNameEnv
}

// Resolve retrieves the environment variable value.
func (r *EnvResolver) Resolve(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error) {
	// Get required 'name' attribute
	name, ok := attrs.Get(AttrName)
	if !ok {
		return "", NewBuiltinError(ErrMsgMissingNameAttr, TagNameEnv)
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
