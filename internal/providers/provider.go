package providers

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Provider represents an LLM provider configuration stored in the database.
// The Config field contains the go-agents ProviderConfig structure as JSON.
type Provider struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Config    json.RawMessage `json:"config"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// CreateCommand contains the data required to create a new provider.
type CreateCommand struct {
	Name   string          `json:"name"`
	Config json.RawMessage `json:"config"`
}

// UpdateCommand contains the data required to update an existing provider.
type UpdateCommand struct {
	Name   string          `json:"name"`
	Config json.RawMessage `json:"config"`
}
