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
- **Hot Start** (`Start`): Pings database to verify connection, signals readiness via WaitGroup
- **Shutdown** (`Stop`): Closes connection pool gracefully

### Startup WaitGroup (Symmetric to Shutdown)

Just as shutdown uses a WaitGroup to coordinate subsystem completion, startup tracks subsystem readiness:

```
Service.Start()
    ├── database.Start(ctx, &startupWg)  // Adds to WaitGroup, signals when ready
    ├── server.Start(ctx, &shutdownWg)   // Existing pattern
    └── go startupWg.Wait() → ready=true // Background readiness tracking
```

### Health Check Caching

Configurable via `SERVICE_PING_INTERVAL`:
- `0s` = ping database on every `/readyz` request
- `>0s` = cache health result for specified duration

### Query Builder Layers

1. **ProjectionMap**: Static structure defining table/column mappings
2. **QueryBuilder**: Fluent builder for filters, sorting, pagination
3. **Execution**: Use generated SQL with `database/sql`

---

## Phase 1: Configuration Updates

### 1.1 Update Root Config (`internal/config/config.go`)

Add `PingInterval` field and `Pagination` section:

```go
package config

import (
	"fmt"
	"os"
	"time"
)

const (
	EnvServiceEnv             = "SERVICE_ENV"
	EnvServiceShutdownTimeout = "SERVICE_SHUTDOWN_TIMEOUT"
	EnvServicePingInterval    = "SERVICE_PING_INTERVAL"

	BaseConfigFile       = "config.toml"
	OverlayConfigPattern = "config.%s.toml"
)

type Config struct {
	Server          ServerConfig     `toml:"server"`
	Database        DatabaseConfig   `toml:"database"`
	Logging         LoggingConfig    `toml:"logging"`
	CORS            CORSConfig       `toml:"cors"`
	Pagination      PaginationConfig `toml:"pagination"`
	ShutdownTimeout string           `toml:"shutdown_timeout"`
	PingInterval    string           `toml:"ping_interval"`
}

func (c *Config) ShutdownTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.ShutdownTimeout)
	return d
}

func (c *Config) PingIntervalDuration() time.Duration {
	d, _ := time.ParseDuration(c.PingInterval)
	return d
}

func (c *Config) Finalize() error {
	c.loadDefaults()
	c.loadEnv()
	return c.validate()
}

func (c *Config) Merge(overlay *Config) {
	if overlay.ShutdownTimeout != "" {
		c.ShutdownTimeout = overlay.ShutdownTimeout
	}
	if overlay.PingInterval != "" {
		c.PingInterval = overlay.PingInterval
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
	if c.PingInterval == "" {
		c.PingInterval = "0s"
	}
	c.Server.loadDefaults()
	c.Database.loadDefaults()
	c.Logging.loadDefaults()
	c.CORS.loadDefaults()
	c.Pagination.loadDefaults()
}

func (c *Config) loadEnv() {
	if v := os.Getenv(EnvServiceShutdownTimeout); v != "" {
		c.ShutdownTimeout = v
	}
	if v := os.Getenv(EnvServicePingInterval); v != "" {
		c.PingInterval = v
	}
	c.Server.loadEnv()
	c.Database.loadEnv()
	c.Logging.loadEnv()
	c.CORS.loadEnv()
	c.Pagination.loadEnv()
}

func (c *Config) validate() error {
	if _, err := time.ParseDuration(c.ShutdownTimeout); err != nil {
		return fmt.Errorf("invalid shutdown_timeout: %w", err)
	}
	if _, err := time.ParseDuration(c.PingInterval); err != nil {
		return fmt.Errorf("invalid ping_interval: %w", err)
	}
	if err := c.Server.validate(); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	if err := c.Database.validate(); err != nil {
		return fmt.Errorf("database: %w", err)
	}
	if err := c.Logging.validate(); err != nil {
		return fmt.Errorf("logging: %w", err)
	}
	if err := c.Pagination.validate(); err != nil {
		return fmt.Errorf("pagination: %w", err)
	}
	return nil
}

func Load() (*Config, error) {
	basePath := BaseConfigFile
	overlayPath := overlayPath()

	data, err := os.ReadFile(basePath)
	if err != nil {
		return nil, fmt.Errorf("read base config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse base config: %w", err)
	}

	if overlayPath != "" {
		overlayData, err := os.ReadFile(overlayPath)
		if err != nil {
			return nil, fmt.Errorf("read overlay config: %w", err)
		}

		var overlay Config
		if err := toml.Unmarshal(overlayData, &overlay); err != nil {
			return nil, fmt.Errorf("parse overlay config: %w", err)
		}

		cfg.Merge(&overlay)
	}

	return &cfg, nil
}

func overlayPath() string {
	if env := os.Getenv(EnvServiceEnv); env != "" {
		path := fmt.Sprintf(OverlayConfigPattern, env)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
```

### 1.2 Create Pagination Config (`internal/config/pagination.go`)

