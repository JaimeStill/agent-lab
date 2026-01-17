# mt07 - Module Polish

## Overview

Small maintenance session to fix two bugs and simplify API module initialization.

**Estimated Time:** 30-45 minutes

**Items:**
1. Fix AddSlash redirect at router level (not working for `/app` → `/app/`)
2. Fix 404 page empty bundle name
3. Extract shared infrastructure to `pkg/runtime`

---

## Phase 1: Fix AddSlash Redirect

### Problem

AddSlash middleware runs inside the module on the *internal* path. When `/app` arrives:
1. Router matches `/app` prefix → routes to app module
2. Module.Serve() strips prefix → internal path is `/`
3. AddSlash sees `/` → already has slash → no redirect

The redirect needs to happen *before* path stripping.

### Solution

Add redirect logic to `pkg/module/router.go` in `ServeHTTP`:

```go
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    path := req.URL.Path
    prefix := extractPrefix(path)

    if m, ok := r.modules[prefix]; ok {
        // Redirect exact prefix match without trailing slash
        // e.g., /app → /app/ (but not /api since API uses TrimSlash)
        if path == prefix {
            target := prefix + "/"
            if req.URL.RawQuery != "" {
                target += "?" + req.URL.RawQuery
            }
            http.Redirect(w, req, target, http.StatusMovedPermanently)
            return
        }

        m.Serve(w, req)
        return
    }

    r.native.ServeHTTP(w, req)
}
```

**Wait** - this redirects ALL modules, including API which uses TrimSlash. We need module-level control.

### Revised Solution

Add a `TrailingSlash` option to Module:

```go
// pkg/module/module.go

type Module struct {
    prefix        string
    router        http.Handler
    middleware    middleware.System
    trailingSlash bool  // If true, redirect /prefix to /prefix/
}

func New(prefix string, router http.Handler) *Module {
    // ... existing code ...
}

func (m *Module) SetTrailingSlash(enabled bool) {
    m.trailingSlash = enabled
}

func (m *Module) TrailingSlash() bool {
    return m.trailingSlash
}
```

Then in Router:

```go
// pkg/module/router.go

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    path := req.URL.Path
    prefix := extractPrefix(path)

    if m, ok := r.modules[prefix]; ok {
        // Handle trailing slash redirect for modules that want it
        if path == prefix && m.TrailingSlash() {
            target := prefix + "/"
            if req.URL.RawQuery != "" {
                target += "?" + req.URL.RawQuery
            }
            http.Redirect(w, req, target, http.StatusMovedPermanently)
            return
        }

        m.Serve(w, req)
        return
    }

    r.native.ServeHTTP(w, req)
}
```

Update module creation in `cmd/server/modules.go`:

```go
appModule, err := app.NewModule("/app")
if err != nil {
    return nil, err
}
appModule.SetTrailingSlash(true)  // Enable redirect for web client
appModule.Use(middleware.AddSlash())
appModule.Use(middleware.Logger(runtime.Logger))

scalarModule := scalar.NewModule("/scalar")
scalarModule.SetTrailingSlash(true)  // Enable redirect for web client
scalarModule.Use(middleware.AddSlash())
```

**Note:** API module does NOT call `SetTrailingSlash(true)` - it uses TrimSlash internally.

### Validation

```bash
curl -I http://localhost:8080/app
# Should return 301 with Location: /app/

curl -I http://localhost:8080/scalar
# Should return 301 with Location: /scalar/

curl -I http://localhost:8080/api/providers/
# Should return 301 with Location: /api/providers (TrimSlash)
```

---

## Phase 2: Fix 404 Page Bundle

### Problem

The 404 error page requests `/app/dist/.css` and `/app/dist/.js` - empty bundle name.

Looking at `pkg/web/template.go`, `ErrorHandler` creates page data without a Bundle field.

### Solution

Update `ErrorHandler` in `pkg/web/template.go` to accept a bundle name:

