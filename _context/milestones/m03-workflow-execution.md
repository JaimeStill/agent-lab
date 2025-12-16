# Milestone 3: Workflow Execution Infrastructure

## Overview

Build infrastructure for executing code-defined workflows with full observability, enabling iterative experimentation on agentic workflow designs.

**Key Insight**: Workflows are Go code registered by name. The database captures execution records, not workflow definitions. This avoids the over-abstraction that plagued the original M3 design.

## Key Decisions

### 1. Code-Defined Workflows

Workflows are Go functions registered at startup by name. No workflow definition CRUD.

**Rationale**:
- Workflows are versioned with code (git)
- Type-safe node composition
- No schema complexity for workflow structures
- Easy to test in isolation
- Clear separation: code defines behavior, database records execution

**Pattern**:
```go
type WorkflowFactory func(ctx context.Context, systems *Systems, params map[string]any) (state.StateGraph, state.State, error)

registry.Register("classify-document", classifyDocumentFactory)
```

### 2. Observer-Based Visibility

Per-stage results and routing decisions captured via go-agents-orchestration's Observer interface. Events are both:
- **Persisted to database** - For historical analysis and comparison
- **Streamed via SSE** - For real-time monitoring during execution

### 3. Infrastructure Only

No classify-docs workflow in M3. This milestone builds the execution engine; classify-docs implementation comes in Milestone 5.

### 4. Sync + SSE Streaming

Two execution modes:
- **Sync**: `POST /api/workflows/{name}/execute` - Wait for completion, return result
- **Streaming**: `POST /api/workflows/{name}/execute/stream` - Real-time SSE progress with cancellation support

No background/polling model needed for iteration workflow.

## Architecture

```
HTTP Request → Handler → Executor → StateGraph (go-agents-orchestration)
                                         ↓
                              PostgresObserver → Database
                              PostgresCheckpointStore → Database
```

### Component Relationships

```
┌───────────────────────────────────────────────────────────────────┐
│                         Handler Layer                             │
│  POST /workflows/{name}/execute → Executor.Execute()              │
│  POST /workflows/{name}/execute/stream → Executor.ExecuteStream() │
│  POST /runs/{id}/cancel → Executor.Cancel()                       │
│  POST /runs/{id}/resume → Executor.Resume()                       │
└───────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│                        Executor                                  │
│  - Resolves workflow from Registry                               │
│  - Creates Run record                                            │
│  - Configures Observer + CheckpointStore                         │
│  - Executes StateGraph                                           │
│  - Updates Run on completion/failure                             │
└──────────────────────────────────────────────────────────────────┘
        │                 │                        │
        ▼                 ▼                        ▼
  ┌───────────┐  ┌──────────────────┐  ┌─────────────────────────┐
  │  Registry │  │ PostgresObserver │  │ PostgresCheckpointStore │
  │           │  │                  │  │                         │
  │ Workflow  │  │ Captures:        │  │ Implements:             │
  │ factories │  │ - node.start     │  │ - Save(state)           │
  │ by name   │  │ - node.complete  │  │ - Load(runID)           │
  │           │  │ - edge.evaluate  │  │ - Delete(runID)         │
  │           │  │ - edge.transition│  │ - List()                │
  └───────────┘  └──────────────────┘  └─────────────────────────┘
```

### Integration with Existing Domains

Workflows access existing domains through a Systems struct:

```go
type Systems struct {
    Agents    agents.System
    Documents documents.System
    Images    images.System
    Logger    *slog.Logger
}
```

This enables workflow nodes to:
- Execute LLM calls via agents domain
- Retrieve document metadata and storage paths
- Request image rendering with enhancement filters

## Command Execution Model (Session 3c)

### Command Graph Composition

Commands compose as a recursive dependency graph:

```
Root Command (receives transaction)
├── Single state mutation
└── Child Commands (direct dependencies)
    ├── Single state mutation
    └── Child Commands...
```

### Three-Phase Execution Lifecycle

| Phase | Name | Purpose |
|-------|------|---------|
| 1 | **Cold Start** | Pre-execution: calculations, validation, state preparation |
| 2 | **Hot Start** | Transaction graph execution: actual mutations |
| 3 | **Post-Commit** | Reactions: events, notifications, observability emissions |

### Application to Workflow Execution

When executing a workflow:
- **Cold Start**: Resolve workflow from registry, validate params, create Run record shell
- **Hot Start**: Execute StateGraph, Observer persists stages/decisions within tx
- **Post-Commit**: Emit SSE events, update external systems, trigger notifications

Commands avoid defining all mutations directly - they compose through child commands that handle their own single mutation.

## Database Schema

### runs

Execution records tracking workflow invocations.

```sql
CREATE TABLE runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    params JSONB,
    result JSONB,
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_runs_workflow_name ON runs(workflow_name);
CREATE INDEX idx_runs_status ON runs(status);
CREATE INDEX idx_runs_created ON runs(created_at DESC);
```

**Status values**: `pending`, `running`, `completed`, `failed`, `cancelled`

### stages

Per-node execution captured via Observer events.

```sql
CREATE TABLE stages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    node_name VARCHAR(255) NOT NULL,
    iteration INT NOT NULL,
    status VARCHAR(50) NOT NULL,
    input_snapshot JSONB,
    output_snapshot JSONB,
    duration_ms INT,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stages_run ON stages(run_id);
CREATE INDEX idx_stages_node ON stages(node_name);
```

**Status values**: `started`, `completed`, `failed`

### decisions

Routing decisions captured via Observer events.

```sql
CREATE TABLE decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    from_node VARCHAR(255) NOT NULL,
    to_node VARCHAR(255),
    predicate_name VARCHAR(255),
    predicate_result BOOLEAN,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_decisions_run ON decisions(run_id);
```

