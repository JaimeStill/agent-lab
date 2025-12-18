# Session 3c: Context Cache

This document captures complete session context for continuity across machines/sessions.

## Session Overview

**Objective**: Connect Observer, CheckpointStore, Registry, and Repository to create a functional workflow execution engine.

**Current Status**: Phase 1 Complete - Awaiting Release

**Next Action**: Release go-agents-orchestration v0.3.0, then proceed to Phase 2

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Observer/CheckpointStore injection | `NewGraphWithDeps(cfg, observer, checkpointStore)` | Cleaner than registry lookup; callers manage their own instances (e.g., per-execution database-backed stores) |
| WorkflowFactory signature | `func(ctx, graph, systems, params) (State, error)` | Factory receives pre-configured graph with observer/checkpoint already wired; factory only adds nodes/edges and returns initial state |
| Registry thread-safety | `sync.RWMutex` on observer and checkpoint store registries | Concurrent safety for multi-workflow execution scenarios |
| Version | v0.3.0 (breaking change) | FailFastNil rename is breaking; requires major version bump per semver |
| Boolean with true default | `FailFastNil *bool` with `FailFast()` accessor | Distinguishes "not set" (nil→true) from "explicitly false" for proper Merge behavior |

---

## Pattern: FailFastNil Convention

### Problem
When unmarshaling JSON into a struct with boolean fields, unspecified fields unmarshal to `false` (zero value). For fields where the default is `true`, this incorrectly overrides the default.

```go
// Config file omits fail_fast entirely
{"max_workers": 4}

// Without *bool: FailFast becomes false (zero value), overriding default true
// With *bool: FailFastNil is nil, accessor returns true (default)
```

### Solution
Use pointer type with accessor method:

```go
type ParallelConfig struct {
    MaxWorkers  int    `json:"max_workers"`
    WorkerCap   int    `json:"worker_cap"`
    FailFastNil *bool  `json:"fail_fast"`  // Pointer enables nil detection
    Observer    string `json:"observer"`
}

// Accessor handles nil-checking and returns effective value
func (c *ParallelConfig) FailFast() bool {
    if c.FailFastNil == nil {
        return true  // default when not specified
    }
    return *c.FailFastNil
}

// Default factory sets pointer to true
func DefaultParallelConfig() ParallelConfig {
    failFast := true
    return ParallelConfig{
        MaxWorkers:  0,
        WorkerCap:   16,
        FailFastNil: &failFast,
        Observer:    "slog",
    }
}

// Merge only copies if source explicitly set (non-nil)
func (c *ParallelConfig) Merge(source *ParallelConfig) {
    // ...
    if source.FailFastNil != nil {
        c.FailFastNil = source.FailFastNil
    }
    // ...
}
```

### Convention
- Field name: `{Name}Nil` (e.g., `FailFastNil`)
- Accessor method: `{Name}()` (e.g., `FailFast()`)
- Only needed for boolean fields with non-false defaults
- Plain `bool` is fine for fields defaulting to `false`

### Documentation
Pattern fully documented in `/home/jaime/code/go-agents-orchestration/pkg/config/doc.go` (lines 80-116).

---

## Pattern: Configuration Merge

### Purpose
Enable layered configuration where loaded configs merge over defaults, preserving zero-values from defaults when source doesn't specify them.

### Semantics by Field Type
| Type | Merge Condition |
|------|-----------------|
| string | Source is non-empty |
| int | Source is greater than zero |
| time.Duration | Source is greater than zero |
| pointer | Source is non-nil |
| nested config | Recursive merge |
| bool (false default) | Source is true |
| *bool (true default) | Source is non-nil |

### Example Usage
```go
cfg := config.DefaultGraphConfig("workflow")
var loaded config.GraphConfig
json.Unmarshal(data, &loaded)
cfg.Merge(&loaded)  // loaded values override defaults where specified
```

---

## Phase 1: go-agents-orchestration v0.3.0

### Status: COMPLETE (awaiting release)

### Validation Results
- `go vet ./...` - PASSES
- `go test ./...` - PASSES (all 28 tests)

### Files Modified

