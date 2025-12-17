package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/agent-lab/pkg/repository"
	"github.com/JaimeStill/go-agents/pkg/agent"
	agtconfig "github.com/JaimeStill/go-agents/pkg/config"
	"github.com/google/uuid"
)

type repo struct {
	db         *sql.DB
	logger     *slog.Logger
	pagination pagination.Config
}

// New creates a new agents repository implementing the System interface.
func New(db *sql.DB, logger *slog.Logger, pagination pagination.Config) System {
	return &repo{
		db:         db,
		logger:     logger.With("system", "agent"),
		pagination: pagination,
	}
}

func (r *repo) List(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Agent], error) {
	page.Normalize(r.pagination)

	qb := query.
		NewBuilder(projection, defaultSort).
		WhereSearch(page.Search, "Name")

	filters.Apply(qb)

	if len(page.Sort) > 0 {
		qb.OrderByFields(page.Sort)
	}

	countSql, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRowContext(ctx, countSql, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count agents: %w", err)
	}

	pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	agents, err := repository.QueryMany(ctx, r.db, pageSQL, pageArgs, scanAgent)
	if err != nil {
		return nil, fmt.Errorf("query agents: %w", err)
	}

	result := pagination.NewPageResult(agents, total, page.Page, page.PageSize)
	return &result, nil
}

func (r *repo) Find(ctx context.Context, id uuid.UUID) (*Agent, error) {
	q, args := query.NewBuilder(projection).BuildSingle("ID", id)

	a, err := repository.QueryOne(ctx, r.db, q, args, scanAgent)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}
	return &a, nil
}

func (r *repo) Create(ctx context.Context, cmd CreateCommand) (*Agent, error) {
	if err := r.validateConfig(cmd.Config); err != nil {
		return nil, err
	}

	q := `
		INSERT INTO agents (name, config)
		VALUES ($1, $2)
		RETURNING id, name, config, created_at, updated_at`

	a, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Agent, error) {
		return repository.QueryOne(ctx, tx, q, []any{cmd.Name, cmd.Config}, scanAgent)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("agent created", "id", a.ID, "name", a.Name)
	return &a, nil
}

func (r *repo) Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Agent, error) {
	if err := r.validateConfig(cmd.Config); err != nil {
		return nil, err
	}

	q := `
		UPDATE agents
		SET name = $1, config = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, name, config, created_at, updated_at`

	a, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Agent, error) {
		return repository.QueryOne(ctx, tx, q, []any{cmd.Name, cmd.Config, id}, scanAgent)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("agent updated", "id", a.ID, "name", a.Name)
	return &a, nil
}

func (r *repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (struct{}, error) {
		err := repository.ExecExpectOne(ctx, tx, "DELETE FROM agents WHERE id = $1", id)
		return struct{}{}, err
	})

	if err != nil {
		return repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("agent deleted", "id", id)
	return nil
}

func (r *repo) validateConfig(config json.RawMessage) error {
	cfg := agtconfig.DefaultAgentConfig()

	var userCfg agtconfig.AgentConfig
	if err := json.Unmarshal(config, &userCfg); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	cfg.Merge(&userCfg)

	if _, err := agent.New(&cfg); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	return nil
}
