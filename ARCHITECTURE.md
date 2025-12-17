# agent-lab Architecture

## Overview

agent-lab is a containerized Go web service for building and orchestrating agentic workflows. This document defines the architectural patterns currently implemented in the system.

**Complete architectural philosophy**: See [_context/web-service-architecture.md](./_context/web-service-architecture.md)
**Future designs**: See [_context/service-design.md](./_context/service-design.md)

## Core Architectural Principles

### 1. State Flows Down, Never Up

State flows through method calls (parameters), not through object initialization, unless the state is owned by that object or process.

**Anti-Pattern** (Reaching Up):
```go
type Handler struct {
    service *Service  // ❌ Storing reference to parent
}

func (h *Handler) Process() {
    sys := h.service.Providers()  // ❌ Reaching up to parent state
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

Example:
```go
svc, err := NewService(cfg)  // Cold Start - Build state graph
if err := svc.Start(); err != nil {  // Hot Start - Activate processes
    log.Fatal(err)
}
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

**Repository Query Methods** (Return Type Determines Verb):

| Verb | Returns | Use Case |
|------|---------|----------|
| `List` | `*PageResult[T]` | Browsing/searching collections (paginated) |
| `Find` | `*T` | Locate single item by ID |
| `Get` | `[]T` | Retrieve all related items (bounded, full slice) |

```go
ListRuns(ctx, page, filters) (*PageResult[Run], error)  // Paginated browsing
FindRun(ctx, id) (*Run, error)                          // Single by ID
GetStages(ctx, runID) ([]Stage, error)                  // All stages for a run
GetDecisions(ctx, runID) ([]Decision, error)            // All decisions for a run
```

**Rationale**: `Get` signals "retrieve this specific bounded set" (e.g., all stages for one run), while `List` signals "browse a potentially large collection with pagination."

### 6. Configuration-Driven Initialization

**Stateful Systems vs Functional Infrastructure**:

- **Stateful Systems**: Use concrete config structs with simplified finalize pattern (Service, Database)
- **Functional Infrastructure**: Use simple function signatures (handlers, middleware, routing, logging)

**Stateful System Pattern**:

All stateful systems use `New*` constructor functions that receive validated config structs.

**Configuration uses simplified pattern**:

```go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Logging  LoggingConfig
    CORS     CORSConfig
}

func (c *Config) Finalize() error {
    c.loadDefaults()   // Apply defaults
    c.loadEnv()        // Apply environment overrides
    return c.validate() // Validate constraints
}
```

**System constructors receive validated config**:

```go
func NewService(cfg *Config) (*Service, error) {
    // Config is already finalized (validated)
    // Extract values and build system

    logger := logger.New(&cfg.Logging)
    routeSys := routes.New(logger.Logger())
    middlewareSys := buildMiddleware(logger, cfg)
    registerRoutes(routeSys)
    handler := middlewareSys.Apply(routeSys.Build())
    serverSys := server.New(&cfg.Server, handler, logger.Logger())

    return &Service{
        logger: logger,
        server: serverSys,
    }, nil
}
```

**Why Concrete Structs**:
- Simpler - no interfaces needed
- Single `Finalize() error` method handles all initialization
- Each config section encapsulates its own defaults/env/validation
- Configuration is ephemeral - discarded after initialization
- Clear, straightforward pattern

## System Architecture

### Directory Structure

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
│   ├── middleware.go         # Middleware composition
│   └── openapi.go            # OpenAPI spec generation
│
└── migrate/              # Migration CLI
    ├── main.go               # Migration entry point
    └── migrations/           # Embedded SQL migrations

internal/                 # Private API: Domain systems
├── config/
│   ├── config.go             # Root configuration
│   ├── server.go             # Server configuration
│   ├── database.go           # Database configuration
│   ├── logging.go            # Logging configuration
│   ├── cors.go               # CORS configuration
│   └── types.go              # Shared types
│
├── lifecycle/
│   └── lifecycle.go          # Lifecycle coordinator
│
├── database/
│   ├── database.go           # Database system
│   └── errors.go             # Package errors
│
├── storage/
│   ├── storage.go            # Storage system interface
│   ├── filesystem.go         # Filesystem implementation
│   └── errors.go             # Package errors
│
├── routes/
│   ├── routes.go             # Route system
│   └── group.go              # Route group definition
│
├── middleware/
│   ├── middleware.go         # Middleware system
│   ├── logger.go             # Logger middleware
│   ├── cors.go               # CORS middleware
│   └── slash.go              # Trailing slash redirect middleware
│
├── providers/            # Providers domain system
│   ├── provider.go           # State structures
│   ├── errors.go             # Domain errors + HTTP status mapping
│   ├── mapping.go            # Projection, scanner, and filters
│   ├── system.go             # System interface
│   ├── repository.go         # Repository implementation
│   ├── handler.go            # Handler struct with route methods
│   └── openapi.go            # OpenAPI schemas and operations
│
├── agents/               # Agents domain system
│   ├── agent.go              # State structures
│   ├── errors.go             # Domain errors + HTTP status mapping
│   ├── mapping.go            # Projection, scanner, and filters
│   ├── system.go             # System interface
│   ├── repository.go         # Repository implementation
│   ├── handler.go            # Handler struct with CRUD + execution methods
│   ├── requests.go           # Execution request types + VisionForm
│   └── openapi.go            # OpenAPI schemas and operations
│
├── documents/            # Documents domain system
│   ├── document.go           # State structures
│   ├── errors.go             # Domain errors + HTTP status mapping
│   ├── mapping.go            # Projection, scanner, and filters
│   ├── system.go             # System interface
│   ├── repository.go         # Repository implementation
│   ├── handler.go            # Handler struct with route methods
│   └── openapi.go            # OpenAPI schemas and operations
│
└── images/               # Images domain system
    ├── image.go              # State structures + RenderOptions
    ├── document.go           # Document interface for rendering
    ├── errors.go             # Domain errors + HTTP status mapping
    ├── mapping.go            # Projection, scanner, filters, and page range parsing
    ├── system.go             # System interface
    ├── repository.go         # Repository implementation + rendering logic
    ├── handler.go            # Handler struct with route methods
    └── openapi.go            # OpenAPI schemas and operations