#### pkg/config/state.go
Added Merge methods:
```go
func (c *CheckpointConfig) Merge(source *CheckpointConfig)
func (c *GraphConfig) Merge(source *GraphConfig)
```

#### pkg/config/hub.go
Added Merge method:
```go
func (c *HubConfig) Merge(source *HubConfig)
```

#### pkg/config/workflows.go
- Changed `FailFast bool` to `FailFastNil *bool`
- Added `FailFast()` accessor method
- Updated `DefaultParallelConfig()` to use pointer
- Added Merge methods:
```go
func (c *ChainConfig) Merge(source *ChainConfig)
func (c *ParallelConfig) Merge(source *ParallelConfig)
func (c *ConditionalConfig) Merge(source *ConditionalConfig)
```

#### pkg/config/doc.go
Added documentation for:
- Configuration Merging section (lines 62-79)
- Boolean Fields with Non-False Defaults section (lines 80-116)

#### pkg/state/graph.go
Added new constructor:
```go
func NewGraphWithDeps(cfg config.GraphConfig, observer observability.Observer, checkpointStore CheckpointStore) (StateGraph, error) {
    if observer == nil {
        observer = observability.NoOpObserver{}
    }

    return &stateGraph{
        name:                cfg.Name,
        nodes:               make(map[string]StateNode),
        edges:               make(map[string][]Edge),
        exitPoints:          make(map[string]bool),
        maxIterations:       cfg.MaxIterations,
        observer:            observer,
        checkpointStore:     checkpointStore,
        checkpointInterval:  cfg.Checkpoint.Interval,
        preserveCheckpoints: cfg.Checkpoint.Preserve,
    }, nil
}
```

#### pkg/state/checkpoint.go
Added thread-safe registry:
```go
var (
    checkpointStores = map[string]CheckpointStore{
        "memory": NewMemoryCheckpointStore(),
    }
    mutex sync.RWMutex
)

func GetCheckpointStore(name string) (CheckpointStore, error) {
    mutex.RLock()
    defer mutex.RUnlock()
    // ...
}

func RegisterCheckpointStore(name string, store CheckpointStore) {
    mutex.Lock()
    defer mutex.Unlock()
    // ...
}
```

#### pkg/observability/registry.go
Added thread-safe registry (same pattern as checkpoint store):
```go
var (
    observers = map[string]Observer{...}
    mutex sync.RWMutex
)
```

#### pkg/workflows/parallel.go
Updated all usages of `cfg.FailFast` to `cfg.FailFast()`:
- Line 172: `"fail_fast": cfg.FailFast(),`
- Line 203: `"fail_fast": cfg.FailFast(),`
- Line 223: `if cfg.FailFast() {`
- Line 248: `cfg.FailFast(),`
- Line 298: `if cfg.FailFast() || len(results) == 0 {`

#### pkg/workflows/doc.go
Updated examples to show FailFastNil pattern (lines 56-70).

#### tests/workflows/parallel_test.go
Updated 6 test functions to use FailFastNil:
- `TestProcessParallel_CollectAllErrors`
- `TestProcessParallel_CollectAllErrors_AllFail`
- `TestProcessParallel_MixedResults`
- `TestProcessParallel_ContextCancellation`
- `TestProcessParallel_EmptyItems`
- `TestProcessParallel_SingleItem`

Pattern used:
```go
failFast := false
cfg := config.ParallelConfig{
    MaxWorkers:  4,
    WorkerCap:   16,
    FailFastNil: &failFast,
    Observer:    "noop",
}
```

#### examples/phase-05-parallel-execution/main.go
Updated to use `cfg.FailFast()` accessor.

#### examples/darpa-procurement/workflow.go
Updated 2 usages to use FailFastNil pattern.

#### CHANGELOG.md
Already updated by user with v0.3.0 entry documenting:
- Breaking: FailFastNil rename with accessor method
- Added: Merge methods for configuration composition
- Added: NewGraphWithDeps for dependency injection
- Added: Thread-safe checkpoint store registry
- Added: Thread-safe observer registry

### Release Commands
```bash
cd /home/jaime/code/go-agents-orchestration
git add -A
git commit -m "v0.3.0: NewGraphWithDeps, thread-safe registries, config Merge methods"
git tag v0.3.0
git push origin main --tags
```

