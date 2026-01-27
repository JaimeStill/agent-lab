// Package infrastructure provides core service initialization for application startup.
// It assembles common dependencies (logging, database, storage) that domain systems require.
package infrastructure

import (
	"fmt"
	"log/slog"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/pkg/database"
	"github.com/JaimeStill/agent-lab/pkg/lifecycle"
	"github.com/JaimeStill/agent-lab/pkg/logging"
	"github.com/JaimeStill/agent-lab/pkg/storage"
)

// Infrastructure holds the core systems required by all domain modules.
// It provides a single point of initialization for lifecycle coordination,
// logging, database access, and file storage.
type Infrastructure struct {
	Lifecycle *lifecycle.Coordinator
	Logger    *slog.Logger
	Database  database.System
	Storage   storage.System
}

// New creates an Infrastructure from the application configuration.
// It initializes all systems but does not start them; call Start separately.
func New(cfg *config.Config) (*Infrastructure, error) {
	lc := lifecycle.New()
	logger := logging.New(&cfg.Logging)

	db, err := database.New(&cfg.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("database init failed: %w", err)
	}

	store, err := storage.New(&cfg.Storage, logger)
	if err != nil {
		return nil, fmt.Errorf("storage init failed: %w", err)
	}

	return &Infrastructure{
		Lifecycle: lc,
		Logger:    logger,
		Database:  db,
		Storage:   store,
	}, nil
}

// Start initializes all infrastructure systems and registers them with the lifecycle coordinator.
func (i *Infrastructure) Start() error {
	if err := i.Database.Start(i.Lifecycle); err != nil {
		return fmt.Errorf("database start failed: %w", err)
	}
	if err := i.Storage.Start(i.Lifecycle); err != nil {
		return fmt.Errorf("storage start failed: %w", err)
	}
	return nil
}
