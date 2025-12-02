# Session 1d: Domain Infrastructure Patterns

**Status**: Complete
**Milestone**: 01 - Foundation & Infrastructure
**Date**: 2025-12-02

## Summary

Established reusable infrastructure patterns that eliminate repetitive boilerplate across domain systems, then refactored providers to use the new infrastructure.

## Deliverables

### New Packages

**pkg/repository** - Database helper functions:
- `Querier`, `Executor`, `Scanner` interfaces for database abstraction
- `ScanFunc[T]` generic type for domain-specific row scanning
- `WithTx[T]` - Transaction wrapper with automatic Begin/Commit/Rollback
- `QueryOne[T]`, `QueryMany[T]` - Generic query executors
- `ExecExpectOne` - Execute statements expecting exactly one affected row
- `MapError` - Domain-agnostic error mapping (sql.ErrNoRows → notFoundErr, pg 23505 → duplicateErr)

**pkg/handlers** - HTTP response utilities:
- `RespondJSON` - Write JSON response with status
- `RespondError` - Log error and write JSON error response

### Enhanced Packages

**pkg/query** - Multi-column sorting:
- `SortField` struct with `Field` and `Descending`
- `ParseSortFields(s string)` - Parse "name,-createdAt" format
- `OrderByFields([]SortField)` - Set multi-column sort order
- `NewBuilder` now takes variadic `...SortField` for default sort
- Removed old single-column sorting (`orderBy`, `descending`, `defaultSort` fields)

**pkg/pagination** - Query parameter parsing:
- `PageRequest.Sort` field replaces `SortBy`/`Descending`
- `PageRequestFromQuery(url.Values, Config)` - Parse GET parameters

### Domain Patterns

**internal/providers/filters.go** - Domain filter pattern:
- `Filters` struct with domain-specific filter fields
- `FiltersFromQuery(url.Values)` - Parse filters from query params
- `Apply(*query.Builder)` - Apply filters to query builder

**internal/providers/scanner.go** - Domain scanner:
- `scanProvider(repository.Scanner)` - Convert row to Provider

**internal/providers/handler.go** - Handler struct pattern:
- Replaces closure-based route wiring with struct methods
- `NewHandler(System, *slog.Logger, pagination.Config)`
- `Routes() routes.Group` - Self-contained route registration
- Individual handler methods: `Create`, `Update`, `Delete`, `GetByID`, `List`, `Search`

**internal/providers/errors.go** - HTTP status mapping:
- `MapHTTPStatus(error) int` - Map domain errors to HTTP status codes

### New Endpoint

`GET /api/providers` - List providers with query parameters:
- `page`, `page_size` - Pagination
- `search` - Full-text search
- `sort` - Multi-column sorting (e.g., `name,-createdAt`)
- `name` - Filter by name (contains)

## Key Decisions

1. **Clean break for sorting**: Removed backward compatibility for old `SortBy`/`Descending` fields in favor of unified `Sort []SortField`

2. **Domain-agnostic error mapping**: `MapError` takes domain errors as parameters rather than knowing about specific domains

3. **Stateless handler utilities**: `pkg/handlers` provides functions, not a base struct

4. **Handler struct pattern**: Domain handlers own their routes and dependencies, improving encapsulation

5. **Scanner in domain package**: Row scanning stays in the domain package where entity structure is defined

## Files Changed

| Action | File |
|--------|------|
| NEW | `pkg/repository/repository.go` |
| NEW | `pkg/repository/errors.go` |
| NEW | `pkg/handlers/handlers.go` |
| NEW | `internal/providers/handler.go` |
| NEW | `internal/providers/scanner.go` |
| NEW | `internal/providers/filters.go` |
| MODIFIED | `pkg/query/builder.go` |
| MODIFIED | `pkg/pagination/pagination.go` |
| MODIFIED | `internal/providers/repository.go` |
| MODIFIED | `internal/providers/system.go` |
| MODIFIED | `internal/providers/errors.go` |
| MODIFIED | `cmd/server/routes.go` |
| DELETED | `internal/providers/handlers.go` |
| DELETED | `internal/providers/routes.go` |

## Test Coverage

- pkg/repository: `MapError` 100%, transaction helpers integration-tested via API
- pkg/handlers: 100%
- pkg/query: 98.9%
- pkg/pagination: Tests added for `PageRequestFromQuery`
- Overall: 78.5%

## Patterns Established

1. **Repository Helpers**: Generic transaction and query execution
2. **Domain Scanner**: `ScanFunc[T]` defined in domain packages
3. **Handler Struct**: Self-contained handler with `Routes()` method
4. **Domain Filters**: Filter struct with `FiltersFromQuery` and `Apply`
5. **HTTP Status Mapping**: Domain error → HTTP status in errors.go
