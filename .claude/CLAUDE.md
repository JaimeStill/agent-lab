# agent-lab

A Go web service for building and orchestrating agentic workflows. Built on go-agents, go-agents-orchestration, and document-context libraries.

## Quick Reference

### Commands

| Action | Command |
|--------|---------|
| Validate | `go vet ./...` |
| Test | `go test ./tests/...` |
| Run | `go run ./cmd/server` |
| Migrate Up | `go run ./cmd/migrate -dsn "..." -up` |
| Migrate Down | `go run ./cmd/migrate -dsn "..." -down` |
| Seed | `go run ./cmd/seed -dsn "..." -all` |
| Build Web | `cd web && bun run build` |

### Session Workflow

**Development Sessions:**
1. Planning → 2. Plan Presentation → 3. Implementation Guide → 4. OpenAPI Maintenance → 5. Developer Execution → 6. Validation → 7. Documentation → 8. Closeout

**Maintenance Sessions:**
1. Planning → 2. Execution → 3. Validation → 4. Closeout

**Implementation Guide Conventions:**
- Session ID: `01a`, `01b` (milestone + letter), `mt01` (maintenance)
- Code blocks: NO comments, NO tests, NO OpenAPI contents
- Existing files: incremental changes; New files: complete implementation

### Architecture

**Layered Composition Architecture (LCA):**
- Cold Start: `New*()` builds state graph
- Hot Start: `Start()` activates processes
- State flows down through method parameters, never up

**Domain System Files:**
```
internal/<domain>/
├── errors.go      # Domain errors (Err prefix)
├── <entity>.go    # Entity types and commands
├── system.go      # System interface + implementation
├── repository.go  # Database operations
├── handler.go     # HTTP handlers
└── openapi.go     # API documentation
```

## Project Structure

```
agent-lab/
├── cmd/server/          # Entry point, composition root
├── internal/            # Domain systems (private)
│   ├── config/          # Server configuration
│   ├── infrastructure/  # Core services (lifecycle, logging, database, storage)
│   ├── api/             # API module assembly
│   └── <domain>/        # Domain systems
├── pkg/                 # Shared utilities (public)
│   ├── logging/         # Logging configuration and factory
│   ├── database/        # Database management
│   └── ...              # Other utilities
├── web/                 # Web clients
│   ├── app/             # Main Lit application
│   └── scalar/          # OpenAPI documentation
└── _context/            # Development documentation
```

## Skills

Skills load automatically based on context. Available skills:

| Skill | Use When |
|-------|----------|
| go-core | Creating packages, errors, interfaces, slog logging |
| go-testing | Writing tests, coverage analysis |
| lca | Implementing systems, lifecycle, Handler() factory |
| go-database | Repository implementation, queries, transactions |
| go-http | HTTP handlers, routes, middleware, SSE |
| go-storage | Blob storage, atomic writes |
| openapi | API documentation, schema definitions |
| agent-execution | Agent integration, vision forms |
| workflow-orchestration | State graphs, observers, checkpoints |
| document-processing | PDF rendering, document-context |
| web-development | Lit components, views, services, CSS |
| development-methodology | Session planning, milestones |

## Session Closeout Checklist

1. Generate summary at `_context/sessions/[session-id]-[title].md`
2. Archive guide to `_context/sessions/.archive/`
3. Update context architecture if patterns changed
4. Update PROJECT.md session status
5. Update README.md only for critical user-facing changes

## Testing Success Criteria

- Happy paths covered
- Security paths covered (validation, injection protection)
- Error types distinguishable
- Integration points verified

## Directory Conventions

**Hidden Directories (`.` prefix)**: Hidden from AI unless explicitly directed

**Context Directories (`_` prefix)**: Available for AI reference

```
.claude/
├── CLAUDE.md              # This file
└── skills/                # On-demand domain skills

_context/
├── [session-id]-*.md      # Active implementation guides
├── milestones/            # Milestone architecture docs
│   └── .archive/          # Completed milestones
├── sessions/              # Session summaries
│   └── .archive/          # Archived guides
└── prompts/               # Session start prompts
```

## Adding New Skills

1. Create directory: `.claude/skills/{skill-name}/`
2. Create `SKILL.md` with YAML frontmatter:
   ```yaml
   ---
   name: skill-name
   description: >
     Brief description with "REQUIRED for..." pattern.
     Include trigger keywords, file patterns, and use cases.
     Max 1024 characters.
   ---
   ```
3. Add sections: When This Skill Applies, Principles, Patterns, Anti-Patterns

## References

| Resource | Description |
|----------|-------------|
| PROJECT.md | Roadmap, milestones, success criteria |
| `_context/milestones/` | Milestone architecture documents |
| go-agents | LLM integration, agent patterns |
| go-agents-orchestration | Workflow orchestration |
| document-context | Document processing |
