package internal_workflows_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/workflows"
)

func TestMapHTTPStatus(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		want   int
	}{
		{"ErrNotFound", workflows.ErrNotFound, http.StatusNotFound},
		{"ErrWorkflowNotFound", workflows.ErrWorkflowNotFound, http.StatusNotFound},
		{"ErrInvalidStatus", workflows.ErrInvalidStatus, http.StatusBadRequest},
		{"wrapped ErrNotFound", fmt.Errorf("wrapped: %w", workflows.ErrNotFound), http.StatusNotFound},
		{"unknown error", errors.New("unknown"), http.StatusInternalServerError},
		{"nil error", nil, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := workflows.MapHTTPStatus(tt.err)
			if got != tt.want {
				t.Errorf("MapHTTPStatus() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestErrors_AreDistinct(t *testing.T) {
	if errors.Is(workflows.ErrNotFound, workflows.ErrWorkflowNotFound) {
		t.Error("ErrNotFound should not be ErrWorkflowNotFound")
	}

	if errors.Is(workflows.ErrNotFound, workflows.ErrInvalidStatus) {
		t.Error("ErrNotFound should not be ErrInvalidStatus")
	}

	if errors.Is(workflows.ErrWorkflowNotFound, workflows.ErrInvalidStatus) {
		t.Error("ErrWorkflowNotFound should not be ErrInvalidStatus")
	}
}
