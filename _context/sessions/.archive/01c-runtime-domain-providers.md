# Session 1c: Runtime/Domain System Separation + Providers System

**Status**: Implementation Guide
**Milestone**: 01 - Foundation & Infrastructure

## Overview

This session introduces the Runtime/Domain system separation pattern and implements the first domain system (Providers). The pattern establishes clear boundaries between long-running infrastructure (Runtime) and stateless domain systems (Domain).

### Two System Categories

| Category | Characteristics | Examples |
|----------|----------------|----------|
| **Runtime Systems** | Long-running, lifecycle-managed, application-scoped | Database |
| **Domain Systems** | Stateless, request-scoped behavior, no lifecycle | Providers, Agents |

### Key Principles

1. **Runtime holds System interfaces** - Domain systems call methods like `runtime.Database.Connection()` to get what they need
2. **Domain systems are independent** - They only depend on Runtime systems, not on each other
3. **Domain systems pre-initialized** - Created at startup in `NewDomain()`, stored in Server struct

### Configuration Ownership

| Prefix | Owner | Examples |
|--------|-------|----------|
| `SERVICE_` | `main()` process-level | `SERVICE_SHUTDOWN_TIMEOUT` |
| `SERVER_` | HTTP server (`cmd/server/`) | `SERVER_HOST`, `SERVER_PORT` |

---

## Phase 0: Structural Refactor

This phase:
- Simplifies config API (`Load()` includes finalization)
- Renames `cmd/service/` to `cmd/server/`
- Absorbs `internal/server/` into `cmd/server/http.go`
- Removes `internal/logger/` (logger is not a System - use `*slog.Logger` directly)

### 0.1 Simplify `internal/config/config.go`

Consolidate `Load()` and `Finalize()` - config should always be valid after loading:

```go
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/pelletier/go-toml/v2"
)

const (
	BaseConfigFile            = "config.toml"
	OverlayConfigPattern      = "config.%s.toml"
	EnvServiceEnv             = "SERVICE_ENV"
	EnvServiceShutdownTimeout = "SERVICE_SHUTDOWN_TIMEOUT"
)

type Config struct {
	Server          ServerConfig      `toml:"server"`
	Database        DatabaseConfig    `toml:"database"`
	Logging         LoggingConfig     `toml:"logging"`
	CORS            CORSConfig        `toml:"cors"`
	Pagination      pagination.Config `toml:"pagination"`
	ShutdownTimeout string            `toml:"shutdown_timeout"`
}

func (c *Config) ShutdownTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.ShutdownTimeout)
	return d
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

	if err := cfg.finalize(); err != nil {
		return nil, fmt.Errorf("finalize config: %w", err)
	}

	return cfg, nil
}

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

func (c *Config) finalize() error {
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
```

### 0.2 Rename Directory

```bash
mv cmd/service cmd/server
```

### 0.3 Create `cmd/server/http.go` (NEW)

Move HTTP server logic from `internal/server/server.go`:

```go
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/lifecycle"
)

type httpServer struct {
	http            *http.Server
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

func newHTTPServer(cfg *config.ServerConfig, handler http.Handler, logger *slog.Logger) *httpServer {
	return &httpServer{
		http: &http.Server{
			Addr:         cfg.Addr(),
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeoutDuration(),
			WriteTimeout: cfg.WriteTimeoutDuration(),
		},
		logger:          logger.With("system", "http"),
		shutdownTimeout: cfg.ShutdownTimeoutDuration(),
	}
}

func (s *httpServer) Start(lc *lifecycle.Coordinator) error {
	go func() {
		s.logger.Info("server listening", "addr", s.http.Addr)
		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("server error", "error", err)
		}
	}()

	lc.OnShutdown(func() {
		<-lc.Context().Done()
		s.logger.Info("shutting down server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		if err := s.http.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("server shutdown error", "error", err)
		} else {
			s.logger.Info("server shutdown complete")
		}
	})

	return nil
}
```

### 0.4 Delete `internal/server/` and `internal/logger/`

```bash
rm -rf internal/server
rm -rf internal/logger
```

### 0.5 Create `cmd/server/logging.go` (NEW)

Move logger initialization logic from `internal/logger/`:

```go
package main

import (
	"log/slog"
	"os"

	"github.com/JaimeStill/agent-lab/internal/config"
)

func newLogger(cfg *config.LoggingConfig) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: cfg.Level.ToSlogLevel(),
	}

	var handler slog.Handler
	if cfg.Format == config.LogFormatJSON {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
```

