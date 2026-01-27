package pkg_logging_test

import (
	"os"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/logging"
)

func TestConfig_Merge(t *testing.T) {
	base := &logging.Config{
		Level:  logging.LevelInfo,
		Format: logging.FormatJSON,
	}

	overlay := &logging.Config{
		Level: logging.LevelDebug,
	}

	base.Merge(overlay)

	if base.Level != logging.LevelDebug {
		t.Errorf("Level = %q, want %q (should merge)", base.Level, logging.LevelDebug)
	}

	if base.Format != logging.FormatJSON {
		t.Errorf("Format = %q, want %q (should not change)", base.Format, logging.FormatJSON)
	}
}

func TestConfig_Merge_EmptyOverlay(t *testing.T) {
	base := &logging.Config{
		Level:  logging.LevelWarn,
		Format: logging.FormatText,
	}

	overlay := &logging.Config{}

	base.Merge(overlay)

	if base.Level != logging.LevelWarn {
		t.Errorf("Level = %q, want %q (should not change)", base.Level, logging.LevelWarn)
	}

	if base.Format != logging.FormatText {
		t.Errorf("Format = %q, want %q (should not change)", base.Format, logging.FormatText)
	}
}

func TestConfig_Finalize_AppliesDefaults(t *testing.T) {
	cfg := &logging.Config{}

	if err := cfg.Finalize(nil); err != nil {
		t.Fatalf("Finalize() failed: %v", err)
	}

	if cfg.Level != logging.LevelInfo {
		t.Errorf("Level = %q, want %q (default)", cfg.Level, logging.LevelInfo)
	}

	if cfg.Format != logging.FormatText {
		t.Errorf("Format = %q, want %q (default)", cfg.Format, logging.FormatText)
	}
}

func TestConfig_Finalize_PreservesValues(t *testing.T) {
	cfg := &logging.Config{
		Level:  logging.LevelDebug,
		Format: logging.FormatJSON,
	}

	if err := cfg.Finalize(nil); err != nil {
		t.Fatalf("Finalize() failed: %v", err)
	}

	if cfg.Level != logging.LevelDebug {
		t.Errorf("Level = %q, want %q (should preserve)", cfg.Level, logging.LevelDebug)
	}

	if cfg.Format != logging.FormatJSON {
		t.Errorf("Format = %q, want %q (should preserve)", cfg.Format, logging.FormatJSON)
	}
}

func TestConfig_Finalize_EnvOverrides(t *testing.T) {
	os.Setenv("TEST_LOG_LEVEL", "error")
	os.Setenv("TEST_LOG_FORMAT", "json")
	defer func() {
		os.Unsetenv("TEST_LOG_LEVEL")
		os.Unsetenv("TEST_LOG_FORMAT")
	}()

	env := &logging.Env{
		Level:  "TEST_LOG_LEVEL",
		Format: "TEST_LOG_FORMAT",
	}

	cfg := &logging.Config{
		Level:  logging.LevelInfo,
		Format: logging.FormatText,
	}

	if err := cfg.Finalize(env); err != nil {
		t.Fatalf("Finalize() failed: %v", err)
	}

	if cfg.Level != logging.LevelError {
		t.Errorf("Level = %q, want %q (env override)", cfg.Level, logging.LevelError)
	}

	if cfg.Format != logging.FormatJSON {
		t.Errorf("Format = %q, want %q (env override)", cfg.Format, logging.FormatJSON)
	}
}

func TestConfig_Finalize_InvalidLevel(t *testing.T) {
	cfg := &logging.Config{
		Level:  "invalid",
		Format: logging.FormatJSON,
	}

	err := cfg.Finalize(nil)
	if err == nil {
		t.Error("Finalize() succeeded with invalid level, want error")
	}
}

func TestConfig_Finalize_InvalidFormat(t *testing.T) {
	cfg := &logging.Config{
		Level:  logging.LevelInfo,
		Format: "invalid",
	}

	err := cfg.Finalize(nil)
	if err == nil {
		t.Error("Finalize() succeeded with invalid format, want error")
	}
}