pkg/                      # Public API: Shared infrastructure
├── handlers/
│   └── handlers.go           # HTTP response utilities
│
├── openapi/              # OpenAPI specification utilities
│   ├── types.go              # OpenAPI 3.1 type definitions
│   ├── components.go         # Shared schemas and responses
│   └── json.go               # JSON serialization
│
├── pagination/
│   ├── config.go             # Pagination configuration
│   └── pagination.go         # PageRequest/PageResult types
│
├── query/
│   ├── projection.go         # ProjectionMap for column mapping
│   └── builder.go            # Fluent query builder
│
└── repository/
    ├── repository.go         # Transaction and query helpers
    └── errors.go             # Domain-agnostic error mapping

web/                      # Web assets and handlers
└── docs/                 # API documentation (Scalar UI)
    ├── docs.go               # Handler serving documentation
    ├── index.html            # HTML template
    ├── scalar.js             # Scalar JavaScript bundle (embedded)
    └── scalar.css            # Scalar stylesheet (embedded)

tests/                    # Black-box tests
├── cmd_server/
├── internal_agents/
├── internal_config/
├── internal_documents/
├── internal_images/
├── internal_lifecycle/
├── internal_middleware/
├── internal_providers/
├── internal_routes/
├── internal_storage/
├── pkg_handlers/
├── pkg_openapi/
├── pkg_pagination/
├── pkg_query/
├── pkg_repository/
└── web_docs/
```

### Component Flow

```
HTTP Request
    ↓
Middleware System (Logger, CORS)
    ↓
Routes System (route registration and grouping)
    ↓
Handler Functions (healthz endpoint)
    ↓
HTTP Response
```

## Lifecycle System (internal/lifecycle)

The Lifecycle system coordinates application startup and shutdown, providing a centralized place for subsystems to register their lifecycle hooks.

### ReadinessChecker Interface

```go
type ReadinessChecker interface {
    Ready() bool
}
```

### Coordinator

```go
type Coordinator struct {
    ctx        context.Context
    cancel     context.CancelFunc
    startupWg  sync.WaitGroup
    shutdownWg sync.WaitGroup
    ready      bool
    readyMu    sync.RWMutex
}

func New() *Coordinator
func (c *Coordinator) Context() context.Context
func (c *Coordinator) OnStartup(fn func())
func (c *Coordinator) OnShutdown(fn func())
func (c *Coordinator) Ready() bool
func (c *Coordinator) WaitForStartup()
func (c *Coordinator) Shutdown(timeout time.Duration) error
```

**Usage Pattern**:
- **OnStartup**: Register tasks that must complete for service readiness (e.g., database ping)
- **OnShutdown**: Register cleanup tasks triggered on context cancellation (e.g., close connections)
- **WaitForStartup**: Called after Start() to block until all startup tasks complete
- **Ready**: Returns true after WaitForStartup completes (one-time gate)

**Subsystem Integration**:
- Database: Uses OnStartup (ping) and OnShutdown (close)
- Server: Uses OnShutdown only (ListenAndServe is long-running)

## Runtime/Domain System Separation (cmd/server)

The server uses a two-tier system separation pattern that clearly distinguishes between infrastructure (Runtime) and business logic (Domain).

### System Categories

| Category | Characteristics | Examples |
|----------|----------------|----------|
| **Runtime Systems** | Long-running, lifecycle-managed, application-scoped | Database |
| **Domain Systems** | Stateless, request-scoped behavior, no lifecycle | Providers, Agents |

### Runtime Structure

Runtime holds infrastructure systems that have lifecycle management:

```go
type Runtime struct {
    Lifecycle  *lifecycle.Coordinator
    Logger     *slog.Logger
    Database   database.System
    Storage    storage.System
    Pagination pagination.Config
}

func NewRuntime(cfg *config.Config) (*Runtime, error) {
    lc := lifecycle.New()
    logger := newLogger(&cfg.Logging)

    dbSys, err := database.New(&cfg.Database, logger)
    if err != nil {
        return nil, fmt.Errorf("database init failed: %w", err)
    }

    storageSys, err := storage.New(&cfg.Storage, logger)
    if err != nil {
        return nil, fmt.Errorf("storage init failed: %w", err)
    }

    return &Runtime{
        Lifecycle:  lc,
        Logger:     logger,
        Database:   dbSys,
        Storage:    storageSys,
        Pagination: cfg.Pagination,
    }, nil
}

func (r *Runtime) Start() error {
    if err := r.Database.Start(r.Lifecycle); err != nil {
        return fmt.Errorf("database start failed: %w", err)
    }

    if err := r.Storage.Start(r.Lifecycle); err != nil {
        return fmt.Errorf("storage start failed: %w", err)
    }

    return nil
}
```

**Note**: Logger is `*slog.Logger` directly, not a System interface. Logging is functional infrastructure (no lifecycle, no commands, no events).

### Domain Structure

Domain holds stateless business logic systems:

```go
type Domain struct {
    Providers providers.System
    Agents    agents.System
    Documents documents.System
    Images    images.System
}

