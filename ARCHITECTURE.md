# agent-lab Architecture

**Status**: Living Document - Flexible guideline that adapts as requirements emerge

## Overview

agent-lab is a containerized Go web service platform for building and orchestrating agentic workflows. It builds upon the foundation of go-agents, go-agents-orchestration, and document-context libraries, providing a production-ready HTTP API for intelligent document processing and workflow execution.

## Architectural Principles

### 1. Layered Composition Architecture (LCA)

The system follows LCA principles established in the underlying Go libraries:

- **Data vs Behavior Separation**: Configuration structs (data) are distinct from domain objects (behavior)
- **Explicit Boundaries**: Validation occurs at transformation boundaries
- **Interface-Based APIs**: Public contracts through interfaces, private implementations
- **Immutability Where Possible**: Configuration immutable after initialization
- **Fail-Fast Validation**: Invalid configurations rejected at creation time

### 2. Service Lifecycle Model

**Long-Running Services** (Application-scoped)
- Initialized at server startup
- Live for the application lifetime
- Examples: Database connection pool, logger, configuration
- Owned by the `Application` struct

**Ephemeral Services** (Request-scoped)
- Initialized per HTTP request
- Exist only for request duration
- Composed from: Application state + request context
- Form dependency chains hierarchically
- Examples: ItemService, OrderService

**Key Benefits:**
- No consolidated "Services" struct that becomes brittle
- Service hierarchies emerge naturally based on use cases
- Request-scoped state flows through service chains
- Clear separation of long-running vs ephemeral concerns

### 3. Configuration-Driven Initialization

All services use `New*` constructor functions following a consistent pattern:

```go
func NewItemService(db *sql.DB, logger *slog.Logger, userID string) (*ItemService, error) {
    // 1. Finalize: Apply defaults
    if logger == nil {
        logger = slog.Default()
    }

    // 2. Validate: Check required dependencies
    if db == nil {
        return nil, errors.New("database required")
    }
    if userID == "" {
        return nil, errors.New("user ID required")
    }

    // 3. Transform: Create service instance
    return &ItemService{
        db:     db,
        logger: logger.With("service", "item", "user_id", userID),
        userID: userID,
    }, nil
}
```

**Pattern: Finalize → Validate → Transform**

This ensures:
- Validation at every initialization boundary
- Objects always in valid state
- Clear error reporting at construction time
- Consistent initialization across all components

## System Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      HTTP Requests                          │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                    Middleware                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                   │
│  │  Logger  │→ │ Recovery │→ │   Auth   │ (future)          │
│  └──────────┘  └──────────┘  └──────────┘                   │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                    Handlers                                 │
│  ┌────────────────────────────────────────────────────┐     │
│  │  Initialize Ephemeral Services per Request         │     │
│  │  (ItemService, OrderService, etc.)                 │     │
│  └────────────────────────────────────────────────────┘     │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│              Ephemeral Services                             │
│  ┌──────────────────────────────────────────────────┐       │
│  │  Business Logic (Queries + Commands)             │       │
│  │  - Validation                                    │       │
│  │  - Transaction Management                        │       │
│  │  - Domain Logic                                  │       │
│  └──────────────────────────────────────────────────┘       │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                    Models                                   │
│  ┌──────────────────────────────────────────────────┐       │
│  │  Pure Data Structures                            │       │
│  │  - Entities                                      │       │
│  │  - Commands                                      │       │
│  │  - Filters                                       │       │
│  └──────────────────────────────────────────────────┘       │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                  Database (PostgreSQL)                      │
└─────────────────────────────────────────────────────────────┘
```

### Application Struct (Long-Running State)

```go
type Application struct {
    config *config.Config
    logger *slog.Logger
    db     *sql.DB
}
```

The Application struct:
- Holds long-running dependencies
- Lives for the entire server lifetime
- Passed to handlers at initialization
- Provides accessor methods for dependencies

### Models (Pure Data Structures)

Models define only data structures with no methods:

```go
// Entity
type Item struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// Command
type CreateItemCommand struct {
    Name        string
    Description string
}

