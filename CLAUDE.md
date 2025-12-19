# agent-lab Development Guide

## Role and Scope

You are an expert in building web services and agentic workflow platforms with Go.

I'm asking for advice and mentorship, not direct code modifications. This is a project I want to execute myself, but I need guidance and sanity checks when making decisions.

You are authorized to create and modify documentation files, but implementation should be guided through detailed planning documents rather than direct code changes.

## Project Overview

agent-lab is a containerized Go web service for building and orchestrating agentic workflows. It builds on:
- **go-agents** (v0.3.0): LLM integration core
- **go-agents-orchestration** (v0.1.0+): Workflow orchestration patterns
- **document-context** (v0.1.0+): Document processing with LCA architecture

The project follows **Layered Composition Architecture (LCA)** principles from these libraries.

## Documentation Hierarchy

| Document | Purpose | Authoritative For |
|----------|---------|-------------------|
| CLAUDE.md | Project orientation, workflow instructions | How we work together |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Patterns implemented in current codebase | Technical implementation details |
| [PROJECT.md](./PROJECT.md) | Vision, goals, milestone roadmap | What we're building and when |
| `_context/milestones/m##-[title].md` | Milestone architecture documents | Technical depth for multi-session milestones |
| [_context/web-service-architecture.md](./_context/web-service-architecture.md) | General web service architecture | Broader architectural philosophy |
| [_context/service-design.md](./_context/service-design.md) | Project-specific conceptual patterns | Future design directions |

**When to reference each:**
- **Implementation questions** → ARCHITECTURE.md
- **Workflow/process questions** → CLAUDE.md (this file)
- **Roadmap/priorities** → PROJECT.md
- **Milestone technical context** → _context/milestones/m##-[title].md
- **Architectural philosophy** → _context/web-service-architecture.md
- **Conceptual/future patterns** → _context/service-design.md

## Development Workflow

### Milestone Planning Session

Before starting a milestone, conduct a **Milestone Planning Session** to ensure alignment on scope, dependencies, and session breakdown.

#### 1. Explore Current State
- Launch explore agents to understand codebase, architecture, dependencies
- Identify patterns established in previous milestones
- Review what infrastructure exists to build on

#### 2. Analyze Roadmap Alignment
- Review milestone deliverables and success criteria from PROJECT.md
- Check for missing dependencies (go.mod, external tools, libraries)
- Evaluate milestone ordering - surface concerns if dependencies seem misaligned
- Identify any deliverables that could be simplified by existing patterns

#### 3. Discuss Strategic Questions
- Surface ordering concerns or alternatives collaboratively
- Resolve through discussion, not assumptions
- Document decisions with rationale

#### 4. Explore External Dependencies
- For milestones using external libraries, explore their APIs
- Identify built-in capabilities that simplify scope
- Understand configuration patterns and integration points

#### 5. Define Session Breakdown
- Break milestone into focused 2-3 hour sessions
- Identify key files and validation criteria per session
- Leverage existing patterns from previous milestones
- Ensure dependency order (sessions build on previous work)

#### 6. Resolve Design Questions
- Address implementation decisions incrementally with context
- Document decisions with rationale in PROJECT.md

#### 7. Finalize and Document
- Update PROJECT.md with session breakdown and design decisions
- Create milestone architecture document at `_context/milestones/m##-[title].md`
- Capture process improvements in CLAUDE.md if workflow evolved

### Milestone Architecture Documents

For milestones with multiple sessions, create an architecture document that preserves technical context across sessions. This prevents rebuilding context from scratch at the start of each session.

**Location**: `_context/milestones/m##-[title].md` (e.g., `m03-workflow-execution.md`)

**Contents**:
- **Key Decisions**: Architectural choices and rationale
- **Schema Design**: Database tables with field descriptions
- **API Design**: Endpoints with request/response formats
- **Integration Patterns**: How this milestone connects to existing domains
- **Interface Definitions**: Key types and contracts
- **Session Breakdown**: Overview of what each session delivers

**Lifecycle**:
- Created during milestone planning
- Referenced by implementation guides for each session
- Updated if sessions reveal new patterns or decisions
- Archived to `_context/milestones/.archive/` after milestone completion

### Development Session Workflow

Each development session follows a structured workflow:

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

**Plan Mode Note**: If using plan mode, the plan file (`.claude/plans/`) serves as this outline. After plan mode approval, proceed to step 3 to create the full implementation guide - do NOT begin implementation.

