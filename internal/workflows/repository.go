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
