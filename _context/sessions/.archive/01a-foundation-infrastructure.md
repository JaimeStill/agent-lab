# Session 1a: Foundation Infrastructure

**Milestone**: 1 - Foundation
**Session**: 1a
**Objective**: Get the server running with basic configuration and health check

## Context

This is the first implementation session for agent-lab. We're establishing the foundation infrastructure that all subsequent sessions will build upon:

- PostgreSQL 17 database container
- TOML-based configuration management with environment variable overrides
- Server system implementing Cold Start/Hot Start lifecycle pattern
- Basic HTTP server with health check endpoint
- Graceful shutdown on SIGTERM/SIGINT

**Architecture Principles Applied**:
- State flows down through method parameters
- Concrete config structs with simplified finalize pattern
- Cold Start (state initialization) vs Hot Start (process activation)
- Composition root pattern (cmd/service owns all subsystems)

## Success Criteria

- `docker compose up -d` starts PostgreSQL 17 container
- Service loads configuration from TOML + environment variables
- `go run cmd/service/main.go` starts service successfully
- `curl http://localhost:8080/healthz` returns `200 OK`
- `Ctrl+C` triggers graceful shutdown

---

## Phase 1: Docker & Database Setup

### Step 1.1: Docker Compose Configuration

**Modular Pattern**: Individual service definitions in `compose/` directory, composed together in root-level files.

**File**: `compose/postgres.yml`

```yaml
services:
  postgres:
    image: postgres:17
    container_name: agent-lab-postgres
    environment:
      POSTGRES_DB: ${POSTGRES_DB:-agent_lab}
      POSTGRES_USER: ${POSTGRES_USER:-agent_lab}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-agent_lab}
    volumes:
      - postgres-data:/var/lib/postgresql/data
    ports:
      - "${POSTGRES_PORT:-5432}:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-agent_lab}"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - agent-lab

volumes:
  postgres-data:

networks:
  agent-lab:
    name: agent-lab
    driver: bridge
```

**File**: `compose/ollama.yml`

```yaml
services:
  ollama:
    image: ollama/ollama:latest
    container_name: agent-lab-ollama
    ports:
      - "${OLLAMA_PORT:-11434}:11434"
    volumes:
      - ${OLLAMA_MODELS_DIR:-~/.ollama}:/root/.ollama
    environment:
      - OLLAMA_KEEP_ALIVE=${OLLAMA_KEEP_ALIVE:-5m}
      - OLLAMA_HOST=0.0.0.0
    devices:
      - nvidia.com/gpu=all
    restart: unless-stopped
    networks:
      - agent-lab

networks:
  agent-lab:
    name: agent-lab
    external: true
```

**File**: `docker-compose.yml`

```yaml
include:
  - compose/postgres.yml
```

**File**: `docker-compose.dev.yml`

```yaml
include:
  - compose/postgres.yml
  - compose/ollama.yml
```

### Step 1.2: Environment Template

**File**: `.env.example`

```env
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
# Application Configuration
# ============================================================================

# Environment-specific config overlay (loads config.{SERVICE_ENV}.toml)
# Examples: local, dev, staging, prod
SERVICE_ENV=local

# Service-level configuration
SERVICE_SHUTDOWN_TIMEOUT=30s

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
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s
SERVER_SHUTDOWN_TIMEOUT=30s

# Logging
LOGGING_LEVEL=info
LOGGING_FORMAT=json

# CORS
CORS_ENABLED=false
CORS_ORIGINS=http://localhost:3000,http://localhost:8080
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization
CORS_ALLOW_CREDENTIALS=false
CORS_MAX_AGE=3600
```

### Step 1.3: Git Ignore

**File**: `.gitignore`

```
.env
config.local.toml
```

---

## Phase 2: Configuration Management

### Configuration Precedence Principle

**All configuration values (scalar or array) are atomic units that replace at each precedence level:**

```
Environment Variables (highest precedence)
    ↓ replaces (not merges)
config.local.toml / config.*.toml
    ↓ replaces (not merges)
config.toml (base configuration)
```

