# Web Service Architecture

**Established**: 2025-11-22
**Purpose**: Define the architectural principles, layer structure, and design patterns for agent-lab

---

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
Layer 1: Server (Composition Root)
  cmd/server/
  ↓ initializes with config

Layer 2: Core Systems (Domain)
  internal/[providers|agents|database|routes|middleware]
  ↓ uses

Layer 3: Public Infrastructure (Shared Toolkit)
  pkg/[query|pagination]
  ↓ uses

Layer 4: External Services
  PostgreSQL, Future: Blob Storage, Cache, etc.
```

**Rules**:
- Each layer only knows about its direct dependencies
- Dependencies only ever provided as interfaces by owners
- State flows DOWN through parameters
- Events flow UP through return values or channels

### 4. State vs Process Initialization

**Cold Start** (State Initialization):
- `New*()` constructor functions
- Builds entire dependency graph
- All configurations → State objects
- All systems created but dormant
- No processes running
- Returns ready-to-start system

**Hot Start** (Process Initialization):
- `Start()` methods
- State objects → Running processes
- Cascade start through dependency graph
- Context boundaries for lifecycle management
- System becomes interactable
- Blocks until shutdown

Example:
```go
// Cold Start - Build state graph
srv, err := NewServer(cfg)  // All systems initialized but dormant

// Hot Start - Activate processes
ctx := context.Background()
err := srv.Start(ctx)  // Cascades start, blocks until shutdown
```

### 5. System Interface Contract

Every system provides:

1. **Internal State** (private) - Only accessible within the system
2. **Internal Processes** (private) - Implementation details
3. **Getter Methods** (public) - Immutable access to state
4. **Commands** (public) - Write operations from owner (downward)
5. **Events** (public, optional) - Notifications to owner (upward)

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

---

## Package Structure

### **Layer 1: Server (Composition Root)**

```
cmd/server/
├── main.go          # Entry point: Load config → Cold start → Hot start
├── server.go        # Server system (owns all subsystems)
└── config.go        # Configuration loading and validation
```

**Responsibilities**:
- Bootstrap application
- Compose all Layer 2 systems
- Manage application lifecycle
- Handle graceful shutdown

### **Layer 2: Core Systems (Domain)**

```
internal/
├── providers/       # Provider Configuration System
│   ├── provider.go      # State structures
│   ├── system.go        # Interface definition
│   ├── repository.go    # System implementation
│   ├── handlers.go      # HTTP handlers
│   └── routes.go        # Route group definition
│
├── agents/          # Agent Configuration System
│   └── (same structure as providers)
│
├── database/        # Database Connection System
│   └── database.go      # Connection pool management
│
├── routes/          # HTTP Route Registration System
│   ├── routes.go        # Route system & group management
│   └── group.go         # Route group definition
│
└── middleware/      # HTTP Middleware System
    └── middleware.go    # Middleware stack (CORS, logging, etc.)
```

**Responsibilities**:
- Domain-specific business logic
- Database operations
- HTTP request handling
- Route registration

### **Layer 3: Public Infrastructure (Shared Toolkit)**

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

---

## Configuration Interface Pattern

### Stateful Systems vs Functional Infrastructure

**Apply config interface pattern to**:
- Systems that own state and other systems
- Complex initialization requiring finalize/validate/transform
- Examples: Server, Database, Providers, Agents

**Use simple parameters for**:
- Stateless or minimal-state utilities
- Functional infrastructure with no owned subsystems
- Examples: Handlers, middleware, routing, query builders

### Config Interface Benefits

1. **Makes required state immediately apparent** - Look at the interface to see all dependencies
2. **Self-describing initialization** - Config can implement its own `Finalize()` and `Validate()`
3. **Configuration graph** - Parent configs contain child configs for owned systems
4. **Clear ownership boundaries** - Explicit relationship between systems
5. **Easier testing** - Mock the interface, not individual parameters

### Configuration Graph Pattern

Parent system configs contain child configs for all owned subsystems:

```go
// Server owns Database, Providers, Agents, etc.
type ServerConfig interface {
    // Subsystem configs (configuration graph)
    Database() DatabaseConfig
    Providers() ProvidersConfig
    Agents() AgentsConfig

    // Direct configuration
    Pagination() pagination.Config
    Logging() LoggingConfig
    HTTP() HTTPServerConfig
    CORS() CORSConfig

    // Self-describing initialization
    Finalize()
    Validate() error
}

