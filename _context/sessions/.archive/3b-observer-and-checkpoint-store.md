# Session 3b: Observer and Checkpoint Store

## Overview

This session implements go-agents-orchestration interfaces (`Observer` and `CheckpointStore`) for PostgreSQL persistence. It includes a maintenance release to go-agents-orchestration to enable proper State serialization.

## Part 1: go-agents-orchestration Maintenance Release

### Phase 1.1: State Struct - Public Fields

**File**: `/home/jaime/code/go-agents-orchestration/pkg/state/state.go`

Replace the State struct definition (lines 24-30):

```go
type State struct {
	Data           map[string]any         `json:"data"`
	Observer       observability.Observer `json:"-"`
	RunID          string                 `json:"run_id"`
	CheckpointNode string                 `json:"checkpoint_node"`
	Timestamp      time.Time              `json:"timestamp"`
}
```

Remove these getter methods (delete lines 32-55):
- `func (s State) RunID() string`
- `func (s State) CheckpointNode() string`
- `func (s State) Timestamp() time.Time`

Update `New()` function (lines 67-87) - change field references:

```go
func New(observer observability.Observer) State {
	if observer == nil {
		observer = observability.NoOpObserver{}
	}

	s := State{
		Data:      make(map[string]any),
		Observer:  observer,
		RunID:     uuid.New().String(),
		Timestamp: time.Now(),
	}

	observer.OnEvent(context.Background(), observability.Event{
		Type:      observability.EventStateCreate,
		Timestamp: s.Timestamp,
		Source:    "state",
		Data:      map[string]any{},
	})

	return s
}
```

Update `Clone()` method - change field references:

```go
func (s State) Clone() State {
	newState := State{
		Data:           maps.Clone(s.Data),
		Observer:       s.Observer,
		RunID:          s.RunID,
		CheckpointNode: s.CheckpointNode,
		Timestamp:      s.Timestamp,
	}

	s.Observer.OnEvent(context.Background(), observability.Event{
		Type:      observability.EventStateClone,
		Timestamp: time.Now(),
		Source:    "state",
		Data:      map[string]any{"keys": len(newState.Data)},
	})

	return newState
}
```

Update `Get()` method:

```go
func (s State) Get(key string) (any, bool) {
	val, exists := s.Data[key]
	return val, exists
}
```

Update `Set()` method:

```go
func (s State) Set(key string, value any) State {
	newState := s.Clone()
	newState.Data[key] = value

	s.Observer.OnEvent(context.Background(), observability.Event{
		Type:      observability.EventStateSet,
		Timestamp: time.Now(),
		Source:    "state",
		Data:      map[string]any{"key": key},
	})

	return newState
}
```

Update `SetCheckpointNode()` method:

```go
func (s State) SetCheckpointNode(node string) State {
	newState := s.Clone()
	newState.CheckpointNode = node
	newState.Timestamp = time.Now()
	return newState
}
```

Update `Merge()` method:

```go
func (s State) Merge(other State) State {
	newState := s.Clone()
	maps.Copy(newState.Data, other.Data)

	s.Observer.OnEvent(context.Background(), observability.Event{
		Type:      observability.EventStateMerge,
		Timestamp: time.Now(),
		Source:    "state",
		Data:      map[string]any{"keys": len(other.Data)},
	})

	return newState
}
```

### Phase 1.2: Edge Struct - Add Name Field

**File**: `/home/jaime/code/go-agents-orchestration/pkg/state/edge.go`

Add `Name` field to Edge struct:

```go
type Edge struct {
	From string
	To   string
	Name string
	Predicate TransitionPredicate
}
```

### Phase 1.3: Checkpoint Store - Update Field Access

**File**: `/home/jaime/code/go-agents-orchestration/pkg/state/checkpoint.go`

Update `memoryCheckpointStore.Save()`:

```go
func (m *memoryCheckpointStore) Save(state State) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.states[state.RunID] = state
	return nil
}
```

### Phase 1.4: Graph - Update Field Access and Enhance Events

**File**: `/home/jaime/code/go-agents-orchestration/pkg/state/graph.go`

Update `Resume()` method - change getter calls to field access and enhance events:

