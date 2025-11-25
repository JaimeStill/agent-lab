# Web Service Architecture

**Established**: 2025-11-22
**Purpose**: Define architectural principles, layer structure, and design patterns for web service development

This document contains both validated patterns (proven in agent-lab codebase) and conceptual architecture (not yet implemented but planned).

---

# Section 1: Validated Patterns

**Status**: These patterns have been implemented and validated in the agent-lab codebase (Session 01a).

## Core Architectural Principles

### 1. State Flows Down, Never Up

**Rule**: State should flow through method calls (parameters), not through object initialization, unless the state is owned by that object or process (function or method).

**Anti-Pattern** (Reaching Up):
```go
type Handler struct {
    app *Application  // ❌ Storing reference to parent
}

func (h *Handler) Process() {
    svc := h.app.GetService()  // ❌ Reaching up to parent state
}
```

**Correct Pattern** (State Flows Down):
```go
func HandleRequest(w, r, system System, logger Logger) {
    // ✓ State injected at call site, flows DOWN
}

func routes(app *Server) http.Handler {
    sys := app.Providers()  // State from owner
    mux.HandleFunc("POST /api/providers", func(w, r) {
        HandleCreate(w, r, sys, logger)  // ✓ State flows DOWN
    })
}
```

**Exception** (State Owned by Object):
```go
type Server struct {
    config *Config  // ✓ Server OWNS this config
    logger *Logger  // ✓ Server OWNS this logger
}

func (s *Server) Start() {
    s.logger.Info("starting")  // ✓ Accessing owned state
}
```

### 2. Systems, Not Services/Models

**Terminology**:
- **System**: A cohesive unit that owns both state and processes
- **State**: Structures that define data (what was "models")
- **Process**: Methods that operate on state (what was "service methods")
- **Interface**: Contract between systems for portability

**Stop using**:
- "Service" - too vague, doesn't describe responsibility
- "Model" - too generic, doesn't describe purpose

**Start using**:
- Specific domain names: `ProviderConfigurationSystem`, `DatabaseSystem`, `HTTPSystem`
- Clear state structures: `Provider`, `Agent`, `SearchRequest`
- Clear processes: `Create()`, `Search()`, `HandleRequest()`

### 3. Dependency Layers

Systems are organized in clear dependency layers:

```
Layer 1: Service (Composition Root)
  cmd/service/
  ↓ initializes with config

Layer 2: Core Systems (Domain)
  internal/[config|logger|routes|middleware|server]
  ↓ uses

Layer 3: Public Infrastructure (Shared Toolkit)
  pkg/[query|pagination] (future)
  ↓ uses

Layer 4: External Services
  PostgreSQL, Future: Blob Storage, Cache, etc.
```

**Rules**:
- Each layer only knows about its direct dependencies
- Dependencies only ever provided as interfaces by owners
- State flows DOWN through parameters
- Events flow UP through return values or channels

### 4. Cold Start vs Hot Start

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

Example:
```go
// Cold Start - Build state graph
svc, err := NewService(cfg)  // All systems initialized but dormant

// Hot Start - Activate processes
if err := svc.Start(); err != nil {
    log.Fatal(err)
}
```

### 5. System Interface Contract

Every system provides:

1. **Internal State** (private) - Only accessible within the system
2. **Internal Processes** (private) - Implementation details
3. **Getter Methods** (public) - Immutable access to state
4. **Commands** (public) - Write operations from owner
5. **Events** (public, optional) - Notifications to owner

### 6. Interface Naming Convention

**Getter Methods** (Nouns - Access State):
```go
// ❌ Wrong - Too verbose
GetId() uuid.UUID
GetName() string
GetConfig() Config

// ✅ Correct - Pure nouns, descriptive of state
Id() uuid.UUID
Name() string
Config() Config
```

**Command Methods** (Verbs - Perform Actions):
```go
// ✅ Action verbs
Start() error
Stop() error
Create(ctx, cmd CreateCommand) (*Provider, error)
Update(ctx, id, cmd UpdateCommand) (*Provider, error)
Delete(ctx, id) error
Search(ctx, req SearchRequest) (*SearchResult, error)
```

**Event Methods** (On* - Notifications):
```go
// ✅ Events with On* prefix
OnProgress() <-chan ProgressEvent
OnShutdown() <-chan struct{}
OnComplete() <-chan CompleteEvent
OnError() <-chan error
```

