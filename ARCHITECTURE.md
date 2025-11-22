# agent-lab Architecture

**Status**: Source of truth for building the system
**Scope**: Milestone 1 foundation - Provider and Agent configuration management

## Overview

agent-lab is a containerized Go web service for building and orchestrating agentic workflows. This document defines the architectural patterns and implementation details for the system's foundation.

**Complete architectural philosophy**: See [_context/web-service-architecture.md](./_context/web-service-architecture.md)

## Core Architectural Principles

### 1. State Flows Down, Never Up

State flows through method calls (parameters), not through object initialization, unless the state is owned by that object or process.

**Anti-Pattern** (Reaching Up):
```go
type Handler struct {
    server *Server  // ❌ Storing reference to parent
}

func (h *Handler) Process() {
    sys := h.server.Providers()  // ❌ Reaching up to parent state
}
```

**Correct Pattern** (State Flows Down):
```go
func HandleCreate(w http.ResponseWriter, r *http.Request, system providers.System, logger *slog.Logger) {
    // ✓ State injected at call site, flows DOWN
}
```

### 2. Systems, Not Services/Models

**Terminology**:
- **System**: A cohesive unit that owns both state and processes
- **State**: Structures that define data
- **Process**: Methods that operate on state
- **Interface**: Contract between systems

