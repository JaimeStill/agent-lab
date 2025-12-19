# Session 3d: API Endpoints Implementation Guide

## Overview

Implement HTTP handlers for workflow execution and inspection, including real-time SSE streaming.

## Architecture

```
Handler.ExecuteStream()
        │
        ▼
ExecuteStream(ctx, name, params) → <-chan ExecutionEvent
        │
        │── Creates MultiObserver(PostgresObserver, StreamingObserver)
        │
        ▼
StateGraph.Execute() ──events──▶ MultiObserver
                                       │
                      ┌────────────────┴────────────────┐
                      │                                 │
                      ▼                                 ▼
              PostgresObserver                  StreamingObserver
              (persists to DB)                  (sends to channel)
                                                       │
                                                       ▼
                                             <-chan ExecutionEvent
                                                       │
                                                       ▼
                                              Handler SSE loop
```

---

## Phase 1: Streaming Infrastructure

### 1.1 Add ExecutionEvent Types to run.go

Add to `internal/workflows/run.go`:

```go
type ExecutionEventType string

const (
	EventStageStart    ExecutionEventType = "stage.start"
	EventStageComplete ExecutionEventType = "stage.complete"
	EventDecision      ExecutionEventType = "decision"
	EventError         ExecutionEventType = "error"
	EventComplete      ExecutionEventType = "complete"
)

type ExecutionEvent struct {
	Type      ExecutionEventType `json:"type"`
	Timestamp time.Time          `json:"timestamp"`
	Data      map[string]any     `json:"data"`
}
```

### 1.2 Create streaming.go

Create new file `internal/workflows/streaming.go`:

```go
package workflows

import (
	"context"
	"sync"
	"time"

	"github.com/JaimeStill/agent-lab/pkg/decode"
	"github.com/JaimeStill/go-agents-orchestration/pkg/observability"
)

type StreamingObserver struct {
	events chan ExecutionEvent
	mu     sync.Mutex
	closed bool
}

func NewStreamingObserver(bufferSize int) *StreamingObserver {
	return &StreamingObserver{
		events: make(chan ExecutionEvent, bufferSize),
	}
}

func (o *StreamingObserver) Events() <-chan ExecutionEvent {
	return o.events
}

func (o *StreamingObserver) Close() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.closed {
		o.closed = true
		close(o.events)
	}
}

func (o *StreamingObserver) OnEvent(ctx context.Context, event observability.Event) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return
	}

	var execEvent *ExecutionEvent

	switch event.Type {
	case observability.EventNodeStart:
		execEvent = o.handleNodeStart(event)
	case observability.EventNodeComplete:
		execEvent = o.handleNodeComplete(event)
	case observability.EventEdgeTransition:
		execEvent = o.handleEdgeTransition(event)
	}

	if execEvent != nil {
		select {
		case o.events <- *execEvent:
		default:
		}
	}
}

func (o *StreamingObserver) handleNodeStart(event observability.Event) *ExecutionEvent {
	data, err := decode.FromMap[NodeStartData](event.Data)
	if err != nil {
		return nil
	}
	return &ExecutionEvent{
		Type:      EventStageStart,
		Timestamp: event.Timestamp,
		Data: map[string]any{
			"node_name": data.Node,
			"iteration": data.Iteration,
		},
	}
}

func (o *StreamingObserver) handleNodeComplete(event observability.Event) *ExecutionEvent {
	data, err := decode.FromMap[NodeCompleteData](event.Data)
	if err != nil {
		return nil
	}
	if data.Error {
		return &ExecutionEvent{
			Type:      EventError,
			Timestamp: event.Timestamp,
			Data: map[string]any{
				"node_name": data.Node,
				"message":   data.ErrorMessage,
			},
		}
	}
	return &ExecutionEvent{
		Type:      EventStageComplete,
		Timestamp: event.Timestamp,
		Data: map[string]any{
			"node_name":       data.Node,
			"iteration":       data.Iteration,
			"output_snapshot": data.OutputSnapshot,
		},
	}
}

func (o *StreamingObserver) handleEdgeTransition(event observability.Event) *ExecutionEvent {
	data, err := decode.FromMap[EdgeTransitionData](event.Data)
	if err != nil {
		return nil
	}
	return &ExecutionEvent{
		Type:      EventDecision,
		Timestamp: event.Timestamp,
		Data: map[string]any{
			"from_node":        data.From,
			"to_node":          data.To,
			"predicate_result": data.PredicateResult,
		},
	}
}

func (o *StreamingObserver) SendComplete(result map[string]any) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return
	}
	select {
	case o.events <- ExecutionEvent{
		Type:      EventComplete,
		Timestamp: time.Now(),
		Data:      map[string]any{"result": result},
	}:
	default:
	}
}

func (o *StreamingObserver) SendError(err error, nodeName string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return
	}
	data := map[string]any{"message": err.Error()}
	if nodeName != "" {
		data["node_name"] = nodeName
	}
	select {
	case o.events <- ExecutionEvent{
		Type:      EventError,
		Timestamp: time.Now(),
		Data:      data,
	}:
	default:
	}
}

type MultiObserver struct {
	observers []observability.Observer
}

func NewMultiObserver(observers ...observability.Observer) *MultiObserver {
	return &MultiObserver{observers: observers}
}

func (m *MultiObserver) OnEvent(ctx context.Context, event observability.Event) {
	for _, obs := range m.observers {
		obs.OnEvent(ctx, event)
	}
}
```

