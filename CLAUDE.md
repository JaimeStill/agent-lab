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
| [_context/web-service-architecture.md](./_context/web-service-architecture.md) | General web service architecture | Broader architectural philosophy |
| [_context/service-design.md](./_context/service-design.md) | Project-specific conceptual patterns | Future design directions |

**When to reference each:**
- **Implementation questions** → ARCHITECTURE.md
- **Workflow/process questions** → CLAUDE.md (this file)
- **Roadmap/priorities** → PROJECT.md
- **Architectural philosophy** → _context/web-service-architecture.md
- **Conceptual/future patterns** → _context/service-design.md

## Development Workflow

### Milestone Planning

Before starting a milestone, conduct a **Milestone Planning Session**:
1. Review milestone objectives and success criteria
2. Break milestone into focused development sessions (2-3 hour chunks)
3. Define validation criteria for each session
4. Ensure dependency order (sessions build on previous work)
5. Document session breakdown in PROJECT.md

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

**File Change Conventions:**
- **Existing files**: Show incremental changes only (what's being added/modified)
- **New files**: Provide complete implementation
- Never replace entire existing files - preserve original architecture integrity

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

#### 6. Documentation Phase
AI adds code documentation and maintains API specifications:
- Add godoc comments to exported types, functions, methods
- Document non-obvious behavior
- **Maintain OpenAPI specification** (`api/openapi.yaml`) after API surface changes

#### 7. Session Closeout

Session closeout ensures documentation stays aligned with the codebase and captures verified patterns.

**7.1 Generate Session Summary**:
- Create `_context/sessions/[session-id]-[session-title].md`
- Document what was implemented, key decisions, and patterns established

**7.2 Archive Implementation Guide**:
- Move guide to `_context/sessions/.archive/[session-id]-[session-title].md`
- We **archive** instead of deleting for reference

**7.3 Update Documentation** (evaluate each):

| Document | Update Criteria |
|----------|-----------------|
| **README.md** | Only critical user-facing changes (new commands, setup steps). Be very conservative. |
| **ARCHITECTURE.md** | Align with latest codebase - add implemented patterns, update examples |
| **web-service-architecture.md** | Move verified patterns from conceptual to validated section. Add new patterns discovered. |
| **service-design.md** | Remove concepts that have been integrated into the codebase |
| **PROJECT.md** | Update session status. Evaluate remaining sessions for adjustments (scope changes, dependencies, reordering) |

**7.4 Commit Preparation**:
- User handles git commit after closeout is complete
- AI prepares commit message summary if requested

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

**Coverage:** 80% minimum, 100% for critical paths (validation, transactions, routing)

See [ARCHITECTURE.md](./ARCHITECTURE.md) for testing patterns and table-driven test examples.

## Directory Conventions

**Hidden Directories (`.` prefix)**: Hidden from AI unless explicitly directed (e.g., `.admin`)

**Context Directories (`_` prefix)**: Available for AI reference (e.g., `_context`)

**Context Structure:**
- `_context/##-[guide-title].md` - Active implementation guides
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