```go
func (g *stateGraph) Resume(ctx context.Context, runID string) (State, error) {
	if g.checkpointStore == nil {
		return State{}, fmt.Errorf("checkpointing not enabled for this graph")
	}

	state, err := g.checkpointStore.Load(runID)
	if err != nil {
		return State{}, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	g.observer.OnEvent(ctx, observability.Event{
		Type:      observability.EventCheckpointLoad,
		Timestamp: time.Now(),
		Source:    g.name,
		Data: map[string]any{
			"node":   state.CheckpointNode,
			"run_id": runID,
		},
	})

	nextNode, err := g.findNextNode(state.CheckpointNode, state)
	if err != nil {
		return State{}, fmt.Errorf("failed to find next node after checkpoint: %w", err)
	}

	g.observer.OnEvent(ctx, observability.Event{
		Type:      observability.EventCheckpointResume,
		Timestamp: time.Now(),
		Source:    g.name,
		Data: map[string]any{
			"checkpoint_node": state.CheckpointNode,
			"resume_node":     nextNode,
			"run_id":          runID,
		},
	})

	return g.execute(ctx, nextNode, state)
}
```

Update `execute()` method - change getter calls and enhance events:

In EventGraphStart (around line 329):
```go
g.observer.OnEvent(ctx, observability.Event{
	Type:      observability.EventGraphStart,
	Timestamp: time.Now(),
	Source:    g.name,
	Data: map[string]any{
		"entry_point": g.entryPoint,
		"run_id":      initialState.RunID,
		"exit_points": len(g.exitPoints),
	},
})
```

In EventNodeStart (around line 393) - add input_snapshot:
```go
g.observer.OnEvent(ctx, observability.Event{
	Type:      observability.EventNodeStart,
	Timestamp: time.Now(),
	Source:    g.name,
	Data: map[string]any{
		"node":           current,
		"iteration":      iterations,
		"input_snapshot": maps.Clone(state.Data),
	},
})
```

In EventNodeComplete (around line 405) - add output_snapshot:
```go
g.observer.OnEvent(ctx, observability.Event{
	Type:      observability.EventNodeComplete,
	Timestamp: time.Now(),
	Source:    g.name,
	Data: map[string]any{
		"node":            current,
		"iteration":       iterations,
		"error":           err != nil,
		"output_snapshot": maps.Clone(newState.Data),
	},
})
```

In checkpoint save event (around line 437):
```go
g.observer.OnEvent(ctx, observability.Event{
	Type:      observability.EventCheckpointSave,
	Timestamp: time.Now(),
	Source:    g.name,
	Data: map[string]any{
		"node":   current,
		"run_id": state.RunID,
	},
})
```

In checkpoint delete (around line 461):
```go
g.checkpointStore.Delete(state.RunID)
```

In EventEdgeTransition (around line 494) - add predicate info:
```go
g.observer.OnEvent(ctx, observability.Event{
	Type:      observability.EventEdgeTransition,
	Timestamp: time.Now(),
	Source:    g.name,
	Data: map[string]any{
		"from":             edge.From,
		"to":               edge.To,
		"edge_index":       i,
		"predicate_name":   edge.Name,
		"predicate_result": true,
	},
})
```

### Phase 1.5: Update Tests

**File**: `/home/jaime/code/go-agents-orchestration/tests/state/state_test.go`

No changes needed - tests don't use the removed getters directly.

**File**: `/home/jaime/code/go-agents-orchestration/tests/state/checkpoint_test.go`

Update all getter calls to field access:

- `s.RunID()` → `s.RunID`
- `s.CheckpointNode()` → `s.CheckpointNode`
- `s.Timestamp()` → `s.Timestamp`

Example changes:

```go
func TestState_CheckpointMetadata(t *testing.T) {
	observer := observability.NoOpObserver{}
	s := state.New(observer)

	if s.RunID == "" {
		t.Error("Expected non-empty RunID")
	}

	if s.CheckpointNode != "" {
		t.Errorf("Expected empty CheckpointNode, got %s", s.CheckpointNode)
	}

	if s.Timestamp.IsZero() {
		t.Error("Expected non-zero Timestamp")
	}
}
```

Apply similar changes throughout:
- `TestState_SetCheckpointNode`
- `TestState_Clone_PreservesCheckpointMetadata`
- `TestState_Set_PreservesCheckpointMetadata`
- `TestState_Merge_PreservesCheckpointMetadata`
- `TestMemoryCheckpointStore_SaveAndLoad`
- `TestMemoryCheckpointStore_Delete`
- `TestMemoryCheckpointStore_List`
- `TestMemoryCheckpointStore_Overwrite`
- `TestGraph_Checkpoint_Disabled`
- `TestGraph_Checkpoint_SaveAtInterval`
- `TestGraph_Checkpoint_PreserveOnSuccess`
- `TestGraph_Resume_FromCheckpoint`
- `TestState_Checkpoint_Method`

