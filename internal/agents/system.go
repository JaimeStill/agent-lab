package agents

import (
	"context"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/go-agents/pkg/agent"
	"github.com/JaimeStill/go-agents/pkg/response"
	"github.com/google/uuid"
)

// System defines the interface for agent configuration management.
// Implementations handle persistence and validation of agent configs.
type System interface {
	Handler() *Handler

	// List returns a paginated list of agents matching the filter criteria.
	List(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Agent], error)

	// Find retrieves an agent configuration by ID.
	// Returns ErrNotFound if the agent does not exist.
	Find(ctx context.Context, id uuid.UUID) (*Agent, error)

	// Create validates and stores a new agent configuration.
	// Returns ErrDuplicate if an agent with the same name exists.
	// Returns ErrInvalidConfig if the configuration fails go-agents validation.
	Create(ctx context.Context, cmd CreateCommand) (*Agent, error)

	// Update modifies an existing agent configuration.
	// Returns ErrNotFound if the agent does not exist.
	// Returns ErrDuplicate if the new name conflicts with another agent.
	// Returns ErrInvalidConfig if the configuration fails go-agents validation.
	Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Agent, error)

	// Delete deletes an agent configuration by ID.
	// Returns ErrNotFound if the agent does not exist.
	Delete(ctx context.Context, id uuid.UUID) error

	// Chat executes a chat completion using the agent configuration.
	// The opts map supports "system_prompt" to override the stored prompt.
	// Token overrides the stored API token if provided.
	Chat(ctx context.Context, id uuid.UUID, prompt string, opts map[string]any, token string) (*response.ChatResponse, error)

	// ChatStream executes a streaming chat completion.
	// Returns a channel that receives chunks as they arrive.
	ChatStream(ctx context.Context, id uuid.UUID, prompt string, opts map[string]any, token string) (<-chan *response.StreamingChunk, error)

	// Vision executes a vision completion with image analysis.
	// Images should be base64-encoded data URIs.
	Vision(ctx context.Context, id uuid.UUID, prompt string, images []string, opts map[string]any, token string) (*response.ChatResponse, error)

	// VisionStream executes a streaming vision completion.
	VisionStream(ctx context.Context, id uuid.UUID, prompt string, images []string, opts map[string]any, token string) (<-chan *response.StreamingChunk, error)

	// Tools executes a tool-use completion with function calling.
	Tools(ctx context.Context, id uuid.UUID, prompt string, tools []agent.Tool, opts map[string]any, token string) (*response.ToolsResponse, error)

	// Embed generates embeddings for the input text.
	Embed(ctx context.Context, id uuid.UUID, input string, opts map[string]any, token string) (*response.EmbeddingsResponse, error)
}
