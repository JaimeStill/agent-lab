# Session 3c: Workflow Execution Engine

## Summary

Connected Observer, CheckpointStore, Registry, and Repository to create a functional workflow execution engine with execution, cancellation, and resume capabilities.

## What Was Implemented

### Phase 1: go-agents-orchestration v0.3.0 (pre-session)
- Configuration Merge methods for layered config composition
- `NewGraphWithDeps(cfg, observer, checkpointStore)` for dependency injection
- Thread-safe observer and checkpoint store registries
- `FailFastNil *bool` pattern for boolean fields with true default

### Phase 2: Repository Write Methods
- `CreateRun` - Insert new run with pending status
- `UpdateRunStarted` - Transition to running, set started_at
- `UpdateRunCompleted` - Set final status, result, error_message, completed_at

### Phase 3: Runtime and Factory Signature
- Renamed `systems.go` → `runtime.go`, `Systems` → `Runtime`
- Added `NewRuntime` constructor with getter methods
- Updated `WorkflowFactory` signature: factory now receives pre-configured graph

### Phase 4: System Interface
- New `system.go` with `System` interface defining public API
- Methods: ListRuns, FindRun, GetStages, GetDecisions, ListWorkflows, Execute, Cancel, Resume

### Phase 5: Executor Implementation
- New `executor.go` implementing `System` interface
- Three-phase lifecycle: Cold Start → Hot Start → Post-Commit
- Active run tracking with `map[uuid.UUID]context.CancelFunc`
- Cancellation via context cancellation
- Resume from failed/cancelled runs using checkpoint store

### Phase 6: Domain Integration
- Added `Workflows workflows.System` to Domain struct
- Wired `workflows.NewRuntime` and `workflows.NewSystem` in `NewDomain`

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Runtime naming | `Runtime` instead of `Systems` | Consistent with server's Runtime pattern; "runtime dependencies a system needs" |
| Runtime constructor | `NewRuntime(...)` with private fields | Avoid raw struct composition; stable API |
| Hardcoded GraphConfig | Checkpoint interval=1, preserve=true | Executor owns execution policy for database-backed workflows |
| WorkflowFactory signature | `func(ctx, graph, runtime, params) (State, error)` | Factory receives pre-configured graph; only adds nodes/edges |

## Patterns Established

### Runtime as a Pattern
`Runtime` is the naming convention for "runtime dependencies a system needs" at any level:
- Server Runtime: Database, Storage, Logger, Pagination
- Workflows Runtime: Agents, Documents, Images, Logger

### Three-Phase Executor Lifecycle
1. **Cold Start**: Resolve factory, validate params, create Run (pending)
2. **Hot Start**: Create observer/checkpoint, build graph, execute
3. **Post-Commit**: Update Run with result/error, cleanup tracking

## Files Modified/Created

| File | Action |
|------|--------|
| `internal/workflows/runtime.go` | Renamed from systems.go, added constructor |
| `internal/workflows/registry.go` | Updated WorkflowFactory signature |
| `internal/workflows/system.go` | New - System interface |
| `internal/workflows/executor.go` | New - Executor implementation |
| `internal/workflows/repository.go` | Added write methods |
| `internal/workflows/errors.go` | Added ErrInvalidStatus |
| `cmd/server/domain.go` | Added Workflows to Domain |
| `ARCHITECTURE.md` | Added Runtime as a Pattern section |
| `tests/internal_workflows/*.go` | Added/updated tests |

## Test Coverage

All tests passing (22 workflow tests):
- Runtime constructor and getters
- System interface compliance
- Executor construction and ListWorkflows
- Registry with updated factory signature
- Error types and HTTP mapping

## Dependencies

- go-agents-orchestration v0.3.0 (released during session)
