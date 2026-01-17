# mt06 - Mountable Modules

## Overview

Refactor the server architecture into isolated, mountable "Modules" where each unit (API, App, Scalar) is self-contained with its own middleware pipeline, internal router, and base path.

**Key Changes:**
- Move `CORSConfig` to `pkg/middleware/config.go` (public packages own their config)
- Create `OpenAPIConfig` in `pkg/openapi/config.go` for OpenAPI specification settings
- Create `APIConfig` in `internal/config/api.go` consolidating API-specific settings (CORS, Pagination, OpenAPI, BasePath)
- Move `internal/middleware/` to `pkg/middleware/` for reusable middleware
- Introduce `pkg/module` for mountable sub-application infrastructure
- Add `Schemas` field to `routes.Group` to co-locate schemas with routes
- Enhance `pkg/openapi` with spec building and serving capabilities
- Web clients use AddSlash middleware + `<base>` tag for relative URLs
- API becomes a Module with isolated middleware (TrimSlash, CORS, Logger)
- Server only has native `/healthz` and `/readyz` endpoints
- Eliminate `cmd/server/openapi.go` - all OpenAPI logic moves to `pkg/openapi` and `internal/api`

**Architecture:**
```
Server (native: /healthz, /readyz)
├── /api     → API Module (TrimSlash, CORS, Logger)
├── /app/    → App Module (AddSlash, Logger)
└── /scalar/ → Scalar Module (AddSlash)
```

---

## Phase 1: Extract Reusable Packages

### 1.1 Create `pkg/middleware/config.go`

Move `CORSConfig` from `internal/config/cors.go` to `pkg/middleware/config.go`. Public packages define the config struct and an env keys struct. The application provides the env key mappings to `Finalize()`:

```go
package middleware

import (
	"os"
	"strconv"
	"strings"
)

type CORSConfig struct {
	Enabled          bool     `toml:"enabled"`
	Origins          []string `toml:"origins"`
	AllowedMethods   []string `toml:"allowed_methods"`
	AllowedHeaders   []string `toml:"allowed_headers"`
	AllowCredentials bool     `toml:"allow_credentials"`
	MaxAge           int      `toml:"max_age"`
}

type CORSEnv struct {
	Enabled          string
	Origins          string
	AllowedMethods   string
	AllowedHeaders   string
	AllowCredentials string
	MaxAge           string
}

func (c *CORSConfig) Finalize(env *CORSEnv) error {
	c.loadDefaults()
	if env != nil {
		c.loadEnv(env)
	}
	return nil
}

func (c *CORSConfig) Merge(overlay *CORSConfig) {
	c.Enabled = overlay.Enabled
	c.AllowCredentials = overlay.AllowCredentials

	if overlay.Origins != nil {
		c.Origins = overlay.Origins
	}
	if overlay.AllowedMethods != nil {
		c.AllowedMethods = overlay.AllowedMethods
	}
	if overlay.AllowedHeaders != nil {
		c.AllowedHeaders = overlay.AllowedHeaders
	}
	if overlay.MaxAge >= 0 {
		c.MaxAge = overlay.MaxAge
	}
}

func (c *CORSConfig) loadDefaults() {
	if len(c.AllowedMethods) == 0 {
		c.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(c.AllowedHeaders) == 0 {
		c.AllowedHeaders = []string{"Content-Type", "Authorization"}
	}
	if c.MaxAge <= 0 {
		c.MaxAge = 3600
	}
}

func (c *CORSConfig) loadEnv(env *CORSEnv) {
	if env.Enabled != "" {
		if v := os.Getenv(env.Enabled); v != "" {
			if enabled, err := strconv.ParseBool(v); err == nil {
				c.Enabled = enabled
			}
		}
	}

	if env.Origins != "" {
		if v := os.Getenv(env.Origins); v != "" {
			origins := strings.Split(v, ",")
			c.Origins = make([]string, 0, len(origins))
			for _, origin := range origins {
				if trimmed := strings.TrimSpace(origin); trimmed != "" {
					c.Origins = append(c.Origins, trimmed)
				}
			}
		}
	}

	if env.AllowedMethods != "" {
		if v := os.Getenv(env.AllowedMethods); v != "" {
			methods := strings.Split(v, ",")
			c.AllowedMethods = make([]string, 0, len(methods))
			for _, method := range methods {
				if trimmed := strings.TrimSpace(method); trimmed != "" {
					c.AllowedMethods = append(c.AllowedMethods, trimmed)
				}
			}
		}
	}

	if env.AllowedHeaders != "" {
		if v := os.Getenv(env.AllowedHeaders); v != "" {
			headers := strings.Split(v, ",")
			c.AllowedHeaders = make([]string, 0, len(headers))
			for _, header := range headers {
				if trimmed := strings.TrimSpace(header); trimmed != "" {
					c.AllowedHeaders = append(c.AllowedHeaders, trimmed)
				}
			}
		}
	}

	if env.AllowCredentials != "" {
		if v := os.Getenv(env.AllowCredentials); v != "" {
			if creds, err := strconv.ParseBool(v); err == nil {
				c.AllowCredentials = creds
			}
		}
	}

	if env.MaxAge != "" {
		if v := os.Getenv(env.MaxAge); v != "" {
			if maxAge, err := strconv.Atoi(v); err == nil {
				c.MaxAge = maxAge
			}
		}
	}
}
```

### 1.2 Create `pkg/openapi/config.go`

Same pattern - config struct plus env struct:

```go
package openapi

import "os"

type Config struct {
	Title       string `toml:"title"`
	Description string `toml:"description"`
	OutputPath  string `toml:"output_path"`
}

type Env struct {
	Title       string
	Description string
	OutputPath  string
}

func (c *Config) Finalize(env *Env) error {
	c.loadDefaults()
	if env != nil {
		c.loadEnv(env)
	}
	return nil
}

func (c *Config) Merge(overlay *Config) {
	if overlay.Title != "" {
		c.Title = overlay.Title
	}
	if overlay.Description != "" {
		c.Description = overlay.Description
	}
	if overlay.OutputPath != "" {
		c.OutputPath = overlay.OutputPath
	}
}

func (c *Config) loadDefaults() {
	if c.Title == "" {
		c.Title = "Agent Lab API"
	}
	if c.Description == "" {
		c.Description = "Containerized web service platform for building and orchestrating agentic workflows."
	}
	if c.OutputPath == "" {
		c.OutputPath = "web/scalar"
	}
}

func (c *Config) loadEnv(env *Env) {
	if env.Title != "" {
		if v := os.Getenv(env.Title); v != "" {
			c.Title = v
		}
	}
	if env.Description != "" {
		if v := os.Getenv(env.Description); v != "" {
			c.Description = v
		}
	}
	if env.OutputPath != "" {
		if v := os.Getenv(env.OutputPath); v != "" {
			c.OutputPath = v
		}
	}
}
```

### 1.3 Delete `internal/config/cors.go`

Remove the file after moving to `pkg/middleware/config.go`.

### 1.4 Move `internal/middleware/` to `pkg/middleware/`

Move all files from `internal/middleware/` to `pkg/middleware/`:
- `middleware.go` (System interface)
- `cors.go` (update to use local `CORSConfig`)
- `slash.go` (TrimSlash)
- `logger.go` (Logger)

### 1.5 Update `pkg/middleware/cors.go`

Update to use the local `CORSConfig` from the same package:

```go
package middleware

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
)

func CORS(cfg *CORSConfig) func(http.Handler) http.Handler {
	// ... implementation unchanged ...
}
```

### 1.6 Add `AddSlash` to `pkg/middleware/slash.go`

Add `AddSlash` and `hasFileExtension` to the existing `slash.go` file:

```go
func AddSlash() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/") && !hasFileExtension(r.URL.Path) {
				target := r.URL.Path + "/"
				if r.URL.RawQuery != "" {
					target += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, target, http.StatusMovedPermanently)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func hasFileExtension(path string) bool {
	lastSlash := strings.LastIndex(path, "/")
	lastDot := strings.LastIndex(path, ".")
	return lastDot > lastSlash
}
```

### 1.7 Delete `internal/middleware/`

Remove the directory after moving to `pkg/middleware/`.

### 1.8 Move `internal/lifecycle/` to `pkg/lifecycle/`

```bash
mkdir -p pkg/lifecycle
mv internal/lifecycle/lifecycle.go pkg/lifecycle/
rm -rf internal/lifecycle
```

### 1.9 Move `internal/storage/` to `pkg/storage/`

```bash
mv internal/storage pkg/storage
```

Move configuration from `internal/config/storage.go` to `pkg/storage/config.go`. Update to use `Env` pattern:

```go
package storage

import (
	"fmt"
	"os"

	"github.com/docker/go-units"
)

type Config struct {
	BasePath         string `toml:"base_path"`
	MaxUploadSize    string `toml:"max_upload_size"`
	maxUploadSizeVal int64
}

type Env struct {
	BasePath      string
	MaxUploadSize string
}

func (c *Config) Finalize(env *Env) error {
	c.loadDefaults()
	if env != nil {
		c.loadEnv(env)
	}
	return c.validate()
}

func (c *Config) Merge(overlay *Config) {
	if overlay.BasePath != "" {
		c.BasePath = overlay.BasePath
	}
	if overlay.MaxUploadSize != "" {
		c.MaxUploadSize = overlay.MaxUploadSize
		c.maxUploadSizeVal = overlay.maxUploadSizeVal
	}
}

func (c *Config) MaxUploadSizeBytes() int64 {
	return c.maxUploadSizeVal
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
		return fmt.Errorf("base_path is required")
	}
	size, err := units.FromHumanSize(c.MaxUploadSize)
	if err != nil {
		return fmt.Errorf("invalid max_upload_size: %w", err)
	}
	c.maxUploadSizeVal = size
	return nil
}
```

Delete `internal/config/storage.go` after moving.

### 1.10 Move `internal/database/` to `pkg/database/`

```bash
mv internal/database pkg/database
```

Move configuration from `internal/config/database.go` to `pkg/database/config.go`. Update to use `Env` pattern:

```go
package database

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
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

type Env struct {
	Host            string
	Port            string
	Name            string
	User            string
	Password        string
	MaxOpenConns    string
	MaxIdleConns    string
	ConnMaxLifetime string
	ConnTimeout     string
}

func (c *Config) Finalize(env *Env) error {
	c.loadDefaults()
	if env != nil {
		c.loadEnv(env)
	}
	return c.validate()
}

func (c *Config) Merge(overlay *Config) {
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

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable connect_timeout=%d",
		c.Host, c.Port, c.User, c.Password, c.Name, int(c.ConnTimeoutDuration().Seconds()),
	)
}

func (c *Config) ConnMaxLifetimeDuration() time.Duration {
	d, _ := time.ParseDuration(c.ConnMaxLifetime)
	return d
}

func (c *Config) ConnTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.ConnTimeout)
	return d
}

func (c *Config) loadDefaults() {
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

func (c *Config) loadEnv(env *Env) {
	if env.Host != "" {
		if v := os.Getenv(env.Host); v != "" {
			c.Host = v
		}
	}
	if env.Port != "" {
		if v := os.Getenv(env.Port); v != "" {
			if port, err := strconv.Atoi(v); err == nil {
				c.Port = port
			}
		}
	}
	if env.Name != "" {
		if v := os.Getenv(env.Name); v != "" {
			c.Name = v
		}
	}
	if env.User != "" {
		if v := os.Getenv(env.User); v != "" {
			c.User = v
		}
	}
	if env.Password != "" {
		if v := os.Getenv(env.Password); v != "" {
			c.Password = v
		}
	}
	if env.MaxOpenConns != "" {
		if v := os.Getenv(env.MaxOpenConns); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				c.MaxOpenConns = n
			}
		}
	}
	if env.MaxIdleConns != "" {
		if v := os.Getenv(env.MaxIdleConns); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				c.MaxIdleConns = n
			}
		}
	}
	if env.ConnMaxLifetime != "" {
		if v := os.Getenv(env.ConnMaxLifetime); v != "" {
			c.ConnMaxLifetime = v
		}
	}
	if env.ConnTimeout != "" {
		if v := os.Getenv(env.ConnTimeout); v != "" {
			c.ConnTimeout = v
		}
	}
}

func (c *Config) validate() error {
	if c.Name == "" {
		return fmt.Errorf("database name is required")
	}
	if c.User == "" {
		return fmt.Errorf("database user is required")
	}
	if _, err := time.ParseDuration(c.ConnMaxLifetime); err != nil {
		return fmt.Errorf("invalid conn_max_lifetime: %w", err)
	}
	if _, err := time.ParseDuration(c.ConnTimeout); err != nil {
		return fmt.Errorf("invalid conn_timeout: %w", err)
	}
	return nil
}
```

