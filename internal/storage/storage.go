package storage

import (
	"context"

	"github.com/JaimeStill/agent-lab/internal/lifecycle"
)

// System defines the storage operations interface for blob storage.
// Implementations handle the underlying storage mechanism (filesystem, cloud, etc.)
// while providing a consistent API for storing and retrieving binary data.
type System interface {
	// Store saves data at the specified key. If the key already exists,
	// its contents are overwritten. Parent directories are created as needed.
	// Returns ErrInvalidKey if the key is empty or contains path traversal.
	Store(ctx context.Context, key string, data []byte) error

	// Retrieve returns the data stored at the specified key.
	// Returns ErrNotFound if the key does not exist.
	// Returns ErrInvalidKey if the key is malformed.
	Retrieve(ctx context.Context, key string) ([]byte, error)

	// Delete deletes the data at the specified key.
	// Returns nil if the key does not exist (idempotent).
	// Returns ErrInvalidKey if the key is malformed.
	Delete(ctx context.Context, key string) error

	// Validate checks if a key exists and is accessible.
	// Returns (true, nil) if the key exists and is readable.
	// Returns (false, nil) if the key does not exist.
	// Returns (false, error) for permission or system errors.
	Validate(ctx context.Context, key string) (bool, error)

	// Start registers lifecycle hooks with the coordinator.
	// For filesystem storage, this creates the base directory.
	Start(lc *lifecycle.Coordinator) error

	Path(ctx context.Context, key string) (string, error)
}
