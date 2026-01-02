package profiles

import (
	"errors"
	"net/http"
)

// Domain errors for profile operations.
var (
	ErrNotFound      = errors.New("profile not found")
	ErrDuplicate     = errors.New("profile name already exists for workflow")
	ErrStageNotFound = errors.New("stage not found")
)

// MapHTTPStatus maps domain errors to appropriate HTTP status codes.
func MapHTTPStatus(err error) int {
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrStageNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrDuplicate) {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}
