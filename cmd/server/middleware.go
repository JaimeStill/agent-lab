package main

import (
	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/middleware"
)

// buildMiddleware creates and configures the middleware stack with logging and CORS.
func buildMiddleware(runtime *Runtime, cfg *config.Config) middleware.System {
	middlewareSys := middleware.New()
	middlewareSys.Use(middleware.TrimSlash())
	middlewareSys.Use(middleware.Logger(runtime.Logger))
	middlewareSys.Use(middleware.CORS(&cfg.CORS))
	return middlewareSys
}
