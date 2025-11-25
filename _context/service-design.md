# agent-lab Service Design

**Status**: Conceptual designs specific to agent-lab
**Scope**: Not yet implemented patterns and systems

This document captures agent-lab specific architectural designs that have not yet been implemented. For general web service patterns and principles, see [web-service-architecture.md](./web-service-architecture.md).

## Database System Enhancement

The Database system will be enhanced beyond simple connection management to support lifecycle operations.

### Enhanced System Interface

```go
type System interface {
    // Getters
    Connection() *sql.DB

    // Commands
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health(ctx context.Context) error
}
```

### Implementation Approach

```go
type database struct {
    conn   *sql.DB
    logger *slog.Logger
}

func New(cfg *DatabaseConfig, logger *slog.Logger) (System, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
        cfg.Host, cfg.Port, cfg.Name, cfg.User, cfg.Password)

    db, err := sql.Open("pgx", dsn)
    if err != nil {
        return nil, fmt.Errorf("open database: %w", err)
    }

    db.SetMaxOpenConns(cfg.MaxOpenConns)
    db.SetMaxIdleConns(cfg.MaxIdleConns)
    db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

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
    d.logger.Info("closing database connection")
    return d.conn.Close()
}

func (d *database) Health(ctx context.Context) error {
    return d.conn.PingContext(ctx)
}
```

## Provider Management System

Provider management enables CRUD operations for LLM provider configurations (OpenAI, Anthropic, Ollama, etc.).

### Directory Structure

```
internal/providers/
├── provider.go       # State structures
├── system.go         # Interface definition
├── repository.go     # System implementation
├── handlers.go       # HTTP handlers
└── routes.go         # Route group definition
```

### State Structures

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

type UpdateCommand struct {
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

type SearchResult struct {
    Data  []Provider
    Total int
    Page  int
}
```

### System Interface

```go
type System interface {
    Create(ctx context.Context, cmd CreateCommand) (*Provider, error)
    Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Provider, error)
    Delete(ctx context.Context, id uuid.UUID) error
    FindByID(ctx context.Context, id uuid.UUID) (*Provider, error)
    Search(ctx context.Context, req SearchRequest) (*SearchResult, error)
}
```

### Repository Implementation Pattern

```go
type repository struct {
    db         *sql.DB
    logger     *slog.Logger
    pagination pagination.Config
}

func New(db *sql.DB, logger *slog.Logger, pagination pagination.Config) (System, error) {
    return &repository{
        db:         db,
        logger:     logger.With("system", "providers"),
        pagination: pagination,
    }, nil
}

func (r *repository) Create(ctx context.Context, cmd CreateCommand) (*Provider, error) {
    // Validate configuration by creating go-agents provider instance
    var cfg config.ProviderConfig
    if err := json.Unmarshal(cmd.Config, &cfg); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    _, err := providers.New(&cfg)
    if err != nil {
        return nil, fmt.Errorf("invalid provider: %w", err)
    }

    // Transaction pattern
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

### HTTP Handler Pattern

Handlers are pure functions that receive state as parameters:

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

func HandleGetByID(
    w http.ResponseWriter,
    r *http.Request,
    system System,
    logger *slog.Logger,
) {
    idStr := r.PathValue("id")
    id, err := uuid.Parse(idStr)
    if err != nil {
        respondError(w, logger, http.StatusBadRequest, fmt.Errorf("invalid id: %w", err))
        return
    }

    result, err := system.FindByID(r.Context(), id)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            respondError(w, logger, http.StatusNotFound, err)
        } else {
            respondError(w, logger, http.StatusInternalServerError, err)
        }
        return
    }

    respondJSON(w, logger, http.StatusOK, result)
}