// Filter
type ItemFilters struct {
    Name   string
    Search string
}
```

**Responsibilities:**
- Define structure of domain data
- JSON serialization tags
- Database mapping (through query patterns)
- NO business logic
- NO database operations
- NO validation logic

### Services (Ephemeral, Request-Scoped)

Services encapsulate business logic and contain both queries (read operations) and commands (write operations):

```go
type ItemService struct {
    db     *sql.DB
    logger *slog.Logger
    userID string  // Request context
}

// Queries (read-only operations)
func (s *ItemService) Get(ctx context.Context, id string) (*models.Item, error)
func (s *ItemService) List(ctx context.Context, filters models.ItemFilters) ([]*models.Item, error)

// Commands (write operations with transactions)
func (s *ItemService) Create(ctx context.Context, cmd models.CreateItemCommand) (*models.Item, error)
func (s *ItemService) Update(ctx context.Context, id string, cmd models.UpdateItemCommand) (*models.Item, error)
func (s *ItemService) Delete(ctx context.Context, id string) error
```

**Queries vs Commands (Conceptual Distinction):**

- **Queries**: Read operations, no mutations, can be cached
- **Commands**: Write operations, always use transactions, mutate state

This is a **conceptual pattern**, not a structural requirement. Services are not split into separate QueryService/CommandService - they're unified per domain.

**Command Pattern:**
```go
func (s *ItemService) Create(ctx context.Context, cmd models.CreateItemCommand) (*models.Item, error) {
    // Begin transaction
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Validate command
    if err := s.validateCreate(cmd); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    // Execute atomic mutation
    query := `
        INSERT INTO items (id, name, description, created_at, updated_at)
        VALUES (gen_random_uuid(), $1, $2, NOW(), NOW())
        RETURNING id, name, description, created_at, updated_at`

    var item models.Item
    err = tx.QueryRowContext(ctx, query, cmd.Name, cmd.Description).Scan(
        &item.ID, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt)
    if err != nil {
        return nil, fmt.Errorf("insert item: %w", err)
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("commit transaction: %w", err)
    }

    s.logger.Info("item created", "item_id", item.ID, "name", item.Name)
    return &item, nil
}
```

**Services Beyond SQL:**

Not all services interact with databases:
```go
// NotificationService - external API calls
type NotificationService struct {
    emailClient *smtp.Client
    logger      *slog.Logger
}

// CacheService - Redis or in-memory cache
type CacheService struct {
    client *redis.Client
    logger *slog.Logger
}
```

### Handlers (HTTP Layer)

Handlers are dedicated structs per domain resource:

```go
type ItemHandler struct {
    app *server.Application
}

func NewItemHandler(app *server.Application) *ItemHandler {
    return &ItemHandler{app: app}
}

func (h *ItemHandler) Create(w http.ResponseWriter, r *http.Request) {
    var cmd models.CreateItemCommand
    if err := readJSON(r, &cmd); err != nil {
        h.clientError(w, http.StatusBadRequest)
        return
    }

    ctx := r.Context()
    userID := "system" // Will come from auth middleware

    // Initialize ephemeral service for this request
    svc, err := services.NewItemService(h.app.DB(), h.app.Logger(), userID)
    if err != nil {
        h.serverError(w, r, err)
        return
    }

    // Execute command
    item, err := svc.Create(ctx, cmd)
    if err != nil {
        h.serverError(w, r, err)
        return
    }

    respondJSON(w, http.StatusCreated, envelope{"item": item})
}
```

**Benefits:**
- Handlers only access what they need (Application dependencies)
- Clear per-domain organization
- Easier testing (mock Application interface)
- Aligns with single responsibility principle

### Routing (http.ServeMux)

Using Go 1.22+ standard library routing:

```go
func (app *application) routes() http.Handler {
    mux := http.NewServeMux()

    // Health check
    mux.HandleFunc("GET /health", healthHandler)

    // Item endpoints
    itemHandler := handlers.NewItemHandler(app)
    mux.HandleFunc("GET /api/items", itemHandler.List)
    mux.HandleFunc("GET /api/items/{id}", itemHandler.Get)
    mux.HandleFunc("POST /api/items", itemHandler.Create)
    mux.HandleFunc("PUT /api/items/{id}", itemHandler.Update)
    mux.HandleFunc("DELETE /api/items/{id}", itemHandler.Delete)

    // Wrap with middleware
    return app.recoverPanic(app.logRequest(mux))
}
```

**Path Parameters:**
```go
id := r.PathValue("id")  // From URL pattern /api/items/{id}
```

**Why http.ServeMux:**
- Zero dependencies
- Go 1.22+ has method and path parameter support
- Sufficient for RESTful APIs
- Native to Go ecosystem

If blocking limitations emerge, re-evaluate with chi or httprouter.

## Configuration Management

### Layered Configuration Loading

Configuration is loaded in priority order:

1. `config.yaml` - Base defaults
2. `config.{ENV}.yaml` - Environment-specific (ENV=development, production, staging, etc.)
3. `config.local.yaml` - Local overrides (gitignored)
4. Environment variables - Highest priority

### Configuration Structure

```go
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Database DatabaseConfig `yaml:"database"`
    Logging  LoggingConfig  `yaml:"logging"`
}

