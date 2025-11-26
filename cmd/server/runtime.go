package main

import (
	"fmt"
	"log/slog"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/database"
	"github.com/JaimeStill/agent-lab/internal/lifecycle"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
)

type Runtime struct {
	Lifecycle  *lifecycle.Coordinator
	Logger     *slog.Logger
	Database   database.System
	Pagination pagination.Config
}

func NewRuntime(cfg *config.Config) (*Runtime, error) {
	lc := lifecycle.New()
	logger := newLogger(&cfg.Logging)

	dbSys, err := database.New(&cfg.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("database init failed: %w", err)
	}

	return &Runtime{
		Lifecycle:  lc,
		Logger:     logger,
		Database:   dbSys,
		Pagination: cfg.Pagination,
	}, nil
}

func (r *Runtime) Start() error {
	if err := r.Database.Start(r.Lifecycle); err != nil {
		return fmt.Errorf("database start failed: %w", err)
	}
	return nil
}
