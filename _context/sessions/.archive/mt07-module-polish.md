# mt07 - Module Polish

## Overview

Maintenance session to fix routing issues and simplify module architecture.

**Items:**
1. Path normalization at router level (replace redirect-based slash middleware)
2. Fix 404 page empty bundle name
3. Extract shared infrastructure to `pkg/runtime`
4. Domain systems own their routes (self-contained HTTP surface)

---

## Phase 1: Path Normalization at Router Level

### Problem

Redirect-based slash middleware (AddSlash/TrimSlash) doesn't work correctly inside modules because:
1. Module.Serve() strips the prefix before middleware runs
2. Middleware generates redirects using the internal path (e.g., `/components/`)
3. Browser receives redirect to internal path, losing the module prefix (e.g., `/app`)

Additionally, Go's ServeMux has automatic redirect behavior for trailing slashes that conflicts with prefix stripping.

### Solution

Replace redirect-based middleware with **path normalization** at the router level. Normalize paths by stripping trailing slashes in-place (no redirect) before routing to modules.

Update `pkg/module/router.go`:

```go
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    path := normalizePath(req)
    prefix := extractPrefix(path)

    if m, ok := r.modules[prefix]; ok {
        m.Serve(w, req)
        return
    }

    r.native.ServeHTTP(w, req)
}

func normalizePath(req *http.Request) string {
    path := req.URL.Path
    if len(path) > 1 && strings.HasSuffix(path, "/") {
        path = strings.TrimSuffix(path, "/")
        req.URL.Path = path
    }
    return path
}
```

### Files to Delete

- `pkg/middleware/slash.go`
- `tests/pkg_middleware/slash_test.go`

### Files to Update

Remove AddSlash/TrimSlash usage from `cmd/server/modules.go`:

```go
appModule, err := app.NewModule("/app")
if err != nil {
    return nil, err
}
appModule.Use(middleware.Logger(runtime.Logger))

scalarModule := scalar.NewModule("/scalar")
```

Remove TrimSlash from `internal/api/api.go` if present.

Update `web/app/pages.go` to use routes without trailing slashes:

```go
var pages = []web.PageDef{
    {Route: "/{$}", Template: "home.html", Title: "Home", Bundle: "app"},
    {Route: "/components", Template: "components.html", Title: "Components", Bundle: "app"},
}
```

### Benefits

- No redirects = no prefix-stripping path mismatch issues
- No per-module middleware configuration for slashes
- Consistent behavior across all modules
- Less code overall

### Validation

```bash
curl -I http://localhost:8080/app
# Should return 200 OK

curl -I http://localhost:8080/app/
# Should return 200 OK (normalized to /app)

curl -I http://localhost:8080/app/components
# Should return 200 OK

curl -I http://localhost:8080/app/components/
# Should return 200 OK (normalized to /app/components)

curl -I http://localhost:8080/api/providers
# Should return 200 OK

curl -I http://localhost:8080/api/providers/
# Should return 200 OK (normalized to /api/providers)
```

---

## Phase 2: Fix 404 Page Bundle

### Problem

The 404 error page requests `/app/dist/.css` and `/app/dist/.js` - empty bundle name.

`ErrorHandler` in `pkg/web/pages.go` creates `PageData` without `Bundle`:

```go
data := PageData{Title: title, BasePath: ts.basePath}
// Missing: Bundle
```

### Solution

Refactor `ErrorHandler` to take a `PageDef` instead of individual parameters, consistent with how `PageHandler` works.

Update `pkg/web/pages.go`:

```go
func (ts *TemplateSet) ErrorHandler(layout string, page PageDef, status int) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(status)
        data := PageData{
            Title:    page.Title,
            Bundle:   page.Bundle,
            BasePath: ts.basePath,
        }
        if err := ts.Render(w, layout, page.Template, data); err != nil {
            http.Error(w, http.StatusText(status), status)
        }
    }
}
```

Update `web/app/app.go` - add `Bundle` to error pages and update usage:

```go
var errorPages = []web.PageDef{
    {Template: "404.html", Title: "Not Found", Bundle: "app"},
}

// In buildRouter:
r.SetFallback(ts.ErrorHandler("app.html", errorPages[0], http.StatusNotFound))
```

