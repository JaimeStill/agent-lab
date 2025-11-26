package internal_config_test

import (
	"os"
	"testing"
	"time"

	"github.com/JaimeStill/agent-lab/internal/config"
)

func TestLoad_BaseConfig(t *testing.T) {
	os.Unsetenv("SERVICE_ENV")

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir("../../"); err != nil {
		t.Fatalf("Failed to change to repo root: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
}

func TestLoad_WithOverlay(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir("../../"); err != nil {
		t.Fatalf("Failed to change to repo root: %v", err)
	}

	testOverlay := `shutdown_timeout = "60s"

[server]
port = 9090
`

	if err := os.WriteFile("config.test.toml", []byte(testOverlay), 0644); err != nil {
		t.Fatalf("Failed to write test overlay: %v", err)
	}
	defer os.Remove("config.test.toml")

	os.Setenv("SERVICE_ENV", "test")
	defer os.Unsetenv("SERVICE_ENV")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() with overlay failed: %v", err)
	}

	if cfg.ShutdownTimeout != "60s" {
		t.Errorf("ShutdownTimeout = %q, want %q", cfg.ShutdownTimeout, "60s")
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 9090)
	}
}

func TestLoad_AppliesDefaults(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir("../../"); err != nil {
		t.Fatalf("Failed to change to repo root: %v", err)
	}

	os.Unsetenv("SERVICE_ENV")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.ShutdownTimeout == "" {
		t.Error("ShutdownTimeout not set to default")
	}

	if cfg.Server.Host == "" {
		t.Error("Server.Host not set to default")
	}

	if cfg.Server.Port == 0 {
		t.Error("Server.Port not set to default")
	}

	if cfg.Logging.Level == "" {
		t.Error("Logging.Level not set to default")
	}
}

func TestLoad_InvalidDuration(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir("../../"); err != nil {
		t.Fatalf("Failed to change to repo root: %v", err)
	}

	testOverlay := `shutdown_timeout = "invalid"`

	if err := os.WriteFile("config.invalid.toml", []byte(testOverlay), 0644); err != nil {
		t.Fatalf("Failed to write test overlay: %v", err)
	}
	defer os.Remove("config.invalid.toml")

	os.Setenv("SERVICE_ENV", "invalid")
	defer os.Unsetenv("SERVICE_ENV")

	_, err = config.Load()
	if err == nil {
		t.Error("Load() succeeded with invalid duration, want error")
	}
}

func TestLoad_EnvVarOverrides(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir("../../"); err != nil {
		t.Fatalf("Failed to change to repo root: %v", err)
	}

	os.Unsetenv("SERVICE_ENV")
	os.Setenv("SERVICE_SHUTDOWN_TIMEOUT", "120s")
	os.Setenv("SERVER_PORT", "3000")
	os.Setenv("LOGGING_LEVEL", "debug")
	defer func() {
		os.Unsetenv("SERVICE_SHUTDOWN_TIMEOUT")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("LOGGING_LEVEL")
	}()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.ShutdownTimeout != "120s" {
		t.Errorf("ShutdownTimeout = %q, want %q (env override)", cfg.ShutdownTimeout, "120s")
	}

	if cfg.Server.Port != 3000 {
		t.Errorf("Server.Port = %d, want %d (env override)", cfg.Server.Port, 3000)
	}

	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %q, want %q (env override)", cfg.Logging.Level, "debug")
	}
}

func TestMerge_RootConfig(t *testing.T) {
	base := &config.Config{}
	base.ShutdownTimeout = "30s"

	overlay := &config.Config{}
	overlay.ShutdownTimeout = "60s"

	base.Merge(overlay)

	if base.ShutdownTimeout != "60s" {
		t.Errorf("ShutdownTimeout = %q after merge, want %q", base.ShutdownTimeout, "60s")
	}
}

func TestShutdownTimeoutDuration(t *testing.T) {
	cfg := &config.Config{
		ShutdownTimeout: "45s",
	}

	duration := cfg.ShutdownTimeoutDuration()
	expected := 45 * time.Second

	if duration != expected {
		t.Errorf("ShutdownTimeoutDuration() = %v, want %v", duration, expected)
	}
}