---

## Phase 2: ExecuteStream Method

### 2.1 Update System Interface

Modify `internal/workflows/system.go` - add `ExecuteStream` to the interface:

```go
ExecuteStream(ctx context.Context, name string, params map[string]any) (<-chan ExecutionEvent, *Run, error)
```

### 2.2 Implement ExecuteStream in executor.go

Add to `internal/workflows/executor.go`:

```go
func (e *executor) ExecuteStream(ctx context.Context, name string, params map[string]any) (<-chan ExecutionEvent, *Run, error) {
	factory, exists := Get(name)
	if !exists {
		return nil, nil, ErrWorkflowNotFound
	}

	run, err := e.repo.CreateRun(ctx, name, params)
	if err != nil {
		return nil, nil, fmt.Errorf("create run: %w", err)
	}

	streamingObs := NewStreamingObserver(100)

	go e.executeStreamAsync(ctx, run.ID, factory, params, streamingObs)

	return streamingObs.Events(), run, nil
}

func (e *executor) executeStreamAsync(ctx context.Context, runID uuid.UUID, factory WorkflowFactory, params map[string]any, streamingObs *StreamingObserver) {
	defer streamingObs.Close()

	execCtx, cancel := context.WithCancel(ctx)
	e.trackRun(runID, cancel)
	defer e.untrackRun(runID)

	_, err := e.repo.UpdateRunStarted(execCtx, runID)
	if err != nil {
		streamingObs.SendError(err, "")
		e.finalizeRun(ctx, runID, StatusFailed, nil, err)
		return
	}

	postgresObs := NewPostgresObserver(e.db, runID, e.logger)
	multiObs := NewMultiObserver(postgresObs, streamingObs)
	checkpointStore := NewPostgresCheckpointStore(e.db, e.logger)

	cfg := config.DefaultGraphConfig(runID.String())
	cfg.Checkpoint.Interval = 1
	cfg.Checkpoint.Preserve = true

	graph, err := state.NewGraphWithDeps(cfg, multiObs, checkpointStore)
	if err != nil {
		streamingObs.SendError(err, "")
		e.finalizeRun(ctx, runID, StatusFailed, nil, err)
		return
	}

	initialState, err := factory(execCtx, graph, e.runtime, params)
	if err != nil {
		streamingObs.SendError(err, "")
		e.finalizeRun(ctx, runID, StatusFailed, nil, err)
		return
	}

	initialState.RunID = runID.String()

	finalState, err := graph.Execute(execCtx, initialState)

	if err != nil {
		if execCtx.Err() != nil {
			errMsg := "execution cancelled"
			streamingObs.SendError(fmt.Errorf(errMsg), "")
			e.repo.UpdateRunCompleted(ctx, runID, StatusCancelled, nil, &errMsg)
			return
		}
		streamingObs.SendError(err, "")
		errMsg := err.Error()
		e.repo.UpdateRunCompleted(ctx, runID, StatusFailed, nil, &errMsg)
		return
	}

	streamingObs.SendComplete(finalState.Data)
	e.repo.UpdateRunCompleted(ctx, runID, StatusCompleted, finalState.Data, nil)
}
```

---

## Phase 3: Handler Implementation

### 3.1 Create handler.go

Create new file `internal/workflows/handler.go`:

```go
package workflows

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/routes"
	"github.com/JaimeStill/agent-lab/pkg/handlers"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

type Handler struct {
	sys        System
	logger     *slog.Logger
	pagination pagination.Config
}

func NewHandler(sys System, logger *slog.Logger, pagination pagination.Config) *Handler {
	return &Handler{
		sys:        sys,
		logger:     logger,
		pagination: pagination,
	}
}

func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix:      "/api/workflows",
		Tags:        []string{"Workflows"},
		Description: "Workflow execution and management",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.ListWorkflows, OpenAPI: Spec.ListWorkflows},
			{Method: "POST", Pattern: "/{name}/execute", Handler: h.Execute, OpenAPI: Spec.Execute},
			{Method: "POST", Pattern: "/{name}/execute/stream", Handler: h.ExecuteStream, OpenAPI: Spec.ExecuteStream},
		},
		Children: []routes.Group{
			{
				Prefix:      "/runs",
				Tags:        []string{"Runs"},
				Description: "Workflow run inspection and control",
				Routes: []routes.Route{
					{Method: "GET", Pattern: "", Handler: h.ListRuns, OpenAPI: Spec.ListRuns},
					{Method: "GET", Pattern: "/{id}", Handler: h.FindRun, OpenAPI: Spec.FindRun},
					{Method: "GET", Pattern: "/{id}/stages", Handler: h.GetStages, OpenAPI: Spec.GetStages},
					{Method: "GET", Pattern: "/{id}/decisions", Handler: h.GetDecisions, OpenAPI: Spec.GetDecisions},
					{Method: "POST", Pattern: "/{id}/cancel", Handler: h.Cancel, OpenAPI: Spec.Cancel},
					{Method: "POST", Pattern: "/{id}/resume", Handler: h.Resume, OpenAPI: Spec.Resume},
				},
			},
		},
	}
}

func (h *Handler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	workflows := h.sys.ListWorkflows()
	handlers.RespondJSON(w, http.StatusOK, workflows)
}

func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	run, err := h.sys.Execute(r.Context(), name, req.Params)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, run)
}

func (h *Handler) ExecuteStream(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	events, run, err := h.sys.ExecuteStream(r.Context(), name, req.Params)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Run-ID", run.ID.String())
	w.WriteHeader(http.StatusOK)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	for event := range events {
		select {
		case <-r.Context().Done():
			return
		default:
		}

		data, err := json.Marshal(event)
		if err != nil {
			h.logger.Error("failed to marshal event", "error", err)
			continue
		}

		fmt.Fprintf(w, "event: %s\n", event.Type)
		fmt.Fprintf(w, "data: %s\n\n", data)

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

func (h *Handler) ListRuns(w http.ResponseWriter, r *http.Request) {
	page := pagination.PageRequestFromQuery(r.URL.Query(), h.pagination)
	filters := RunFiltersFromQuery(r.URL.Query())

	result, err := h.sys.ListRuns(r.Context(), page, filters)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusInternalServerError, err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, result)
}

func (h *Handler) FindRun(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	run, err := h.sys.FindRun(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, run)
}

func (h *Handler) GetStages(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	stages, err := h.sys.GetStages(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, stages)
}

func (h *Handler) GetDecisions(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	decisions, err := h.sys.GetDecisions(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, decisions)
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	if err := h.sys.Cancel(r.Context(), id); err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Resume(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	run, err := h.sys.Resume(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, run)
}

type ExecuteRequest struct {
	Params map[string]any `json:"params,omitempty"`
}
```

---

## Phase 4: Route Registration

### 4.1 Update routes.go

Modify `cmd/server/routes.go`:

**Add import:**
```go
"github.com/JaimeStill/agent-lab/internal/workflows"
```

**Add handler registration after imagesHandler:**
```go
workflowHandler := workflows.NewHandler(
	domain.Workflows,
	runtime.Logger,
	runtime.Pagination,
)
r.RegisterGroup(workflowHandler.Routes())
```

**Add schema registration:**
```go
components.AddSchemas(workflows.Spec.Schemas())
```

---

## Validation Checklist

- [ ] `go vet ./...` passes
- [ ] All handlers respond correctly
- [ ] SSE stream emits proper event format
- [ ] Run filters work (workflow_name, status)
- [ ] Cancel stops active runs
- [ ] Resume continues from checkpoint
- [ ] OpenAPI spec generates correctly