// ProvidersConfig has its own dependencies
type ProvidersConfig interface {
    DB() *sql.DB
    Logger() *slog.Logger
    Pagination() pagination.Config

    Finalize()
    Validate() error
}
```

---

## System Design Patterns

### Server System (Layer 1)

```go
// cmd/server/config.go - Configuration interfaces
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

// cmd/server/server.go
type Server struct {
    config     ServerConfig
    db         database.System
    logger     *slog.Logger
    providers  providers.System
    agents     agents.System
    middleware middleware.System
    routes     routes.System
}

// Cold Start - Initialize state dependency graph
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

// Hot Start - Activate processes
func (s *Server) Start(ctx context.Context) error {
    // Cascade start to subsystems
    if err := s.db.Start(ctx); err != nil {
        return err
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

// Getters - Nouns (access owned state)
func (s *Server) Config() *Config { return s.config }
func (s *Server) Logger() *slog.Logger { return s.logger }
func (s *Server) Providers() providers.System { return s.providers }
func (s *Server) Agents() agents.System { return s.agents }
```

### Domain System (Layer 2)

```go
// internal/providers/system.go - Configuration & System interfaces
type ProvidersConfig interface {
    DB() *sql.DB
    Logger() *slog.Logger
    Pagination() pagination.Config

    Finalize()
    Validate() error
}

type System interface {
    // Commands - Verbs (actions)
    Create(ctx context.Context, cmd CreateCommand) (*Provider, error)
    Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Provider, error)
    Delete(ctx context.Context, id uuid.UUID) error
    Search(ctx context.Context, req SearchRequest) (*SearchResult, error)

    // Getters - Nouns (state access)
    // Note: For repository pattern, these are still verbs since they're queries
    FindByID(ctx context.Context, id uuid.UUID) (*Provider, error)
}

// internal/providers/repository.go - Implementation
type repository struct {
    db         *sql.DB
    logger     *slog.Logger
    pagination pagination.Config
}

func New(cfg ProvidersConfig) (System, error) {
    // 1. Finalize: Apply defaults
    cfg.Finalize()

    // 2. Validate: Check requirements
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

// internal/providers/handlers.go - Functional infrastructure (simple parameters)
func HandleCreate(
    w http.ResponseWriter,
    r *http.Request,
    system System,  // Interface injected from owner
    logger *slog.Logger,
) {
    // Parse request
    var cmd CreateCommand
    if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
        respondError(w, logger, http.StatusBadRequest, err)
        return
    }

    // Execute on system (state flows DOWN)
    result, err := system.Create(r.Context(), cmd)
    if err != nil {
        respondError(w, logger, http.StatusInternalServerError, err)
        return
    }

    respondJSON(w, logger, http.StatusCreated, result)
}

// NOTE: Handlers are functional infrastructure - they use simple parameters,
// not config interfaces. They're stateless and well-encapsulated as-is.

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
            // More routes...
        },
    }
}
```

### Route System (Layer 2)

**Functional Infrastructure** - Uses simple constructor, not config interface. Stateless utility for organizing routes.

```go
// internal/routes/routes.go
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

// internal/routes/group.go
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

### Database System (Layer 2)

```go
// internal/database/database.go - Configuration & System interfaces
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

type System interface {
    // Getters - Nouns (state access)
    Connection() *sql.DB

    // Commands - Verbs (actions)
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}

type database struct {
    conn   *sql.DB
    logger *slog.Logger
}

func New(cfg DatabaseConfig) (System, error) {
    // 1. Finalize: Apply defaults
    cfg.Finalize()

    // 2. Validate: Check requirements
    if err := cfg.Validate(); err != nil {
        return nil, err
    }

    // 3. Transform: Build connection
    dsn := fmt.Sprintf(
        "host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
        cfg.Host(), cfg.Port(), cfg.Name(), cfg.User(), cfg.Password())

    db, err := sql.Open("pgx", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to open: %w", err)
    }

    db.SetMaxOpenConns(cfg.MaxOpenConns())
    db.SetMaxIdleConns(cfg.MaxIdleConns())
    db.SetConnMaxLifetime(cfg.ConnMaxLifetime())

    return &database{
        conn:   db,
        logger: cfg.Logger().With("system", "database"),
    }, nil
}

// Getter - Noun (state access)
func (d *database) Connection() *sql.DB {
    return d.conn
}

// Command - Verb (action)
func (d *database) Start(ctx context.Context) error {
    d.logger.Info("starting database connection")

    pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    if err := d.conn.PingContext(pingCtx); err != nil {
        return fmt.Errorf("failed to ping: %w", err)
    }

    d.logger.Info("database connection established")
    return nil
}
```

