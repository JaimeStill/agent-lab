package workflows

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/agent-lab/pkg/query"
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

	countBuilder := query.NewBuilder(runProjection)
	filters.Apply(countBuilder)

	countQuery, countArgs := countBuilder.BuildCount()

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count runs: %w", err)
	}

	pageBuilder := query.NewBuilder(runProjection)
	filters.Apply(pageBuilder)
	pageBuilder.OrderBy(runDefaultSort)

	pageQuery, pageArgs := pageBuilder.BuildPage(page)

	rows, err := r.db.QueryContext(ctx, pageQuery, pageArgs...)
	if err != nil {
		return nil, fmt.Errorf("query runs: %w", err)
	}
	defer rows.Close()

	var runs []Run
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan run: %w", err)
		}
		runs = append(runs, run)
	}
}