func NewDomain(runtime *Runtime) *Domain {
    docs := documents.New(
        runtime.Database.Connection(),
        runtime.Storage,
        runtime.Logger,
        runtime.Pagination,
    )

    return &Domain{
        Providers: providers.New(
            runtime.Database.Connection(),
            runtime.Logger,
            runtime.Pagination,
        ),
        Agents: agents.New(
            runtime.Database.Connection(),
            runtime.Logger,
            runtime.Pagination,
        ),
        Documents: docs,
        Images: images.New(
            docs,  // Domain dependency first
            runtime.Database.Connection(),
            runtime.Storage,
            runtime.Logger,
            runtime.Pagination,
        ),
    }
}
```

**Key Principles**:
1. **Runtime holds System interfaces** - Domain systems call methods like `runtime.Database.Connection()` to get what they need
2. **Domain dependencies first** - Cross-domain dependencies listed before runtime dependencies in constructors
3. **Domain systems pre-initialized** - Created at startup in `NewDomain()`, stored in Server struct

### Server Structure

Server ties Runtime and Domain together:

```go
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
```

### Hot Start Pattern

```go
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
```

### Main Entry Point

```go
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

**Note**: `config.Load()` now includes finalization internally - no separate `Finalize()` call needed.

### Graceful Shutdown

```go
func (s *Server) Shutdown(timeout time.Duration) error {
    s.runtime.Logger.Info("initiating shutdown")
    return s.runtime.Lifecycle.Shutdown(timeout)
}
```

**Lifecycle Coordinator Shutdown**:
```go
func (c *Coordinator) Shutdown(timeout time.Duration) error {
    c.cancel()  // Cancel context, triggering OnShutdown hooks

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

**Shutdown Flow**:
1. **Signal received** (SIGINT/SIGTERM)
2. **Service.Shutdown()** called → delegates to lifecycle.Shutdown()
3. **Lifecycle cancels context** → triggers all OnShutdown hooks
4. **Subsystems clean up**: Server gracefully closes connections, Database closes pool
5. **WaitGroup completes** → Shutdown returns → main() exits

**Key Points**:
- Lifecycle coordinator centralizes shutdown orchestration
- OnShutdown hooks wait for context cancellation before cleanup
- Timeout prevents indefinite hangs
- No data loss - in-flight requests complete before shutdown

## HTTP Server (cmd/server/http.go)

The HTTP server is implemented directly in the cmd/server package (not a separate internal package) since it's only used by the server entry point.

### Implementation

```go
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

**Note**: HTTP server uses OnShutdown only (not OnStartup) because ListenAndServe is a long-running process started in a goroutine.

## Database System (internal/database)

The Database system manages PostgreSQL connection pooling and lifecycle.

### System Interface

```go
type System interface {
    Connection() *sql.DB
    Start(lc *lifecycle.Coordinator) error
}
```

### Implementation

```go
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

**Note**: Database uses both OnStartup (ping to verify connectivity) and OnShutdown (close connection pool).

## Storage System (internal/storage)

The Storage system provides blob storage abstractions for file persistence.

### System Interface

```go
type System interface {
    Store(ctx context.Context, key string, data []byte) error
    Retrieve(ctx context.Context, key string) ([]byte, error)
    Delete(ctx context.Context, key string) error
    Validate(ctx context.Context, key string) (bool, error)
    Start(lc *lifecycle.Coordinator) error
}
```

### Error Types

```go
var (
    ErrNotFound         = errors.New("storage: key not found")
    ErrPermissionDenied = errors.New("storage: permission denied")
    ErrInvalidKey       = errors.New("storage: invalid key")
)
```

### Filesystem Implementation

```go
type filesystem struct {
    basePath string
    logger   *slog.Logger
}

func New(cfg *config.StorageConfig, logger *slog.Logger) (System, error) {
    if cfg.BasePath == "" {
        return nil, fmt.Errorf("base_path required")
    }

    absPath, err := filepath.Abs(cfg.BasePath)
    if err != nil {
        return nil, fmt.Errorf("resolve base_path: %w", err)
    }

    return &filesystem{
        basePath: absPath,
        logger:   logger.With("system", "storage"),
    }, nil
}

func (f *filesystem) Start(lc *lifecycle.Coordinator) error {
    f.logger.Info("starting storage system", "base_path", f.basePath)

    lc.OnStartup(func() {
        if err := os.MkdirAll(f.basePath, 0755); err != nil {
            f.logger.Error("storage initialization failed", "error", err)
            return
        }
        f.logger.Info("storage directory initialized")
    })

    return nil
}
```

### Atomic File Writes

Store uses temp file + rename for crash safety:

```go
func (f *filesystem) Store(ctx context.Context, key string, data []byte) error {
    path, err := f.fullPath(key)
    if err != nil {
        return err
    }

    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("create directory: %w", err)
    }

    tmpPath := path + ".tmp"
    if err := os.WriteFile(tmpPath, data, 0644); err != nil {
        return fmt.Errorf("write temp file: %w", err)
    }

    if err := os.Rename(tmpPath, path); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("rename temp file: %w", err)
    }

    return nil
}
```

### Delete with Directory Cleanup

Delete removes the file and cleans up empty parent directories (but never the base path):

```go
func (f *filesystem) Delete(ctx context.Context, key string) error {
    path, err := f.fullPath(key)
    if err != nil {
        return err
    }

    dir := filepath.Dir(path)

    if err := os.Remove(path); err != nil {
        if errors.Is(err, fs.ErrNotExist) {
            return nil // Idempotent
        }
        return err
    }

    // Clean up empty parent directory
    if dir != f.basePath && strings.HasPrefix(dir, f.basePath) {
        entries, err := os.ReadDir(dir)
        if err != nil {
            f.logger.Warn("failed to read directory for cleanup", "dir", dir, "error", err)
            return nil
        }

        if len(entries) == 0 {
            if err := os.Remove(dir); err != nil && !errors.Is(err, fs.ErrNotExist) {
                f.logger.Warn("failed to remove empty directory", "dir", dir, "error", err)
            }
        }
    }

    return nil
}
```

### Path Traversal Protection

The `fullPath` helper validates keys and prevents directory traversal:

```go
func (f *filesystem) fullPath(key string) (string, error) {
    if key == "" {
        return "", ErrInvalidKey
    }

    cleaned := filepath.Clean(key)
    if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
        return "", ErrInvalidKey
    }

    fullPath := filepath.Join(f.basePath, cleaned)

    if !strings.HasPrefix(fullPath, f.basePath) {
        return "", ErrInvalidKey
    }

    return fullPath, nil
}
```

**Note**: Storage uses OnStartup only (directory creation). No OnShutdown needed for filesystem.

## Route System (internal/routes)

**Functional Infrastructure**: Routes system is stateless infrastructure for organizing HTTP routing. Uses simple constructor, not config interface.

### System Interface

```go
type System interface {
    RegisterGroup(group Group)
    RegisterRoute(route Route)
    Build() http.Handler
}
```

### Implementation

```go
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

