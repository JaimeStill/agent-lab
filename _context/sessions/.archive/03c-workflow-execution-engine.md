# Session 3c: Workflow Execution Engine

## Problem Context

Session 3a created the workflow infrastructure (schema, types, repository reads, registry). Session 3b implemented the PostgresObserver and PostgresCheckpointStore. This session connects these components to create a functional execution engine that can:

1. Execute registered workflows with full observability
2. Persist execution state for resume capability
3. Support cancellation of running workflows
4. Track active runs for management

## Architecture Approach

The Executor orchestrates workflow execution through a three-phase lifecycle:

1. **Cold Start**: Resolve workflow factory, validate params, create Run record (pending)
2. **Hot Start**: Create observer/checkpoint instances, build graph, execute
3. **Post-Commit**: Update Run with result/error, clean up tracking

The Executor implements the System interface, which combines repository reads with execution operations.

## Implementation Phases

### Phase 1: go-agents-orchestration v0.3.0

#### 1.1 Add Merge methods to configuration types

Following the go-agents pattern, add Merge methods for configuration composition.

**File**: `/home/jaime/code/go-agents-orchestration/pkg/config/state.go`

Add after `DefaultCheckpointConfig`:

```go
func (c *CheckpointConfig) Merge(source *CheckpointConfig) {
	if source.Store != "" {
		c.Store = source.Store
	}

	if source.Interval > 0 {
		c.Interval = source.Interval
	}

	if source.Preserve {
		c.Preserve = source.Preserve
	}
}
```

Add after `DefaultGraphConfig`:

```go
func (c *GraphConfig) Merge(source *GraphConfig) {
	if source.Name != "" {
		c.Name = source.Name
	}

	if source.Observer != "" {
		c.Observer = source.Observer
	}

	if source.MaxIterations > 0 {
		c.MaxIterations = source.MaxIterations
	}

	c.Checkpoint.Merge(&source.Checkpoint)
}
```

**File**: `/home/jaime/code/go-agents-orchestration/pkg/config/hub.go`

Add after `DefaultHubConfig`:

```go
func (c *HubConfig) Merge(source *HubConfig) {
	if source.Name != "" {
		c.Name = source.Name
	}

	if source.ChannelBufferSize > 0 {
		c.ChannelBufferSize = source.ChannelBufferSize
	}

	if source.DefaultTimeout > 0 {
		c.DefaultTimeout = source.DefaultTimeout
	}

	if source.Logger != nil {
		c.Logger = source.Logger
	}
}
```

**File**: `/home/jaime/code/go-agents-orchestration/pkg/config/workflows.go`

Add after `DefaultChainConfig`:

```go
func (c *ChainConfig) Merge(source *ChainConfig) {
	if source.CaptureIntermediateStates {
		c.CaptureIntermediateStates = source.CaptureIntermediateStates
	}

	if source.Observer != "" {
		c.Observer = source.Observer
	}
}
```

Update `ParallelConfig` struct to use pointer for FailFast with accessor method (enables distinguishing "not set" from "explicitly false"):

```go
type ParallelConfig struct {
	MaxWorkers  int    `json:"max_workers"`
	WorkerCap   int    `json:"worker_cap"`
	FailFastNil *bool  `json:"fail_fast"`
	Observer    string `json:"observer"`
}

func (c *ParallelConfig) FailFast() bool {
	if c.FailFastNil == nil {
		return true
	}
	return *c.FailFastNil
}
```

Update `DefaultParallelConfig` to set pointer value:

```go
func DefaultParallelConfig() ParallelConfig {
	failFast := true
	return ParallelConfig{
		MaxWorkers:  0,
		WorkerCap:   16,
		FailFastNil: &failFast,
		Observer:    "slog",
	}
}
```

Add after `DefaultParallelConfig`:

```go
func (c *ParallelConfig) Merge(source *ParallelConfig) {
	if source.MaxWorkers > 0 {
		c.MaxWorkers = source.MaxWorkers
	}

	if source.WorkerCap > 0 {
		c.WorkerCap = source.WorkerCap
	}

	if source.FailFastNil != nil {
		c.FailFastNil = source.FailFastNil
	}

	if source.Observer != "" {
		c.Observer = source.Observer
	}
}
```

Add after `DefaultConditionalConfig`:

```go
func (c *ConditionalConfig) Merge(source *ConditionalConfig) {
	if source.Observer != "" {
		c.Observer = source.Observer
	}
}
```

#### 1.2 Add mutex to checkpoint store registry

**File**: `/home/jaime/code/go-agents-orchestration/pkg/state/checkpoint.go`

