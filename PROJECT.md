# agent-lab Project

**Status**: Active Development - Milestone 5 In Progress

## Overview

agent-lab is a containerized Go web service platform for building and orchestrating agentic workflows. It provides a laboratory environment for iteratively designing, testing, and refining intelligent workflows, then deploying them operationally.

Built on the go-agents ecosystem:
- **go-agents** (v0.3.0): LLM integration with multi-provider support
- **go-agents-orchestration** (v0.1.0+): Workflow orchestration patterns
- **document-context** (v0.1.1): PDF processing with LCA architecture

## Long-Term Vision

### Core Value Proposition

agent-lab enables organizations to:

1. **Iteratively Develop Intelligent Workflows**: Design multi-agent workflows with full observability into decision-making and confidence scoring
2. **Refine Through Experimentation**: Test workflow configurations, analyze results, adjust parameters, and re-execute without redeployment
3. **Deploy Operationally**: Transition refined workflows from lab environment to production with bulk processing and monitoring
4. **Maintain Enterprise Standards**: RBAC, audit logging, air-gap deployment, and compliance requirements

### Platform Capabilities

**Workflow Lab Environment**:
- Multi-agent workflow designer with configurable stages
- Real-time execution monitoring via SSE streaming
- Detailed execution traces showing agent reasoning and confidence scores
- Side-by-side comparison of workflow variants
- Document preview with detected markings visualization
- Confidence score evolution graphs (D3.js visualizations)

**Operational Deployment**:
- One-click workflow execution from lab to operations
- Bulk document processing with queue-based execution
- Standardized output formats integrating with existing systems
- Webhook notifications for completion events
- Result export (JSON, JSONL, CSV)

**Ecosystem Integration**:
- Complete go-agents ecosystem through unified HTTP API
- Multi-provider LLM support (Ollama, Azure AI Foundry)
- Adaptive document processing with filter adjustments
- OpenAPI specification with Scalar interactive interface

**Enterprise Ready**:
- RBAC with resource ownership and sharing (Phase 8)
- Audit logging for compliance requirements
- Azure cloud integration (Phase 8)
- Air-gap deployable with embedded assets

### Technology Principles

1. **Build in CI, Deploy Without Node.js**: Web assets built during CI, embedded in Go binary, container has no Node.js runtime
2. **Minimal Dependencies**: Only essential, industry-recognized libraries
3. **Embedded Assets**: All dependencies embedded via `go:embed`, self-hosted
4. **Tree-Shaken Bundles**: Vite builds eliminate unused code for optimal bundle sizes
5. **Standards-Forward**: Lit Web Components, TC39 Signals, SSE, Fetch API
6. **Air-Gap Compatible**: Single Go binary with embedded assets for air-gapped environments

## Success Criteria

### Primary Goal

**Refine classify-docs workflow from 96.3% → 100% accuracy** through iterative experimentation enabled by agent-lab tooling.

The classify-docs prototype demonstrates:
- Document classification via vision API analysis
- Per-page processing with context accumulation
- Conservative confidence scoring (HIGH/MEDIUM/LOW)
- 96.3% accuracy on 27-document test set

agent-lab will enable experimentation with:
- Multi-stage workflows (marking identification → classification → QA)
- Agent collaboration with feedback loops
- Adaptive image processing for problematic pages
- Semantic confidence scoring (0.0-1.0 scale with tangible factors)
- Two-person integrity through QA agent validation

### Secondary Goals

1. **Production-Ready Classification Platform**: Deliver reliable document classification capability meeting customer requirements
2. **Foundation for Multi-Workflow Orchestration**: Establish patterns applicable to data extraction, content generation, analysis, and translation workflows

### Success Metrics

- Classification accuracy: 100% on test document set
- Workflow iteration cycle: < 5 minutes (design → test → analyze → adjust)
- Execution observability: Complete trace of agent decisions and confidence factors
- Operational reliability: Bulk processing with error handling and retry logic

## Technology Stack

### Backend

- **Language**: Go 1.25.2+
- **Database**: PostgreSQL 17 (containerized)
- **Data Access**: Raw SQL with `database/sql` + pgx driver
- **Templating**: `html/template` for server-side rendering
- **Asset Management**: `go:embed` for embedded static assets
- **Libraries**:
  - go-agents v0.3.0 (LLM integration)
  - go-agents-orchestration v0.1.0+ (workflow patterns)
  - document-context v0.1.1 (PDF processing)

### Frontend

- **Build**: Bun + Vite + TypeScript (CI only, not in container)
- **Framework**: Lit 3.x for web components (`@lit/context`, `@lit-labs/signals`)
- **State Management**: Signal-based reactivity via `@lit-labs/signals`
- **Real-Time**: Server-Sent Events (SSE) for execution monitoring
- **HTTP**: Fetch API for REST interactions
- **Visualization**: D3.js (embedded) for confidence score graphs
- **Architecture**: See `_context/milestones/m05-workflow-lab-interface.md` for details

### API

- **REST**: CRUD operations for resources (providers, agents, workflows, documents)
- **SSE**: Real-time event streaming for execution progress
- **OpenAPI**: Specification with Scalar interactive documentation
- **HTML Fragments**: Server-side rendered partials for dynamic updates

### Deployment

- **Development**: Docker Compose (PostgreSQL 17 + agent-lab service)
- **Production**: Kubernetes (Phase 8)
- **External Dependencies**: PostgreSQL 17, ImageMagick 7+
- **Cloud Platform**: Azure (Phase 8)

## Architecture Principles

See `.claude/skills/` for domain-specific patterns (loaded on-demand via context system).

**Key Principles**:
- **State Flows Down, Never Up**: State flows through method parameters unless owned by the object/process
- **Systems, Not Services/Models**: Use domain-specific terminology, clear separation of stateful systems vs functional infrastructure
- **Cold Start/Hot Start Lifecycle**: State initialization (`New*()`) separate from process activation (`Start()`)
- **Configuration-Driven Initialization**: Encapsulated config interfaces with finalize → validate → transform pattern
- **Package Organization**: cmd/server (process), pkg/ (public API), internal/ (private API)
- **Async-First Execution**: All workflows non-blocking with real-time monitoring
- **Experimental Platform**: Provide primitives for workflow experimentation, not prescribed implementations

## Development Process

### Milestone Structure

