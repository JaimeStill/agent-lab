# Session 01b: Database & Query Infrastructure

**Status**: Completed
**Date**: 2025-11-25
**Milestone**: 01 - Foundation & Infrastructure

## Summary

Session 1b established the database connectivity and query infrastructure that will support all domain systems. Building on the foundation from Session 1a, this session introduced the lifecycle coordinator pattern, database system with connection pooling, migration tooling, and query/pagination utilities.

## Implemented Components

### New Packages

| Package | Purpose |
|---------|---------|
| `internal/lifecycle` | Application lifecycle coordination (startup/shutdown orchestration) |
| `internal/database` | PostgreSQL connection management with lifecycle integration |
| `pkg/pagination` | Pagination configuration and request/result structures |
| `pkg/query` | SQL query builder with projection mapping |
| `cmd/migrate` | Database migration CLI using golang-migrate |

### Key Files Created

- `internal/lifecycle/lifecycle.go` - Coordinator type with OnStartup/OnShutdown hooks
- `internal/database/database.go` - Database system with connection pooling
- `internal/database/errors.go` - Package-level errors
- `pkg/pagination/config.go` - Pagination configuration with env var support
- `pkg/pagination/pagination.go` - PageRequest and PageResult types
- `pkg/query/projection.go` - ProjectionMap for table/column mapping
- `pkg/query/builder.go` - Fluent query builder
- `cmd/migrate/main.go` - Migration CLI with embedded migrations
- `cmd/migrate/migrations/000001_initial_schema.up.sql` - Initial migration

### Modified Files

- `internal/config/config.go` - Added Pagination config section
- `internal/server/server.go` - Updated to use lifecycle.Coordinator
- `cmd/service/service.go` - Integrated lifecycle coordinator and database
- `cmd/service/routes.go` - Added `/readyz` endpoint
- `cmd/service/main.go` - Simplified shutdown using lifecycle coordinator
- `config.toml` - Added pagination section

## Architectural Patterns Established

### Lifecycle Coordinator Pattern

The `lifecycle.Coordinator` formalizes startup/shutdown orchestration:

- **OnStartup(fn)**: Register tasks that must complete for service readiness
- **OnShutdown(fn)**: Register cleanup tasks triggered on context cancellation
- **Ready()**: One-time gate - true after all startup tasks complete
- **WaitForStartup()**: Blocks until all startup tasks finish

This pattern decouples subsystem lifecycle from the Service struct, allowing subsystems to register their own startup/shutdown behavior without the Service knowing implementation details.

### ReadinessChecker Interface

Decouples readiness checking from the Coordinator implementation:
```go
type ReadinessChecker interface {
    Ready() bool
}
```

This allows routes to depend on an interface rather than the concrete Coordinator type.

### Subsystem Start Pattern

Subsystems receive the lifecycle coordinator and register their hooks:

- **Database**: Uses both OnStartup (ping) and OnShutdown (close connection)
- **Server**: Uses OnShutdown only (ListenAndServe is long-running, started in goroutine)

### Query Builder Three-Layer Architecture

1. **ProjectionMap**: Static structure defining table/column mappings
2. **Builder**: Fluent API for filters, sorting, pagination
3. **Execution**: Generated SQL used with database/sql

## Key Decisions

### Lifecycle Coordinator vs Direct Context/WaitGroup

**Decision**: Introduced `lifecycle.Coordinator` to encapsulate context, WaitGroups, and readiness state.

**Rationale**: The Service struct was accumulating lifecycle-related fields (ctx, cancel, shutdownWg). Encapsulating these in a dedicated type:
- Clarifies the startup/shutdown contract
- Allows subsystems to register hooks without Service knowing details
- Provides clean readiness tracking with thread-safe access

### No OnRun Hook

**Decision**: Removed OnRun from the lifecycle coordinator design.

**Rationale**: Long-running processes like `ListenAndServe` should be started in goroutines directly by the subsystem. The lifecycle coordinator tracks startup tasks (that complete) and shutdown tasks (triggered by context cancellation), not ongoing processes.

### Pagination Config in pkg/

**Decision**: Placed pagination Config in `pkg/pagination` with TOML tags and config methods.

**Rationale**: This allows `internal/config` to import and embed the pagination config while keeping pagination types co-located. The pagination package owns its own configuration behavior.

## Test Coverage

All new packages have comprehensive black-box tests:

- `tests/pkg_pagination/` - Config finalization, PageRequest normalization, PageResult creation
- `tests/pkg_query/` - ProjectionMap, Builder with various filter combinations
- `tests/internal_lifecycle/` - Coordinator startup, shutdown, readiness, timeout handling

Server tests updated to use the new lifecycle.Coordinator pattern.

## Validation Results

- Service starts successfully with database connection
- `/healthz` returns 200 (liveness)
- `/readyz` returns 503 before startup complete, 200 after (readiness)
- Migration CLI runs successfully
- Graceful shutdown closes all subsystems in order
- All tests passing (100% for new packages)

## Dependencies Added

- `github.com/jackc/pgx/v5` - PostgreSQL driver
- `github.com/golang-migrate/migrate/v4` - Database migrations
