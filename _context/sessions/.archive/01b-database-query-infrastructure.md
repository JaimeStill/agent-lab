# Session 1b: Database & Query Infrastructure

**Status**: Implementation Guide
**Milestone**: 01 - Foundation & Infrastructure
**Predecessor**: Session 1a (Foundation Infrastructure) - Completed

## Problem Context

Session 1a established the core service infrastructure (config, logger, routes, middleware, server). Session 1b builds the database connectivity and query infrastructure that will support all domain systems (Providers, Agents, Workflows).

**Current State**:
- `DatabaseConfig` exists but is unused
- Service has no database connectivity
- No query building utilities exist
- Only `/healthz` endpoint (liveness)

**Target State**:
- Database system with Cold Start/Hot Start lifecycle
- `/readyz` endpoint reflecting subsystem operational state
- Migration CLI tool for schema management
- Query builder for parameterized SQL generation
- Pagination utilities for search operations

## Architecture Approach

### Cold Start / Hot Start for Database

Following the established pattern from Session 1a:

- **Cold Start** (`New`): Opens connection pool, configures settings, does NOT verify connectivity
- **Hot Start** (`Start`): Pings database to verify connection, signals readiness
- **Shutdown**: Closes connection pool gracefully when context is cancelled

### Lifecycle Coordinator

The `internal/lifecycle.Coordinator` formalizes startup/shutdown orchestration:

```
Service
    └── lifecycle.Coordinator
            ├── OnStartup(fn)   // Tasks that must complete for readiness
            ├── OnShutdown(fn)  // Cleanup tasks during shutdown
            ├── Ready() bool    // Readiness state
            └── Context()       // Cancellation context

database.Start(lc)
    ├── lc.OnStartup(ping)      // Blocks readiness until ping completes
    └── lc.OnShutdown(close)    // Closes connection on context cancellation

server.Start(lc)
    ├── go ListenAndServe()     // Long-running, not tracked by lifecycle
    └── lc.OnShutdown(shutdown) // Graceful shutdown on context cancellation
```

Readiness is a one-time gate: once all `OnStartup` tasks complete, service is ready.

### Query Builder Layers

1. **ProjectionMap**: Static structure defining table/column mappings
2. **QueryBuilder**: Fluent builder for filters, sorting, pagination
3. **Execution**: Use generated SQL with `database/sql`

---

## Phase 1: Pagination Package

Create `pkg/pagination` first since `internal/config` will import from it.

### 1.1 Create `pkg/pagination/pagination.go`

This package hosts the pagination Config (with TOML tags and config methods) along with request/response types. The Config is imported by `internal/config` rather than duplicating the structure.

```go
package pagination

import (
	"fmt"
	"os"
	"strconv"
)

const (
	EnvDefaultPageSize = "PAGINATION_DEFAULT_PAGE_SIZE"
	EnvMaxPageSize     = "PAGINATION_MAX_PAGE_SIZE"
)

type Config struct {
	DefaultPageSize int `toml:"default_page_size"`
	MaxPageSize     int `toml:"max_page_size"`
}

func (c *Config) Finalize() error {
	c.loadDefaults()
	c.loadEnv()
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
	if c.DefaultPageSize == 0 {
		c.DefaultPageSize = 20
	}
	if c.MaxPageSize == 0 {
		c.MaxPageSize = 100
	}
}

func (c *Config) loadEnv() {
	if v := os.Getenv(EnvDefaultPageSize); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.DefaultPageSize = n
		}
	}
	if v := os.Getenv(EnvMaxPageSize); v != "" {
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

type PageRequest struct {
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	Search     *string `json:"search,omitempty"`
	SortBy     string  `json:"sort_by,omitempty"`
	Descending bool    `json:"descending,omitempty"`
}

func (r *PageRequest) Normalize(cfg Config) {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 {
		r.PageSize = cfg.DefaultPageSize
	}
	if r.PageSize > cfg.MaxPageSize {
		r.PageSize = cfg.MaxPageSize
	}
}

func (r *PageRequest) Offset() int {
	return (r.Page - 1) * r.PageSize
}

type PageResult[T any] struct {
	Data       []T `json:"data"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalPages int `json:"total_pages"`
}

