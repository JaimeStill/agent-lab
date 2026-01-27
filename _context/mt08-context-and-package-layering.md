# mt08: Context Optimization and Package Layering

## Overview

This maintenance session addresses two related infrastructure concerns:

1. **Context Optimization** - Restructure `.claude/` for automatic skill triggering
2. **Package Layering Fix** - Eliminate `pkg/` → `internal/` dependency violation

Additionally, the **web-development skill** is completely rewritten to reflect the new Lit-based architecture validated in the go-lit POC.

---

## Phase 1: Context Optimization

### 1.1 Delete Rules Directory

The `rules/` directory is not part of Claude Code's spec. Delete it after consolidating content:

```bash
rm -rf .claude/rules/
```

### 1.2 Rewrite CLAUDE.md

Replace the current CLAUDE.md with a consolidated version under 200 lines:

**File:** `.claude/CLAUDE.md`

```markdown
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
│   ├── api/             # API module assembly
│   └── <domain>/        # Domain systems
├── pkg/                 # Shared infrastructure (public)
│   ├── config/          # Configuration types
│   ├── runtime/         # Infrastructure composition
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

## References

| Resource | Description |
|----------|-------------|
| PROJECT.md | Roadmap, milestones, success criteria |
| `_context/milestones/` | Milestone architecture documents |
| go-agents | LLM integration, agent patterns |
| go-agents-orchestration | Workflow orchestration |
| document-context | Document processing |
```

### 1.3 Update Skill Descriptions

Update each skill's YAML frontmatter to use trigger-optimized descriptions with "REQUIRED for..." pattern:

**go-core:**
```yaml
description: >
  REQUIRED for Go code organization and error handling. Use when creating
  packages, defining domain errors, naming interfaces, or using slog.
  Triggers: errors.go, Err prefix, interface naming, package structure,
  slog, structured logging, "where should I put", "error handling".
```

**go-testing:**
```yaml
description: >
  REQUIRED for writing Go tests. Use when creating test files, writing
  table-driven tests, or checking coverage.
  Triggers: _test.go, TestXxx, t.Run, black-box testing, table-driven,
  test coverage, "write tests for", "how do I test".
```

**lca:**
```yaml
description: >
  REQUIRED for Layered Composition Architecture. Use when implementing
  domain systems, lifecycle coordination, Infrastructure, or Handler() factory.
  Triggers: System interface, Cold Start, Hot Start, New* constructor,
  Start(), Infrastructure, Handler(), state flows down, config.go, Finalize().
```

**go-database:**
```yaml
description: >
  REQUIRED for database access patterns. Use when writing repositories,
  building queries, implementing pagination, or handling transactions.
  Triggers: repository.go, sql.DB, sql.Tx, query.Builder, ProjectionMap,
  QueryOne, QueryMany, WithTx, pagination, mapping.go, ScanFunc.
```

**go-http:**
```yaml
description: >
  REQUIRED for HTTP handler implementation. Use when creating handlers,
  defining routes, implementing middleware, or SSE streaming.
  Triggers: handler.go, Handler struct, Routes(), middleware, module.Module,
  RespondJSON, RespondError, CORS, SSE, "create endpoint", "HTTP handler".
```

**go-storage:**
```yaml
description: >
  REQUIRED for blob storage operations. Use when implementing file
  persistence, storage validation, or path handling.
  Triggers: storage.System, Store, Retrieve, Delete, Validate, Path,
  atomic writes, path traversal, storage keys, filesystem.
```

**openapi:**
```yaml
description: >
  REQUIRED for OpenAPI specification. Use when defining API schemas,
  documenting endpoints, or generating specs.
  Triggers: openapi.go, Spec.*, SchemaRef, RequestBodyJSON, ResponseRef,
  Schemas(), "document API", "OpenAPI schema", Scalar UI.
```

**agent-execution:**
```yaml
description: >
  REQUIRED for agent execution patterns. Use when implementing LLM calls,
  vision forms, token handling, or streaming responses.
  Triggers: agent.Agent, VisionForm, ParseVisionForm, ChatStream, VisionStream,
  constructAgent, multipart/form-data, base64, token injection.
```

