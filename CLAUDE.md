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

### System Lifecycle

**Stateful Systems** (Application-scoped):
- Long-running systems that own state and processes
- Examples: Server, Database, Providers, Agents
- Initialized at server startup, live for application lifetime
- Use encapsulated config interface pattern

**Functional Infrastructure** (Stateless):
- Pure functions or minimal-state utilities
- Examples: Handlers, middleware, routing, query builders
- Use simple function signatures

**Cold Start vs Hot Start**:

**Cold Start** (State Initialization):
- `New*()` constructor functions
- Builds entire dependency graph
- All configurations → State objects
- All systems created but dormant
- No processes running

**Hot Start** (Process Activation):
- `Start()` methods
- State objects → Running processes
- Cascade start through dependency graph
- Context boundaries for lifecycle management
- System becomes interactable

### Initialization Pattern

**Stateful Systems**: Use config interface pattern following: **Finalize → Validate → Transform**

```go
type ProvidersConfig interface {
    DB() *sql.DB
    Logger() *slog.Logger
    Pagination() pagination.Config

    Finalize()
    Validate() error
}

func New(cfg ProvidersConfig) (System, error) {
    // 1. Finalize: Apply defaults
    cfg.Finalize()

    // 2. Validate: Check required dependencies
    if err := cfg.Validate(); err != nil {
        return nil, err
    }

    // 3. Transform: Create validated instance
    return &repository{
        db:         cfg.DB(),
        logger:     cfg.Logger().With("system", "providers"),
        pagination: cfg.Pagination(),
    }, nil
}
```

**Why Config Interfaces**:
- Makes required state immediately apparent
- Config can define its own finalize/validate/transform behaviors
- Configuration graph for owned objects lives in parent config
- Clear ownership boundaries
- Easier testing (mock the interface)

**Functional Infrastructure**: Use simple parameters (no config interface needed)

See [ARCHITECTURE.md](./ARCHITECTURE.md) for complete patterns.

### State vs Process

**State** = Pure Data (no methods):
- Define structure only: entities, commands, filters
- Located in domain packages (e.g., `internal/providers/provider.go`)
- Examples: `Provider`, `CreateCommand`, `SearchRequest`

**Process** = Behavior (system methods):
- System methods that operate on state
- Contain both queries (read) and commands (write)
- Commands always use transactions
- Located in system implementations (e.g., `internal/providers/repository.go`)

**Handlers** = HTTP Layer (functional infrastructure):
- Pure functions that receive state as parameters
- State flows DOWN through parameters
- Located in domain packages (e.g., `internal/providers/handlers.go`)

