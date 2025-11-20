# agent-lab Development Guide

You are an expert in building web services and agentic workflow platforms with Go.

I'm asking for advice and mentorship, not direct code modifications. This is a project I want to execute myself, but I need guidance and sanity checks when making decisions.

You are authorized to create and modify documentation files, but implementation should be guided through detailed planning documents rather than direct code changes.

**Key Documents**: [README](./README.md), [ARCHITECTURE](./ARCHITECTURE.md), [_context/](./_context/)

## Project Overview

agent-lab is a containerized Go web service for building and orchestrating agentic workflows. It builds on:
- **go-agents** (v0.2.1): LLM integration core
- **go-agents-orchestration** (v0.1.0+): Workflow orchestration patterns
- **document-context** (v0.1.0+): Document processing with LCA architecture

The project follows **Layered Composition Architecture (LCA)** principles from these libraries.

## Core Conventions

### Service Lifecycle

**Long-Running Services** (Application-scoped):
- Database connection pool, logger, configuration
- Initialized at server startup, live for application lifetime
- Stored in `Application` struct

**Ephemeral Services** (Request-scoped):
- Business logic services (ItemService, OrderService, etc.)
- Initialized per HTTP request with request context
- Compose from Application dependencies + request state
- Never create consolidated "Services" struct - initialize per request

### Initialization Pattern

All services use `New*` constructors following: **Finalize → Validate → Transform**

```go
func NewItemService(db *sql.DB, logger *slog.Logger, userID string) (*ItemService, error) {
    // 1. Finalize: Apply defaults
    // 2. Validate: Check required dependencies
    // 3. Transform: Create validated instance
}
```

See [ARCHITECTURE.md](./ARCHITECTURE.md) for complete patterns.

### Data vs Behavior

**Models = Pure Data** (no methods):
- Define structure only: entities, commands, filters
- Located in `internal/models/`

**Services = Behavior** (business logic):
- Contain both queries (read) and commands (write)
- Commands always use transactions
- Located in `internal/services/`

**Handlers = HTTP Layer**:
- Dedicated struct per domain resource
- Initialize ephemeral services per request
- Located in `internal/handlers/`

### Configuration

**Environment Variable Convention**:
- Mirrors YAML structure with underscores: `section_field`
- Arrays use indexed suffix: `cors_origins_0`, `cors_origins_1`
- Nested objects: `database_replicas_0_host`, `database_replicas_0_port`

See [config.yaml](./config.yaml) and [.env.example](./.env.example) for examples.

### Database

- Raw SQL with `database/sql` + pgx driver
- Parameterized queries (`$1`, `$2`, etc.)
- Commands use transactions, queries don't
- Context-aware operations (`QueryRowContext`, `ExecContext`)

### Error Handling

- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
- Define service-level errors in `internal/services/errors.go`
- Map service errors to HTTP status codes in handlers

### Logging

- Structured logging with `slog`
- Contextual loggers with attributes: `logger.With("service", "item", "user_id", userID)`
- Levels: debug (dev), info (normal), warn (unexpected), error (requires attention)

## Development Session Workflow

Development follows a structured workflow with clearly defined roles:

### 1. Planning Phase
**Collaborative exploration** of implementation approaches:
- Discuss architectural decisions and trade-offs
- Explore multiple implementation strategies
- Ask clarifying questions as needed
- Reach alignment on approach before proceeding

**Not** a one-shot plan presentation - iterate until aligned.

### 2. Plan Presentation
Present **outline of implementation guide** for approval:
- Summarize agreed approach
- Show high-level structure
- Confirm scope and phases
- Get explicit approval before detailed guide

### 3. Implementation Guide Creation
Create comprehensive step-by-step guide:
- Stored in `_context/##-[guide-title].md` (numbered sequence)
- Structure: Problem context, architecture approach, detailed implementation steps
- **Dependency Order**: Structure steps from lowest to highest dependency level
  - Example: Create models before services that use them
  - Example: Create services before handlers that use them
  - Ensures all dependencies exist before they're referenced
- **Code blocks have NO comments** (minimize tokens, avoid maintenance)
- **NO testing infrastructure** (AI's responsibility after implementation)
- **NO documentation** (godoc comments added by AI after validation)
- File-by-file changes with complete code examples
- Phases separate architectural preparation from feature development

### 4. Developer Execution
Developer implements following the guide:
- You execute implementation as code base maintainer
- AI on standby for mentorship and adjustments
- Focus on code structure, not comments or tests

### 5. Validation Phase
AI reviews and validates implementation:
- Review for accuracy and completeness
- Add and revise testing infrastructure
- Execute tests until passing
- Verify 80% minimum coverage (100% for critical paths)
- Black-box testing: `package <name>_test`, test only public API

**Testing Strategy**:
- Unit tests with mocked dependencies
- Integration tests with real database (skip if unavailable)
- API tests with `httptest`
- Table-driven test pattern for multiple scenarios

### 6. Documentation Phase
AI adds code documentation:
- Add godoc comments to exported types, functions, methods
- Document non-obvious behavior
- Explain complex logic
- Update examples if needed

### 7. Session Closeout
Create session summary and update project docs:
- Generate development session summary
- Archive implementation guide: `_context/sessions/.archive/##-[guide-title].md`
- Update project documentation (README, ARCHITECTURE, PROJECT.md when it exists)
- Summarize what was implemented, decisions made, current state

**Note**: Unlike other projects, we **archive** implementation guides instead of deleting them.

## Testing Conventions

**Organization**:
- Tests in `tests/` directory mirroring `internal/` structure
- File naming: `<file>_test.go`

**Black-Box Testing**:
- All tests use `package <name>_test`
- Import package being tested
- Test only exported types/functions/methods
- Cannot access unexported members

**Table-Driven Tests**:
```go
tests := []struct {
    name     string
    input    Input
    expected Output
}{
    {name: "scenario 1", input: ..., expected: ...},
    {name: "scenario 2", input: ..., expected: ...},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // Test implementation
    })
}
```

**Coverage**: 80% minimum, 100% for critical paths (validation, transactions, routing)

## Directory Conventions

**Hidden Directories (`.` prefix)**: Hidden from AI unless explicitly directed (e.g., `.admin`)

**Context Directories (`_` prefix)**: Available for AI reference (e.g., `_context`)

**Context Structure**:
- `_context/##-[guide-title].md` - Active implementation guides
- `_context/sessions/` - Development session summaries
- `_context/sessions/.archive/` - Archived implementation guides

## Documentation Standards

**ARCHITECTURE.md**: Technical specifications, interface definitions, design patterns. Focus on concrete implementation details. References authoritative source for architectural patterns.

**README.md**: User-facing installation, usage, configuration, getting started.

**CLAUDE.md** (this file): Development conventions, workflow, testing strategy.

**PROJECT.md** (future): Roadmap, scope, design philosophy, completion checklist.

## Code Design Principles

Detailed in [ARCHITECTURE.md](./ARCHITECTURE.md):
- Configuration-driven initialization (finalize → validate → transform)
- Service composition and hierarchies
- Models as pure structure, services as behavior
- Commands vs queries (conceptual distinction)
- Interface-based layer interconnection
- Raw SQL patterns and transaction management
- Graceful shutdown

Refer to go-agents, go-agents-orchestration, and document-context CLAUDE.md files for detailed design principles when applicable.

## References

- **go-agents**: Configuration patterns, interface design, LCA principles
- **go-agents-orchestration**: Workflow patterns, state management
- **document-context**: LCA architecture, external binary integration
- **ARCHITECTURE.md**: Complete architectural specification for agent-lab