**Complete Example**:
```go
type ProviderSystem interface {
    // Getters - Nouns (state access)
    Id() uuid.UUID
    Name() string

    // Commands - Verbs (actions)
    Create(ctx context.Context, cmd CreateCommand) (*Provider, error)
    Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Provider, error)
    Delete(ctx context.Context, id uuid.UUID) error
    Search(ctx context.Context, req SearchRequest) (*SearchResult, error)

    // Events - On* prefix (notifications)
    OnProviderCreated() <-chan ProviderEvent
    OnProviderDeleted() <-chan ProviderEvent
}
```

## Package Structure

### **Layer 1: Service (Composition Root)**

```
cmd/service/
├── main.go          # Entry point: Load config → Cold start → Hot start
├── service.go       # Service (composition root, owns all subsystems)
├── middleware.go    # Middleware configuration
└── routes.go        # Route registration
```

**Responsibilities**:
- Bootstrap application
- Compose all Layer 2 systems
- Manage application lifecycle
- Handle graceful shutdown

### **Layer 2: Core Systems (Domain)**

**Currently Implemented**:
```
internal/
├── config/          # Configuration System
│   ├── config.go        # Root configuration
│   ├── server.go        # Server configuration
│   ├── database.go      # Database configuration
│   ├── logging.go       # Logging configuration
│   ├── cors.go          # CORS configuration
│   └── types.go         # Shared types
│
├── logger/          # Logger System
│   └── logger.go        # Structured logging
│
├── routes/          # HTTP Route Registration System
│   ├── routes.go        # Route system
│   └── group.go         # Route group definition
│
├── middleware/      # HTTP Middleware System
│   ├── middleware.go    # Middleware stack
│   ├── logger.go        # Logger middleware
│   └── cors.go          # CORS middleware
│
└── server/          # HTTP Server System
    └── server.go        # Server lifecycle management
```

**Future** (not yet implemented, see Section 2):
```
internal/
├── providers/       # Provider Configuration System
├── agents/          # Agent Configuration System
└── database/        # Database Connection System
```

**Responsibilities**:
- Domain-specific business logic
- Database operations
- HTTP request handling
- Route registration

### **Layer 3: Public Infrastructure (Shared Toolkit)**

**Future** (not yet implemented, see Section 2):
```
pkg/
├── query/           # SQL Query Building Toolkit
│   ├── builder.go       # Query builder
│   └── projection.go    # Column projection mapping
│
└── pagination/      # Pagination Utilities
    └── pagination.go    # Page request/result structures
```

**Responsibilities**:
- Reusable utilities
- No domain-specific logic
- Can be imported by external services
- Like a "mod API" for extending the server

**Why `pkg/`**:
- Makes infrastructure public and reusable
- Other services/tools can import these utilities
- Clear separation: `pkg/` = public API, `internal/` = private implementation

## Configuration Management

### Configuration Precedence

**Principle**: All configuration values (scalar or array) are atomic units that replace at each precedence level.

```
Environment Variables (highest precedence)
    ↓ replaces (not merges)
config.{env}.toml (overlay)
    ↓ replaces (not merges)
config.toml (base configuration)
```

**Key Principles:**
1. **Atomic Replacement**: Configuration values are never merged - presence indicates complete replacement
2. **Array Format**: Array values use comma-separated strings in environment variables
3. **Consistent Behavior**: Scalar and array configs follow the same precedence rules
4. **Predictable**: What you see at each level is exactly what you get

**Examples:**

Scalar value replacement:
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

Array value replacement:
```toml
# config.toml
[cors]
origins = ["http://localhost:3000", "http://localhost:8080"]
```
```bash
# Environment override (replaces entire array, does not merge)
CORS_ORIGINS="http://example.com,http://other.com"
# Result: origins = ["http://example.com", "http://other.com"]
```

### Encapsulated Configuration Sections

**Principle**: Each configuration section owns its environment mapping, finalization, and validation logic through internal methods.

**Section Pattern**:
```go
type SectionConfig struct {
    Field1 string `toml:"field1"`
    Field2 int    `toml:"field2"`
}

func (c *SectionConfig) loadDefaults() {
    // Apply defaults for any zero-value fields
    if c.Field1 == "" {
        c.Field1 = "default"
    }
    if c.Field2 == 0 {
        c.Field2 = 42
    }
}

func (c *SectionConfig) loadEnv() {
    // Map SECTION_FIELD environment variables to struct fields
    if v := os.Getenv("SECTION_FIELD1"); v != "" {
        c.Field1 = v
    }
    if v := os.Getenv("SECTION_FIELD2"); v != "" {
        if n, err := strconv.Atoi(v); err == nil {
            c.Field2 = n
        }
    }
}

func (c *SectionConfig) validate() error {
    // Validate field constraints
    if c.Field1 == "" {
        return fmt.Errorf("field1 required")
    }
    return nil
}
```

