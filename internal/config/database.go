package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	// EnvDatabaseHost overrides the database host address.
	EnvDatabaseHost = "DATABASE_HOST"

	// EnvDatabasePort overrides the database port.
	EnvDatabasePort = "DATABASE_PORT"

	// EnvDatabaseName overrides the database name.
	EnvDatabaseName = "DATABASE_NAME"

	// EnvDatabaseUser overrides the database user.
	EnvDatabaseUser = "DATABASE_USER"

	// EnvDatabasePassword overrides the database password.
	EnvDatabasePassword = "DATABASE_PASSWORD"

	// EnvDatabaseMaxOpenConns overrides the maximum number of open connections.
	EnvDatabaseMaxOpenConns = "DATABASE_MAX_OPEN_CONNS"

	// EnvDatabaseMaxIdleConns overrides the maximum number of idle connections.
	EnvDatabaseMaxIdleConns = "DATABASE_MAX_IDLE_CONNS"

	// EnvDatabaseConnMaxLifetime overrides the connection maximum lifetime.
	EnvDatabaseConnMaxLifetime = "DATABASE_CONN_MAX_LIFETIME"

	// EnvDatabaseConnTimeout overrides the connection timeout.
	EnvDatabaseConnTimeout = "DATABASE_CONN_TIMEOUT"
)

// DatabaseConfig contains database connection configuration.
type DatabaseConfig struct {
	Host            string `toml:"host"`
	Port            int    `toml:"port"`
	Name            string `toml:"name"`
	User            string `toml:"user"`
	Password        string `toml:"password"`
	MaxOpenConns    int    `toml:"max_open_conns"`
	MaxIdleConns    int    `toml:"max_idle_conns"`
	ConnMaxLifetime string `toml:"conn_max_lifetime"`
	ConnTimeout     string `toml:"conn_timeout"`
}

// ConnMaxLifetimeDuration parses and returns the connection max lifetime as a time.Duration.
func (c *DatabaseConfig) ConnMaxLifetimeDuration() time.Duration {
	d, _ := time.ParseDuration(c.ConnMaxLifetime)
	return d
}

// ConnTimeoutDuration parses and returns the connection timeout as a time.Duration.
func (c *DatabaseConfig) ConnTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.ConnTimeout)
	return d
}

func (c *DatabaseConfig) Dsn() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		c.Host, c.Port, c.Name, c.User, c.Password,
	)
}

// Finalize applies defaults, loads environment overrides, and validates the database configuration.
func (c *DatabaseConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()
	return c.validate()
}

// Merge applies values from overlay configuration that differ from zero values.
func (c *DatabaseConfig) Merge(overlay *DatabaseConfig) {
	if overlay.Host != "" {
		c.Host = overlay.Host
	}
	if overlay.Port != 0 {
		c.Port = overlay.Port
	}
	if overlay.Name != "" {
		c.Name = overlay.Name
	}
	if overlay.User != "" {
		c.User = overlay.User
	}
	if overlay.Password != "" {
		c.Password = overlay.Password
	}
	if overlay.MaxOpenConns != 0 {
		c.MaxOpenConns = overlay.MaxOpenConns
	}
	if overlay.MaxIdleConns != 0 {
		c.MaxIdleConns = overlay.MaxIdleConns
	}
	if overlay.ConnMaxLifetime != "" {
		c.ConnMaxLifetime = overlay.ConnMaxLifetime
	}
	if overlay.ConnTimeout != "" {
		c.ConnTimeout = overlay.ConnTimeout
	}
}

func (c *DatabaseConfig) loadEnv() {
	if v := os.Getenv(EnvDatabaseHost); v != "" {
		c.Host = v
	}
	if v := os.Getenv(EnvDatabasePort); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Port = port
		}
	}
	if v := os.Getenv(EnvDatabaseName); v != "" {
		c.Name = v
	}
	if v := os.Getenv(EnvDatabaseUser); v != "" {
		c.User = v
	}
	if v := os.Getenv(EnvDatabasePassword); v != "" {
		c.Password = v
	}
	if v := os.Getenv(EnvDatabaseMaxOpenConns); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxOpenConns = n
		}
	}
	if v := os.Getenv(EnvDatabaseMaxIdleConns); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxIdleConns = n
		}
	}
	if v := os.Getenv(EnvDatabaseConnMaxLifetime); v != "" {
		c.ConnMaxLifetime = v
	}
	if v := os.Getenv(EnvDatabaseConnTimeout); v != "" {
		c.ConnTimeout = v
	}
}

func (c *DatabaseConfig) loadDefaults() {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		c.Port = 5432
	}
	if c.MaxOpenConns == 0 {
		c.MaxOpenConns = 25
	}
	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = 5
	}
	if c.ConnMaxLifetime == "" {
		c.ConnMaxLifetime = "15m"
	}
	if c.ConnTimeout == "" {
		c.ConnTimeout = "5s"
	}
}

func (c *DatabaseConfig) validate() error {
	if c.Name == "" {
		return fmt.Errorf("name required")
	}
	if c.User == "" {
		return fmt.Errorf("user required")
	}
	if _, err := time.ParseDuration(c.ConnMaxLifetime); err != nil {
		return fmt.Errorf("invalid conn_max_lifetime: %w", err)
	}
	if _, err := time.ParseDuration(c.ConnTimeout); err != nil {
		return fmt.Errorf("invalid conn_timeout: %w", err)
	}
	return nil
}
