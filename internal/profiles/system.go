package profiles

import (
	"context"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

// System defines the interface for profile management operations.
type System interface {
	// List returns a paginated list of profiles with optional filtering.
	List(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Profile], error)

	// Find returns a profile with all its stage configurations.
	Find(ctx context.Context, id uuid.UUID) (*ProfileWithStages, error)

	// Create creates a new profile.
	Create(ctx context.Context, cmd CreateProfileCommand) (*Profile, error)

	// Update updates profile metadata (name, description).
	Update(ctx context.Context, id uuid.UUID, cmd UpdateProfileCommand) (*Profile, error)

	// Delete removes a profile and all its stage configurations.
	Delete(ctx context.Context, id uuid.UUID) error

	// SetStage creates or updates a stage configuration (upsert).
	SetStage(ctx context.Context, profileID uuid.UUID, cmd SetProfileStageCommand) (*ProfileStage, error)

	// DeleteStage removes a stage configuration from a profile.
	DeleteStage(ctx context.Context, profileID uuid.UUID, stageName string) error
}
