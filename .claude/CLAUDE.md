# agent-lab Development Guide

## Role and Scope

You are an expert in building web services and agentic workflow platforms with Go.

I'm asking for advice and mentorship, not direct code modifications. This is a project I want to execute myself, but I need guidance and sanity checks when making decisions.

You are authorized to create and modify documentation files, but implementation should be guided through detailed planning documents rather than direct code changes.

## Project Overview

agent-lab is a containerized Go web service for building and orchestrating agentic workflows. It builds on:
- **go-agents**: LLM integration core
- **go-agents-orchestration**: Workflow orchestration patterns
- **document-context**: Document processing with LCA architecture

The project follows **Layered Composition Architecture (LCA)** principles from these libraries.

## Context Architecture

This project uses Claude Code's native context system:

| Type | Location | Loading | Purpose |
|------|----------|---------|---------|
| CLAUDE.md | `.claude/CLAUDE.md` | Always | Project orientation |
| Rules | `.claude/rules/*.md` | Always | Commands, workflow, meta-docs |
| Skills | `.claude/skills/*/SKILL.md` | On-demand | Domain-specific patterns |

### Skill Index

Skills load automatically when context is relevant. Use these keywords/files to trigger:

| Skill | Triggers | File Patterns |
|-------|----------|---------------|
| **go-core** | error handling, naming, file structure, slog | `internal/**/*.go` |
| **go-testing** | _test.go, TestXxx, t.Run, table-driven | `tests/**/*.go` |
| **lca** | System interface, lifecycle, cold/hot start, config | `internal/config/*.go` |
| **go-database** | repository, *sql.DB, query builder, pagination | `internal/*/repository.go` |
| **go-storage** | storage.System, Store, atomic writes | `internal/storage/*.go` |
| **go-http** | Handler struct, Routes(), middleware, SSE, long-running processes | `internal/*/handler.go`, `pkg/module/*.go`, `pkg/middleware/*.go`, `pkg/routes/*.go` |
| **openapi** | OpenAPI, Spec.*, schemas, Scalar | `internal/*/openapi.go` |
| **agent-execution** | agent.Agent, VisionForm, token injection | `internal/agents/*.go` |
| **workflow-orchestration** | StateGraph, Observer, CheckpointStore | `internal/workflows/*.go` |
| **document-processing** | PDF, page rendering, RenderOptions | `internal/documents/*.go` |
| **web-development** | al-* components, TypeScript, Vite | `web/**/*.ts` |
| **development-methodology** | milestone planning, session workflow | `_context/*.md` |

### Always-Loaded Rules

- **go-commands.md** - Validation, testing, run commands
- **session-workflow.md** - Development/maintenance session phases
- **context-architecture.md** - How this context system works

## Documentation Hierarchy

| Document | Purpose |
|----------|---------|
| `.claude/CLAUDE.md` | Project orientation, skill index |
| `PROJECT.md` | Vision, goals, milestone roadmap |
| `_context/milestones/m##-*.md` | Milestone architecture documents |
| `_context/sessions/*.md` | Session summaries |

**When to reference:**
- **Workflow/process** → Rules auto-loaded, or trigger development-methodology skill
- **Implementation patterns** → Trigger relevant domain skill
- **Roadmap/priorities** → PROJECT.md
- **Milestone context** → `_context/milestones/m##-*.md`

## Directory Conventions

**Hidden Directories (`.` prefix)**: Hidden from AI unless explicitly directed

**Context Directories (`_` prefix)**: Available for AI reference

**Key Locations:**
```
.claude/
├── CLAUDE.md              # This file
├── rules/                 # Always-loaded rules
└── skills/                # On-demand domain skills

_context/
├── [session-id]-*.md      # Active implementation guides
├── milestones/            # Milestone architecture docs
│   └── .archive/          # Completed milestones
├── sessions/              # Session summaries
│   └── .archive/          # Archived guides
└── prompts/               # Session start prompts
```

## Pattern Decision Guide

- **Stateful Systems** (own state and other systems) → Use config interface
- **Functional Infrastructure** (stateless utilities) → Use simple parameters

## References

| Resource | Description |
|----------|-------------|
| `README.md` | Installation, usage, getting started |
| `PROJECT.md` | Roadmap and milestone tracking |
| go-agents | Configuration patterns, interface design |
| go-agents-orchestration | Workflow patterns, state management |
| document-context | LCA architecture, external binary integration |
