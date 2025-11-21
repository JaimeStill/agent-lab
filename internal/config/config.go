package config

import (
	"fmt"
	"os"
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
	Database        string        `toml:"database"`
	User            string        `toml:"user"`
	Password        string        `toml:"password"`
	MaxOpenConns    int           `toml:"max_open_conns"`
	MaxIdleConns    int           `toml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `toml:"conn_max_lifetime"`
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
}