func (s *system) RegisterGroup(group Group) {
    s.groups = append(s.groups, group)
}

func (s *system) RegisterRoute(route Route) {
    s.routes = append(s.routes, route)
}

func (s *system) Build() http.Handler {
    mux := http.NewServeMux()

    for _, route := range s.routes {
        pattern := fmt.Sprintf("%s %s", route.Method, route.Pattern)
        mux.HandleFunc(pattern, route.Handler)
        s.logger.Info("registered route", "method", route.Method, "pattern", route.Pattern)
    }

    for _, group := range s.groups {
        for _, route := range group.Routes {
            pattern := fmt.Sprintf("%s %s%s", route.Method, group.Prefix, route.Pattern)
            mux.HandleFunc(pattern, route.Handler)
            s.logger.Info("registered route", "method", route.Method, "pattern", pattern)
        }
    }

    return mux
}
```

### Route Structures

```go
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
    OpenAPI *openapi.Operation  // Optional OpenAPI operation definition
}
```

### Usage Example

```go
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

## Middleware System (internal/middleware)

**Functional Infrastructure**: Middleware is stateless infrastructure that wraps HTTP handlers. Uses simple constructor with minimal parameters.

### System Interface

```go
type System interface {
    Use(mw func(http.Handler) http.Handler)
    Apply(handler http.Handler) http.Handler
}
```

### Implementation

```go
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

### Logger Middleware

```go
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

### CORS Middleware

```go
func CORS(cfg *config.CORSConfig) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")

            if origin != "" && isAllowedOrigin(origin, cfg.Origins) {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Set("Access-Control-Allow-Credentials", strconv.FormatBool(cfg.Credentials))

                if len(cfg.Headers) > 0 {
                    w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.Headers, ", "))
                }

                if len(cfg.Methods) > 0 {
                    w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.Methods, ", "))
                }

                if cfg.MaxAge > 0 {
                    w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
                }
            }

            if r.Method == http.MethodOptions {
                w.WriteHeader(http.StatusNoContent)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

### Usage Example

```go
func buildMiddleware(runtime *Runtime, cfg *config.Config) middleware.System {
    middlewareSys := middleware.New()
    middlewareSys.Use(middleware.Logger(runtime.Logger))
    middlewareSys.Use(middleware.CORS(&cfg.CORS))
    return middlewareSys
}
```

**Why Simple Constructor**: Middleware has minimal state and is functional infrastructure. No complex initialization or owned subsystems, so config interface would be overkill.

## API Documentation

### OpenAPI Specification

The API is documented using OpenAPI 3.1 specification generated at server startup.

**Generation Flow**:
1. Server startup calls `registerRoutes()`
2. Domain handlers registered with optional `OpenAPI` metadata on routes
3. `loadOrGenerateSpec()` generates spec in memory from route metadata
4. Compares with existing file, writes only if changed
5. Spec served from memory at `/api/openapi.json`

**Environment-Specific Specs**:
- Output location: `api/openapi.{env}.json`
- Environment determined by `SERVICE_ENV` (default: `local`)
- Example: `api/openapi.local.json`, `api/openapi.prod.json`

**Config-Driven Metadata**:
- `version`: API version (config.toml or `SERVICE_VERSION`)
- `domain`: Server URL (config.toml or `SERVICE_DOMAIN`)

### Schema Ownership

**Infrastructure** (`pkg/openapi/components.go`):
- Shared responses: `BadRequest`, `NotFound`, `Conflict`
- Shared schemas: `PageRequest`

**Domains** (`internal/<domain>/openapi.go`):
- Domain-specific schemas (e.g., `Provider`, `CreateProviderCommand`)
- Operation definitions (e.g., `Spec.Create`, `Spec.List`)

```go
// internal/providers/openapi.go
var Spec = spec{
    Create: &openapi.Operation{
        Summary:     "Create provider",
        Description: "Validates and stores a new provider configuration",
        RequestBody: openapi.RequestBodyJSON("CreateProviderCommand", true),
        Responses: map[int]*openapi.Response{
            201: openapi.ResponseJSON("Provider created", "Provider"),
            400: openapi.ResponseRef("BadRequest"),
            409: openapi.ResponseRef("Conflict"),
        },
    },
    // ... other operations
}

