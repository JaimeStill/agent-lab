package internal_config_test

import (
	"testing"

	"github.com/JaimeStill/agent-lab/internal/config"
)

func TestLogLevel_ToSlogLevel(t *testing.T) {
	tests := []struct {
		level    config.LogLevel
		expected string
	}{
		{config.LogLevelDebug, "DEBUG"},
		{config.LogLevelInfo, "INFO"},
		{config.LogLevelWarn, "WARN"},
		{config.LogLevelError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			slogLevel := tt.level.ToSlogLevel()
			if slogLevel.String() != tt.expected {
				t.Errorf("ToSlogLevel() = %q, want %q", slogLevel.String(), tt.expected)
			}
		})
	}
}

func TestLogLevel_Validate(t *testing.T) {
	validLevels := []config.LogLevel{
		config.LogLevelDebug,
		config.LogLevelInfo,
		config.LogLevelWarn,
		config.LogLevelError,
	}

	for _, level := range validLevels {
		t.Run(string(level), func(t *testing.T) {
			if err := level.Validate(); err != nil {
				t.Errorf("Validate() failed for valid level %q: %v", level, err)
			}
		})
	}

	invalidLevel := config.LogLevel("invalid")
	if err := invalidLevel.Validate(); err == nil {
		t.Error("Validate() succeeded for invalid level, want error")
	}
}

func TestLogFormat_Validate(t *testing.T) {
	validFormats := []config.LogFormat{
		config.LogFormatText,
		config.LogFormatJSON,
	}

	for _, format := range validFormats {
		t.Run(string(format), func(t *testing.T) {
			if err := format.Validate(); err != nil {
				t.Errorf("Validate() failed for valid format %q: %v", format, err)
			}
		})
	}

	invalidFormat := config.LogFormat("invalid")
	if err := invalidFormat.Validate(); err == nil {
		t.Error("Validate() succeeded for invalid format, want error")
	}
}
