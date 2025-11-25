// Package pagination provides types and utilities for paginated data queries.
package pagination

import (
	"fmt"
	"os"
	"strconv"
)

// Environment variable names for pagination configuration.
const (
	EnvPaginationDefaultPageSize = "PAGINATION_DEFAULT_PAGE_SIZE"
	EnvPaginationMaxPageSize     = "PAGINATION_MAX_PAGE_SIZE"
)

// Config holds pagination settings including default and maximum page sizes.
type Config struct {
	DefaultPageSize int `toml:"default_page_size"`
	MaxPageSize     int `toml:"max_page_size"`
}

// Finalize applies defaults, loads environment overrides, and validates the configuration.
func (c *Config) Finalize() error {
	c.loadDefaults()
	c.loadEnv()
	return c.validate()
}

// Merge applies non-zero values from overlay onto the receiver.
func (c *Config) Merge(overlay *Config) {
	if overlay.DefaultPageSize != 0 {
		c.DefaultPageSize = overlay.DefaultPageSize
	}
	if overlay.MaxPageSize != 0 {
		c.MaxPageSize = overlay.MaxPageSize
	}
}

func (c *Config) loadDefaults() {
	if c.DefaultPageSize <= 0 {
		c.DefaultPageSize = 20
	}
	if c.MaxPageSize <= 0 {
		c.MaxPageSize = 100
	}
}

func (c *Config) loadEnv() {
	if v := os.Getenv(EnvPaginationDefaultPageSize); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.DefaultPageSize = n
		}
	}
	if v := os.Getenv(EnvPaginationMaxPageSize); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxPageSize = n
		}
	}
}

func (c *Config) validate() error {
	if c.DefaultPageSize < 1 {
		return fmt.Errorf("default_page_size must be positive")
	}
	if c.MaxPageSize < 1 {
		return fmt.Errorf("max_page_size must be positive")
	}
	if c.DefaultPageSize > c.MaxPageSize {
		return fmt.Errorf("default_page_size cannot exceed max_page_size")
	}
	return nil
}
