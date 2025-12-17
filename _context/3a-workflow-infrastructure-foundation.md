# Session 3a: Workflow Infrastructure Foundation

## Objective

Establish database schema, core types, registry, and read-only repository for the workflows domain.

## Pre-Implementation: Documentation Updates

**Status**: ✅ Complete

The Command Execution Model has been added to:
- `_context/web-service-architecture.md` (Section 2)
- `_context/milestones/m03-workflow-execution.md`

---

## Phase 1: Database Migration

### File: `cmd/migrate/migrations/000006_workflows.up.sql`

```sql
CREATE TABLE runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    params JSONB,
    result JSONB,
    error_message TEXT,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_runs_workflow_name ON runs(workflow_name);
CREATE INDEX idx_runs_status ON runs(status);
CREATE INDEX idx_runs_created_at ON runs(created_at DESC);

CREATE TABLE stages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    node_name TEXT NOT NULL,
    iteration INTEGER NOT NULL,
    status TEXT NOT NULL,
    input_snapshot JSONB,
    output_snapshot JSONB,
    duration_ms INTEGER,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_stages_run_id ON stages(run_id);
CREATE INDEX idx_stages_node_name ON stages(node_name);

CREATE TABLE decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    from_node TEXT NOT NULL,
    to_node TEXT,
    predicate_name TEXT,
    predicate_result BOOLEAN,
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_decisions_run_id ON decisions(run_id);

CREATE TABLE checkpoints (
    run_id TEXT PRIMARY KEY,
    state_data JSONB NOT NULL,
    checkpoint_node TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### File: `cmd/migrate/migrations/000006_workflows.down.sql`

```sql
DROP INDEX IF EXISTS idx_decisions_run_id;
DROP TABLE IF EXISTS decisions;

DROP INDEX IF EXISTS idx_stages_run_id;
DROP INDEX IF EXISTS idx_stages_node_name;
DROP TABLE IF EXISTS stages;

DROP INDEX IF EXISTS idx_runs_workflow_name;
DROP INDEX IF EXISTS idx_runs_status;
DROP INDEX IF EXISTS idx_runs_created_at;
DROP TABLE IF EXISTS runs;

DROP TABLE IF EXISTS checkpoints;
```

---

## Phase 2: Core Types

### File: `internal/workflows/run.go`

```go
package workflows

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type RunStatus string

const (
	StatusPending   RunStatus = "pending"
	StatusRunning   RunStatus = "running"
	StatusCompleted RunStatus = "completed"
	StatusFailed    RunStatus = "failed"
	StatusCancelled RunStatus = "cancelled"
)

type StageStatus string

const (
	StageStarted   StageStatus = "started"
	StageCompleted StageStatus = "completed"
	StageFailed    StageStatus = "failed"
)

