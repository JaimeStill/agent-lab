package internal_config_test

import (
	"testing"

	"github.com/JaimeStill/agent-lab/internal/config"
)

func TestLoggingConfig_Merge(t *testing.T) {
	base := &config.LoggingConfig{
		Level:  config.LogLevelInfo,
		Format: config.LogFormatJSON,
	}

	overlay := &config.LoggingConfig{
		Level: config.LogLevelDebug,
	}

	base.Merge(overlay)

	if base.Level != config.LogLevelDebug {
		t.Errorf("Level = %q, want %q (should merge)", base.Level, config.LogLevelDebug)
	}

	if base.Format != config.LogFormatJSON {
		t.Errorf("Format = %q, want %q (should not change)", base.Format, config.LogFormatJSON)
	}
}

func TestLoggingConfig_Finalize(t *testing.T) {
	cfg := &config.LoggingConfig{}

	if err := cfg.Finalize(); err != nil {
		t.Fatalf("Finalize() failed: %v", err)
	}

	if cfg.Level == "" {
		t.Error("Level not set to default")
	}

	if cfg.Format == "" {
		t.Error("Format not set to default")
	}
}

func TestLoggingConfig_Validate_InvalidLevel(t *testing.T) {
	cfg := &config.LoggingConfig{
		Level:  "invalid",
		Format: config.LogFormatJSON,
	}

	err := cfg.Finalize()
	if err == nil {
		t.Error("Finalize() succeeded with invalid level, want error")
	}
}

func TestLoggingConfig_Validate_InvalidFormat(t *testing.T) {
	cfg := &config.LoggingConfig{
		Level:  config.LogLevelInfo,
		Format: "invalid",
	}

	err := cfg.Finalize()
	if err == nil {
		t.Error("Finalize() succeeded with invalid format, want error")
	}
}