```go
package config

import (
	"fmt"
	"os"
	"strconv"
)

const (
	EnvPaginationDefaultPageSize = "PAGINATION_DEFAULT_PAGE_SIZE"
	EnvPaginationMaxPageSize     = "PAGINATION_MAX_PAGE_SIZE"
)

type PaginationConfig struct {
	DefaultPageSize int `toml:"default_page_size"`
	MaxPageSize     int `toml:"max_page_size"`
}

func (c *PaginationConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()
	return c.validate()
}

func (c *PaginationConfig) Merge(overlay *PaginationConfig) {
	if overlay.DefaultPageSize != 0 {
		c.DefaultPageSize = overlay.DefaultPageSize
	}
	if overlay.MaxPageSize != 0 {
		c.MaxPageSize = overlay.MaxPageSize
	}
}

func (c *PaginationConfig) loadDefaults() {
	if c.DefaultPageSize == 0 {
		c.DefaultPageSize = 20
	}
	if c.MaxPageSize == 0 {
		c.MaxPageSize = 100
	}
}

func (c *PaginationConfig) loadEnv() {
	if v := os.Getenv(EnvPaginationDefaultPageSize); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.DefaultPageSize = n
		}
	}
	if v := os.Getenv(EnvPaginationMaxPageSize); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxPageSize = n
		}
	}
}

func (c *PaginationConfig) validate() error {
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

### 1.3 Update `config.toml`

Add pagination section and ping_interval:

```toml
shutdown_timeout = "30s"
ping_interval = "0s"

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

## Phase 2: Pagination Package

### 2.1 Create `pkg/pagination/pagination.go`

```go
package pagination

type Config struct {
	DefaultPageSize int
	MaxPageSize     int
}

func (c *Config) Finalize() {
	if c.DefaultPageSize == 0 {
		c.DefaultPageSize = 20
	}
	if c.MaxPageSize == 0 {
		c.MaxPageSize = 100
	}
}

type PageRequest struct {
	Page       int     `json:"page"`
	PageSize   int     `json:"pageSize"`
	Search     *string `json:"search,omitempty"`
	SortBy     string  `json:"sortBy,omitempty"`
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
	PageSize   int `json:"pageSize"`
	TotalPages int `json:"totalPages"`
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
	for i, field := range fields {
		col := b.projection.Column(field)
		clauses[i] = fmt.Sprintf("%s ILIKE $%%d", col)
	}

	b.conditions = append(b.conditions, condition{
		clause: "(" + strings.Join(clauses, " OR ") + ")",
		args:   []any{"%" + *search + "%"},
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
		placeholders[i] = fmt.Sprintf("$%%d")
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
	where, args, nextParam := b.buildWhere(1)
	orderBy := b.buildOrderBy()

	offset := (page - 1) * pageSize

	sql := fmt.Sprintf(
		"SELECT %s FROM %s%s%s OFFSET %d ROWS FETCH NEXT %d ROWS ONLY",
		b.projection.Columns(),
		b.projection.Table(),
		where,
		orderBy,
		offset,
		pageSize,
	)

	_ = nextParam
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

## Phase 4: Database System

### 4.1 Create `internal/database/database.go`

```go
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/JaimeStill/agent-lab/internal/config"
)

var ErrNotReady = errors.New("database not ready")

type System interface {
	Connection() *sql.DB
	Start(ctx context.Context, wg *sync.WaitGroup) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) error
	Ready() bool
}

type database struct {
	conn        *sql.DB
	logger      *slog.Logger
	connTimeout time.Duration
	ready       bool
	mu          sync.RWMutex
}

func New(cfg *config.DatabaseConfig, logger *slog.Logger) (System, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.Name, cfg.User, cfg.Password,
	)

	db, err := sql.Open("pgx", dsn)
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

func (d *database) Start(ctx context.Context, wg *sync.WaitGroup) error {
	d.logger.Info("starting database connection")

	wg.Add(1)
	go func() {
		defer wg.Done()

		pingCtx, cancel := context.WithTimeout(ctx, d.connTimeout)
		defer cancel()

		if err := d.conn.PingContext(pingCtx); err != nil {
			d.logger.Error("database ping failed", "error", err)
			return
		}

		d.mu.Lock()
		d.ready = true
		d.mu.Unlock()

		d.logger.Info("database connection established")
	}()

	return nil
}

func (d *database) Stop(ctx context.Context) error {
	d.logger.Info("closing database connection")

	d.mu.Lock()
	d.ready = false
	d.mu.Unlock()

	if err := d.conn.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}

	d.logger.Info("database connection closed")
	return nil
}

func (d *database) Health(ctx context.Context) error {
	d.mu.RLock()
	ready := d.ready
	d.mu.RUnlock()

	if !ready {
		return ErrNotReady
	}

	return d.conn.PingContext(ctx)
}

func (d *database) Ready() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.ready
}
```

---

## Phase 5: Migration CLI

### 5.1 Create directory structure

```
migrations/
cmd/migrate/
```

### 5.2 Create `migrations/000001_initial_schema.up.sql`

```sql
-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Schema version tracking handled by golang-migrate
```

### 5.3 Create `migrations/000001_initial_schema.down.sql`

```sql
-- Drop UUID extension
DROP EXTENSION IF EXISTS "uuid-ossp";
```

### 5.4 Create `cmd/migrate/main.go`

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
		*dsn = os.Getenv("DATABASE_DSN")
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

### 5.5 Move migrations to cmd/migrate

The migrations should be embedded from within the cmd/migrate package:

```
cmd/migrate/
├── main.go
└── migrations/
    ├── 000001_initial_schema.up.sql
    └── 000001_initial_schema.down.sql