func HandleSearch(
    w http.ResponseWriter,
    r *http.Request,
    system System,
    logger *slog.Logger,
) {
    var req SearchRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, logger, http.StatusBadRequest, err)
        return
    }

    result, err := system.Search(r.Context(), req)
    if err != nil {
        respondError(w, logger, http.StatusInternalServerError, err)
        return
    }

    respondJSON(w, logger, http.StatusOK, result)
}
```

### Route Group Definition

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

## Agent Management System

Agent management follows the same pattern as Provider management but for go-agents agent configurations.

### Directory Structure

```
internal/agents/
├── agent.go          # State structures
├── system.go         # Interface definition
├── repository.go     # System implementation
├── handlers.go       # HTTP handlers
└── routes.go         # Route group definition
```

### State Structures

```go
type Agent struct {
    ID         uuid.UUID
    Name       string
    ProviderID uuid.UUID
    Config     json.RawMessage
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type CreateCommand struct {
    Name       string
    ProviderID uuid.UUID
    Config     json.RawMessage
}

type UpdateCommand struct {
    Name       string
    ProviderID uuid.UUID
    Config     json.RawMessage
}

type SearchRequest struct {
    Name       *string
    ProviderID *uuid.UUID
    Page       int
    PageSize   int
    SortBy     string
    Desc       bool
}
```

### Validation Pattern

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

    // Proceed with database insertion using transaction pattern
}
```

## Query Infrastructure

Three-layer architecture for building parameterized SQL queries.

### Layer 1: ProjectionMap (Structure Definition)

Static, reusable query structure per domain entity:

```go
var providerProjection = query.NewProjectionMap("public", "providers", "p").
    Project("id", "Id").
    Project("name", "Name").
    Project("config", "Config").
    Project("created_at", "CreatedAt").
    Project("updated_at", "UpdatedAt")
```

**Purpose**:
- Defines tables, joins, column mappings
- Resolves view property names to `table.column` references
- Reusable across all queries for the entity

### Layer 2: QueryBuilder (Operations)

Fluent builder for filters, sorting, pagination:

```go
type QueryBuilder interface {
    WhereEquals(field string, value interface{}) QueryBuilder
    WhereContains(field string, value *string) QueryBuilder
    WhereSearch(search *string, fields ...string) QueryBuilder
    OrderBy(field string, descending bool) QueryBuilder

    BuildCount() (sql string, args []interface{})
    BuildPage(page, pageSize int) (sql string, args []interface{})
    BuildSingle(id uuid.UUID) (sql string, args []interface{})
}
```

**Features**:
- Automatic null-checking: only applies filters when values are non-null
- Fluent API for composing queries
- Generates complete SQL with parameterized args

### Layer 3: Execution (database/sql)

Execute generated SQL with database/sql:

```go
func (r *repository) Search(ctx context.Context, req SearchRequest) (*SearchResult, error) {
    qb := query.NewBuilder(providerProjection, "Name").
        WhereContains("Name", req.Name).
        OrderBy(req.SortBy, req.Desc)

    // Execute count query
    countSQL, countArgs := qb.BuildCount()
    var total int
    err := r.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total)
    if err != nil {
        return nil, fmt.Errorf("count: %w", err)
    }

    // Execute page query
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

## Pagination Infrastructure

Reusable pagination structures for all search operations.

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

**Usage Pattern**:
- Never expose simple GET all endpoints
- Always use paginated search with filters
- Two-query pattern: COUNT for total, SELECT with OFFSET/FETCH

## Database Schema Design

### Providers Table

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

### Agents Table

```sql
CREATE TABLE agents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    config      JSONB NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(name, provider_id)
);

CREATE INDEX idx_agents_name ON agents(name);
CREATE INDEX idx_agents_provider_id ON agents(provider_id);
CREATE INDEX idx_agents_created_at ON agents(created_at DESC);
```

### Migration Strategy

- Sequential numbered migrations: `001_initial_schema.sql`, `002_add_agents.sql`
- Applied manually via psql or migration tool
- Track applied migrations in `schema_migrations` table

## Error Handling Patterns

### Domain-Level Errors

```go
// internal/providers/errors.go
var (
    ErrNotFound     = errors.New("provider not found")
    ErrInvalidInput = errors.New("invalid provider input")
    ErrDuplicate    = errors.New("provider name already exists")
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

### Error Mapping Pattern

```go
func HandleCreate(w http.ResponseWriter, r *http.Request, system System, logger *slog.Logger) {
    result, err := system.Create(r.Context(), cmd)
    if err != nil {
        switch {
        case errors.Is(err, ErrInvalidInput):
            respondError(w, logger, http.StatusBadRequest, err)
        case errors.Is(err, ErrDuplicate):
            respondError(w, logger, http.StatusConflict, err)
        default:
            respondError(w, logger, http.StatusInternalServerError, err)
        }
        return
    }

    respondJSON(w, logger, http.StatusCreated, result)
}
```

## Testing Strategy

### Unit Tests (Black-Box)

Test systems with mocked dependencies:

```go
package providers_test

import (
    "context"
    "testing"

    "github.com/JaimeStill/agent-lab/internal/providers"
)

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

Test with real database (skip if unavailable):

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

### Table-Driven Test Pattern

```go
func TestProviderValidation(t *testing.T) {
    tests := []struct {
        name      string
        cmd       CreateCommand
        expectErr bool
    }{
        {
            name: "valid openai provider",
            cmd: CreateCommand{
                Name:   "openai",
                Config: json.RawMessage(`{"type":"openai","apiKey":"sk-..."}`),
            },
            expectErr: false,
        },
        {
            name: "missing api key",
            cmd: CreateCommand{
                Name:   "openai",
                Config: json.RawMessage(`{"type":"openai"}`),
            },
            expectErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := repo.Create(context.Background(), tt.cmd)
            if tt.expectErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

## Deployment Considerations

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
    container_name: agent-lab-service
    ports:
      - "8080:8080"
    environment:
      DATABASE_HOST: postgres
      DATABASE_PORT: 5432
      DATABASE_NAME: agent_lab
      DATABASE_USER: agent_lab
      DATABASE_PASSWORD: secure_password
    depends_on:
      - postgres
```

### Kubernetes Deployment

Configuration via environment variables:
- ConfigMap for non-sensitive settings
- Secrets for database credentials, API keys
- Service discovery for database host

## Integration with go-agents Ecosystem

### Provider Configuration Validation

Provider configurations are validated by creating go-agents provider instances during Create/Update operations:

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

**Validation ensures**:
- Configuration structure matches go-agents expectations
- Required fields are present (API keys, endpoints, etc.)
- Provider type is supported
- Early failure before database persistence

### Agent Configuration Validation

Agent configurations are validated similarly:

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

**Benefits**:
- Leverages go-agents validation logic
- Prevents invalid configurations from being stored
- Ensures configurations can be instantiated when needed
- Consistent validation between agent-lab and go-agents
