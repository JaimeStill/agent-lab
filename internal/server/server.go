// Package server provides HTTP server lifecycle management with graceful shutdown.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/lifecycle"
)

// System manages the HTTP server lifecycle including startup and shutdown.
type System interface {
	Start(lc *lifecycle.Coordinator) error
}

type server struct {
	http            *http.Server
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

// New creates a server system with the specified configuration, handler, and logger.
func New(cfg *config.ServerConfig, handler http.Handler, logger *slog.Logger) System {
	return &server{
		http: &http.Server{
			Addr:         cfg.Addr(),
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeoutDuration(),
			WriteTimeout: cfg.WriteTimeoutDuration(),
		},
		logger:          logger,
		shutdownTimeout: cfg.ShutdownTimeoutDuration(),
	}
}

// Start begins listening for HTTP requests and sets up graceful shutdown on context cancellation.
func (s *server) Start(lc *lifecycle.Coordinator) error {
	go func() {
		s.logger.Info("server listening", "addr", s.http.Addr)
		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("server error", "error", err)
		}
	}()

	lc.OnShutdown(func() {
		<-lc.Context().Done()
		s.logger.Info("shutting down server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		if err := s.http.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("server shutdown error", "error", err)
		} else {
			s.logger.Info("server shutdown complete")
		}
	})

	return nil
}
