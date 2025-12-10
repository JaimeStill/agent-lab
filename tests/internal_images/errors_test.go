package internal_images_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/images"
)

func TestMapHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			"not found error",
			images.ErrNotFound,
			http.StatusNotFound,
		},
		{
			"wrapped not found error",
			fmt.Errorf("failed: %w", images.ErrNotFound),
			http.StatusNotFound,
		},
		{
			"duplicate error",
			images.ErrDuplicate,
			http.StatusConflict,
		},
		{
			"wrapped duplicate error",
			fmt.Errorf("failed: %w", images.ErrDuplicate),
			http.StatusConflict,
		},
		{
			"document not found error",
			images.ErrDocumentNotFound,
			http.StatusNotFound,
		},
		{
			"wrapped document not found error",
			fmt.Errorf("failed: %w", images.ErrDocumentNotFound),
			http.StatusNotFound,
		},
		{
			"unsupported format error",
			images.ErrUnsupportedFormat,
			http.StatusBadRequest,
		},
		{
			"wrapped unsupported format error",
			fmt.Errorf("failed: %w", images.ErrUnsupportedFormat),
			http.StatusBadRequest,
		},
		{
			"invalid page range error",
			images.ErrInvalidPageRange,
			http.StatusBadRequest,
		},
		{
			"wrapped invalid page range error",
			fmt.Errorf("failed: %w", images.ErrInvalidPageRange),
			http.StatusBadRequest,
		},
		{
			"page out of range error",
			images.ErrPageOutOfRange,
			http.StatusBadRequest,
		},
		{
			"wrapped page out of range error",
			fmt.Errorf("failed: %w", images.ErrPageOutOfRange),
			http.StatusBadRequest,
		},
		{
			"invalid render option error",
			images.ErrInvalidRenderOption,
			http.StatusBadRequest,
		},
		{
			"wrapped invalid render option error",
			fmt.Errorf("failed: %w", images.ErrInvalidRenderOption),
			http.StatusBadRequest,
		},
		{
			"render failed error",
			images.ErrRenderFailed,
			http.StatusInternalServerError,
		},
		{
			"wrapped render failed error",
			fmt.Errorf("failed: %w", images.ErrRenderFailed),
			http.StatusInternalServerError,
		},
		{
			"unknown error",
			errors.New("unknown error"),
			http.StatusInternalServerError,
		},
		{
			"generic error",
			fmt.Errorf("something went wrong"),
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := images.MapHTTPStatus(tt.err)
			if got != tt.wantStatus {
				t.Errorf("MapHTTPStatus() = %d, want %d", got, tt.wantStatus)
			}
		})
	}
}

func TestErrorValues(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{"ErrNotFound", images.ErrNotFound, "image not found"},
		{"ErrDuplicate", images.ErrDuplicate, "image already exists"},
		{"ErrDocumentNotFound", images.ErrDocumentNotFound, "document not found"},
		{"ErrUnsupportedFormat", images.ErrUnsupportedFormat, "document format is not supported for rendering"},
		{"ErrInvalidPageRange", images.ErrInvalidPageRange, "invalid page range"},
		{"ErrPageOutOfRange", images.ErrPageOutOfRange, "page number out of range"},
		{"ErrInvalidRenderOption", images.ErrInvalidRenderOption, "invalid render option"},
		{"ErrRenderFailed", images.ErrRenderFailed, "render failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.wantMsg {
				t.Errorf("error message = %q, want %q", tt.err.Error(), tt.wantMsg)
			}
		})
	}
}
