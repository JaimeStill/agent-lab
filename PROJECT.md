# agent-lab Project

**Status**: Active Development - Planning Phase

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
5. **Standards-Forward**: Web Components, TC39 Signals, SSE, Fetch API
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
- **Components**: Web Components for encapsulated UI elements
- **State Management**: TC39 Signals for reactive state
- **Real-Time**: Server-Sent Events (SSE) for execution monitoring
- **HTTP**: Fetch API for REST interactions
- **Visualization**: D3.js (embedded) for confidence score graphs
- **Architecture**: See [web/README.md](./web/README.md) for full details

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

See [ARCHITECTURE.md](./ARCHITECTURE.md) for complete technical specifications and [_context/web-service-architecture.md](./_context/web-service-architecture.md) for architectural philosophy.

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

See [CLAUDE.md](./CLAUDE.md) for detailed development session workflow.

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

#### Session 3b: Observer and Checkpoint Store

**Objective**: Implement go-agents-orchestration interfaces for persistence.

**Deliverables**:
- PostgresObserver implementing `observability.Observer`
- PostgresCheckpointStore implementing `state.CheckpointStore`
- Registration with go-agents-orchestration registries
- Stage/decision recording from Observer events

**Validation**: Observer captures events to database, checkpoints save/load correctly.

#### Session 3c: Workflow Execution Engine

**Objective**: Connect components to execute workflows.

**Deliverables**:
- Executor: combines registry + observer + checkpoint store
- Dependencies struct for domain access (agents, documents, images)
- Run lifecycle: pending → running → completed/failed/cancelled
- Context cancellation propagation for workflow abort
- SSE event channel for streaming progress

**Validation**: Execute test workflow, verify stages/decisions recorded, cancellation works.

#### Session 3d: API Endpoints

**Objective**: HTTP interface for execution and inspection.

**Deliverables**:
- Handler with routes following agents/handler.go pattern
- OpenAPI specification
- Execute endpoint (sync completion)
- Execute/stream endpoint (SSE progress)
- Cancel endpoint (abort running workflow)
- List runs, get run details, stages, decisions endpoints
- Resume endpoint

**Validation**: Full API workflow via curl/tests, SSE streaming works.

#### Session 3e: Sample Workflow and Integration Tests

**Objective**: Validate infrastructure with working example.

**Deliverables**:
- Sample "echo" workflow (simple chain: input → transform → output)
- Sample workflow using conditional routing
- Integration tests covering execution, checkpoints, resume
- ARCHITECTURE.md updates

**Validation**: Integration tests pass, patterns documented.

---

**Success Criteria**:
- Execute workflow via API, receive results
- Real-time SSE streaming of execution progress
- Cancel running workflow via API
- Resume workflow from checkpoint
- Query execution history with stages and routing decisions

### Milestone 4: Real-Time Monitoring & SSE

**Objective**: Enable real-time observability of workflow execution.

**Deliverables**:
- SSE streaming endpoint (`GET /api/runs/{id}/stream`)
- Event publishing from workflow execution (step_started, step_completed, confidence_scored)
- Selective event persistence (`execution_events` table)
- Execution history API (`GET /api/runs`, filtering and pagination)
- Run details endpoint (`GET /api/runs/{id}` with trace data)
- Client-side EventSource integration (vanilla JS)

**Success Criteria**:
- Client establishes SSE connection, receives real-time events
- Heartbeat keeps connection alive (30-second interval)
- Stream closes on execution completion
- Persisted events queryable via history API
- Execution trace shows step-by-step progression

### Milestone 5: classify-docs Workflow Integration

**Objective**: Implement document classification workflow using go-agents-orchestration.

**Deliverables**:
- System prompt management (storage in workflows table)
- Sequential workflow implementation using `ProcessChain`
- Per-page classification with vision API
- Confidence scoring algorithm (0.0-1.0 scale with semantic meaning)
- Marking detection and spatial combination logic
- Classification result aggregation across pages
- Validation against 27-document test set

**Success Criteria**:
- Execute classification workflow on test document
- Achieve baseline 96.3% accuracy (matching prototype)
- Confidence scores reflect tangible factors (marking clarity, consistency, spatial distribution)
- Execution trace shows per-page analysis progression
- Results include detected markings with positions

### Milestone 6: Workflow Lab Interface

**Objective**: Build web interface for workflow monitoring and iteration.

**Deliverables**:
- OpenAPI Scalar interface (established in Milestone 1, enhanced)
- Document preview web component (view rendered pages)
- Execution monitoring interface (SSE client with progress display)
- Confidence score visualization (D3.js line/bar charts)
- Results display with detected markings overlay
- Side-by-side comparison of execution runs
- Workflow parameter adjustment UI

**Success Criteria**:
- View document pages with enhancement filter controls
- Monitor execution in real-time with progress indicators
- Visualize confidence score evolution across pages
- Compare multiple runs side-by-side
- Adjust agent options and filter overrides, re-execute
- Complete iteration cycle in < 5 minutes

### Milestone 7: Operational Features

**Objective**: Enable production-ready bulk processing and operations.

**Deliverables**:
- Bulk document processing (`POST /api/workflows/{id}/execute/bulk`)
- Execution history filtering and search (status, date range, workflow)
- RBAC foundations (ownership model, defer authentication to Phase 8)
- Audit logging (execution events, user actions)
- Result export API (JSON, JSONL, CSV formats)
- Webhook support for completion notifications (Phase 7+)

**Success Criteria**:
- Submit batch of documents for classification
- Monitor bulk execution progress
- Filter execution history by status and workflow
- Export results in multiple formats
- Audit log captures all significant actions

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

**Phase**: Milestone 3 - Workflow Execution Infrastructure

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

**In Progress**:
- Milestone 3: Workflow Execution Infrastructure
  - Session 3a: Workflow Infrastructure Foundation ✅
  - Session 3b: Observer and Checkpoint Store (pending)

**Recently Completed**:
- Milestone 2: Document Upload & Processing ✅
  - Session 02a: Blob Storage Infrastructure ✅
  - Session 02b: Documents Domain System ✅
  - Session 02c: document-context Integration ✅
- Maintenance Session m01: document-context Format Support ✅
  - Migrated format support from agent-lab shims to document-context v0.1.1
  - Added ParseImageFormat, Open, IsSupported, SupportedFormats to document-context
  - Improved agent config validation with Default + Merge pattern

**Next Steps**:
- Begin Session 3b: Observer and Checkpoint Store

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

**Development Workflow**: See [CLAUDE.md](./CLAUDE.md) for development session workflow and conventions.

**Architecture Guidelines**: See [ARCHITECTURE.md](./ARCHITECTURE.md) for technical specifications and patterns.

## License

TBD (To be determined during open-source preparation)