func (spec) Schemas() map[string]*openapi.Schema {
    return map[string]*openapi.Schema{
        "Provider": { /* ... */ },
        "CreateProviderCommand": { /* ... */ },
    }
}
```

### Route Integration

Routes reference domain operations:

```go
func (h *Handler) Routes() routes.Group {
    return routes.Group{
        Prefix:      "/api/providers",
        Tags:        []string{"Providers"},
        Description: "Provider configuration management",
        Routes: []routes.Route{
            {Method: "POST", Pattern: "", Handler: h.Create, OpenAPI: Spec.Create},
            {Method: "GET", Pattern: "", Handler: h.List, OpenAPI: Spec.List},
            // ...
        },
    }
}
```

### Scalar UI

Interactive documentation served at `/docs` endpoint.

**Architecture**:
- Self-hosted (no CDN dependencies)
- Assets embedded via `go:embed`
- Air-gap compatible

**Files** (`web/docs/`):
- `index.html` - HTML template loading Scalar
- `scalar.js` - Standalone JavaScript bundle
- `scalar.css` - Stylesheet
- `docs.go` - Handler serving assets

**Updating Scalar**:
```bash
./web/update-scalar.sh
```

### Import Hierarchy

```
pkg/openapi          - Types and helpers only (no internal imports)
internal/<domain>    - Domain-owned schemas and operations
cmd/server/openapi.go - Generator logic (imports pkg and internal)
web/docs             - Scalar UI handler (imports internal/routes)
```

## Logger Helper (cmd/server/logging.go)

Logger initialization is a simple helper function, not a System. The `*slog.Logger` is used directly since:
- No lifecycle management needed (no Start/Shutdown)
- No commands or events
- slog.Handler provides the extension point if needed

```go
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

## Pagination Package (pkg/pagination)

Reusable pagination utilities for all search operations.

### Configuration

```go
type Config struct {
    DefaultPageSize int `toml:"default_page_size"`
    MaxPageSize     int `toml:"max_page_size"`
}

func (c *Config) Finalize() error  // Applies defaults, env vars, validates
func (c *Config) Merge(overlay *Config)
```

### Request/Response Types

```go
type PageRequest struct {
    Page     int               `json:"page"`
    PageSize int               `json:"page_size"`
    Search   *string           `json:"search,omitempty"`
    Sort     []query.SortField `json:"sort,omitempty"`
}

func (r *PageRequest) Normalize(cfg Config)  // Clamps to valid ranges
func (r *PageRequest) Offset() int           // Calculates SQL OFFSET

// PageRequestFromQuery parses pagination from URL query parameters
// Supports: page, page_size, search, sort (comma-separated, "-" prefix for desc)
func PageRequestFromQuery(values url.Values, cfg Config) PageRequest

type PageResult[T any] struct {
    Data       []T `json:"data"`
    Total      int `json:"total"`
    Page       int `json:"page"`
    PageSize   int `json:"page_size"`
    TotalPages int `json:"total_pages"`
}

func NewPageResult[T any](data []T, total, page, pageSize int) PageResult[T]
```

## Query Package (pkg/query)

Three-layer architecture for building parameterized SQL queries.

### Layer 1: ProjectionMap (Structure Definition)

Static, reusable query structure per domain entity:

```go
type ProjectionMap struct {
    schema     string
    table      string
    alias      string
    columns    map[string]string  // viewName -> alias.column
    columnList []string           // ordered columns
}

func NewProjectionMap(schema, table, alias string) *ProjectionMap
func (p *ProjectionMap) Project(column, viewName string) *ProjectionMap
func (p *ProjectionMap) Table() string      // "schema.table alias"
func (p *ProjectionMap) Column(viewName string) string  // "alias.column"
func (p *ProjectionMap) Columns() string    // "alias.col1, alias.col2, ..."
```

**Usage**:
```go
var providerProjection = query.NewProjectionMap("public", "providers", "p").
    Project("id", "ID").
    Project("name", "Name").
    Project("config", "Config")
```

### Layer 2: Builder (Operations)

Fluent builder for filters, sorting, pagination:

```go
// SortField represents a column in ORDER BY clause
type SortField struct {
    Field      string
    Descending bool
}

// ParseSortFields parses "name,-createdAt" into []SortField
func ParseSortFields(s string) []SortField

type Builder struct {
    projection        *ProjectionMap
    conditions        []condition
    orderByFields     []SortField
    defaultSortFields []SortField
}

func NewBuilder(projection *ProjectionMap, defaultSort ...SortField) *Builder

// Filter methods (nil/empty values are ignored)
func (b *Builder) WhereEquals(field string, value any) *Builder
func (b *Builder) WhereContains(field string, value *string) *Builder
func (b *Builder) WhereIn(field string, values []any) *Builder
func (b *Builder) WhereSearch(search *string, fields ...string) *Builder
func (b *Builder) OrderByFields(fields []SortField) *Builder

// SQL generation
func (b *Builder) BuildCount() (sql string, args []any)
func (b *Builder) BuildPage(page, pageSize int) (sql string, args []any)
func (b *Builder) BuildSingle(idField string, id any) (sql string, args []any)
func (b *Builder) BuildSingleOrNull() (sql string, args []any)  // Use accumulated filters
```

**Usage**:
```go
qb := query.NewBuilder(providerProjection, query.SortField{Field: "Name"}).
    WhereContains("Name", req.Name).
    WhereSearch(req.Search, "Name", "Config")

if len(req.Sort) > 0 {
    qb.OrderByFields(req.Sort)
}

countSQL, countArgs := qb.BuildCount()
pageSQL, pageArgs := qb.BuildPage(req.Page, req.PageSize)
```

### Layer 3: Execution

Execute generated SQL with database/sql:

```go
// Count query
var total int
err := db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total)

// Page query
rows, err := db.QueryContext(ctx, pageSQL, pageArgs...)
```

## Repository Package (pkg/repository)

Generic database helpers that reduce boilerplate while keeping domain logic in calling code.

### Interfaces

```go
// Querier is implemented by *sql.DB, *sql.Tx, and *sql.Conn
type Querier interface {
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Executor is implemented by *sql.DB, *sql.Tx, and *sql.Conn
type Executor interface {
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Scanner abstracts row scanning (implemented by *sql.Row and *sql.Rows)
type Scanner interface {
    Scan(dest ...any) error
}

// ScanFunc converts a Scanner into a typed value (defined per domain)
type ScanFunc[T any] func(Scanner) (T, error)
```

