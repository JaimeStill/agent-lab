// Package profiles provides workflow profile management for configuring
// stage-level agent and prompt settings. Profiles enable A/B testing and
// prompt iteration without code changes.
package profiles

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Profile represents a named configuration set for a workflow.
// Multiple profiles can exist for the same workflow, enabling
// experimentation with different agent and prompt configurations.
type Profile struct {
	ID           uuid.UUID `json:"id"`
	WorkflowName string    `json:"workflow_name"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ProfileStage configures a single stage within a workflow profile.
// Each stage can override the default agent and system prompt.
type ProfileStage struct {
	ProfileID    uuid.UUID       `json:"profile_id"`
	StageName    string          `json:"stage_name"`
	AgentID      *uuid.UUID      `json:"agent_id,omitempty"`
	SystemPrompt *string         `json:"system_prompt,omitempty"`
	Options      json.RawMessage `json:"options,omitempty"`
}

// ProfileWithStages combines a profile with all its stage configurations.
// This is the primary type used when loading profiles for workflow execution.
type ProfileWithStages struct {
	Profile
	Stages []ProfileStage `json:"stages"`
}

// NewProfileWithStages creates a ProfileWithStages with the given stage configurations.
// This is typically used to define hardcoded default profiles in workflow packages.
func NewProfileWithStages(stages ...ProfileStage) *ProfileWithStages {
	return &ProfileWithStages{
		Stages: stages,
	}
}

// Stage returns the configuration for the named stage, or nil if not found.
// The returned pointer references the actual slice element, allowing modification.
func (p *ProfileWithStages) Stage(name string) *ProfileStage {
	for i, s := range p.Stages {
		if s.StageName == name {
			return &p.Stages[i]
		}
	}
	return nil
}

// CreateProfileCommand contains the data needed to create a new profile.
type CreateProfileCommand struct {
	WorkflowName string  `json:"workflow_name"`
	Name         string  `json:"name"`
	Description  *string `json:"description,omitempty"`
}

// UpdateProfileCommand contains the data needed to update profile metadata.
type UpdateProfileCommand struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// SetProfileStageCommand contains the data needed to create or update a stage configuration.
// Uses upsert semantics - creates if not exists, updates if exists.
type SetProfileStageCommand struct {
	StageName    string          `json:"stage_name"`
	AgentID      *uuid.UUID      `json:"agent_id,omitempty"`
	SystemPrompt *string         `json:"system_prompt,omitempty"`
	Options      json.RawMessage `json:"options,omitempty"`
}
