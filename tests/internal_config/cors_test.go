package internal_config_test

import (
	"testing"

	"github.com/JaimeStill/agent-lab/internal/config"
)

func TestCORSConfig_Merge_Arrays(t *testing.T) {
	base := &config.CORSConfig{
		Origins:        []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
	}

	overlay := &config.CORSConfig{
		Origins:        []string{"http://example.com", "http://other.com"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	}

	base.Merge(overlay)

	if len(base.Origins) != 2 {
		t.Errorf("Origins length = %d, want %d (atomic replacement)", len(base.Origins), 2)
	}

	if base.Origins[0] != "http://example.com" {
		t.Errorf("Origins[0] = %q, want %q", base.Origins[0], "http://example.com")
	}

	if len(base.AllowedMethods) != 2 {
		t.Errorf("AllowedMethods length = %d, want %d (should not change)", len(base.AllowedMethods), 2)
	}

	if len(base.AllowedHeaders) != 2 {
		t.Errorf("AllowedHeaders length = %d, want %d (atomic replacement)", len(base.AllowedHeaders), 2)
	}
}

func TestCORSConfig_Merge_Booleans(t *testing.T) {
	base := &config.CORSConfig{
		Enabled:          true,
		AllowCredentials: true,
	}

	overlay := &config.CORSConfig{
		Enabled:          false,
		AllowCredentials: false,
	}

	base.Merge(overlay)

	if base.Enabled != false {
		t.Error("Enabled should be false after merge (boolean merge)")
	}

	if base.AllowCredentials != false {
		t.Error("AllowCredentials should be false after merge (boolean merge)")
	}
}

func TestCORSConfig_Merge_MaxAge(t *testing.T) {
	base := &config.CORSConfig{
		MaxAge: 3600,
	}

	overlay := &config.CORSConfig{
		MaxAge: 0,
	}

	base.Merge(overlay)

	if base.MaxAge != 0 {
		t.Errorf("MaxAge = %d, want %d (zero is valid)", base.MaxAge, 0)
	}
}

func TestCORSConfig_Finalize(t *testing.T) {
	cfg := &config.CORSConfig{}

	if err := cfg.Finalize(); err != nil {
		t.Fatalf("Finalize() failed: %v", err)
	}
}
