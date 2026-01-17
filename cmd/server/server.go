package main

import (
	"time"

	"github.com/JaimeStill/agent-lab/internal/config"
	_ "github.com/JaimeStill/agent-lab/workflows"
)

// Server coordinates the lifecycle of all subsystems.
type Server struct {
	runtime *Runtime
	modules *Modules
	http    *httpServer
}

// NewServer creates and initializes the service with all subsystems.
func NewServer(cfg *config.Config) (*Server, error) {
	runtime, err := NewRuntime(cfg)
	if err != nil {
		return nil, err
	}

	modules, err := NewModules(runtime, cfg)
	if err != nil {
		return nil, err
	}

	router := buildRouter(runtime)
	modules.Mount(router)

	runtime.Logger.Info(
		"server initialized",
		"addr", cfg.Server.Addr(),
		"version", cfg.Version,
	)

	return &Server{
		runtime: runtime,
		modules: modules,
		http:    newHTTPServer(&cfg.Server, router, runtime.Logger),
	}, nil
}

// Start begins all subsystems and returns when they are ready.
func (s *Server) Start() error {
	s.runtime.Logger.Info("starting service")

	if err := s.runtime.Start(); err != nil {
		return err
	}

	if err := s.http.Start(s.runtime.Lifecycle); err != nil {
		return err
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