type ServerConfig struct {
    Port         int           `yaml:"port"`
    Host         string        `yaml:"host"`
    ReadTimeout  time.Duration `yaml:"read_timeout"`
    WriteTimeout time.Duration `yaml:"write_timeout"`
    IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

type DatabaseConfig struct {
    Host        string        `yaml:"host"`
    Port        int           `yaml:"port"`
    Database    string        `yaml:"database"`
    User        string        `yaml:"user"`
    Password    string        `yaml:"password"`
    MaxConns    int           `yaml:"max_conns"`
    MinConns    int           `yaml:"min_conns"`
    MaxIdleTime time.Duration `yaml:"max_idle_time"`
}

type LoggingConfig struct {
    Level  string `yaml:"level"`   // debug, info, warn, error
    Format string `yaml:"format"`  // text, json
}
```

### Environment Variable Convention

Environment variables mirror the YAML structure using underscores:

**Simple values:**
```bash
server_port=8080
database_host=localhost
database_password=secret
logging_level=debug
```

**Arrays (indexed convention - Kubernetes pattern):**
```bash
cors_origins_0=http://localhost:3000
cors_origins_1=http://localhost:4000
```

**Nested objects in arrays:**
```bash
database_replicas_0_host=db1
database_replicas_0_port=5432
database_replicas_0_weight=1

database_replicas_1_host=db2
database_replicas_1_port=5432
database_replicas_1_weight=2
```

**Rationale:**
- Mirrors YAML structure (intuitive mapping)
- Self-documenting (clear what each variable controls)
- Scales naturally (adding fields is straightforward)
- Standard Kubernetes/cloud-native convention
- Supports complex nested arrays

### External Configuration Sources

Environment variables work implicitly across deployment scenarios:

- **Local Development**: `.env` file or shell export
- **Docker**: Environment variables in docker-compose.yml
- **Kubernetes**: ConfigMaps and Secrets mounted as environment variables
- **Cloud Platforms**: Platform-provided environment variables

No explicit external config loading needed - standard environment variable mechanism handles all cases.

## Database Architecture

### Connection Management

Using `database/sql` with pgx driver (raw SQL approach):

```go
func Open(cfg DatabaseConfig) (*sql.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
        cfg.Host, cfg.Port, cfg.Database, cfg.User, cfg.Password)

    db, err := sql.Open("pgx", dsn)
    if err != nil {
        return nil, fmt.Errorf("open database: %w", err)
    }

    // Configure connection pool
    db.SetMaxOpenConns(cfg.MaxConns)
    db.SetMaxIdleConns(cfg.MinConns)
    db.SetConnMaxIdleTime(cfg.MaxIdleTime)

    // Verify connection
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        return nil, fmt.Errorf("ping database: %w", err)
    }

    return db, nil
}
```

### Query Patterns

**Single Row Query:**
```go
query := `SELECT id, name, description FROM items WHERE id = $1`
var item Item
err := db.QueryRowContext(ctx, query, id).Scan(&item.ID, &item.Name, &item.Description)
if err == sql.ErrNoRows {
    return nil, ErrNotFound
}
```

**Multiple Rows Query:**
```go
query := `SELECT id, name, description FROM items ORDER BY created_at DESC`
rows, err := db.QueryContext(ctx, query)
if err != nil {
    return nil, err
}
defer rows.Close()