```go
func (ts *TemplateSet) ErrorHandler(layout, page string, status int, message, bundle string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(status)
        ts.Render(w, layout, page, PageData{
            Title:    message,
            BasePath: ts.basePath,
            Bundle:   bundle,
        })
    }
}
```

Update `web/app/app.go` to pass the bundle:

```go
r.SetFallback(ts.ErrorHandler(
    "app.html",
    "404.html",
    http.StatusNotFound,
    "Not Found",
    "app",  // Add bundle name
))
```

### Validation

Navigate to `http://localhost:8080/app/nonexistent/` - should render styled 404 page.

---

## Phase 3: Extract Shared Infrastructure

### Problem

API module initialization unpacks and repacks the same fields:

```go
// cmd/server/modules.go - unpacking
api.NewModule(cfg, runtime.Logger, runtime.Database, runtime.Storage, runtime.Lifecycle)

// internal/api/runtime.go - repacking
&Runtime{Logger: logger, Database: db, Storage: store, Lifecycle: lc, Pagination: ...}
```

### Solution

Create `pkg/runtime/infrastructure.go`:

```go
package runtime

import (
    "log/slog"

    "github.com/JaimeStill/agent-lab/pkg/database"
    "github.com/JaimeStill/agent-lab/pkg/lifecycle"
    "github.com/JaimeStill/agent-lab/pkg/storage"
)

type Infrastructure struct {
    Logger    *slog.Logger
    Database  database.System
    Storage   storage.System
    Lifecycle *lifecycle.Coordinator
}
```

Update `cmd/server/runtime.go`:

```go
package main

import (
    // ...
    "github.com/JaimeStill/agent-lab/pkg/runtime"
)

type Runtime struct {
    runtime.Infrastructure
}

func NewRuntime(cfg *config.Config) (*Runtime, error) {
    // ... existing initialization ...

    return &Runtime{
        Infrastructure: runtime.Infrastructure{
            Logger:    logger,
            Database:  db,
            Storage:   store,
            Lifecycle: lc,
        },
    }, nil
}
```

Update `internal/api/api.go`:

```go
func NewModule(cfg *config.Config, infra *runtime.Infrastructure) (*module.Module, error) {
    rt := newRuntime(cfg, infra)
    domain := newDomain(rt)
    // ... rest unchanged ...
}
```

Update `internal/api/runtime.go`:

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
            Logger:    infra.Logger.With("module", "api"),
            Database:  infra.Database,
            Storage:   infra.Storage,
            Lifecycle: infra.Lifecycle,
        },
        Pagination: cfg.API.Pagination,
    }
}
```

Update `internal/api/domain.go` to use `*apiRuntime` (lowercase, unexported).

Update `cmd/server/modules.go`:

```go
func NewModules(runtime *Runtime, cfg *config.Config) (*Modules, error) {
    apiModule, err := api.NewModule(cfg, &runtime.Infrastructure)
    if err != nil {
        return nil, err
    }
    // ... rest unchanged ...
}
```

### Validation

```bash
go vet ./...
go test ./tests/...
go run ./cmd/server
```

---

## Files Changed

| File | Change |
|------|--------|
| `pkg/module/module.go` | Add `trailingSlash` field and `SetTrailingSlash()` |
| `pkg/module/router.go` | Handle trailing slash redirect based on module setting |
| `pkg/web/template.go` | Add bundle parameter to `ErrorHandler` |
| `web/app/app.go` | Pass bundle name to `ErrorHandler` |
| `pkg/runtime/infrastructure.go` | **New** - shared infrastructure struct |
| `cmd/server/runtime.go` | Embed `runtime.Infrastructure` |
| `cmd/server/modules.go` | Pass infrastructure to API, set trailing slash on web modules |
| `internal/api/api.go` | Simplified signature with infrastructure |
| `internal/api/runtime.go` | Compose infrastructure, add API-specific fields |
| `internal/api/domain.go` | Update to use unexported runtime type |

## Tests to Update

- `tests/pkg_module/` - Add tests for `TrailingSlash` behavior
- `tests/pkg_web/` - Update `ErrorHandler` test for bundle parameter