### 0.6 Update `cmd/server/service.go`

Remove `internal/server` and `internal/logger` imports, use local helpers:

```go
package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/database"
	"github.com/JaimeStill/agent-lab/internal/lifecycle"
	"github.com/JaimeStill/agent-lab/internal/routes"
)

type Service struct {
	lifecycle *lifecycle.Coordinator
	logger    *slog.Logger
	database  database.System
	server    *httpServer
}

func NewService(cfg *config.Config) (*Service, error) {
	lc := lifecycle.New()
	logger := newLogger(&cfg.Logging)
	routeSys := routes.New(logger)

	dbSys, err := database.New(&cfg.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("database init failed: %w", err)
	}

	middlewareSys := buildMiddleware(logger, cfg)
	registerRoutes(routeSys, lc)
	handler := middlewareSys.Apply(routeSys.Build())

	serverSys := newHTTPServer(&cfg.Server, handler, logger)

	return &Service{
		lifecycle: lc,
		logger:    logger,
		database:  dbSys,
		server:    serverSys,
	}, nil
}

func (s *Service) Start() error {
	s.logger.Info("starting service")

	if err := s.database.Start(s.lifecycle); err != nil {
		return fmt.Errorf("database start failed: %w", err)
	}

	if err := s.server.Start(s.lifecycle); err != nil {
		return fmt.Errorf("server start failed: %w", err)
	}

	go func() {
		s.lifecycle.WaitForStartup()
		s.logger.Info("all subsystems ready")
	}()

	s.logger.Info("service started")
	return nil
}

func (s *Service) Shutdown(timeout time.Duration) error {
	s.logger.Info("initiating shutdown")

	if err := s.lifecycle.Shutdown(timeout); err != nil {
		return err
	}

	s.logger.Info("all subsystems shut down successfully")
	return nil
}
```

### 0.7 Update `cmd/server/main.go`

Simplify to use consolidated `Load()`:

```go
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/JaimeStill/agent-lab/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config load failed:", err)
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

	if err := svc.Shutdown(cfg.ShutdownTimeoutDuration()); err != nil {
		log.Fatal("shutdown failed:", err)
	}

	log.Println("service stopped gracefully")
}
```

### 0.8 Update Test Directory

```bash
mv tests/cmd_service tests/cmd_server
```

Update test file package declaration if needed.

### 0.9 Verify Phase 0

```bash
go build ./...
go test ./tests/cmd_server/...
```

---

## Phase 1: Runtime/Domain Refactoring

### 1.1 Create `cmd/server/runtime.go` (NEW)

```go
package main

import (
	"fmt"
	"log/slog"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/database"
	"github.com/JaimeStill/agent-lab/internal/lifecycle"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
)

type Runtime struct {
	Lifecycle  *lifecycle.Coordinator
	Logger     *slog.Logger
	Database   database.System
	Pagination pagination.Config
}

func NewRuntime(cfg *config.Config) (*Runtime, error) {
	lc := lifecycle.New()
	logger := newLogger(&cfg.Logging)

	dbSys, err := database.New(&cfg.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("database init failed: %w", err)
	}

	return &Runtime{
		Lifecycle:  lc,
		Logger:     logger,
		Database:   dbSys,
		Pagination: cfg.Pagination,
	}, nil
}

func (r *Runtime) Start() error {
	if err := r.Database.Start(r.Lifecycle); err != nil {
		return fmt.Errorf("database start failed: %w", err)
	}
	return nil
}
```

### 1.2 Create `cmd/server/domain.go` (NEW)

Initially empty - will be populated in Phase 4 after Providers system is created.

```go
package main

type Domain struct {
}

func NewDomain(runtime *Runtime) *Domain {
	return &Domain{}
}
```

### 1.3 Rename and Update `cmd/server/service.go` → `cmd/server/server.go`

```bash
mv cmd/server/service.go cmd/server/server.go
```

Replace the entire file:

