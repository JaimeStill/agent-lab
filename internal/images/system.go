package images

import (
	"context"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

// System defines the interface for image management operations.
type System interface {
	Handler() *Handler

	// List returns a paginated list of images matching the provided filters.
	List(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Image], error)

	// Find retrieves an image record by its ID.
	Find(ctx context.Context, id uuid.UUID) (*Image, error)

	// Data retrieves the raw image bytes and content type for an image.
	Data(ctx context.Context, id uuid.UUID) ([]byte, string, error)

	// Render creates images from document pages based on the provided options.
	// Returns the created Image records for all rendered pages.
	Render(ctx context.Context, documentID uuid.UUID, cmd RenderOptions) ([]Image, error)

	// Delete deletes an image from storage and the database.
	Delete(ctx context.Context, id uuid.UUID) error
}
