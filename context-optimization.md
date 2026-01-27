# Context Optimization Recommendations for agent-lab

This document provides recommendations for restructuring agent-lab's `.claude/` context system to enable automatic skill triggering, aligned with Claude Code's official plugin and skill specifications.

---

## Current State Analysis

### Issues Identified

1. **Non-standard `rules/` directory**: Claude Code has no `rules/` concept. Content in `rules/` is not automatically loaded.

2. **Skill descriptions lack trigger phrases**: Current descriptions are informational rather than directive. They don't tell Claude *when* to invoke the skill.

3. **Manual invocation required**: Skills must be explicitly called because Claude can't match natural requests to skill descriptions.

### Current Structure
```
.claude/
├── CLAUDE.md                    # Orientation + skill index
├── rules/                       # NON-STANDARD - not auto-loaded
│   ├── go-commands.md
│   ├── session-workflow.md
│   └── context-architecture.md
└── skills/                      # 12 domain skills
    ├── go-core/SKILL.md
    ├── go-testing/SKILL.md
    ├── lca/SKILL.md
    └── ...
```

---

## Recommended Changes

### 1. Consolidate Rules into CLAUDE.md

The `rules/` directory is not part of Claude Code's spec. Content that should always be loaded must go in `CLAUDE.md`.

**Action**: Merge critical rules into CLAUDE.md sections:

```markdown
# CLAUDE.md

## Quick Reference

### Commands
- Validation: `go vet ./...`
- Testing: `go test ./tests/...`
- Running: `go run ./cmd/server`
- Migrations: `go run ./cmd/migrate -up`

### Session Workflow
Development sessions follow: Planning → Implementation → Validation → Documentation → Closeout
```

Keep CLAUDE.md under 200 lines. Link to detailed docs rather than duplicating content.

### 2. Restructure Skills with Trigger-Optimized Descriptions

The `description` field determines automatic invocation. Rewrite descriptions using the pattern that works (e.g., omarchy skill):

**Pattern for automatic triggering:**
```yaml
---
name: skill-name
description: >
  REQUIRED for [specific scenario]. Use when [explicit triggers].
  Triggers: [keyword1], [keyword2], [file patterns], [action verbs].
---
```

### 3. Skill Description Rewrites

#### go-core (Current)
```yaml
description: >
  File structure, errors.go, naming, slog, domain system
```

#### go-core (Optimized)
```yaml
description: >
  REQUIRED for Go file organization and error handling patterns.
  Use when creating new packages, defining errors, naming interfaces,
  or setting up structured logging with slog.
  Triggers: errors.go, interface naming, package structure, slog logger,
  "where should I put", "how should I name", "error handling pattern".
```

---

#### go-testing (Current)
```yaml
description: >
  _test.go, TestXxx, black-box testing, coverage
```

#### go-testing (Optimized)
```yaml
description: >
  REQUIRED for writing Go tests. Use when creating test files,
  writing table-driven tests, or checking test coverage.
  Triggers: _test.go, TestXxx, black-box testing, test coverage,
  "write tests for", "how do I test", "table-driven test".
```

---

#### lca (Current)
```yaml
description: >
  System interface, lifecycle, Infrastructure, Handler() factory
```

#### lca (Optimized)
```yaml
description: >
  REQUIRED for Layered Composition Architecture patterns. Use when
  implementing domain systems, lifecycle coordination, or Handler() factories.
  Triggers: System interface, Cold Start, Hot Start, state flows down,
  Infrastructure, Handler() factory, "implement a system", "domain system".
```

---

#### go-database (Current)
```yaml
description: >
  repository.go, sql.DB, query.Builder, pagination
```

#### go-database (Optimized)
```yaml
description: >
  REQUIRED for database access patterns. Use when writing repositories,
  building queries, implementing pagination, or handling transactions.
  Triggers: repository.go, sql.DB, query.Builder, ProjectionMap,
  QueryOne, QueryMany, WithTx, pagination, "database query", "repository pattern".
```

---

#### go-http (Current)
```yaml
description: >
  Handler struct, Routes(), middleware, SSE, path normalization
```

#### go-http (Optimized)
```yaml
description: >
  REQUIRED for HTTP handler implementation. Use when creating handlers,
  defining routes, implementing middleware, or setting up SSE streaming.
  Triggers: handler.go, Handler struct, Routes(), middleware, SSE,
  RespondJSON, RespondError, "create an endpoint", "HTTP handler".
```

---

#### openapi (Current)
```yaml
description: >
  openapi.go, Spec.*, schemas, Scalar UI
```

