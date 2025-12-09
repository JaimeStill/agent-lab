package internal_documents_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/documents"
)

func TestMapHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			"not found error",
			documents.ErrNotFound,
			http.StatusNotFound,
		},
		{
			"wrapped not found error",
			fmt.Errorf("failed: %w", documents.ErrNotFound),
			http.StatusNotFound,
		},
		{
			"duplicate error",
			documents.ErrDuplicate,
			http.StatusConflict,
		},
		{
			"wrapped duplicate error",
			fmt.Errorf("failed: %w", documents.ErrDuplicate),
			http.StatusConflict,
		},
		{
			"file too large error",
			documents.ErrFileTooLarge,
			http.StatusRequestEntityTooLarge,
		},
		{
			"wrapped file too large error",
			fmt.Errorf("failed: %w", documents.ErrFileTooLarge),
			http.StatusRequestEntityTooLarge,
		},
		{
			"invalid file error",
			documents.ErrInvalidFile,
			http.StatusBadRequest,
		},
		{
			"wrapped invalid file error",
			fmt.Errorf("failed: %w", documents.ErrInvalidFile),
			http.StatusBadRequest,
		},
		{
			"unknown error",
			errors.New("unknown error"),
			http.StatusInternalServerError,
		},
		{
			"nil-ish generic error",
			fmt.Errorf("something went wrong"),
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := documents.MapHTTPStatus(tt.err)
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
		{"ErrNotFound", documents.ErrNotFound, "document not found"},
		{"ErrDuplicate", documents.ErrDuplicate, "document storage key already exists"},
		{"ErrFileTooLarge", documents.ErrFileTooLarge, "file exceeds maximum upload size"},
		{"ErrInvalidFile", documents.ErrInvalidFile, "invalid file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.wantMsg {
				t.Errorf("error message = %q, want %q", tt.err.Error(), tt.wantMsg)
			}
		})
	}
}
