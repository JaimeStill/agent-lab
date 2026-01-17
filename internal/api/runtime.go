package api

import (
	"log/slog"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/pkg/database"
	"github.com/JaimeStill/agent-lab/pkg/lifecycle"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/agent-lab/pkg/storage"
)

type Runtime struct {
	Logger     *slog.Logger
	Database   database.System
	Storage    storage.System
	Lifecycle  *lifecycle.Coordinator
	Pagination pagination.Config
}

func NewRuntime(
	cfg *config.Config,
	logger *slog.Logger,
	db database.System,
	store storage.System,
	lc *lifecycle.Coordinator,
) *Runtime {
	return &Runtime{
		Logger:     logger.With("module", "api"),
		Database:   db,
		Storage:    store,
		Lifecycle:  lc,
		Pagination: cfg.API.Pagination,
	}
}
