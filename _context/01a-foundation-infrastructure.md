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
- Config interface pattern for stateful systems (Server)
- Cold Start (state initialization) vs Hot Start (process activation)
- Composition root pattern (cmd/server owns all subsystems)

## Success Criteria

- `docker compose up -d` starts PostgreSQL 17 container
- Server loads configuration from TOML + environment variables
- `go run cmd/server/main.go` starts server successfully
- `curl http://localhost:8080/healthz` returns `200 OK`
- `Ctrl+C` triggers graceful shutdown

---

## Phase 1: Docker & Database Setup

### Step 1.1: Docker Compose Configuration

**File**: `docker-compose.yml`

```yaml
services:
  postgres:
    image: postgres:17-alpine
    container_name: agent-lab-postgres
    restart: unless-stopped
    ports:
      - "5432:5432"
    volumes:
      - ./.data/postgres:/var/lib/postgresql/data
    environment:
      POSTGRES_DB: ${POSTGRES_DB:-agent_lab}
      POSTGRES_USER: ${POSTGRES_USER:-agent_lab}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-agent_lab}
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-agent_lab}"]
      interval: 10s
      timeout: 5s
      retries: 5
```

### Step 1.2: Environment Template

**File**: `.env.example`

```env
POSTGRES_DB=agent_lab
POSTGRES_USER=agent_lab
POSTGRES_PASSWORD=agent_lab
POSTGRES_PORT=5432

SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s

LOGGING_LEVEL=info
LOGGING_FORMAT=json

CORS_ORIGINS_0=http://localhost:3000
```

### Step 1.3: Git Ignore

**File**: `.gitignore`

```
.env
.data/
config.local.toml
```

---

## Phase 2: Configuration Management

### Step 2.1: Base Configuration

**File**: `config.toml`

```toml
[server]
host = "0.0.0.0"
port = 8080
read_timeout = "30s"
write_timeout = "30s"

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

[logging]
level = "info"
format = "json"

[cors]
origins = []
```

### Step 2.2: Server Configuration Struct

**File**: `cmd/server/config.go`

```go
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Server   ServerConfig   `toml:"server"`
	Database DatabaseConfig `toml:"database"`
	Logging  LoggingConfig  `toml:"logging"`
	CORS     CORSConfig     `toml:"cors"`
}

type ServerConfig struct {
	Host         string `toml:"host"`
	Port         int    `toml:"port"`
	ReadTimeout  string `toml:"read_timeout"`
	WriteTimeout string `toml:"write_timeout"`
}

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

type LoggingConfig struct {
	Level  string `toml:"level"`
	Format string `toml:"format"`
}

type CORSConfig struct {
	Origins []string `toml:"origins"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	applyEnvOverrides(&cfg)
	cfg.finalize()

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("SERVER_READ_TIMEOUT"); v != "" {
		cfg.Server.ReadTimeout = v
	}
	if v := os.Getenv("SERVER_WRITE_TIMEOUT"); v != "" {
		cfg.Server.WriteTimeout = v
	}

	if v := os.Getenv("DATABASE_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("DATABASE_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Database.Port = port
		}
	}
	if v := os.Getenv("DATABASE_NAME"); v != "" {
		cfg.Database.Name = v
	}
	if v := os.Getenv("DATABASE_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("DATABASE_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}

	if v := os.Getenv("LOGGING_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv("LOGGING_FORMAT"); v != "" {
		cfg.Logging.Format = v
	}

	i := 0
	for {
		key := fmt.Sprintf("CORS_ORIGINS_%d", i)
		if v := os.Getenv(key); v != "" {
			if i == 0 {
				cfg.CORS.Origins = []string{}
			}
			cfg.CORS.Origins = append(cfg.CORS.Origins, v)
			i++
		} else {
			break
		}
	}
}