Update the registry declaration to use mutex:

```go
var (
	checkpointStores = map[string]CheckpointStore{
		"memory": NewMemoryCheckpointStore(),
	}
	mutex sync.RWMutex
)
```

Update `GetCheckpointStore`:

```go
func GetCheckpointStore(name string) (CheckpointStore, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	store, exists := checkpointStores[name]
	if !exists {
		return nil, fmt.Errorf("unknown checkpoint store: %s", name)
	}
	return store, nil
}
```

Update `RegisterCheckpointStore`:

```go
func RegisterCheckpointStore(name string, store CheckpointStore) {
	mutex.Lock()
	defer mutex.Unlock()
	checkpointStores[name] = store
}
```

#### 1.3 Add mutex to observer registry

**File**: `/home/jaime/code/go-agents-orchestration/pkg/observability/registry.go`

Add import for `sync` package.

Update the registry declaration:

```go
var (
	observers = map[string]Observer{
		"noop": NoOpObserver{},
		"slog": NewSlogObserver(slog.Default()),
	}
	mutex sync.RWMutex
)
```

Update `GetObserver`:

```go
func GetObserver(name string) (Observer, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	obs, exists := observers[name]
	if !exists {
		return nil, fmt.Errorf("unknown observer: %s", name)
	}
	return obs, nil
}
```

Update `RegisterObserver`:

```go
func RegisterObserver(name string, observer Observer) {
	mutex.Lock()
	defer mutex.Unlock()
	observers[name] = observer
}
```

#### 1.4 Add NewGraphWithDeps

**File**: `/home/jaime/code/go-agents-orchestration/pkg/state/graph.go`

Add after `NewGraph` function:

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

#### 1.5 Update CHANGELOG.md

**File**: `/home/jaime/code/go-agents-orchestration/CHANGELOG.md`

Add after the title line:

```markdown
## [v0.3.0] - 2025-12-18

**Breaking**:

- `pkg/config` - ParallelConfig.FailFast field renamed to FailFastNil with accessor method

  The `FailFast` field is now `FailFastNil *bool` with a `FailFast()` method that handles nil-checking and returns the effective boolean value (defaulting to true when nil). This enables distinguishing between "not set" (nil, uses default true) and "explicitly set to false", which is required for proper Merge behavior where unspecified fields preserve defaults.

**Added**:

- `pkg/config` - Merge methods for configuration composition

  Added `Merge(*Config)` methods to all configuration types following the go-agents pattern. Enables layered configuration where loaded configs merge over defaults, with zero-values preserved from defaults.

- `pkg/state` - NewGraphWithDeps for dependency injection

  Added `NewGraphWithDeps(cfg, observer, checkpointStore)` constructor that accepts Observer and CheckpointStore instances directly instead of resolving from global registries. Enables cleaner integration where callers manage their own instances (e.g., per-execution database-backed stores).

- `pkg/state` - Thread-safe checkpoint store registry

  Added `sync.RWMutex` protection to the checkpoint store registry. `GetCheckpointStore` and `RegisterCheckpointStore` are now safe for concurrent use.

- `pkg/observability` - Thread-safe observer registry

  Added `sync.RWMutex` protection to the observer registry. `GetObserver` and `RegisterObserver` are now safe for concurrent use.

```

#### 1.6 Validation and Release

```bash
cd /home/jaime/code/go-agents-orchestration
go vet ./...
go test ./...
git add -A
git commit -m "v0.3.0: NewGraphWithDeps, thread-safe registries, config Merge methods"
git tag v0.3.0
git push origin main --tags
```

---

### Phase 2: Repository Write Methods

**File**: `/home/jaime/code/agent-lab/internal/workflows/repository.go`

Add import for `encoding/json`.

#### 2.1 CreateRun

```go
func (r *repo) CreateRun(ctx context.Context, workflowName string, params map[string]any) (*Run, error) {
	var paramsJSON json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshal params: %w", err)
		}
		paramsJSON = data
	}

	const q = `
		INSERT INTO runs (workflow_name, status, params)
		VALUES ($1, $2, $3)
		RETURNING id, workflow_name, status, params, result, error_message,
		          started_at, completed_at, created_at, updated_at
	`

	run, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Run, error) {
		return repository.QueryOne(ctx, tx, q, []any{
			workflowName, StatusPending, paramsJSON,
		}, scanRun)
	})

	if err != nil {
		return nil, fmt.Errorf("create run: %w", err)
	}

	return &run, nil
}
```

#### 2.2 UpdateRunStarted

