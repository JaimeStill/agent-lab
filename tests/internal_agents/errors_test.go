package internal_agents_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/agents"
)

func TestMapHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			"not found error",
			agents.ErrNotFound,
			http.StatusNotFound,
		},
		{
			"wrapped not found error",
			fmt.Errorf("failed: %w", agents.ErrNotFound),
			http.StatusNotFound,
		},
		{
			"duplicate error",
			agents.ErrDuplicate,
			http.StatusConflict,
		},
		{
			"wrapped duplicate error",
			fmt.Errorf("failed: %w", agents.ErrDuplicate),
			http.StatusConflict,
		},
		{
			"invalid config error",
			agents.ErrInvalidConfig,
			http.StatusBadRequest,
		},
		{
			"wrapped invalid config error",
			fmt.Errorf("failed: %w", agents.ErrInvalidConfig),
			http.StatusBadRequest,
		},
		{
			"execution error",
			agents.ErrExecution,
			http.StatusBadGateway,
		},
		{
			"wrapped execution error",
			fmt.Errorf("failed: %w", agents.ErrExecution),
			http.StatusBadGateway,
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
			got := agents.MapHTTPStatus(tt.err)
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
		{"ErrNotFound", agents.ErrNotFound, "agent not found"},
		{"ErrDuplicate", agents.ErrDuplicate, "agent name already exists"},
		{"ErrInvalidConfig", agents.ErrInvalidConfig, "invalid agent config"},
		{"ErrExecution", agents.ErrExecution, "agent execution failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.wantMsg {
				t.Errorf("error message = %q, want %q", tt.err.Error(), tt.wantMsg)
			}
		})
	}
}
