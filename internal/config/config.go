// Package config provides application configuration management with support for
// TOML files, environment variable overrides, and configuration overlays.
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/JaimeStill/agent-lab/pkg/database"
	"github.com/JaimeStill/agent-lab/pkg/storage"
	"github.com/pelletier/go-toml/v2"
)

const (
	// BaseConfigFile is the primary configuration file name.
	BaseConfigFile = "config.toml"

	// OverlayConfigPattern is the file name pattern for environment-specific overlays.
	OverlayConfigPattern = "config.%s.toml"

	EnvServiceDomain = "SERVICE_DOMAIN"

	// EnvServiceEnv specifies the environment name for configuration overlays.
	EnvServiceEnv = "SERVICE_ENV"

	// EnvServiceShutdownTimeout overrides the service shutdown timeout.
	EnvServiceShutdownTimeout = "SERVICE_SHUTDOWN_TIMEOUT"

	EnvServiceVersion = "SERVICE_VERSION"
)

var databaseEnv = &database.Env{
	Host:            "DATABASE_HOST",
	Port:            "DATABASE_PORT",
	Name:            "DATABASE_NAME",
	User:            "DATABASE_USER",
	Password:        "DATABASE_PASSWORD",
	MaxOpenConns:    "DATABASE_MAX_OPEN_CONNS",
	MaxIdleConns:    "DATABASE_MAX_IDLE_CONNS",
	ConnMaxLifetime: "DATABASE_CONN_MAX_LIFETIME",
	ConnTimeout:     "DATABASE_CONN_TIMEOUT",
}

var storageEnv = &storage.Env{
	BasePath:      "STORAGE_BASE_PATH",
	MaxUploadSize: "STORAGE_MAX_UPLOAD_SIZE",
}

// Config represents the root service configuration.
type Config struct {
	Server          ServerConfig    `toml:"server"`
	Database        database.Config `toml:"database"`
	Logging         LoggingConfig   `toml:"logging"`
	Storage         storage.Config  `toml:"storage"`
	API             APIConfig       `toml:"api"`
	Domain          string          `toml:"version"`
	ShutdownTimeout string          `toml:"shutdown_timeout"`
	Version         string          `toml:"version"`
}

// Env returns the current environment name from the SERVICE_ENV variable or "local".
func (c *Config) Env() string {
	if env := os.Getenv(EnvServiceEnv); env != "" {
		return env
	}
	return "local"
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

	if err := cfg.finalize(); err != nil {
		return nil, fmt.Errorf("finalize config: %w", err)
	}

	return cfg, nil
}

// Finalize applies defaults, loads environment overrides, and validates the configuration.
func (c *Config) finalize() error {
	c.loadDefaults()
	c.loadEnv()

	if err := c.validate(); err != nil {
		return err
	}
	if err := c.Server.Finalize(); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	if err := c.Database.Finalize(databaseEnv); err != nil {
		return fmt.Errorf("database: %w", err)
	}
	if err := c.Logging.Finalize(); err != nil {
		return fmt.Errorf("logging: %w", err)
	}
	if err := c.Storage.Finalize(storageEnv); err != nil {
		return fmt.Errorf("storage: %w", err)
	}
	if err := c.API.Finalize(); err != nil {
		return fmt.Errorf("api: %w", err)
	}
	return nil
}

// Merge applies values from overlay configuration that differ from zero values.
func (c *Config) Merge(overlay *Config) {
	if overlay.Domain != "" {
		c.Domain = overlay.Domain
	}
	if overlay.ShutdownTimeout != "" {
		c.ShutdownTimeout = overlay.ShutdownTimeout
	}
	if overlay.Version != "" {
		c.Version = overlay.Version
	}
	c.Server.Merge(&overlay.Server)
	c.Database.Merge(&overlay.Database)
	c.Logging.Merge(&overlay.Logging)
	c.Storage.Merge(&overlay.Storage)
	c.API.Merge(&overlay.API)
}

func (c *Config) loadDefaults() {
	if c.Domain == "" {
		c.Domain = "http://localhost:8080"
	}
	if c.ShutdownTimeout == "" {
		c.ShutdownTimeout = "30s"
	}
	if c.Version == "" {
		c.Version = "0.1.0"
	}

}

func (c *Config) loadEnv() {
	if v := os.Getenv(EnvServiceDomain); v != "" {
		c.Domain = v
	}
	if v := os.Getenv(EnvServiceShutdownTimeout); v != "" {
		c.ShutdownTimeout = v
	}
	if v := os.Getenv(EnvServiceVersion); v != "" {
		c.Version = v
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