**workflow-orchestration:**
```yaml
description: >
  REQUIRED for workflow implementation. Use when building state graphs,
  implementing observers, or managing checkpoints.
  Triggers: StateGraph, WorkflowFactory, Observer, CheckpointStore,
  ProcessParallel, graph.Execute, graph.Resume, workflow_runs.
```

**document-processing:**
```yaml
description: >
  REQUIRED for document processing. Use when working with PDFs, image
  rendering, or document-context library integration.
  Triggers: document-context, PDF, page rendering, RenderOptions,
  page range, documents domain, images domain, pdfcpu, ImageMagick.
```

**web-development:**
```yaml
description: >
  REQUIRED for web client development with Lit. Use when creating views,
  components, elements, services, or styling with CSS layers.
  Triggers: web/app/client/, LitElement, @customElement, @provide, @consume,
  SignalWatcher, design/tokens, "create component", "add view".
```

**development-methodology:**
```yaml
description: >
  REQUIRED for development process. Use when planning milestones,
  starting sessions, creating guides, or conducting reviews.
  Triggers: milestone planning, session workflow, implementation guide,
  PROJECT.md, session closeout, maintenance session, "start session".
```

---

## Phase 2: Package Layering Fix

### 2.1 Create pkg/config Package

**File:** `pkg/config/types.go`

```go
package config

import (
	"fmt"
	"log/slog"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

func (l LogLevel) Validate() error {
	switch l {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
		return nil
	default:
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", l)
	}
}

func (l LogLevel) ToSlogLevel() slog.Level {
	switch l {
	case LogLevelDebug:
		return slog.LevelDebug
	case LogLevelWarn:
		return slog.LevelWarn
	case LogLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type LogFormat string

const (
	LogFormatText LogFormat = "text"
	LogFormatJSON LogFormat = "json"
)

func (f LogFormat) Validate() error {
	switch f {
	case LogFormatText, LogFormatJSON:
		return nil
	default:
		return fmt.Errorf("invalid log format: %s (must be text or json)", f)
	}
}
```

**File:** `pkg/config/logging.go`

```go
package config

import "os"

const (
	EnvLoggingLevel  = "LOGGING_LEVEL"
	EnvLoggingFormat = "LOGGING_FORMAT"
)

type LoggingConfig struct {
	Level  LogLevel  `toml:"level"`
	Format LogFormat `toml:"format"`
}

func (c *LoggingConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()
	return c.validate()
}

func (c *LoggingConfig) Merge(overlay *LoggingConfig) {
	if overlay.Level != "" {
		c.Level = overlay.Level
	}
	if overlay.Format != "" {
		c.Format = overlay.Format
	}
}

func (c *LoggingConfig) loadDefaults() {
	if c.Level == "" {
		c.Level = LogLevelInfo
	}
	if c.Format == "" {
		c.Format = LogFormatText
	}
}

func (c *LoggingConfig) loadEnv() {
	if v := os.Getenv(EnvLoggingLevel); v != "" {
		c.Level = LogLevel(v)
	}
	if v := os.Getenv(EnvLoggingFormat); v != "" {
		c.Format = LogFormat(v)
	}
}

func (c *LoggingConfig) validate() error {
	if err := c.Level.Validate(); err != nil {
		return err
	}
	return c.Format.Validate()
}
```

### 2.2 Update pkg/runtime

**File:** `pkg/runtime/logging.go`

Update imports from `internal/config` to `pkg/config`:

```go
package runtime

import (
	"log/slog"
	"os"

	"github.com/JaimeStill/agent-lab/pkg/config"
)

func newLogger(cfg *config.LoggingConfig) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: cfg.Level.ToSlogLevel(),
	}

	var handler slog.Handler
	if cfg.Format == config.LogFormatJSON {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
```

**File:** `pkg/runtime/infrastructure.go`

Update to accept an interface instead of concrete `*config.Config`:

```go
package runtime

import (
	"fmt"
	"log/slog"

	"github.com/JaimeStill/agent-lab/pkg/config"
	"github.com/JaimeStill/agent-lab/pkg/database"
	"github.com/JaimeStill/agent-lab/pkg/lifecycle"
	"github.com/JaimeStill/agent-lab/pkg/storage"
)

type InfrastructureConfig interface {
	LoggingConfig() *config.LoggingConfig
	DatabaseConfig() *database.Config
	StorageConfig() *storage.Config
}

type Infrastructure struct {
	Lifecycle *lifecycle.Coordinator
	Logger    *slog.Logger
	Database  database.System
	Storage   storage.System
}

func New(cfg InfrastructureConfig) (*Infrastructure, error) {
	lc := lifecycle.New()
	logger := newLogger(cfg.LoggingConfig())

	db, err := database.New(cfg.DatabaseConfig(), logger)
	if err != nil {
		return nil, fmt.Errorf("database init failed: %w", err)
	}

	store, err := storage.New(cfg.StorageConfig(), logger)
	if err != nil {
		return nil, fmt.Errorf("storage init failed: %w", err)
	}

	return &Infrastructure{
		Lifecycle: lc,
		Logger:    logger,
		Database:  db,
		Storage:   store,
	}, nil
}

func (i *Infrastructure) Start() error {
	if err := i.Database.Start(i.Lifecycle); err != nil {
		return fmt.Errorf("database start failed: %w", err)
	}
	if err := i.Storage.Start(i.Lifecycle); err != nil {
		return fmt.Errorf("storage start failed: %w", err)
	}
	return nil
}
```

### 2.3 Update internal/config

**File:** `internal/config/config.go`

Remove `LogLevel` and `LogFormat` types (now in `pkg/config`), embed `pkg/config.LoggingConfig`:

```go
package config

import (
	"fmt"
	"os"
	"time"

	pkgconfig "github.com/JaimeStill/agent-lab/pkg/config"
	"github.com/JaimeStill/agent-lab/pkg/database"
	"github.com/JaimeStill/agent-lab/pkg/storage"
	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Server          ServerConfig           `toml:"server"`
	Database        database.Config        `toml:"database"`
	Logging         pkgconfig.LoggingConfig `toml:"logging"`
	Storage         storage.Config         `toml:"storage"`
	API             APIConfig              `toml:"api"`
	Domain          string                 `toml:"domain"`
	ShutdownTimeout string                 `toml:"shutdown_timeout"`
	Version         string                 `toml:"version"`
}

func (c *Config) LoggingConfig() *pkgconfig.LoggingConfig {
	return &c.Logging
}

func (c *Config) DatabaseConfig() *database.Config {
	return &c.Database
}

func (c *Config) StorageConfig() *storage.Config {
	return &c.Storage
}
```

**File:** `internal/config/types.go`

Delete this file (types moved to `pkg/config/types.go`).

**File:** `internal/config/logging.go`

Delete this file (moved to `pkg/config/logging.go`).

---

## Phase 3: Web Development Skill Rewrite

The complete rewrite is a separate deliverable: `.claude/skills/web-development/SKILL.md`

See the plan deliverables for the full content based on go-lit POC patterns.

---

## Validation

### Package Layering

```bash
go vet ./...
grep -r "internal/config" pkg/
```

Second command should return no results.

### Context Optimization

```bash
wc -l .claude/CLAUDE.md
ls .claude/rules/
```

CLAUDE.md should be under 200 lines. Rules directory should not exist.

### Automatic Triggering Test

Test with natural language prompts:
- "How should I structure this domain?" → should trigger go-core
- "Write tests for this function" → should trigger go-testing
- "Create an HTTP handler" → should trigger go-http
- "Implement a workflow" → should trigger workflow-orchestration

---

## Closeout Checklist

1. [ ] Delete `.claude/rules/` directory
2. [ ] Rewrite `.claude/CLAUDE.md` (under 200 lines)
3. [ ] Update all 12 skill descriptions with trigger-optimized format
4. [ ] Create `pkg/config/` with types.go and logging.go
5. [ ] Update `pkg/runtime/` to use pkg/config
6. [ ] Update `internal/config/` to embed pkg/config types
7. [ ] Delete `internal/config/types.go` and `internal/config/logging.go`
8. [ ] Complete rewrite of `.claude/skills/web-development/SKILL.md`
9. [ ] Run `go vet ./...` - passes
10. [ ] Verify no `pkg/` → `internal/` imports
11. [ ] Archive this guide to `_context/sessions/.archive/`
12. [ ] Create summary at `_context/sessions/mt08-context-and-package-layering.md`
13. [ ] Update PROJECT.md with maintenance session status
