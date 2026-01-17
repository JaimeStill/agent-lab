package api

import (
	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/agent-lab/pkg/runtime"
)

// Runtime extends Infrastructure with API-specific configuration.
type Runtime struct {
	*runtime.Infrastructure
	Pagination pagination.Config
}

// NewRuntime creates an API runtime with a module-scoped logger.
func NewRuntime(
	cfg *config.Config,
	infra *runtime.Infrastructure,
) *Runtime {
	return &Runtime{
		Infrastructure: &runtime.Infrastructure{
			Lifecycle: infra.Lifecycle,
			Logger:    infra.Logger.With("module", "api"),
			Database:  infra.Database,
			Storage:   infra.Storage,
		},
		Pagination: cfg.API.Pagination,
	}
}
