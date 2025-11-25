// Package logger provides structured logging configuration and initialization.
package logger

import (
	"log/slog"
	"os"

	"github.com/JaimeStill/agent-lab/internal/config"
)

// System provides access to the configured logger instance.
type System interface {
	Logger() *slog.Logger
}

type logger struct {
	logger *slog.Logger
}

// New creates a logger system with the specified configuration.
func New(cfg *config.LoggingConfig) System {
	opts := &slog.HandlerOptions{
		Level: cfg.Level.ToSlogLevel(),
	}

	var handler slog.Handler
	if cfg.Format == config.LogFormatJSON {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return &logger{
		logger: slog.New(handler),
	}
}

// Logger returns the configured slog.Logger instance.
func (l *logger) Logger() *slog.Logger {
	return l.logger
}
