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
	repo       *repo
	runtime    *Runtime
	db         *sql.DB
	logger     *slog.Logger
	activeRuns map[uuid.UUID]context.CancelFunc
	mu         sync.RWMutex
}

// NewSystem creates a new workflows System with the provided dependencies.
// The System handles workflow execution, cancellation, and resumption.
func NewSystem(
	runtime *Runtime,
	db *sql.DB,
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
	e.mu.RLock()
	cancel, exists := e.activeRuns[runID]
	e.mu.RUnlock()

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
	e.mu.Lock()
	defer e.mu.Unlock()
	e.activeRuns[id] = cancel
}

func (e *executor) untrackRun(id uuid.UUID) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.activeRuns, id)
}

func (e *executor) finalizeRun(ctx context.Context, id uuid.UUID, status RunStatus, result map[string]any, err error) (*Run, error) {
	errMsg := err.Error()
	run, updateErr := e.repo.UpdateRunCompleted(ctx, id, status, result, &errMsg)
	if updateErr != nil {
		e.logger.Error("failed to finalize run", "id", id, "error", updateErr)
	}
	return run, err
}
