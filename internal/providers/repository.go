package providers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/agent-lab/pkg/query"
	agtconfig "github.com/JaimeStill/go-agents/pkg/config"
	agtproviders "github.com/JaimeStill/go-agents/pkg/providers"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type repository struct {
	db         *sql.DB
	logger     *slog.Logger
	pagination pagination.Config
}

// New creates a new providers repository with the given dependencies.
func New(db *sql.DB, logger *slog.Logger, pagination pagination.Config) System {
	return &repository{
		db:         db,
		logger:     logger.With("system", "provider"),
		pagination: pagination,
	}
}

func (r *repository) Create(ctx context.Context, cmd CreateCommand) (*Provider, error) {
	if err := r.validateConfig(cmd.Config); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	q := `
		INSERT INTO providers (name, config)
		VALUES ($1, $2)
		RETURNING id, name, config, created_at, updated_at`

	var p Provider
	err = tx.
		QueryRowContext(ctx, q, cmd.Name, cmd.Config).
		Scan(
			&p.ID, &p.Name, &p.Config, &p.CreatedAt, &p.UpdatedAt,
		)
	if err != nil {
		if isDuplicateError(err) {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("insert provider: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	r.logger.Info("provider created", "id", p.ID, "name", p.Name)
	return &p, nil
}

func (r *repository) Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Provider, error) {
	if err := r.validateConfig(cmd.Config); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	q := `
		UPDATE providers
		SET name = $1, config = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, name, config, created_at, updated_at`

	var p Provider
	err = tx.
		QueryRowContext(ctx, q, cmd.Name, cmd.Config, id).
		Scan(
			&p.ID, &p.Name, &p.Config, &p.CreatedAt, &p.UpdatedAt,
		)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		if isDuplicateError(err) {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("update provider: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	r.logger.Info("provider updated", "id", p.ID, "name", p.Name)
	return &p, nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, "DELETE FROM providers WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete provider: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	r.logger.Info("provider deleted", "id", id)
	return nil
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*Provider, error) {
	q, args := query.NewBuilder(projection, "Name").BuildSingle("Id", id)

	var p Provider
	err := r.db.QueryRowContext(ctx, q, args...).Scan(
		&p.ID, &p.Name, &p.Config, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query provider: %w", err)
	}
	return &p, nil
}

func (r *repository) Search(ctx context.Context, page pagination.PageRequest) (*pagination.PageResult[Provider], error) {
	page.Normalize(r.pagination)

	qb := query.NewBuilder(projection, "Name").
		WhereSearch(page.Search, "Name").
		OrderBy(page.SortBy, page.Descending)

	countSql, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRowContext(ctx, countSql, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count providers: %w", err)
	}

	pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	rows, err := r.db.QueryContext(ctx, pageSQL, pageArgs...)
	if err != nil {
		return nil, fmt.Errorf("query providers: %w", err)
	}
	defer rows.Close()

	providers := make([]Provider, 0)
	for rows.Next() {
		var p Provider
		if err := rows.Scan(&p.ID, &p.Name, &p.Config, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan provider: %w", err)
		}
		providers = append(providers, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	result := pagination.NewPageResult(providers, total, page.Page, page.PageSize)
	return &result, nil
}

func (r *repository) validateConfig(config json.RawMessage) error {
	var cfg agtconfig.ProviderConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	if _, err := agtproviders.Create(&cfg); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	return nil
}

func isDuplicateError(err error) bool {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		return pgErr.Code == "23505"
	}
	return false
}
