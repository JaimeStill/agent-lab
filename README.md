# agent-lab

Containerized web service platform for building and orchestrating agentic workflows.

## Overview

agent-lab provides a Go-based web service architecture for developing intelligent agentic workflows. Built on the foundation of:
- [go-agents](https://github.com/JaimeStill/go-agents) - LLM integration core
- [go-agents-orchestration](https://github.com/JaimeStill/go-agents-orchestration) - Workflow patterns
- [document-context](https://github.com/JaimeStill/document-context) - Document processing

## Quick Start

### Prerequisites

- Go 1.25.4 or later
- PostgreSQL 17 or later

### Configuration

1. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

2. Update `.env` with your local settings, or create `config.local.yaml` for YAML-based overrides.

3. Environment variables use underscore-separated paths that mirror the YAML structure:
   ```bash
   database_host=localhost
   database_port=5432
   server_port=8080
   ```

### Running Locally

```bash
go run ./cmd/server
```

The server will start on the configured port (default: 8080).

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
├── config.yaml              # Base configuration
├── config.development.yaml  # Development environment
├── config.production.yaml   # Production environment
└── config.local.yaml        # Local overrides (gitignored)
```

## Configuration Management

Configuration is loaded in the following order (later sources override earlier ones):

1. `config.yaml` - Base defaults
2. `config.{ENV}.yaml` - Environment-specific (where ENV=development, production, etc.)
3. `config.local.yaml` - Local overrides (gitignored)
4. Environment variables - Highest priority

### Environment Variable Naming

Environment variables mirror the YAML structure using underscores:

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

## Development Status

This project is in active development. The architecture is designed to be flexible and evolve based on emerging requirements.

## License

[License TBD]
