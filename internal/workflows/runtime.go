package workflows

import (
	"log/slog"

	"github.com/JaimeStill/agent-lab/internal/agents"
	"github.com/JaimeStill/agent-lab/internal/documents"
	"github.com/JaimeStill/agent-lab/internal/images"
	"github.com/JaimeStill/agent-lab/internal/profiles"
)

// Runtime aggregates runtime dependencies for workflow execution.
// It provides access to domain systems that workflow factories need
// to interact with during execution.
type Runtime struct {
	agents    agents.System
	documents documents.System
	images    images.System
	profiles  profiles.System
	logger    *slog.Logger
}

// NewRuntime creates a new Runtime with the provided dependencies.
func NewRuntime(
	agents agents.System,
	documents documents.System,
	images images.System,
	profiles profiles.System,
	logger *slog.Logger,
) *Runtime {
	return &Runtime{
		agents:    agents,
		documents: documents,
		images:    images,
		profiles:  profiles,
		logger:    logger,
	}
}

// Agents returns the agents system for LLM operations.
func (r *Runtime) Agents() agents.System { return r.agents }

// Documents returns the documents system for file operations.
func (r *Runtime) Documents() documents.System { return r.documents }

// Images returns the images system for image operations.
func (r *Runtime) Images() images.System { return r.images }

func (r *Runtime) Profiles() profiles.System { return r.profiles }

// Logger returns the logger for workflow logging.
func (r *Runtime) Logger() *slog.Logger { return r.logger }