```go
func (r *repo) UpdateRunStarted(ctx context.Context, id uuid.UUID) (*Run, error) {
	const q = `
		UPDATE runs
		SET status = $1, started_at = NOW(), updated_at = NOW()
		WHERE id = $2
		RETURNING id, workflow_name, status, params, result, error_message,
		          started_at, completed_at, created_at, updated_at
	`

	run, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Run, error) {
		return repository.QueryOne(ctx, tx, q, []any{StatusRunning, id}, scanRun)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, nil)
	}

	return &run, nil
}
```

#### 2.3 UpdateRunCompleted

```go
func (r *repo) UpdateRunCompleted(ctx context.Context, id uuid.UUID, status RunStatus, result map[string]any, errorMsg *string) (*Run, error) {
	var resultJSON json.RawMessage
	if result != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("marshal result: %w", err)
		}
		resultJSON = data
	}

	const q = `
		UPDATE runs
		SET status = $1, result = $2, error_message = $3,
		    completed_at = NOW(), updated_at = NOW()
		WHERE id = $4
		RETURNING id, workflow_name, status, params, result, error_message,
		          started_at, completed_at, created_at, updated_at
	`

	run, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Run, error) {
		return repository.QueryOne(ctx, tx, q, []any{
			status, resultJSON, errorMsg, id,
		}, scanRun)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, nil)
	}

	return &run, nil
}
```

---

### Phase 3: Update WorkflowFactory Signature

#### 3.1 Rename Systems to Runtime

**File**: `/home/jaime/code/agent-lab/internal/workflows/systems.go` → `runtime.go`

Rename the file and update the struct name:

```go
package workflows

import (
	"log/slog"

	"github.com/JaimeStill/agent-lab/internal/agents"
	"github.com/JaimeStill/agent-lab/internal/documents"
	"github.com/JaimeStill/agent-lab/internal/images"
)

type Runtime struct {
	agents    agents.System
	documents documents.System
	images    images.System
	logger    *slog.Logger
}

func NewRuntime(
	agents agents.System,
	documents documents.System,
	images images.System,
	logger *slog.Logger,
) *Runtime {
	return &Runtime{
		agents:    agents,
		documents: documents,
		images:    images,
		logger:    logger,
	}
}

func (r *Runtime) Agents() agents.System    { return r.agents }
func (r *Runtime) Documents() documents.System { return r.documents }
func (r *Runtime) Images() images.System    { return r.images }
func (r *Runtime) Logger() *slog.Logger     { return r.logger }
```

#### 3.2 Update Factory Signature

**File**: `/home/jaime/code/agent-lab/internal/workflows/registry.go`

Update the factory type to receive a pre-configured graph:

```go
type WorkflowFactory func(ctx context.Context, graph state.StateGraph, runtime *Runtime, params map[string]any) (state.State, error)
```

Update `Register` function signature:

```go
func Register(name string, factory WorkflowFactory, description string) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.factories[name] = factory
	registry.info[name] = WorkflowInfo{Name: name, Description: description}
}
```

---

### Phase 4: System Interface

**File**: `/home/jaime/code/agent-lab/internal/workflows/system.go` (new file)

```go
package workflows

import (
	"context"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

type System interface {
	ListRuns(ctx context.Context, page pagination.PageRequest, filters RunFilters) (*pagination.PageResult[Run], error)
	FindRun(ctx context.Context, id uuid.UUID) (*Run, error)
	GetStages(ctx context.Context, runID uuid.UUID) ([]Stage, error)
	GetDecisions(ctx context.Context, runID uuid.UUID) ([]Decision, error)
	ListWorkflows() []WorkflowInfo
	Execute(ctx context.Context, name string, params map[string]any) (*Run, error)
	Cancel(ctx context.Context, runID uuid.UUID) error
	Resume(ctx context.Context, runID uuid.UUID) (*Run, error)
}
```

---

### Phase 5: Executor Implementation

**File**: `/home/jaime/code/agent-lab/internal/workflows/executor.go` (new file)

