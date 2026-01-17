package storage

import (
	"fmt"
	"os"

	"github.com/docker/go-units"
)

// Config contains blob storage configuration.
type Config struct {
	// BasePath is the root directory for filesystem storage.
	// Default: ".data/blobs"
	BasePath         string `toml:"base_path"`
	MaxUploadSize    string `toml:"max_upload_size"`
	maxUploadSizeVal int64
}

type Env struct {
	BasePath      string
	MaxUploadSize string
}

func (c *Config) MaxUploadSizeBytes() int64 {
	return c.maxUploadSizeVal
}

// Finalize applies defaults, loads environment overrides, and validates the storage configuration.
func (c *Config) Finalize(env *Env) error {
	c.loadDefaults()
	if env != nil {
		c.loadEnv(env)
	}
	return c.validate()
}

// Merge applies values from overlay configuration that differ from zero values.
func (c *Config) Merge(overlay *Config) {
	if overlay.BasePath != "" {
		c.BasePath = overlay.BasePath
	}

	if size, err := units.FromHumanSize(overlay.MaxUploadSize); err == nil {
		c.MaxUploadSize = overlay.MaxUploadSize
		c.maxUploadSizeVal = size
	}
}

func (c *Config) loadDefaults() {
	if c.BasePath == "" {
		c.BasePath = ".data/blobs"
	}
	if c.MaxUploadSize == "" {
		c.MaxUploadSize = "100MB"
	}
}

func (c *Config) loadEnv(env *Env) {
	if env.BasePath != "" {
		if v := os.Getenv(env.BasePath); v != "" {
			c.BasePath = v
		}
	}
	if env.MaxUploadSize != "" {
		if v := os.Getenv(env.MaxUploadSize); v != "" {
			c.MaxUploadSize = v
		}
	}
}

func (c *Config) validate() error {
	if c.BasePath == "" {
		return fmt.Errorf("base_path required")
	}

	size, err := units.FromHumanSize(c.MaxUploadSize)
	if err != nil {
		return fmt.Errorf("invalid max_upload_size: %w", err)
	}
	if size <= 0 {
		return fmt.Errorf("max_upload_size must be positive")
	}
	c.maxUploadSizeVal = size

	return nil
}
