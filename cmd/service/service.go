package main

import (
	"fmt"
	"time"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/database"
	"github.com/JaimeStill/agent-lab/internal/lifecycle"
	"github.com/JaimeStill/agent-lab/internal/logger"
	"github.com/JaimeStill/agent-lab/internal/routes"
	"github.com/JaimeStill/agent-lab/internal/server"
)

// Service coordinates the lifecycle of all subsystems.
type Service struct {
	lifecycle *lifecycle.Coordinator
	logger    logger.System
	database  database.System
	server    server.System
}

// NewService creates and initializes the service with all subsystems.
func NewService(cfg *config.Config) (*Service, error) {
	lc := lifecycle.New()
	loggerSys := logger.New(&cfg.Logging)
	routeSys := routes.New(loggerSys.Logger())

	dbSys, err := database.New(&cfg.Database, loggerSys.Logger())
	if err != nil {
		return nil, fmt.Errorf("database init failed: %w", err)
	}

	middlewareSys := buildMiddleware(loggerSys, cfg)
	registerRoutes(routeSys, lc)
	handler := middlewareSys.Apply(routeSys.Build())

	serverSys := server.New(&cfg.Server, handler, loggerSys.Logger())

	return &Service{
		lifecycle: lc,
		logger:    loggerSys,
		database:  dbSys,
		server:    serverSys,
	}, nil
}

// Start begins all subsystems and returns when they are ready.
func (s *Service) Start() error {
	s.logger.Logger().Info("starting service")

	if err := s.database.Start(s.lifecycle); err != nil {
		return fmt.Errorf("database start failed: %w", err)
	}

	if err := s.server.Start(s.lifecycle); err != nil {
		return fmt.Errorf("server start failed: %w", err)
	}

	go func() {
		s.lifecycle.WaitForStartup()
		s.logger.Logger().Info("all subsystems ready")
	}()

	s.logger.Logger().Info("service started")
	return nil
}

// Shutdown gracefully stops all subsystems within the provided context deadline.
func (s *Service) Shutdown(timeout time.Duration) error {
	s.logger.Logger().Info("initiating shutdown")

	if err := s.lifecycle.Shutdown(timeout); err != nil {
		return err
	}

	s.logger.Logger().Info("all subsystems shut down successfully")
	return nil
}