items := []*Item{}
for rows.Next() {
    var item Item
    if err := rows.Scan(&item.ID, &item.Name, &item.Description); err != nil {
        return nil, err
    }
    items = append(items, &item)
}
return items, rows.Err()
```

**Transaction Pattern:**
```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

// Execute mutations within transaction
_, err = tx.ExecContext(ctx, query, args...)
if err != nil {
    return err
}

return tx.Commit()
```

### Why Raw SQL?

- **Full control**: See exactly what queries execute
- **Performance**: No ORM overhead
- **Learning**: Understand database/sql patterns at low level
- **Simplicity**: Minimal dependencies (just driver)
- **Flexibility**: Can add query builders (squirrel, goqu) later if needed

Future evolution may introduce query building patterns similar to the .NET ProjectionMap/QueryBuilder approach, but only when complexity justifies it.

### Database Migrations

SQL migrations are stored in `/migrations` directory:

```
migrations/
├── 000001_create_items.up.sql
├── 000001_create_items.down.sql
├── 000002_add_users.up.sql
└── 000002_add_users.down.sql
```

**Migration tools**: Use `golang-migrate/migrate` or similar tool (to be configured during development).

## Middleware Patterns

### Request Logging

```go
func (app *application) logRequest(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        app.logger.Info("request",
            "method", r.Method,
            "uri", r.URL.RequestURI(),
            "addr", r.RemoteAddr)
        next.ServeHTTP(w, r)
    })
}
```

### Panic Recovery

```go
func (app *application) recoverPanic(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                w.Header().Set("Connection", "close")
                app.logger.Error("panic recovered",
                    "error", err,
                    "method", r.Method,
                    "uri", r.URL.RequestURI())
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

### Middleware Chaining

```go
func (app *application) routes() http.Handler {
    mux := http.NewServeMux()
    // ... register routes

    // Chain middleware (innermost to outermost)
    return app.recoverPanic(app.logRequest(mux))
}
```

For more complex chaining, consider `justinas/alice` or similar composition library.

## Error Handling

### Error Response Pattern

```go
type envelope map[string]any

func respondJSON(w http.ResponseWriter, status int, data any) error {
    js, err := json.MarshalIndent(data, "", "  ")
    if err != nil {
        return err
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    w.Write(js)
    w.Write([]byte("\n"))

    return nil
}

func respondError(w http.ResponseWriter, status int, message string) {
    respondJSON(w, status, envelope{"error": message})
}
```

### Service-Level Errors

```go
var (
    ErrNotFound     = errors.New("resource not found")
    ErrInvalidInput = errors.New("invalid input")
    ErrUnauthorized = errors.New("unauthorized")
)
```

### Handler Error Helpers

```go
func (h *ItemHandler) serverError(w http.ResponseWriter, r *http.Request, err error) {
    h.app.Logger().Error("server error",
        "error", err,
        "method", r.Method,
        "uri", r.URL.RequestURI())
    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

func (h *ItemHandler) notFound(w http.ResponseWriter) {
    respondError(w, http.StatusNotFound, "resource not found")
}
```

## Logging

### Structured Logging with slog

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

logger.Info("server starting", "port", 8080, "env", "production")
logger.Error("database error", "error", err, "query", query)
```

### Contextual Logging

Services receive loggers with context:

```go
func NewItemService(db *sql.DB, logger *slog.Logger, userID string) (*ItemService, error) {
    return &ItemService{
        logger: logger.With("service", "item", "user_id", userID),
    }, nil
}

// All logs from this service include service and user_id attributes
s.logger.Info("creating item", "name", name)
// Output: {"level":"info","service":"item","user_id":"abc123","name":"laptop"}
```

## Testing Strategy

### Unit Tests

Test services with mocked database:

```go
func TestItemService_Create(t *testing.T) {
    // Mock database
    mockDB := &MockDB{
        execFunc: func(query string, args ...interface{}) (sql.Result, error) {
            return mockResult{}, nil
        },
    }

    svc, _ := NewItemService(mockDB, slog.Default(), "test-user")

    item, err := svc.Create(context.Background(), CreateItemCommand{
        Name: "Test Item",
    })

    require.NoError(t, err)
    assert.Equal(t, "Test Item", item.Name)
}
```

### Integration Tests

Test with real database (Docker test containers):

```go
func TestItemService_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    db := setupTestDatabase(t)
    defer cleanupTestDatabase(t, db)

    svc, _ := NewItemService(db, slog.Default(), "test-user")

    // Test with real database
    item, err := svc.Create(context.Background(), CreateItemCommand{
        Name: "Integration Test Item",
    })

    require.NoError(t, err)

    // Verify in database
    retrieved, err := svc.Get(context.Background(), item.ID)
    require.NoError(t, err)
    assert.Equal(t, item.Name, retrieved.Name)
}
```

### API Tests

Test handlers with httptest:

```go
func TestItemHandler_Create(t *testing.T) {
    app := setupTestApplication(t)
    handler := NewItemHandler(app)

    body := `{"name":"API Test Item"}`
    req := httptest.NewRequest("POST", "/api/items", strings.NewReader(body))
    w := httptest.NewRecorder()

    handler.Create(w, req)

    assert.Equal(t, http.StatusCreated, w.Code)

    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    assert.Equal(t, "API Test Item", response["item"].(map[string]interface{})["name"])
}
```

## Server Lifecycle

### Graceful Shutdown

```go
func run() error {
    // ... setup ...

    srv := &http.Server{
        Addr:    ":8080",
        Handler: app.routes(),
    }

    // Channel for shutdown signal
    shutdown := make(chan error, 1)

    // Start server
    go func() {
        if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            shutdown <- err
        }
    }()

    // Listen for interrupt
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

    // Block until signal or error
    select {
    case err := <-shutdown:
        return err
    case sig := <-quit:
        logger.Info("shutting down", "signal", sig)
    }

    // Graceful shutdown with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    return srv.Shutdown(ctx)
}
```

## Deployment Considerations

### Environment Selection

Set `ENV` environment variable to select configuration:

```bash
ENV=development go run ./cmd/server  # Loads config.development.yaml
ENV=production go run ./cmd/server   # Loads config.production.yaml
```

### Docker Deployment

Environment variables passed to container:

```bash
docker run \
  -e ENV=production \
  -e database_host=db.example.com \
  -e database_password=secret \
  -p 8080:8080 \
  agent-lab:latest
