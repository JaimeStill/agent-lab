# agent-lab Architecture

**Status**: Current implementation state
**Scope**: Milestone 1 foundation - Infrastructure and service lifecycle

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
- **cmd/service**: The process (composition root, entry point)
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

### 6. Configuration-Driven Initialization

**Stateful Systems vs Functional Infrastructure**:

- **Stateful Systems**: Use concrete config structs with simplified finalize pattern (Service, Database, Logger)
- **Functional Infrastructure**: Use simple function signatures (handlers, middleware, routing)

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
cmd/service/          # Process: Composition root
├── main.go               # Entry point
├── service.go            # Service system
├── middleware.go         # Middleware configuration
└── routes.go             # Route registration

internal/             # Private API: Domain systems
├── config/
│   ├── config.go         # Root configuration
│   ├── server.go         # Server configuration
│   ├── database.go       # Database configuration
│   ├── logging.go        # Logging configuration
│   ├── cors.go           # CORS configuration
│   └── types.go          # Shared types
│
├── logger/
│   └── logger.go         # Logger system
│
├── routes/
│   ├── routes.go         # Route system
│   └── group.go          # Route group definition
│
├── middleware/
│   ├── middleware.go     # Middleware system
│   ├── logger.go         # Logger middleware
│   └── cors.go           # CORS middleware
│
└── server/
    └── server.go         # HTTP server system

tests/                # Black-box tests
├── internal_config/      # Config package tests
├── internal_logger/      # Logger package tests
├── internal_routes/      # Routes package tests
├── internal_middleware/  # Middleware package tests
├── internal_server/      # Server package tests
└── cmd_service/          # Service integration tests
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

## Service System (cmd/service)

The Service system is the composition root that owns all subsystems and manages the application lifecycle.

### Service Structure

```go
type Service struct {
    ctx        context.Context
    cancel     context.CancelFunc
    shutdownWg sync.WaitGroup

    logger logger.System
    server server.System
}
```

**Note**: Service is stateless - it only holds references to subsystems. Configuration is ephemeral and not stored.

### Cold Start Pattern

```go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Logging  LoggingConfig
    CORS     CORSConfig

    ShutdownTimeout string `toml:"shutdown_timeout"`
}

func (c *Config) Finalize() error {
    c.loadDefaults()
    c.loadEnv()
    return c.validate()
}

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
```

**Simplified Configuration Pattern**:
- Config is a concrete struct, not an interface
- Single `Finalize() error` method orchestrates initialization (vs old pattern of separate Finalize() then Validate() calls)
- Finalize() cascades through config graph: loadDefaults → loadEnv → validate → child Finalize()
- Each config section handles its own defaults, environment variables, and validation
- Config is ephemeral - used only during initialization, then discarded

### Hot Start Pattern

```go
func (s *Service) Start() error {
    s.logger.Logger().Info("starting service")

    if err := s.server.Start(s.ctx, &s.shutdownWg); err != nil {
        return fmt.Errorf("server start failed: %w", err)
    }

    s.logger.Logger().Info("service started")
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

### Graceful Shutdown

The service implements graceful shutdown using context cancellation and coordinated timeout handling.

**Signal Handling (main.go)**:
```go
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
```

**Service Shutdown**:
```go
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

**HTTP Server Shutdown (server.Start)**:
```go
func (s *server) Start(ctx context.Context, wg *sync.WaitGroup) error {
    wg.Go(func() {
        if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            s.logger.Error("server error", "error", err)
        }
    })

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
```

**Shutdown Flow**:
1. **Signal received** (SIGINT/SIGTERM) → context cancelled in main
2. **Service.Shutdown()** called → cancels service context
3. **HTTP server** gracefully closes connections (configurable timeout)
4. **WaitGroup** waits for all subsystems to complete
5. **Service.Shutdown()** returns → main() exits

**Key Points**:
- Context cascades through all systems for coordinated shutdown
- HTTP server gets configurable timeout to finish in-flight requests
- WaitGroup ensures all subsystems complete before exit
- No data loss - processes complete before shutdown

## Server System (internal/server)

The Server system encapsulates the HTTP server and manages its lifecycle.

### System Interface

```go
type System interface {
    Start(ctx context.Context, wg *sync.WaitGroup) error
    Stop(ctx context.Context) error
}
```

### Implementation

```go
type server struct {
    http            *http.Server
    logger          *slog.Logger
    shutdownTimeout time.Duration
}

func New(cfg *config.ServerConfig, handler http.Handler, logger *slog.Logger) System {
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

func (s *server) Start(ctx context.Context, wg *sync.WaitGroup) error {
    wg.Go(func() {
        if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            s.logger.Error("server error", "error", err)
        }
    })

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
func buildMiddleware(loggerSys logger.System, cfg *config.Config) middleware.System {
    middlewareSys := middleware.New()
    middlewareSys.Use(middleware.Logger(loggerSys.Logger()))
    middlewareSys.Use(middleware.CORS(&cfg.CORS))
    return middlewareSys
}
```

**Why Simple Constructor**: Middleware has minimal state and is functional infrastructure. No complex initialization or owned subsystems, so config interface would be overkill.

## Logger System (internal/logger)

### System Interface

```go
type System interface {
    Logger() *slog.Logger
}
```

### Implementation

```go
type logger struct {
    logger *slog.Logger
}

func New(cfg *config.LoggingConfig) System {
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

func (l *logger) Logger() *slog.Logger {
    return l.logger
}

func parseLevel(level string) slog.Level {
    switch level {
    case "debug":
        return slog.LevelDebug
    case "info":
        return slog.LevelInfo
    case "warn":
        return slog.LevelWarn
    case "error":
        return slog.LevelError
    default:
        return slog.LevelInfo
    }
}
```

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

```
tests/
├── internal_config/      # Mirrors internal/config
│   ├── config_test.go
│   ├── server_test.go
│   ├── database_test.go
│   ├── logging_test.go
│   ├── cors_test.go
│   └── types_test.go
│
├── internal_logger/      # Mirrors internal/logger
│   └── logger_test.go
│
├── internal_routes/      # Mirrors internal/routes
│   ├── routes_test.go
│   └── group_test.go
│
├── internal_middleware/  # Mirrors internal/middleware
│   ├── middleware_test.go
│   ├── logger_test.go
│   └── cors_test.go
│
├── internal_server/      # Mirrors internal/server
│   └── server_test.go
│
└── cmd_service/          # Mirrors cmd/service
    └── service_test.go
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

**Stateful Systems** (Service, Server, Logger):
```go
func NewService(cfg *Config) (*Service, error) {
    // Config is already finalized (validated)
    // Build system from config values
}
```

**Functional Infrastructure** (Handlers, middleware, routing):
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