**Key Principles:**
1. **Atomic Replacement**: Configuration values are never merged - presence indicates complete replacement
2. **Array Format**: Array values use comma-separated strings in environment variables
3. **Consistent Behavior**: Scalar and array configs follow the same precedence rules
4. **Predictable**: What you see at each level is exactly what you get

**Examples:**

Scalar value:
```toml
# config.toml
[server]
port = 8080
```
```bash
# Environment override (replaces)
SERVER_PORT=9090
# Result: port = 9090
```

Array value:
```toml
# config.toml
[cors]
origins = ["http://localhost:3000", "http://localhost:8080"]
```
```bash
# Environment override (replaces, not merges)
CORS_ORIGINS="http://example.com,http://other.com"
# Result: origins = ["http://example.com", "http://other.com"]
```

---

### Step 2.1: Base Configuration

**File**: `config.toml`

```toml
# Configuration is loaded on startup and validated once.
# Changes require service restart (Kubernetes rolling updates provide zero downtime).

# Service-level configuration
shutdown_timeout = "30s"

# HTTP server configuration
[server]
host = "0.0.0.0"
port = 8080
read_timeout = "30s"
write_timeout = "30s"
shutdown_timeout = "30s"

# Database configuration
[database]
host = "localhost"
port = 5432
name = "agent_lab"
user = "agent_lab"
password = ""
max_open_conns = 25
max_idle_conns = 5
conn_max_lifetime = "15m"
conn_timeout = "5s"

# Logging configuration
[logging]
level = "info"
format = "json"

# CORS configuration
[cors]
enabled = false
origins = []
allowed_methods = ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
allowed_headers = ["Content-Type", "Authorization"]
allow_credentials = false
max_age = 3600
```

### Step 2.2: Configuration Structures

**Pattern**: Configuration is service-specific infrastructure in `internal/config/`. Config is ephemeral: loaded on startup, validated once, passed to systems. Changes require service restart.

**Package Structure**:
```
internal/config/
├── config.go    # Root Config + LoadConfig
├── types.go     # Typed enums (LogLevel, LogFormat)
├── server.go    # ServerConfig
├── database.go  # DatabaseConfig
├── logging.go   # LoggingConfig
└── cors.go      # CORSConfig
```

**Configuration Principle**:
- Configuration is ephemeral (loaded on startup, not modified at runtime)
- Finalize → Validate → Transform pattern
- Config changes require restart (K8s rolling updates provide zero downtime)

### Step 2.3: Root Configuration

**File**: `internal/config/config.go`

```go
package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

const (
	BaseConfigFile       = "config.toml"
	OverlayConfigPattern = "config.%s.toml"
)

type Config struct {
	ShutdownTimeout string         `toml:"shutdown_timeout"`
	Server          ServerConfig   `toml:"server"`
	Database        DatabaseConfig `toml:"database"`
	Logging         LoggingConfig  `toml:"logging"`
	CORS            CORSConfig     `toml:"cors"`
}

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

	return nil
}

func (c *Config) loadDefaults() {
	if c.ShutdownTimeout == "" {
		c.ShutdownTimeout = "30s"
	}
}

func (c *Config) loadEnv() {
	if v := os.Getenv("SERVICE_SHUTDOWN_TIMEOUT"); v != "" {
		c.ShutdownTimeout = v
	}
}

func (c *Config) validate() error {
	if _, err := time.ParseDuration(c.ShutdownTimeout); err != nil {
		return fmt.Errorf("invalid shutdown_timeout: %w", err)
	}
	return nil
}

func (c *Config) ShutdownTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.ShutdownTimeout)
	return d
}

func (c *Config) Merge(overlay *Config) {
	if overlay.ShutdownTimeout != "" {
		c.ShutdownTimeout = overlay.ShutdownTimeout
	}
	c.Server.Merge(&overlay.Server)
	c.Database.Merge(&overlay.Database)
	c.Logging.Merge(&overlay.Logging)
	c.CORS.Merge(&overlay.CORS)
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
	if env := os.Getenv("SERVICE_ENV"); env != "" {
		overlayPath := fmt.Sprintf(OverlayConfigPattern, env)
		if _, err := os.Stat(overlayPath); err == nil {
			return overlayPath
		}
	}
	return ""
}
```