**Milestones** are high-level stepping stones toward the project vision and goals. Each milestone represents a complete, validated capability that moves the platform forward.

**Development Sessions** are focused, manageable implementation chunks that complete a milestone incrementally. Each session should be completable in 2-3 hours.

### Workflow

1. **Milestone Planning** - Break milestone into focused development sessions
2. **Session Execution** - Implement → Validate → Commit
3. **Milestone Review** - After all sessions complete, review and adjust
4. **Milestone Completion** - Confirm all success criteria met

**Benefits**:
- Incremental progress with regular validation
- Manageable cognitive load per session
- Clear completion criteria at each level
- Regular commit points for working code

Session workflow is documented in the development-methodology skill. See `.claude/CLAUDE.md` for project orientation.

---

## Iterative Milestones

### Milestone 1: Foundation - Provider & Agent Configuration Management

**Objective**: Establish foundation with core systems for agent configuration management.

**Development Sessions**:

#### Session 1a: Foundation Infrastructure ✅

**Status**: Completed (2025-11-24)

**Implemented**:
- Configuration system (TOML base + overlay + env var atomic precedence)
- Logger system (slog-based with configurable level/format)
- Routes system (registration and grouping support)
- Middleware system (composable stack with logger and CORS)
- Server system (HTTP lifecycle with graceful shutdown)
- Service composition (cold start, hot start, coordinated shutdown)
- Health check endpoint (`/healthz`)
- Comprehensive testing (100% coverage, black-box, table-driven)
- Complete godoc documentation

**Validation**: ✅ Service starts, health check responds, graceful shutdown works, all tests passing

**Architectural Decisions**:
- Configuration is ephemeral (not stored in systems)
- Atomic config replacement (not merging)
- Simplified finalize pattern (one method: defaults → env → validate)
- Local env var constants (co-located with config sections)
- Runtime config updates scrapped (k8s makes unnecessary)

**Impact on Future Sessions**: Session 1b will not implement runtime configuration updates (removed from scope). Database and query infrastructure remain as planned.

#### Session 1b: Database & Query Infrastructure ✅

**Status**: Completed (2025-11-25)

**Implemented**:
- Lifecycle coordinator system (internal/lifecycle) for startup/shutdown orchestration
- Database system with connection pool and lifecycle integration
- Migration CLI tool (cmd/migrate) with embedded migrations
- Query builder infrastructure (pkg/query) - ProjectionMap and Builder
- Pagination utilities (pkg/pagination) - Config, PageRequest, PageResult
- Readiness endpoint (`/readyz`) reflecting subsystem operational state
- Comprehensive testing (100% coverage for new packages)
- Complete godoc documentation

**Validation**: ✅ Database connects, migrations run, `/readyz` reflects readiness state, graceful shutdown works

**Architectural Additions**:
- Lifecycle Coordinator pattern: Centralizes startup/shutdown orchestration
- ReadinessChecker interface: Decouples readiness from concrete coordinator
- OnStartup/OnShutdown hooks: Subsystems register lifecycle behaviors
- One-time readiness gate: Ready() becomes true after WaitForStartup()

#### Session 1c: Runtime/Domain System Separation + Providers System ✅

**Status**: Completed (2025-11-26)

**Implemented**:
- Runtime/Domain system separation pattern
  - Runtime struct: Lifecycle, Logger, Database, Pagination (application-scoped)
  - Domain struct: Providers system (request-scoped behavior)
  - Server struct: runtime, domain, http
- Database schema: `providers` table with migration
- Providers domain system (repository pattern with query builder)
- Provider CRUD + Search endpoints with go-agents config validation
- Domain error pattern with HTTP status code mapping
- Logger simplified to `*slog.Logger` (not a System interface)
- Config Load() consolidation (includes finalization internally)

**Validation**: ✅ All provider endpoints working (Create/Read/Update/Delete/Search), error handling verified, graceful shutdown works

**Architectural Additions**:
- Runtime vs Domain: Clear separation of lifecycle-managed infrastructure from stateless business logic
- Domain systems pre-initialized at startup, stored in Server struct
- Logger as functional infrastructure (not a System)
- Domain errors defined in `errors.go` with handler mapping to HTTP status codes

#### Session 1d: Domain Infrastructure Patterns ✅

**Status**: Completed (2025-12-02)

**Implemented**:
- `pkg/repository` package: `WithTx`, `QueryOne`, `QueryMany`, `ExecExpectOne`, `MapError`
- `pkg/handlers` package: `RespondJSON`, `RespondError` stateless functions
- `pkg/query` enhancements: `SortField`, `ParseSortFields`, `OrderByFields`, `NewBuilder` with variadic default sort
- `pkg/pagination`: `PageRequestFromQuery`, `PageRequest.Sort []SortField` (removed old `SortBy`/`Descending`)
- Domain filter pattern: `Filters` struct, `FiltersFromQuery`, `Apply(*query.Builder)`
- Handler struct pattern: `Handler` with `Routes()` method, replaces closure-based wiring
- Providers refactored: scanner.go, filters.go, handler.go (deleted handlers.go, routes.go)
- New endpoint: `GET /api/providers` with query parameter support

**Validation**: ✅ All endpoints working, sorting/filtering/search verified, 78.5% test coverage

**Architectural Additions**:
- Repository Helpers: Generic transaction and query execution
- Domain Scanner: `ScanFunc[T]` defined in domain packages
- Handler Struct: Self-contained handler with `Routes()` method
- Domain Filters: Filter struct with `FiltersFromQuery` and `Apply`
- HTTP Status Mapping: `MapHTTPStatus(error)` in domain errors.go

#### Session 1e: Agents System ✅

**Status**: Completed (2025-12-02)

**Implemented**:
- Database schema: `agents` table with JSONB config (decoupled from providers)
- Agents domain system following refined patterns from 1d
- Agent CRUD + Search endpoints with GET and POST variants
- Agent execution endpoints: Chat, ChatStream, Vision, VisionStream, Tools, Embed
- SSE streaming for Chat and Vision endpoints
- VisionForm pattern for multipart/form-data image uploads
- Token injection pattern for Azure authentication at request time
- Config validation via go-agents agent.New() during create/update

**Validation**: ✅ All CRUD endpoints working, execution endpoints tested with Ollama and Azure agents

