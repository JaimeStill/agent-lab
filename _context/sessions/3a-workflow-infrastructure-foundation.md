# Session 3a: Workflow Infrastructure Foundation

**Status**: Completed
**Date**: 2025-12-17

## Objective

Establish database schema, core types, registry, and read-only repository for the workflows domain.

## Implemented

### Database Schema
- Migration `000006_workflows` with 4 tables:
  - `runs` - Workflow execution records
  - `stages` - Per-node execution via Observer events
  - `decisions` - Routing decisions with predicate results
  - `checkpoints` - State persistence for resume capability

### Core Types (`internal/workflows/run.go`)
- `Run` - Execution record with status, params, result, timestamps
- `Stage` - Node execution with input/output snapshots, duration, iteration
- `Decision` - Routing decision with from/to nodes, predicate info, reason
- `WorkflowInfo` - Registry metadata (name, description)
- Status constants: `StatusPending`, `StatusRunning`, `StatusCompleted`, `StatusFailed`, `StatusCancelled`
- Stage status constants: `StageStarted`, `StageCompleted`, `StageFailed`

### Domain Errors (`internal/workflows/errors.go`)
- `ErrNotFound` - Run/stage/decision not found
- `ErrWorkflowNotFound` - Workflow not registered
- `ErrInvalidStatus` - Invalid status transition
- `MapHTTPStatus(error) int` - HTTP status code mapping

### Mapping Infrastructure (`internal/workflows/mapping.go`)
- Projections for runs, stages, decisions
- Scanners for all types
- `RunFilters` with `FiltersFromQuery` and `Apply` methods
- Default sort fields per entity type

### Global Registry (`internal/workflows/registry.go`)
- `WorkflowFactory` type for creating StateGraph and State
- Thread-safe registry with `sync.RWMutex`
- `Register(name, factory, description)` - Register workflow factories
- `Get(name)` - Retrieve factory by name
- `List()` - Return all registered workflow info

### Systems Struct (`internal/workflows/systems.go`)
- Placeholder for Session 3c
- Provides access to domain systems (Agents, Documents, Images) for workflow execution

### Read-Only Repository (`internal/workflows/repository.go`)
- `New(db, logger, pagination)` - Constructor
- `ListRuns(ctx, page, filters)` - Paginated run listing
- `FindRun(ctx, id)` - Single run by ID
- `GetStages(ctx, runID)` - All stages for a run
- `GetDecisions(ctx, runID)` - All decisions for a run

### Query Builder Extension (`pkg/query/builder.go`)
- Added `Build()` method for unbounded SELECT queries
- Supports GetStages/GetDecisions which need all related records without pagination

## Key Decisions

### Repository Query Method Naming Convention
Established and documented naming convention for repository methods:
| Verb | Returns | Use Case |
|------|---------|----------|
| `List` | `*PageResult[T]` | Browsing/searching collections (paginated) |
| `Find` | `*T` | Locate single item by ID |
| `Get` | `[]T` | Retrieve all related items (bounded, full slice) |

This convention is documented in ARCHITECTURE.md.

### JSON Tag Convention
All JSON tags use snake_case (e.g., `json:"workflow_name"`), consistent with the rest of the codebase.

### Projection Field Names
Use "ID" (uppercase) for projection mappings, consistent with established pattern.

## Validation

- Migration runs successfully
- All types compile with proper JSON tags
- `go vet ./...` passes
- Tests created and passing:
  - `tests/pkg_query/builder_test.go` - Build() method tests
  - `tests/internal_workflows/` - Registry, errors, mapping, run type tests

## Files Created/Modified

| File | Purpose |
|------|---------|
| `cmd/migrate/migrations/000006_workflows.up.sql` | Create workflow tables |
| `cmd/migrate/migrations/000006_workflows.down.sql` | Drop workflow tables |
| `internal/workflows/run.go` | Core types and status constants |
| `internal/workflows/errors.go` | Domain errors |
| `internal/workflows/mapping.go` | Projections, scanners, filters |
| `internal/workflows/registry.go` | Global workflow registry |
| `internal/workflows/systems.go` | Systems struct placeholder |
| `internal/workflows/repository.go` | Read-only repository |
| `pkg/query/builder.go` | Added Build() method |
| `tests/pkg_query/builder_test.go` | Build() method tests |
| `tests/internal_workflows/registry_test.go` | Registry tests |
| `tests/internal_workflows/errors_test.go` | Error tests |
| `tests/internal_workflows/mapping_test.go` | Mapping tests |
| `tests/internal_workflows/run_test.go` | Run type tests |

## Session Scope Boundaries

### In Scope (Completed)
- Database schema (4 tables)
- Core types (Run, Stage, Decision, WorkflowInfo)
- Domain errors
- Mapping infrastructure (projections, scanners, filters)
- Global registry (Register, Get, List)
- Systems struct placeholder
- Read-only repository (ListRuns, FindRun, GetStages, GetDecisions)

### Out of Scope (Deferred)
- Write operations in repository (3c - part of command model)
- Observer implementation (3b)
- Checkpoint store implementation (3b)
- Executor (3c)
- Handler and API endpoints (3d)
- OpenAPI specification (3d)
