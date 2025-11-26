package providers

import "errors"

// Domain errors for the providers system.
var (
	// ErrNotFound indicates the requested provider does not exist.
	ErrNotFound = errors.New("provider not found")

	// ErrDuplicate indicates a provider with the same name already exists.
	ErrDuplicate = errors.New("provider name already exists")

	// ErrInvalidConfig indicates the provider configuration failed validation.
	ErrInvalidConfig = errors.New("invalid provider config")
)