Delete `internal/config/database.go` after moving.

### 1.11 Update all imports

Update all files that import `internal/middleware` to use `pkg/middleware`:
- `cmd/server/middleware.go` (will be deleted later)
- `cmd/server/server.go`
- Any test files

Update all files that import `internal/lifecycle` to use `pkg/lifecycle`:
- `cmd/server/runtime.go`
- `cmd/server/http.go`
- Any test files

Update all files that import `internal/storage` to use `pkg/storage`:
- `cmd/server/runtime.go`
- Domain handlers that use storage
- Any test files

Update all files that import `internal/database` to use `pkg/database`:
- `cmd/server/runtime.go`
- Any test files

Update `internal/config/config.go` to use the new package locations:
- `pkg/database.Config` instead of local `DatabaseConfig`
- `pkg/storage.Config` instead of local `StorageConfig`

---

## Phase 2: Configuration Restructuring

### 2.1 Create `internal/config/api.go`

Consolidate API-specific configuration. This file defines the env key mappings and passes them to the public package `Finalize()` methods:

```go
package config

import (
	"fmt"
	"os"

	"github.com/JaimeStill/agent-lab/pkg/middleware"
	"github.com/JaimeStill/agent-lab/pkg/openapi"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
)

var corsEnv = &middleware.CORSEnv{
	Enabled:          "API_CORS_ENABLED",
	Origins:          "API_CORS_ORIGINS",
	AllowedMethods:   "API_CORS_ALLOWED_METHODS",
	AllowedHeaders:   "API_CORS_ALLOWED_HEADERS",
	AllowCredentials: "API_CORS_ALLOW_CREDENTIALS",
	MaxAge:           "API_CORS_MAX_AGE",
}

var openAPIEnv = &openapi.Env{
	Title:       "API_OPENAPI_TITLE",
	Description: "API_OPENAPI_DESCRIPTION",
	OutputPath:  "API_OPENAPI_OUTPUT_PATH",
}

var paginationEnv = &pagination.Env{
	DefaultPageSize: "API_PAGINATION_DEFAULT_PAGE_SIZE",
	MaxPageSize:     "API_PAGINATION_MAX_PAGE_SIZE",
}

type APIConfig struct {
	BasePath   string                `toml:"base_path"`
	CORS       middleware.CORSConfig `toml:"cors"`
	Pagination pagination.Config     `toml:"pagination"`
	OpenAPI    openapi.Config        `toml:"openapi"`
}

func (c *APIConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()

	if err := c.CORS.Finalize(corsEnv); err != nil {
		return fmt.Errorf("cors: %w", err)
	}
	if err := c.Pagination.Finalize(paginationEnv); err != nil {
		return fmt.Errorf("pagination: %w", err)
	}
	if err := c.OpenAPI.Finalize(openAPIEnv); err != nil {
		return fmt.Errorf("openapi: %w", err)
	}
	return nil
}

func (c *APIConfig) Merge(overlay *APIConfig) {
	if overlay.BasePath != "" {
		c.BasePath = overlay.BasePath
	}
	c.CORS.Merge(&overlay.CORS)
	c.Pagination.Merge(&overlay.Pagination)
	c.OpenAPI.Merge(&overlay.OpenAPI)
}

func (c *APIConfig) loadDefaults() {
	if c.BasePath == "" {
		c.BasePath = "/api"
	}
}

func (c *APIConfig) loadEnv() {
	if v := os.Getenv("API_BASE_PATH"); v != "" {
		c.BasePath = v
	}
}
```

### 2.2 Update `internal/config/config.go`

Update to use configs from `pkg/database` and `pkg/storage`, and add env key definitions:

```go
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/JaimeStill/agent-lab/pkg/database"
	"github.com/JaimeStill/agent-lab/pkg/storage"
	"github.com/pelletier/go-toml/v2"
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

type Config struct {
	Server          ServerConfig    `toml:"server"`
	Database        database.Config `toml:"database"`
	Logging         LoggingConfig   `toml:"logging"`
	Storage         storage.Config  `toml:"storage"`
	API             APIConfig       `toml:"api"`
	Domain          string          `toml:"domain"`
	ShutdownTimeout string          `toml:"shutdown_timeout"`
	Version         string          `toml:"version"`
}

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
```

Delete `internal/config/database.go` and `internal/config/storage.go` - these configs are now in their respective `pkg/` packages.

### 2.3 Update `config.toml`

Restructure to nest CORS, Pagination, and OpenAPI under `[api]`:

