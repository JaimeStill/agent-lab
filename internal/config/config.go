package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Server     ServerConfig     `toml:"server"`
	Database   DatabaseConfig   `toml:"database"`
	Pagination PaginationConfig `toml:"pagination"`
	CORS       CORSConfig       `toml:"cors"`
	Logging    LoggingConfig    `toml:"logging"`
}

type ServerConfig struct {
	Host            string        `toml:"host"`
	Port            int           `toml:"port"`
	ReadTimeout     time.Duration `toml:"read_timeout"`
	WriteTimeout    time.Duration `toml:"write_timeout"`
	IdleTimeout     time.Duration `toml:"idle_timeout"`
	ShutdownTimeout time.Duration `toml:"shutdown_timeout"`
}

type DatabaseConfig struct {
	Host            string        `toml:"host"`
	Port            int           `toml:"port"`
	Name            string        `toml:"name"`
	User            string        `toml:"user"`
	Password        string        `toml:"password"`
	MaxOpenConns    int           `toml:"max_open_conns"`
	MaxIdleConns    int           `toml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `toml:"conn_max_lifetime"`
	ConnTimeout     time.Duration `toml:"conn_timeout"`
}

type PaginationConfig struct {
	DefaultPageSize int `toml:"default_page_size"`
	MaxPageSize     int `toml:"max_page_size"`
}

type CORSConfig struct {
	Origins     []string `toml:"origins"`
	Methods     []string `toml:"methods"`
	Headers     []string `toml:"headers"`
	Credentials bool     `toml:"credentials"`
}

type LoggingConfig struct {
	Level  string `toml:"level"`
	Format string `toml:"format"`
}

func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	applyEnvironmentOverrides(&config)

	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func applyEnvironmentOverrides(config *Config) {
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}

	if host := os.Getenv("DATABASE_HOST"); host != "" {
		config.Database.Host = host
	}
	if port := os.Getenv("DATABASE_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Database.Port = p
		}
	}
	if db := os.Getenv("DATABASE_NAME"); db != "" {
		config.Database.Name = db
	}
	if user := os.Getenv("DATABASE_USER"); user != "" {
		config.Database.User = user
	}
	if password := os.Getenv("DATABASE_PASSWORD"); password != "" {
		config.Database.Password = password
	}

	if defaultPageSize := os.Getenv("PAGINATION_DEFAULT_PAGE_SIZE"); defaultPageSize != "" {
		if ps, err := strconv.Atoi(defaultPageSize); err == nil {
			config.Pagination.DefaultPageSize = ps
		}
	}
	if maxPageSize := os.Getenv("PAGINATION_MAX_PAGE_SIZE"); maxPageSize != "" {
		if ps, err := strconv.Atoi(maxPageSize); err == nil {
			config.Pagination.MaxPageSize = ps
		}
	}

	if origins := os.Getenv("CORS_ORIGINS"); origins != "" {
		config.CORS.Origins = strings.Split(origins, ",")
	}

	if level := os.Getenv("LOGGING_LEVEL"); level != "" {
		config.Logging.Level = level
	}
	if format := os.Getenv("LOGGING_FORMAT"); format != "" {
		config.Logging.Format = format
	}
}

func validate(config *Config) error {
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	if config.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if config.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}

	if config.Pagination.DefaultPageSize <= 0 {
		return fmt.Errorf("default page size must be positive")
	}
	if config.Pagination.MaxPageSize <= 0 {
		return fmt.Errorf("max page size must be positive")
	}
	if config.Pagination.DefaultPageSize > config.Pagination.MaxPageSize {
		return fmt.Errorf("default page size cannot exceed max page size")
	}

	return nil
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		c.Host, c.Port, c.Name, c.User, c.Password,
	)
}
