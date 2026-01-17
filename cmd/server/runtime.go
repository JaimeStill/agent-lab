package main

import (
	"fmt"
	"log/slog"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/pkg/database"
	"github.com/JaimeStill/agent-lab/pkg/lifecycle"
	"github.com/JaimeStill/agent-lab/pkg/storage"
)

type Runtime struct {
	Lifecycle *lifecycle.Coordinator
	Logger    *slog.Logger
	Database  database.System
	Storage   storage.System
}

func NewRuntime(cfg *config.Config) (*Runtime, error) {
	lc := lifecycle.New()
	logger := newLogger(&cfg.Logging)

	dbSys, err := database.New(&cfg.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("database init failed: %w", err)
	}

	storageSys, err := storage.New(&cfg.Storage, logger)
	if err != nil {
		return nil, fmt.Errorf("storage init failed: %w", err)
	}

	return &Runtime{
		Lifecycle: lc,
		Logger:    logger,
		Database:  dbSys,
		Storage:   storageSys,
	}, nil
}

func (r *Runtime) Start() error {
	if err := r.Database.Start(r.Lifecycle); err != nil {
		return fmt.Errorf("database start failed: %w", err)
	}

	if err := r.Storage.Start(r.Lifecycle); err != nil {
		return fmt.Errorf("storage start failed: %w", err)
	}
	return nil
}