### Validation

Navigate to `http://localhost:8080/app/nonexistent` - should render styled 404 page with proper CSS/JS loading.

---

## Phase 3: Extract Shared Infrastructure

### Problem

API module initialization unpacks and repacks the same fields:

```go
// cmd/server/modules.go - unpacking 4 fields into separate args
api.NewModule(cfg, runtime.Logger, runtime.Database, runtime.Storage, runtime.Lifecycle)

// internal/api/api.go - receives 5 params
func NewModule(cfg, logger, db, store, lc)

// internal/api/runtime.go - repacks into struct
&Runtime{Logger: logger, Database: db, Storage: store, Lifecycle: lc, Pagination: ...}
```

### Solution

Move server runtime to `pkg/runtime`. The server's runtime IS the infrastructure - no wrapper needed.

#### Step 1: Create `pkg/runtime/infrastructure.go` (new file)

```go
package runtime

import (
    "fmt"
    "log/slog"

    "github.com/JaimeStill/agent-lab/internal/config"
    "github.com/JaimeStill/agent-lab/pkg/database"
    "github.com/JaimeStill/agent-lab/pkg/lifecycle"
    "github.com/JaimeStill/agent-lab/pkg/storage"
)

type Infrastructure struct {
    Lifecycle *lifecycle.Coordinator
    Logger    *slog.Logger
    Database  database.System
    Storage   storage.System
}

func New(cfg *config.Config) (*Infrastructure, error) {
    lc := lifecycle.New()
    logger := newLogger(&cfg.Logging)

    db, err := database.New(&cfg.Database, logger)
    if err != nil {
        return nil, fmt.Errorf("database init failed: %w", err)
    }

    store, err := storage.New(&cfg.Storage, logger)
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

#### Step 2: Create `pkg/runtime/logging.go` (new file)

Move logger creation from `cmd/server/logging.go`:

```go
package runtime