### Step 2.4: Typed Enums

**File**: `internal/config/types.go`

```go
package config

import (
	"fmt"
	"log/slog"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

func (l LogLevel) Validate() error {
	switch l {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
		return nil
	default:
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", l)
	}
}

func (l LogLevel) ToSlogLevel() slog.Level {
	switch l {
	case LogLevelDebug:
		return slog.LevelDebug
	case LogLevelInfo:
		return slog.LevelInfo
	case LogLevelWarn:
		return slog.LevelWarn
	case LogLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type LogFormat string

const (
	LogFormatText LogFormat = "text"
	LogFormatJSON LogFormat = "json"
)

func (f LogFormat) Validate() error {
	switch f {
	case LogFormatText, LogFormatJSON:
		return nil
	default:
		return fmt.Errorf("invalid log format: %s (must be text or json)", f)
	}
}
```

### Step 2.5: Server Configuration

**File**: `internal/config/server.go`

```go
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type ServerConfig struct {
	Host            string `toml:"host"`
	Port            int    `toml:"port"`
	ReadTimeout     string `toml:"read_timeout"`
	WriteTimeout    string `toml:"write_timeout"`
	ShutdownTimeout string `toml:"shutdown_timeout"`
}

func (c *ServerConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()
	return c.validate()
}

func (c *ServerConfig) loadDefaults() {
	if c.Host == "" {
		c.Host = "0.0.0.0"
	}
	if c.Port == 0 {
		c.Port = 8080
	}
	if c.ReadTimeout == "" {
		c.ReadTimeout = "30s"
	}
	if c.WriteTimeout == "" {
		c.WriteTimeout = "30s"
	}
	if c.ShutdownTimeout == "" {
		c.ShutdownTimeout = "30s"
	}
}

func (c *ServerConfig) loadEnv() {
	if v := os.Getenv("SERVER_HOST"); v != "" {
		c.Host = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Port = port
		}
	}
	if v := os.Getenv("SERVER_READ_TIMEOUT"); v != "" {
		c.ReadTimeout = v
	}
	if v := os.Getenv("SERVER_WRITE_TIMEOUT"); v != "" {
		c.WriteTimeout = v
	}
	if v := os.Getenv("SERVER_SHUTDOWN_TIMEOUT"); v != "" {
		c.ShutdownTimeout = v
	}
}

func (c *ServerConfig) validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}
	if _, err := time.ParseDuration(c.ReadTimeout); err != nil {
		return fmt.Errorf("invalid read_timeout: %w", err)
	}
	if _, err := time.ParseDuration(c.WriteTimeout); err != nil {
		return fmt.Errorf("invalid write_timeout: %w", err)
	}
	if _, err := time.ParseDuration(c.ShutdownTimeout); err != nil {
		return fmt.Errorf("invalid shutdown_timeout: %w", err)
	}
	return nil
}

func (c *ServerConfig) ReadTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.ReadTimeout)
	return d
}

func (c *ServerConfig) WriteTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.WriteTimeout)
	return d
}

func (c *ServerConfig) ShutdownTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.ShutdownTimeout)
	return d
}

func (c *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c *ServerConfig) Merge(overlay *ServerConfig) {
	if overlay.Host != "" {
		c.Host = overlay.Host
	}
	if overlay.Port != 0 {
		c.Port = overlay.Port
	}
	if overlay.ReadTimeout != "" {
		c.ReadTimeout = overlay.ReadTimeout
	}
	if overlay.WriteTimeout != "" {
		c.WriteTimeout = overlay.WriteTimeout
	}
	if overlay.ShutdownTimeout != "" {
		c.ShutdownTimeout = overlay.ShutdownTimeout
	}
}
```