```toml
# Configuration is loaded on startup and validated once.
# Changes require service restart (Kubernetes rolling updates provide zero downtime).

# Service-level configuration
domain = "http://localhost:8080"
shutdown_timeout = "30s"
version = "0.1.0"

# HTTP server configuration
[server]
host = "0.0.0.0"
port = 8080
read_timeout = "1m"
write_timeout = "15m"
shutdown_timeout = "30s"

# Database configuration
[database]
host = "localhost"
port = 5432
name = "agent_lab"
user = "agent_lab"
password = "agent_lab"
max_open_conns = 25
max_idle_conns = 5
conn_max_lifetime = "15m"
conn_timeout = "5s"

# Logging configuration
[logging]
level = "info"
format = "text"

# Storage configuration
[storage]
base_path = ".data/blobs"
max_upload_size = "100MB"

# API module configuration
[api]
base_path = "/api"

[api.cors]
enabled = false
origins = []
allowed_methods = ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
allowed_headers = ["Content-Type", "Authorization"]
allow_credentials = false
max_age = 3600

[api.pagination]
default_page_size = 20
max_page_size = 100

[api.openapi]
title = "Agent Lab API"
description = "Containerized web service platform for building and orchestrating agentic workflows."
output_path = "web/scalar"
```

### 2.4 Update `.env`

Update environment variable names to reflect nested structure:

```bash
# ============================================================================
# Docker Compose - Container Configuration
# ============================================================================

# PostgreSQL Container
POSTGRES_DB=agent_lab
POSTGRES_USER=agent_lab
POSTGRES_PASSWORD=agent_lab
POSTGRES_PORT=5432

# Ollama Container (optional - for development with local LLM)
OLLAMA_PORT=11434
OLLAMA_MODELS_DIR=~/.ollama
OLLAMA_KEEP_ALIVE=5m

# ============================================================================
# Service Configuration
# ============================================================================

# Environment-specific config overlay (loads config.{SERVICE_ENV}.toml)

SERVICE_DOMAIN=http://localhost:8080

# Examples: local, dev, staging, prod
SERVICE_ENV=local

# Service-level configuration
SERVICE_SHUTDOWN_TIMEOUT=30s

SERVICE_VERSION=0.1.0

# ============================================================================
# Application Configuration - TOML Overrides
# ============================================================================
# Pattern: SECTION_FIELD (matches config.toml structure)
# Every TOML field has a corresponding environment variable override
# ============================================================================

# Database Connection (application config)
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=agent_lab
DATABASE_USER=agent_lab
DATABASE_PASSWORD=agent_lab
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=5
DATABASE_CONN_MAX_LIFETIME=15m
DATABASE_CONN_TIMEOUT=5s

# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_READ_TIMEOUT=1m
SERVER_WRITE_TIMEOUT=15m
SERVER_SHUTDOWN_TIMEOUT=30s

# Logging
LOGGING_LEVEL=info
LOGGING_FORMAT=json

# Storage
STORAGE_BASE_PATH=.data/blobs
STORAGE_MAX_UPLOAD_SIZE=100MB

# API Module
API_BASE_PATH=/api

# API CORS
API_CORS_ENABLED=false
API_CORS_ORIGINS=http://localhost:3000,http://localhost:8080
API_CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
API_CORS_ALLOWED_HEADERS=Content-Type,Authorization
API_CORS_ALLOW_CREDENTIALS=false
API_CORS_MAX_AGE=3600

# API Pagination
API_PAGINATION_DEFAULT_PAGE_SIZE=20
API_PAGINATION_MAX_PAGE_SIZE=100

# API OpenAPI
API_OPENAPI_TITLE=Agent Lab API
API_OPENAPI_DESCRIPTION=Containerized web service platform for building and orchestrating agentic workflows.
API_OPENAPI_OUTPUT_PATH=web/scalar

# ============================================================================
# CLI Tools
# ============================================================================

# Migration CLI (cmd/migrate)
# Full PostgreSQL connection string for golang-migrate
DATABASE_DSN=postgres://agent_lab:agent_lab@localhost:5432/agent_lab?sslmode=disable
```

### 2.5 Update `pkg/pagination/config.go`

Update to use the `Env` pattern (like CORS and OpenAPI configs):

```go
package pagination

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DefaultPageSize int `toml:"default_page_size"`
	MaxPageSize     int `toml:"max_page_size"`
}

type Env struct {
	DefaultPageSize string
	MaxPageSize     string
}

func (c *Config) Finalize(env *Env) error {
	c.loadDefaults()
	if env != nil {
		c.loadEnv(env)
	}
	return c.validate()
}

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

func (c *Config) loadEnv(env *Env) {
	if env.DefaultPageSize != "" {
		if v := os.Getenv(env.DefaultPageSize); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				c.DefaultPageSize = n
			}
		}
	}
	if env.MaxPageSize != "" {
		if v := os.Getenv(env.MaxPageSize); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				c.MaxPageSize = n
			}
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
```

**Key changes:**
- Remove const block with hardcoded env var names
- Add `Env` struct
- Update `Finalize(env *Env)` to accept env keys
- Update `loadEnv(env *Env)` to use dynamic keys

---

## Phase 3: Module Infrastructure

### 3.1 Create `pkg/module/module.go`

Module composes `middleware.System` instead of managing its own slice. Prefix is validated to be a single-level sub-path. Module knows how to serve itself via `Serve()`.

```go
package module

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/JaimeStill/agent-lab/pkg/middleware"
)

type Module struct {
	prefix     string
	router     http.Handler
	middleware middleware.System
}

func New(prefix string, router http.Handler) *Module {
	if err := validatePrefix(prefix); err != nil {
		panic(err)
	}
	return &Module{
		prefix:     prefix,
		router:     router,
		middleware: middleware.New(),
	}
}

func (m *Module) Use(mw func(http.Handler) http.Handler) {
	m.middleware.Use(mw)
}

func (m *Module) Prefix() string {
	return m.prefix
}

func (m *Module) Handler() http.Handler {
	return m.middleware.Apply(m.router)
}

func (m *Module) Serve(w http.ResponseWriter, req *http.Request) {
	path := extractPath(req.URL.Path, m.prefix)
	request := cloneRequest(req, path)
	m.Handler().ServeHTTP(w, request)
}

func validatePrefix(prefix string) error {
	if prefix == "" {
		return fmt.Errorf("module prefix cannot be empty")
	}
	if !strings.HasPrefix(prefix, "/") {
		return fmt.Errorf("module prefix must start with /: %s", prefix)
	}
	if strings.Count(prefix, "/") != 1 {
		return fmt.Errorf("module prefix must be single-level sub-path: %s", prefix)
	}
	return nil
}

func extractPath(fullPath, prefix string) string {
	path := fullPath[len(prefix):]
	if path == "" {
		return "/"
	}
	return path
}

func cloneRequest(req *http.Request, path string) *http.Request {
	r2 := new(http.Request)
	*r2 = *req
	r2.URL = new(url.URL)
	*r2.URL = *req.URL
	r2.URL.Path = path
	r2.URL.RawPath = ""
	return r2
}
```

