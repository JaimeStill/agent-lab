package pkg_middleware_test

import (
	"os"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/middleware"
)

func TestCORSConfig_Finalize_Defaults(t *testing.T) {
	cfg := &middleware.CORSConfig{}

	if err := cfg.Finalize(nil); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}

	if len(cfg.AllowedMethods) == 0 {
		t.Error("AllowedMethods should have defaults")
	}

	if len(cfg.AllowedHeaders) == 0 {
		t.Error("AllowedHeaders should have defaults")
	}

	if cfg.MaxAge <= 0 {
		t.Error("MaxAge should have default value")
	}
}

func TestCORSConfig_Finalize_PreservesValues(t *testing.T) {
	cfg := &middleware.CORSConfig{
		Enabled:          true,
		Origins:          []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET"},
		AllowedHeaders:   []string{"Authorization"},
		AllowCredentials: true,
		MaxAge:           7200,
	}

	if err := cfg.Finalize(nil); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}

	if !cfg.Enabled {
		t.Error("Enabled should be preserved")
	}

	if len(cfg.Origins) != 1 || cfg.Origins[0] != "http://localhost:3000" {
		t.Error("Origins should be preserved")
	}

	if len(cfg.AllowedMethods) != 1 || cfg.AllowedMethods[0] != "GET" {
		t.Error("AllowedMethods should be preserved")
	}

	if len(cfg.AllowedHeaders) != 1 || cfg.AllowedHeaders[0] != "Authorization" {
		t.Error("AllowedHeaders should be preserved")
	}

	if !cfg.AllowCredentials {
		t.Error("AllowCredentials should be preserved")
	}

	if cfg.MaxAge != 7200 {
		t.Errorf("MaxAge = %d, want 7200", cfg.MaxAge)
	}
}

func TestCORSConfig_Finalize_EnvOverrides(t *testing.T) {
	os.Setenv("TEST_CORS_ENABLED", "true")
	os.Setenv("TEST_CORS_ORIGINS", "http://localhost:3000, http://localhost:8080")
	os.Setenv("TEST_CORS_METHODS", "GET, POST")
	os.Setenv("TEST_CORS_HEADERS", "Content-Type, Authorization")
	os.Setenv("TEST_CORS_CREDENTIALS", "true")
	os.Setenv("TEST_CORS_MAX_AGE", "7200")
	defer func() {
		os.Unsetenv("TEST_CORS_ENABLED")
		os.Unsetenv("TEST_CORS_ORIGINS")
		os.Unsetenv("TEST_CORS_METHODS")
		os.Unsetenv("TEST_CORS_HEADERS")
		os.Unsetenv("TEST_CORS_CREDENTIALS")
		os.Unsetenv("TEST_CORS_MAX_AGE")
	}()

	cfg := &middleware.CORSConfig{}
	env := &middleware.CORSEnv{
		Enabled:          "TEST_CORS_ENABLED",
		Origins:          "TEST_CORS_ORIGINS",
		AllowedMethods:   "TEST_CORS_METHODS",
		AllowedHeaders:   "TEST_CORS_HEADERS",
		AllowCredentials: "TEST_CORS_CREDENTIALS",
		MaxAge:           "TEST_CORS_MAX_AGE",
	}

	if err := cfg.Finalize(env); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}

	if !cfg.Enabled {
		t.Error("Enabled should be true from env")
	}

	if len(cfg.Origins) != 2 {
		t.Errorf("Origins length = %d, want 2", len(cfg.Origins))
	}

	if len(cfg.AllowedMethods) != 2 {
		t.Errorf("AllowedMethods length = %d, want 2", len(cfg.AllowedMethods))
	}

	if len(cfg.AllowedHeaders) != 2 {
		t.Errorf("AllowedHeaders length = %d, want 2", len(cfg.AllowedHeaders))
	}

	if !cfg.AllowCredentials {
		t.Error("AllowCredentials should be true from env")
	}

	if cfg.MaxAge != 7200 {
		t.Errorf("MaxAge = %d, want 7200", cfg.MaxAge)
	}
}

func TestCORSConfig_Merge(t *testing.T) {
	tests := []struct {
		name               string
		base               middleware.CORSConfig
		overlay            middleware.CORSConfig
		wantEnabled        bool
		wantOriginsLen     int
		wantMethodsLen     int
		wantCredentials    bool
	}{
		{
			name: "overlay overrides all",
			base: middleware.CORSConfig{
				Enabled:          false,
				Origins:          []string{"http://a.com"},
				AllowedMethods:   []string{"GET"},
				AllowCredentials: false,
			},
			overlay: middleware.CORSConfig{
				Enabled:          true,
				Origins:          []string{"http://b.com", "http://c.com"},
				AllowedMethods:   []string{"GET", "POST", "PUT"},
				AllowCredentials: true,
			},
			wantEnabled:     true,
			wantOriginsLen:  2,
			wantMethodsLen:  3,
			wantCredentials: true,
		},
		{
			name: "nil slices preserve base",
			base: middleware.CORSConfig{
				Origins:        []string{"http://a.com"},
				AllowedMethods: []string{"GET", "POST"},
			},
			overlay: middleware.CORSConfig{
				Origins:        nil,
				AllowedMethods: nil,
			},
			wantOriginsLen: 1,
			wantMethodsLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.base.Merge(&tt.overlay)

			if tt.base.Enabled != tt.wantEnabled {
				t.Errorf("Enabled = %v, want %v", tt.base.Enabled, tt.wantEnabled)
			}

			if len(tt.base.Origins) != tt.wantOriginsLen {
				t.Errorf("Origins length = %d, want %d", len(tt.base.Origins), tt.wantOriginsLen)
			}

			if len(tt.base.AllowedMethods) != tt.wantMethodsLen {
				t.Errorf("AllowedMethods length = %d, want %d", len(tt.base.AllowedMethods), tt.wantMethodsLen)
			}

			if tt.base.AllowCredentials != tt.wantCredentials {
				t.Errorf("AllowCredentials = %v, want %v", tt.base.AllowCredentials, tt.wantCredentials)
			}
		})
	}
}
