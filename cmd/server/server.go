package main

import (
	"time"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/pkg/runtime"
	_ "github.com/JaimeStill/agent-lab/workflows"
)

// Server coordinates the lifecycle of all subsystems.
type Server struct {
	infra   *runtime.Infrastructure
	modules *Modules
	http    *httpServer
}

// NewServer creates and initializes the service with all subsystems.
func NewServer(cfg *config.Config) (*Server, error) {
	infra, err := runtime.New(cfg)
	if err != nil {
		return nil, err
	}

	modules, err := NewModules(infra, cfg)
	if err != nil {
		return nil, err
	}

	router := buildRouter(infra)
	modules.Mount(router)

	infra.Logger.Info(
		"server initialized",
		"addr", cfg.Server.Addr(),
		"version", cfg.Version,
	)

	return &Server{
		infra:   infra,
		modules: modules,
		http:    newHTTPServer(&cfg.Server, router, infra.Logger),
	}, nil
}

// Start begins all subsystems and returns when they are ready.
func (s *Server) Start() error {
	s.infra.Logger.Info("starting service")

	if err := s.infra.Start(); err != nil {
		return err
	}

	if err := s.http.Start(s.infra.Lifecycle); err != nil {
		return err
	}

	go func() {
		s.infra.Lifecycle.WaitForStartup()
		s.infra.Logger.Info("all subsystems ready")
	}()

	return nil
}

// Shutdown gracefully stops all subsystems within the provided context deadline.
func (s *Server) Shutdown(timeout time.Duration) error {
	s.infra.Logger.Info("initiating shutdown")
	return s.infra.Lifecycle.Shutdown(timeout)
}