**Architectural Additions**:
- VisionForm Pattern: Centralized multipart parsing with base64 image conversion
- Token Injection: Runtime token injection into Provider.Options["token"]
- SSE Streaming: Standard text/event-stream format with flush after each chunk
- Handler with Execution: Handler struct with both CRUD and execution methods

#### Session 1f: OpenAPI Specification & Scalar UI Integration ✅

**Status**: Completed (2025-12-05)

**Implemented**:
- OpenAPI 3.1 specification infrastructure (`pkg/openapi`)
- Domain-owned OpenAPI schemas and operations (`internal/providers/openapi.go`, `internal/agents/openapi.go`)
- Integrated spec generation at server startup (`cmd/server/openapi.go`)
- Environment-specific spec caching (`api/openapi.{env}.json`)
- Self-hosted Scalar UI with embedded assets (`web/docs`)
- TrimSlash middleware for trailing slash redirects
- Configuration extensions: Version, Domain, Env() method

**Validation**: ✅ Scalar UI loads at `/docs`, all endpoints documented, "Try It" functionality works

**Architectural Additions**:
- Domain-Owned OpenAPI: Each domain owns its schemas and operations in `openapi.go`
- Route OpenAPI Integration: Routes carry optional `OpenAPI` field referencing domain specs
- Spec Generation Flow: Routes → Generate in memory → Compare with file → Write only if changed
- Embedded Assets: `go:embed` for air-gap compatible Scalar UI

---

**Milestone Success Criteria**:
- Create Ollama provider configuration via API
- Create gpt-4o agent configuration with provider reference
- Search providers and agents with filters and pagination
- Configuration validation using go-agents structures
- Graceful server shutdown on SIGTERM/SIGINT

### Milestone 2: Document Upload & Processing

**Objective**: Enable PDF upload with document-context integration for rendering.

**Deliverables**:
- Document upload API (`POST /api/documents`, multipart/form-data)
- Database schema: `documents` table with metadata
- Blob storage system with filesystem implementation (`.data/blobs/documents/`)
- document-context integration (PDF → page extraction → image rendering)
- Filesystem cache configuration (`.data/cache/images/`)
- Enhancement filter configuration API
- Document preview endpoint (`GET /api/documents/{id}/pages/{num}`)

**Design Decisions**:
- **Blob Storage**: Interface-based abstraction from start (filesystem in M2, Azure in M8)
- **Page Count Extraction**: On upload (blocking) - metadata complete immediately
- **Cache Strategy**: Simple filesystem, no eviction (leverage document-context built-in cache)
- **Enhancement Filters**: Leverage document-context configuration (brightness, contrast, saturation, rotation, background)

**Development Sessions**:

#### Session 2a: Blob Storage Infrastructure ✅

**Status**: Completed (2025-12-08)

**Implemented**:
- `internal/storage` - Storage System interface (Store, Retrieve, Delete, Validate)
- Filesystem implementation with configurable base path
- Configuration integration (`StorageConfig` with `STORAGE_BASE_PATH` env var)
- Directory initialization on startup via lifecycle OnStartup hook
- Path traversal protection in key validation
- Atomic file writes (temp file + rename)
- Comprehensive test coverage for critical paths

**Validation**: ✅ Store/retrieve/delete files via filesystem storage, path traversal blocked

**Architectural Decisions**:
- Interface in `internal/storage/` (not pkg/) to avoid import boundary issues with lifecycle
- `Validate` method returns `(bool, error)` - false/nil for not exists, false/err for permission issues
- Testing success measured by critical path coverage, not arbitrary percentages

#### Session 2b: Documents Domain System ✅

**Status**: Completed (2025-12-09)

**Implemented**:
- Database schema: `documents` table with indexes (omitted metadata JSONB per YAGNI)
- Migration `000004_documents` (up/down)
- Documents domain system (document.go, errors.go, filters.go, system.go, repository.go, handler.go, openapi.go, projection.go, scanner.go)
- Upload API with PDF page count extraction via pdfcpu
- Full CRUD endpoints with pagination and filtering
- Delete removes blob and database record; cleans up empty parent directories
- Storage-first atomicity with rollback on DB failure
- MaxUploadSize config (human-readable via docker/go-units)

**Validation**: ✅ Upload PDF with page_count extracted, update name, delete (blob + record + empty directory removed)

**Architectural Decisions**:
- Omit metadata JSONB (add when concrete use case emerges)
- pdfcpu integrated directly (not via document-context)
- Documents domain constructs storage keys (`documents/{uuid}/{filename}`)
- SortFields type with flexible JSON unmarshaling (string or array)
- OpenAPI Properties aligned with OpenAPI 3.1 (Properties ARE Schemas)

#### Session 2c: document-context Integration ✅

**Status**: Completed (2025-12-10)

**Implemented**:
- Images domain system (`internal/images/`) - first-class database entities
- Database schema: `images` table with full render options storage
- Page range expressions: `1`, `1-5`, `1,3,5`, `1-5,10,15-20`, `-3`, `5-`
- document-context integration for ImageMagick rendering
- API endpoints: render, list, get metadata, get binary, delete
- Deduplication with optional force re-render
- Storage system `Path()` method for document-context integration

