// Package documents provides document upload, storage, and management functionality.
// It supports PDF metadata extraction and integrates with blob storage for file persistence.
package documents

import (
	"time"

	"github.com/google/uuid"
)

// Document represents a stored document with metadata.
type Document struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	PageCount   *int      `json:"page_count,omitempty"`
	StorageKey  string    `json:"storage_key"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateCommand contains the data required to create a new document.
// Data holds the raw file bytes to be stored.
type CreateCommand struct {
	Name        string
	Filename    string
	ContentType string
	SizeBytes   int64
	PageCount   *int
	Data        []byte
}

// UpdateCommand contains the fields that can be modified on an existing document.
// Only the display name can be changed; the stored file is immutable.
type UpdateCommand struct {
	Name string
}