### Step 2.6: Database Configuration

**File**: `internal/config/database.go`

```go
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

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

func (c *DatabaseConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()
	return c.validate()
}

func (c *DatabaseConfig) loadEnv() {
	if v := os.Getenv("DATABASE_HOST"); v != "" {
		c.Host = v
	}
	if v := os.Getenv("DATABASE_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Port = port
		}
	}
	if v := os.Getenv("DATABASE_NAME"); v != "" {
		c.Name = v
	}
	if v := os.Getenv("DATABASE_USER"); v != "" {
		c.User = v
	}
	if v := os.Getenv("DATABASE_PASSWORD"); v != "" {
		c.Password = v
	}
	if v := os.Getenv("DATABASE_MAX_OPEN_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxOpenConns = n
		}
	}
	if v := os.Getenv("DATABASE_MAX_IDLE_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxIdleConns = n
		}
	}
	if v := os.Getenv("DATABASE_CONN_MAX_LIFETIME"); v != "" {
		c.ConnMaxLifetime = v
	}
	if v := os.Getenv("DATABASE_CONN_TIMEOUT"); v != "" {
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

func (c *DatabaseConfig) ConnMaxLifetimeDuration() time.Duration {
	d, _ := time.ParseDuration(c.ConnMaxLifetime)
	return d
}

func (c *DatabaseConfig) ConnTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.ConnTimeout)
	return d
}

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
```

### Step 2.7: Logging Configuration

**File**: `internal/config/logging.go`

```go
package config

import (
	"fmt"
	"os"
)

type LoggingConfig struct {
	Level  LogLevel  `toml:"level"`
	Format LogFormat `toml:"format"`
}

func (c *LoggingConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()
	return c.validate()
}

func (c *LoggingConfig) loadEnv() {
	if v := os.Getenv("LOGGING_LEVEL"); v != "" {
		c.Level = LogLevel(v)
	}
	if v := os.Getenv("LOGGING_FORMAT"); v != "" {
		c.Format = LogFormat(v)
	}
}

func (c *LoggingConfig) loadDefaults() {
	if c.Level == "" {
		c.Level = LogLevelInfo
	}
	if c.Format == "" {
		c.Format = LogFormatJSON
	}
}

func (c *LoggingConfig) validate() error {
	if err := c.Level.Validate(); err != nil {
		return err
	}
	if err := c.Format.Validate(); err != nil {
		return err
	}
	return nil
}

func (c *LoggingConfig) Merge(overlay *LoggingConfig) {
	if overlay.Level != "" {
		c.Level = overlay.Level
	}
	if overlay.Format != "" {
		c.Format = overlay.Format
	}
}
```

### Step 2.8: CORS Configuration

**File**: `internal/config/cors.go`

```go
package config

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

func (c *CORSConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()
	return nil
}

func (c *CORSConfig) loadDefaults() {
	if len(c.AllowedMethods) == 0 {
		c.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(c.AllowedHeaders) == 0 {
		c.AllowedHeaders = []string{"Content-Type", "Authorization"}
	}
	if c.MaxAge == 0 {
		c.MaxAge = 3600
	}
}

func (c *CORSConfig) loadEnv() {
	if v := os.Getenv("CORS_ENABLED"); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			c.Enabled = enabled
		}
	}

	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		origins := strings.Split(v, ",")
		c.Origins = make([]string, 0, len(origins))
		for _, origin := range origins {
			if trimmed := strings.TrimSpace(origin); trimmed != "" {
				c.Origins = append(c.Origins, trimmed)
			}
		}
	}

	if v := os.Getenv("CORS_ALLOWED_METHODS"); v != "" {
		methods := strings.Split(v, ",")
		c.AllowedMethods = make([]string, 0, len(methods))
		for _, method := range methods {
			if trimmed := strings.TrimSpace(method); trimmed != "" {
				c.AllowedMethods = append(c.AllowedMethods, trimmed)
			}
		}
	}

	if v := os.Getenv("CORS_ALLOWED_HEADERS"); v != "" {
		headers := strings.Split(v, ",")
		c.AllowedHeaders = make([]string, 0, len(headers))
		for _, header := range headers {
			if trimmed := strings.TrimSpace(header); trimmed != "" {
				c.AllowedHeaders = append(c.AllowedHeaders, trimmed)
			}
		}
	}

	if v := os.Getenv("CORS_ALLOW_CREDENTIALS"); v != "" {
		if creds, err := strconv.ParseBool(v); err == nil {
			c.AllowCredentials = creds
		}
	}

	if v := os.Getenv("CORS_MAX_AGE"); v != "" {
		if maxAge, err := strconv.Atoi(v); err == nil {
			c.MaxAge = maxAge
		}
	}
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
```

