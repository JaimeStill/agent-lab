package documents

import (
	"errors"
	"net/http"
)

// Domain errors for document operations.
var (
	ErrNotFound     = errors.New("document not found")
	ErrDuplicate    = errors.New("document storage key already exists")
	ErrFileTooLarge = errors.New("file exceeds maximum upload size")
	ErrInvalidFile  = errors.New("invalid file")
)

// MapHTTPStatus converts domain errors to appropriate HTTP status codes.
func MapHTTPStatus(err error) int {
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrDuplicate) {
		return http.StatusConflict
	}
	if errors.Is(err, ErrFileTooLarge) {
		return http.StatusRequestEntityTooLarge
	}
	if errors.Is(err, ErrInvalidFile) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
