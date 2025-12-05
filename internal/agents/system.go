package agents

import (
	"context"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

// System defines the interface for agent configuration management.
// Implementations handle persistence and validation of agent configs.
type System interface {
	// Create validates and stores a new agent configuration.
	// Returns ErrDuplicate if an agent with the same name exists.
	// Returns ErrInvalidConfig if the configuration fails go-agents validation.
	Create(ctx context.Context, cmd CreateCommand) (*Agent, error)

	// Update modifies an existing agent configuration.
	// Returns ErrNotFound if the agent does not exist.
	// Returns ErrDuplicate if the new name conflicts with another agent.
	// Returns ErrInvalidConfig if the configuration fails go-agents validation.
	Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Agent, error)

	// Delete removes an agent configuration by ID.
	// Returns ErrNotFound if the agent does not exist.
	Delete(ctx context.Context, id uuid.UUID) error

	// GetByID retrieves an agent configuration by ID.
	// Returns ErrNotFound if the agent does not exist.
	GetByID(ctx context.Context, id uuid.UUID) (*Agent, error)

	// Search returns a paginated list of agents matching the search criteria.
	Search(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Agent], error)
}
