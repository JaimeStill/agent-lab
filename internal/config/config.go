// Package config provides application configuration management with support for
// TOML files, environment variable overrides, and configuration overlays.
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/pelletier/go-toml/v2"
)

const (
	// BaseConfigFile is the primary configuration file name.
	BaseConfigFile = "config.toml"

	// OverlayConfigPattern is the file name pattern for environment-specific overlays.
	OverlayConfigPattern = "config.%s.toml"

	// EnvServiceEnv specifies the environment name for configuration overlays.
	EnvServiceEnv = "SERVICE_ENV"

	// EnvServiceShutdownTimeout overrides the service shutdown timeout.
	EnvServiceShutdownTimeout = "SERVICE_SHUTDOWN_TIMEOUT"
)

// Config represents the root service configuration.
type Config struct {
	Server          ServerConfig      `toml:"server"`
	Database        DatabaseConfig    `toml:"database"`
	Logging         LoggingConfig     `toml:"logging"`
	CORS            CORSConfig        `toml:"cors"`
	Pagination      pagination.Config `toml:"pagination"`
	ShutdownTimeout string            `toml:"shutdown_timeout"`
}

// ShutdownTimeoutDuration parses and returns the shutdown timeout as a time.Duration.
func (c *Config) ShutdownTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.ShutdownTimeout)
	return d
}

// Load reads and parses the base configuration file and applies any environment-specific overlay.
func Load() (*Config, error) {
	cfg, err := load(BaseConfigFile)
	if err != nil {
		return nil, err
	}

	if path := overlayPath(); path != "" {
		overlay, err := load(path)
		if err != nil {
			return nil, fmt.Errorf("load overlay %s: %w", path, err)
		}
		cfg.Merge(overlay)
	}
	return cfg, nil
}

// Finalize applies defaults, loads environment overrides, and validates the configuration.
func (c *Config) Finalize() error {
	c.loadDefaults()
	c.loadEnv()

	if err := c.validate(); err != nil {
		return err
	}
	if err := c.Server.Finalize(); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	if err := c.Database.Finalize(); err != nil {
		return fmt.Errorf("database: %w", err)
	}
	if err := c.Logging.Finalize(); err != nil {
		return fmt.Errorf("logging: %w", err)
	}
	if err := c.CORS.Finalize(); err != nil {
		return fmt.Errorf("cors: %w", err)
	}
	if err := c.Pagination.Finalize(); err != nil {
		return fmt.Errorf("pagination: %w", err)
	}
	return nil
}

// Merge applies values from overlay configuration that differ from zero values.
func (c *Config) Merge(overlay *Config) {
	if overlay.ShutdownTimeout != "" {
		c.ShutdownTimeout = overlay.ShutdownTimeout
	}
	c.Server.Merge(&overlay.Server)
	c.Database.Merge(&overlay.Database)
	c.Logging.Merge(&overlay.Logging)
	c.CORS.Merge(&overlay.CORS)
	c.Pagination.Merge(&overlay.Pagination)
}

func (c *Config) loadDefaults() {
	if c.ShutdownTimeout == "" {
		c.ShutdownTimeout = "30s"
	}
}

func (c *Config) loadEnv() {
	if v := os.Getenv(EnvServiceShutdownTimeout); v != "" {
		c.ShutdownTimeout = v
	}
}

func (c *Config) validate() error {
	if _, err := time.ParseDuration(c.ShutdownTimeout); err != nil {
		return fmt.Errorf("invalid shutdown_timeout: %w", err)
	}
	return nil
}

func load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

func overlayPath() string {
	if env := os.Getenv(EnvServiceEnv); env != "" {
		overlayPath := fmt.Sprintf(OverlayConfigPattern, env)
		if _, err := os.Stat(overlayPath); err == nil {
			return overlayPath
		}
	}
	return ""
}
