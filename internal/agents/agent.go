// Package agents provides the domain system for managing AI agent configurations
// and executing agent capabilities (chat, vision, tools, embeddings).
package agents

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Agent represents an AI agent configuration stored in the database.
type Agent struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Config    json.RawMessage `json:"config"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// CreateCommand contains the data required to create a new agent.
type CreateCommand struct {
	Name   string          `json:"name"`
	Config json.RawMessage `json:"config"`
}

// UpdateCommand contains the data required to update an existing agent.
type UpdateCommand struct {
	Name   string          `json:"name"`
	Config json.RawMessage `json:"config"`
}
