package config

import (
	"fmt"
	"os"

	"github.com/docker/go-units"
)

const (
	// EnvStorageBasePath overrides the storage base path.
	EnvStorageBasePath      = "STORAGE_BASE_PATH"
	EnvStorageMaxUploadSize = "STORAGE_MAX_UPLOAD_SIZE"
)

// StorageConfig contains blob storage configuration.
type StorageConfig struct {
	// BasePath is the root directory for filesystem storage.
	// Default: ".data/blobs"
	BasePath         string `toml:"base_path"`
	MaxUploadSize    string `toml:"max_upload_size"`
	maxUploadSizeVal int64
}

func (c *StorageConfig) MaxUploadSizeBytes() int64 {
	return c.maxUploadSizeVal
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

	if size, err := units.FromHumanSize(overlay.MaxUploadSize); err == nil {
		c.MaxUploadSize = overlay.MaxUploadSize
		c.maxUploadSizeVal = size
	}
}

func (c *StorageConfig) loadDefaults() {
	if c.BasePath == "" {
		c.BasePath = ".data/blobs"
	}
	if c.MaxUploadSize == "" {
		c.MaxUploadSize = "100MB"
	}
}

func (c *StorageConfig) loadEnv() {
	if v := os.Getenv(EnvStorageBasePath); v != "" {
		c.BasePath = v
	}
	if v := os.Getenv(EnvStorageMaxUploadSize); v != "" {
		c.MaxUploadSize = v
	}
}

func (c *StorageConfig) validate() error {
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
