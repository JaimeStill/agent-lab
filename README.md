# agent-lab

Containerized web service platform for building and orchestrating agentic workflows.

## Overview

agent-lab provides a Go-based web service architecture for developing intelligent agentic workflows. Built on the foundation of:
- [go-agents](https://github.com/JaimeStill/go-agents) - LLM integration core
- [go-agents-orchestration](https://github.com/JaimeStill/go-agents-orchestration) - Workflow patterns
- [document-context](https://github.com/JaimeStill/document-context) - Document processing

## Development Status

This project is in active development. The architecture is designed to be flexible and evolve based on emerging requirements.

## Documentation

- **[ARCHITECTURE.md](./ARCHITECTURE.md)** - Comprehensive architectural documentation and design principles
- **[CLAUDE.md](./CLAUDE.md)** - Development conventions and guidelines for working with this codebase

## Project Structure

```
agent-lab/
├── cmd/server/              # Application entry point
├── internal/
│   ├── config/              # Configuration loading and management
│   ├── server/              # Application and HTTP routing
│   ├── database/            # Database connection management
│   ├── models/              # Pure data structures
│   ├── services/            # Business logic (ephemeral services)
│   ├── handlers/            # HTTP request handlers
│   └── middleware/          # HTTP middleware
├── migrations/              # SQL database migrations
├── config.toml              # Base configuration
├── config.development.toml  # Development environment
├── config.production.toml   # Production environment
└── config.local.toml        # Local overrides (gitignored)
```

## Quick Start

### Prerequisites

- **Go 1.25.4** or later
- **Docker & Docker Compose**
- **PostgreSQL 17** (containerized via Docker Compose)
- **(Optional)** NVIDIA GPU + nvidia-container-toolkit for Ollama

### Development Setup

agent-lab uses a modular Docker Compose structure that allows you to mix and match services based on your development needs:

#### 1. PostgreSQL Only (Default)

Start just the database for local development where agent-lab runs on your host:

```bash
# Start PostgreSQL container
docker compose up -d

# Run agent-lab on host
go run cmd/server/main.go
```

**Connection**: agent-lab connects to `localhost:5432`

#### 2. PostgreSQL + Ollama (Local LLM Testing)

Start database and Ollama for testing with local LLM models:

```bash
# Start PostgreSQL + Ollama containers
docker compose -f docker-compose.dev.yml up -d

# Run agent-lab on host
go run cmd/server/main.go
```

**Connections**:
- PostgreSQL: `localhost:5432`
- Ollama: `http://localhost:11434`

**Note**: Ollama will automatically pull the `llama3.2:3b` model on first startup. Models are persisted to `~/.ollama`.

#### 3. Fully Containerized (Future)

When agent-lab has a Dockerfile, you can run everything in containers:

```bash
# Start all services (future)
docker compose -f docker-compose.full.yml up -d
```

### Configuration

1. **Copy environment template**:
   ```bash
   cp .env.example .env
   ```

2. **Customize settings** in `.env`:
   ```env
   # PostgreSQL
   POSTGRES_DB=agent_lab
   POSTGRES_USER=agent_lab
   POSTGRES_PASSWORD=agent_lab
   POSTGRES_PORT=5432

   # Ollama (optional)
   OLLAMA_PORT=11434
   OLLAMA_MODELS_DIR=~/.ollama

   # Agent-Lab Server
   SERVER_PORT=8080
   ```

3. **Override with TOML** (optional): Create `config.local.toml` for structured configuration overrides.

### Useful Commands

**Start services**:
```bash
docker compose up -d                          # PostgreSQL only
docker compose -f docker-compose.dev.yml up -d  # PostgreSQL + Ollama
```

**Stop services**:
```bash
docker compose down                          # Stop current stack
docker compose down -v                       # Stop and remove volumes
```

**View logs**:
```bash
docker compose logs -f postgres              # PostgreSQL logs
docker compose logs -f ollama                # Ollama logs (if using dev)
```

**Check service health**:
```bash
docker compose ps                            # Service status
```

### Database Migrations

Migrations will be managed using [golang-migrate](https://github.com/golang-migrate/migrate). Setup instructions coming soon.

## Configuration Management

Configuration is loaded in the following order (later sources override earlier ones):

1. `config.toml` - Base defaults
2. `config.{ENV}.toml` - Environment-specific (where ENV=development, production, etc.)
3. `config.local.toml` - Local overrides (gitignored)
4. Environment variables - Highest priority

### Environment Variable Naming

Environment variables mirror the TOML structure using underscores:

**Simple values:**
```bash
server_port=8080
database_host=localhost
logging_level=debug
```

**Arrays (indexed convention):**
```bash
cors_origins_0=http://localhost:3000
cors_origins_1=http://localhost:4000
```

**Nested objects in arrays:**
```bash
database_replicas_0_host=db1
database_replicas_0_port=5432
database_replicas_1_host=db2
database_replicas_1_port=5432
```

## License

[License TBD]