type Run struct {
	ID           uuid.UUID       `json:"id"`
	WorkflowName string          `json:"workflow_name"`
	Status       RunStatus       `json:"status"`
	Params       json.RawMessage `json:"params,omitempty"`
	Result       json.RawMessage `json:"result,omitempty"`
	ErrorMessage *string         `json:"error_message,omitempty"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type Stage struct {
	ID             uuid.UUID       `json:"id"`
	RunID          uuid.UUID       `json:"run_id"`
	NodeName       string          `json:"node_name"`
	Iteration      int             `json:"iteration"`
	Status         StageStatus     `json:"status"`
	InputSnapshot  json.RawMessage `json:"input_snapshot,omitempty"`
	OutputSnapshot json.RawMessage `json:"output_snapshot,omitempty"`
	DurationMs     *int            `json:"duration_ms,omitempty"`
	ErrorMessage   *string         `json:"error_message,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

type Decision struct {
	ID              uuid.UUID `json:"id"`
	RunID           uuid.UUID `json:"run_id"`
	FromNode        string    `json:"from_node"`
	ToNode          *string   `json:"to_node,omitempty"`
	PredicateName   *string   `json:"predicate_name,omitempty"`
	PredicateResult *bool     `json:"predicate_result,omitempty"`
	Reason          *string   `json:"reason,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type WorkflowInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
```

---

## Phase 3: Domain Errors

### File: `internal/workflows/errors.go`

```go
package workflows

import (
	"errors"
	"net/http"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrWorkflowNotFound = errors.New("workflow not registered")
	ErrInvalidStatus    = errors.New("invalid status transition")
)

func MapHTTPStatus(err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrWorkflowNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrInvalidStatus):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
```

---

## Phase 4: Mapping Infrastructure

### File: `internal/workflows/mapping.go`

```go
package workflows

import (
	"net/url"

	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/agent-lab/pkg/repository"
)

var runProjection = query.NewProjectionMap("public", "runs", "r").
	Project("id", "ID").
	Project("workflow_name", "WorkflowName").
	Project("status", "Status").
	Project("params", "Params").
	Project("result", "Result").
	Project("error_message", "ErrorMessage").
	Project("started_at", "StartedAt").
	Project("completed_at", "CompletedAt").
	Project("created_at", "CreatedAt").
	Project("updated_at", "UpdatedAt")

var runDefaultSort = query.SortField{Field: "CreatedAt", Descending: true}
var stageDefaultSort = query.SortField{Field: "CreatedAt", Descending: false}
var decisionDefaultSort = query.SortField{Field: "CreatedAt", Descending: false}

func scanRun(s repository.Scanner) (Run, error) {
	var r Run
	err := s.Scan(
		&r.ID,
		&r.WorkflowName,
		&r.Status,
		&r.Params,
		&r.Result,
		&r.ErrorMessage,
		&r.StartedAt,
		&r.CompletedAt,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	return r, err
}

var stageProjection = query.NewProjectionMap("public", "stages", "s").
	Project("id", "ID").
	Project("run_id", "RunID").
	Project("node_name", "NodeName").
	Project("iteration", "Iteration").
	Project("status", "Status").
	Project("input_snapshot", "InputSnapshot").
	Project("output_snapshot", "OutputSnapshot").
	Project("duration_ms", "DurationMs").
	Project("error_message", "ErrorMessage").
	Project("created_at", "CreatedAt")

func scanStage(s repository.Scanner) (Stage, error) {
	var st Stage
	err := s.Scan(
		&st.ID,
		&st.RunID,
		&st.NodeName,
		&st.Iteration,
		&st.Status,
		&st.InputSnapshot,
		&st.OutputSnapshot,
		&st.DurationMs,
		&st.ErrorMessage,
		&st.CreatedAt,
	)
	return st, err
}

var decisionProjection = query.NewProjectionMap("public", "decisions", "d").
	Project("id", "ID").
	Project("run_id", "RunID").
	Project("from_node", "FromNode").
	Project("to_node", "ToNode").
	Project("predicate_name", "PredicateName").
	Project("predicate_result", "PredicateResult").
	Project("reason", "Reason").
	Project("created_at", "CreatedAt")

func scanDecision(s repository.Scanner) (Decision, error) {
	var d Decision
	err := s.Scan(
		&d.ID,
		&d.RunID,
		&d.FromNode,
		&d.ToNode,
		&d.PredicateName,
		&d.PredicateResult,
		&d.Reason,
		&d.CreatedAt,
	)
	return d, err
}

type RunFilters struct {
	WorkflowName *string
	Status       *string
}

func RunFiltersFromQuery(values url.Values) RunFilters {
	var f RunFilters

	if wn := values.Get("workflow_name"); wn != "" {
		f.WorkflowName = &wn
	}

	if s := values.Get("status"); s != "" {
		f.Status = &s
	}

	return f
}

func (f RunFilters) Apply(b *query.Builder) *query.Builder {
	return b.
		WhereEquals("WorkflowName", f.WorkflowName).
		WhereEquals("Status", f.Status)
}
```

---

## Phase 5: Registry

### File: `internal/workflows/registry.go`

```go
package workflows

import (
	"context"
	"sync"

	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
)

type WorkflowFactory func(ctx context.Context, systems *Systems, params map[string]any) (state.StateGraph, state.State, error)

type workflowRegistry struct {
	factories map[string]WorkflowFactory
	info      map[string]WorkflowInfo
	mu        sync.RWMutex
}

var registry = &workflowRegistry{
	factories: make(map[string]WorkflowFactory),
	info:      make(map[string]WorkflowInfo),
}

func Register(name string, factory WorkflowFactory, description string) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.factories[name] = factory
	registry.info[name] = WorkflowInfo{Name: name, Description: description}
}

func Get(name string) (WorkflowFactory, bool) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	factory, exists := registry.factories[name]
	return factory, exists
}

func List() []WorkflowInfo {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	result := make([]WorkflowInfo, 0, len(registry.info))
	for _, info := range registry.info {
		result = append(result, info)
	}
	return result
}
```

---

## Phase 6: Systems Struct

### File: `internal/workflows/systems.go`

```go
package workflows

import (
	"log/slog"

	"github.com/JaimeStill/agent-lab/internal/agents"
	"github.com/JaimeStill/agent-lab/internal/documents"
	"github.com/JaimeStill/agent-lab/internal/images"
)

type Systems struct {
	Agents    agents.System
	Documents documents.System
	Images    images.System
	Logger    *slog.Logger
}
```

---

## Phase 7: Repository (Read Operations)

### File: `internal/workflows/repository.go`

```go
package workflows

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/agent-lab/pkg/repository"
	"github.com/google/uuid"
)

type repo struct {
	db         *sql.DB
	logger     *slog.Logger
	pagination pagination.Config
}

func New(db *sql.DB, logger *slog.Logger, pagination pagination.Config) *repo {
	return &repo{
		db:         db,
		logger:     logger.With("system", "workflows"),
		pagination: pagination,
	}
}

func (r *repo) ListRuns(ctx context.Context, page pagination.PageRequest, filters RunFilters) (*pagination.PageResult[Run], error) {
	page.Normalize(r.pagination)

	qb := query.NewBuilder(runProjection, runDefaultSort)
	filters.Apply(qb)

	if len(page.Sort) > 0 {
		qb.OrderByFields(page.Sort)
	}

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count runs: %w", err)
	}

	pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	runs, err := repository.QueryMany(ctx, r.db, pageSQL, pageArgs, scanRun)
	if err != nil {
		return nil, fmt.Errorf("query runs: %w", err)
	}

	result := pagination.NewPageResult(runs, total, page.Page, page.PageSize)
	return &result, nil
}

func (r *repo) FindRun(ctx context.Context, id uuid.UUID) (*Run, error) {
	q, args := query.NewBuilder(runProjection).BuildSingle("ID", id)

	run, err := repository.QueryOne(ctx, r.db, q, args, scanRun)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, nil)
	}

	return &run, nil
}

func (r *repo) GetStages(ctx context.Context, runID uuid.UUID) ([]Stage, error) {
	qb := query.NewBuilder(stageProjection, stageDefaultSort)
	qb.WhereEquals("RunID", &runID)

	q, args := qb.BuildSelect()

	stages, err := repository.QueryMany(ctx, r.db, q, args, scanStage)
	if err != nil {
		return nil, fmt.Errorf("query stages: %w", err)
	}

	return stages, nil
}

func (r *repo) GetDecisions(ctx context.Context, runID uuid.UUID) ([]Decision, error) {
	qb := query.NewBuilder(decisionProjection, decisionDefaultSort)
	qb.WhereEquals("RunID", &runID)

	q, args := qb.BuildSelect()

	decisions, err := repository.QueryMany(ctx, r.db, q, args, scanDecision)
	if err != nil {
		return nil, fmt.Errorf("query decisions: %w", err)
	}

	return decisions, nil
}
```

---

## Validation Criteria

After implementation, verify:

1. **Migration runs successfully**:
   ```bash
   go run ./cmd/migrate up
   ```

2. **Code compiles without errors**:
   ```bash
   go vet ./...
   ```

3. **Registry works correctly** (manual verification in later session)

4. **All types have proper JSON tags**

---

## Session Scope Boundaries

### In Scope (3a)
- Database schema (4 tables)
- Core types (Run, Stage, Decision, WorkflowInfo)
- Domain errors
- Mapping infrastructure (projections, scanners, filters)
- Global registry (Register, Get, List)
- Systems struct placeholder
- Read-only repository (ListRuns, FindRun, GetStages, GetDecisions)

### Out of Scope (deferred)
- Write operations in repository (3c - part of command model)
- Observer implementation (3b)
- Checkpoint store implementation (3b)
- Executor (3c)
- Handler and API endpoints (3d)
- OpenAPI specification (3d)

---

## File Summary

| File | Purpose |
|------|---------|
| `cmd/migrate/migrations/000006_workflows.up.sql` | Create runs, stages, decisions, checkpoints tables |
| `cmd/migrate/migrations/000006_workflows.down.sql` | Drop tables in reverse order |
| `internal/workflows/run.go` | Run, Stage, Decision, WorkflowInfo types |
| `internal/workflows/errors.go` | Domain errors + MapHTTPStatus |
| `internal/workflows/mapping.go` | Projections, scanners, filters |
| `internal/workflows/registry.go` | Global workflow registry |
| `internal/workflows/systems.go` | Systems struct placeholder |
| `internal/workflows/repository.go` | Read-only repository |