**API Endpoints**:
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/images/{documentId}/render` | Render document pages |
| GET | `/api/images` | List images with optional filters |
| GET | `/api/images/{id}` | Get image metadata |
| GET | `/api/images/{id}/data` | Get raw image binary |
| DELETE | `/api/images/{id}` | Delete image |

**Validation**: ✅ All endpoints tested with curl, 52 unit tests passing

**Architectural Decisions**:
- Images as first-class resources (not nested under documents)
- Cross-domain dependency: images → documents (unidirectional)
- Optional DocumentID filter for flexible querying
- ImageMagick neutral defaults (brightness=100, contrast=0, saturation=100)

---

**Success Criteria**:
- Upload multi-page PDF, extract metadata (page count) ✅
- Render page as PNG with default settings (300 DPI) ✅
- Apply enhancement filters (brightness, contrast, saturation, rotation) ✅
- Serve rendered page image for preview ✅
- Deduplication: Same render options returns existing image without re-rendering ✅

### Milestone 3: Workflow Execution Infrastructure

**Objective**: Build infrastructure for executing code-defined workflows with full observability, enabling iterative experimentation on agentic workflow designs.

**Architecture Document**: `_context/milestones/m03-workflow-execution.md`

**Key Decisions**:

1. **Code-defined workflows** - Workflows are Go code registered by name. No workflow definition CRUD - that comes after we understand patterns better through iteration.

2. **Visibility via Observer** - Per-stage results + routing decisions captured via go-agents-orchestration's Observer interface. Events both persisted to database AND streamed via SSE.

3. **Infrastructure only** - No classify-docs workflow in M3. This milestone builds the execution engine; classify-docs implementation comes in Milestone 5.

4. **Sync + SSE streaming** - Two execution modes: sync (wait for completion) and streaming (real-time SSE progress with cancellation support). No background/polling model needed for iteration.

**Database Schema** (4 tables):
- `runs` - Execution records (id, workflow_name, status, params, result)
- `stages` - Per-node execution via Observer events
- `decisions` - Routing decisions (from_node, to_node, predicate_result, reason)
- `checkpoints` - State persistence for resume capability

**API Endpoints**:

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/workflows` | List registered workflows |
| POST | `/api/workflows/{name}/execute` | Execute workflow (sync) |
| POST | `/api/workflows/{name}/execute/stream` | Execute with SSE progress |
| GET | `/api/runs` | List runs with filters |
| GET | `/api/runs/{id}` | Get run details |
| GET | `/api/runs/{id}/stages` | Get execution stages |
| GET | `/api/runs/{id}/decisions` | Get routing decisions |
| POST | `/api/runs/{id}/cancel` | Cancel running workflow |
| POST | `/api/runs/{id}/resume` | Resume from checkpoint |

**Development Sessions**:

#### Session 3a: Workflow Infrastructure Foundation ✅

**Status**: Completed (2025-12-17)

**Implemented**:
- Database schema: runs, stages, decisions, checkpoints tables
- Core types: Run, Stage, Decision, WorkflowInfo with status constants
- Domain errors with HTTP status mapping
- Mapping infrastructure: projections, scanners, filters
- Global registry: Register, Get, List functions
- Systems struct placeholder for Session 3c
- Read-only repository: ListRuns, FindRun, GetStages, GetDecisions
- Query builder Build() method for unbounded SELECT queries

**Validation**: ✅ Migration runs, registry works, all tests passing

**Architectural Additions**:
- Repository Query Method Naming: List (PageResult), Find (single), Get (full slice)
- WorkflowFactory type for StateGraph and State creation
- Thread-safe workflow registry with sync.RWMutex

#### Session 3b: Observer and Checkpoint Store ✅

**Status**: Completed (2025-12-17)

**Implemented**:
- go-agents-orchestration v0.2.0 maintenance release (State public fields with JSON tags)
- PostgresObserver implementing `observability.Observer`
- PostgresCheckpointStore implementing `state.CheckpointStore`
- Stage/decision recording from Observer events
- Duration calculation via internal start time tracking

**Validation**: ✅ Interface compliance verified, all tests passing

#### Session 3c: Workflow Execution Engine ✅

**Status**: Completed (2025-12-18)

**Implemented**:
- go-agents-orchestration v0.3.0 release (NewGraphWithDeps, thread-safe registries, config Merge methods)
- Runtime struct (renamed from Systems) with constructor and getters
- Repository write methods: CreateRun, UpdateRunStarted, UpdateRunCompleted
- Updated WorkflowFactory signature: factory receives pre-configured graph
- System interface for workflows domain
- Executor implementation with three-phase lifecycle (Cold Start → Hot Start → Post-Commit)
- Active run tracking with context cancellation
- Resume capability from failed/cancelled runs

**Validation**: ✅ All tests passing, go vet clean

**Architectural Additions**:
- Runtime as a Pattern: naming convention for "runtime dependencies a system needs"
- Three-Phase Executor Lifecycle: Cold Start (create run), Hot Start (execute), Post-Commit (finalize)
- Hardcoded GraphConfig policy: executor owns checkpointing policy for database-backed workflows

#### Session 3d: API Endpoints ✅

**Status**: Completed (2025-12-19)

**Implemented**:
- Handler with routes following agents/handler.go pattern
- OpenAPI specification with all operations and schemas
- Execute endpoint (sync completion)
- Execute/stream endpoint (SSE progress with MultiObserver pattern)
- Cancel endpoint (abort running workflow)
- List runs, get run details, stages, decisions endpoints
- Resume endpoint
- Route Children pattern for hierarchical route groups
- `pkg/decode` package for map[string]any to struct conversion
- StreamingObserver and MultiObserver implementations
- Go file structure convention documented

**Validation**: ✅ All endpoints implemented, SSE streaming works, 34 new tests passing

#### Session 3e: Sample Workflows ✅

**Status**: Completed (2025-12-19)

**Implemented**:
- Agent System capability methods (Chat, ChatStream, Vision, VisionStream, Tools, Embed)
- System prompt override via `opts["system_prompt"]` (stored config as fallback)
- Handler delegation to System for capability execution
- Sample workflows package (`internal/workflows/samples/`)
- Summarize workflow (single-node, text summarization)
- Reasoning workflow (multi-node: analyze → reason → conclude)
- Query builder `isNil()` fix for nil pointer detection
- Nullable JSON scan fix for runs/stages tables
- DeleteRun endpoint for run cleanup

**Validation**: ✅ Both workflows execute with live LLM calls, stages recorded, all tests passing

---

**Success Criteria**:
- Execute workflow via API, receive results
- Real-time SSE streaming of execution progress
- Cancel running workflow via API
- Resume workflow from checkpoint
- Query execution history with stages and routing decisions

### ~~Milestone 4: Real-Time Monitoring & SSE~~ (Absorbed)

**Status**: Absorbed into Milestone 3 and Milestone 5

**Rationale**: During Milestone 3 review, we identified that M3's SSE streaming implementation fulfilled M4's backend requirements. The remaining items are either minor (heartbeat) or frontend-focused (client JS).

**Deliverables Completed in M3:**
- ✅ SSE streaming endpoint (`POST /api/workflows/{name}/execute/stream`)
- ✅ Event publishing (stage.start, stage.complete, decision, error, complete)
- ✅ Event persistence (stages + decisions tables)
- ✅ Execution history API (`GET /api/workflows/runs` with filters)
- ✅ Run details endpoint (`GET /api/workflows/runs/{id}`)

