package providers

import (
	"errors"
	"net/http"
)

// Domain errors for the providers system.
var (
	// ErrNotFound indicates the requested provider does not exist.
	ErrNotFound = errors.New("provider not found")

	// ErrDuplicate indicates a provider with the same name already exists.
	ErrDuplicate = errors.New("provider name already exists")

	// ErrInvalidConfig indicates the provider configuration failed validation.
	ErrInvalidConfig = errors.New("invalid provider config")
)

func MapHTTPStatus(err error) int {
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrDuplicate) {
		return http.StatusConflict
	}
	if errors.Is(err, ErrInvalidConfig) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