#### 3. Implementation Guide Creation
Create comprehensive step-by-step guide:
- Stored in `_context/[session-id]-[session-title].md`
- Session ID format: `01a`, `01b`, `02a`, etc. (milestone + letter)
- Structure: Problem context, architecture approach, detailed implementation steps
- **Dependency Order**: Structure steps from lowest to highest dependency level
- Phases separate architectural preparation from feature development

**Code Block Conventions:**
- Code blocks have NO comments (minimize tokens, avoid maintenance)
- NO testing infrastructure (AI's responsibility after implementation)
- NO documentation (godoc comments added by AI after validation)
- NO OpenAPI `openapi.go` file contents (AI creates these in Step 4)
  - DO include route definitions with `OpenAPI: Spec.*` references
  - DO include schema registration in routes.go (`components.AddSchemas(...)`)
  - The AI prepares the actual `openapi.go` files before developer execution

**File Change Conventions:**
- **Existing files**: Show incremental changes only (what's being added/modified)
- **New files**: Provide complete implementation
- Never replace entire existing files - preserve original architecture integrity

#### 4. OpenAPI Maintenance
Prepare API specification infrastructure before implementation:
- Create/update domain `openapi.go` with operations and schemas
- Ensure schema types align with OpenAPI 3.1 (Properties are Schemas)
- Verify helper functions produce valid references
- See `_context/openapi-integration.md` for patterns and conventions

This ensures handlers can reference `Spec.{Operation}` without compilation errors.

#### 5. Developer Execution
Developer implements following the guide:
- You execute implementation as code base maintainer
- AI on standby for mentorship and adjustments
- Focus on code structure, not comments or tests

#### 6. Validation Phase
AI reviews and validates implementation:
- Review for accuracy and completeness
- Add and revise testing infrastructure
- Execute tests until passing
- Verify critical paths are covered (see Testing Conventions)
- Black-box testing: `package <name>_test`, test only public API

#### 7. Documentation Phase
AI adds code documentation:
- Add godoc comments to exported types, functions, methods
- Document non-obvious behavior

#### 8. Session Closeout

Session closeout ensures documentation stays aligned with the codebase and captures verified patterns.

**8.1 Generate Session Summary**:
- Create `_context/sessions/[session-id]-[session-title].md`
- Document what was implemented, key decisions, and patterns established

**8.2 Archive Implementation Guide**:
- Move guide to `_context/sessions/.archive/[session-id]-[session-title].md`
- We **archive** instead of deleting for reference

**8.3 Update Documentation** (evaluate each):

| Document | Update Criteria |
|----------|-----------------|
| **README.md** | Only critical user-facing changes (new commands, setup steps). Be very conservative. |
| **ARCHITECTURE.md** | Align with latest codebase - add implemented patterns, update examples |
| **Milestone architecture doc** | Update if session revealed new patterns, decisions, or scope adjustments for remaining sessions |
| **web-service-architecture.md** | Move verified patterns from conceptual to validated section. Add new patterns discovered. |
| **service-design.md** | Remove concepts that have been integrated into the codebase |
| **PROJECT.md** | Update session status. Evaluate remaining sessions for adjustments (scope changes, dependencies, reordering) |

### Maintenance Session Workflow

Maintenance sessions differ from development sessions - they focus on cleanup, refactoring, or cross-repository coordination rather than new feature development.

**When to Use**:
- Migrating functionality between repositories (e.g., shims → library)
- Refactoring patterns across multiple domains
- Pre-milestone cleanup to reduce technical debt
- Cross-repository version coordination

**Session ID Format**: `m##` (e.g., `m01`, `m02`)

**Workflow**:

1. **Planning Phase** - Review scope and dependencies across repositories
2. **Implementation Guide** - Create guide at `_context/m##-[title].md`
3. **Developer Execution** - Follow guide, coordinating releases if needed
4. **AI Validation** - Validate changes and adjust tests
5. **Session Closeout**:
   - Archive guide to `_context/sessions/.archive/m##-[title].md`
   - Create summary at `_context/sessions/m##-[title].md`
   - Update PROJECT.md with maintenance session status
   - Update CLAUDE.md if workflow patterns evolved

**Key Differences from Development Sessions**:
- May span multiple repositories
- Library releases may be required between phases
- Focus on consolidation rather than new capability
- Tests may need adjustment rather than creation

### Post Session Milestone Review

After completing a development session (especially mid-milestone), conduct a **Post Session Milestone Review** to ensure the roadmap stays aligned with lessons learned:

**When to Trigger:**
- After any session where friction points or improvement opportunities were identified
- When architectural patterns evolved during implementation
- When scope adjustments are needed for remaining sessions

**Workflow:**

1. **Reflect on Current State**
   - What was implemented vs. what was planned?
   - What patterns emerged or were validated?
   - What technical debt was introduced (if any)?

2. **Identify Improvement Areas**
   - Call out friction points encountered during implementation
   - Note patterns that could be better abstracted
   - Identify missing infrastructure or utilities

3. **Determine Approach**
   - Decide: address now vs. defer to future session
   - If deferring: which session should handle it?
   - If addressing now: does it warrant a dedicated session?

4. **Update Roadmap**
   - Adjust remaining session scopes in PROJECT.md
   - Add new sessions if needed
   - Reorder sessions if dependencies changed
   - Document rationale for changes

**Output:** Updated PROJECT.md roadmap with adjustments documented.

**Note:** This is a planning-only activity. No code changes - only roadmap updates.

### Milestone Review

After completing all sessions in a milestone:
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

**Organization:**
- Tests in `tests/` directory mirroring `internal/` structure
- File naming: `<file>_test.go`

**Black-Box Testing:**
- All tests use `package <name>_test`
- Import package being tested
- Test only exported types/functions/methods

**Coverage Success Criteria:**

Testing success is measured by coverage of critical paths, not arbitrary percentages:

- ✅ **Happy paths** - Normal operation flows work correctly
- ✅ **Security paths** - Input validation, path traversal prevention, injection protection
- ✅ **Error types** - Domain errors are defined and distinguishable
- ✅ **Integration points** - Lifecycle hooks, system boundaries, external dependencies

Uncovered code is acceptable when it consists of:
- Defensive error handling for OS-level failures (disk full, permission denied)
- Edge cases requiring filesystem mocking or special conditions
- Error wrapping paths that don't affect behavior

See [ARCHITECTURE.md](./ARCHITECTURE.md) for testing patterns and table-driven test examples.

## Go Commands

**Validation:** Use `go vet ./...` to check for errors, NOT `go build`. Build is expensive and unnecessary for validation.

**Testing:** Use `go test ./tests/...` for running tests.

## Go File Structure

Go files should be organized in the following order (when each type of element is present):

1. **package** - Package declaration
2. **imports** - Import statements
3. **constants** - `const` blocks
4. **global variables** - `var` blocks
5. **interfaces** - Interface definitions
6. **pure types / enums** - Types without methods (structs used as data, type aliases, enums)
7. **structures + methods** - Structs with their associated methods grouped together
8. **functions** - Standalone functions

**Example:**
```go
package workflows

import (
    "context"
    "time"
)

const defaultBufferSize = 100

var ErrNotFound = errors.New("not found")

type System interface {
    Execute(ctx context.Context) error
}

type ExecuteRequest struct {
    Params map[string]any `json:"params,omitempty"`
}

type Handler struct {
    sys    System
    logger *slog.Logger
}

func NewHandler(sys System, logger *slog.Logger) *Handler {
    return &Handler{sys: sys, logger: logger}
}

func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
    // ...
}
```

## Directory Conventions

**Hidden Directories (`.` prefix)**: Hidden from AI unless explicitly directed (e.g., `.admin`)

**Context Directories (`_` prefix)**: Available for AI reference (e.g., `_context`)

**Context Structure:**
- `_context/##-[guide-title].md` - Active implementation guides
- `_context/milestones/` - Milestone architecture documents (persistent cross-session context)
- `_context/milestones/.archive/` - Archived milestone documents (after milestone completion)
- `_context/sessions/` - Development session summaries
- `_context/sessions/.archive/` - Archived implementation guides

## Technical Patterns

All technical patterns are documented in [ARCHITECTURE.md](./ARCHITECTURE.md):
- System Lifecycle (Cold Start/Hot Start)
- Initialization Pattern (Finalize → Validate → Transform)
- State vs Process separation
- Configuration precedence
- Database patterns (raw SQL, parameterized queries)
- Query Engine (ProjectionMap, QueryBuilder)
- Cross-Domain Dependencies
- Error handling and logging

**Pattern Decision Guide:**
- **Stateful Systems** (own state and other systems) → Use config interface
- **Functional Infrastructure** (stateless utilities) → Use simple parameters

## References

| Resource | Description |
|----------|-------------|
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Complete architectural specification |
| [README.md](./README.md) | Installation, usage, getting started |
| [PROJECT.md](./PROJECT.md) | Roadmap and milestone tracking |
| [_context/web-service-architecture.md](./_context/web-service-architecture.md) | Architectural philosophy |
| go-agents | Configuration patterns, interface design, LCA principles |
| go-agents-orchestration | Workflow patterns, state management |
| document-context | LCA architecture, external binary integration |
