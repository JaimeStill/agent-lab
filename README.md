# agent-lab

Containerized web service platform for building and orchestrating agentic workflows.

## Overview

agent-lab provides a Go-based web service architecture for developing intelligent agentic workflows. Built on the foundation of:
- [go-agents](https://github.com/JaimeStill/go-agents) - LLM integration core
- [go-agents-orchestration](https://github.com/JaimeStill/go-agents-orchestration) - Workflow patterns
- [document-context](https://github.com/JaimeStill/document-context) - Document processing

## Project Structure

```
agent-lab/
├── cmd/
│   ├── server/           # HTTP server entry point and composition
│   ├── migrate/          # Database migration CLI
│   └── seed/             # Database seeding CLI
├── internal/             # Private packages
│   ├── config/           # Configuration management
│   ├── database/         # Database connection management
│   ├── lifecycle/        # Startup/shutdown coordination
│   ├── middleware/       # HTTP middleware
│   ├── routes/           # Route registration
│   ├── storage/          # Blob storage abstraction
│   ├── providers/        # Provider domain (LLM configurations)
│   ├── agents/           # Agents domain (LLM agents)
│   ├── documents/        # Documents domain (file upload/management)
│   ├── images/           # Images domain (document rendering)
│   ├── profiles/         # Profiles domain (workflow configuration)
│   └── workflows/        # Workflows domain (execution infrastructure)
├── workflows/            # Workflow definitions
│   ├── summarize/        # Text summarization workflow
│   ├── reasoning/        # Multi-step reasoning workflow
│   └── classify/         # Document classification workflow
├── pkg/                  # Public packages
│   ├── handlers/         # HTTP response utilities
│   ├── openapi/          # OpenAPI spec utilities
│   ├── pagination/       # Pagination utilities
│   ├── query/            # SQL query builder
│   ├── decode/           # Type-safe map decoding
│   └── repository/       # Database helpers
├── web/                  # Web assets
│   └── docs/             # API documentation (Scalar UI)
├── tests/                # Black-box tests
├── compose/              # Docker Compose files
└── config.toml           # Base configuration
```

## Quick Start

### Running the Service

```bash
# Start PostgreSQL
docker compose -f compose/postgres.yml up -d

# Run database migrations
go run ./cmd/migrate -dsn "postgres://agent_lab:agent_lab@localhost:5432/agent_lab?sslmode=disable" -up

# Seed database
go run ./cmd/seed -dsn "postgres://agent_lab:agent_lab@localhost:5432/agent_lab?sslmode=disable" -all

# Build and run server
go build -o bin/server ./cmd/server
./bin/server

# Health check (liveness)
curl http://localhost:8080/healthz

# Readiness check (subsystems operational)
curl http://localhost:8080/readyz

# API documentation (Scalar UI)
open http://localhost:8080/docs
```

### Configuration

Configuration loads from `config.toml` with optional environment-specific overlays (`config.{env}.toml`) and environment variable overrides.

Set `SERVICE_ENV` to load an overlay:
```bash
SERVICE_ENV=dev ./bin/server  # Loads config.dev.toml
```

See [config.toml](./config.toml) for available settings.

### Testing

```bash
# Run all tests
go test ./tests/... -v

# Run with coverage
go test ./tests/... -cover
```

## API Overview

All endpoints are documented in the interactive Scalar UI at `http://localhost:8080/docs`.

| Domain | Prefix | Description |
|--------|--------|-------------|
| Providers | `/api/providers` | LLM provider configurations (Ollama, Azure, etc.) |
| Agents | `/api/agents` | Agent definitions with execution endpoints (Chat, Vision, Tools, Embed) |
| Documents | `/api/documents` | Document upload and management |
| Images | `/api/images` | Document page rendering with enhancement filters |
| Profiles | `/api/profiles` | Workflow stage configurations for A/B testing |
| Workflows | `/api/workflows` | Workflow execution with SSE streaming |

**Getting Started Order**: Providers → Agents → (Documents → Images for document workflows) → Profiles → Workflows

## Sample Workflows

agent-lab includes sample workflows that demonstrate live agent integration. Test these via the Scalar API documentation at `http://localhost:8080/docs`.

### Prerequisites

1. Ensure the server is running
2. Create an agent profile via `POST /api/agents` (or use an existing one)
3. Note the agent's UUID for use in workflow params

### Available Workflows

List registered workflows:
```
GET /api/workflows
```

### Summarize Workflow

Single-node workflow that summarizes input text using an AI agent.

**Endpoint**: `POST /api/workflows/summarize/execute`

**Request Body** (for Scalar interface):
```json
{
  "params": {
    "agent_id": "<AGENT_UUID>",
    "text": "The quick brown fox jumps over the lazy dog. This sentence contains every letter of the alphabet and is commonly used for typing practice."
  }
}
```

**Optional Parameters**:
- `system_prompt` - Override the default summarization prompt
- `token` - Runtime API token override

### Reasoning Workflow

Multi-node workflow that performs step-by-step reasoning: analyze → reason → conclude.

**Endpoint**: `POST /api/workflows/reasoning/execute`

**Request Body** (for Scalar interface):
```json
{
  "params": {
    "agent_id": "<AGENT_UUID>",
    "problem": "If all roses are flowers and some flowers fade quickly, can we conclude that some roses fade quickly?"
  }
}
```

**Optional Parameters**:
- `analyze_system_prompt` - Override the analyze node prompt
- `reason_system_prompt` - Override the reason node prompt
- `conclude_system_prompt` - Override the conclude node prompt
- `token` - Runtime API token override

### Viewing Results

**List all runs**:
```
GET /api/workflows/runs
```

**Get a specific run**:
```
GET /api/workflows/runs/{run_id}
```

**View execution stages** (for multi-node workflows):
```
GET /api/workflows/runs/{run_id}/stages
```

## Documentation

- **[ARCHITECTURE.md](./ARCHITECTURE.md)** - Technical specifications and design patterns
- **[AGENTS.md](./AGENTS.md)** - Agents API guide with curl examples
- **[CLAUDE.md](./CLAUDE.md)** - Development conventions and workflow
- **[PROJECT.md](./PROJECT.md)** - Project roadmap and milestones
- **[_context/web-service-architecture.md](./_context/web-service-architecture.md)** - Architectural philosophy

## License

All rights reserved.