**Root Config Pattern**:
```go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Logging  LoggingConfig
    CORS     CORSConfig
}

func (c *Config) Finalize() error {
    c.loadDefaults()
    c.loadEnv()
    return c.validate()
}

func (c *Config) loadDefaults() {
    c.Server.loadDefaults()
    c.Database.loadDefaults()
    c.Logging.loadDefaults()
    c.CORS.loadDefaults()
}

func (c *Config) loadEnv() {
    c.Server.loadEnv()
    c.Database.loadEnv()
    c.Logging.loadEnv()
    c.CORS.loadEnv()
}

func (c *Config) validate() error {
    if err := c.Server.validate(); err != nil {
        return fmt.Errorf("server: %w", err)
    }
    if err := c.Database.validate(); err != nil {
        return fmt.Errorf("database: %w", err)
    }
    if err := c.Logging.validate(); err != nil {
        return fmt.Errorf("logging: %w", err)
    }
    return nil
}
```

### Environment Variable Convention

**Pattern**: `SECTION_FIELD` (all uppercase, underscores for nesting)

**Docker vs Application**:
- **Docker Compose variables** (e.g., `POSTGRES_*`, `OLLAMA_*`) → Configure containers
- **Application variables** (e.g., `DATABASE_*`, `SERVER_*`) → Override TOML config

**Complete Mapping**:
```
TOML Section → Environment Variable

[server]
  host              → SERVER_HOST
  port              → SERVER_PORT
  read_timeout      → SERVER_READ_TIMEOUT
  write_timeout     → SERVER_WRITE_TIMEOUT

[database]
  host              → DATABASE_HOST
  port              → DATABASE_PORT
  name              → DATABASE_NAME
  user              → DATABASE_USER
  password          → DATABASE_PASSWORD

[logging]
  level             → LOGGING_LEVEL
  format            → LOGGING_FORMAT

[cors]
  origins           → CORS_ORIGINS (comma-separated)
  credentials       → CORS_CREDENTIALS (true/false)
  headers           → CORS_HEADERS (comma-separated)
  methods           → CORS_METHODS (comma-separated)
  max_age           → CORS_MAX_AGE
```

### Configuration Loading Flow

```go
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

// Usage in main()
func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }

    // Finalize: defaults → env → validate
    if err := cfg.Finalize(); err != nil {
        log.Fatal("config invalid:", err)
    }

    // Config is now validated and ready for use
    svc, err := NewService(cfg)
    // ...
}
```

**Simplified Pattern**:
1. **Load()**: Loads base TOML + optional overlay
2. **Finalize()**: Single public method orchestrates defaults → env → validate
3. **Internal methods**: loadDefaults(), loadEnv(), validate() are implementation details
4. **Caller controls**: main() decides when to finalize

**Benefits**:
1. **Implicit guarantee**: Every TOML field has a corresponding env override
2. **Co-located logic**: Environment mapping lives with the struct it modifies
3. **Self-contained**: Each section is independently testable
4. **Clear mapping**: Easy to see what env vars exist per section
5. **Single method**: One call does everything (Finalize())

## Configuration is Ephemeral

**Principle**: Configuration exists only to initialize systems, then it's discarded.

### Stateful Systems vs Functional Infrastructure

**Stateful Systems** (use concrete config structs):
- Systems that own state and other systems
- Complex initialization requiring defaults → env → validate
- Examples: Service, Database, Providers, Agents

**Functional Infrastructure** (use simple parameters):
- Stateless or minimal-state utilities
- Functional infrastructure with no owned subsystems
- Examples: Handlers, middleware, routing, query builders

### Simplified Pattern Benefits

1. **No interfaces needed** - Concrete structs are simpler and sufficient
2. **Single Finalize() method** - One call does defaults → env → validate
3. **Internal methods** - loadDefaults(), loadEnv(), validate() are implementation details
4. **Configuration graph** - Config struct contains child config structs
5. **Ephemeral** - Config is discarded after initialization

### Configuration Lifecycle

```
Load TOML → Config struct → Finalize() → NewService(cfg) → [Config discarded]
                              ↓
                    defaults → env → validate
```

## Validated System Patterns

### Service System (Composition Root)

**Service is the composition root** - it owns all systems but doesn't store config (config is ephemeral).