func NewPageResult[T any](data []T, total, page, pageSize int) PageResult[T] {
	totalPages := total / pageSize
	if total%pageSize != 0 {
		totalPages++
	}
	if totalPages < 1 {
		totalPages = 1
	}

	if data == nil {
		data = []T{}
	}

	return PageResult[T]{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}
```

---

## Phase 2: Configuration Updates

### 2.1 Update Root Config (`internal/config/config.go`)

**Pattern Note**: Root-level `loadDefaults()`, `loadEnv()`, and `validate()` only handle root-level fields. Each sub-config's `Finalize()` method handles its own defaults/env/validation.

**Add import** (add to existing imports):
```go
"github.com/JaimeStill/agent-lab/pkg/pagination"
```

**Add `Pagination` field to Config struct**:
```go
type Config struct {
	Server          ServerConfig       `toml:"server"`
	Database        DatabaseConfig     `toml:"database"`
	Logging         LoggingConfig      `toml:"logging"`
	CORS            CORSConfig         `toml:"cors"`
	Pagination      pagination.Config  `toml:"pagination"`
	ShutdownTimeout string             `toml:"shutdown_timeout"`
}
```

**Update Finalize** - add Pagination.Finalize() call:
```go
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
```

**Update Merge** - add Pagination merge:
```go
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
```

### 2.2 Update `config.toml`

Add pagination section:

```toml
shutdown_timeout = "30s"

[server]
host = "0.0.0.0"
port = 8080
read_timeout = "30s"
write_timeout = "30s"
shutdown_timeout = "30s"

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
format = "text"

[cors]
origins = ["http://localhost:3000"]
credentials = true
headers = ["Content-Type", "Authorization"]
methods = ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
max_age = 3600

[pagination]
default_page_size = 20
max_page_size = 100
```

---

## Phase 3: Query Builder Package

### 3.1 Create `pkg/query/projection.go`

```go
package query

import (
	"fmt"
	"strings"
)

type ProjectionMap struct {
	schema     string
	table      string
	alias      string
	columns    map[string]string
	columnList []string
}

func NewProjectionMap(schema, table, alias string) *ProjectionMap {
	return &ProjectionMap{
		schema:     schema,
		table:      table,
		alias:      alias,
		columns:    make(map[string]string),
		columnList: make([]string, 0),
	}
}

func (p *ProjectionMap) Project(column, viewName string) *ProjectionMap {
	qualified := fmt.Sprintf("%s.%s", p.alias, column)
	p.columns[viewName] = qualified
	p.columnList = append(p.columnList, qualified)
	return p
}

func (p *ProjectionMap) Table() string {
	return fmt.Sprintf("%s.%s %s", p.schema, p.table, p.alias)
}

func (p *ProjectionMap) Column(viewName string) string {
	if col, ok := p.columns[viewName]; ok {
		return col
	}
	return viewName
}

func (p *ProjectionMap) Columns() string {
	return strings.Join(p.columnList, ", ")
}

func (p *ProjectionMap) ColumnList() []string {
	return p.columnList
}

func (p *ProjectionMap) Alias() string {
	return p.alias
}
```

### 3.2 Create `pkg/query/builder.go`

```go
package query

import (
	"fmt"
	"strings"
)

type condition struct {
	clause string
	args   []any
}

type Builder struct {
	projection  *ProjectionMap
	conditions  []condition
	orderBy     string
	descending  bool
	defaultSort string
}

func NewBuilder(projection *ProjectionMap, defaultSort string) *Builder {
	return &Builder{
		projection:  projection,
		conditions:  make([]condition, 0),
		defaultSort: defaultSort,
	}
}

func (b *Builder) WhereEquals(field string, value any) *Builder {
	if value == nil {
		return b
	}
	col := b.projection.Column(field)
	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s = $%%d", col),
		args:   []any{value},
	})
	return b
}

func (b *Builder) WhereContains(field string, value *string) *Builder {
	if value == nil || *value == "" {
		return b
	}
	col := b.projection.Column(field)
	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s ILIKE $%%d", col),
		args:   []any{"%" + *value + "%"},
	})
	return b
}

func (b *Builder) WhereSearch(search *string, fields ...string) *Builder {
	if search == nil || *search == "" || len(fields) == 0 {
		return b
	}

	clauses := make([]string, len(fields))
	args := make([]any, len(fields))
	searchPattern := "%" + *search + "%"

	for i, field := range fields {
		col := b.projection.Column(field)
		clauses[i] = fmt.Sprintf("%s ILIKE $%%d", col)
		args[i] = searchPattern
	}

	b.conditions = append(b.conditions, condition{
		clause: "(" + strings.Join(clauses, " OR ") + ")",
		args:   args,
	})
	return b
}

