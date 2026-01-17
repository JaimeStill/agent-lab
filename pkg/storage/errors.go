// Package storage provides blob storage abstractions for the agent-lab service.
// It defines a System interface for storage operations and includes a filesystem
// implementation suitable for development and single-node deployments.
package storage

import "errors"

// Storage errors returned by System implementations.
var (
	// ErrNotFound indicates the requested key does not exist in storage.
	ErrNotFound = errors.New("storage: key not found")

	// ErrPermissionDenied indicates insufficient permissions to access the key.
	ErrPermissionDenied = errors.New("storage: permission denied")

	// ErrInvalidKey indicates the key is malformed or contains invalid characters.
	// This includes empty keys and path traversal attempts.
	ErrInvalidKey = errors.New("storage: invalid key")
)