### Transaction Helper

```go
// WithTx executes fn within a transaction, handling Begin/Commit/Rollback
func WithTx[T any](ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) (T, error)) (T, error)
```

### Query Helpers

```go
// QueryOne executes a query expecting a single row
func QueryOne[T any](ctx context.Context, q Querier, query string, args []any, scan ScanFunc[T]) (T, error)

// QueryMany executes a query expecting multiple rows
func QueryMany[T any](ctx context.Context, q Querier, query string, args []any, scan ScanFunc[T]) ([]T, error)

// ExecExpectOne executes a statement expecting exactly one affected row
func ExecExpectOne(ctx context.Context, e Executor, query string, args ...any) error
```

### Error Mapping

```go
// MapError translates database errors to domain errors
// sql.ErrNoRows → notFoundErr, pg 23505 → duplicateErr
func MapError(err error, notFoundErr, duplicateErr error) error
```

**Usage**:
```go
func (r *repo) Create(ctx context.Context, cmd CreateCommand) (*Provider, error) {
    q := `INSERT INTO providers (name, config) VALUES ($1, $2)
          RETURNING id, name, config, created_at, updated_at`

    p, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Provider, error) {
        return repository.QueryOne(ctx, tx, q, []any{cmd.Name, cmd.Config}, scanProvider)
    })

    if err != nil {
        return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
    }

    return &p, nil
}
```

## Handlers Package (pkg/handlers)

Stateless HTTP response utilities for JSON APIs.

```go
// RespondJSON writes a JSON response with status code
func RespondJSON(w http.ResponseWriter, status int, data any)

// RespondError logs the error and writes a JSON error response
func RespondError(w http.ResponseWriter, logger *slog.Logger, status int, err error)
```

**Usage**:
```go
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    var cmd CreateCommand
    if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
        handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
        return
    }

    result, err := h.sys.Create(r.Context(), cmd)
    if err != nil {
        handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
        return
    }

    handlers.RespondJSON(w, http.StatusCreated, result)
}
```

## Agent Execution Patterns

### VisionForm Pattern

Centralized multipart form parsing for vision endpoints with validation and base64 conversion:

```go
type VisionForm struct {
    Prompt  string
    Images  []string  // base64 data URIs
    Options map[string]any
    Token   string
}

func ParseVisionForm(r *http.Request, maxMemory int64) (*VisionForm, error) {
    if err := r.ParseMultipartForm(maxMemory); err != nil {
        return nil, fmt.Errorf("failed to parse multipart form: %w", err)
    }

    form := &VisionForm{
        Prompt: r.FormValue("prompt"),
        Token:  r.FormValue("token"),
    }

    // Validate required fields
    if form.Prompt == "" {
        return nil, fmt.Errorf("prompt is required")
    }

    // Parse optional JSON options
    if optStr := r.FormValue("options"); optStr != "" {
        if err := json.Unmarshal([]byte(optStr), &form.Options); err != nil {
            return nil, fmt.Errorf("invalid options JSON: %w", err)
        }
    }

    // Convert uploaded images to base64 data URIs
    files := r.MultipartForm.File["images"]
    if len(files) == 0 {
        return nil, fmt.Errorf("at least one image is required")
    }

    images, err := prepareImages(files)
    if err != nil {
        return nil, err
    }
    form.Images = images

    return form, nil
}
```

**Usage**:
```go
func (h *Handler) Vision(w http.ResponseWriter, r *http.Request) {
    form, err := ParseVisionForm(r, 32<<20)  // 32 MB memory limit
    if err != nil {
        handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
        return
    }

    agt, err := h.constructAgent(r.Context(), id, form.Token)
    // ... execute vision with form.Prompt, form.Images, form.Options
}
```

### Token Injection Pattern

Runtime token injection for providers requiring authentication (e.g., Azure):

```go
func (h *Handler) constructAgent(ctx context.Context, id uuid.UUID, token string) (agent.Agent, error) {
    record, err := h.sys.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }

    var cfg agtconfig.AgentConfig
    if err := json.Unmarshal(record.Config, &cfg); err != nil {
        return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
    }

    // Inject token at request time
    if token != "" {
        if cfg.Provider.Options == nil {
            cfg.Provider.Options = make(map[string]any)
        }
        cfg.Provider.Options["token"] = token
    }

    agt, err := agent.New(&cfg)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
    }

    return agt, nil
}
```

**Key Points**:
- Token passed at request time, not stored in database
- Stored config uses placeholder value (e.g., `"token": "token"`) for validation
- Real token injected into `Provider.Options["token"]` before agent construction

### SSE Streaming Pattern

Server-Sent Events for streaming responses:

```go
func (h *Handler) writeSSEStream(w http.ResponseWriter, r *http.Request, stream <-chan *response.StreamingChunk) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.WriteHeader(http.StatusOK)

    if f, ok := w.(http.Flusher); ok {
        f.Flush()
    }

    for chunk := range stream {
        if chunk.Error != nil {
            data, _ := json.Marshal(map[string]string{"error": chunk.Error.Error()})
            fmt.Fprintf(w, "data: %s\n\n", data)
            if f, ok := w.(http.Flusher); ok {
                f.Flush()
            }
            return
        }

        select {
        case <-r.Context().Done():
            return
        default:
        }

        data, _ := json.Marshal(chunk)
        fmt.Fprintf(w, "data: %s\n\n", data)
        if f, ok := w.(http.Flusher); ok {
            f.Flush()
        }
    }

    fmt.Fprintf(w, "data: [DONE]\n\n")
    if f, ok := w.(http.Flusher); ok {
        f.Flush()
    }
}
```