```go
package workflows

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/go-agents-orchestration/pkg/config"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
	"github.com/google/uuid"
)

type executor struct {
	repo         *repo
	runtime      *Runtime
	db           *sql.DB
	logger       *slog.Logger
	activeRuns   map[uuid.UUID]context.CancelFunc
	activeRunsMu sync.RWMutex
}

func NewSystem(
	db *sql.DB,
	runtime *Runtime,
	logger *slog.Logger,
	pagination pagination.Config,
) System {
	return &executor{
		repo:       New(db, logger, pagination),
		runtime:    runtime,
		db:         db,
		logger:     logger.With("system", "workflows"),
		activeRuns: make(map[uuid.UUID]context.CancelFunc),
	}
}

func (e *executor) ListRuns(ctx context.Context, page pagination.PageRequest, filters RunFilters) (*pagination.PageResult[Run], error) {
	return e.repo.ListRuns(ctx, page, filters)
}

func (e *executor) FindRun(ctx context.Context, id uuid.UUID) (*Run, error) {
	return e.repo.FindRun(ctx, id)
}

func (e *executor) GetStages(ctx context.Context, runID uuid.UUID) ([]Stage, error) {
	return e.repo.GetStages(ctx, runID)
}

func (e *executor) GetDecisions(ctx context.Context, runID uuid.UUID) ([]Decision, error) {
	return e.repo.GetDecisions(ctx, runID)
}

func (e *executor) ListWorkflows() []WorkflowInfo {
	return List()
}

func (e *executor) Execute(ctx context.Context, name string, params map[string]any) (*Run, error) {
	factory, exists := Get(name)
	if !exists {
		return nil, ErrWorkflowNotFound
	}

	run, err := e.repo.CreateRun(ctx, name, params)
	if err != nil {
		return nil, fmt.Errorf("create run: %w", err)
	}

	execCtx, cancel := context.WithCancel(ctx)
	e.trackRun(run.ID, cancel)
	defer e.untrackRun(run.ID)

	run, err = e.repo.UpdateRunStarted(execCtx, run.ID)
	if err != nil {
		return e.finalizeRun(ctx, run.ID, StatusFailed, nil, err)
	}

	observer := NewPostgresObserver(e.db, run.ID, e.logger)
	checkpointStore := NewPostgresCheckpointStore(e.db, e.logger)

	cfg := config.DefaultGraphConfig(name)
	cfg.Checkpoint.Interval = 1
	cfg.Checkpoint.Preserve = true

	graph, err := state.NewGraphWithDeps(cfg, observer, checkpointStore)
	if err != nil {
		return e.finalizeRun(ctx, run.ID, StatusFailed, nil, err)
	}

	initialState, err := factory(execCtx, graph, e.runtime, params)
	if err != nil {
		return e.finalizeRun(ctx, run.ID, StatusFailed, nil, err)
	}

	initialState.RunID = run.ID.String()

	finalState, err := graph.Execute(execCtx, initialState)

	if err != nil {
		if execCtx.Err() != nil {
			errMsg := "execution cancelled"
			return e.repo.UpdateRunCompleted(ctx, run.ID, StatusCancelled, nil, &errMsg)
		}
		errMsg := err.Error()
		return e.repo.UpdateRunCompleted(ctx, run.ID, StatusFailed, nil, &errMsg)
	}

	return e.repo.UpdateRunCompleted(ctx, run.ID, StatusCompleted, finalState.Data, nil)
}

func (e *executor) Cancel(ctx context.Context, runID uuid.UUID) error {
	e.activeRunsMu.RLock()
	cancel, exists := e.activeRuns[runID]
	e.activeRunsMu.RUnlock()

	if !exists {
		run, err := e.repo.FindRun(ctx, runID)
		if err != nil {
			return err
		}
		if run.Status != StatusRunning {
			return ErrInvalidStatus
		}
		return ErrNotFound
	}

	cancel()
	return nil
}

func (e *executor) Resume(ctx context.Context, runID uuid.UUID) (*Run, error) {
	run, err := e.repo.FindRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	if run.Status != StatusFailed && run.Status != StatusCancelled {
		return nil, ErrInvalidStatus
	}

	factory, exists := Get(run.WorkflowName)
	if !exists {
		return nil, ErrWorkflowNotFound
	}

	var params map[string]any
	if run.Params != nil {
		if err := json.Unmarshal(run.Params, &params); err != nil {
			return nil, fmt.Errorf("unmarshal params: %w", err)
		}
	}

	execCtx, cancel := context.WithCancel(ctx)
	e.trackRun(run.ID, cancel)
	defer e.untrackRun(run.ID)

	run, err = e.repo.UpdateRunStarted(execCtx, run.ID)
	if err != nil {
		return nil, err
	}

	observer := NewPostgresObserver(e.db, run.ID, e.logger)
	checkpointStore := NewPostgresCheckpointStore(e.db, e.logger)

	cfg := config.DefaultGraphConfig(run.WorkflowName)
	cfg.Checkpoint.Interval = 1
	cfg.Checkpoint.Preserve = true

	graph, err := state.NewGraphWithDeps(cfg, observer, checkpointStore)
	if err != nil {
		return e.finalizeRun(ctx, run.ID, StatusFailed, nil, err)
	}

	_, err = factory(execCtx, graph, e.runtime, params)
	if err != nil {
		return e.finalizeRun(ctx, run.ID, StatusFailed, nil, err)
	}

	finalState, err := graph.Resume(execCtx, run.ID.String())

	if err != nil {
		if execCtx.Err() != nil {
			errMsg := "execution cancelled"
			return e.repo.UpdateRunCompleted(ctx, run.ID, StatusCancelled, nil, &errMsg)
		}
		errMsg := err.Error()
		return e.repo.UpdateRunCompleted(ctx, run.ID, StatusFailed, nil, &errMsg)
	}

	return e.repo.UpdateRunCompleted(ctx, run.ID, StatusCompleted, finalState.Data, nil)
}

func (e *executor) trackRun(id uuid.UUID, cancel context.CancelFunc) {
	e.activeRunsMu.Lock()
	defer e.activeRunsMu.Unlock()
	e.activeRuns[id] = cancel
}

func (e *executor) untrackRun(id uuid.UUID) {
	e.activeRunsMu.Lock()
	defer e.activeRunsMu.Unlock()
	delete(e.activeRuns, id)
}

func (e *executor) finalizeRun(ctx context.Context, id uuid.UUID, status RunStatus, result map[string]any, err error) (*Run, error) {
	errMsg := err.Error()
	run, updateErr := e.repo.UpdateRunCompleted(ctx, id, status, result, &errMsg)
	if updateErr != nil {
		e.logger.Error("failed to finalize run", "id", id, "error", updateErr)
		return nil, err
	}
	return run, err
}
```