**Package Organization**:
- **cmd/server**: The process (composition root, entry point)
- **pkg/**: Public API (shared infrastructure, reusable toolkit)
- **internal/**: Private API (domain systems, business logic)

### Configuration

**Format**: TOML (Tom's Obvious, Minimal Language)

**Configuration Precedence Principle**:
All configuration values (scalar or array) are atomic units that replace at each precedence level:
```
Environment Variables (highest precedence)
    ↓ replaces (not merges)
config.local.toml / config.*.toml
    ↓ replaces (not merges)
config.toml (base configuration)
```

**Key Principles**:
- **Atomic Replacement**: Values never merge - presence indicates complete replacement
- **Array Format**: Use comma-separated strings in environment variables
- **Consistent Behavior**: Scalar and array configs follow same precedence rules

**Environment Variable Convention**:
- Mirrors TOML structure with underscores: `SECTION_FIELD` (uppercase)
- Scalar values: `SERVER_PORT=9090`
- Array values: `CORS_ORIGINS="http://example.com,http://other.com"` (comma-separated)

See [config.toml](./config.toml) for examples.

### Database

- Raw SQL with `database/sql` + pgx driver
- Parameterized queries (`$1`, `$2`, etc.)
- Commands use transactions, queries don't
- Context-aware operations (`QueryRowContext`, `ExecContext`)

### Query Engine

**Never expose simple GET all endpoints** - Always use paginated search with filters.

Three-layer architecture in `pkg/query`:

**Layer 1: ProjectionMap** (Structure Definition):
- Static, reusable query structure per domain entity
- Defines tables, joins, column mappings
- Resolves view property names to `table.column` references

**Layer 2: QueryBuilder** (Operations):
- Fluent builder for filters, sorting, pagination
- Methods: `WhereEquals`, `WhereContains`, `WhereSearch`, `OrderBy`
- Automatic null-checking: only applies filters when values are non-null
- Generates: `BuildCount()`, `BuildPage()`, `BuildSingle()`

**Layer 3: Execution** (database/sql):
- Execute generated SQL + args with `QueryContext`/`ExecContext`
- Two-query pattern: COUNT for total, SELECT with OFFSET/FETCH

**Example Usage**:
```go
// Define once per domain entity (static)
var providerProjection = query.NewProjectionMap("public", "providers", "p").
    Project("id", "Id").
    Project("name", "Name")

// Use in system methods
qb := query.NewBuilder(providerProjection, "Name").
    WhereContains("Name", filters.Name).
    WhereSearch(page.Search, "Name").
    OrderBy(page.SortBy, page.Descending)

// Execute count + page queries
countSQL, countArgs := qb.BuildCount()
pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
```

### Error Handling

- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
- Define system-level errors in domain packages (e.g., `internal/providers/errors.go`)
- Map system errors to HTTP status codes in handlers

### Logging

- Structured logging with `slog`
- Contextual loggers with attributes: `logger.With("system", "providers")`
- Levels: debug (dev), info (normal), warn (unexpected), error (requires attention)

## Development Workflow

### Milestone Planning

Before starting a milestone, conduct a **Milestone Planning Session**:
1. Review milestone objectives and success criteria
2. Break milestone into focused development sessions (2-3 hour chunks)
3. Define validation criteria for each session
4. Ensure dependency order (sessions build on previous work)
5. Document session breakdown in PROJECT.md

### Development Session Workflow

Each development session follows a structured workflow with clearly defined roles:

#### 1. Planning Phase
**Collaborative exploration** of implementation approaches:
- Discuss architectural decisions and trade-offs
- Explore multiple implementation strategies
- Ask clarifying questions as needed
- Reach alignment on approach before proceeding

**Not** a one-shot plan presentation - iterate until aligned.

#### 2. Plan Presentation
Present **outline of implementation guide** for approval:
- Summarize agreed approach
- Show high-level structure
- Confirm scope and phases
- Get explicit approval before detailed guide

#### 3. Implementation Guide Creation
Create comprehensive step-by-step guide:
- Stored in `_context/[session-id]-[session-title].md`
- Session ID format: `01a`, `01b`, `02a`, etc. (milestone + letter)
- Structure: Problem context, architecture approach, detailed implementation steps
- **Dependency Order**: Structure steps from lowest to highest dependency level
  - Example: Create state structures before systems that use them
  - Example: Create systems before handlers that use them
  - Ensures all dependencies exist before they're referenced
- **Code blocks have NO comments** (minimize tokens, avoid maintenance)
- **NO testing infrastructure** (AI's responsibility after implementation)
- **NO documentation** (godoc comments added by AI after validation)
- File-by-file changes with complete code examples
- Phases separate architectural preparation from feature development

#### 4. Developer Execution
Developer implements following the guide:
- You execute implementation as code base maintainer
- AI on standby for mentorship and adjustments
- Focus on code structure, not comments or tests

#### 5. Validation Phase
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

#### 6. Documentation Phase
AI adds code documentation and maintains API specifications:
- Add godoc comments to exported types, functions, methods
- Document non-obvious behavior
- Explain complex logic
- Update examples if needed
- **Maintain OpenAPI specification** (`api/openapi.yaml`) after any API surface changes (when applicable)
  - Update paths, request/response schemas, error codes
  - Keep examples current and accurate
  - Ensure consistency with actual implementation
  - Treat OpenAPI spec maintenance like tests and docs: AI responsibility after changes

#### 7. Session Closeout
Create session summary and commit:
- Generate development session summary
- Archive implementation guide: `_context/sessions/.archive/[session-id]-[session-title].md`
- Commit working code with descriptive message
- Update project status in PROJECT.md

**Note**: Unlike other projects, we **archive** implementation guides instead of deleting them.

---

### Milestone Review

After completing all sessions in a milestone, conduct a **Milestone Review**:
1. **Validate Success Criteria** - Confirm all milestone objectives met
2. **Integration Testing** - Test milestone as cohesive unit
3. **Identify Adjustments** - Document any final changes needed
4. **Execute Adjustments** - Make necessary refinements
5. **Final Validation** - Confirm milestone complete
6. **Documentation Update** - Update PROJECT.md milestone status

### Milestone Completion

Mark milestone complete only after:
- All development sessions completed
- Milestone review passed
- All success criteria validated
- Integration tests passing
- Documentation updated

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

**ARCHITECTURE.md**: Technical specifications, interface definitions, design patterns. Source of truth for building the system. Focus on concrete implementation details for Milestone 1.

**README.md**: User-facing installation, usage, configuration, getting started.

**CLAUDE.md** (this file): Development conventions, workflow, testing strategy.

**PROJECT.md** (when it exists): Roadmap, scope, design philosophy, completion checklist.

## Code Design Principles

Detailed in [ARCHITECTURE.md](./ARCHITECTURE.md):
- State flows down, never up (unless state is owned by object/process)
- Systems, not services/models (use domain-specific terminology)
- Cold Start/Hot Start lifecycle separation
- Configuration-driven initialization (finalize → validate → transform)
- Encapsulated config interfaces for stateful systems
- Simple parameters for functional infrastructure
- State as pure structure, processes as system methods
- Commands use transactions, queries don't
- Interface-based layer interconnection
- Raw SQL patterns
- Graceful shutdown

**Pattern Decision Guide**:
- **Stateful Systems** (own state and other systems) → Use config interface
- **Functional Infrastructure** (stateless utilities) → Use simple parameters

Refer to go-agents, go-agents-orchestration, and document-context CLAUDE.md files for detailed design principles when applicable.

## References

- **web-service-architecture.md**: Complete architectural philosophy and design decisions
- **go-agents**: Configuration patterns, interface design, LCA principles
- **go-agents-orchestration**: Workflow patterns, state management
- **document-context**: LCA architecture, external binary integration
- **ARCHITECTURE.md**: Complete architectural specification for agent-lab