#### openapi (Optimized)
```yaml
description: >
  REQUIRED for OpenAPI specification. Use when defining API schemas,
  documenting endpoints, or generating OpenAPI specs.
  Triggers: openapi.go, Spec.*, SchemaRef, RequestBodyJSON, ResponseRef,
  "document this API", "OpenAPI schema", "API spec".
```

---

#### workflow-orchestration (Current)
```yaml
description: >
  StateGraph, Observer, CheckpointStore
```

#### workflow-orchestration (Optimized)
```yaml
description: >
  REQUIRED for workflow and state graph implementation. Use when
  building workflows, implementing observers, or managing checkpoints.
  Triggers: StateGraph, WorkflowFactory, Observer, CheckpointStore,
  executor, "create a workflow", "state machine", "checkpoint".
```

---

### 4. Remove Non-Standard Directories

After consolidating into CLAUDE.md and optimizing skills:

```
.claude/
├── CLAUDE.md                    # Always loaded - orientation + quick reference
├── skills/                      # Auto-triggered by description matching
│   ├── go-core/SKILL.md
│   ├── go-testing/SKILL.md
│   ├── lca/SKILL.md
│   ├── go-database/SKILL.md
│   ├── go-http/SKILL.md
│   ├── openapi/SKILL.md
│   ├── workflow-orchestration/SKILL.md
│   ├── agent-execution/SKILL.md
│   ├── document-processing/SKILL.md
│   ├── go-storage/SKILL.md
│   ├── web-development/SKILL.md
│   └── development-methodology/SKILL.md
└── agents/                      # Custom subagent definitions (if any)
```

### 5. CLAUDE.md Template

```markdown
# agent-lab

A reference web service platform demonstrating Go patterns with go-agents integration.

## Quick Reference

### Commands
| Action | Command |
|--------|---------|
| Validate | `go vet ./...` |
| Test | `go test ./tests/...` |
| Run | `go run ./cmd/server` |
| Migrate Up | `go run ./cmd/migrate -up` |
| Migrate Down | `go run ./cmd/migrate -down` |
| Seed | `go run ./cmd/seed -all` |

### Session Workflow
- **Development**: Planning → Implementation → Validation → Documentation → Closeout
- **Maintenance**: Planning → Execution → Validation → Closeout

### Architecture Overview
- **Layered Composition Architecture (LCA)**: Cold Start builds state, Hot Start activates
- **Domain Systems**: Implement System interface with Handler() factory
- **File Organization**: errors.go, entity.go, system.go, repository.go, handler.go, openapi.go

## Skills Index

Skills load automatically based on context. Available skills:

| Skill | Triggers When |
|-------|---------------|
| go-core | Creating packages, defining errors, naming interfaces |
| go-testing | Writing tests, coverage analysis |
| lca | Implementing domain systems, lifecycle patterns |
| go-database | Repository implementation, queries, transactions |
| go-http | HTTP handlers, routes, middleware, SSE |
| openapi | API documentation, schema definitions |
| workflow-orchestration | State graphs, observers, checkpoints |
| agent-execution | Agent integration, vision forms, token injection |
| document-processing | PDF rendering, document-context integration |
| go-storage | Blob storage, atomic writes |
| web-development | Frontend components, templates, Vite |
| development-methodology | Session planning, milestones |

## Project Structure

```
agent-lab/
├── cmd/server/          # Entry point
├── internal/            # Domain systems
│   ├── <domain>/
│   │   ├── errors.go
│   │   ├── <entity>.go
│   │   ├── system.go
│   │   ├── repository.go
│   │   ├── handler.go
│   │   └── openapi.go
├── pkg/                 # Shared infrastructure
└── web/                 # Frontend clients
```
```

---

## Verification Checklist

After restructuring:

- [ ] `rules/` directory removed
- [ ] Critical content merged into CLAUDE.md
- [ ] All skill descriptions rewritten with trigger phrases
- [ ] Each description includes "REQUIRED for...", "Use when...", "Triggers:"
- [ ] CLAUDE.md under 200 lines
- [ ] Test automatic triggering by asking natural questions:
  - "How should I structure this domain?"
  - "Write tests for this function"
  - "Create an HTTP handler for users"
  - "Implement a workflow for document processing"

---

## Migration Steps

1. **Backup current `.claude/` directory**
2. **Rewrite CLAUDE.md** using template above
3. **Rewrite each skill description** using optimized patterns
4. **Delete `rules/` directory**
5. **Test automatic triggering** with natural language requests
6. **Iterate on descriptions** based on what triggers and what doesn't

---

## References

- [Claude Code Skills Documentation](https://code.claude.com/docs/en/skills)
- [Claude Code Plugins Documentation](https://code.claude.com/docs/en/plugins)