### 3.2 Create `pkg/module/router.go`

Router is now simplified - it only handles routing, while Module handles serving.

```go
package module

import (
	"net/http"
	"strings"
)

type Router struct {
	modules map[string]*Module
	native  *http.ServeMux
}

func NewRouter() *Router {
	return &Router{
		modules: make(map[string]*Module),
		native:  http.NewServeMux(),
	}
}

func (r *Router) HandleNative(pattern string, handler http.HandlerFunc) {
	r.native.HandleFunc(pattern, handler)
}

func (r *Router) Mount(m *Module) {
	r.modules[m.prefix] = m
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	prefix := extractPrefix(req.URL.Path)

	if m, ok := r.modules[prefix]; ok {
		m.Serve(w, req)
		return
	}

	r.native.ServeHTTP(w, req)
}

func extractPrefix(path string) string {
	parts := strings.SplitN(path, "/", 3)
	if len(parts) >= 2 {
		return "/" + parts[1]
	}
	return path
}
```

---

## Phase 4: Routes and OpenAPI Updates

### 4.1 Add `Schemas` field and spec methods to `pkg/routes/group.go`

Update the `Group` struct and add methods to populate an OpenAPI spec. This keeps `pkg/openapi` as a low-level types package while `pkg/routes` handles the integration:

```go
type Group struct {
	Prefix      string
	Tags        []string
	Description string
	Routes      []Route
	Children    []Group
	Schemas     map[string]*openapi.Schema
}

func (g *Group) AddToSpec(spec *openapi.Spec) {
	g.addOperations("", spec)
}

func (g *Group) addOperations(parentPrefix string, spec *openapi.Spec) {
	fullPrefix := parentPrefix + g.Prefix

	for name, schema := range g.Schemas {
		spec.Components.Schemas[name] = schema
	}

	for _, route := range g.Routes {
		if route.OpenAPI == nil {
			continue
		}

		path := fullPrefix + route.Pattern
		op := route.OpenAPI

		if len(op.Tags) == 0 {
			op.Tags = g.Tags
		}

		if spec.Paths[path] == nil {
			spec.Paths[path] = &openapi.PathItem{}
		}

		switch route.Method {
		case "GET":
			spec.Paths[path].Get = op
		case "POST":
			spec.Paths[path].Post = op
		case "PUT":
			spec.Paths[path].Put = op
		case "DELETE":
			spec.Paths[path].Delete = op
		}
	}

	for _, child := range g.Children {
		child.addOperations(fullPrefix, spec)
	}
}
```

### 4.2 Add spec factory and helpers to `pkg/openapi`

Add to `pkg/openapi/spec.go` (new file):

```go
package openapi

import "net/http"

func NewSpec(title, version string) *Spec {
	return &Spec{
		OpenAPI: "3.1.0",
		Info: &Info{
			Title:   title,
			Version: version,
		},
		Components: NewComponents(),
		Paths:      make(map[string]*PathItem),
	}
}

func (s *Spec) SetDescription(desc string) {
	s.Info.Description = desc
}

func (s *Spec) AddServer(url string) {
	s.Servers = append(s.Servers, &Server{URL: url})
}

func ServeSpec(specBytes []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(specBytes)
	}
}
```

### 4.3 Add `Register` function to `pkg/routes`

Add route registration logic to `pkg/routes/group.go`:

```go
func Register(mux *http.ServeMux, spec *openapi.Spec, groups ...Group) {
	for _, group := range groups {
		group.AddToSpec(spec)
		registerGroup(mux, "", group)
	}
}

func registerGroup(mux *http.ServeMux, parentPrefix string, group Group) {
	fullPrefix := parentPrefix + group.Prefix
	for _, route := range group.Routes {
		pattern := route.Method + " " + fullPrefix + route.Pattern
		mux.HandleFunc(pattern, route.Handler)
	}
	for _, child := range group.Children {
		registerGroup(mux, fullPrefix, child)
	}
}
```

---

## Phase 5: API Module

The API module is a self-contained subsystem with its own runtime and domain layers. It receives shared infrastructure (Logger, DB, Storage) from the server and owns API-specific concerns (Pagination, Domain Handlers, CORS, OpenAPI).

**Ownership Principle**: Configuration structure reflects ownership boundaries. Everything under `[api]` in config belongs to the API module.

### 5.1 Create `internal/api/runtime.go`

API module's runtime holds API-specific infrastructure. It receives shared infrastructure from the server and owns API-specific config:

```go
package api

import (
	"log/slog"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/pkg/database"
	"github.com/JaimeStill/agent-lab/pkg/lifecycle"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/agent-lab/pkg/storage"
)

type Runtime struct {
	Logger     *slog.Logger
	DB         database.System
	Storage    storage.System
	Lifecycle  *lifecycle.Coordinator
	Pagination pagination.Config
}

func NewRuntime(
	cfg *config.APIConfig,
	logger *slog.Logger,
	db database.System,
	store storage.System,
	lc *lifecycle.Coordinator,
) *Runtime {
	return &Runtime{
		Logger:     logger.With("module", "api"),
		Database:   db,
		Storage:    store,
		Lifecycle:  lc,
		Pagination: cfg.Pagination,
	}
}
```

