package workflows

import (
	"errors"
	"net/http"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrWorkflowNotFound = errors.New("workflow not registered")
	ErrInvalidStatus    = errors.New("invalid status transition")
)

func MapHTTPStatus(err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrWorkflowNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrInvalidStatus):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