func (b *Builder) WhereIn(field string, values []any) *Builder {
	if len(values) == 0 {
		return b
	}
	col := b.projection.Column(field)
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = "$%d"
	}
	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", ")),
		args:   values,
	})
	return b
}

func (b *Builder) OrderBy(field string, descending bool) *Builder {
	if field != "" {
		b.orderBy = b.projection.Column(field)
	}
	b.descending = descending
	return b
}

func (b *Builder) buildWhere(startParam int) (string, []any, int) {
	if len(b.conditions) == 0 {
		return "", nil, startParam
	}

	clauses := make([]string, 0, len(b.conditions))
	args := make([]any, 0)
	paramIdx := startParam

	for _, cond := range b.conditions {
		clause := cond.clause
		for _, arg := range cond.args {
			clause = strings.Replace(clause, "$%d", fmt.Sprintf("$%d", paramIdx), 1)
			args = append(args, arg)
			paramIdx++
		}
		clauses = append(clauses, clause)
	}

	return " WHERE " + strings.Join(clauses, " AND "), args, paramIdx
}

func (b *Builder) buildOrderBy() string {
	orderCol := b.orderBy
	if orderCol == "" {
		orderCol = b.projection.Column(b.defaultSort)
	}

	dir := "ASC"
	if b.descending {
		dir = "DESC"
	}

	return fmt.Sprintf(" ORDER BY %s %s", orderCol, dir)
}

func (b *Builder) BuildCount() (string, []any) {
	where, args, _ := b.buildWhere(1)
	sql := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", b.projection.Table(), where)
	return sql, args
}

func (b *Builder) BuildPage(page, pageSize int) (string, []any) {
	where, args, _ := b.buildWhere(1)
	orderBy := b.buildOrderBy()
	offset := (page - 1) * pageSize

	sql := fmt.Sprintf(
		"SELECT %s FROM %s%s%s LIMIT %d OFFSET %d",
		b.projection.Columns(),
		b.projection.Table(),
		where,
		orderBy,
		pageSize,
		offset,
	)

	return sql, args
}

func (b *Builder) BuildSingle(idField string, id any) (string, []any) {
	col := b.projection.Column(idField)
	sql := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s = $1",
		b.projection.Columns(),
		b.projection.Table(),
		col,
	)
	return sql, []any{id}
}
```

---

## Phase 4: Lifecycle Package

### 4.1 Create `internal/lifecycle/lifecycle.go`

The lifecycle coordinator encapsulates context management, startup/shutdown WaitGroups, and readiness tracking.

```go
package lifecycle

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ReadinessChecker interface {
	Ready() bool
}

type Coordinator struct {
	ctx        context.Context
	cancel     context.CancelFunc
	startupWg  sync.WaitGroup
	shutdownWg sync.WaitGroup
	ready      bool
	readyMu    sync.RWMutex
}

func New() *Coordinator {
	ctx, cancel := context.WithCancel(context.Background())
	return &Coordinator{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (c *Coordinator) Context() context.Context {
	return c.ctx
}

func (c *Coordinator) OnStartup(fn func()) {
	c.startupWg.Go(fn)
}

func (c *Coordinator) OnShutdown(fn func()) {
	c.shutdownWg.Go(fn)
}

func (c *Coordinator) Ready() bool {
	c.readyMu.RLock()
	defer c.readyMu.RUnlock()
	return c.ready
}

func (c *Coordinator) WaitForStartup() {
	c.startupWg.Wait()
	c.readyMu.Lock()
	c.ready = true
	c.readyMu.Unlock()
}

func (c *Coordinator) Shutdown(timeout time.Duration) error {
	c.cancel()

	done := make(chan struct{})
	go func() {
		c.shutdownWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout after %v", timeout)
	}
}
```

---

## Phase 5: Database System

### 5.1 Create `internal/database/errors.go`

Each package encapsulates its errors in a dedicated file for discoverability.

```go
package database

import "errors"

var ErrNotReady = errors.New("database not ready")
```

### 5.2 Create `internal/database/database.go`

```go
package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/lifecycle"
)

type System interface {
	Connection() *sql.DB
	Start(lc *lifecycle.Coordinator) error
}

type database struct {
	conn        *sql.DB
	logger      *slog.Logger
	connTimeout time.Duration
}

func New(cfg *config.DatabaseConfig, logger *slog.Logger) (System, error) {
	db, err := sql.Open("pgx", cfg.Dsn())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetimeDuration())

	return &database{
		conn:        db,
		logger:      logger.With("system", "database"),
		connTimeout: cfg.ConnTimeoutDuration(),
	}, nil
}