---

## Phase 2: Repository Write Methods

### File: `/home/jaime/code/agent-lab/internal/workflows/repository.go`

### Methods to Add

#### CreateRun
- Insert new run with status=pending
- Marshal params to JSON
- Use `repository.WithTx()` for transaction safety
- RETURNING full row via `scanRun`

#### UpdateRunStarted
- Set status=running, started_at=NOW(), updated_at=NOW()
- Use `repository.MapError()` for ErrNotFound handling

#### UpdateRunCompleted
- Set status, result (JSON), error_message, completed_at, updated_at
- Marshal result to JSON if non-nil

### Implementation Details
See implementation guide section "Phase 2: Repository Write Methods" for complete code.

---

## Phase 3: Update WorkflowFactory Signature

### File: `/home/jaime/code/agent-lab/internal/workflows/registry.go`

### Change
```go
// Before
type WorkflowFactory func(ctx context.Context, systems *Systems, params map[string]any) (state.StateGraph, state.State, error)

// After
type WorkflowFactory func(ctx context.Context, graph state.StateGraph, systems *Systems, params map[string]any) (state.State, error)
```

### Rationale
Factory no longer creates the graph - it receives a pre-configured graph with observer and checkpoint store already injected. Factory's responsibility is now:
1. Add nodes to graph
2. Add edges to graph
3. Set entry point and exit points
4. Return initial state

---

## Phase 4: System Interface

### File: `/home/jaime/code/agent-lab/internal/workflows/system.go` (new)

### Interface Definition
```go
type System interface {
    // Read operations (delegated to repository)
    ListRuns(ctx context.Context, page pagination.PageRequest, filters RunFilters) (*pagination.PageResult[Run], error)
    FindRun(ctx context.Context, id uuid.UUID) (*Run, error)
    GetStages(ctx context.Context, runID uuid.UUID) ([]Stage, error)
    GetDecisions(ctx context.Context, runID uuid.UUID) ([]Decision, error)

    // Registry operations
    ListWorkflows() []WorkflowInfo

    // Execution operations
    Execute(ctx context.Context, name string, params map[string]any) (*Run, error)
    Cancel(ctx context.Context, runID uuid.UUID) error
    Resume(ctx context.Context, runID uuid.UUID) (*Run, error)
}
```

---

## Phase 5: Executor Implementation

### File: `/home/jaime/code/agent-lab/internal/workflows/executor.go` (new)

### Struct
```go
type executor struct {
    repo         *repo
    systems      *Systems
    db           *sql.DB
    logger       *slog.Logger
    activeRuns   map[uuid.UUID]context.CancelFunc
    activeRunsMu sync.RWMutex
}
```

### Three-Phase Lifecycle

#### 1. Cold Start
- Resolve factory from registry
- Validate params
- Create Run record (status=pending)

#### 2. Hot Start
- Create PostgresObserver with run ID
- Create PostgresCheckpointStore
- Build graph with NewGraphWithDeps
- Call factory to add nodes/edges
- Set RunID on initial state
- Execute graph

#### 3. Post-Commit
- Update Run with result/error
- Remove from active runs tracking
- Clean up resources

### Key Methods

#### Execute(ctx, name, params)
Full execution flow for new workflow run.

#### Cancel(ctx, runID)
- Lookup in activeRuns map
- Call cancel function to trigger context cancellation
- Workers will exit on ctx.Done()

#### Resume(ctx, runID)
- Load run from database
- Validate status is failed/cancelled
- Unmarshal params from run record
- Create observer/checkpoint store
- Build graph, call factory
- Call `graph.Resume(ctx, runID)` instead of Execute

### Helper Methods
- `trackRun(id, cancel)` - Add to activeRuns map (mutex protected)
- `untrackRun(id)` - Remove from activeRuns map (mutex protected)
- `finalizeRun(ctx, id, status, result, err)` - Update run and return

### Integration with Session 3b
Uses `PostgresObserver` and `PostgresCheckpointStore` from session 3b:
- `NewPostgresObserver(db, runID, logger)` - Writes stages/decisions to database
- `NewPostgresCheckpointStore(db, logger)` - Persists state checkpoints

---

