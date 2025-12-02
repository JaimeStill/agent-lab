package agents

import (
	"context"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

// System defines the interface for agent storage and retrieval operations.
type System interface {
	Create(ctx context.Context, cmd CreateCommand) (*Agent, error)
	Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Agent, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*Agent, error)
	Search(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Agent], error)
}
