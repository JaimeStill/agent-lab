package providers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/agent-lab/pkg/repository"
	agtconfig "github.com/JaimeStill/go-agents/pkg/config"
	agtproviders "github.com/JaimeStill/go-agents/pkg/providers"
	"github.com/google/uuid"
)

type repo struct {
	db         *sql.DB
	logger     *slog.Logger
	pagination pagination.Config
}

// New creates a new providers repository with the given dependencies.
func New(db *sql.DB, logger *slog.Logger, pagination pagination.Config) System {
	return &repo{
		db:         db,
		logger:     logger.With("system", "provider"),
		pagination: pagination,
	}
}

func (r *repo) List(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Provider], error) {
	page.Normalize(r.pagination)

	qb := query.NewBuilder(projection, defaultSort).
		WhereSearch(page.Search, "Name")

	filters.Apply(qb)

	if len(page.Sort) > 0 {
		qb.OrderByFields(page.Sort)
	}

	countSql, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRowContext(ctx, countSql, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count providers: %w", err)
	}

	pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	providers, err := repository.QueryMany(ctx, r.db, pageSQL, pageArgs, scanProvider)
	if err != nil {
		return nil, fmt.Errorf("query providers: %w", err)
	}

	result := pagination.NewPageResult(providers, total, page.Page, page.PageSize)
	return &result, nil
}

func (r *repo) Find(ctx context.Context, id uuid.UUID) (*Provider, error) {
	q, args := query.NewBuilder(projection).BuildSingle("ID", id)

	p, err := repository.QueryOne(ctx, r.db, q, args, scanProvider)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	return &p, nil
}

func (r *repo) Create(ctx context.Context, cmd CreateCommand) (*Provider, error) {
	if err := r.validateConfig(cmd.Config); err != nil {
		return nil, err
	}

	q := `
		INSERT INTO providers (name, config)
		VALUES ($1, $2)
		RETURNING id, name, config, created_at, updated_at`

	p, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Provider, error) {
		return repository.QueryOne(ctx, tx, q, []any{cmd.Name, cmd.Config}, scanProvider)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("provider created", "id", p.ID, "name", p.Name)
	return &p, nil
}

func (r *repo) Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Provider, error) {
	if err := r.validateConfig(cmd.Config); err != nil {
		return nil, err
	}

	q := `
		UPDATE providers
		SET name = $1, config = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, name, config, created_at, updated_at`

	p, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Provider, error) {
		return repository.QueryOne(ctx, tx, q, []any{cmd.Name, cmd.Config, id}, scanProvider)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("provider updated", "id", p.ID, "name", p.Name)
	return &p, nil
}

func (r *repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (struct{}, error) {
		err := repository.ExecExpectOne(ctx, tx, "DELETE FROM providers WHERE id = $1", id)
		return struct{}{}, err
	})

	if err != nil {
		return repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("provider deleted", "id", id)
	return nil
}

func (r *repo) validateConfig(config json.RawMessage) error {
	var cfg agtconfig.ProviderConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	if _, err := agtproviders.Create(&cfg); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	return nil
}
