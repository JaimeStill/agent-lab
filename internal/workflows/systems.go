package workflows

import (
	"log/slog"

	"github.com/JaimeStill/agent-lab/internal/agents"
	"github.com/JaimeStill/agent-lab/internal/documents"
	"github.com/JaimeStill/agent-lab/internal/images"
)

// Systems provides access to domain systems for workflow execution.
type Systems struct {
	Agents    agents.System
	Documents documents.System
	Images    images.System
	Logger    *slog.Logger
}