---

### Phase 6: Domain Integration

**File**: `/home/jaime/code/agent-lab/cmd/server/domain.go`

Add import for workflows package.

Update Domain struct:

```go
type Domain struct {
	Providers providers.System
	Agents    agents.System
	Documents documents.System
	Images    images.System
	Workflows workflows.System
}
```

Update NewDomain function:

```go
func NewDomain(runtime *Runtime) *Domain {
	docs := documents.New(
		runtime.Database.Connection(),
		runtime.Storage,
		runtime.Logger,
		runtime.Pagination,
	)

	agentsSys := agents.New(
		runtime.Database.Connection(),
		runtime.Logger,
		runtime.Pagination,
	)

	imagesSys := images.New(
		docs,
		runtime.Database.Connection(),
		runtime.Storage,
		runtime.Logger,
		runtime.Pagination,
	)

	workflowRuntime := workflows.NewRuntime(agentsSys, docs, imagesSys, runtime.Logger)

	return &Domain{
		Providers: providers.New(
			runtime.Database.Connection(),
			runtime.Logger,
			runtime.Pagination,
		),
		Agents:    agentsSys,
		Documents: docs,
		Images:    imagesSys,
		Workflows: workflows.NewSystem(
			runtime.Database.Connection(),
			workflowRuntime,
			runtime.Logger,
			runtime.Pagination,
		),
	}
}
```

---

### Phase 7: Update go.mod

**File**: `/home/jaime/code/agent-lab/go.mod`

After releasing go-agents-orchestration v0.3.0:

```bash
cd /home/jaime/code/agent-lab
go get github.com/JaimeStill/go-agents-orchestration@v0.3.0
go mod tidy
```

---

## Validation

```bash
cd /home/jaime/code/agent-lab
go vet ./...
go test ./tests/...
```

## Files Summary

| Repository | File | Action |
|------------|------|--------|
| go-agents-orchestration | `pkg/config/state.go` | Add Merge methods |
| go-agents-orchestration | `pkg/config/hub.go` | Add Merge method |
| go-agents-orchestration | `pkg/config/workflows.go` | Add Merge methods |
| go-agents-orchestration | `pkg/state/graph.go` | Add NewGraphWithDeps |
| go-agents-orchestration | `pkg/state/checkpoint.go` | Add mutex |
| go-agents-orchestration | `pkg/observability/registry.go` | Add mutex |
| go-agents-orchestration | `CHANGELOG.md` | v0.3.0 entry |
| agent-lab | `internal/workflows/repository.go` | Add write methods |
| agent-lab | `internal/workflows/registry.go` | Update factory signature |
| agent-lab | `internal/workflows/systems.go` → `runtime.go` | Rename file and struct |
| agent-lab | `internal/workflows/system.go` | New file |
| agent-lab | `internal/workflows/executor.go` | New file |
| agent-lab | `cmd/server/domain.go` | Add Workflows to Domain |
| agent-lab | `go.mod` | Update dependency |