**File**: `/home/jaime/code/go-agents-orchestration/tests/state/graph_test.go`

Search for and update any `RunID()`, `CheckpointNode()`, or `Timestamp()` calls.

### Phase 1.6: Validation and Release

```bash
cd /home/jaime/code/go-agents-orchestration
go vet ./...
go test ./tests/...
git add .
git commit -m "feat: make State fields public for JSON serialization

BREAKING CHANGE: State struct fields are now public with JSON tags.
Removed redundant getter methods: RunID(), CheckpointNode(), Timestamp().
Enhanced observer events with input/output snapshots and predicate names."
git tag v0.2.0
git push origin main --tags
```

---

## Part 2: agent-lab Implementation

### Phase 2.1: Update go.mod

**File**: `/home/jaime/code/agent-lab/go.mod`

Update the go-agents-orchestration dependency:

```bash
cd /home/jaime/code/agent-lab
go get github.com/JaimeStill/go-agents-orchestration@v0.2.0
go mod tidy
```

### Phase 2.2: PostgresCheckpointStore

**File**: `/home/jaime/code/agent-lab/internal/workflows/checkpoint.go`

```go
package workflows

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/JaimeStill/agent-lab/pkg/repository"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
)

type PostgresCheckpointStore struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewPostgresCheckpointStore(db *sql.DB, logger *slog.Logger) *PostgresCheckpointStore {
	return &PostgresCheckpointStore{
		db:     db,
		logger: logger,
	}
}

func (s *PostgresCheckpointStore) Save(st state.State) error {
	stateData, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	const query = `
		INSERT INTO checkpoints (run_id, state_data, checkpoint_node, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (run_id) DO UPDATE SET
			state_data = EXCLUDED.state_data,
			checkpoint_node = EXCLUDED.checkpoint_node,
			updated_at = NOW()
	`

	_, err = s.db.ExecContext(context.Background(), query,
		st.RunID, stateData, st.CheckpointNode)
	if err != nil {
		return fmt.Errorf("save checkpoint: %w", err)
	}

	s.logger.Debug("checkpoint saved", "run_id", st.RunID, "node", st.CheckpointNode)
	return nil
}

func (s *PostgresCheckpointStore) Load(runID string) (state.State, error) {
	const query = `SELECT state_data FROM checkpoints WHERE run_id = $1`

	var stateData []byte
	err := s.db.QueryRowContext(context.Background(), query, runID).Scan(&stateData)
	if err != nil {
		if err == sql.ErrNoRows {
			return state.State{}, fmt.Errorf("checkpoint not found: %s", runID)
		}
		return state.State{}, fmt.Errorf("query checkpoint: %w", err)
	}

	var st state.State
	if err := json.Unmarshal(stateData, &st); err != nil {
		return state.State{}, fmt.Errorf("unmarshal state: %w", err)
	}

	s.logger.Debug("checkpoint loaded", "run_id", runID)
	return st, nil
}

func (s *PostgresCheckpointStore) Delete(runID string) error {
	const query = `DELETE FROM checkpoints WHERE run_id = $1`

	_, err := s.db.ExecContext(context.Background(), query, runID)
	if err != nil {
		return fmt.Errorf("delete checkpoint: %w", err)
	}

	s.logger.Debug("checkpoint deleted", "run_id", runID)
	return nil
}

func (s *PostgresCheckpointStore) List() ([]string, error) {
	const query = `SELECT run_id FROM checkpoints ORDER BY created_at DESC`

	ids, err := repository.QueryMany(context.Background(), s.db, query, nil,
		func(sc repository.Scanner) (string, error) {
			var id string
			err := sc.Scan(&id)
			return id, err
		})
	if err != nil {
		return nil, fmt.Errorf("list checkpoints: %w", err)
	}

	return ids, nil
}
```

### Phase 2.3: PostgresObserver

**File**: `/home/jaime/code/agent-lab/internal/workflows/observer.go`

```go
package workflows

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/JaimeStill/go-agents-orchestration/pkg/observability"
	"github.com/google/uuid"
)

type PostgresObserver struct {
	db         *sql.DB
	runID      uuid.UUID
	logger     *slog.Logger
	mu         sync.Mutex
	startTimes map[string]time.Time
}

func NewPostgresObserver(db *sql.DB, runID uuid.UUID, logger *slog.Logger) *PostgresObserver {
	return &PostgresObserver{
		db:         db,
		runID:      runID,
		logger:     logger,
		startTimes: make(map[string]time.Time),
	}
}

func (o *PostgresObserver) OnEvent(ctx context.Context, event observability.Event) {
	o.mu.Lock()
	defer o.mu.Unlock()

	switch event.Type {
	case observability.EventNodeStart:
		o.handleNodeStart(ctx, event)
	case observability.EventNodeComplete:
		o.handleNodeComplete(ctx, event)
	case observability.EventEdgeTransition:
		o.handleEdgeTransition(ctx, event)
	default:
		o.logger.Debug("unhandled event", "type", event.Type, "source", event.Source)
	}
}

func (o *PostgresObserver) handleNodeStart(ctx context.Context, event observability.Event) {
	nodeName, _ := event.Data["node"].(string)
	iteration, _ := event.Data["iteration"].(int)
	inputSnapshot, _ := event.Data["input_snapshot"].(map[string]any)

	key := fmt.Sprintf("%s:%d", nodeName, iteration)
	o.startTimes[key] = event.Timestamp

	var inputData []byte
	if inputSnapshot != nil {
		data, err := json.Marshal(inputSnapshot)
		if err != nil {
			o.logger.Error("failed to marshal input snapshot", "error", err, "node", nodeName)
		} else {
			inputData = data
		}
	}

	const query = `
		INSERT INTO stages (run_id, node_name, iteration, status, input_snapshot, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := o.db.ExecContext(ctx, query,
		o.runID, nodeName, iteration, StageStarted, inputData, event.Timestamp)
	if err != nil {
		o.logger.Error("failed to insert stage", "error", err, "node", nodeName)
	}
}

func (o *PostgresObserver) handleNodeComplete(ctx context.Context, event observability.Event) {
	nodeName, _ := event.Data["node"].(string)
	iteration, _ := event.Data["iteration"].(int)
	hasError, _ := event.Data["error"].(bool)
	outputSnapshot, _ := event.Data["output_snapshot"].(map[string]any)

	status := StageCompleted
	if hasError {
		status = StageFailed
	}

	key := fmt.Sprintf("%s:%d", nodeName, iteration)
	var durationMs *int
	if startTime, ok := o.startTimes[key]; ok {
		duration := int(event.Timestamp.Sub(startTime).Milliseconds())
		durationMs = &duration
		delete(o.startTimes, key)
	}

	var outputData []byte
	if outputSnapshot != nil {
		data, err := json.Marshal(outputSnapshot)
		if err != nil {
			o.logger.Error("failed to marshal output snapshot", "error", err, "node", nodeName)
		} else {
			outputData = data
		}
	}

	const query = `
		UPDATE stages
		SET status = $1, duration_ms = $2, output_snapshot = $3
		WHERE run_id = $4 AND node_name = $5 AND iteration = $6
	`

	_, err := o.db.ExecContext(ctx, query,
		status, durationMs, outputData, o.runID, nodeName, iteration)
	if err != nil {
		o.logger.Error("failed to update stage", "error", err, "node", nodeName)
	}
}

func (o *PostgresObserver) handleEdgeTransition(ctx context.Context, event observability.Event) {
	fromNode, _ := event.Data["from"].(string)
	toNode, _ := event.Data["to"].(string)
	predicateName, _ := event.Data["predicate_name"].(string)
	predicateResult, _ := event.Data["predicate_result"].(bool)

	var predNamePtr *string
	if predicateName != "" {
		predNamePtr = &predicateName
	}

	const query = `
		INSERT INTO decisions (run_id, from_node, to_node, predicate_name, predicate_result, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := o.db.ExecContext(ctx, query,
		o.runID, fromNode, toNode, predNamePtr, predicateResult, event.Timestamp)
	if err != nil {
		o.logger.Error("failed to insert decision", "error", err, "from", fromNode, "to", toNode)
	}
}
```

### Phase 2.4: Validation

```bash
cd /home/jaime/code/agent-lab
go vet ./...
go test ./tests/...
```

---

## File Summary

### go-agents-orchestration (Maintenance Release)

| File | Changes |
|------|---------|
| `pkg/state/state.go` | Public fields with JSON tags, remove getters |
| `pkg/state/edge.go` | Add Name field |
| `pkg/state/checkpoint.go` | Update field access |
| `pkg/state/graph.go` | Update field access, enhance events with snapshots |
| `tests/state/checkpoint_test.go` | Update all getter calls to field access |
| `tests/state/graph_test.go` | Update any getter calls |

### agent-lab (New Files)

| File | Purpose |
|------|---------|
| `internal/workflows/checkpoint.go` | PostgresCheckpointStore implementation |
| `internal/workflows/observer.go` | PostgresObserver implementation |