```go
// Service owns all subsystems
type Service struct {
    ctx        context.Context
    cancel     context.CancelFunc
    shutdownWg sync.WaitGroup

    logger logger.System
    server server.System
}

// Cold Start - Initialize state dependency graph
func NewService(cfg *Config) (*Service, error) {
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

// Hot Start - Activate processes
func (s *Service) Start() error {
    s.logger.Logger().Info("starting service")

    if err := s.server.Start(s.ctx, &s.shutdownWg); err != nil {
        return fmt.Errorf("server start failed: %w", err)
    }

    s.logger.Logger().Info("service started")
    return nil
}

// Graceful Shutdown
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

### Server System

```go
type System interface {
    Start(ctx context.Context, wg *sync.WaitGroup) error
    Stop(ctx context.Context) error
}

type server struct {
    http            *http.Server
    logger          *slog.Logger
    shutdownTimeout time.Duration
}

func New(cfg *ServerConfig, handler http.Handler, logger *slog.Logger) System {
    return &server{
        http: &http.Server{
            Addr:         cfg.Addr(),
            Handler:      handler,
            ReadTimeout:  cfg.ReadTimeoutDuration(),
            WriteTimeout: cfg.WriteTimeoutDuration(),
        },
        logger:          logger,
        shutdownTimeout: cfg.ShutdownTimeoutDuration(),
    }
}
```

### Logger System

```go
type System interface {
    Logger() *slog.Logger
}

type logger struct {
    logger *slog.Logger
}

func New(cfg *LoggingConfig) System {
    var handler slog.Handler

    opts := &slog.HandlerOptions{
        Level: parseLevel(cfg.Level),
    }

    switch cfg.Format {
    case "json":
        handler = slog.NewJSONHandler(os.Stdout, opts)
    default:
        handler = slog.NewTextHandler(os.Stdout, opts)
    }

    return &logger{
        logger: slog.New(handler),
    }
}
```

### Route System

**Functional Infrastructure** - Uses simple constructor, not config interface. Stateless utility for organizing routes.

```go
type System interface {
    RegisterGroup(group Group)
    RegisterRoute(route Route)
    Build() http.Handler
}

type system struct {
    groups []Group
    routes []Route
    logger *slog.Logger
}

func New(logger *slog.Logger) System {
    return &system{
        groups: make([]Group, 0),
        routes: make([]Route, 0),
        logger: logger,
    }
}

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

### Middleware System

**Functional Infrastructure** - Stateless utility for composing middleware stack.

```go
type System interface {
    Use(mw func(http.Handler) http.Handler)
    Apply(handler http.Handler) http.Handler
}

type middleware struct {
    stack []func(http.Handler) http.Handler
}

func New() System {
    return &middleware{
        stack: make([]func(http.Handler) http.Handler, 0),
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

## Graceful Shutdown

Service implements graceful shutdown using context cancellation and coordinated timeout handling:

```go
// main.go
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

    ctx, stop := signal.NotifyContext(
        context.Background(),
        os.Interrupt,
        syscall.SIGTERM,
    )
    defer stop()

    if err := svc.Start(); err != nil {
        log.Fatal("service failed:", err)
    }

    <-ctx.Done()

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := svc.Shutdown(shutdownCtx); err != nil {
        log.Fatal("shutdown failed:", err)
    }

    log.Println("service stopped gracefully")
}
```

**Shutdown Flow**:
1. **Signal received** (SIGINT/SIGTERM) → context cancelled
2. **Service.Shutdown()** called → cancels service context
3. **HTTP server** gracefully closes connections (configurable timeout)
4. **WaitGroup** waits for all subsystems to complete
5. **Service.Shutdown()** returns → main() exits

## Testing Strategy

### Black-Box Testing

All tests use `package <name>_test` and import the package being tested:

```go
package config_test

import (
    "testing"

    "github.com/JaimeStill/agent-lab/internal/config"
)

