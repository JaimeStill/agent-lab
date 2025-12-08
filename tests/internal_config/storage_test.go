package internal_config_test

import (
	"os"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/config"
)

func TestStorageConfig_Finalize_Defaults(t *testing.T) {
	cfg := &config.StorageConfig{}

	if err := cfg.Finalize(); err != nil {
		t.Fatalf("Finalize() failed: %v", err)
	}

	if cfg.BasePath != ".data/blobs" {
		t.Errorf("BasePath = %q, want %q", cfg.BasePath, ".data/blobs")
	}
}

func TestStorageConfig_Finalize_EnvOverride(t *testing.T) {
	os.Setenv("STORAGE_BASE_PATH", "/custom/path")
	defer os.Unsetenv("STORAGE_BASE_PATH")

	cfg := &config.StorageConfig{}

	if err := cfg.Finalize(); err != nil {
		t.Fatalf("Finalize() failed: %v", err)
	}

	if cfg.BasePath != "/custom/path" {
		t.Errorf("BasePath = %q, want %q", cfg.BasePath, "/custom/path")
	}
}

func TestStorageConfig_Finalize_PreservesExisting(t *testing.T) {
	cfg := &config.StorageConfig{
		BasePath: "/existing/path",
	}

	if err := cfg.Finalize(); err != nil {
		t.Fatalf("Finalize() failed: %v", err)
	}

	if cfg.BasePath != "/existing/path" {
		t.Errorf("BasePath = %q, want %q", cfg.BasePath, "/existing/path")
	}
}

func TestStorageConfig_Merge(t *testing.T) {
	tests := []struct {
		name     string
		base     config.StorageConfig
		overlay  config.StorageConfig
		expected string
	}{
		{
			name:     "overlay replaces base",
			base:     config.StorageConfig{BasePath: "/base"},
			overlay:  config.StorageConfig{BasePath: "/overlay"},
			expected: "/overlay",
		},
		{
			name:     "empty overlay preserves base",
			base:     config.StorageConfig{BasePath: "/base"},
			overlay:  config.StorageConfig{},
			expected: "/base",
		},
		{
			name:     "overlay sets empty base",
			base:     config.StorageConfig{},
			overlay:  config.StorageConfig{BasePath: "/overlay"},
			expected: "/overlay",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.base.Merge(&tt.overlay)
			if tt.base.BasePath != tt.expected {
				t.Errorf("BasePath = %q, want %q", tt.base.BasePath, tt.expected)
			}
		})
	}
}