**Note**: Configuration is ephemeral - no config.System needed. Config is loaded once at startup, validated, and passed to systems. Changes require service restart (Kubernetes rolling updates provide zero downtime).

---

## Phase 3: Server System

**Pattern**: Service is the composition root - it wires up all internal systems but delegates implementation to them.

### Step 3.1: Logger System

**File**: `internal/logger/logger.go`

```go
package logger

import (
	"log/slog"
	"os"

	"github.com/JaimeStill/agent-lab/internal/config"
)

type System interface {
	Logger() *slog.Logger
}

type logger struct {
	logger *slog.Logger
}

func New(cfg *config.LoggingConfig) System {
	opts := &slog.HandlerOptions{
		Level: cfg.Level.ToSlogLevel(),
	}

	var handler slog.Handler
	if cfg.Format == config.LogFormatJSON {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return &logger{
		logger: slog.New(handler),
	}
}

func (l *logger) Logger() *slog.Logger {
	return l.logger
}
```

### Step 3.2: Route System

**File**: `internal/routes/routes.go`

```go
package routes

import (
	"log/slog"
	"net/http"
)

type System interface {
	RegisterRoute(route Route)
	RegisterGroup(group Group)
	Build() http.Handler
}

type routes struct {
	routes []Route
	groups []Group
	logger *slog.Logger
}

func New(logger *slog.Logger) System {
	return &routes{
		logger: logger,
		routes: []Route{},
		groups: []Group{},
	}
}

func (r *routes) RegisterRoute(route Route) {
	r.routes = append(r.routes, route)
}

func (r *routes) RegisterGroup(group Group) {
	r.groups = append(r.groups, group)
}

func (r *routes) Build() http.Handler {
	mux := http.NewServeMux()

	for _, route := range r.routes {
		mux.HandleFunc(route.Method+" "+route.Pattern, route.Handler)
	}

	for _, group := range r.groups {
		r.registerGroup(mux, group)
	}

	return mux
}

func (r *routes) registerGroup(mux *http.ServeMux, group Group) {
	for _, route := range group.Routes {
		pattern := group.Prefix + route.Pattern
		mux.HandleFunc(route.Method+" "+pattern, route.Handler)
	}
}
```

**File**: `internal/routes/group.go`

```go
package routes

import "net/http"

type Group struct {
	Prefix      string
	Tags        []string
	Description string
	Routes      []Route
}

type Route struct {
	Method  string
	Pattern string
	Handler http.HandlerFunc
}
```

### Step 3.3: Middleware System

**Pattern**: Middleware system is generic and composable. Specific middleware (Logger, CORS) are separate, reusable functions.

**File**: `internal/middleware/middleware.go`

```go
package middleware

import "net/http"

type System interface {
	Use(mw func(http.Handler) http.Handler)
	Apply(handler http.Handler) http.Handler
}

type middleware struct {
	stack []func(http.Handler) http.Handler
}

func New() System {
	return &middleware{
		stack: []func(http.Handler) http.Handler{},
	}
}

func (m *middleware) Use(mw func(http.Handler) http.Handler) {
	m.stack = append(m.stack, mw)
}

func (m *middleware) Apply(handler http.Handler) http.Handler {
	for i := len(m.stack) - 1; i >= 0; i-- {
		handler = m.stack[i](handler)
	}
	return handler
}
```