---

## Main Entry Point

```go
// cmd/server/main.go
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

---

## Current Milestone Scope

**What We're Building NOW (Milestone 1)**:

✅ Layer 1: Server system with graceful shutdown
✅ Layer 2: Provider Configuration System (CRUD + Search)
✅ Layer 2: Agent Configuration System (CRUD + Search)
✅ Layer 2: Database System (connection pool)
✅ Layer 2: Routes System (smart grouping)
✅ Layer 2: Middleware System (CORS, logging)
✅ Layer 3: Query Builder (pkg/query)
✅ Layer 3: Pagination (pkg/pagination)

**What We're NOT Building Yet**:

❌ Authentication/Authorization
❌ OpenAPI generation infrastructure
❌ Agent Protocol execution (Chat, Vision, Tools, Embed)
❌ Document processing
❌ Workflow orchestration
❌ Blob storage
❌ Caching layer

**Principle**: Build only what's needed for the current milestone. Add infrastructure when there's a concrete requirement.

---

## Key Decisions

### 1. No PaginationService
- Eliminated `PaginationService` - it was just a config wrapper
- Use `PaginationConfig` directly
- Pagination logic as functions in `pkg/pagination`

### 2. Long-Running vs Ephemeral Systems
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

### 3. Routes and Middleware as Systems
- Not just functions - proper packages with clear responsibilities
- Routes system manages smart grouping and registration
- Middleware system manages request/response pipeline
- Each domain defines its own route group

### 4. Handler Pattern
- Handlers are **pure functions**, not structs with stored state
- State flows DOWN through parameters
- Each handler receives: `(w, r, system, logger)`
- Handlers live in domain packages (e.g., `providers/handlers.go`)

### 5. Interface Boundaries
- Systems expose interfaces, not concrete types
- Enables testing with mocks
- Clear contract between layers
- Owner defines interface, implementation is private

### 6. Naming Conventions
- **Getters**: Pure nouns - `Id()`, `Name()`, `Connection()`
- **Commands**: Action verbs - `Start()`, `Create()`, `Update()`
- **Events**: On* prefix - `OnShutdown()`, `OnProgress()`

---

## Migration Notes

**From Previous Architecture**:

| Old Concept | New Concept | Rationale |
|-------------|-------------|-----------|
| `Application` struct in `internal/app` | `Server` struct in `cmd/server` | Server IS the application; no need for extra layer |
| `PaginationService` | `PaginationConfig` | Just config + utility functions; not a stateful system |
| Handlers store `*Application` | Handlers receive state as params | State flows DOWN, never stored upward |
| Systems created per-request | Systems are long-running | No current need for request context |
| Routes in `routes()` method | Routes system with smart grouping | Better organization, domain boundaries |
| Flat middleware | Middleware system | Proper package, composable stack |
| Everything in `internal/` | Split `internal/` + `pkg/` | Public infrastructure vs private implementation |
| `GetId()`, `GetName()` | `Id()`, `Name()` | Pure nouns for getters |

**Breaking Changes**:
- Complete restructure required
- Cannot incrementally migrate
- Start fresh with new architecture

---

## Testing Strategy

### Unit Tests
- Test systems with mocked dependencies
- Test handlers with mocked systems
- Test route registration
- Black-box testing (`package name_test`)

### Integration Tests
- Test with real database (Docker)
- Test full HTTP request/response cycle
- Test graceful shutdown

### What to Test
- ✅ Repository operations (CRUD, search)
- ✅ Query builder correctness
- ✅ Pagination calculations
- ✅ Handler request/response formats
- ✅ Route registration
- ✅ Middleware behavior

### What NOT to Test Yet
- ❌ Authentication (doesn't exist)
- ❌ Authorization (doesn't exist)
- ❌ OpenAPI generation (not building yet)

---

## Next Steps

1. **Re-align Documentation**:
   - Update ARCHITECTURE.md with this design
   - Update CLAUDE.md with new principles
   - Update PROJECT.md with new vocabulary
   - Update README.md with new structure

2. **Rewrite Implementation Guide**:
   - Completely rewrite `_context/01-milestone-1-foundation.md`
   - Follow new architecture principles
   - Focus only on Milestone 1 scope

3. **Implement**:
   - Execute updated implementation guide
   - Validate with tests
   - Document with godoc

4. **Future Milestones**:
   - Add authentication when needed
   - Add agent protocols when needed
   - Add OpenAPI generation when beneficial
   - Each addition follows established patterns
