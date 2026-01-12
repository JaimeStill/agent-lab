# Context Architecture Optimization - Planning Document

## Problem Statement

The agent-lab project's context documentation has grown organically and is becoming unwieldy:

| Document | Lines | Purpose |
|----------|-------|---------|
| CLAUDE.md | ~400 | Role, workflow, conventions, patterns |
| ARCHITECTURE.md | ~800 | Technical patterns, examples, code structure |
| _context/frontend-architecture-design.md | ~580 | Web client architecture |
| _context/service-design.md | ~100 | Conceptual patterns |
| _context/web-service-architecture.md | ~900 | Architectural philosophy |

**Issues**:
1. All context loads on every session regardless of task relevance
2. No progressive disclosure - detailed patterns compete with quick references
3. Growing maintenance burden as new patterns emerge (Milestone 5 will add web component patterns)
4. Duplication across documents
5. Unclear hierarchy - what's authoritative for what?

## Solution: Claude Code Context Layering

Based on patterns from `~/code/claude-context`, restructure using Claude Code's native context features:

### Loading Behavior Tiers

| Tier | Location | Loading | Use For |
|------|----------|---------|---------|
| Always | `.claude/CLAUDE.md` | Every session | Navigation map, role, workflow instructions |
| Always | `.claude/rules/*.md` | Every session | Core principles that apply universally |
| On-Demand | `.claude/skills/` | When triggered | Detailed patterns, loaded by relevance |
| Reference | `_context/milestones/` | Manual | Milestone-specific architecture docs |

### Key Insight: CLAUDE.md as Navigation Map

Current CLAUDE.md tries to be everything. New CLAUDE.md should:
- Define role and scope
- List available skills with trigger hints
- Provide quick reference for common commands
- Point to deeper resources without duplicating them

Skills handle the detailed patterns, loading only when Claude determines relevance based on:
- Keywords in the skill's description
- File patterns being edited
- Task context

---

## Proposed Structure

```
.claude/
├── CLAUDE.md                           # Lean navigation + workflow
├── rules/
│   ├── go-general.md                   # Always: naming, file structure, error handling
│   ├── go-testing.md                   # Always: testing conventions
│   └── documentation.md                # Always: doc standards, session workflow
└── skills/
    ├── go-packages/
    │   └── SKILL.md                    # Package/library development patterns
    ├── go-web-services/
    │   └── SKILL.md                    # HTTP handlers, middleware, routing
    ├── go-database/
    │   └── SKILL.md                    # Repository pattern, query builder, migrations
    ├── openapi/
    │   └── SKILL.md                    # OpenAPI spec development, domain schemas
    ├── lca-architecture/
    │   └── SKILL.md                    # Layered Composition Architecture patterns
    ├── workflow-orchestration/
    │   └── SKILL.md                    # go-agents-orchestration patterns
    ├── document-processing/
    │   └── SKILL.md                    # document-context integration patterns
    ├── web-components/
    │   └── SKILL.md                    # Web Component patterns (Milestone 5)
    └── milestone-planning/
        └── SKILL.md                    # Planning session workflow

_context/
├── milestones/                         # Milestone architecture docs (unchanged)
│   └── m05-workflow-lab-interface.md
├── sessions/                           # Session summaries (unchanged)
└── .archive/                           # Retired documents
```

---

## Content Migration Plan

### CLAUDE.md (New - Lean Navigation)

**Keep**:
- Role and Scope section
- Documentation Hierarchy table
- Development Session Workflow (summary only, detail in skill)
- Go Commands section
- Directory Conventions section
- References table

**Extract to Rules**:
- Go File Structure conventions → `rules/go-general.md`
- Testing Conventions → `rules/go-testing.md`

**Extract to Skills**:
- Milestone Planning Session workflow → `skills/milestone-planning/`
- Development Session Workflow (detailed) → `skills/milestone-planning/`
- Technical Patterns section → appropriate domain skills

### ARCHITECTURE.md (Evaluate for Extraction)

**Content Inventory**:
- System Lifecycle (Cold Start/Hot Start) → `skills/lca-architecture/`
- Initialization Pattern → `skills/lca-architecture/`
- Configuration patterns → `skills/go-packages/`
- Database patterns → `skills/go-database/`
- Query Engine patterns → `skills/go-database/`
- Cross-Domain Dependencies → `skills/lca-architecture/`
- Error handling → `rules/go-general.md`
- Naming conventions → `rules/go-general.md`
- Handler patterns → `skills/go-web-services/`
- Testing patterns → `rules/go-testing.md`

**Decision**: ARCHITECTURE.md may become a lean reference pointing to skills, or be retired entirely if skills provide complete coverage.

### _context/web-service-architecture.md

**Disposition**: Extract validated patterns to appropriate skills, archive the document. This is architectural philosophy - once patterns are proven in code, they belong in skills.

### _context/service-design.md

**Disposition**: Review for any patterns not yet implemented. If conceptual/future-focused, archive. If validated, extract to skills.

### _context/frontend-architecture-design.md

**Disposition**: This informed Milestone 5 planning. Content will be superseded by:
1. `_context/milestones/m05-workflow-lab-interface.md` (architecture doc)
2. `skills/web-components/` (patterns once established)

Archive after Milestone 5 Session 05c establishes patterns.

---

## Rules Detail

### rules/go-general.md

