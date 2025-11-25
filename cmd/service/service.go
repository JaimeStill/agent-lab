package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/logger"
	"github.com/JaimeStill/agent-lab/internal/routes"
	"github.com/JaimeStill/agent-lab/internal/server"
)

// Service coordinates the lifecycle of all subsystems.
type Service struct {
	ctx        context.Context
	cancel     context.CancelFunc
	shutdownWg sync.WaitGroup

	logger logger.System
	server server.System
}

// NewService creates and initializes the service with all subsystems.
func NewService(cfg *config.Config) (*Service, error) {
	ctx, cancel := context.WithCancel(context.Background())

	loggerSys := logger.New(&cfg.Logging)
	routeSys := routes.New(loggerSys.Logger())

	middlewareSys := buildMiddleware(loggerSys, cfg)
	registerRoutes(routeSys)
	handler := middlewareSys.Apply(routeSys.Build())

	serverSys := server.New(&cfg.Server, handler, loggerSys.Logger())

	return &Service{
		ctx:    ctx,
		cancel: cancel,
		logger: loggerSys,
		server: serverSys,
	}, nil
}

// Start begins all subsystems and returns when they are ready.
func (s *Service) Start() error {
	s.logger.Logger().Info("starting service")

	if err := s.server.Start(s.ctx, &s.shutdownWg); err != nil {
		return fmt.Errorf("server start failed: %w", err)
	}

	s.logger.Logger().Info("service started")
	return nil
}

// Shutdown gracefully stops all subsystems within the provided context deadline.
func (s *Service) Shutdown(ctx context.Context) error {
	s.logger.Logger().Info("initiating shutdown")

	s.cancel()

	done := make(chan struct{})
	go func() {
		s.shutdownWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Logger().Info("all subsystems shut down successfully")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout: %w", ctx.Err())
	}
}