```

Update the embed directive in main.go accordingly (it already points to `migrations/*.sql`).

---

## Phase 6: Service Integration

### 6.1 Update `cmd/service/service.go`

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/database"
	"github.com/JaimeStill/agent-lab/internal/logger"
	"github.com/JaimeStill/agent-lab/internal/routes"
	"github.com/JaimeStill/agent-lab/internal/server"
)

type Service struct {
	ctx        context.Context
	cancel     context.CancelFunc
	shutdownWg sync.WaitGroup
	startupWg  sync.WaitGroup

	logger   logger.System
	database database.System
	server   server.System

	pingInterval time.Duration
	lastPing     time.Time
	lastHealth   error
	healthMu     sync.RWMutex
	ready        bool
	readyMu      sync.RWMutex
}

func NewService(cfg *config.Config) (*Service, error) {
	ctx, cancel := context.WithCancel(context.Background())

	loggerSys := logger.New(&cfg.Logging)

	dbSys, err := database.New(&cfg.Database, loggerSys.Logger())
	if err != nil {
		cancel()
		return nil, fmt.Errorf("database init failed: %w", err)
	}

	svc := &Service{
		ctx:          ctx,
		cancel:       cancel,
		logger:       loggerSys,
		database:     dbSys,
		pingInterval: cfg.PingIntervalDuration(),
	}

	routeSys := routes.New(loggerSys.Logger())
	middlewareSys := buildMiddleware(loggerSys, cfg)
	registerRoutes(routeSys, svc)
	handler := middlewareSys.Apply(routeSys.Build())
	serverSys := server.New(&cfg.Server, handler, loggerSys.Logger())

	svc.server = serverSys

	return svc, nil
}

func (s *Service) Start() error {
	s.logger.Logger().Info("starting service")

	if err := s.database.Start(s.ctx, &s.startupWg); err != nil {
		return fmt.Errorf("database start failed: %w", err)
	}

	if err := s.server.Start(s.ctx, &s.shutdownWg); err != nil {
		return fmt.Errorf("server start failed: %w", err)
	}

	go func() {
		s.startupWg.Wait()
		s.readyMu.Lock()
		s.ready = true
		s.readyMu.Unlock()
		s.logger.Logger().Info("all subsystems ready")
	}()

	s.logger.Logger().Info("service started")
	return nil
}

func (s *Service) Shutdown(ctx context.Context) error {
	s.logger.Logger().Info("initiating shutdown")

	s.cancel()

	if err := s.database.Stop(ctx); err != nil {
		s.logger.Logger().Error("database stop failed", "error", err)
	}

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

func (s *Service) Ready() bool {
	s.readyMu.RLock()
	defer s.readyMu.RUnlock()
	return s.ready
}

func (s *Service) CheckHealth(ctx context.Context) error {
	if !s.Ready() {
		return fmt.Errorf("service not ready")
	}

	if s.pingInterval > 0 {
		s.healthMu.RLock()
		lastPing := s.lastPing
		lastHealth := s.lastHealth
		s.healthMu.RUnlock()

		if time.Since(lastPing) < s.pingInterval {
			return lastHealth
		}
	}

	err := s.database.Health(ctx)

	if s.pingInterval > 0 {
		s.healthMu.Lock()
		s.lastPing = time.Now()
		s.lastHealth = err
		s.healthMu.Unlock()
	}

	return err
}

func (s *Service) Logger() *slog.Logger {
	return s.logger.Logger()
}
```

### 6.2 Update `cmd/service/routes.go`

```go
package main

import (
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/routes"
)

func registerRoutes(r routes.System, svc *Service) {
	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: "/healthz",
		Handler: handleHealthCheck,
	})

	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: "/readyz",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			handleReadinessCheck(w, r, svc)
		},
	})
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleReadinessCheck(w http.ResponseWriter, r *http.Request, svc *Service) {
	if err := svc.CheckHealth(r.Context()); err != nil {
		svc.Logger().Warn("readiness check failed", "error", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("NOT READY"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("READY"))
}
```

---

## Phase 7: Dependencies

### 7.1 Add Go module dependencies

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
- [ ] Config loads with new pagination and ping_interval fields
- [ ] Service starts and connects to database
- [ ] `/healthz` returns 200
- [ ] `/readyz` returns 503 before database ready, 200 after
- [ ] Migration CLI runs: `go run ./cmd/migrate -dsn "postgres://..." -up`
- [ ] Graceful shutdown closes database connection
- [ ] All existing tests still pass

---

## Notes

- Code blocks intentionally have no comments (per CLAUDE.md conventions)
- Testing will be added by AI after implementation validation
- Godoc comments will be added by AI after testing
- The `toml` import in config.go references the existing import from Session 1a
