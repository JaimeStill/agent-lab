package main

import (
	"fmt"
	"time"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/routes"
)

// Server coordinates the lifecycle of all subsystems.
type Server struct {
	runtime *Runtime
	domain  *Domain
	http    *httpServer
}

// NewServer creates and initializes the service with all subsystems.
func NewServer(cfg *config.Config) (*Server, error) {
	runtime, err := NewRuntime(cfg)
	if err != nil {
		return nil, err
	}

	domain := NewDomain(runtime)

	routeSys := routes.New(runtime.Logger)
	middlewareSys := buildMiddleware(runtime, cfg)

	registerRoutes(routeSys, runtime, domain)
	handler := middlewareSys.Apply(routeSys.Build())

	http := newHTTPServer(&cfg.Server, handler, runtime.Logger)
	return &Server{
		runtime: runtime,
		domain:  domain,
		http:    http,
	}, nil
}

// Start begins all subsystems and returns when they are ready.
func (s *Server) Start() error {
	s.runtime.Logger.Info("starting service")

	if err := s.runtime.Start(); err != nil {
		return err
	}

	if err := s.http.Start(s.runtime.Lifecycle); err != nil {
		return fmt.Errorf("http server start failed: %w", err)
	}

	go func() {
		s.runtime.Lifecycle.WaitForStartup()
		s.runtime.Logger.Info("all subsystems ready")
	}()

	return nil
}

// Shutdown gracefully stops all subsystems within the provided context deadline.
func (s *Server) Shutdown(timeout time.Duration) error {
	s.runtime.Logger.Info("initiating shutdown")
	return s.runtime.Lifecycle.Shutdown(timeout)
}