func (d *database) Connection() *sql.DB {
	return d.conn
}

func (d *database) Start(lc *lifecycle.Coordinator) error {
	d.logger.Info("starting database system")

	lc.OnStartup(func() {
		pingCtx, cancel := context.WithTimeout(lc.Context(), d.connTimeout)
		defer cancel()

		if err := d.conn.PingContext(pingCtx); err != nil {
			d.logger.Error("database ping failed", "error", err)
			return
		}

		d.logger.Info("database connection established")
	})

	lc.OnShutdown(func() {
		<-lc.Context().Done()
		d.logger.Info("closing database connection")

		if err := d.conn.Close(); err != nil {
			d.logger.Error("database close failed", "error", err)
			return
		}

		d.logger.Info("database connection closed")
	})

	return nil
}
```

---

## Phase 6: Migration CLI

### 6.1 Create directory structure

```
cmd/migrate/
cmd/migrate/migrations/
```

### 6.2 Create `cmd/migrate/migrations/000001_initial_schema.up.sql`

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
```

### 6.3 Create `cmd/migrate/migrations/000001_initial_schema.down.sql`

```sql
DROP EXTENSION IF EXISTS "uuid-ossp";
```

### 6.4 Create `cmd/migrate/main.go`

```go
package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

const EnvDatabaseDSN = "DATABASE_DSN"

//go:embed migrations/*.sql
var migrations embed.FS

func main() {
	var (
		dsn     = flag.String("dsn", "", "Database connection string")
		up      = flag.Bool("up", false, "Run all up migrations")
		down    = flag.Bool("down", false, "Run all down migrations")
		steps   = flag.Int("steps", 0, "Number of migrations (positive=up, negative=down)")
		version = flag.Bool("version", false, "Print current migration version")
		force   = flag.Int("force", -1, "Force set version (use with caution)")
	)
	flag.Parse()

	if *dsn == "" {
		*dsn = os.Getenv(EnvDatabaseDSN)
	}
	if *dsn == "" {
		log.Fatal("database connection string required: use -dsn flag or DATABASE_DSN env var")
	}

	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		log.Fatalf("failed to create migration source: %v", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, *dsn)
	if err != nil {
		log.Fatalf("failed to create migrator: %v", err)
	}
	defer m.Close()

	switch {
	case *version:
		v, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("failed to get version: %v", err)
		}
		fmt.Printf("version: %d, dirty: %v\n", v, dirty)

	case *force >= 0:
		if err := m.Force(*force); err != nil {
			log.Fatalf("failed to force version: %v", err)
		}
		fmt.Printf("forced to version %d\n", *force)

	case *up:
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("failed to run up migrations: %v", err)
		}
		fmt.Println("migrations applied successfully")

	case *down:
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("failed to run down migrations: %v", err)
		}
		fmt.Println("migrations reverted successfully")

	case *steps != 0:
		if err := m.Steps(*steps); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("failed to run migrations: %v", err)
		}
		fmt.Printf("applied %d migration steps\n", *steps)

	default:
		fmt.Println("usage: migrate -dsn <connection-string> [-up|-down|-steps N|-version|-force N]")
		flag.PrintDefaults()
	}
}
```

---

## Phase 7: Service Integration

### 7.1 Update `cmd/service/service.go`

**Incremental changes** to add database system and use lifecycle coordinator.

**Add imports**:
```go
"time"

"github.com/JaimeStill/agent-lab/internal/database"
"github.com/JaimeStill/agent-lab/internal/lifecycle"
```

**Update Service struct** - delegates lifecycle to coordinator:
```go
type Service struct {
	lifecycle *lifecycle.Coordinator
	logger    logger.System
	database  database.System
	server    server.System
}
```

**Update NewService** - creates lifecycle coordinator first, passes to routes for readiness:
```go
func NewService(cfg *config.Config) (*Service, error) {
	lc := lifecycle.New()
	loggerSys := logger.New(&cfg.Logging)
	routeSys := routes.New(loggerSys.Logger())

	dbSys, err := database.New(&cfg.Database, loggerSys.Logger())
	if err != nil {
		return nil, fmt.Errorf("database init failed: %w", err)
	}

	middlewareSys := buildMiddleware(loggerSys, cfg)
	registerRoutes(routeSys, lc)
	handler := middlewareSys.Apply(routeSys.Build())

	serverSys := server.New(&cfg.Server, handler, loggerSys.Logger())

	return &Service{
		lifecycle: lc,
		logger:    loggerSys,
		database:  dbSys,
		server:    serverSys,
	}, nil
}
```