**Key Points**:
- `text/event-stream` content type
- Each chunk prefixed with `data: ` and followed by `\n\n`
- Flush after each chunk for real-time delivery
- Check context cancellation for client disconnect
- Final `[DONE]` marker signals stream completion

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

### TOML-Based Configuration

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

[logging]
level = "info"
format = "json"

[cors]
origins = ["http://localhost:3000"]
credentials = true
headers = ["Content-Type", "Authorization"]
methods = ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
max_age = 3600
```

### Encapsulated Configuration Pattern

**Principle**: Each configuration section owns its finalization logic through internal methods.

Each configuration struct implements internal methods:
1. **loadDefaults()** - Applies defaults for zero-value fields
2. **loadEnv()** - Maps `SECTION_FIELD` environment variables (replaces TOML values)
3. **validate()** - Validates field constraints

**Root Config Structure**:

```go
type Config struct {
    Server   ServerConfig   `toml:"server"`
    Database DatabaseConfig `toml:"database"`
    Logging  LoggingConfig  `toml:"logging"`
    CORS     CORSConfig     `toml:"cors"`

    ShutdownTimeout string `toml:"shutdown_timeout"`
}

func (c *Config) Finalize() error {
    c.loadDefaults()
    c.loadEnv()
    return c.validate()
}

func (c *Config) loadDefaults() {
    if c.ShutdownTimeout == "" {
        c.ShutdownTimeout = "30s"
    }
    c.Server.loadDefaults()
    c.Database.loadDefaults()
    c.Logging.loadDefaults()
    c.CORS.loadDefaults()
}

func (c *Config) loadEnv() {
    if v := os.Getenv(EnvServiceShutdownTimeout); v != "" {
        c.ShutdownTimeout = v
    }
    c.Server.loadEnv()
    c.Database.loadEnv()
    c.Logging.loadEnv()
    c.CORS.loadEnv()
}

func (c *Config) validate() error {
    if _, err := time.ParseDuration(c.ShutdownTimeout); err != nil {
        return fmt.Errorf("invalid shutdown_timeout: %w", err)
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
    return nil
}
```

**Section Config Example (ServerConfig)**:

```go
type ServerConfig struct {
    Host            string `toml:"host"`
    Port            int    `toml:"port"`
    ReadTimeout     string `toml:"read_timeout"`
    WriteTimeout    string `toml:"write_timeout"`
    ShutdownTimeout string `toml:"shutdown_timeout"`
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
    if v := os.Getenv(EnvServerHost); v != "" {
        c.Host = v
    }
    if v := os.Getenv(EnvServerPort); v != "" {
        if port, err := strconv.Atoi(v); err == nil {
            c.Port = port
        }
    }
    if v := os.Getenv(EnvServerReadTimeout); v != "" {
        c.ReadTimeout = v
    }
    if v := os.Getenv(EnvServerWriteTimeout); v != "" {
        c.WriteTimeout = v
    }
    if v := os.Getenv(EnvServerShutdownTimeout); v != "" {
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
```

### Environment Variable Convention

Environment variables follow the pattern: `SECTION_FIELD` (uppercase with underscores)

**TOML to Environment Variable Mapping**:

```
SERVICE_SHUTDOWN_TIMEOUT     → shutdown_timeout (root level)

[server]
  host                       → SERVER_HOST
  port                       → SERVER_PORT
  read_timeout               → SERVER_READ_TIMEOUT
  write_timeout              → SERVER_WRITE_TIMEOUT
  shutdown_timeout           → SERVER_SHUTDOWN_TIMEOUT

[database]
  host                       → DATABASE_HOST
  port                       → DATABASE_PORT
  name                       → DATABASE_NAME
  user                       → DATABASE_USER
  password                   → DATABASE_PASSWORD

[logging]
  level                      → LOGGING_LEVEL
  format                     → LOGGING_FORMAT

[cors]
  origins                    → CORS_ORIGINS (comma-separated)
  credentials                → CORS_CREDENTIALS (true/false)
  headers                    → CORS_HEADERS (comma-separated)
  methods                    → CORS_METHODS (comma-separated)
  max_age                    → CORS_MAX_AGE
```

**Docker vs Application Config**:

- `POSTGRES_*`, `OLLAMA_*` → Configure Docker containers
- `DATABASE_*`, `SERVER_*`, etc. → Override application TOML config

This separation allows running PostgreSQL locally (Docker) or connecting to managed databases (production) with the same application config pattern.

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

**Loading Sequence**:
1. Parse base TOML file (`config.toml`) into structs
2. Check for overlay file based on `SERVICE_ENV` environment variable
3. If overlay exists, parse and merge into base config
4. Apply environment variable overrides via `Finalize()` → `loadEnv()`
5. Apply defaults via `Finalize()` → `loadDefaults()`
6. Validate via `Finalize()` → `validate()`
7. Return validated configuration

## Database Patterns

### Connection Management

```go
import (
    "database/sql"
    _ "github.com/jackc/pgx/v5/stdlib"
)

func openDatabase(cfg *DatabaseConfig) (*sql.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
        cfg.Host, cfg.Port, cfg.Name, cfg.User, cfg.Password)

    db, err := sql.Open("pgx", dsn)
    if err != nil {
        return nil, fmt.Errorf("open database: %w", err)
    }

    return db, nil
}
```

### Transaction Pattern

Commands use `repository.WithTx` for automatic transaction management:

```go
func (r *repo) Create(ctx context.Context, cmd CreateCommand) (*Provider, error) {
    q := `INSERT INTO providers (name, config) VALUES ($1, $2)
          RETURNING id, name, config, created_at, updated_at`

    p, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Provider, error) {
        return repository.QueryOne(ctx, tx, q, []any{cmd.Name, cmd.Config}, scanProvider)
    })

    if err != nil {
        return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
    }

    return &p, nil
}
```

### Query Pattern

Queries use repository helpers without transactions:

```go
func (r *repo) FindByID(ctx context.Context, id uuid.UUID) (*Provider, error) {
    q, args := query.NewBuilder(projection).BuildSingle("ID", id)

    p, err := repository.QueryOne(ctx, r.db, q, args, scanProvider)
    if err != nil {
        return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
    }

    return &p, nil
}
```

### Domain Scanner Pattern

Each domain defines a scanner function in its mapping.go file:

```go
// internal/providers/mapping.go
func scanProvider(s repository.Scanner) (Provider, error) {
    var p Provider
    err := s.Scan(&p.ID, &p.Name, &p.Config, &p.CreatedAt, &p.UpdatedAt)
    return p, err
}
```

### Cross-Domain Dependencies

Domain systems can depend on other domain systems when operations require
cross-domain data access. Dependencies must be unidirectional to prevent cycles.

**Pattern**: Inject dependent systems via constructor with domain dependencies first:
```go
// images depends on documents (unidirectional)
func New(
    docs documents.System,  // Domain dependencies first
    db *sql.DB,             // Then runtime dependencies
    storage storage.System,
    logger *slog.Logger,
    pagination pagination.Config,
) System {
    return &repo{
        db:         db,
        documents:  docs,
        storage:    storage,
        logger:     logger,
        pagination: pagination,
    }
}
```

**Rules**:
- Domain dependencies listed before runtime dependencies in constructor signatures
- Dependencies flow one direction only (A → B, never B → A)
- Inject via constructor, not method parameters
- Use interface types to avoid import cycles
- Wire at server startup in `cmd/server/domain.go`

## Error Handling

### Encapsulated Package Errors

Each package defines its errors in a dedicated `errors.go` file for discoverability and consistent organization.

```go
// internal/database/errors.go
package database

