# Session mt06: Mountable Modules

**Date**: 2026-01-16

**Status**: Completed

## Overview

Refactored the server architecture into isolated, mountable "Modules" where each unit (API, App, Scalar) is self-contained with its own middleware pipeline, internal router, and base path. This architectural change improves separation of concerns and enables each module to be tested independently.

## What Was Implemented

### New Packages

**`pkg/module`** - Modular HTTP routing infrastructure:
- `Module` type - Self-contained HTTP handler with middleware chain
- `Router` type - Top-level router that mounts modules and handles native routes
- Prefix validation (single-level sub-paths only)
- Path stripping for internal routing

**`pkg/middleware`** (moved from `internal/middleware`):
- `AddSlash()` - Redirects paths without trailing slash (enables relative URLs)
- `TrimSlash()` - Removes trailing slashes from requests
- `CORS()` - Cross-origin resource sharing middleware
- `Logger()` - Request logging middleware
- `CORSConfig`, `CORSEnv` - Configuration with env override support

**`pkg/database`** (moved from `internal/database`):
- `System` interface for database lifecycle management
- Configuration with `Env` struct for environment variable mapping

**`pkg/storage`** (moved from `internal/storage`):
- Storage system with config moved to same package

**`pkg/lifecycle`** (moved from `internal/lifecycle`):
- Lifecycle coordination for startup/shutdown

**`internal/api`** - API module encapsulation:
- `Runtime` - API-specific infrastructure (Logger, DB, Storage, Lifecycle, Pagination)
- `Domain` - Domain systems (Providers, Agents, Documents, Images, Profiles, Workflows)
- Route registration with OpenAPI spec building

### Refactored Components

**Web Clients**:
- `web/app/app.go` - `NewModule(basePath)` pattern replacing `NewHandler()`
- `web/scalar/scalar.go` - `NewModule(basePath)` pattern
- Templates use `<base href="{{ .BasePath }}/">` for relative URL resolution

**Server**:
- `cmd/server/modules.go` - Module creation and mounting
- `cmd/server/server.go` - Simplified to coordinate modules
- Native routes (`/healthz`, `/readyz`) registered on module Router
- Deleted: `cmd/server/routes.go`, `cmd/server/middleware.go`, `cmd/server/domain.go`

**Configuration**:
- `APIConfig` consolidates API settings (CORS, Pagination, OpenAPI, BasePath)
- Public packages define config structs with `Env` structs for env var mapping
- Application passes env key mappings to `Finalize(env)`

### Routes System Enhancement

**`pkg/routes`**:
- `Group.AddToSpec()` - Populates OpenAPI spec from route groups
- `Register()` - Registers routes on ServeMux with spec integration
- Tags inherited by child operations

### OpenAPI System Enhancement

**`pkg/openapi`**:
- `NewSpec(title, version)` - Spec factory function
- `SetDescription()`, `AddServer()` - Spec configuration
- `ServeSpec()` - Handler factory for serving spec JSON

## Key Architectural Decisions

1. **Module Ownership**: Each module is isolated with own middleware pipeline
2. **AddSlash + `<base>` Pattern**: Web clients embrace trailing slashes, API normalizes them away
3. **Env Struct Pattern**: Public packages define config + env struct, app passes key mappings
4. **API Module Encapsulation**: API owns its runtime (Pagination) and domain (all handlers)
5. **Shared Infrastructure Flows Down**: Server provides Logger, DB, Storage, Lifecycle to modules

## Files Created

| File | Purpose |
|------|---------|
| `pkg/module/module.go` | Module type with middleware chain |
| `pkg/module/router.go` | Top-level router for module mounting |
| `pkg/middleware/config.go` | CORSConfig with Env struct |
| `pkg/middleware/slash.go` | AddSlash middleware |
| `pkg/openapi/spec.go` | Spec factory and helpers |
| `pkg/database/config.go` | Database config with Env struct |
| `pkg/storage/config.go` | Storage config with Env struct |
| `internal/api/api.go` | API module creation |
| `internal/api/runtime.go` | API-specific infrastructure |
| `internal/api/domain.go` | Domain systems |
| `internal/api/routes.go` | Route registration |
| `internal/config/api.go` | API configuration section |
| `cmd/server/modules.go` | Module initialization and mounting |

## Files Deleted

- `cmd/server/routes.go`
- `cmd/server/middleware.go`
- `cmd/server/domain.go`
- `cmd/server/openapi.go`
- `internal/config/cors.go`
- `internal/config/database.go`
- `internal/config/storage.go`
- `internal/middleware/` (moved to `pkg/middleware/`)
- `internal/lifecycle/` (moved to `pkg/lifecycle/`)
- `internal/database/` (moved to `pkg/database/`)
- `internal/storage/` (moved to `pkg/storage/`)
- `internal/routes/` (replaced by `pkg/routes/` and `internal/api/`)
- `api/` directory (OpenAPI spec now in `web/scalar/`)

## Tests Added

| Package | Test File |
|---------|-----------|
| `pkg/module` | `module_test.go`, `router_test.go` |
| `pkg/middleware` | `middleware_test.go`, `slash_test.go`, `cors_test.go`, `logger_test.go`, `config_test.go` |
| `pkg/routes` | `group_test.go` |
| `pkg/database` | `config_test.go`, `errors_test.go` |

## Documentation Updates

- Updated `.claude/CLAUDE.md` with go-http file patterns
- Updated `.claude/skills/go-http/SKILL.md` with Module pattern
- Updated `.claude/skills/web-development/SKILL.md` with NewModule pattern
- Updated `.claude/skills/openapi/SKILL.md` with handler prefix and schema registration
- Updated `README.md` with new project structure

## Verification

All 24 test packages pass:
```
ok      github.com/JaimeStill/agent-lab/tests/pkg_module
ok      github.com/JaimeStill/agent-lab/tests/pkg_middleware
ok      github.com/JaimeStill/agent-lab/tests/pkg_routes
ok      github.com/JaimeStill/agent-lab/tests/pkg_database
... (all others passing)
```

Server routes verified:
- `GET /healthz` - 200 OK
- `GET /readyz` - 200 OK
- `GET /api/providers` - JSON response
- `GET /app` → redirects to `/app/`
- `GET /scalar` → redirects to `/scalar/`