### 5.2 Create `internal/api/domain.go`

API module's domain holds domain systems (not handlers). Systems receive runtime infrastructure and may have dependencies on other systems:

```go
package api

import (
	"github.com/JaimeStill/agent-lab/internal/agents"
	"github.com/JaimeStill/agent-lab/internal/documents"
	"github.com/JaimeStill/agent-lab/internal/images"
	"github.com/JaimeStill/agent-lab/internal/profiles"
	"github.com/JaimeStill/agent-lab/internal/providers"
	"github.com/JaimeStill/agent-lab/internal/workflows"
)

type Domain struct {
	Providers providers.System
	Agents    agents.System
	Documents documents.System
	Images    images.System
	Profiles  profiles.System
	Workflows workflows.System
}

func NewDomain(runtime *Runtime) *Domain {
	providersSys := providers.New(
		runtime.Database.Connection(),
		runtime.Logger,
		runtime.Pagination,
	)

	agentsSys := agents.New(
		runtime.Database.Connection(),
		runtime.Logger,
		runtime.Pagination,
	)

	documentsSys := documents.New(
		runtime.Database.Connection(),
		runtime.Storage,
		runtime.Logger,
		runtime.Pagination,
	)

	imagesSys := images.New(
		documentsSys,
		runtime.Database.Connection(),
		runtime.Storage,
		runtime.Logger,
		runtime.Pagination,
	)

	profilesSys := profiles.New(
		runtime.Database.Connection(),
		runtime.Logger,
		runtime.Pagination,
	)

	workflowRuntime := workflows.NewRuntime(
		agentsSys,
		documentsSys,
		imagesSys,
		profilesSys,
		runtime.Lifecycle,
		runtime.Logger,
	)

	workflowsSys := workflows.NewSystem(
		workflowRuntime,
		runtime.Database.Connection(),
		runtime.Logger,
		runtime.Pagination,
	)

	return &Domain{
		Providers: providersSys,
		Agents:    agentsSys,
		Documents: documentsSys,
		Images:    imagesSys,
		Profiles:  profilesSys,
		Workflows: workflowsSys,
	}
}
```

### 5.3 Create `internal/api/routes.go`

Route registration creates handlers from domain systems and registers their routes using `routes.Register`:

```go
package api

import (
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/agents"
	"github.com/JaimeStill/agent-lab/internal/documents"
	"github.com/JaimeStill/agent-lab/internal/images"
	"github.com/JaimeStill/agent-lab/internal/profiles"
	"github.com/JaimeStill/agent-lab/internal/providers"
	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/agent-lab/pkg/openapi"
	"github.com/JaimeStill/agent-lab/pkg/routes"
)

func registerRoutes(mux *http.ServeMux, spec *openapi.Spec, runtime *Runtime, domain *Domain) {
	routes.Register(mux, spec,
		providers.NewHandler(domain.Providers, runtime.Pagination).Routes(),
		agents.NewHandler(domain.Agents, runtime.Pagination).Routes(),
		documents.NewHandler(domain.Documents, runtime.Pagination).Routes(),
		images.NewHandler(domain.Images, runtime.Pagination).Routes(),
		profiles.NewHandler(domain.Profiles, runtime.Pagination).Routes(),
		workflows.NewHandler(domain.Workflows, runtime.Pagination).Routes(),
	)
}
```

### 5.4 Create `internal/api/api.go`

The API module coordinates its runtime and domain, builds the OpenAPI spec, and configures middleware:

```go
package api

import (
	"log/slog"
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/pkg/database"
	"github.com/JaimeStill/agent-lab/pkg/lifecycle"
	"github.com/JaimeStill/agent-lab/pkg/middleware"
	"github.com/JaimeStill/agent-lab/pkg/module"
	"github.com/JaimeStill/agent-lab/pkg/openapi"
	"github.com/JaimeStill/agent-lab/pkg/storage"
)

func NewModule(
	cfg *config.APIConfig,
	version, serverURL string,
	logger *slog.Logger,
	db database.System,
	store storage.System,
	lc *lifecycle.Coordinator,
) (*module.Module, error) {
	runtime := NewRuntime(cfg, logger, db, store, lc)
	domain := NewDomain(runtime)

	spec := openapi.NewSpec(cfg.OpenAPI.Title, version)
	spec.SetDescription(cfg.OpenAPI.Description)
	spec.AddServer(serverURL)

	mux := http.NewServeMux()
	registerRoutes(mux, spec, runtime, domain)

	specBytes, err := openapi.MarshalJSON(spec)
	if err != nil {
		return nil, err
	}
	mux.HandleFunc("GET /openapi.json", openapi.ServeSpec(specBytes))

	m := module.New(cfg.BasePath, mux)
	m.Use(middleware.TrimSlash())
	m.Use(middleware.CORS(&cfg.CORS))
	m.Use(middleware.Logger(runtime.Logger))

	return m, nil
}
```

---

## Phase 6: Refactor Web App Module

### 6.1 Update `web/app/app.go`

Replace entire file with new Module-based implementation:

```go
package app

import (
	"embed"
	"net/http"

	"github.com/JaimeStill/agent-lab/pkg/module"
	"github.com/JaimeStill/agent-lab/pkg/web"
)

//go:embed dist/*
var distFS embed.FS

//go:embed public/*
var publicFS embed.FS

//go:embed server/layouts/*
var layoutFS embed.FS

//go:embed server/pages/*
var pageFS embed.FS

var publicFiles = []string{
	"favicon.ico",
	"favicon-16x16.png",
	"favicon-32x32.png",
	"apple-touch-icon.png",
	"site.webmanifest",
}

var pages = []web.PageDef{
	{Route: "/{$}", Template: "home.html", Title: "Home", Bundle: "app"},
	{Route: "/components/", Template: "components.html", Title: "Components", Bundle: "app"},
}

var errorPages = []web.PageDef{
	{Template: "404.html", Title: "Not Found"},
}

func NewModule(basePath string) (*module.Module, error) {
	allPages := append(pages, errorPages...)
	ts, err := web.NewTemplateSet(
		layoutFS,
		pageFS,
		"server/layouts/*.html",
		"server/pages",
		basePath,
		allPages,
	)
	if err != nil {
		return nil, err
	}

	router := buildRouter(ts)
	return module.New(basePath, router), nil
}

func buildRouter(ts *web.TemplateSet) http.Handler {
	r := web.NewRouter()
	r.SetFallback(ts.ErrorHandler(
		"app.html",
		"404.html",
		http.StatusNotFound,
		"Not Found",
	))

	for _, page := range pages {
		r.HandleFunc("GET "+page.Route, ts.PageHandler("app.html", page))
	}

	r.Handle("GET /dist/", http.FileServer(http.FS(distFS)))

	for _, route := range web.PublicFileRoutes(publicFS, "public", publicFiles...) {
		r.HandleFunc(route.Method+" "+route.Pattern, route.Handler)
	}

	return r
}
```

**Note:** Page routes now include trailing slash (`/components/`) to match AddSlash redirect behavior.

### 6.2 Update `web/app/server/layouts/app.html`

Replace entire file with `<base>` tag and relative URLs:

```html
<!DOCTYPE html>
<html lang="en">

<head>
  <base href="{{ .BasePath }}/">
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .Title }} - Agent Lab</title>
  <link rel="icon" type="image/x-icon" href="favicon.ico">
  <link rel="apple-touch-icon" sizes="180x180" href="apple-touch-icon.png">
  <link rel="icon" type="image/png" sizes="32x32" href="favicon-32x32.png">
  <link rel="icon" type="image/png" sizes="16x16" href="favicon-16x16.png">
  <link rel="stylesheet" href="dist/{{ .Bundle }}.css">
</head>

<body>
  <nav class="app-nav">
    <a href="./" class="app-nav-brand">Agent Lab</a>
    <div class="app-nav-links">
      <a href="workflows/">Workflows</a>
      <a href="documents/">Documents</a>
      <a href="profiles/">Profiles</a>
      <a href="agents/">Agents</a>
      <a href="providers/">Providers</a>
      <a href="components/">Components</a>
    </div>
  </nav>
  <main class="app-content">
    {{ block "content" . }}{{ end }}
  </main>

  <script type="module" src="dist/{{ .Bundle }}.js"></script>
</body>

</html>
```

---

## Phase 7: Refactor Scalar Module

### 7.1 Update `web/scalar/scalar.go`

Replace entire file with Module-based implementation:

```go
package scalar

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/JaimeStill/agent-lab/pkg/module"
)

//go:embed index.html scalar.css scalar.js
var staticFS embed.FS

func NewModule(basePath string) *module.Module {
	router := buildRouter(basePath)
	return module.New(basePath, router)
}

func buildRouter(basePath string) http.Handler {
	mux := http.NewServeMux()

	tmpl := template.Must(template.ParseFS(staticFS, "index.html"))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, map[string]string{"BasePath": basePath})
	})

	mux.Handle("GET /", http.FileServer(http.FS(staticFS)))

	return mux
}
```

### 7.2 Update `web/scalar/index.html`

Replace entire file with `<base>` tag template:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <base href="{{ .BasePath }}/">
  <meta charset="UTF-8">
  <title>Agent Lab - API Documentation</title>
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <link rel="stylesheet" href="scalar.css">
  <style>
    :root {
      --scalar-font: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
      --scalar-font-code: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, Monaco, 'Courier New', monospace;
    }
  </style>
</head>
<body>
  <div id="api-reference"></div>
  <script type="module" src="scalar.js"></script>
</body>
</html>
```

---

## Phase 8: Update Server Initialization

### 8.1 Update `cmd/server/runtime.go`

Remove Pagination from server runtime - it now belongs to the API module:

- Remove `Pagination pagination.Config` from the `Runtime` struct
- Remove `Pagination: cfg.Pagination,` from the struct initialization in `NewRuntime`
- Remove the `pkg/pagination` import

### 8.2 Delete `cmd/server/domain.go`

Domain handlers are now created by the API module internally.

### 8.3 Create `cmd/server/modules.go`

Encapsulate module initialization and mounting:

```go
package main

import (
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/api"
	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/pkg/middleware"
	"github.com/JaimeStill/agent-lab/pkg/module"
	"github.com/JaimeStill/agent-lab/web/app"
	"github.com/JaimeStill/agent-lab/web/scalar"
)

type Modules struct {
	API    *module.Module
	App    *module.Module
	Scalar *module.Module
}

func NewModules(runtime *Runtime, cfg *config.Config) (*Modules, error) {
	apiModule, err := api.NewModule(
		&cfg.API,
		cfg.Version,
		cfg.Domain,
		runtime.Logger,
		runtime.Database,
		runtime.Storage,
		runtime.Lifecycle,
	)
	if err != nil {
		return nil, err
	}

	appModule, err := app.NewModule("/app")
	if err != nil {
		return nil, err
	}
	appModule.Use(middleware.AddSlash())
	appModule.Use(middleware.Logger(runtime.Logger))

	scalarModule := scalar.NewModule("/scalar")
	scalarModule.Use(middleware.AddSlash())

	return &Modules{
		API:    apiModule,
		App:    appModule,
		Scalar: scalarModule,
	}, nil
}

func (m *Modules) Mount(router *module.Router) {
	router.Mount(m.API)
	router.Mount(m.App)
	router.Mount(m.Scalar)
}

func buildRouter(runtime *Runtime) *module.Router {
	router := module.NewRouter()

	router.HandleNative("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	router.HandleNative("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if !runtime.Lifecycle.Ready() {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("NOT READY"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("READY"))
	})

	return router
}
```

### 8.4 Replace `cmd/server/server.go`

Simplified server that coordinates modules:

```go
package main

