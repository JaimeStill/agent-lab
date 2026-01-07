package workflows

import (
	"context"
	"database/sql"
	"encoding/json"
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

// New creates a new workflows repository.
func New(db *sql.DB, logger *slog.Logger, pagination pagination.Config) *repo {
	return &repo{
		db:         db,
		logger:     logger.With("system", "workflows"),
		pagination: pagination,
	}
}

// ListRuns returns a paginated list of workflow runs.
func (r *repo) ListRuns(ctx context.Context, page pagination.PageRequest, filters RunFilters) (*pagination.PageResult[Run], error) {
	page.Normalize(r.pagination)

	qb := query.NewBuilder(runProjection, runDefaultSort)
	filters.Apply(qb)

	if len(page.Sort) > 0 {
		qb.OrderByFields(page.Sort)
	}

	countSql, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRowContext(ctx, countSql, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count runs: %w", err)
	}

	pageSql, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	runs, err := repository.QueryMany(ctx, r.db, pageSql, pageArgs, scanRun)
	if err != nil {
		return nil, fmt.Errorf("query runs: %w", err)
	}

	result := pagination.NewPageResult(runs, total, page.Page, page.PageSize)
	return &result, nil
}

// FindRun retrieves a single run by ID.
func (r *repo) FindRun(ctx context.Context, id uuid.UUID) (*Run, error) {
	q, args := query.NewBuilder(runProjection).BuildSingle("ID", id)

	run, err := repository.QueryOne(ctx, r.db, q, args, scanRun)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, nil)
	}

	return &run, nil
}

// CreateRun inserts a new workflow run with pending status.
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
		RETURNING id, workflow_name, status, params, result, error_message, started_at, completed_at, created_at, updated_at
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

// UpdateRunStarted transitions a run to running status and sets started_at.
func (r *repo) UpdateRunStarted(ctx context.Context, id uuid.UUID) (*Run, error) {
	const q = `
		UPDATE runs
		SET status = $1, started_at = NOW(), updated_at = NOW()
		WHERE id = $2
		RETURNING id, workflow_name, status, params, result, error_message, started_at, completed_at, created_at, updated_at
	`

	run, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Run, error) {
		return repository.QueryOne(ctx, tx, q, []any{StatusRunning, id}, scanRun)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, nil)
	}

	return &run, nil
}

// UpdateRunCompleted sets the final status, result, and completion time for a run.
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
		SET status = $1, result = $2, error_message = $3, completed_at = NOW(), updated_at = NOW()
		WHERE id = $4
		RETURNING id, workflow_name, status, params, result, error_message, started_at, completed_at, created_at, updated_at
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

// GetStages retrieves all stages for a workflow run.
func (r *repo) GetStages(ctx context.Context, runID uuid.UUID) ([]Stage, error) {
	qb := query.NewBuilder(stageProjection, stageDefaultSort)
	qb.WhereEquals("RunID", &runID)

	q, args := qb.Build()

	stages, err := repository.QueryMany(ctx, r.db, q, args, scanStage)
	if err != nil {
		return nil, fmt.Errorf("query stages: %w", err)
	}

	return stages, nil
}

// GetDecisions retrieves all routing decisions for a workflow run.
func (r *repo) GetDecisions(ctx context.Context, runID uuid.UUID) ([]Decision, error) {
	qb := query.NewBuilder(decisionProjection, decisionDefaultSort)
	qb.WhereEquals("RunID", &runID)

	q, args := qb.Build()

	decisions, err := repository.QueryMany(ctx, r.db, q, args, scanDecision)
	if err != nil {
		return nil, fmt.Errorf("query decisions: %w", err)
	}

	return decisions, nil
}

// DeleteRun deletes a workflow run and its related data (stages, decisions, checkpoints).
func (r *repo) DeleteRun(ctx context.Context, id uuid.UUID) error {
	_, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (struct{}, error) {
		err := repository.ExecExpectOne(ctx, tx, "DELETE FROM runs WHERE id = $1", id)
		return struct{}{}, err
	})

	if err != nil {
		return repository.MapError(err, ErrNotFound, nil)
	}

	r.logger.Info("run deleted", "id", id)
	return nil
}
