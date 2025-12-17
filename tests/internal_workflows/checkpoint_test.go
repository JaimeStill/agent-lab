package internal_workflows_test

import (
	"log/slog"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
)

func TestNewPostgresCheckpointStore(t *testing.T) {
	logger := slog.Default()
	store := workflows.NewPostgresCheckpointStore(nil, logger)

	if store == nil {
		t.Fatal("NewPostgresCheckpointStore() returned nil")
	}
}

func TestPostgresCheckpointStore_ImplementsInterface(t *testing.T) {
	logger := slog.Default()
	store := workflows.NewPostgresCheckpointStore(nil, logger)

	var _ state.CheckpointStore = store
}
