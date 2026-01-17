package documents

import (
	"context"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

// System defines the document management operations.
// Implementations handle blob storage and database persistence.
type System interface {
	Handler(maxUploadSize int64) *Handler
	List(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Document], error)
	Find(ctx context.Context, id uuid.UUID) (*Document, error)
	Create(ctx context.Context, cmd CreateCommand) (*Document, error)
	Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Document, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
