package execution

import "github.com/itsatony/go-cuserr"

// NewConfigValidationError creates a validation error for execution config.
func NewConfigValidationError(msg string) error {
	return cuserr.NewValidationError(ErrCodeExecution, msg)
}

// NewProviderError creates a validation error with provider metadata.
func NewProviderError(msg, provider string) error {
	return cuserr.NewValidationError(ErrCodeExecution, msg).
		WithMetadata(MetaKeyProvider, provider)
}
