package agents

import (
	"errors"
	"net/http"
)

// Domain errors for agent operations.
var (
	ErrNotFound      = errors.New("agent not found")
	ErrDuplicate     = errors.New("agent name already exists")
	ErrInvalidConfig = errors.New("invalid agent config")
	ErrExecution     = errors.New("agent execution failed")
)

// MapHTTPStatus maps domain errors to appropriate HTTP status codes.
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
	if errors.Is(err, ErrExecution) {
		return http.StatusBadGateway
	}
	return http.StatusInternalServerError
}
