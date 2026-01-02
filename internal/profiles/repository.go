package profiles

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

func New(db *sql.DB, logger *slog.Logger, pagination pagination.Config) System {
	return &repo{
		db:         db,
		logger:     logger.With("system", "profiles"),
		pagination: pagination,
	}
}

func (r *repo) List(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Profile], error) {
	page.Normalize(r.pagination)

	qb := query.
		NewBuilder(profileProjection, defaultSort).
		WhereSearch(page.Search, "Name")

	filters.Apply(qb)

	if len(page.Sort) > 0 {
		qb.OrderByFields(page.Sort)
	}

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count profiles: %w", err)
	}

	pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	profiles, err := repository.QueryMany(ctx, r.db, pageSQL, pageArgs, scanProfile)
	if err != nil {
		return nil, fmt.Errorf("query profiles: %w", err)
	}

	result := pagination.NewPageResult(profiles, total, page.Page, page.PageSize)
	return &result, nil
}

func (r *repo) Find(ctx context.Context, id uuid.UUID) (*ProfileWithStages, error) {
	profileSQL, profileArgs := query.NewBuilder(profileProjection, defaultSort).
		WhereEquals("ID", &id).
		Build()

	profile, err := repository.QueryOne(ctx, r.db, profileSQL, profileArgs, scanProfile)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	stageSQL, stageArgs := query.NewBuilder(stageProjection, query.SortField{Field: "StageName"}).
		WhereEquals("ProfileID", &id).
		Build()

	stages, err := repository.QueryMany(ctx, r.db, stageSQL, stageArgs, scanProfileStage)
	if err != nil {
		return nil, fmt.Errorf("query stages: %w", err)
	}

	return &ProfileWithStages{
		Profile: profile,
		Stages:  stages,
	}, nil
}

func (r *repo) Create(ctx context.Context, cmd CreateProfileCommand) (*Profile, error) {
	q := `
		INSERT INTO profiles(workflow_name, name, description)
		VALUES ($1, $2, $3)
		RETURNING id, workflow_name, name, description, created_at, updated_at`

	profile, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Profile, error) {
		return repository.QueryOne(ctx, tx, q, []any{cmd.WorkflowName, cmd.Name, cmd.Description}, scanProfile)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("profile created", "id", profile.ID, "workflow", profile.WorkflowName, "name", profile.Name)
	return &profile, nil
}

func (r *repo) Update(ctx context.Context, id uuid.UUID, cmd UpdateProfileCommand) (*Profile, error) {
	q := `
		UPDATE profiles
		SET name = $2, description = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING id, workflow_name, name, description, created_at, updated_at`

	profile, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Profile, error) {
		return repository.QueryOne(ctx, tx, q, []any{id, cmd.Name, cmd.Description}, scanProfile)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("profile updated", "id", profile.ID, "name", profile.Name)
	return &profile, nil
}

func (r *repo) Delete(ctx context.Context, id uuid.UUID) error {
	q := `DELETE FROM profiles WHERE id = $1`

	if err := repository.ExecExpectOne(ctx, r.db, q, id); err != nil {
		return repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("profile deleted", "id", id)
	return nil
}

func (r *repo) SetStage(ctx context.Context, profileID uuid.UUID, cmd SetProfileStageCommand) (*ProfileStage, error) {
	if _, err := r.Find(ctx, profileID); err != nil {
		return nil, err
	}

	q := `
		INSERT INTO profile_stages (profile_id, stage_name, agent_id, system_prompt, options)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (profile_id, stage_name)
		DO UPDATE SET agent_id = $3, system_prompt = $4, options = $5
		RETURNING profile_id, stage_name, agent_id, system_prompt, options`

	stage, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (ProfileStage, error) {
		return repository.QueryOne(ctx, tx, q, []any{
			profileID, cmd.StageName, cmd.AgentID, cmd.SystemPrompt, cmd.Options,
		}, scanProfileStage)
	})

	if err != nil {
		return nil, fmt.Errorf("set stage: %w", err)
	}

	r.logger.Info("stage set", "profile_id", profileID, "stage", cmd.StageName)
	return &stage, nil
}

func (r *repo) DeleteStage(ctx context.Context, profileID uuid.UUID, stageName string) error {
	q := `DELETE FROM profile_stages WHERE profile_id = $1 AND stage_name = $2`

	if err := repository.ExecExpectOne(ctx, r.db, q, profileID, stageName); err != nil {
		return repository.MapError(err, ErrStageNotFound, ErrDuplicate)
	}

	r.logger.Info("stage deleted", "profile_id", profileID, "stage", stageName)
	return nil
}
