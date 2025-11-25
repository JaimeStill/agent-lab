package main

import (
	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/logger"
	"github.com/JaimeStill/agent-lab/internal/middleware"
)

// buildMiddleware creates and configures the middleware stack with logging and CORS.
func buildMiddleware(loggerSys logger.System, cfg *config.Config) middleware.System {
	middlewareSys := middleware.New()
	middlewareSys.Use(middleware.Logger(loggerSys.Logger()))
	middlewareSys.Use(middleware.CORS(&cfg.CORS))
	return middlewareSys
}
