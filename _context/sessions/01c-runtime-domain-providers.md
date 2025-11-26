# Session 1c: Runtime/Domain System Separation + Providers System

**Date**: 2025-11-26
**Status**: Completed

## Objective

Establish Runtime/Domain system separation pattern and implement the first domain system (Providers) with CRUD and search operations.

## What Was Built

### Runtime/Domain System Separation

Introduced a two-tier system separation pattern that clearly distinguishes between infrastructure (Runtime) and business logic (Domain):

**Runtime** (application-scoped, lifecycle-managed):
- `Lifecycle` - Coordinator for startup/shutdown orchestration
- `Logger` - `*slog.Logger` directly (not a System interface)
- `Database` - Connection pool with lifecycle integration
- `Pagination` - Configuration for search operations

**Domain** (request-scoped, stateless):
- `Providers` - Provider configuration management

**Server** ties Runtime and Domain together with HTTP server.

### Providers Domain System

Complete domain system implementation:

- **State Structures** (`provider.go`): Provider, CreateCommand, UpdateCommand
- **Domain Errors** (`errors.go`): ErrNotFound, ErrDuplicate, ErrInvalidConfig
- **Query Projection** (`projection.go`): Static column mapping for query builder
- **System Interface** (`system.go`): Create, Update, Delete, FindByID, Search
- **Repository** (`repository.go`): Implementation with transactions and validation
- **HTTP Handlers** (`handlers.go`): Request/response handling with error mapping
- **Routes** (`routes.go`): Route group for `/api/providers` endpoints

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/providers` | Create provider |
| GET | `/api/providers/{id}` | Get provider by ID |
| PUT | `/api/providers/{id}` | Update provider |
| DELETE | `/api/providers/{id}` | Delete provider |
| POST | `/api/providers/search` | Search with pagination |

### Database Migration

```sql
CREATE TABLE providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    config JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_providers_name ON providers(name);
```

## Architectural Decisions

### Logger is NOT a System

Logger doesn't fit the System pattern because:
- No lifecycle management needed (no Start/Shutdown)
- No commands or events
- `slog.Handler` provides the extension point if needed

Result: `*slog.Logger` used directly in Runtime, not wrapped in an interface.

### Config Load() Consolidation

`Load()` now includes finalization internally:
- Loads base TOML
- Merges overlay if present
- Applies defaults
- Applies environment overrides
- Validates

No separate `Finalize()` call needed by consumers.

### Domain Errors Pattern

Each domain package defines errors in `errors.go`:
```go
var (
    ErrNotFound      = errors.New("provider not found")
    ErrDuplicate     = errors.New("provider name already exists")
    ErrInvalidConfig = errors.New("invalid provider config")
)
```

Handlers map domain errors to HTTP status codes:
- `ErrNotFound` → 404
- `ErrDuplicate` → 409
- `ErrInvalidConfig` → 400

### Provider Config Validation

Configs validated using go-agents:
1. Unmarshal JSON into `agtconfig.ProviderConfig`
2. Attempt `agtproviders.Create(&cfg)` to validate
3. Wrap errors with `ErrInvalidConfig`

## Files Changed

### New Files
- `cmd/migrate/migrations/002_providers.sql`
- `cmd/server/runtime.go`
- `cmd/server/domain.go`
- `cmd/server/http.go`
- `cmd/server/logging.go`
- `internal/providers/` (entire package)

### Modified Files
- `cmd/server/main.go` - Updated for new patterns
- `cmd/server/server.go` - Composition of Runtime + Domain + HTTP
- `cmd/server/routes.go` - Provider routes registration
- `cmd/server/middleware.go` - Updated for Runtime pattern
- `internal/config/config.go` - Load() includes finalization
- Removed: `internal/logger/` (replaced by logging.go helper)
- Removed: `internal/server/` (replaced by http.go in cmd/server)

### Test Updates
- `tests/internal_config/config_test.go` - Restructured for Load() pattern
- Renamed: `tests/cmd_service/` → `tests/cmd_server/`

## Session Review Notes

Issues identified for future sessions:
1. **Error Mapping Pattern**: Need better pattern for domain error → HTTP status mapping
2. **GET Search Endpoint**: Need alternative to POST search that parses query parameters

These are deferred to Session 1d or a dedicated refactoring session.

## Validation Results

All API endpoints tested and working:
- Create provider with Ollama config: 201 Created
- Get provider by ID: 200 OK
- Search providers: 200 OK with pagination
- Update provider: 200 OK
- Delete provider: 204 No Content
- Duplicate name: 409 Conflict
- Invalid config: 400 Bad Request
- Not found: 404 Not Found

## Next Session

**Session 1d: Agents System**
- Database schema: `agents` table with foreign key to providers
- Agents domain system following providers pattern
- Agent CRUD + Search endpoints
- Provider reference validation