**Deferred to M5 (Workflow Lab Interface):**
- Client-side EventSource integration (vanilla JS)
- Heartbeat mechanism (if needed)
- Reconnect to running workflow (if needed)

---

### Milestone 4: classify-docs Workflow Integration

**Objective**: Implement document classification workflow with parallel processing and A/B testing capability.

**Architecture Document**: `_context/milestones/m04-classify-docs.md`

**Key Decisions**:

1. **Workflow Profiles** - Database-stored configurations for workflow stages (agents, prompts). Named "profiles" for clean API design (`/api/profiles`).

2. **Workflow Directory Separation** - Move workflow definitions from `internal/workflows/samples/` to top-level `workflows/` directory. Infrastructure stays in `internal/workflows/`.

3. **Parallel Detection** - Use go-agents-orchestration's `ProcessParallel` for concurrent page analysis with automatic worker pool sizing.

4. **Numeric Clarity Scoring** - 0.0-1.0 scale with threshold-based enhancement triggering (< 0.7).

**Database Schema** (2 tables):
- `profiles` - Named configurations per workflow (workflow_name, name, is_default)
- `profile_stages` - Per-stage configuration (profile_id, stage_name, agent_id, system_prompt, options)

**API Endpoints**:

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/profiles` | Create workflow profile |
| GET | `/api/profiles` | List profiles (filter by workflow_name) |
| GET | `/api/profiles/{id}` | Get profile with stages |
| PUT | `/api/profiles/{id}` | Update profile metadata |
| DELETE | `/api/profiles/{id}` | Delete profile and stages |
| POST | `/api/profiles/{id}/stages` | Add/update stage config |
| DELETE | `/api/profiles/{id}/stages/{stage}` | Remove stage config |

**Development Sessions**:

#### Session 4a: Profiles Infrastructure & Workflow Migration ✅

**Deliverables**:
- Database migration for profiles and profile_stages tables
- Profile domain: types, repository, handler, OpenAPI
- Move existing workflows to `workflows/` directory
- Update Runtime with profiles system access
- Profile resolution: explicit profile_id → DB profile, else → hardcoded default
- Stage AgentID override support (stage.AgentID takes precedence over params)

**Key Files**:
- `cmd/migrate/migrations/000007_profiles.up.sql`
- `internal/profiles/` (new domain)
- `internal/workflows/profile.go` (shared helpers)
- `workflows/init.go` (single import aggregation)
- `workflows/summarize/` (profile.go + summarize.go)
- `workflows/reasoning/` (profile.go + reasoning.go)

**Validation**: ✅ Profile CRUD, stage save, both profile scenarios (system_prompt only, agent_id configured)

#### Session 4b: classify-docs Types and Detection Stage ✅

**Deliverables**:
- Type definitions (PageImage, PageDetection, MarkingInfo, FilterSuggestion)
- Detection system prompt and JSON response parser
- Init node (load profile, document, render images)
- Detect node using ProcessParallel
- Secure token handling (token in request body, not persisted)

**Key Files**:
- `workflows/classify/errors.go`
- `workflows/classify/parse.go`
- `workflows/classify/profile.go`
- `workflows/classify/classify.go`

**Validation**: ✅ Workflow executes through detect stage, parallel execution verified, token security implemented

#### Session 4c: Enhancement, Classification, and Scoring ✅

**Deliverables**:
- Enhancement conditional node (re-render low-clarity pages)
- Classification node and prompt
- Scoring node with weighted factors
- Complete workflow graph assembly

**Key Files**:
- `workflows/classify/classify.go` (types, nodes, factory, helpers)
- `workflows/classify/profile.go` (system prompts, DefaultProfile)
- `workflows/classify/parse.go` (generic parser, classification/scoring parsers)
- `workflows/classify/errors.go` (new error types)

**Validation**: ✅ Full workflow execution with Azure GPT-5-mini, tested with challenging faded-marking document

#### Session 4d: Data Security and Seed Infrastructure

**Deliverables**:
- Add `Secrets` field to `state.State` in go-agents-orchestration (architectural fix for token leakage)
- Update agent-lab to use new Secrets API for token handling
- Create `cmd/seed` infrastructure for profile experimentation
- Profile seeding with transactional execution

**Phase 1: go-agents-orchestration**:
- Add `Secrets map[string]any` field with `json:"-"` tag (never persisted/observed)
- Add `SetSecret`, `GetSecret`, `DeleteSecret` methods following immutable pattern
- Update `Clone()` to also clone Secrets map
- Release new version

**Phase 2: agent-lab Integration**:
- Update `executor.go` to use `SetSecret("token", token)`
- Update `ExtractAgentParams` to use `GetSecret("token")`
- Update go-agents-orchestration dependency

**Phase 3: cmd/seed Infrastructure**:
- Create `cmd/seed/main.go` (CLI entry point with flags)
- Create `cmd/seed/seeder.go` (Seeder interface and registry)
- Create `cmd/seed/profiles.go` (Profile seeding using internal/profiles types)
- Create embedded seed data files for classify workflow profiles

**Key Files**:
- `go-agents-orchestration/pkg/state/state.go`
- `agent-lab/internal/workflows/executor.go`
- `agent-lab/internal/workflows/profile.go`
- `agent-lab/cmd/seed/` (new directory)

**Validation**: ✅ Token NOT in checkpoints, stages, runs, or SSE events; seed command creates profiles correctly

#### Session 4e: Performance and Accuracy Refinement ✅

**Status**: Completed (2026-01-07)

**Implemented**:
- WriteTimeout increased from 3m to 15m for long-running workflows
- Parallel PDF rendering with dynamic worker pool (`max(min(NumCPU, pageCount), 1)`)
- Dynamic worker detection using `DefaultParallelConfig()` (replaces hardcoded WorkerCap=4)
- Lifecycle context pattern for HTTP-initiated long-running processes
- Consolidated Execute + ExecuteStream into single streaming endpoint
- Terminology refinement: clarity (per-page) vs legibility (per-marking)
- Enhanced system prompts for better faded marking detection

**Performance Results** (27-page PDF):
- Total execution: 3m19s (timeout) → 1m40s (2x improvement)
- Init stage: 1m28s → 13.4s (6.6x improvement via parallel rendering)
- Detect stage: 1m13s → 28.6s (2.6x improvement via dynamic workers)

**Accuracy Validation**:
- ~96% marking detection accuracy on 27-page multi-document PDF
- Correct classification: SECRET//NOFORN
- Minor OCR variances acceptable (WNINTEL→WINITEL)

**Architectural Additions**:
- Lifecycle Context Pattern: Long-running processes use `runtime.Lifecycle().Context()` to survive HTTP disconnection while respecting server shutdown
- Parallel Worker Pool: Dynamic sizing based on `runtime.NumCPU()` and workload size

**Validation**: ✅ 27-page PDF completes in 1m40s, ~96% accuracy, all tests passing

---

**Success Criteria**:
- Workflow profiles CRUD via API
- Existing workflows moved to `workflows/` and functioning
- classify-docs workflow registered and executable
- Parallel page detection with ProcessParallel
- Conditional enhancement for low-clarity pages
- Classification with alternatives when ambiguous
- Confidence scoring with weighted factors (0.0-1.0)
- A/B testing capability via profile_id parameter
- Baseline accuracy: match 96.3% prototype

### Milestone 5: Workflow Lab Interface

**Objective**: Build web interface for workflow monitoring and iteration.

**Architecture Document**: `_context/milestones/m05-workflow-lab-interface.md`

**Prerequisite**: Maintenance Session mt08 (Context Optimization and Package Layering)

**Key Decisions**:

1. **Lit SPA Architecture** - Single HTML shell served by Go for all `/app/*` routes. Client-side router handles view mounting. Hard boundary: Go owns data/routing, Lit owns presentation entirely.

2. **Three-Tier Component Hierarchy** - Views (provide services via `@provide`) → Stateful Components (consume via `@consume`) → Pure Elements (props in, events out). Patterns validated in go-lit POC.

3. **Context-Based Services** - `@lit/context` for dependency injection. Each domain has consolidated `service.ts` exporting context, interface, and factory.

4. **Signal-Based Reactivity** - `@lit-labs/signals` with `SignalWatcher` mixin for fine-grained reactive updates.

5. **@layer CSS Architecture** - Cascade layers (reset → theme → layout → components) with CSS custom properties for design tokens. Dark/light theme via `prefers-color-scheme`.

**Technology Stack**:
- Build: Bun + Vite + TypeScript (CI only, not in container)
- Framework: Lit 3.x (`lit`, `@lit/context`, `@lit-labs/signals`)
- Components: `al-` prefix with external CSS via `?inline` imports
- Styling: @layer-based CSS with design tokens
- Charts: D3.js (lazy-loaded/tree-shaken)
- Router: Custom client-side router (History API)

**Existing Infrastructure** (from previous 05a-05c sessions):
- Vite build pipeline
- CSS layer structure (partial)
- Go `pkg/web` template infrastructure
- Module mounting pattern

This infrastructure will be adapted for Lit rather than discarded. Session numbering resets to acknowledge the architectural shift.

**Development Sessions**:

#### Session 5a: Lit Migration

**Objective**: Adapt existing web infrastructure for Lit

**Deliverables**:
- Add Lit dependencies (`lit`, `@lit/context`, `@lit-labs/signals`)
- Create client-side router from go-lit patterns
- Update `app.ts` entry point for Lit
- Convert Go routes to single catch-all shell pattern
- Create home view as baseline

**Validation**: Router mounts views, navigation works, Go serves shell correctly

---

#### Session 5b: Design System

**Objective**: Complete CSS layer architecture

**Deliverables**:
- Complete design tokens (spacing, typography, colors)
- Layout utilities (stack, cluster, constrain)
- Element base styles for Shadow DOM components
- App-shell scroll architecture (100dvh, flex layout)

**Validation**: Tokens apply correctly, dark/light themes work, scroll regions function

---

#### Session 5c: Service Infrastructure

**Objective**: Establish service patterns for domain data

**Deliverables**:
- API client utility (fetch wrapper with Result type)
- Service pattern template (context + interface + factory)
- Signal-based state management patterns
- SSE client utility for streaming

**Validation**: Services load data, signals trigger re-renders

---

#### Session 5d: Provider/Agent Config

**Objective**: CRUD UI patterns with Lit components

**Deliverables**:
- Provider list/edit views
- Agent list/edit views
- Form handling patterns (FormData extraction)
- CRUD operation patterns

**Validation**: CRUD operations work for providers and agents

---

#### Session 5e: Document Upload

**Objective**: Document management with storage integration

**Deliverables**:
- Document upload form (multipart)
- Document list view with metadata
- Page viewer component
- Image rendering controls

**Validation**: Upload documents, render pages with filters, view images

---

#### Session 5f: Profile Management

**Objective**: Nested resource management

**Deliverables**:
- Profile list/edit views
- Stage editor component
- Nested form patterns
- Profile cloning for A/B testing

**Validation**: Create profiles, add stages, modify configurations

---

#### Session 5g: Workflow Execution

**Objective**: Launch workflow executions

**Deliverables**:
- Workflow list view
- Execution form (profile selection, params)
- Execute action with navigation to monitoring

**Validation**: Select workflow, configure params, trigger execution

---

#### Session 5h: Run Monitoring

**Objective**: Real-time execution monitoring

**Deliverables**:
- Run detail view with SSE integration
- Stage timeline visualization
- Decision flow display
- Run actions (cancel, resume)

**Validation**: Execute workflow, watch real-time progress

---

#### Session 5i: Visualization

**Objective**: Confidence score visualization

**Deliverables**:
- Confidence chart component (evaluate D3 vs native)
- Results summary display
- Per-page confidence breakdown

**Validation**: Execute classify-docs, view confidence charts

---

#### Session 5j: Comparison

**Objective**: Side-by-side run comparison

**Deliverables**:
- Run comparison component
- Diff highlighting
- Re-execute with modifications
- Complete iteration cycle

**Validation**: Compare runs, identify differences, re-execute

---

**Success Criteria**:
- View document pages with enhancement filter controls
- Monitor execution in real-time with progress indicators
- Visualize confidence score evolution across pages
- Compare multiple runs side-by-side
- Adjust agent options and filter overrides, re-execute
- Complete iteration cycle in < 5 minutes

### Milestone 6: Operational Features

**Objective**: Enable production-ready bulk processing and operations.

**Deliverables**:
- Bulk document processing (`POST /api/workflows/{id}/execute/bulk`)
- Execution history filtering and search (status, date range, workflow)
- RBAC foundations (ownership model, defer authentication to Phase 8)
- Audit logging (execution events, user actions)
- Result export API (JSON, JSONL, CSV formats)
- Webhook support for completion notifications (Phase 6+)

**Success Criteria**:
- Submit batch of documents for classification
- Monitor bulk execution progress
- Filter execution history by status and workflow
- Export results in multiple formats
- Audit log captures all significant actions

### Milestone 7: User Workflow Interface

**Objective**: Build user-facing interfaces that make workflow consumption accessible to non-technical users, abstracting away orchestration complexity (nodes, edges, profiles, stategraphs).

**Analogy**: Claude Code : Claude Cowork :: Workflow Lab (M5) : User Workflow Interface (M7)

**Architecture Document**: `_context/milestones/m07-user-workflow-interface.md` (created when milestone begins)

**Key Decisions**:

1. **Separate routes, same web client** - RBAC controls access, not separate apps
   - User workflows: `/app/workflows/{workflow-name}` (e.g., `/app/workflows/classify-docs`)
   - Lab interface: `/app/lab/workflows/{workflow-name}` (e.g., `/app/lab/workflows/classify-docs`)

2. **Reusable components** - Maximize component reuse between workflows
   - Queue/upload components
   - Processing status indicators
   - Result summary patterns
   - Validation panels

3. **One workflow run per document** - Clean execution model
   - Multiple document uploads create queued workflow runs
   - Parallel workflow execution (across documents) deferred for exploration

4. **Human-in-the-loop actions**:
   - Validate (confirm AI decision)
   - Override (correct with reason)
   - Re-run (same or modified parameters)
   - Annotate (add notes for audit)
   - Export (single or batch)

**Route Structure**:
```
/app/workflows/classify-docs/
├── /                    # Upload + queue view
├── /queue               # Processing queue status
├── /results/{run-id}    # Document result detail
└── /export              # Batch export options
```

**Development Sessions** (proposed):

| Session | Focus | Deliverables |
|---------|-------|--------------|
| 7a | Route Infrastructure | Consumer route structure, queue data model |
| 7b | Upload & Queue UI | Document uploader, queue management, status display |
| 7c | Result Display | Result summary, document detail view |
| 7d | Human Validation | Validation panel, override workflow, annotations |
| 7e | Workflow Visualization | classify-docs specific result views, page viewer |
| 7f | Batch Operations | Multi-select, batch export, batch actions |

**Success Criteria**:
- Non-technical user can upload documents and view results without understanding workflow internals
- Human-in-the-loop validation captures corrections with audit trail
- Queue provides clear visibility into processing status
- Results display is task-focused (classification outcome) not implementation-focused (nodes, stages)
- Batch export enables integration with downstream systems

---

### Milestone 8: Production Deployment

**Objective**: Deploy to Azure with production integrations.

**Deliverables**:
- Azure blob storage system (replace filesystem implementation)
- Azure AI Foundry integration with Entra ID authentication
- Managed identity for service authentication
- Kubernetes manifests (deployment, service, ingress, configmap, secret)
- Production configuration (environment-specific overrides)
- Application Insights integration (observability)
- Air-gap deployment validation

**Success Criteria**:
- Deploy to Azure Kubernetes Service (AKS)
- Authenticate to Azure AI Foundry with managed identity
- Store documents in Azure Blob Storage
- Production workload handles concurrent executions
- Monitoring via Application Insights
- Air-gap deployment successfully builds and runs

## Current Status

**Phase**: Milestone 4 Complete

**Completed**:
- Session 01: Foundation architecture design (ARCHITECTURE.md)
- Session 02: Project vision and roadmap establishment (PROJECT.md)
- Library ecosystem analysis (go-agents, go-agents-orchestration, document-context)
- classify-docs prototype requirements mapping
- **Session 01a: Foundation Infrastructure** ✅
  - Core service infrastructure (config, logger, routes, middleware, server)
  - Cold/Hot start lifecycle with graceful shutdown
  - 100% test coverage with comprehensive black-box testing
  - Full documentation (godoc, README, ARCHITECTURE, session summary)
- **Session 01b: Database & Query Infrastructure** ✅
  - Lifecycle coordinator pattern for startup/shutdown orchestration
  - Database system with connection pool and lifecycle integration
  - Migration CLI and infrastructure
  - Query builder (ProjectionMap, Builder) and pagination utilities
  - Readiness endpoint (`/readyz`)
  - 100% test coverage for new packages
- **Session 01c: Runtime/Domain System Separation + Providers System** ✅
  - Runtime/Domain system separation pattern
  - Providers domain system with CRUD + Search endpoints
  - Domain errors with HTTP status code mapping
  - go-agents provider config validation
  - Logger simplified to functional infrastructure
- **Session 01d: Domain Infrastructure Patterns** ✅
  - Repository helpers (pkg/repository): WithTx, QueryOne, QueryMany, ExecExpectOne, MapError
  - Handler utilities (pkg/handlers): RespondJSON, RespondError
  - Query enhancements: SortField, ParseSortFields, OrderByFields
  - Domain filter pattern: Filters struct with FiltersFromQuery and Apply
  - Handler struct pattern: Handler with Routes() method
- **Session 01e: Agents System** ✅
  - Agents domain system with CRUD + Search + Execution endpoints
  - Agent execution: Chat, ChatStream, Vision, VisionStream, Tools, Embed
  - SSE streaming for Chat and Vision
  - VisionForm pattern for multipart image uploads
  - Token injection for Azure authentication
- **Session 01f: OpenAPI Specification & Scalar UI Integration** ✅
  - OpenAPI 3.1 spec infrastructure (pkg/openapi)
  - Domain-owned schemas and operations
  - Integrated spec generation at server startup
  - Self-hosted Scalar UI at `/docs` endpoint
  - TrimSlash middleware for trailing slash redirects

**Recently Completed**:
- Milestone 3: Workflow Execution Infrastructure ✅
  - Session 3a: Workflow Infrastructure Foundation ✅
  - Session 3b: Observer and Checkpoint Store ✅
  - Session 3c: Workflow Execution Engine ✅
  - Session 3d: API Endpoints ✅
  - Session 3e: Sample Workflows ✅
- Milestone 2: Document Upload & Processing ✅
  - Session 02a: Blob Storage Infrastructure ✅
  - Session 02b: Documents Domain System ✅
  - Session 02c: document-context Integration ✅
- Maintenance Session m01: document-context Format Support ✅
  - Migrated format support from agent-lab shims to document-context v0.1.1
  - Added ParseImageFormat, Open, IsSupported, SupportedFormats to document-context
  - Improved agent config validation with Default + Merge pattern
- Maintenance Session m02: go-agents-orchestration v0.2.0 ✅
  - State struct public fields with JSON tags for checkpoint serialization
  - Edge.Name field for predicate identification
  - Enhanced observer events with state snapshots
- Maintenance Session m03: Native MultiObserver Support ✅
  - Added MultiObserver to go-agents-orchestration v0.3.1
  - Migrated shim from agent-lab to library
  - Enables broadcasting events to multiple observers
- Maintenance Session mt04: Context Architecture Optimization ✅
  - Restructured context from monolithic files to Claude Code native skills/rules
  - Created 12 on-demand skills for domain-specific patterns
  - Created 3 always-loaded rules (243 lines vs ~4,900 previously)
  - Archived ARCHITECTURE.md, CLAUDE.md, web-service-architecture.md, service-design.md
- Maintenance Session mt05: Web Architecture Refactor ✅
  - Extracted `pkg/routes` types (Route, Group, System interface)
  - Created `pkg/web` infrastructure (TemplateSet, Router, DistServer, PublicFileRoutes)
  - Established isolated web clients pattern: `web/app/`, `web/scalar/`
  - Each client is fully self-contained with Mount() function
  - Per-client Vite configs (`client.config.ts`) merged by root `vite.config.ts`
  - URL routing: `/app/*` for main app, `/scalar/*` for OpenAPI UI
  - Added Makefile for development workflow
  - Pre-parse templates at startup for zero per-request overhead
- Maintenance Session mt06: Mountable Modules ✅
  - Refactored server architecture into isolated, mountable modules
  - Created `pkg/module` with Module and Router types for modular HTTP routing
  - Moved `internal/middleware`, `internal/lifecycle`, `internal/database`, `internal/storage` to `pkg/`
  - Created `internal/api` module encapsulating Runtime, Domain, and route registration
  - Web clients use `NewModule(basePath)` pattern with `<base>` tag for relative URLs
  - AddSlash middleware for web clients, TrimSlash for API
  - Config pattern: public packages define Env struct, app passes key mappings

**In Progress**:
- **Maintenance Session mt08: Context Optimization and Package Layering** (prerequisite for M5)
  - Context optimization (remove rules/, consolidate CLAUDE.md, trigger-optimized skill descriptions)
  - Package layering fix (create pkg/config, eliminate pkg/ → internal/ dependency)
  - Web development skill rewrite (Lit architecture patterns)
  - Implementation guide: `_context/mt08-context-and-package-layering.md`
- **Milestone 5: Workflow Lab Interface** (architecture reset)
  - Previous sessions 5a-5c established foundation infrastructure (preserved, will be adapted)
  - Architecture document rewritten for Lit SPA approach
  - Session numbering reset; new sessions 5a-5j defined
  - Blocked by: mt08 completion

**Recently Completed**:
- **Maintenance Session mt07: Module Polish** ✅
  - Path normalization at router level (replaced redirect-based slash middleware)
  - Fixed 404 page empty bundle name with PageDef pattern
  - Extracted shared `pkg/runtime.Infrastructure`
  - Added Handler() factory to domain systems for simplified route registration
  - Comprehensive infrastructure review (tests, .claude, comments, docs)
- **Maintenance Session mt06: Mountable Modules** ✅
  - Refactored server architecture into isolated, mountable modules
  - Created `pkg/module` with Module and Router types
  - Moved infrastructure packages to `pkg/` (middleware, lifecycle, database, storage)
  - Created `internal/api` module encapsulating Runtime, Domain, and route registration
  - Web clients use `NewModule(basePath)` pattern with `<base>` tag
- **Milestone 4: classify-docs Workflow Integration** ✅
  - Session 4a: Profiles Infrastructure & Workflow Migration ✅
  - Session 4b: classify-docs Types and Detection Stage ✅
  - Session 4c: Enhancement, Classification, and Scoring ✅
  - Session 4d: Data Security and Seed Infrastructure ✅
  - Session 4e: Performance and Accuracy Refinement ✅
  - Milestone Review ✅

**Next Steps**:
1. Execute Maintenance Session mt08 (Context Optimization and Package Layering)
2. Continue Milestone 5: Session 5a (Lit Migration)

## Future Phases (Beyond Milestone 8)

### Multi-Workflow Support

Expand beyond classification to additional workflow types:
- **Data Extraction**: Extract structured data from documents
- **Content Generation**: Generate reports, summaries, documentation
- **Analysis**: Analyze documents for compliance, sentiment, topics
- **Translation**: Multi-language document translation

### Visual Workflow Designer

Drag-and-drop interface for workflow composition:
- Node-based workflow design (agents, routing, conditions)
- Visual state flow representation
- Real-time validation of workflow configuration
- Template library for common patterns

### Advanced Orchestration

Leverage complete go-agents-orchestration capabilities:
- **Parallel Execution**: Concurrent page analysis with worker pools
- **Conditional Routing**: Decision-based workflow branches
- **State Graphs**: Complex multi-stage workflows with checkpointing
- **Workflow Composition**: Reusable sub-workflows and modules

### External Integrations

Connect agent-lab to external systems:
- **Data Sources**: Azure Blob Storage, S3, databases, SharePoint
- **Output Destinations**: Webhooks, queues, databases, file systems
- **Identity Providers**: Entra ID, Active Directory, SAML, OAuth
- **Monitoring**: Prometheus, Grafana, Application Insights

### A/B Testing Framework

Compare workflow variants for optimization:
- Configure multiple workflow versions
- Split traffic across variants
- Compare accuracy and confidence metrics
- Promote winning variant to production

### Workflow Versioning

Manage workflow evolution:
- Version control for workflow configurations
- Rollback to previous versions
- Deployment history tracking
- Impact analysis before promotion

## Contributing

This project is currently in early development. Contributions welcome as the project matures.

**Development Workflow**: Session workflow documented in the development-methodology skill.

**Architecture Patterns**: Domain-specific patterns in `.claude/skills/` (loaded on-demand).

## License

TBD (To be determined during open-source preparation)
