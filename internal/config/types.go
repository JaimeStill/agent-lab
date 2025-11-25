package config

import (
	"fmt"
	"log/slog"
)

// LogLevel represents the minimum severity level for log output.
type LogLevel string

const (
	// LogLevelDebug enables all log levels including debug messages.
	LogLevelDebug LogLevel = "debug"

	// LogLevelInfo enables info, warn, and error messages.
	LogLevelInfo LogLevel = "info"

	// LogLevelWarn enables warn and error messages.
	LogLevelWarn LogLevel = "warn"

	// LogLevelError enables only error messages.
	LogLevelError LogLevel = "error"
)

// Validate checks if the log level is one of the recognized values.
func (l LogLevel) Validate() error {
	switch l {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
		return nil
	default:
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", l)
	}
}

// ToSlogLevel converts the log level to its corresponding slog.Level value.
func (l LogLevel) ToSlogLevel() slog.Level {
	switch l {
	case LogLevelDebug:
		return slog.LevelDebug
	case LogLevelWarn:
		return slog.LevelWarn
	case LogLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// LogFormat represents the output format for log messages.
type LogFormat string

const (
	// LogFormatText outputs logs in human-readable text format.
	LogFormatText LogFormat = "text"

	// LogFormatJSON outputs logs in JSON format for structured logging.
	LogFormatJSON LogFormat = "json"
)

// Validate checks if the log format is one of the recognized values.
func (f LogFormat) Validate() error {
	switch f {
	case LogFormatText, LogFormatJSON:
		return nil
	default:
		return fmt.Errorf("invalid log format: %s (must be text or json)", f)
	}
}
