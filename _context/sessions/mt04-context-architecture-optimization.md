# MT04: Context Architecture Optimization

## Summary

Restructured agent-lab's context documentation from monolithic files (~4,900+ lines always loaded) to Claude Code's native skills/rules system with progressive disclosure.

## What Was Done

### New Context Architecture

Created `.claude/` structure with:
- **CLAUDE.md** (105 lines): Lean navigation with skill index
- **3 Rules** (138 lines): Always-loaded workflow and command references
- **12 Skills**: On-demand domain-specific patterns

### Always-Loaded Content (243 lines total)

| File | Lines | Purpose |
|------|-------|---------|
| CLAUDE.md | 105 | Project orientation, skill index |
| rules/go-commands.md | 35 | Validation, testing, run commands |
| rules/session-workflow.md | 43 | Session phases, closeout checklist |
| rules/context-architecture.md | 60 | Meta-docs about context system |

### On-Demand Skills

| Skill | Domain |
|-------|--------|
| go-core | File structure, error handling, naming |
| go-testing | Black-box testing, table-driven tests |
| lca | System lifecycle, cold/hot start, config |
| go-database | Repository, query builder, pagination |
| go-storage | Blob storage, atomic writes |
| go-http | Handlers, middleware, routes, SSE |
| openapi | API specification, schemas |
| agent-execution | Token injection, vision forms, streaming |
| workflow-orchestration | StateGraph, observers, checkpoints |
| document-processing | PDF rendering, image enhancement |
| web-components | Placeholder for M5 |
| development-methodology | Milestone-anchored workflow |

### Archived Documents

Moved to `_context/.archive/`:
- ARCHITECTURE.md (76KB) - Source code is now the reference
- CLAUDE.md (17KB) - Replaced by lean .claude/CLAUDE.md
- web-service-architecture.md (34KB) - Patterns extracted to skills
- service-design.md (3KB) - Concepts integrated into codebase

### Updated Session Prompts

Simplified `_context/prompts/*.md` to reference auto-loaded rules instead of explicit CLAUDE.md instructions.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| ARCHITECTURE.md | Archive | Source code is the reference; skills handle patterns |
| Rules scope | Minimal (~138 lines) | Only workflow + commands |
| Skill count | 12 | Balance between granularity and discoverability |
| go-core vs lca | Separate skills | Go principles distinct from LCA patterns |
| go-testing | Standalone skill | Testing patterns are a distinct concern |

## Patterns Established

- **YAML frontmatter** for skill triggering with `name` and `description` fields
- **Keyword triggers** in description for context-aware loading
- **File pattern triggers** for automatic skill activation
- **Directory conventions**: `.archive/` for retired documents

## Next Steps

- AGENTS.md decision pending (keep for curl examples or archive)
- web-components skill to be populated during M5
- Monitor skill triggering effectiveness in future sessions
