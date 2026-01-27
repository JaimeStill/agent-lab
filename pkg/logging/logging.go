// Package logging provides structured logging configuration and initialization.
// It wraps slog with configurable log levels and output formats.
package logging

import (
	"fmt"
	"log/slog"
	"os"
)

// New creates a configured slog.Logger based on the provided configuration.
// It returns a text or JSON handler based on the Format setting.
func New(cfg *Config) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: cfg.Level.ToSlogLevel(),
	}

	var handler slog.Handler
	if cfg.Format == FormatJSON {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

// Level represents a logging severity level.
type Level string

// Log level constants.
const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// Validate checks if the level is a valid logging level.
func (l Level) Validate() error {
	switch l {
	case LevelDebug, LevelInfo, LevelWarn, LevelError:
		return nil
	default:
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", l)
	}
}

// ToSlogLevel converts the Level to its slog.Level equivalent.
// Unknown levels default to slog.LevelInfo.
func (l Level) ToSlogLevel() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Format represents the log output format.
type Format string

// Log format constants.
const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Validate checks if the format is a valid logging format.
func (f Format) Validate() error {
	switch f {
	case FormatText, FormatJSON:
		return nil
	default:
		return fmt.Errorf("invalid log format: %s (must be text or json)", f)
	}
}
