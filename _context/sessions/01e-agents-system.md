# Session 01e: Agents System - Summary

**Session**: 01e
**Milestone**: 01 - Foundation & Infrastructure
**Status**: Complete

## What Was Implemented

### Database Schema
- `agents` table with UUID primary key, unique name, JSONB config, timestamps
- Indexes on `name` and `created_at`

### Domain Package (`internal/agents/`)
| File | Purpose |
|------|---------|
| `agent.go` | Entity, CreateCommand, UpdateCommand |
| `errors.go` | ErrNotFound, ErrDuplicate, ErrInvalidConfig, ErrExecution + MapHTTPStatus |
| `projection.go` | query.ProjectionMap for agents table |
| `filters.go` | Filters struct, FiltersFromQuery, Apply |
| `scanner.go` | scanAgent(repository.Scanner) |
| `system.go` | System interface |
| `repository.go` | repo struct implementing System |
| `requests.go` | ChatRequest, ToolsRequest, EmbedRequest, VisionForm, ParseVisionForm |
| `handler.go` | Handler struct with CRUD + execution methods |

### API Endpoints

**CRUD**:
- `POST /api/agents` - Create agent
- `GET /api/agents` - List agents (paginated)
- `GET /api/agents/{id}` - Get by ID
- `PUT /api/agents/{id}` - Update agent
- `DELETE /api/agents/{id}` - Delete agent
- `POST /api/agents/search` - Search (POST body)

**Execution**:
- `POST /api/agents/{id}/chat` - Chat (JSON)
- `POST /api/agents/{id}/chat/stream` - Chat (SSE)
- `POST /api/agents/{id}/vision` - Vision (multipart/form-data)
- `POST /api/agents/{id}/vision/stream` - Vision (multipart/form-data, SSE)
- `POST /api/agents/{id}/tools` - Tools (JSON)
- `POST /api/agents/{id}/embed` - Embeddings (JSON)

### Key Features
- **Token Authentication**: Optional `token` field in requests for Azure API key/bearer token injection at runtime
- **Vision File Uploads**: Multipart/form-data with automatic base64 conversion
- **SSE Streaming**: Server-Sent Events for chat and vision streaming endpoints
- **Config Validation**: Agent configs validated by constructing go-agents Agent during create/update

## Key Decisions

1. **Config Storage**: Full embedded `AgentConfig` as JSONB, decoupled from providers table
2. **Agent Construction**: Ephemeral per-request via `agent.New(cfg)` - no caching
3. **Token Injection**: Token passed at request time, injected into `cfg.Provider.Options["token"]`
4. **Vision Form Pattern**: `VisionForm` struct with `ParseVisionForm` function for clean multipart handling
5. **Placeholder Token**: Azure configs require `"token": "token"` placeholder for validation

## Patterns Established

### VisionForm Pattern
Centralized multipart form parsing with validation and base64 conversion:
```go
type VisionForm struct {
    Prompt  string
    Images  []string  // base64 data URIs
    Options map[string]any
    Token   string
}

func ParseVisionForm(r *http.Request, maxMemory int64) (*VisionForm, error)
```

### Token Injection Pattern
Runtime token injection for Azure authentication:
```go
if token != "" {
    if cfg.Provider.Options == nil {
        cfg.Provider.Options = make(map[string]any)
    }
    cfg.Provider.Options["token"] = token
}
```

### SSE Streaming Pattern
Standard SSE format with `data:` prefix and flush after each chunk.

## Test Coverage

Unit tests created for:
- `MapHTTPStatus` - 100%
- `FiltersFromQuery` - 100%
- `Filters.Apply` - 100%
- `ParseVisionForm` - 93.8%
- `prepareImages` - 86.7%

Handler and repository functions require integration tests with database.

## Documentation Created

- `AGENTS.md` - Comprehensive API guide with curl examples for all endpoints
- Godoc comments added to all exported types, functions, and methods

## Validated Agents

| Agent | Provider | Capabilities |
|-------|----------|--------------|
| ollama-agent | Ollama | Chat, Tools |
| vision-agent | Ollama | Chat, Vision |
| embeddings-agent | Ollama | Embeddings |
| azure-key-agent | Azure (API Key) | Chat |
| azure-entra-agent | Azure (Bearer) | Chat |