**Update Start** - subsystems register with lifecycle coordinator:
```go
func (s *Service) Start() error {
	s.logger.Logger().Info("starting service")

	if err := s.database.Start(s.lifecycle); err != nil {
		return fmt.Errorf("database start failed: %w", err)
	}

	if err := s.server.Start(s.lifecycle); err != nil {
		return fmt.Errorf("server start failed: %w", err)
	}

	go func() {
		s.lifecycle.WaitForStartup()
		s.logger.Logger().Info("all subsystems ready")
	}()

	s.logger.Logger().Info("service started")
	return nil
}
```

**Update Shutdown** - delegates to lifecycle coordinator:
```go
func (s *Service) Shutdown(timeout time.Duration) error {
	s.logger.Logger().Info("initiating shutdown")

	if err := s.lifecycle.Shutdown(timeout); err != nil {
		return err
	}

	s.logger.Logger().Info("all subsystems shut down successfully")
	return nil
}
```

### 7.2 Update `cmd/service/routes.go`

**Update registerRoutes** to receive `lifecycle.ReadinessChecker`:

```go
func registerRoutes(r routes.System, ready lifecycle.ReadinessChecker) {
	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: "/healthz",
		Handler: handleHealthCheck,
	})

	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: "/readyz",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			handleReadinessCheck(w, ready)
		},
	})
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleReadinessCheck(w http.ResponseWriter, ready lifecycle.ReadinessChecker) {
	if !ready.Ready() {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("NOT READY"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("READY"))
}
```

### 7.3 Update `internal/server/server.go`

**Update System interface** to use lifecycle coordinator:
```go
type System interface {
	Start(lc *lifecycle.Coordinator) error
}
```

**Update Start method** - server starts in goroutine, registers shutdown handler:
```go
func (s *server) Start(lc *lifecycle.Coordinator) error {
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

**Remove Stop method** - no longer needed, shutdown handled via lifecycle.

**Add import**:
```go
"github.com/JaimeStill/agent-lab/internal/lifecycle"
```

### 7.4 Update `cmd/service/main.go`

The `Shutdown()` method now takes `time.Duration` directly instead of `context.Context`.

**Remove import** (no longer needed):
```go
"context"
```

**Update shutdown call** - pass duration instead of context:
```go
<-sigChan

if err := svc.Shutdown(cfg.ShutdownTimeoutDuration()); err != nil {
	log.Fatal("shutdown failed:", err)
}

log.Println("service stopped gracefully")
```

**Complete main.go after changes**:
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

	if err := svc.Shutdown(cfg.ShutdownTimeoutDuration()); err != nil {
		log.Fatal("shutdown failed:", err)
	}

	log.Println("service stopped gracefully")
}
```

---

## Phase 8: Dependencies

### 8.1 Add Go module dependencies

Run these commands:

```bash
go get github.com/jackc/pgx/v5
go get github.com/golang-migrate/migrate/v4
go mod tidy
```

---

## Validation Checklist

After implementation, verify:

- [ ] `go build ./...` succeeds
- [ ] Config loads with new pagination fields
- [ ] Lifecycle coordinator manages startup/shutdown correctly
- [ ] Service starts and connects to database
- [ ] `/healthz` returns 200
- [ ] `/readyz` returns 503 before subsystems ready, 200 after
- [ ] Migration CLI runs: `go run ./cmd/migrate -dsn "postgres://..." -up`
- [ ] Graceful shutdown closes all subsystems
- [ ] All existing tests still pass

---

## Notes

- Code blocks intentionally have no comments (per CLAUDE.md conventions)
- Testing will be added by AI after implementation validation
- Godoc comments will be added by AI after testing
- The `toml` import in config.go references the existing import from Session 1a
- The `lifecycle.Coordinator` encapsulates context, WaitGroups, and readiness - Service no longer manages these directly
- Server uses `OnShutdown` only (not `OnStartup`) since `ListenAndServe` is a long-running process
- Database uses both `OnStartup` (ping) and `OnShutdown` (close)