```

### Kubernetes Deployment

ConfigMaps and Secrets mounted as environment variables:

```yaml
env:
  - name: ENV
    value: "production"
  - name: database_host
    valueFrom:
      configMapKeyRef:
        name: agent-lab-config
        key: database_host
  - name: database_password
    valueFrom:
      secretKeyRef:
        name: agent-lab-secrets
        key: database_password
```

## Future Evolution

This architecture is designed to evolve based on emerging requirements:

### Event-Driven Side Effects

When cross-service reactions become necessary, introduce event system:
- Event bus for publishing domain events
- Event handlers for side effects
- Async processing for non-critical operations

### Query Builder Patterns

If dynamic query building becomes repetitive:
- Introduce squirrel or goqu for programmatic SQL construction
- Consider ProjectionMap pattern from .NET architecture
- Keep raw SQL for complex queries

### Authentication & Authorization

When multi-user support is needed:
- JWT-based authentication middleware
- User context extraction from tokens
- Role-based authorization
- Integration with identity providers

### Observability

As system complexity grows:
- Distributed tracing (OpenTelemetry)
- Metrics collection (Prometheus)
- Enhanced structured logging
- Request correlation IDs

## Conclusion

This architecture provides a solid foundation for building agent-lab while remaining flexible enough to adapt as requirements emerge. Key principles:

- Start with standard library, add dependencies when justified
- Learn low-level patterns before introducing abstractions
- Separate long-running and ephemeral concerns
- Validate at boundaries, fail fast
- Configuration-driven initialization
- Clear separation of data and behavior

The architecture will evolve through practical experience building features, not through premature optimization or over-engineering.
