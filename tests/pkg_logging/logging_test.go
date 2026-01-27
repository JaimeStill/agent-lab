package pkg_logging_test

import (
	"log/slog"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/logging"
)

func TestLevel_ToSlogLevel(t *testing.T) {
	tests := []struct {
		level    logging.Level
		expected slog.Level
	}{
		{logging.LevelDebug, slog.LevelDebug},
		{logging.LevelInfo, slog.LevelInfo},
		{logging.LevelWarn, slog.LevelWarn},
		{logging.LevelError, slog.LevelError},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			got := tt.level.ToSlogLevel()
			if got != tt.expected {
				t.Errorf("ToSlogLevel() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLevel_ToSlogLevel_DefaultsToInfo(t *testing.T) {
	invalid := logging.Level("unknown")
	got := invalid.ToSlogLevel()
	if got != slog.LevelInfo {
		t.Errorf("ToSlogLevel() for unknown level = %v, want %v (default)", got, slog.LevelInfo)
	}
}

func TestLevel_Validate(t *testing.T) {
	validLevels := []logging.Level{
		logging.LevelDebug,
		logging.LevelInfo,
		logging.LevelWarn,
		logging.LevelError,
	}

	for _, level := range validLevels {
		t.Run(string(level), func(t *testing.T) {
			if err := level.Validate(); err != nil {
				t.Errorf("Validate() failed for valid level %q: %v", level, err)
			}
		})
	}
}

func TestLevel_Validate_Invalid(t *testing.T) {
	invalid := logging.Level("invalid")
	if err := invalid.Validate(); err == nil {
		t.Error("Validate() succeeded for invalid level, want error")
	}
}

func TestFormat_Validate(t *testing.T) {
	validFormats := []logging.Format{
		logging.FormatText,
		logging.FormatJSON,
	}

	for _, format := range validFormats {
		t.Run(string(format), func(t *testing.T) {
			if err := format.Validate(); err != nil {
				t.Errorf("Validate() failed for valid format %q: %v", format, err)
			}
		})
	}
}

func TestFormat_Validate_Invalid(t *testing.T) {
	invalid := logging.Format("invalid")
	if err := invalid.Validate(); err == nil {
		t.Error("Validate() succeeded for invalid format, want error")
	}
}

func TestNew_ReturnsLogger(t *testing.T) {
	cfg := &logging.Config{
		Level:  logging.LevelInfo,
		Format: logging.FormatText,
	}

	logger := logging.New(cfg)
	if logger == nil {
		t.Fatal("New() returned nil logger")
	}
}

func TestNew_JSONFormat(t *testing.T) {
	cfg := &logging.Config{
		Level:  logging.LevelDebug,
		Format: logging.FormatJSON,
	}

	logger := logging.New(cfg)
	if logger == nil {
		t.Fatal("New() returned nil logger")
	}
}