**Package Organization**:
- **cmd/server**: The process (composition root, entry point)
- **pkg/**: Public API (shared infrastructure, reusable toolkit)
- **internal/**: Private API (domain systems, business logic)

### 3. Cold Start vs Hot Start

**Cold Start** (State Initialization):
- `New*()` constructor functions
- Builds entire dependency graph
- All configurations → State objects
- All systems created but dormant
- No processes running
- Returns ready-to-start system

**Hot Start** (Process Activation):
- `Start()` methods
- State objects → Running processes
- Cascade start through dependency graph
- Context boundaries for lifecycle management
- System becomes interactable
- Blocks until shutdown

Example:
```go
srv, err := NewServer(cfg)  // Cold Start - Build state graph
ctx := context.Background()
err := srv.Start(ctx)  // Hot Start - Activate processes
```

### 4. System Interface Contract

Every system provides:

1. **Internal State** (private) - Only accessible within the system
2. **Internal Processes** (private) - Implementation details
3. **Getter Methods** (public) - Immutable access to state
4. **Commands** (public) - Write operations from owner
5. **Events** (public, optional) - Notifications to owner

### 5. Interface Naming Convention

**Getters** (Nouns - Access State):
```go
Id() uuid.UUID
Name() string
Connection() *sql.DB
```

**Commands** (Verbs - Perform Actions):
```go
Start(ctx context.Context) error
Create(ctx context.Context, cmd CreateCommand) (*Provider, error)
Search(ctx context.Context, req SearchRequest) (*SearchResult, error)
```

**Events** (On* - Notifications):
```go
OnShutdown() <-chan struct{}
OnError() <-chan error
```

### 6. Configuration-Driven Initialization

**Stateful Systems vs Functional Infrastructure**:

- **Stateful Systems**: Use encapsulated config interface pattern (Server, Database, Providers, Agents)
- **Functional Infrastructure**: Use simple function signatures (handlers, middleware, routing)

**Stateful System Pattern**:

All stateful systems use `New*` constructor functions with encapsulated config interfaces following: **Finalize → Validate → Transform**

```go
type ProvidersConfig interface {
    DB() *sql.DB
    Logger() *slog.Logger
    Pagination() pagination.Config

    Finalize()
    Validate() error
}

func New(cfg ProvidersConfig) (System, error) {
    // 1. Finalize: Apply defaults
    cfg.Finalize()

    // 2. Validate: Check required dependencies
    if err := cfg.Validate(); err != nil {
        return nil, err
    }

    // 3. Transform: Create validated instance
    return &repository{
        db:         cfg.DB(),
        logger:     cfg.Logger().With("system", "providers"),
        pagination: cfg.Pagination(),
    }, nil
}
```

**Why Config Interfaces**:
- Makes required state immediately apparent
- Config can define its own finalize/validate/transform behaviors
- Configuration graph for owned objects lives in parent config
- Clear ownership boundaries
- Easier testing (mock the interface)

## System Architecture

### Directory Structure

```
cmd/server/           # Process: Composition root
├── main.go               # Entry point
├── server.go             # Server system
└── config.go             # Configuration loading

internal/             # Private API: Domain systems
├── providers/
│   ├── provider.go       # State structures
│   ├── system.go         # Interface definition
│   ├── repository.go     # System implementation
│   ├── handlers.go       # HTTP handlers
│   └── routes.go         # Route group definition
│
├── agents/
│   └── (same structure as providers)
│
├── database/
│   └── database.go       # Database system
│
├── routes/
│   ├── routes.go         # Route system
│   └── group.go          # Route group definition
│
└── middleware/
    └── middleware.go     # Middleware system

pkg/                  # Public API: Shared infrastructure
├── query/
│   ├── builder.go        # Query builder
│   └── projection.go     # Column projection mapping
│
└── pagination/
    └── pagination.go     # Page request/result structures
```

### Component Flow

```
HTTP Request
    ↓
Middleware System (CORS, logging, recovery)
    ↓
Routes System (smart grouping, domain boundaries)
    ↓
Handler Functions (pure functions, state injected)
    ↓
Domain Systems (providers, agents)
    ↓
Database System (connection pool)
    ↓
PostgreSQL
```

## Server System (cmd/server)

The Server system is the composition root that owns all subsystems and manages the application lifecycle.

### Server Structure

```go
type Server struct {
    config     *Config
    db         database.System
    logger     *slog.Logger
    providers  providers.System
    agents     agents.System
    middleware middleware.System
    routes     routes.System
}
```

### Cold Start Pattern

```go
type ServerConfig interface {
    Database() DatabaseConfig
    Providers() ProvidersConfig
    Agents() AgentsConfig
    Pagination() pagination.Config
    Logging() LoggingConfig
    HTTP() HTTPServerConfig
    CORS() CORSConfig

    Finalize()
    Validate() error
}

func NewServer(cfg ServerConfig) (*Server, error) {
    // 1. Finalize: Apply defaults across config graph
    cfg.Finalize()

    // 2. Validate: Check requirements across config graph
    if err := cfg.Validate(); err != nil {
        return nil, err
    }

    // 3. Transform: Build dependency graph
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: parseLevel(cfg.Logging().Level()),
    }))

    db, err := database.New(cfg.Database())
    if err != nil {
        return nil, fmt.Errorf("database: %w", err)
    }

    providers, err := providers.New(cfg.Providers())
    if err != nil {
        return nil, fmt.Errorf("providers: %w", err)
    }

    agents, err := agents.New(cfg.Agents())
    if err != nil {
        return nil, fmt.Errorf("agents: %w", err)
    }

    middleware := middleware.New(cfg.CORS(), logger)

    return &Server{
        config:     cfg,
        db:         db,
        logger:     logger,
        providers:  providers,
        agents:     agents,
        middleware: middleware,
    }, nil
}
```

**Configuration Graph**:
- Parent config (`ServerConfig`) contains child configs for owned systems
- Each child config (`DatabaseConfig`, `ProvidersConfig`) can have its own finalize/validate
- Clear ownership: Server owns Database, Providers, Agents - configs reflect this hierarchy

### Hot Start Pattern

```go
func (s *Server) Start(ctx context.Context) error {
    // Cascade start to subsystems
    if err := s.db.Start(ctx); err != nil {
        return fmt.Errorf("database start: %w", err)
    }

    // Build HTTP handler
    handler := s.buildHandler()

    // Start HTTP server (blocking)
    return s.startHTTP(ctx, handler)
}

func (s *Server) buildHandler() http.Handler {
    routeSys := routes.New(s.logger)

    // Register route groups (state flows DOWN)
    routeSys.RegisterGroup(providers.Routes(s.providers, s.logger))
    routeSys.RegisterGroup(agents.Routes(s.agents, s.logger))

    handler := routeSys.Build()
    return s.middleware.Apply(handler)
}
```

### Getter Methods

```go
func (s *Server) Config() *Config              { return s.config }
func (s *Server) Logger() *slog.Logger         { return s.logger }
func (s *Server) Providers() providers.System  { return s.providers }
func (s *Server) Agents() agents.System        { return s.agents }
```

### Main Entry Point

```go
func main() {
    // 1. Load configuration
    cfg, err := LoadConfig("config.toml")
    if err != nil {
        log.Fatal("config load failed:", err)
    }

    // 2. Cold Start - Initialize state
    srv, err := NewServer(cfg)
    if err != nil {
        log.Fatal("server init failed:", err)
    }

    // 3. Context with signal handling
    ctx, stop := signal.NotifyContext(
        context.Background(),
        os.Interrupt,
        syscall.SIGTERM,
    )
    defer stop()

    // 4. Hot Start - Start processes (blocking)
    if err := srv.Start(ctx); err != nil {
        log.Fatal("server failed:", err)
    }

    log.Println("server stopped gracefully")
}
```

## Domain Systems (internal/)

### System Interface Pattern

```go
type System interface {
    // Commands - Verbs (actions)
    Create(ctx context.Context, cmd CreateCommand) (*Provider, error)
    Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Provider, error)
    Delete(ctx context.Context, id uuid.UUID) error
    Search(ctx context.Context, req SearchRequest) (*SearchResult, error)

    // Getters - Nouns (queries)
    FindByID(ctx context.Context, id uuid.UUID) (*Provider, error)
}
```

### Configuration Interface

```go
type ProvidersConfig interface {
    DB() *sql.DB
    Logger() *slog.Logger
    Pagination() pagination.Config

    Finalize()
    Validate() error
}
```

### Repository Implementation

```go
type repository struct {
    db         *sql.DB
    logger     *slog.Logger
    pagination pagination.Config
}

func New(cfg ProvidersConfig) (System, error) {
    cfg.Finalize()

    if err := cfg.Validate(); err != nil {
        return nil, err
    }

    return &repository{
        db:         cfg.DB(),
        logger:     cfg.Logger().With("system", "providers"),
        pagination: cfg.Pagination(),
    }, nil
}

func (r *repository) Create(ctx context.Context, cmd CreateCommand) (*Provider, error) {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback()

    query := `
        INSERT INTO providers (id, name, config, created_at, updated_at)
        VALUES (gen_random_uuid(), $1, $2, NOW(), NOW())
        RETURNING id, name, config, created_at, updated_at`

    var p Provider
    err = tx.QueryRowContext(ctx, query, cmd.Name, cmd.Config).Scan(
        &p.ID, &p.Name, &p.Config, &p.CreatedAt, &p.UpdatedAt)
    if err != nil {
        return nil, fmt.Errorf("insert provider: %w", err)
    }

    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("commit transaction: %w", err)
    }

    r.logger.Info("provider created", "id", p.ID, "name", p.Name)
    return &p, nil
}
```

### State Structures

State structures are pure data with no methods:

```go
type Provider struct {
    ID        uuid.UUID
    Name      string
    Config    json.RawMessage
    CreatedAt time.Time
    UpdatedAt time.Time
}

type CreateCommand struct {
    Name   string
    Config json.RawMessage
}

type SearchRequest struct {
    Name     *string
    Page     int
    PageSize int
    SortBy   string
    Desc     bool
}
```

## Handler Pattern (internal/)

**Functional Infrastructure**: Handlers are pure functions that receive state as parameters. They don't use config interfaces because they're stateless and already well-encapsulated.

```go
func HandleCreate(
    w http.ResponseWriter,
    r *http.Request,
    system System,
    logger *slog.Logger,
) {
    var cmd CreateCommand
    if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
        respondError(w, logger, http.StatusBadRequest, err)
        return
    }

    result, err := system.Create(r.Context(), cmd)
    if err != nil {
        respondError(w, logger, http.StatusInternalServerError, err)
        return
    }

    respondJSON(w, logger, http.StatusCreated, result)
}
```

**Why Simple Parameters**: Handlers are functional infrastructure - they route requests to systems. The `(w, r)` pair is the action context, `system` and `logger` are the minimal dependencies. No need for config interfaces here.

## Route System (internal/routes)

**Functional Infrastructure**: Routes system is stateless infrastructure for organizing HTTP routing. Uses simple constructor, not config interface.

### Smart Route Grouping

```go
type System struct {
    groups []Group
    logger *slog.Logger
}

func New(logger *slog.Logger) System {
    return System{logger: logger}
}

func (s *System) RegisterGroup(group Group) {
    s.groups = append(s.groups, group)
}

func (s *System) Build() http.Handler {
    mux := http.NewServeMux()

    mux.HandleFunc("GET /healthz", func(w, r) {
        w.WriteHeader(http.StatusOK)
    })

    for _, group := range s.groups {
        s.registerGroup(mux, group)
    }

    return mux
}
```

### Route Group Definition

```go
type Group struct {
    Prefix      string
    Tags        []string
    Description string
    Middleware  []func(http.Handler) http.Handler
    Routes      []Route
}

type Route struct {
    Method  string
    Pattern string
    Handler http.HandlerFunc
}
```

### Domain Route Groups

```go
func Routes(system System, logger *slog.Logger) routes.Group {
    return routes.Group{
        Prefix:      "/api/providers",
        Tags:        []string{"Providers"},
        Description: "Provider configuration management",

        Routes: []routes.Route{
            {
                Method:  "POST",
                Pattern: "",
                Handler: func(w, r) {
                    HandleCreate(w, r, system, logger)
                },
            },
            {
                Method:  "GET",
                Pattern: "/{id}",
                Handler: func(w, r) {
                    HandleGetByID(w, r, system, logger)
                },
            },
            {
                Method:  "PUT",
                Pattern: "/{id}",
                Handler: func(w, r) {
                    HandleUpdate(w, r, system, logger)
                },
            },
            {
                Method:  "DELETE",
                Pattern: "/{id}",
                Handler: func(w, r) {
                    HandleDelete(w, r, system, logger)
                },
            },
            {
                Method:  "POST",
                Pattern: "/search",
                Handler: func(w, r) {
                    HandleSearch(w, r, system, logger)
                },
            },
        },
    }
}
```

## Database System (internal/database)

### Configuration Interface

```go
type DatabaseConfig interface {
    Host() string
    Port() int
    Name() string
    User() string
    Password() string
    MaxOpenConns() int
    MaxIdleConns() int
    ConnMaxLifetime() time.Duration
    ConnTimeout() time.Duration

    Logger() *slog.Logger

    Finalize()
    Validate() error
}
```

### System Interface

```go
type System interface {
    // Getters - Nouns (state access)
    Connection() *sql.DB

    // Commands - Verbs (actions)
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}
```

### Implementation

```go
type database struct {
    conn   *sql.DB
    logger *slog.Logger
}

func New(cfg DatabaseConfig) (System, error) {
    cfg.Finalize()

    if err := cfg.Validate(); err != nil {
        return nil, err
    }

    dsn := fmt.Sprintf(
        "host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
        cfg.Host(), cfg.Port(), cfg.Name(), cfg.User(), cfg.Password())

    db, err := sql.Open("pgx", dsn)
    if err != nil {
        return nil, fmt.Errorf("open database: %w", err)
    }

    db.SetMaxOpenConns(cfg.MaxOpenConns())
    db.SetMaxIdleConns(cfg.MaxIdleConns())
    db.SetConnMaxLifetime(cfg.ConnMaxLifetime())

    return &database{
        conn:   db,
        logger: cfg.Logger().With("system", "database"),
    }, nil
}

func (d *database) Connection() *sql.DB {
    return d.conn
}

func (d *database) Start(ctx context.Context) error {
    d.logger.Info("starting database connection")

    pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    if err := d.conn.PingContext(pingCtx); err != nil {
        return fmt.Errorf("ping failed: %w", err)
    }

    d.logger.Info("database connection established")
    return nil
}
```

## Query Infrastructure (pkg/query)

### Three-Layer Architecture

**Layer 1: ProjectionMap** (Structure Definition):
- Static, reusable query structure per domain entity
- Defines tables, joins, column mappings
- Resolves view property names to `table.column` references

**Layer 2: QueryBuilder** (Operations):
- Fluent builder for filters, sorting, pagination
- Methods: `WhereEquals`, `WhereContains`, `WhereSearch`, `OrderBy`
- Automatic null-checking: only applies filters when values are non-null
- Generates: `BuildCount()`, `BuildPage()`, `BuildSingle()`

**Layer 3: Execution** (database/sql):
- Execute generated SQL + args with `QueryContext`/`ExecContext`
- Two-query pattern: COUNT for total, SELECT with OFFSET/FETCH

### Example Usage

```go
var providerProjection = query.NewProjectionMap("public", "providers", "p").
    Project("id", "Id").
    Project("name", "Name").
    Project("config", "Config").
    Project("created_at", "CreatedAt").
    Project("updated_at", "UpdatedAt")

func (r *repository) Search(ctx context.Context, req SearchRequest) (*SearchResult, error) {
    qb := query.NewBuilder(providerProjection, "Name").
        WhereContains("Name", req.Name).
        OrderBy(req.SortBy, req.Desc)

    countSQL, countArgs := qb.BuildCount()
    var total int
    err := r.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total)
    if err != nil {
        return nil, fmt.Errorf("count: %w", err)
    }

    pageSQL, pageArgs := qb.BuildPage(req.Page, req.PageSize)
    rows, err := r.db.QueryContext(ctx, pageSQL, pageArgs...)
    if err != nil {
        return nil, fmt.Errorf("query: %w", err)
    }
    defer rows.Close()

    var providers []Provider
    for rows.Next() {
        var p Provider
        err := rows.Scan(&p.ID, &p.Name, &p.Config, &p.CreatedAt, &p.UpdatedAt)
        if err != nil {
            return nil, fmt.Errorf("scan: %w", err)
        }
        providers = append(providers, p)
    }

    return &SearchResult{
        Data:  providers,
        Total: total,
        Page:  req.Page,
    }, nil
}
```

## Pagination Infrastructure (pkg/pagination)

### Configuration

```go
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
```

### Request/Response Structures

```go
type PageRequest struct {
    Page     int
    PageSize int
}

type PageResult[T any] struct {
    Data  []T `json:"data"`
    Total int `json:"total"`
    Page  int `json:"page"`
}
```

## Configuration Management

### TOML-Based Configuration

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

[pagination]
default_page_size = 20
max_page_size = 100

[logging]
level = "info"
format = "json"

[cors]
origins = ["http://localhost:3000"]
```

### Environment Variable Override

Environment variables mirror the TOML structure using underscores:

```bash
server_port=8080
database_host=localhost
database_password=secret
cors_origins_0=http://localhost:3000
cors_origins_1=http://localhost:4000
```

## Middleware System (internal/middleware)

**Functional Infrastructure**: Middleware is stateless infrastructure that wraps HTTP handlers. Uses simple constructor with minimal parameters.

### Middleware Stack

```go
type System struct {
    cors   CORSConfig
    logger *slog.Logger
}

func New(cors CORSConfig, logger *slog.Logger) System {
    return System{
        cors:   cors,
        logger: logger,
    }
}

func (s System) Apply(handler http.Handler) http.Handler {
    handler = s.recoverPanic(handler)
    handler = s.logRequest(handler)
    handler = s.enableCORS(handler)
    return handler
}

func (s System) logRequest(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        s.logger.Info("request",
            "method", r.Method,
            "uri", r.URL.RequestURI(),
            "addr", r.RemoteAddr)
        next.ServeHTTP(w, r)
    })
}

func (s System) recoverPanic(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                w.Header().Set("Connection", "close")
                s.logger.Error("panic recovered", "error", err)
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

**Why Simple Constructor**: Middleware has minimal state (CORS config, logger) and is functional infrastructure. No complex initialization or owned subsystems, so config interface would be overkill.

## Database Patterns

### Transaction Pattern

Commands always use transactions:

```go
func (r *repository) Create(ctx context.Context, cmd CreateCommand) (*Provider, error) {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Execute mutations within transaction
    var p Provider
    err = tx.QueryRowContext(ctx, query, args...).Scan(&p.ID, &p.Name, ...)
    if err != nil {
        return nil, fmt.Errorf("insert: %w", err)
    }

    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("commit: %w", err)
    }

    return &p, nil
}
```

### Query Pattern

Queries don't use transactions:

```go
func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*Provider, error) {
    query := `SELECT id, name, config, created_at, updated_at FROM providers WHERE id = $1`

    var p Provider
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &p.ID, &p.Name, &p.Config, &p.CreatedAt, &p.UpdatedAt)
    if err == sql.ErrNoRows {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("query: %w", err)
    }

    return &p, nil
}
```

## Error Handling

### Service-Level Errors

```go
var (
    ErrNotFound     = errors.New("resource not found")
    ErrInvalidInput = errors.New("invalid input")
)
```

### HTTP Response Helpers

```go
func respondJSON(w http.ResponseWriter, logger *slog.Logger, status int, data any) {
    js, err := json.MarshalIndent(data, "", "  ")
    if err != nil {
        logger.Error("marshal error", "error", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    w.Write(js)
    w.Write([]byte("\n"))
}

func respondError(w http.ResponseWriter, logger *slog.Logger, status int, err error) {
    logger.Error("handler error", "error", err, "status", status)
    respondJSON(w, logger, status, map[string]string{"error": err.Error()})
}
```

## Testing Strategy

### Unit Tests

Test systems with mocked dependencies:

```go
func TestProviderRepository_Create(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    repo := providers.New(db, slog.Default(), pagination.Config{})

    provider, err := repo.Create(context.Background(), providers.CreateCommand{
        Name:   "test-provider",
        Config: json.RawMessage(`{"type":"openai"}`),
    })

    require.NoError(t, err)
    assert.Equal(t, "test-provider", provider.Name)
}
```

### Integration Tests

Test with real database:

```go
func TestProviderAPI_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    srv := setupTestServer(t)
    defer srv.Close()

    body := `{"name":"test-provider","config":{"type":"openai"}}`
    req := httptest.NewRequest("POST", "/api/providers", strings.NewReader(body))
    w := httptest.NewRecorder()

    srv.Handler.ServeHTTP(w, req)

    assert.Equal(t, http.StatusCreated, w.Code)
}
```

## Deployment

### Docker Compose

```yaml
services:
  postgres:
    image: postgres:17-alpine
    container_name: agent-lab-postgres
    ports:
      - "5432:5432"
    volumes:
      - ./.data/postgres:/var/lib/postgresql/data
    environment:
      POSTGRES_PASSWORD: postgres

  agent-lab:
    build: .
    container_name: agent-lab-server
    ports:
      - "8080:8080"
    environment:
      database_host: postgres
      database_port: 5432
      database_name: agent_lab
      database_user: agent_lab
      database_password: secure_password
    depends_on:
      - postgres
```

### Graceful Shutdown

```go
func (s *Server) startHTTP(ctx context.Context, handler http.Handler) error {
    srv := &http.Server{
        Addr:         fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port),
        Handler:      handler,
        ReadTimeout:  s.config.Server.ReadTimeout,
        WriteTimeout: s.config.Server.WriteTimeout,
    }

    shutdown := make(chan error, 1)
    go func() {
        <-ctx.Done()
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        shutdown <- srv.Shutdown(shutdownCtx)
    }()

    s.logger.Info("starting HTTP server", "addr", srv.Addr)

    if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
        return err
    }

    return <-shutdown
}
```

## Integration with go-agents Ecosystem

### Provider Validation

Provider configurations are validated by creating go-agents provider instances:

```go
func (r *repository) Create(ctx context.Context, cmd CreateCommand) (*Provider, error) {
    var cfg config.ProviderConfig
    if err := json.Unmarshal(cmd.Config, &cfg); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    _, err := providers.New(&cfg)
    if err != nil {
        return nil, fmt.Errorf("invalid provider: %w", err)
    }

    // Proceed with database insertion
}
```

### Agent Validation

Agent configurations are validated by creating go-agents agent instances:

```go
func (r *repository) Create(ctx context.Context, cmd CreateCommand) (*Agent, error) {
    var cfg config.AgentConfig
    if err := json.Unmarshal(cmd.Config, &cfg); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    _, err := agent.New(&cfg)
    if err != nil {
        return nil, fmt.Errorf("invalid agent: %w", err)
    }

    // Proceed with database insertion
}
```

## Pattern Decision Guide

### When to Use Config Interfaces

**Use for Stateful Systems**:
- System owns internal state
- System owns other systems (configuration graph)
- Complex initialization with finalize/validate/transform
- Examples: Server, Database, Providers, Agents

```go
type SystemConfig interface {
    // Required dependencies
    DependencyA() DependencyA
    DependencyB() DependencyB

    // Owned subsystem configs
    SubsystemA() SubsystemAConfig
    SubsystemB() SubsystemBConfig

    Finalize()
    Validate() error
}

func New(cfg SystemConfig) (System, error)
```

### When to Use Simple Parameters

**Use for Functional Infrastructure**:
- Stateless or minimal state
- No owned subsystems
- Clear, limited dependencies
- Examples: Handlers, middleware, routing, query builders

```go
func HandleCreate(w http.ResponseWriter, r *http.Request, system System, logger *slog.Logger)
func New(logger *slog.Logger) System
func NewBuilder(projection ProjectionMap, defaultSort string) *Builder
```

**Rule of Thumb**: If it owns state and other systems, use config interface. If it's functional infrastructure, use simple parameters.

## References

- **web-service-architecture.md**: Complete architectural philosophy and design decisions
- **go-agents**: Configuration patterns, interface design, LCA principles
- **go-agents-orchestration**: Workflow patterns (future milestones)
- **document-context**: LCA architecture (future milestones)
