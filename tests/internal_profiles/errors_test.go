package internal_profiles_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/profiles"
)

func TestMapHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			"not found error",
			profiles.ErrNotFound,
			http.StatusNotFound,
		},
		{
			"wrapped not found error",
			fmt.Errorf("failed: %w", profiles.ErrNotFound),
			http.StatusNotFound,
		},
		{
			"duplicate error",
			profiles.ErrDuplicate,
			http.StatusConflict,
		},
		{
			"wrapped duplicate error",
			fmt.Errorf("failed: %w", profiles.ErrDuplicate),
			http.StatusConflict,
		},
		{
			"stage not found error",
			profiles.ErrStageNotFound,
			http.StatusNotFound,
		},
		{
			"wrapped stage not found error",
			fmt.Errorf("failed: %w", profiles.ErrStageNotFound),
			http.StatusNotFound,
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
			got := profiles.MapHTTPStatus(tt.err)
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
		{"ErrNotFound", profiles.ErrNotFound, "profile not found"},
		{"ErrDuplicate", profiles.ErrDuplicate, "profile name already exists for workflow"},
		{"ErrStageNotFound", profiles.ErrStageNotFound, "stage not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.wantMsg {
				t.Errorf("error message = %q, want %q", tt.err.Error(), tt.wantMsg)
			}
		})
	}
}