import (
	"time"

	"github.com/JaimeStill/agent-lab/internal/config"
	_ "github.com/JaimeStill/agent-lab/workflows"
)

type Server struct {
	runtime *Runtime
	modules *Modules
	http    *httpServer
}

func NewServer(cfg *config.Config) (*Server, error) {
	runtime, err := NewRuntime(cfg)
	if err != nil {
		return nil, err
	}

	modules, err := NewModules(runtime, cfg)
	if err != nil {
		return nil, err
	}

	router := buildRouter(runtime)
	modules.Mount(router)

	runtime.Logger.Info(
		"server initialized",
		"addr", cfg.Server.Addr(),
		"version", cfg.Version,
	)

	return &Server{
		runtime: runtime,
		modules: modules,
		http:    newHTTPServer(&cfg.Server, router, runtime.Logger),
	}, nil
}

func (s *Server) Start() error {
	s.runtime.Logger.Info("starting service")

	if err := s.runtime.Start(); err != nil {
		return err
	}

	if err := s.http.Start(s.runtime.Lifecycle); err != nil {
		return err
	}

	go func() {
		s.runtime.Lifecycle.WaitForStartup()
		s.runtime.Logger.Info("all subsystems ready")
	}()

	return nil
}

func (s *Server) Shutdown(timeout time.Duration) error {
	s.runtime.Logger.Info("initiating shutdown")
	return s.runtime.Lifecycle.Shutdown(timeout)
}
```

### 8.5 Delete `cmd/server/routes.go`

This file is no longer needed - route registration is handled by modules.

### 8.6 Delete `cmd/server/middleware.go`

This file is no longer needed - each module configures its own middleware.

### 8.7 Update `cmd/server/http.go`

Update the lifecycle import from `internal/lifecycle` to `pkg/lifecycle`:

```go
import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/pkg/lifecycle"
)
```

The rest of the file remains unchanged - it already accepts `http.Handler` which works with the module router.

---

## Phase 9: Cleanup

### 9.1 Delete `cmd/server/openapi.go`

This file is no longer needed - OpenAPI spec generation is now handled by `internal/api` using `pkg/openapi`.

### 9.2 Delete `internal/routes/`

This directory is obsolete - route registration is now handled by `internal/api/routes.go`.

### 9.3 Evaluate `pkg/routes/system.go`

The `routes.System` implementation may no longer be needed. Evaluate:
- Domain handlers still return `routes.Group` (keep `pkg/routes/` types)
- The `routes.New()` factory and `System` interface may be removable
- Keep if any other code depends on it

### 9.4 Remove `api/` directory

1. Delete `api/README.md` - Content moved to `.claude/skills/openapi/SKILL.md`
2. Delete `api/openapi.local.json` - Will be regenerated in `web/scalar/`
3. Delete the `api/` directory

### 9.5 Update `.gitignore`

Add `web/scalar/openapi.*.json` to `.gitignore` (or update the existing `api/openapi.*.json` pattern).

---

## Verification

1. **Build check:**
   ```bash
   go vet ./...
   ```

2. **Web assets:**
   ```bash
   cd web && bun run build
   ```

3. **Run server:**
   ```bash
   go run ./cmd/server
   ```

4. **Test routes:**
   | Route | Expected Behavior |
   |-------|-------------------|
   | `GET /healthz` | 200 OK |
   | `GET /readyz` | 200 OK |
   | `GET /api/providers` | JSON response |
   | `GET /api/providers/` | Redirects to `/api/providers` |
   | `GET /api/openapi.json` | OpenAPI spec |
   | `GET /app` | Redirects to `/app/` |
   | `GET /app/` | Home page |
   | `GET /app/components` | Redirects to `/app/components/` |
   | `GET /app/components/` | Components page |
   | `GET /app/dist/app.js` | JS bundle (no redirect) |
   | `GET /app?debug=true` | Redirects to `/app/?debug=true` |
   | `GET /scalar` | Redirects to `/scalar/` |
   | `GET /scalar/` | Scalar UI |
   | `GET /scalar/scalar.js` | Scalar bundle (no redirect) |

5. **Verify relative URLs:**
   - Open browser dev tools on `/app/`
   - Check that all asset URLs resolve correctly
   - Navigate between pages, verify links work

6. **Run tests:**
   ```bash
   go test ./tests/...
   ```
   Update tests as needed for new module structure.

---

## Test Updates

Tests will need updates for the new Module-based architecture:

- `tests/web/app/app_test.go` - Update to test `NewModule()` instead of `NewHandler()`
- `tests/web_scalar/scalar_test.go` - Update to test `NewModule()`
- Add `tests/pkg_module/` for module infrastructure tests
- Add `tests/internal_api/` for API module tests
- Middleware tests may need adjustment for AddSlash

---

## Session Closeout Notes

During closeout, capture the following module design principles in the skills (likely LCA or a new module skill):

1. **Module Integration with Cold/Hot Start**
   - Modules are self-contained subsystems that can have their own runtime (infrastructure) and domain (business logic) layers
   - Server provides shared infrastructure during cold start; modules initialize their own internal layers
   - Module.Start() can be used for hot start if modules need async initialization

2. **Runtime + Domain Ownership**
   - Server Runtime: Truly shared infrastructure (Logger, Lifecycle, DB, Storage)
   - Module Runtime: Module-specific infrastructure (e.g., API owns Pagination)
   - Module Domain: Module-specific business logic (e.g., API owns all handlers)
   - Shared infrastructure flows down from server to modules

3. **Ownership Hierarchy Principle**
   - **Configuration structure reflects ownership boundaries**
   - Root-level config sections (`[database]`, `[storage]`) → Server owns
   - Nested config sections (`[api.cors]`, `[api.pagination]`) → Module owns
   - If only one module uses a resource AND it's configured under that module → module owns it
   - General-purpose infrastructure stays at server level even if currently used by one module
