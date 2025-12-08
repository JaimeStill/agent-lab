package config

import (
	"fmt"
	"os"
)

const (
	// EnvStorageBasePath overrides the storage base path.
	EnvStorageBasePath = "STORAGE_BASE_PATH"
)

// StorageConfig contains blob storage configuration.
type StorageConfig struct {
	// BasePath is the root directory for filesystem storage.
	// Default: ".data/blobs"
	BasePath string `toml:"base_path"`
}

// Finalize applies defaults, loads environment overrides, and validates the storage configuration.
func (c *StorageConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()
	return c.validate()
}

// Merge applies values from overlay configuration that differ from zero values.
func (c *StorageConfig) Merge(overlay *StorageConfig) {
	if overlay.BasePath != "" {
		c.BasePath = overlay.BasePath
	}
}

func (c *StorageConfig) loadDefaults() {
	if c.BasePath == "" {
		c.BasePath = ".data/blobs"
	}
}

func (c *StorageConfig) loadEnv() {
	if v := os.Getenv(EnvStorageBasePath); v != "" {
		c.BasePath = v
	}
}

func (c *StorageConfig) validate() error {
	if c.BasePath == "" {
		return fmt.Errorf("base_path required")
	}
	return nil
}