**File**: `internal/middleware/logger.go`

```go
package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Info("request",
				"method", r.Method,
				"uri", r.URL.RequestURI(),
				"addr", r.RemoteAddr,
				"duration", time.Since(start))
		})
	}
}
```

**File**: `internal/middleware/cors.go`

```go
package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/JaimeStill/agent-lab/internal/config"
)

func CORS(cfg *config.CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.Enabled || len(cfg.Origins) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			allowed := false
			for _, allowedOrigin := range cfg.Origins {
				if origin == allowedOrigin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))

				if cfg.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				if cfg.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.MaxAge))
				}
			}

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
```

### Step 3.4: Server System

**File**: `internal/server/server.go`

```go
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/JaimeStill/agent-lab/internal/config"
)

type System interface {
	Start(ctx context.Context, wg *sync.WaitGroup) error
	Stop(ctx context.Context) error
}

type server struct {
	http            *http.Server
	shutdownTimeout time.Duration
	logger          *slog.Logger
}

func New(cfg *config.ServerConfig, handler http.Handler, logger *slog.Logger) System {
	return &server{
		http: &http.Server{
			Addr:         cfg.Addr(),
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeoutDuration(),
			WriteTimeout: cfg.WriteTimeoutDuration(),
		},
		shutdownTimeout: cfg.ShutdownTimeoutDuration(),
		logger:          logger,
	}
}

func (s *server) Start(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(1)

	go func() {
		defer wg.Done()

		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("server error", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		s.logger.Info("shutting down server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		if err := s.http.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("server shutdown error", "error", err)
		} else {
			s.logger.Info("server shutdown complete")
		}
	}()

	s.logger.Info("server started", "addr", s.http.Addr)
	return nil
}

func (s *server) Stop(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
```

### Step 3.5: Service Composition

**Pattern**: Service-level routes are co-located with service composition in `cmd/service/`. Domain routes (added in future sessions) remain in their respective domain packages.

**File**: `cmd/service/routes.go`

```go
package main

import (
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/routes"
)

func registerRoutes(r routes.System) {
	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: "/healthz",
		Handler: handleHealthCheck,
	})
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
```

**File**: `cmd/service/middleware.go`

```go
package main

import (
	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/logger"
	"github.com/JaimeStill/agent-lab/internal/middleware"
)

func buildMiddleware(loggerSys logger.System, cfg *config.Config) middleware.System {
	middlewareSys := middleware.New()
	middlewareSys.Use(middleware.Logger(loggerSys.Logger()))
	middlewareSys.Use(middleware.CORS(&cfg.CORS))
	return middlewareSys
}
```

**File**: `cmd/service/service.go`

```go
package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/logger"
	"github.com/JaimeStill/agent-lab/internal/middleware"
	"github.com/JaimeStill/agent-lab/internal/routes"
	"github.com/JaimeStill/agent-lab/internal/server"
)

type Service struct {
	ctx        context.Context
	cancel     context.CancelFunc
	shutdownWg sync.WaitGroup

	logger logger.System
	server server.System
}

func NewService(cfg *config.Config) (*Service, error) {
	ctx, cancel := context.WithCancel(context.Background())

	loggerSys := logger.New(&cfg.Logging)
	routeSys := routes.New(loggerSys.Logger())

	middlewareSys := buildMiddleware(loggerSys, cfg)

	registerRoutes(routeSys)

	handler := middlewareSys.Apply(routeSys.Build())

	serverSys := server.New(&cfg.Server, handler, loggerSys.Logger())

	return &Service{
		ctx:    ctx,
		cancel: cancel,
		logger: loggerSys,
		server: serverSys,
	}, nil
}

func (s *Service) Start() error {
	s.logger.Logger().Info("starting service")

	if err := s.server.Start(s.ctx, &s.shutdownWg); err != nil {
		return fmt.Errorf("server start failed: %w", err)
	}

	s.logger.Logger().Info("service started")
	return nil
}

func (s *Service) Shutdown(ctx context.Context) error {
	s.logger.Logger().Info("initiating shutdown")

	s.cancel()

	done := make(chan struct{})
	go func() {
		s.shutdownWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Logger().Info("all subsystems shut down successfully")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout: %w", ctx.Err())
	}
}
```

