package cmd_service_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/JaimeStill/agent-lab/internal/config"
)

func getAvailablePort(t *testing.T) int {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port
}

func TestServiceIntegration_ConfigLoad(t *testing.T) {
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
		t.Fatalf("config.Load() failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("config.Load() returned nil")
	}

	cfg.Database.Name = "test_db"
	cfg.Database.User = "test_user"

	if err := cfg.Finalize(); err != nil {
		t.Fatalf("config.Finalize() failed: %v", err)
	}
}

func TestServiceIntegration_HealthEndpoint(t *testing.T) {
	port := getAvailablePort(t)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:            "localhost",
			Port:            port,
			ReadTimeout:     "5s",
			WriteTimeout:    "5s",
			ShutdownTimeout: "5s",
		},
		Logging: config.LoggingConfig{
			Level:  config.LogLevelError,
			Format: config.LogFormatJSON,
		},
		CORS: config.CORSConfig{
			Enabled: false,
		},
		ShutdownTimeout: "5s",
	}

	cfg.Database.Name = "test_db"
	cfg.Database.User = "test_user"

	if err := cfg.Finalize(); err != nil {
		t.Fatalf("config.Finalize() failed: %v", err)
	}

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/healthz", port))
	if err == nil {
		resp.Body.Close()
		t.Fatal("Server should not be running yet")
	}
}

func TestServiceIntegration_CORSHeaders(t *testing.T) {
	port := getAvailablePort(t)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:            "localhost",
			Port:            port,
			ReadTimeout:     "5s",
			WriteTimeout:    "5s",
			ShutdownTimeout: "5s",
		},
		Logging: config.LoggingConfig{
			Level:  config.LogLevelError,
			Format: config.LogFormatJSON,
		},
		CORS: config.CORSConfig{
			Enabled:        true,
			Origins:        []string{"http://example.com"},
			AllowedMethods: []string{"GET", "POST"},
			AllowedHeaders: []string{"Content-Type"},
		},
		ShutdownTimeout: "5s",
	}

	cfg.Database.Name = "test_db"
	cfg.Database.User = "test_user"

	if err := cfg.Finalize(); err != nil {
		t.Fatalf("config.Finalize() failed: %v", err)
	}
}

func TestServiceIntegration_ComponentIntegration(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:            "localhost",
			Port:            8080,
			ReadTimeout:     "30s",
			WriteTimeout:    "30s",
			ShutdownTimeout: "30s",
		},
		Logging: config.LoggingConfig{
			Level:  config.LogLevelInfo,
			Format: config.LogFormatJSON,
		},
		CORS: config.CORSConfig{
			Enabled: false,
		},
		ShutdownTimeout: "30s",
	}

	cfg.Database.Name = "test_db"
	cfg.Database.User = "test_user"

	if err := cfg.Finalize(); err != nil {
		t.Fatalf("config.Finalize() failed: %v", err)
	}
}

func TestServiceIntegration_ShutdownTimeout(t *testing.T) {
	cfg := &config.Config{
		ShutdownTimeout: "5s",
	}

	cfg.Database.Name = "test_db"
	cfg.Database.User = "test_user"

	if err := cfg.Finalize(); err != nil {
		t.Fatalf("config.Finalize() failed: %v", err)
	}

	duration := cfg.ShutdownTimeoutDuration()
	expected := 5 * time.Second

	if duration != expected {
		t.Errorf("ShutdownTimeoutDuration() = %v, want %v", duration, expected)
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	select {
	case <-ctx.Done():
	case <-time.After(6 * time.Second):
		t.Error("Timeout context did not fire within expected duration")
	}
}