func (c *Config) finalize() {
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.ReadTimeout == "" {
		c.Server.ReadTimeout = "30s"
	}
	if c.Server.WriteTimeout == "" {
		c.Server.WriteTimeout = "30s"
	}

	if c.Database.Host == "" {
		c.Database.Host = "localhost"
	}
	if c.Database.Port == 0 {
		c.Database.Port = 5432
	}
	if c.Database.MaxOpenConns == 0 {
		c.Database.MaxOpenConns = 25
	}
	if c.Database.MaxIdleConns == 0 {
		c.Database.MaxIdleConns = 5
	}
	if c.Database.ConnMaxLifetime == "" {
		c.Database.ConnMaxLifetime = "15m"
	}
	if c.Database.ConnTimeout == "" {
		c.Database.ConnTimeout = "5s"
	}

	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
}

func (c *Config) validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if _, err := time.ParseDuration(c.Server.ReadTimeout); err != nil {
		return fmt.Errorf("invalid read_timeout: %w", err)
	}
	if _, err := time.ParseDuration(c.Server.WriteTimeout); err != nil {
		return fmt.Errorf("invalid write_timeout: %w", err)
	}

	if c.Database.Name == "" {
		return fmt.Errorf("database name required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user required")
	}

	if _, err := time.ParseDuration(c.Database.ConnMaxLifetime); err != nil {
		return fmt.Errorf("invalid conn_max_lifetime: %w", err)
	}
	if _, err := time.ParseDuration(c.Database.ConnTimeout); err != nil {
		return fmt.Errorf("invalid conn_timeout: %w", err)
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[strings.ToLower(c.Logging.Level)] {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
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

func (c *LoggingConfig) LevelValue() string {
	return strings.ToLower(c.Level)
}
```

---

## Phase 3: Server System

### Step 3.1: Server Structure

**File**: `cmd/server/server.go`

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type Server struct {
	config *Config
	logger *slog.Logger
	server *http.Server
}

func NewServer(cfg *Config) (*Server, error) {
	logger := createLogger(cfg.Logging)

	srv := &Server{
		config: cfg,
		logger: logger,
	}

	return srv, nil
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("starting server",
		"host", s.config.Server.Host,
		"port", s.config.Server.Port)

	handler := s.buildHandler()

	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port),
		Handler:      handler,
		ReadTimeout:  s.config.Server.ReadTimeoutDuration(),
		WriteTimeout: s.config.Server.WriteTimeoutDuration(),
	}

	shutdown := make(chan error, 1)
	go func() {
		<-ctx.Done()
		s.logger.Info("shutdown signal received")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.server.Shutdown(shutdownCtx); err != nil {
			shutdown <- fmt.Errorf("server shutdown: %w", err)
			return
		}
		shutdown <- nil
	}()

	s.logger.Info("server started", "addr", s.server.Addr)

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server failed: %w", err)
	}

	return <-shutdown
}

func (s *Server) buildHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealthCheck)

	return s.applyCORS(s.logRequests(mux))
}

func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("request",
			"method", r.Method,
			"uri", r.URL.RequestURI(),
			"addr", r.RemoteAddr,
			"duration", time.Since(start))
	})
}

func (s *Server) applyCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(s.config.CORS.Origins) > 0 {
			origin := r.Header.Get("Origin")
			for _, allowed := range s.config.CORS.Origins {
				if origin == allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
					break
				}
			}

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func createLogger(cfg LoggingConfig) *slog.Logger {
	var level slog.Level
	switch cfg.LevelValue() {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
```

### Step 3.2: Main Entry Point

**File**: `cmd/server/main.go`

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg, err := LoadConfig("config.toml")
	if err != nil {
		log.Fatal("config load failed:", err)
	}

	srv, err := NewServer(cfg)
	if err != nil {
		log.Fatal("server init failed:", err)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if err := srv.Start(ctx); err != nil {
		log.Fatal("server failed:", err)
	}

	log.Println("server stopped gracefully")
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
   go run cmd/server/main.go
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
# 1. Start PostgreSQL
docker compose up -d

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