### Step 3.6: Main Entry Point

**File**: `cmd/service/main.go`

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JaimeStill/agent-lab/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config load failed:", err)
	}

	if err := cfg.Finalize(); err != nil {
		log.Fatal("config finalize failed:", err)
	}

	svc, err := NewService(cfg)
	if err != nil {
		log.Fatal("service init failed:", err)
	}

	if err := svc.Start(); err != nil {
		log.Fatal("service start failed:", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeoutDuration())
	defer cancel()

	if err := svc.Shutdown(shutdownCtx); err != nil {
		log.Fatal("shutdown failed:", err)
	}

	log.Println("service stopped gracefully")
}
```

---

## Phase 4: Dependencies

### Step 4.1: Go Module Initialization

**File**: `go.mod`

```go
module github.com/JaimeStill/agent-lab

go 1.23

require github.com/pelletier/go-toml/v2 v2.2.3
```

---

## Phase 5: README Updates

### Step 5.1: Update Quick Start

Update the **Quick Start** section in `README.md` to reflect the current state:

**File**: `README.md` (update Quick Start section)

```markdown
## Quick Start

### Prerequisites

- **Go 1.23** or later
- **Docker & Docker Compose**

### Development Setup

1. **Start PostgreSQL container**:
   ```bash
   docker compose up -d
   ```

2. **Copy environment template**:
   ```bash
   cp .env.example .env
   ```

3. **Run the server**:
   ```bash
   go run cmd/service/main.go
   ```

4. **Verify health check**:
   ```bash
   curl http://localhost:8080/healthz
   # Expected: OK
   ```

5. **Graceful shutdown**:
   ```bash
   # Press Ctrl+C in terminal running server
   # Expected: "server stopped gracefully"
   ```

### Useful Commands

**Start PostgreSQL**:
```bash
docker compose up -d
```

**Stop PostgreSQL**:
```bash
docker compose down
```

**View PostgreSQL logs**:
```bash
docker compose logs -f postgres
```

**Check service health**:
```bash
docker compose ps
```
```

---

## Implementation Order

Execute in this exact order to maintain dependency flow:

1. **Phase 1**: Docker & Database Setup (Steps 1.1-1.3)
2. **Phase 2**: Configuration Management (Steps 2.1-2.2)
3. **Phase 3**: Server System (Steps 3.1-3.2)
4. **Phase 4**: Dependencies (Step 4.1)
5. **Phase 5**: README Updates (Step 5.1)

After completing all phases:
```bash
# 1. Start PostgreSQL (choose one)
docker compose up -d                          # PostgreSQL only
docker compose -f docker-compose.dev.yml up -d  # PostgreSQL + Ollama

# 2. Install dependencies
go mod tidy

# 3. Run server
go run cmd/server/main.go

# 4. Test health check (in another terminal)
curl http://localhost:8080/healthz

# 5. Test graceful shutdown (Ctrl+C in server terminal)
```

---

## Validation Checklist

- [ ] PostgreSQL container starts successfully
- [ ] Configuration loads from `config.toml`
- [ ] Environment variables override config values
- [ ] Server starts without errors
- [ ] Health check endpoint returns `200 OK`
- [ ] Requests are logged with method, URI, duration
- [ ] CORS headers applied when origins configured
- [ ] `Ctrl+C` triggers graceful shutdown
- [ ] Shutdown completes within 30 seconds
- [ ] "server stopped gracefully" message appears

---

## Next Session

**Session 1b: Database & Query Infrastructure** will add:
- Database system with connection pool
- Migrations infrastructure
- Query builder (pkg/query)
- Pagination utilities (pkg/pagination)