import "errors"

var ErrNotReady = errors.New("database not ready")
```

**Convention**:
- Package-level errors live in `errors.go`
- Use `Err` prefix for exported error variables
- Error messages are lowercase, no punctuation
- Enables clean external usage: `database.ErrNotReady`, `providers.ErrNotFound`

**Directory Structure**:
```
internal/database/
├── errors.go      # Package errors
└── database.go    # Implementation

internal/providers/
├── errors.go      # Package errors (ErrNotFound, ErrInvalidConfig, etc.)
├── mapping.go     # Projection, scanner, and filters
├── provider.go    # State definitions
└── repository.go  # System implementation
```

### Error Wrapping

```go
import "fmt"

if err := doSomething(); err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

### Structured Logging

```go
logger.Info("operation succeeded", "id", id, "name", name)
logger.Error("operation failed", "error", err, "id", id)
```

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
func TestLoad_Scenarios(t *testing.T) {
    tests := []struct {
        name      string
        setup     func()
        cleanup   func()
        expectErr bool
    }{
        {
            name:      "loads base config",
            setup:     func() {},
            cleanup:   func() {},
            expectErr: false,
        },
        {
            name: "invalid duration returns error",
            setup: func() {
                os.Setenv("SERVER_READ_TIMEOUT", "invalid")
            },
            cleanup: func() {
                os.Unsetenv("SERVER_READ_TIMEOUT")
            },
            expectErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tt.setup()
            defer tt.cleanup()

            _, err := config.Load()
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

```
tests/
├── cmd_server/           # Server integration tests
├── internal_agents/      # Agents domain tests
├── internal_config/      # Config package tests
├── internal_documents/   # Documents domain tests
├── internal_images/      # Images domain tests
├── internal_lifecycle/   # Lifecycle coordinator tests
├── internal_middleware/  # Middleware package tests
├── internal_providers/   # Providers domain tests
├── internal_routes/      # Routes package tests
├── internal_storage/     # Storage system tests
├── pkg_handlers/         # HTTP handlers tests
├── pkg_openapi/          # OpenAPI package tests
├── pkg_pagination/       # Pagination package tests
├── pkg_query/            # Query builder tests
├── pkg_repository/       # Repository helpers tests
└── web_docs/             # Documentation UI tests
```

## Pattern Decision Guide

### Configuration Pattern

**All stateful systems use concrete config structs** with the simplified pattern:

```go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    // ... other sections
}

func (c *Config) Finalize() error {
    c.loadDefaults()   // Apply defaults to all sections
    c.loadEnv()        // Apply environment variable overrides
    return c.validate() // Validate all sections
}

func (c *Config) loadDefaults() { /* cascade to sections */ }
func (c *Config) loadEnv()      { /* cascade to sections */ }
func (c *Config) validate() error { /* cascade to sections */ }
```

**Each config section** follows the same pattern with internal methods.

### System Constructor Pattern

**Stateful Systems** (Server, Database):
```go
func NewServer(cfg *Config) (*Server, error) {
    // Config is already finalized (validated)
    // Build system from config values
}
```

**Functional Infrastructure** (Handlers, middleware, routing, logging):
```go
func HandleCreate(w http.ResponseWriter, r *http.Request, system System, logger *slog.Logger)
func New(logger *slog.Logger) System
```

### Configuration Lifecycle

```
Load TOML → Config struct → Finalize() → NewService() → [Config discarded]
                              ↓
                    defaults → env → validate
```

**Key principle**: Configuration is ephemeral - used only for initialization, then garbage collected.

## References

- **web-service-architecture.md**: Complete architectural philosophy and design decisions
- **service-design.md**: Future designs and patterns not yet implemented
- **go-agents**: Configuration patterns, interface design, LCA principles
- **go-agents-orchestration**: Workflow patterns (future milestones)
- **document-context**: LCA architecture (future milestones)
