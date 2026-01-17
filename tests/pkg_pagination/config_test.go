package pkg_pagination_test

import (
	"os"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
)

func TestConfig_Finalize_Defaults(t *testing.T) {
	cfg := &pagination.Config{}

	if err := cfg.Finalize(nil); err != nil {
		t.Fatalf("Finalize() failed: %v", err)
	}

	if cfg.DefaultPageSize != 20 {
		t.Errorf("DefaultPageSize = %d, want 20", cfg.DefaultPageSize)
	}

	if cfg.MaxPageSize != 100 {
		t.Errorf("MaxPageSize = %d, want 100", cfg.MaxPageSize)
	}
}

func TestConfig_Finalize_PreservesValues(t *testing.T) {
	cfg := &pagination.Config{
		DefaultPageSize: 10,
		MaxPageSize:     50,
	}

	if err := cfg.Finalize(nil); err != nil {
		t.Fatalf("Finalize() failed: %v", err)
	}

	if cfg.DefaultPageSize != 10 {
		t.Errorf("DefaultPageSize = %d, want 10", cfg.DefaultPageSize)
	}

	if cfg.MaxPageSize != 50 {
		t.Errorf("MaxPageSize = %d, want 50", cfg.MaxPageSize)
	}
}

func TestConfig_Finalize_EnvOverrides(t *testing.T) {
	os.Setenv("TEST_PAGINATION_DEFAULT_PAGE_SIZE", "15")
	os.Setenv("TEST_PAGINATION_MAX_PAGE_SIZE", "75")
	defer func() {
		os.Unsetenv("TEST_PAGINATION_DEFAULT_PAGE_SIZE")
		os.Unsetenv("TEST_PAGINATION_MAX_PAGE_SIZE")
	}()

	cfg := &pagination.Config{}
	env := &pagination.ConfigEnv{
		DefaultPageSize: "TEST_PAGINATION_DEFAULT_PAGE_SIZE",
		MaxPageSize:     "TEST_PAGINATION_MAX_PAGE_SIZE",
	}

	if err := cfg.Finalize(env); err != nil {
		t.Fatalf("Finalize() failed: %v", err)
	}

	if cfg.DefaultPageSize != 15 {
		t.Errorf("DefaultPageSize = %d, want 15 (env override)", cfg.DefaultPageSize)
	}

	if cfg.MaxPageSize != 75 {
		t.Errorf("MaxPageSize = %d, want 75 (env override)", cfg.MaxPageSize)
	}
}

func TestConfig_Finalize_ValidationErrors(t *testing.T) {
	tests := []struct {
		name            string
		defaultPageSize int
		maxPageSize     int
	}{
		{"default exceeds max", 50, 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &pagination.Config{
				DefaultPageSize: tt.defaultPageSize,
				MaxPageSize:     tt.maxPageSize,
			}

			err := cfg.Finalize(nil)
			if err == nil {
				t.Error("Finalize() succeeded, want error")
			}
		})
	}
}

func TestConfig_Merge(t *testing.T) {
	tests := []struct {
		name                string
		base                pagination.Config
		overlay             pagination.Config
		wantDefaultPageSize int
		wantMaxPageSize     int
	}{
		{
			name:                "overlay overrides base",
			base:                pagination.Config{DefaultPageSize: 20, MaxPageSize: 100},
			overlay:             pagination.Config{DefaultPageSize: 10, MaxPageSize: 50},
			wantDefaultPageSize: 10,
			wantMaxPageSize:     50,
		},
		{
			name:                "zero values do not override",
			base:                pagination.Config{DefaultPageSize: 20, MaxPageSize: 100},
			overlay:             pagination.Config{DefaultPageSize: 0, MaxPageSize: 0},
			wantDefaultPageSize: 20,
			wantMaxPageSize:     100,
		},
		{
			name:                "partial overlay",
			base:                pagination.Config{DefaultPageSize: 20, MaxPageSize: 100},
			overlay:             pagination.Config{DefaultPageSize: 15, MaxPageSize: 0},
			wantDefaultPageSize: 15,
			wantMaxPageSize:     100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.base.Merge(&tt.overlay)

			if tt.base.DefaultPageSize != tt.wantDefaultPageSize {
				t.Errorf("DefaultPageSize = %d, want %d", tt.base.DefaultPageSize, tt.wantDefaultPageSize)
			}

			if tt.base.MaxPageSize != tt.wantMaxPageSize {
				t.Errorf("MaxPageSize = %d, want %d", tt.base.MaxPageSize, tt.wantMaxPageSize)
			}
		})
	}
}
