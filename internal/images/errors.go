// Package images provides document page rendering and image management capabilities.
// It supports rendering PDF pages to PNG or JPEG images with configurable options
// for DPI, quality, brightness, contrast, saturation, and rotation.
package images

import (
	"errors"
	"net/http"
)

// Domain errors for image operations.
var (
	ErrNotFound            = errors.New("image not found")
	ErrDuplicate           = errors.New("image already exists")
	ErrDocumentNotFound    = errors.New("document not found")
	ErrUnsupportedFormat   = errors.New("document format is not supported for rendering")
	ErrInvalidPageRange    = errors.New("invalid page range")
	ErrPageOutOfRange      = errors.New("page number out of range")
	ErrInvalidRenderOption = errors.New("invalid render option")
	ErrRenderFailed        = errors.New("render failed")
)

// MapHTTPStatus maps domain errors to appropriate HTTP status codes.
func MapHTTPStatus(err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrDuplicate):
		return http.StatusConflict
	case errors.Is(err, ErrDocumentNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrUnsupportedFormat):
		return http.StatusBadRequest
	case errors.Is(err, ErrInvalidPageRange):
		return http.StatusBadRequest
	case errors.Is(err, ErrPageOutOfRange):
		return http.StatusBadRequest
	case errors.Is(err, ErrInvalidRenderOption):
		return http.StatusBadRequest
	case errors.Is(err, ErrRenderFailed):
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
