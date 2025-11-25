package internal_logger_test

import (
	"log/slog"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/logger"
)

func TestNew_JSONFormat(t *testing.T) {
	cfg := &config.LoggingConfig{
		Level:  config.LogLevelInfo,
		Format: config.LogFormatJSON,
	}

	sys := logger.New(cfg)
	if sys == nil {
		t.Fatal("New() returned nil")
	}

	log := sys.Logger()
	if log == nil {
		t.Fatal("Logger() returned nil")
	}
}

func TestNew_TextFormat(t *testing.T) {
	cfg := &config.LoggingConfig{
		Level:  config.LogLevelDebug,
		Format: config.LogFormatText,
	}

	sys := logger.New(cfg)
	if sys == nil {
		t.Fatal("New() returned nil")
	}

	log := sys.Logger()
	if log == nil {
		t.Fatal("Logger() returned nil")
	}
}

func TestLogger_NotNil(t *testing.T) {
	cfg := &config.LoggingConfig{
		Level:  config.LogLevelInfo,
		Format: config.LogFormatJSON,
	}

	sys := logger.New(cfg)
	log := sys.Logger()

	if log == nil {
		t.Error("Logger() should return non-nil *slog.Logger")
	}
}

func TestNew_AllLogLevels(t *testing.T) {
	tests := []struct {
		name  string
		level config.LogLevel
	}{
		{"debug", config.LogLevelDebug},
		{"info", config.LogLevelInfo},
		{"warn", config.LogLevelWarn},
		{"error", config.LogLevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.LoggingConfig{
				Level:  tt.level,
				Format: config.LogFormatJSON,
			}

			sys := logger.New(cfg)
			if sys == nil {
				t.Fatalf("New() with level %s returned nil", tt.level)
			}

			log := sys.Logger()
			if log == nil {
				t.Fatalf("Logger() with level %s returned nil", tt.level)
			}

			if !log.Enabled(nil, tt.level.ToSlogLevel()) {
				t.Errorf("Logger not enabled for configured level %s", tt.level)
			}
		})
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	cfg := &config.LoggingConfig{
		Level:  config.LogLevelWarn,
		Format: config.LogFormatJSON,
	}

	sys := logger.New(cfg)
	log := sys.Logger()

	if log.Enabled(nil, slog.LevelDebug) {
		t.Error("Logger should not be enabled for DEBUG when configured for WARN")
	}

	if log.Enabled(nil, slog.LevelInfo) {
		t.Error("Logger should not be enabled for INFO when configured for WARN")
	}

	if !log.Enabled(nil, slog.LevelWarn) {
		t.Error("Logger should be enabled for WARN when configured for WARN")
	}

	if !log.Enabled(nil, slog.LevelError) {
		t.Error("Logger should be enabled for ERROR when configured for WARN")
	}
}