import (
    "log/slog"
    "os"

    "github.com/JaimeStill/agent-lab/internal/config"
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

#### Step 3: Delete server runtime files

- **Delete** `cmd/server/runtime.go`
- **Delete** `cmd/server/logging.go`

#### Step 4: Update `cmd/server/server.go`

Use `*runtime.Infrastructure` directly:

```go
package main

import (
    "time"

    "github.com/JaimeStill/agent-lab/internal/config"
    "github.com/JaimeStill/agent-lab/pkg/runtime"
    _ "github.com/JaimeStill/agent-lab/workflows"
)

type Server struct {
    infra   *runtime.Infrastructure
    modules *Modules
    http    *httpServer
}

func NewServer(cfg *config.Config) (*Server, error) {
    infra, err := runtime.New(cfg)
    if err != nil {
        return nil, err
    }

    modules, err := NewModules(infra, cfg)
    if err != nil {
        return nil, err
    }

    router := buildRouter(infra)
    modules.Mount(router)

    infra.Logger.Info(
        "server initialized",
        "addr", cfg.Server.Addr(),
        "version", cfg.Version,
    )

    return &Server{
        infra:   infra,
        modules: modules,
        http:    newHTTPServer(&cfg.Server, router, infra.Logger),
    }, nil
}

func (s *Server) Start() error {
    s.infra.Logger.Info("starting service")

    if err := s.infra.Start(); err != nil {
        return err
    }

    if err := s.http.Start(s.infra.Lifecycle); err != nil {
        return err
    }

    go func() {
        s.infra.Lifecycle.WaitForStartup()
        s.infra.Logger.Info("all subsystems ready")
    }()

    return nil
}

func (s *Server) Shutdown(timeout time.Duration) error {
    s.infra.Logger.Info("initiating shutdown")
    return s.infra.Lifecycle.Shutdown(timeout)
}
```

#### Step 5: Update `cmd/server/modules.go`

Change parameter from `*Runtime` to `*runtime.Infrastructure`:

```go
func NewModules(infra *runtime.Infrastructure, cfg *config.Config) (*Modules, error) {
    apiModule, err := api.NewModule(cfg, infra)
    if err != nil {
        return nil, err
    }

    appModule, err := app.NewModule("/app")
    if err != nil {
        return nil, err
    }
    appModule.Use(middleware.Logger(infra.Logger))

    scalarModule := scalar.NewModule("/scalar")

    return &Modules{
        API:    apiModule,
        App:    appModule,
        Scalar: scalarModule,
    }, nil
}

func buildRouter(infra *runtime.Infrastructure) *module.Router {
    router := module.NewRouter()

    router.HandleNative("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })

    router.HandleNative("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
        if !infra.Lifecycle.Ready() {
            w.WriteHeader(http.StatusServiceUnavailable)
            w.Write([]byte("NOT READY"))
            return
        }
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("READY"))
    })

    return router
}
```

#### Step 6: Update `internal/api/api.go`

Simplified signature (5 params → 2 params):

```go
package api

import (
    "net/http"

    "github.com/JaimeStill/agent-lab/internal/config"
    "github.com/JaimeStill/agent-lab/pkg/middleware"
    "github.com/JaimeStill/agent-lab/pkg/module"
    "github.com/JaimeStill/agent-lab/pkg/openapi"
    "github.com/JaimeStill/agent-lab/pkg/runtime"
)

func NewModule(cfg *config.Config, infra *runtime.Infrastructure) (*module.Module, error) {
    rt := newRuntime(cfg, infra)
    domain := newDomain(rt)

    spec := openapi.NewSpec(cfg.API.OpenAPI.Title, cfg.Version)
    spec.SetDescription(cfg.API.OpenAPI.Description)
    spec.AddServer(cfg.Domain)

    mux := http.NewServeMux()
    registerRoutes(mux, spec, rt, domain, cfg)

    specBytes, err := openapi.MarshalJSON(spec)
    if err != nil {
        return nil, err
    }
    mux.HandleFunc("GET /openapi.json", openapi.ServeSpec(specBytes))

    m := module.New(cfg.API.BasePath, mux)
    m.Use(middleware.CORS(&cfg.API.CORS))
    m.Use(middleware.Logger(rt.Logger))

    return m, nil
}
```

#### Step 7: Update `internal/api/runtime.go`

Unexport and embed infrastructure:

```go
package api

import (
    "github.com/JaimeStill/agent-lab/internal/config"
    "github.com/JaimeStill/agent-lab/pkg/pagination"
    "github.com/JaimeStill/agent-lab/pkg/runtime"
)

type apiRuntime struct {
    *runtime.Infrastructure
    Pagination pagination.Config
}

func newRuntime(cfg *config.Config, infra *runtime.Infrastructure) *apiRuntime {
    return &apiRuntime{
        Infrastructure: &runtime.Infrastructure{
            Lifecycle: infra.Lifecycle,
            Logger:    infra.Logger.With("module", "api"),
            Database:  infra.Database,
            Storage:   infra.Storage,
        },
        Pagination: cfg.API.Pagination,
    }
}
```

Note: We create a new Infrastructure with a modified Logger (`.With("module", "api")`) rather than reusing the pointer.

#### Step 8: Update `internal/api/domain.go`

Change parameter type and unexport:

```go
func newDomain(rt *apiRuntime) *Domain {
    // ... rest unchanged, but references change from runtime.X to rt.X ...
}
```

#### Step 9: Update `internal/api/routes.go`

Update `registerRoutes` signature:

```go
func registerRoutes(mux *http.ServeMux, spec *openapi.Spec, rt *apiRuntime, domain *Domain, cfg *config.Config) {
    // ... unchanged ...
}
```

### Validation

```bash
go vet ./...
go test ./tests/...
go run ./cmd/server
```

Verify API endpoints still work:
```bash
curl http://localhost:8080/api/providers
curl http://localhost:8080/api/openapi.json
```

---

## Phase 4: Simplify Handler Initialization

### Problem

`internal/api/routes.go` manually creates handlers for each domain, requiring repetitive boilerplate and passing the same logger/pagination to each:

```go
func registerRoutes(mux, spec, runtime, domain, cfg) {
    providerHandler := providers.NewHandler(domain.Providers, runtime.Logger, runtime.Pagination)
    agentsHandler := agents.NewHandler(domain.Agents, runtime.Logger, runtime.Pagination)
    documentsHandler := documents.NewHandler(domain.Documents, runtime.Logger, runtime.Pagination, cfg.Storage.MaxUploadSizeBytes())
    // ... create all handlers ...

    routes.Register(mux, cfg.API.BasePath, spec,
        providerHandler.Routes(),
        // ...
    )
}
```

### Solution

Add a `Handler()` factory method to each System that creates the handler on demand. This:
- Avoids circular dependency (Handler depends on System, System doesn't store Handler)
- Allows domain-specific parameters (e.g., `maxUploadSize` for documents)
- Simplifies route registration

#### Pattern for Most Domains

**system.go** - Add Handler to interface:
```go
type System interface {
    // ... existing methods ...

    Handler() *Handler
}
```

**repository.go** - Add Handler method:
```go
func (r *repo) Handler() *Handler {
    return NewHandler(r, r.logger, r.pagination)
}
```

#### Pattern for Documents (with extra parameter)

**system.go** - Handler takes maxUploadSize:
```go
type System interface {
    // ... existing methods ...

    Handler(maxUploadSize int64) *Handler
}
```

**repository.go** - Add Handler method:
```go
func (r *repo) Handler(maxUploadSize int64) *Handler {
    return NewHandler(r, r.logger, r.pagination, maxUploadSize)
}
```

#### Pattern for Workflows (executor implements System)

**system.go** - Add Handler to interface:
```go
type System interface {
    // ... existing methods ...

    Handler() *Handler
}
```

**executor.go** - Add Handler method (not repository.go):
```go
func (e *executor) Handler() *Handler {
    return NewHandler(e, e.logger, e.pagination)
}
```

> **Note:** Workflows is different because `repo` only implements repository methods. The `executor` implements the full System interface, so `Handler()` goes on executor.

#### Step 1: Update `internal/providers`

**system.go** - Add Handler to interface:
```go
type System interface {
    List(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Provider], error)
    Find(ctx context.Context, id uuid.UUID) (*Provider, error)
    Create(ctx context.Context, cmd CreateCommand) (*Provider, error)
    Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Provider, error)
    Delete(ctx context.Context, id uuid.UUID) error
    Handler() *Handler
}
```

**repository.go** - Add Handler method:
```go
func (r *repo) Handler() *Handler {
    return NewHandler(r, r.logger, r.pagination)
}
```

#### Step 2: Update `internal/agents`

Same pattern as providers.

#### Step 3: Update `internal/documents`

**system.go** - Handler takes maxUploadSize:
```go
type System interface {
    // ... existing methods ...
    Handler(maxUploadSize int64) *Handler
}
```

**repository.go** - Add Handler method:
```go
func (r *repo) Handler(maxUploadSize int64) *Handler {
    return NewHandler(r, r.logger, r.pagination, maxUploadSize)
}
```

#### Step 4: Update `internal/images`

Same pattern as providers.

#### Step 5: Update `internal/profiles`

Same pattern as providers.

#### Step 6: Update `internal/workflows`

**system.go** - Add Handler to interface:
```go
type System interface {
    // ... existing methods ...
    Handler() *Handler
}
```

**executor.go** - Add Handler method:
```go
func (e *executor) Handler() *Handler {
    return NewHandler(e, e.logger, e.pagination)
}
```

#### Step 7: Simplify `internal/api/routes.go`

```go
package api

import (
    "net/http"

    "github.com/JaimeStill/agent-lab/internal/config"
    "github.com/JaimeStill/agent-lab/pkg/openapi"
    "github.com/JaimeStill/agent-lab/pkg/routes"
)

func registerRoutes(mux *http.ServeMux, spec *openapi.Spec, domain *Domain, cfg *config.Config) {
    routes.Register(
        mux,
        cfg.API.BasePath,
        spec,
        domain.Providers.Handler().Routes(),
        domain.Agents.Handler().Routes(),
        domain.Documents.Handler(cfg.Storage.MaxUploadSizeBytes()).Routes(),
        domain.Images.Handler().Routes(),
        domain.Profiles.Handler().Routes(),
        domain.Workflows.Handler().Routes(),
    )
}
```

#### Step 8: Update `internal/api/api.go`

Remove `rt` parameter from `registerRoutes` call:

```go
func NewModule(cfg *config.Config, infra *runtime.Infrastructure) (*module.Module, error) {
    rt := newRuntime(cfg, infra)
    domain := newDomain(rt)

    spec := openapi.NewSpec(cfg.API.OpenAPI.Title, cfg.Version)
    spec.SetDescription(cfg.API.OpenAPI.Description)
    spec.AddServer(cfg.Domain)

    mux := http.NewServeMux()
    registerRoutes(mux, spec, domain, cfg)  // rt removed

    // ... rest unchanged ...
}
```

### Validation

```bash
go vet ./...
go test ./tests/...
go run ./cmd/server
```

Verify all API endpoints still work:
```bash
curl http://localhost:8080/api/providers
curl http://localhost:8080/api/agents
curl http://localhost:8080/api/documents
curl http://localhost:8080/api/workflows
```

---

## Files Changed

### Phase 1: Path Normalization

| File | Change |
|------|--------|
| `pkg/module/router.go` | Add `normalizePath()` function |
| `pkg/middleware/slash.go` | **Delete** |
| `tests/pkg_middleware/slash_test.go` | **Delete** |
| `cmd/server/modules.go` | Remove AddSlash usage |
| `web/app/app.go` | Remove trailing slashes from routes |
| `web/app/server/layouts/app.html` | Remove trailing slashes from nav hrefs |

### Phase 2: 404 Bundle Fix

| File | Change |
|------|--------|
| `pkg/web/pages.go` | Refactor `ErrorHandler` to take `PageDef` |
| `web/app/app.go` | Add `Bundle` to errorPages, update `ErrorHandler` call |

### Phase 3: Infrastructure Extraction

| File | Change |
|------|--------|
| `pkg/runtime/infrastructure.go` | **New** - Infrastructure struct with `New()` and `Start()` |
| `pkg/runtime/logging.go` | **New** - `newLogger()` moved from cmd/server |
| `cmd/server/runtime.go` | **Delete** |
| `cmd/server/logging.go` | **Delete** |
| `cmd/server/server.go` | Use `*runtime.Infrastructure` directly, rename field to `infra` |
| `cmd/server/modules.go` | Change param to `*runtime.Infrastructure`, update `buildRouter` |
| `internal/api/api.go` | Simplified signature (5 params → 2 params), use `newRuntime`/`newDomain` |
| `internal/api/runtime.go` | Rename to unexported `apiRuntime`, embed `*runtime.Infrastructure` |
| `internal/api/domain.go` | Rename to unexported `newDomain`, change param to `*apiRuntime` |
| `internal/api/routes.go` | Update `registerRoutes` param type to `*apiRuntime` |

### Phase 4: Simplify Handler Initialization

| File | Change |
|------|--------|
| `internal/providers/system.go` | Add `Handler() *Handler` to interface |
| `internal/providers/repository.go` | Add `Handler()` method |
| `internal/agents/system.go` | Add `Handler() *Handler` to interface |
| `internal/agents/repository.go` | Add `Handler()` method |
| `internal/documents/system.go` | Add `Handler(maxUploadSize int64) *Handler` to interface |
| `internal/documents/repository.go` | Add `Handler(maxUploadSize int64)` method |
| `internal/images/system.go` | Add `Handler() *Handler` to interface |
| `internal/images/repository.go` | Add `Handler()` method |
| `internal/profiles/system.go` | Add `Handler() *Handler` to interface |
| `internal/profiles/repository.go` | Add `Handler()` method |
| `internal/workflows/system.go` | Add `Handler() *Handler` to interface |
| `internal/workflows/executor.go` | Add `Handler()` method (NOTE: uses executor, not repo) |
| `internal/api/routes.go` | Simplify to `domain.X.Handler().Routes()` |
| `internal/api/api.go` | Remove `rt` from `registerRoutes` call |

## Tests to Update

- `tests/pkg_module/router_test.go` - Add tests for `normalizePath` behavior
- `tests/pkg_middleware/` - Remove slash_test.go, update middleware_test.go if needed
- `tests/pkg_web/` - Update `ErrorHandler` test for new `PageDef` signature