```go
package main

import (
	"fmt"
	"time"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/routes"
)

type Server struct {
	runtime *Runtime
	domain  *Domain
	http    *httpServer
}

func NewServer(cfg *config.Config) (*Server, error) {
	runtime, err := NewRuntime(cfg)
	if err != nil {
		return nil, err
	}

	domain := NewDomain(runtime)

	routeSys := routes.New(runtime.Logger)
	middlewareSys := buildMiddleware(runtime, cfg)

	registerRoutes(routeSys, runtime, domain)
	handler := middlewareSys.Apply(routeSys.Build())

	httpSrv := newHTTPServer(&cfg.Server, handler, runtime.Logger)

	return &Server{
		runtime: runtime,
		domain:  domain,
		http:    httpSrv,
	}, nil
}

func (s *Server) Start() error {
	s.runtime.Logger.Info("starting server")

	if err := s.runtime.Start(); err != nil {
		return err
	}

	if err := s.http.Start(s.runtime.Lifecycle); err != nil {
		return fmt.Errorf("http server start failed: %w", err)
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

### 1.4 Update `cmd/server/main.go`

Update to use `Server` instead of `Service`:

```go
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/JaimeStill/agent-lab/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config load failed:", err)
	}

	srv, err := NewServer(cfg)
	if err != nil {
		log.Fatal("server init failed:", err)
	}

	if err := srv.Start(); err != nil {
		log.Fatal("server start failed:", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan

	if err := srv.Shutdown(cfg.ShutdownTimeoutDuration()); err != nil {
		log.Fatal("shutdown failed:", err)
	}

	log.Println("server stopped gracefully")
}
```

### 1.5 Update `cmd/server/routes.go`

Update function signature to receive Runtime and Domain:

```go
package main

import (
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/routes"
)

func registerRoutes(r routes.System, runtime *Runtime, domain *Domain) {
	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: "/healthz",
		Handler: handleHealthCheck,
	})

	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: "/readyz",
		Handler: func(w http.ResponseWriter, req *http.Request) {
			handleReadinessCheck(w, runtime.Lifecycle)
		},
	})
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleReadinessCheck(w http.ResponseWriter, ready interface{ Ready() bool }) {
	if !ready.Ready() {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("NOT READY"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("READY"))
}
```

### 1.6 Update `cmd/server/middleware.go`

Update to receive Runtime instead of individual dependencies:

```go
package main

import (
	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/middleware"
)

func buildMiddleware(runtime *Runtime, cfg *config.Config) middleware.System {
	middlewareSys := middleware.New()
	middlewareSys.Use(middleware.Logger(runtime.Logger))
	middlewareSys.Use(middleware.CORS(&cfg.CORS))
	return middlewareSys
}
```

---

## Phase 2: Database Schema

### 2.1 Create `cmd/migrate/migrations/000002_providers.up.sql` (NEW)

```sql
CREATE TABLE providers (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL UNIQUE,
    config     JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_providers_name ON providers(name);
CREATE INDEX idx_providers_created_at ON providers(created_at DESC);
```

### 2.2 Create `cmd/migrate/migrations/000002_providers.down.sql` (NEW)

```sql
DROP INDEX IF EXISTS idx_providers_created_at;
DROP INDEX IF EXISTS idx_providers_name;
DROP TABLE IF EXISTS providers;
```

---

## Phase 3: Providers Domain System

### 3.1 Create `internal/providers/provider.go` (NEW)

```go
package providers

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Provider struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Config    json.RawMessage `json:"config"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type CreateCommand struct {
	Name   string          `json:"name"`
	Config json.RawMessage `json:"config"`
}

type UpdateCommand struct {
	Name   string          `json:"name"`
	Config json.RawMessage `json:"config"`
}
```

### 3.2 Create `internal/providers/errors.go` (NEW)

```go
package providers

import "errors"

var (
	ErrNotFound  = errors.New("provider not found")
	ErrDuplicate = errors.New("provider name already exists")
)
```

### 3.3 Create `internal/providers/projection.go` (NEW)

```go
package providers

import "github.com/JaimeStill/agent-lab/pkg/query"

var projection = query.NewProjectionMap("public", "providers", "p").
	Project("id", "Id").
	Project("name", "Name").
	Project("config", "Config").
	Project("created_at", "CreatedAt").
	Project("updated_at", "UpdatedAt")
```

### 3.4 Create `internal/providers/system.go` (NEW)

```go
package providers

import (
	"context"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

type System interface {
	Create(ctx context.Context, cmd CreateCommand) (*Provider, error)
	Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Provider, error)
	Delete(ctx context.Context, id uuid.UUID) error
	FindByID(ctx context.Context, id uuid.UUID) (*Provider, error)
	Search(ctx context.Context, page pagination.PageRequest) (*pagination.PageResult[Provider], error)
}
```

### 3.5 Create `internal/providers/repository.go` (NEW)

```go
package providers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/agent-lab/pkg/query"
	goconfig "github.com/JaimeStill/go-agents/pkg/config"
	goproviders "github.com/JaimeStill/go-agents/pkg/providers"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type repository struct {
	db         *sql.DB
	logger     *slog.Logger
	pagination pagination.Config
}

func New(db *sql.DB, logger *slog.Logger, pagination pagination.Config) System {
	return &repository{
		db:         db,
		logger:     logger.With("system", "providers"),
		pagination: pagination,
	}
}

func (r *repository) Create(ctx context.Context, cmd CreateCommand) (*Provider, error) {
	if err := r.validateConfig(cmd.Config); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	q := `
		INSERT INTO providers (name, config)
		VALUES ($1, $2)
		RETURNING id, name, config, created_at, updated_at`

	var p Provider
	err = tx.QueryRowContext(ctx, q, cmd.Name, cmd.Config).Scan(
		&p.ID, &p.Name, &p.Config, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if isDuplicateError(err) {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("insert provider: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	r.logger.Info("provider created", "id", p.ID, "name", p.Name)
	return &p, nil
}

func (r *repository) Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Provider, error) {
	if err := r.validateConfig(cmd.Config); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	q := `
		UPDATE providers
		SET name = $1, config = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, name, config, created_at, updated_at`

	var p Provider
	err = tx.QueryRowContext(ctx, q, cmd.Name, cmd.Config, id).Scan(
		&p.ID, &p.Name, &p.Config, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		if isDuplicateError(err) {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("update provider: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	r.logger.Info("provider updated", "id", p.ID, "name", p.Name)
	return &p, nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, "DELETE FROM providers WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete provider: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	r.logger.Info("provider deleted", "id", id)
	return nil
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*Provider, error) {
	q, args := query.NewBuilder(projection, "Name").BuildSingle("Id", id)

	var p Provider
	err := r.db.QueryRowContext(ctx, q, args...).Scan(
		&p.ID, &p.Name, &p.Config, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query provider: %w", err)
	}

	return &p, nil
}

func (r *repository) Search(ctx context.Context, page pagination.PageRequest) (*pagination.PageResult[Provider], error) {
	page.Normalize(r.pagination)

	qb := query.NewBuilder(projection, "Name").
		WhereSearch(page.Search, "Name").
		OrderBy(page.SortBy, page.Descending)

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count providers: %w", err)
	}

	pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	rows, err := r.db.QueryContext(ctx, pageSQL, pageArgs...)
	if err != nil {
		return nil, fmt.Errorf("query providers: %w", err)
	}
	defer rows.Close()

	providers := make([]Provider, 0)
	for rows.Next() {
		var p Provider
		if err := rows.Scan(&p.ID, &p.Name, &p.Config, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan provider: %w", err)
		}
		providers = append(providers, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	result := pagination.NewPageResult(providers, total, page.Page, page.PageSize)
	return &result, nil
}

func (r *repository) validateConfig(config json.RawMessage) error {
	var cfg goconfig.ProviderConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid config structure: %w", err)
	}

	if _, err := goproviders.Create(&cfg); err != nil {
		return fmt.Errorf("invalid provider config: %w", err)
	}

	return nil
}

func isDuplicateError(err error) bool {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		return pgErr.Code == "23505"
	}
	return false
}
```

### 3.6 Create `internal/providers/handlers.go` (NEW)

```go
package providers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

func HandleCreate(w http.ResponseWriter, r *http.Request, sys System, logger *slog.Logger) {
	var cmd CreateCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		respondError(w, logger, http.StatusBadRequest, err)
		return
	}

	result, err := sys.Create(r.Context(), cmd)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrDuplicate) {
			status = http.StatusConflict
		}
		respondError(w, logger, status, err)
		return
	}

	respondJSON(w, http.StatusCreated, result)
}

func HandleUpdate(w http.ResponseWriter, r *http.Request, sys System, logger *slog.Logger) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, logger, http.StatusBadRequest, err)
		return
	}

	var cmd UpdateCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		respondError(w, logger, http.StatusBadRequest, err)
		return
	}

	result, err := sys.Update(r.Context(), id, cmd)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, ErrDuplicate) {
			status = http.StatusConflict
		}
		respondError(w, logger, status, err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func HandleDelete(w http.ResponseWriter, r *http.Request, sys System, logger *slog.Logger) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, logger, http.StatusBadRequest, err)
		return
	}

	if err := sys.Delete(r.Context(), id); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, logger, status, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func HandleGetByID(w http.ResponseWriter, r *http.Request, sys System, logger *slog.Logger) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, logger, http.StatusBadRequest, err)
		return
	}

	result, err := sys.FindByID(r.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, logger, status, err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func HandleSearch(w http.ResponseWriter, r *http.Request, sys System, logger *slog.Logger) {
	var page pagination.PageRequest
	if err := json.NewDecoder(r.Body).Decode(&page); err != nil {
		respondError(w, logger, http.StatusBadRequest, err)
		return
	}

	result, err := sys.Search(r.Context(), page)
	if err != nil {
		respondError(w, logger, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, logger *slog.Logger, status int, err error) {
	logger.Error("handler error", "error", err, "status", status)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
```

### 3.7 Create `internal/providers/routes.go` (NEW)

```go
package providers

import (
	"log/slog"
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/routes"
)

func Routes(sys System, logger *slog.Logger) routes.Group {
	return routes.Group{
		Prefix:      "/api/providers",
		Tags:        []string{"Providers"},
		Description: "Provider configuration management",
		Routes: []routes.Route{
			{
				Method:  "POST",
				Pattern: "",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					HandleCreate(w, r, sys, logger)
				},
			},
			{
				Method:  "GET",
				Pattern: "/{id}",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					HandleGetByID(w, r, sys, logger)
				},
			},
			{
				Method:  "PUT",
				Pattern: "/{id}",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					HandleUpdate(w, r, sys, logger)
				},
			},
			{
				Method:  "DELETE",
				Pattern: "/{id}",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					HandleDelete(w, r, sys, logger)
				},
			},
			{
				Method:  "POST",
				Pattern: "/search",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					HandleSearch(w, r, sys, logger)
				},
			},
		},
	}
}
```

---

## Phase 4: Integration

### 4.1 Update `cmd/server/domain.go`

Replace with full implementation:

```go
package main

import (
	"github.com/JaimeStill/agent-lab/internal/providers"
)

type Domain struct {
	Providers providers.System
}

func NewDomain(runtime *Runtime) *Domain {
	return &Domain{
		Providers: providers.New(
			runtime.Database.Connection(),
			runtime.Logger,
			runtime.Pagination,
		),
	}
}
```

### 4.2 Update `cmd/server/routes.go`

Add providers routes registration:

```go
package main

import (
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/providers"
	"github.com/JaimeStill/agent-lab/internal/routes"
)

func registerRoutes(r routes.System, runtime *Runtime, domain *Domain) {
	r.RegisterGroup(providers.Routes(domain.Providers, runtime.Logger))

	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: "/healthz",
		Handler: handleHealthCheck,
	})

	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: "/readyz",
		Handler: func(w http.ResponseWriter, req *http.Request) {
			handleReadinessCheck(w, runtime.Lifecycle)
		},
	})
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleReadinessCheck(w http.ResponseWriter, ready interface{ Ready() bool }) {
	if !ready.Ready() {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("NOT READY"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("READY"))
}
```

---

## Dependencies

Add to `go.mod`:

```
github.com/google/uuid v1.6.0
```

Run after implementation:

```bash
go mod tidy
```

---

## Final Directory Structure

```
cmd/
├── server/               # HTTP server entry point
│   ├── main.go               # Entry point, signal handling
│   ├── server.go             # Server struct (composition root)
│   ├── runtime.go            # Runtime struct (lifecycle, database, pagination)
│   ├── domain.go             # Domain struct (providers, agents)
│   ├── http.go               # HTTP server lifecycle
│   ├── logging.go            # Logger initialization helper
│   ├── routes.go             # Route registration
│   └── middleware.go         # Middleware composition
│
└── migrate/              # Migration CLI
    ├── main.go
    └── migrations/

internal/
├── config/
├── database/
├── lifecycle/
├── middleware/
├── routes/
└── providers/            # NEW - Providers domain system
    ├── provider.go
    ├── errors.go
    ├── projection.go
    ├── system.go
    ├── repository.go
    ├── handlers.go
    └── routes.go
```

---

## Validation Checklist

After implementation, verify:

- [ ] `go build ./...` succeeds
- [ ] Server starts with Runtime/Domain pattern
- [ ] Existing endpoints work (`/healthz`, `/readyz`)
- [ ] Run migrations: `go run ./cmd/migrate -dsn "..." -up`
- [ ] Provider CRUD operations via API work
- [ ] Provider search with pagination works
- [ ] Graceful shutdown still works
