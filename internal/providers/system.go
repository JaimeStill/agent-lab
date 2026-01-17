// Package providers implements LLM provider configuration management.
// It provides CRUD operations for storing and validating provider configurations
// that integrate with the go-agents library.
package providers

import (
	"context"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

// System defines the interface for provider configuration management.
// Implementations handle persistence and validation of provider configs.
type System interface {
	Handler() *Handler

	// List returns a paginated list of providers matching the filter criteria.
	List(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Provider], error)

	// Find retrieves a provider configuration by ID.
	// Returns ErrNotFound if the provider does not exist.
	Find(ctx context.Context, id uuid.UUID) (*Provider, error)

	// Create validates and stores a new provider configuration.
	// Returns ErrDuplicate if a provider with the same name exists.
	// Returns ErrInvalidConfig if the configuration fails go-agents validation.
	Create(ctx context.Context, cmd CreateCommand) (*Provider, error)

	// Update modifies an existing provider configuration.
	// Returns ErrNotFound if the provider does not exist.
	// Returns ErrDuplicate if the new name conflicts with another provider.
	// Returns ErrInvalidConfig if the configuration fails go-agents validation.
	Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Provider, error)

	// Delete deletes a provider configuration by ID.
	// Returns ErrNotFound if the provider does not exist.
	Delete(ctx context.Context, id uuid.UUID) error
}