**Scope**: Universal Go conventions for this project

**Content**:
- Naming conventions (interfaces, methods, data mutations)
- File structure order (package, imports, const, var, interfaces, types, structs+methods, functions)
- Error handling patterns
- Import organization
- Comment/documentation standards

### rules/go-testing.md

**Scope**: Testing conventions

**Content**:
- Black-box testing (`package name_test`)
- Test file location (`tests/` directory)
- Table-driven test patterns
- Coverage success criteria (critical paths, not percentages)
- What to test vs what to skip

### rules/documentation.md

**Scope**: Documentation and session workflow

**Content**:
- Session closeout checklist
- When to update which documents
- Implementation guide conventions
- Code block conventions (no comments, no tests, no OpenAPI contents)

---

## Skills Detail

### skills/go-packages/SKILL.md

**Triggers**: "package design", "library development", "public API", "pkg/"

**Content**:
- Package organization (cmd/, pkg/, internal/)
- Public vs internal API design
- Configuration patterns (interfaces, finalize/validate)
- Versioning considerations

### skills/go-web-services/SKILL.md

**Triggers**: "handler", "HTTP", "endpoint", "middleware", "routes"

**Content**:
- Handler struct pattern
- Routes() method pattern
- Middleware composition
- Request/response patterns
- Domain filters pattern

### skills/go-database/SKILL.md

**Triggers**: "repository", "database", "query", "migration", "SQL"

**Content**:
- Repository pattern with helpers
- Query builder usage
- Projection maps and scanners
- Migration conventions
- Transaction handling

### skills/openapi/SKILL.md

**Triggers**: "OpenAPI", "openapi.go", "schema", "Spec.", "Scalar"

**Content**:
- Domain-owned OpenAPI pattern
- Schema definition patterns
- Operation definition patterns
- Route integration (OpenAPI field)
- Spec generation flow

### skills/lca-architecture/SKILL.md

**Triggers**: "LCA", "layered composition", "system lifecycle", "cold start", "hot start"

**Content**:
- System vs functional infrastructure
- Cold Start / Hot Start lifecycle
- Initialization pattern (finalize → validate → transform)
- State vs Process separation
- Cross-domain dependencies

### skills/workflow-orchestration/SKILL.md

**Triggers**: "workflow", "orchestration", "StateGraph", "Observer", "checkpoint"

**Content**:
- go-agents-orchestration patterns
- Workflow factory pattern
- Observer implementation
- Checkpoint store pattern
- Executor lifecycle

### skills/document-processing/SKILL.md

**Triggers**: "document-context", "PDF", "image rendering", "ImageMagick"

**Content**:
- document-context integration
- Render options and filters
- Storage integration patterns
- Binary streaming patterns

### skills/web-components/SKILL.md

**Triggers**: "web component", "al-", "custom element", "template", "CSS token"

**Content** (to be established in Milestone 5):
- Component base class patterns
- Atomic design organization
- CSS @layer architecture
- Design token usage
- Go template integration
- SSE client patterns
- Signal usage patterns

### skills/milestone-planning/SKILL.md

**Triggers**: "milestone planning", "planning session", "session breakdown"

**Content**:
- Milestone Planning Session workflow (detailed)
- Architecture document creation
- Session breakdown guidelines
- Post-session review process

---

## Migration Execution Plan

### Phase 1: Structure Setup
1. Create `.claude/` directory structure
2. Create empty rule and skill files with frontmatter
3. Update `.gitignore` if needed

### Phase 2: Rules Migration
1. Extract universal conventions to `rules/go-general.md`
2. Extract testing patterns to `rules/go-testing.md`
3. Extract documentation workflow to `rules/documentation.md`

### Phase 3: Skills Creation
1. Create skill files with proper YAML frontmatter (name, description with triggers)
2. Migrate content from ARCHITECTURE.md to appropriate skills
3. Migrate content from _context/ documents to appropriate skills

### Phase 4: CLAUDE.md Rewrite
1. Create new lean CLAUDE.md
2. Include skill inventory with trigger hints
3. Preserve essential quick references
4. Point to skills for detailed patterns

### Phase 5: Cleanup
1. Archive retired documents to `_context/.archive/`
2. Update any cross-references
3. Validate skill triggering works as expected

---

## Validation Criteria

1. **Baseline context reduced**: New CLAUDE.md + rules significantly smaller than old CLAUDE.md
2. **Progressive disclosure works**: Skills load only when relevant keywords appear
3. **No lost knowledge**: All patterns from current documents exist in new structure
4. **Discoverability**: CLAUDE.md clearly indicates what skills exist
5. **Maintenance path clear**: New patterns have obvious home in skill structure

---

## Open Questions

1. **ARCHITECTURE.md fate**: Lean reference document pointing to skills, or retire entirely?
2. **Skill granularity**: Are the proposed skills too coarse or too fine?
3. **Cross-project reuse**: Should we create a plugin for go-agents ecosystem patterns?
4. **Milestone docs**: Do they stay in `_context/milestones/` or move under `.claude/`?
5. **Existing _context/ usage**: Any documents that should remain as-is?

---

## Reference

- claude-context repository: `~/code/claude-context`
- Claude Code skills documentation: `~/code/claude-context/docs/concepts/skills.md`
- Claude Code rules documentation: `~/code/claude-context/docs/concepts/memory.md`