## Phase 6: Domain Integration

### File: `/home/jaime/code/agent-lab/cmd/server/domain.go`

### Changes

#### Update Domain struct
```go
type Domain struct {
    Providers providers.System
    Agents    agents.System
    Documents documents.System
    Images    images.System
    Workflows workflows.System  // Add this
}
```

#### Update NewDomain function
Create Systems struct for workflow access to other domains:
```go
systems := &workflows.Systems{
    Agents:    agentsSys,
    Documents: docs,
    Images:    imagesSys,
    Logger:    runtime.Logger,
}
```

Wire workflows.NewSystem:
```go
Workflows: workflows.NewSystem(
    runtime.Database.Connection(),
    systems,
    runtime.Logger,
    runtime.Pagination,
),
```

### Systems Struct
Already defined in `/home/jaime/code/agent-lab/internal/workflows/registry.go`:
```go
type Systems struct {
    Agents    agents.System
    Documents documents.System
    Images    images.System
    Logger    *slog.Logger
}
```

---

## Phase 7: Update go.mod

### Commands
```bash
cd /home/jaime/code/agent-lab
go get github.com/JaimeStill/go-agents-orchestration@v0.3.0
go mod tidy
```

---

## Error Types

### Existing (from session 3a)
Located in `/home/jaime/code/agent-lab/internal/workflows/errors.go`:
- `ErrNotFound` - Run not found
- `ErrWorkflowNotFound` - Workflow not registered

### May Need to Add
- `ErrInvalidStatus` - For Cancel/Resume on wrong status

---

## Validation Strategy

After all phases complete:

```bash
cd /home/jaime/code/agent-lab
go vet ./...
go test ./tests/...
```

AI will then:
1. Review implementation for accuracy
2. Add/revise tests for new functionality
3. Execute tests until passing
4. Add godoc comments to exported types/functions

---

## Reference Documents

| Document | Location | Purpose |
|----------|----------|---------|
| Implementation Guide | `_context/3c-workflow-execution-engine.md` | Complete step-by-step implementation |
| Milestone Architecture | `_context/milestones/m03-workflow-execution.md` | Cross-session technical context |
| Session 3b Summary | `_context/sessions/3b-observer-and-checkpoint-store.md` | Previous session deliverables |
| Plan Mode Outline | `.claude/plans/snug-puzzling-babbage.md` | Original plan approval |
| go-agents-orchestration docs | `pkg/config/doc.go`, `pkg/workflows/doc.go` | Pattern documentation |

---

## Session Workflow Progress

| Step | Status | Notes |
|------|--------|-------|
| Planning Phase | Complete | Aligned on NewGraphWithDeps, factory signature, thread-safety |
| Plan Presentation | Complete | Approved in plan mode |
| Implementation Guide Creation | Complete | Full guide at `_context/3c-workflow-execution-engine.md` |
| Phase 1: go-agents-orchestration | Complete | All changes made, tests pass, awaiting release |
| Phase 2: Repository writes | Pending | CreateRun, UpdateRunStarted, UpdateRunCompleted |
| Phase 3: Factory signature | Pending | Update WorkflowFactory type |
| Phase 4: System interface | Pending | New file: system.go |
| Phase 5: Executor | Pending | New file: executor.go |
| Phase 6: Domain wiring | Pending | Update domain.go |
| Phase 7: go.mod update | Pending | After v0.3.0 release |
| Validation Phase | Pending | After all implementation phases |
| Documentation Phase | Pending | Godoc comments |
| Session Closeout | Pending | Summary, archive, doc updates |

---

## Important Notes

1. **User implements, AI validates** - Following CLAUDE.md workflow, user executes implementation from guide, AI validates afterward

2. **CHANGELOG already updated** - User updated go-agents-orchestration CHANGELOG.md during Phase 1

3. **FailFastNil is breaking** - Any code using `cfg.FailFast` directly must change to `cfg.FailFast()`

4. **PostgresObserver/CheckpointStore exist** - Created in session 3b, ready for use in executor

5. **Systems struct exists** - Already defined in registry.go from session 3a

6. **ErrInvalidStatus may be needed** - Check if it exists in errors.go; add if missing for Cancel/Resume validation