### checkpoints

State persistence for resume capability.

```sql
CREATE TABLE checkpoints (
    run_id VARCHAR(255) PRIMARY KEY,
    state_data JSONB NOT NULL,
    checkpoint_node VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/workflows` | List registered workflows |
| POST | `/api/workflows/{name}/execute` | Execute workflow (sync) |
| POST | `/api/workflows/{name}/execute/stream` | Execute with SSE progress |
| GET | `/api/runs` | List runs with filters |
| GET | `/api/runs/{id}` | Get run details |
| GET | `/api/runs/{id}/stages` | Get execution stages |
| GET | `/api/runs/{id}/decisions` | Get routing decisions |
| POST | `/api/runs/{id}/cancel` | Cancel running workflow |
| POST | `/api/runs/{id}/resume` | Resume from checkpoint |

### SSE Streaming Events

During `/execute/stream`, events are streamed as they occur:

| Event Type | Description | Data |
|------------|-------------|------|
| `stage.start` | Node beginning execution | `{node_name, iteration}` |
| `stage.complete` | Node finished | `{node_name, iteration, duration_ms, output_snapshot}` |
| `decision` | Routing decision made | `{from_node, to_node, predicate_result, reason}` |
| `error` | Error occurred | `{message, node_name}` |
| `complete` | Workflow finished | `{result}` |

## Key Interfaces

### go-agents-orchestration Observer

```go
// From go-agents-orchestration/pkg/observability/observer.go
type Observer interface {
    OnEvent(ctx context.Context, event Event)
}

type Event struct {
    Type      EventType
    Timestamp time.Time
    Source    string
    Data      map[string]any
}
```

**Relevant event types**:
- `EventGraphStart`, `EventGraphComplete`
- `EventNodeStart`, `EventNodeComplete`
- `EventEdgeEvaluate`, `EventEdgeTransition`
- `EventCheckpointSave`, `EventCheckpointLoad`, `EventCheckpointResume`

### go-agents-orchestration CheckpointStore

```go
// From go-agents-orchestration/pkg/state/checkpoint.go
type CheckpointStore interface {
    Save(state State) error
    Load(runID string) (State, error)
    Delete(runID string) error
    List() ([]string, error)
}
```

### Workflow Registry

```go
type WorkflowFactory func(ctx context.Context, systems *Systems, params map[string]any) (state.StateGraph, state.State, error)

type Registry interface {
    Register(name string, factory WorkflowFactory)
    Get(name string) (WorkflowFactory, bool)
    List() []WorkflowInfo
}
```

### Executor

```go
type Executor interface {
    Execute(ctx context.Context, name string, params map[string]any) (*Run, error)
    ExecuteStream(ctx context.Context, name string, params map[string]any) (<-chan ExecutionEvent, error)
    Cancel(ctx context.Context, runID uuid.UUID) error
    Resume(ctx context.Context, runID uuid.UUID) (*Run, error)
}
```

## Session Breakdown

### Session 3a: Workflow Infrastructure Foundation

**Objective**: Database schema and core types.

**Files**:
- `cmd/migrate/migrations/000005_workflows.up.sql`
- `cmd/migrate/migrations/000005_workflows.down.sql`
- `internal/workflows/run.go` - Run, Stage, Decision types
- `internal/workflows/errors.go` - Domain errors
- `internal/workflows/projection.go` - Query projections
- `internal/workflows/scanner.go` - Row scanners
- `internal/workflows/filters.go` - Query filters
- `internal/workflows/registry.go` - Workflow registration
- `internal/workflows/repository.go` - Run CRUD

### Session 3b: Observer and Checkpoint Store

**Objective**: Implement go-agents-orchestration interfaces for persistence.

**Files**:
- `internal/workflows/observer.go` - PostgresObserver
- `internal/workflows/checkpoint.go` - PostgresCheckpointStore

### Session 3c: Workflow Execution Engine

**Objective**: Connect components to execute workflows.

**Files**:
- `internal/workflows/executor.go` - Execution coordination
- `internal/workflows/systems.go` - Domain access struct
- `internal/workflows/system.go` - System interface

### Session 3d: API Endpoints

**Objective**: HTTP interface for execution and inspection.

**Files**:
- `internal/workflows/handler.go` - HTTP handlers
- `internal/workflows/openapi.go` - OpenAPI specification

### Session 3e: Sample Workflow and Integration Tests

**Objective**: Validate infrastructure with working example.

**Files**:
- `internal/workflows/samples/echo.go` - Echo workflow
- `internal/workflows/samples/conditional.go` - Conditional routing workflow
- `tests/workflows/` - Integration tests

## Reference Files

### Existing Patterns
- `internal/agents/handler.go` - Handler struct pattern
- `internal/agents/system.go` - System interface pattern
- `internal/documents/repository.go` - Repository pattern

### go-agents-orchestration
- `pkg/observability/observer.go` - Observer interface
- `pkg/state/checkpoint.go` - CheckpointStore interface
- `pkg/state/graph.go` - StateGraph implementation

## Risk Areas

| Risk | Mitigation |
|------|------------|
| Observer performance overhead | Start sync; add async batching if needed |
| State serialization | Use json.RawMessage; test with large states |
| Domain coupling | Clean interfaces via Systems struct |
| Cancellation edge cases | Comprehensive tests for mid-execution cancel |
| Checkpoint consistency | Atomic writes with transaction support |

## What M3 Does NOT Include

- Workflow definition CRUD (workflows live in code)
- classify-docs workflow implementation (Milestone 5)
- Batch/bulk execution (Milestone 7)
- Web UI (Milestone 6)
- Background/async execution with polling (sync + SSE streaming covers iteration needs)
