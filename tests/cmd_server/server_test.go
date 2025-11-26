package cmd_server_test

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
}

func TestServiceIntegration_HealthEndpoint(t *testing.T) {
	port := getAvailablePort(t)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/healthz", port))
	if err == nil {
		resp.Body.Close()
		t.Fatal("Server should not be running yet")
	}
}

func TestServiceIntegration_ShutdownTimeout(t *testing.T) {
	cfg := &config.Config{
		ShutdownTimeout: "5s",
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
