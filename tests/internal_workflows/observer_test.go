package internal_workflows_test

import (
	"log/slog"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/go-agents-orchestration/pkg/observability"
	"github.com/google/uuid"
)

func TestNewPostgresObserver(t *testing.T) {
	runID := uuid.New()
	logger := slog.Default()
	observer := workflows.NewPostgresObserver(nil, runID, logger)

	if observer == nil {
		t.Fatal("NewPostgresObserver() returned nil")
	}
}

func TestPostgresObserver_ImplementsInterface(t *testing.T) {
	runID := uuid.New()
	logger := slog.Default()
	observer := workflows.NewPostgresObserver(nil, runID, logger)

	var _ observability.Observer = observer
}
