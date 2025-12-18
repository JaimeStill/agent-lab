package workflows

import (
	"log/slog"

	"github.com/JaimeStill/agent-lab/internal/agents"
	"github.com/JaimeStill/agent-lab/internal/documents"
	"github.com/JaimeStill/agent-lab/internal/images"
)

// Runtime aggregates runtime dependencies for workflow execution.
// It provides access to domain systems that workflow factories need
// to interact with during execution.
type Runtime struct {
	agents    agents.System
	documents documents.System
	images    images.System
	logger    *slog.Logger
}

// NewRuntime creates a new Runtime with the provided dependencies.
func NewRuntime(
	agents agents.System,
	documents documents.System,
	images images.System,
	logger *slog.Logger,
) *Runtime {
	return &Runtime{
		agents:    agents,
		documents: documents,
		images:    images,
		logger:    logger,
	}
}

// Agents returns the agents system for LLM operations.
func (r *Runtime) Agents() agents.System { return r.agents }

// Documents returns the documents system for file operations.
func (r *Runtime) Documents() documents.System { return r.documents }

// Images returns the images system for image operations.
func (r *Runtime) Images() images.System { return r.images }

// Logger returns the logger for workflow logging.
func (r *Runtime) Logger() *slog.Logger { return r.logger }
