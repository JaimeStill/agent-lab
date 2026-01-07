package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"github.com/JaimeStill/agent-lab/internal/profiles"
	"github.com/google/uuid"
)

func init() {
	registerSeeder(&ProfileSeeder{})
}

// ProfileSeedData represents the JSON structure for profile seed files.
type ProfileSeedData struct {
	Profiles []profiles.ProfileWithStages `json:"profiles"`
}

// ProfileSeeder implements Seeder for workflow profiles and their stages.
// It loads seed data from an embedded file or an external file path.
type ProfileSeeder struct {
	file string
}

// Name returns "profiles" as the seeder identifier.
func (s *ProfileSeeder) Name() string {
	return "profiles"
}

// Description returns a human-readable description of this seeder.
func (s *ProfileSeeder) Description() string {
	return "Seeds workflow profiles and their stage configurations"
}

// SetFile configures an external seed file path, overriding the embedded default.
func (s *ProfileSeeder) SetFile(path string) {
	s.file = path
}

// Seed loads profile data and saves profiles and stages to the database.
// Uses save semantics (insert or update) for idempotent execution.
func (s *ProfileSeeder) Seed(ctx context.Context, tx *sql.Tx) error {
	data, err := s.loadSeedData()
	if err != nil {
		return err
	}

	for _, p := range data.Profiles {
		profileID, err := s.saveProfile(ctx, tx, p.Profile)
		if err != nil {
			return fmt.Errorf("save profile %s/%s: %w", p.WorkflowName, p.Name, err)
		}

		for _, stage := range p.Stages {
			if err := s.saveStage(ctx, tx, profileID, stage); err != nil {
				return fmt.Errorf("save stage %s for profile %s: %w", stage.StageName, p.Name, err)
			}
		}
	}

	return nil
}

func (s *ProfileSeeder) loadSeedData() (*ProfileSeedData, error) {
	var content []byte
	var err error

	if s.file != "" {
		content, err = os.ReadFile(s.file)
		if err != nil {
			return nil, fmt.Errorf("read seed file: %w", err)
		}
	} else {
		content, err = seedFiles.ReadFile("seeds/classify_profiles.json")
		if err != nil {
			return nil, fmt.Errorf("read embedded seed file: %w", err)
		}
	}

	var data ProfileSeedData
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("parse seed data: %w", err)
	}

	return &data, nil
}

func (s *ProfileSeeder) saveProfile(ctx context.Context, tx *sql.Tx, p profiles.Profile) (uuid.UUID, error) {
	const query = `
		INSERT INTO profiles (id, workflow_name, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (workflow_name, name) DO UPDATE SET
			description = EXCLUDED.description,
			updated_at = NOW()
		RETURNING id`

	id := uuid.New()
	var returnedID uuid.UUID
	err := tx.QueryRowContext(ctx, query, id, p.WorkflowName, p.Name, p.Description).Scan(&returnedID)
	if err != nil {
		return uuid.Nil, err
	}

	return returnedID, nil
}

func (s *ProfileSeeder) saveStage(ctx context.Context, tx *sql.Tx, profileID uuid.UUID, stage profiles.ProfileStage) error {
	const query = `
		INSERT INTO profile_stages (profile_id, stage_name, agent_id, system_prompt, options)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (profile_id, stage_name) DO UPDATE SET
			agent_id = EXCLUDED.agent_id,
			system_prompt = EXCLUDED.system_prompt,
			options = EXCLUDED.options`

	_, err := tx.ExecContext(ctx, query, profileID, stage.StageName, stage.AgentID, stage.SystemPrompt, stage.Options)
	return err
}