func TestLoad(t *testing.T) {
    cfg, err := config.Load()
    if err != nil {
        t.Fatalf("Load() failed: %v", err)
    }

    if cfg == nil {
        t.Fatal("Load() returned nil config")
    }
}
```

### Table-Driven Tests

```go
func TestServerConfig_Validate(t *testing.T) {
    tests := []struct {
        name      string
        cfg       config.ServerConfig
        expectErr bool
    }{
        {
            name: "valid config",
            cfg: config.ServerConfig{
                Host:         "localhost",
                Port:         8080,
                ReadTimeout:  "30s",
                WriteTimeout: "30s",
            },
            expectErr: false,
        },
        {
            name: "invalid port",
            cfg: config.ServerConfig{
                Host:         "localhost",
                Port:         99999,
                ReadTimeout:  "30s",
                WriteTimeout: "30s",
            },
            expectErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tt.cfg.loadDefaults()
            err := tt.cfg.validate()
            if tt.expectErr && err == nil {
                t.Error("expected error but got nil")
            }
            if !tt.expectErr && err != nil {
                t.Errorf("unexpected error: %v", err)
            }
        })
    }
}
```

### Test Organization

Tests mirror package structure:
```
tests/
├── internal_config/      # Mirrors internal/config
├── internal_logger/      # Mirrors internal/logger
├── internal_routes/      # Mirrors internal/routes
├── internal_middleware/  # Mirrors internal/middleware
├── internal_server/      # Mirrors internal/server
└── cmd_service/          # Mirrors cmd/service
```

---

# Section 2: Conceptual Architecture

**Status**: These patterns are planned but not yet implemented in agent-lab.

## Domain System Pattern

**Domain systems receive concrete dependencies** - no config interfaces needed.

```go
// internal/providers/system.go
type System interface {
    Create(ctx context.Context, cmd CreateCommand) (*Provider, error)
    Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Provider, error)
    Delete(ctx context.Context, id uuid.UUID) error
    Search(ctx context.Context, req SearchRequest) (*SearchResult, error)
    FindByID(ctx context.Context, id uuid.UUID) (*Provider, error)
}

// internal/providers/repository.go
type repository struct {
    db              *sql.DB
    logger          *slog.Logger
    defaultPageSize int
}

func New(db *sql.DB, logger *slog.Logger, defaultPageSize int) (System, error) {
    if db == nil {
        return nil, fmt.Errorf("db required")
    }
    if logger == nil {
        return nil, fmt.Errorf("logger required")
    }
    if defaultPageSize <= 0 {
        defaultPageSize = 20
    }

    return &repository{
        db:              db,
        logger:          logger.With("system", "providers"),
        defaultPageSize: defaultPageSize,
    }, nil
}
```

## Handler Pattern

Handlers are **pure functions**, not structs with stored state:

```go
// internal/providers/handlers.go - Functional infrastructure (simple parameters)
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

## Route Group Pattern

Domain packages define route groups:

```go
// internal/providers/routes.go - Route group
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
        },
    }
}
```

## Database System Pattern

**Database system receives concrete config struct** - applies its own defaults and validation.

```go
// internal/database/database.go
type System interface {
    Connection() *sql.DB
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health(ctx context.Context) error
}

type database struct {
    conn   *sql.DB
    logger *slog.Logger
}

func New(cfg *DatabaseConfig, logger *slog.Logger) (System, error) {
    if cfg == nil {
        return nil, fmt.Errorf("config required")
    }
    if logger == nil {
        return nil, fmt.Errorf("logger required")
    }

    dsn := fmt.Sprintf(
        "host=%s port=%d dbname=%s user=%s password=%s sslmode=disable connect_timeout=%d",
        cfg.Host, cfg.Port, cfg.Name, cfg.User, cfg.Password, int(cfg.ConnTimeoutDuration().Seconds()))

    db, err := sql.Open("pgx", dsn)
    if err != nil {
        return nil, fmt.Errorf("open failed: %w", err)
    }

    db.SetMaxOpenConns(cfg.MaxOpenConns)
    db.SetMaxIdleConns(cfg.MaxIdleConns)
    db.SetConnMaxLifetime(cfg.ConnMaxLifetimeDuration())

    return &database{
        conn:   db,
        logger: logger.With("system", "database"),
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

func (d *database) Stop(ctx context.Context) error {
    d.logger.Info("stopping database connection")
    return d.conn.Close()
}

func (d *database) Health(ctx context.Context) error {
    return d.conn.PingContext(ctx)
}
```

## Query Infrastructure Pattern

Three-layer architecture for building parameterized SQL queries:

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

## Pagination Pattern

Reusable pagination structures for all search operations:

```go
// pkg/pagination/pagination.go
type Config struct {
    DefaultPageSize int
    MaxPageSize     int
}

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

## Key Design Decisions

### 1. Long-Running vs Ephemeral Systems

**Long-Running** (Application-Scoped):
- Provider Configuration System
- Agent Configuration System
- Database System
- Created once during cold start
- Live for application lifetime

**Ephemeral** (Request-Scoped):
- None currently
- Future: When systems need request context (userID, auth)
- Created per-request in handlers

### 2. Handler Pattern

- Handlers are **pure functions**, not structs with stored state
- State flows DOWN through parameters
- Each handler receives: `(w, r, system, logger)`
- Handlers live in domain packages (e.g., `providers/handlers.go`)

### 3. Interface Boundaries

- Systems expose interfaces, not concrete types
- Enables testing with mocks
- Clear contract between layers
- Owner defines interface, implementation is private
