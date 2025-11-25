package internal_config_test

import (
	"testing"
	"time"

	"github.com/JaimeStill/agent-lab/internal/config"
)

func TestServerConfig_Merge(t *testing.T) {
	base := &config.ServerConfig{
		Host:            "localhost",
		Port:            8080,
		ReadTimeout:     "30s",
		WriteTimeout:    "30s",
		ShutdownTimeout: "30s",
	}

	overlay := &config.ServerConfig{
		Port:         9090,
		WriteTimeout: "60s",
	}

	base.Merge(overlay)

	if base.Host != "localhost" {
		t.Errorf("Host = %q, want %q (should not change)", base.Host, "localhost")
	}

	if base.Port != 9090 {
		t.Errorf("Port = %d, want %d (should merge)", base.Port, 9090)
	}

	if base.ReadTimeout != "30s" {
		t.Errorf("ReadTimeout = %q, want %q (should not change)", base.ReadTimeout, "30s")
	}

	if base.WriteTimeout != "60s" {
		t.Errorf("WriteTimeout = %q, want %q (should merge)", base.WriteTimeout, "60s")
	}
}

func TestServerConfig_Addr(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		expected string
	}{
		{"default", "0.0.0.0", 8080, "0.0.0.0:8080"},
		{"localhost", "localhost", 3000, "localhost:3000"},
		{"ipv6", "::1", 8080, "::1:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ServerConfig{
				Host: tt.host,
				Port: tt.port,
			}

			addr := cfg.Addr()
			if addr != tt.expected {
				t.Errorf("Addr() = %q, want %q", addr, tt.expected)
			}
		})
	}
}

func TestServerConfig_DurationGetters(t *testing.T) {
	cfg := &config.ServerConfig{
		ReadTimeout:     "30s",
		WriteTimeout:    "60s",
		ShutdownTimeout: "90s",
	}

	tests := []struct {
		name     string
		got      time.Duration
		expected time.Duration
	}{
		{"ReadTimeout", cfg.ReadTimeoutDuration(), 30 * time.Second},
		{"WriteTimeout", cfg.WriteTimeoutDuration(), 60 * time.Second},
		{"ShutdownTimeout", cfg.ShutdownTimeoutDuration(), 90 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestServerConfig_Finalize(t *testing.T) {
	cfg := &config.ServerConfig{}

	if err := cfg.Finalize(); err != nil {
		t.Fatalf("Finalize() failed: %v", err)
	}

	if cfg.Host == "" {
		t.Error("Host not set to default")
	}

	if cfg.Port == 0 {
		t.Error("Port not set to default")
	}

	if cfg.ReadTimeout == "" {
		t.Error("ReadTimeout not set to default")
	}

	if cfg.WriteTimeout == "" {
		t.Error("WriteTimeout not set to default")
	}

	if cfg.ShutdownTimeout == "" {
		t.Error("ShutdownTimeout not set to default")
	}
}

func TestServerConfig_Validate_InvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"port negative", -1},
		{"port too high", 65536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ServerConfig{
				Host:            "localhost",
				Port:            tt.port,
				ReadTimeout:     "30s",
				WriteTimeout:    "30s",
				ShutdownTimeout: "30s",
			}

			err := cfg.Finalize()
			if err == nil {
				t.Error("Finalize() succeeded with invalid port, want error")
			}
		})
	}
}
