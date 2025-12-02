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
│   ├── service/          # Service entry point and composition
│   └── migrate/          # Database migration CLI
├── internal/             # Private packages
│   ├── config/           # Configuration management
│   ├── database/         # Database connection management
│   ├── lifecycle/        # Startup/shutdown coordination
│   ├── logger/           # Logging system
│   ├── middleware/       # HTTP middleware
│   ├── routes/           # Route registration
│   └── server/           # HTTP server
├── pkg/                  # Public packages
│   ├── pagination/       # Pagination utilities
│   └── query/            # SQL query builder
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

# Build and run server
go build -o bin/server ./cmd/server
./bin/server

# Health check (liveness)
curl http://localhost:8080/healthz

# Readiness check (subsystems operational)
curl http://localhost:8080/readyz
```

### Configuration

Configuration loads from `config.toml` with optional environment-specific overlays (`config.{env}.toml`) and environment variable overrides.

Set `SERVICE_ENV` to load an overlay:
```bash
SERVICE_ENV=dev ./bin/service  # Loads config.dev.toml
```

See [config.toml](./config.toml) for available settings.

### Testing

```bash
# Run all tests
go test ./tests/... -v

# Run with coverage
go test ./tests/... -cover
```

## Documentation

- **[ARCHITECTURE.md](./ARCHITECTURE.md)** - Technical specifications and design patterns
- **[CLAUDE.md](./CLAUDE.md)** - Development conventions and workflow
- **[PROJECT.md](./PROJECT.md)** - Project roadmap and milestones
- **[_context/web-service-architecture.md](./_context/web-service-architecture.md)** - Architectural philosophy

## License

All rights reserved.
