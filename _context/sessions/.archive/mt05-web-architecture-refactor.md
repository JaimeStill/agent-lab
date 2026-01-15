# mt05 - Web Architecture Refactor

## Overview

Refactor the web client architecture to reduce organizational friction and establish reusable infrastructure in `pkg/web` and `pkg/routes`.

## Phase 1: Extract `pkg/routes` Types

Extract reusable route types from `internal/routes` to `pkg/routes`.

### 1.1 Create pkg/routes/route.go

```go
package routes

import (
	"net/http"

	"github.com/JaimeStill/agent-lab/pkg/openapi"
)

type Route struct {
	Method  string
	Pattern string
	Handler http.HandlerFunc
	OpenAPI *openapi.Operation
}
```

### 1.2 Create pkg/routes/group.go

```go
package routes

type Group struct {
	Prefix      string
	Tags        []string
	Description string
	Routes      []Route
	Children    []Group
}
```

### 1.3 Create pkg/routes/system.go

```go
package routes

import "net/http"

type System interface {
	RegisterGroup(group Group)
	RegisterRoute(route Route)
	Build() http.Handler
	Groups() []Group
	Routes() []Route
}
```

### 1.4 Update internal/routes/routes.go

Replace type definitions with imports from `pkg/routes`:

```go
package routes

import (
	"log/slog"
	"net/http"

	pkgroutes "github.com/JaimeStill/agent-lab/pkg/routes"
)

type routes struct {
	routes []pkgroutes.Route
	groups []pkgroutes.Group
	logger *slog.Logger
}

func New(logger *slog.Logger) pkgroutes.System {
	return &routes{
		logger: logger,
		groups: []pkgroutes.Group{},
		routes: []pkgroutes.Route{},
	}
}

func (r *routes) Groups() []pkgroutes.Group {
	return r.groups
}

func (r *routes) Routes() []pkgroutes.Route {
	return r.routes
}

func (r *routes) RegisterRoute(route pkgroutes.Route) {
	r.routes = append(r.routes, route)
}

func (r *routes) RegisterGroup(group pkgroutes.Group) {
	r.groups = append(r.groups, group)
}

func (r *routes) Build() http.Handler {
	mux := http.NewServeMux()

	for _, route := range r.routes {
		mux.HandleFunc(route.Method+" "+route.Pattern, route.Handler)
	}

	for _, group := range r.groups {
		r.registerGroup(mux, "", group)
	}

	return mux
}

func (r *routes) registerGroup(mux *http.ServeMux, parentPrefix string, group pkgroutes.Group) {
	fullPrefix := parentPrefix + group.Prefix
	for _, route := range group.Routes {
		pattern := fullPrefix + route.Pattern
		mux.HandleFunc(route.Method+" "+pattern, route.Handler)
	}
	for _, child := range group.Children {
		r.registerGroup(mux, fullPrefix, child)
	}
}
```

### 1.5 Delete internal/routes/group.go

The types are now in `pkg/routes`. Remove this file.

### 1.6 Update all imports

Update all files that import `internal/routes` to use the types from `pkg/routes`:

- Domain handlers (`internal/*/handler.go`) - change `routes.Route` and `routes.Group` references
- `cmd/server/routes.go` - update type references

**Pattern:** Keep importing `internal/routes` for `routes.New()`, but use `pkg/routes` types:

```go
import (
	"github.com/JaimeStill/agent-lab/internal/routes"
	pkgroutes "github.com/JaimeStill/agent-lab/pkg/routes"
)

// Use routes.New() for constructor
// Use pkgroutes.Route, pkgroutes.Group for types
```

**Or simpler:** Have `internal/routes` re-export the types:

```go
// internal/routes/routes.go - add at top after imports
type (
	Route  = pkgroutes.Route
	Group  = pkgroutes.Group
	System = pkgroutes.System
)
```

This allows existing code to continue using `routes.Route` without changes.

---

## Phase 2: Create `pkg/web` Infrastructure

### 2.1 Create pkg/web/pages.go

Uses pre-parsing at startup to avoid per-request template cloning. All pages are parsed once when `NewTemplateSet` is called, providing fail-fast behavior and zero per-request parsing overhead.

```go
package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/JaimeStill/agent-lab/pkg/routes"
)

type PageDef struct {
	Route    string
	Template string
	Title    string
	Bundle   string
}

type PageData struct {
	Title  string
	Bundle string
	Data   any
}

type TemplateSet struct {
	pages map[string]*template.Template
}

func NewTemplateSet(layoutFS, pageFS embed.FS, layoutGlob, pageSubdir string, pages []PageDef) (*TemplateSet, error) {
	layouts, err := template.ParseFS(layoutFS, layoutGlob)
	if err != nil {
		return nil, err
	}

	pageSub, err := fs.Sub(pageFS, pageSubdir)
	if err != nil {
		return nil, err
	}

	pageTemplates := make(map[string]*template.Template, len(pages))
	for _, p := range pages {
		t, err := layouts.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone layouts for %s: %w", p.Template, err)
		}
		_, err = t.ParseFS(pageSub, p.Template)
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", p.Template, err)
		}
		pageTemplates[p.Template] = t
	}

	return &TemplateSet{pages: pageTemplates}, nil
}

func (ts *TemplateSet) Render(w http.ResponseWriter, layoutName, pagePath string, data PageData) error {
	t, ok := ts.pages[pagePath]
	if !ok {
		return fmt.Errorf("template not found: %s", pagePath)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return t.ExecuteTemplate(w, layoutName, data)
}

func (ts *TemplateSet) PageHandler(layout string, page PageDef) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:  page.Title,
			Bundle: page.Bundle,
		}
		if err := ts.Render(w, layout, page.Template, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (ts *TemplateSet) PageRoutes(prefix, layout string, pages []PageDef) routes.Group {
	routeList := make([]routes.Route, len(pages))
	for i, page := range pages {
		routeList[i] = routes.Route{
			Method:  "GET",
			Pattern: page.Route,
			Handler: ts.PageHandler(layout, page),
		}
	}
	return routes.Group{
		Prefix: prefix,
		Routes: routeList,
	}
}
```

### 2.2 Create pkg/web/static.go

Separates bundle serving (Vite-built assets) from public file serving (favicons, manifest):

- `DistServer` - serves Vite-built JS/CSS from `dist/` at `/dist/`
- `PublicFile` - serves individual public files at root level
- `PublicFileRoutes` - generates routes for public files

```go
package web

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"time"

	"github.com/JaimeStill/agent-lab/pkg/routes"
)

func DistServer(fsys embed.FS, subdir, urlPrefix string) http.HandlerFunc {
	sub, err := fs.Sub(fsys, subdir)
	if err != nil {
		panic("failed to create sub-filesystem: " + err.Error())
	}
	server := http.StripPrefix(urlPrefix, http.FileServer(http.FS(sub)))
	return func(w http.ResponseWriter, r *http.Request) {
		server.ServeHTTP(w, r)
	}
}

func PublicFile(fsys embed.FS, subdir, filename string) http.HandlerFunc {
	path := subdir + "/" + filename
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := fsys.ReadFile(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, filename, time.Time{}, bytes.NewReader(data))
	}
}

func PublicFileRoutes(fsys embed.FS, subdir string, files ...string) []routes.Route {
	routeList := make([]routes.Route, len(files))
	for i, file := range files {
		routeList[i] = routes.Route{
			Method:  "GET",
			Pattern: "/" + file,
			Handler: PublicFile(fsys, subdir, file),
		}
	}
	return routeList
}

func ServeEmbeddedFile(data []byte, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}
}
```

---

## Phase 3: Directory Restructure

Execute the following file operations:

### 3.1 Create client/ directory

```bash
mkdir -p web/client
mv web/src/entries/shared.ts web/client/app.ts
mv web/src/design web/client/
mv web/src/core web/client/
mv web/src/components web/client/
rm -rf web/src
```

### 3.2 Create scalar/ directory

```bash
mv web/docs web/scalar
mv web/client/app.ts.bak web/scalar/app.ts 2>/dev/null || true
# Actually, docs.ts needs to come from src/entries which we already moved
# So we need to grab it before removing src/
```

**Corrected order:**

```bash
# First, save docs.ts before removing src/
mv web/src/entries/docs.ts /tmp/docs-app.ts

# Create client directory
mkdir -p web/client
mv web/src/entries/shared.ts web/client/app.ts
mv web/src/design web/client/
mv web/src/core web/client/
mv web/src/components web/client/
rm -rf web/src

# Create scalar directory (rename from docs)
mv web/docs web/scalar
mv /tmp/docs-app.ts web/scalar/app.ts
mv web/scalar/docs.go web/scalar/scalar.go
```

### 3.3 Create server/ directory

```bash
mkdir -p web/server
mv web/templates/layouts web/server/
mv web/templates/pages web/server/
rm -rf web/templates

# Flatten page directories
mv web/server/pages/home/home.html web/server/pages/home.html
mv web/server/pages/components/components.html web/server/pages/components.html
rm -rf web/server/pages/home web/server/pages/components
```

### 3.4 Keep public/ directory

The `public/` directory name aligns with the `Public()` function and `publicFiles` variable. No rename needed.

---

## Phase 4: Update Vite Configuration

### 4.1 Update web/vite.config.ts

```typescript
import { defineConfig } from 'vite'
import { resolve } from 'path'

export default defineConfig({
  define: {
    'process.env.NODE_ENV': JSON.stringify('production'),
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    lib: {
      entry: {
        app: resolve(__dirname, 'client/app.ts'),
        scalar: resolve(__dirname, 'scalar/app.ts'),
      },
      formats: ['es'],
      fileName: (_, entryName) => `${entryName}.js`,
    },
    cssCodeSplit: true,
    sourcemap: true,
    minify: true,
  },
  resolve: {
    alias: {
      '@core': resolve(__dirname, 'client/core'),
      '@design': resolve(__dirname, 'client/design'),
      '@components': resolve(__dirname, 'client/components'),
    },
  },
})
```

### 4.2 Update web/scalar/index.html

Update the script and stylesheet references to use `/dist/`:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Agent Lab - API Documentation</title>
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <link rel="stylesheet" href="/dist/scalar.css">
  <style>
    :root {
      --scalar-font: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
      --scalar-font-code: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, Monaco, 'Courier New', monospace;
    }
  </style>
</head>
<body>
  <div id="api-reference"></div>
  <script type="module" src="/dist/scalar.js"></script>
</body>
</html>
```

### 4.3 Update web/tsconfig.json

Update paths and include for the new directory structure:

```json
{
  "compilerOptions": {
    "target": "ES2024",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "noEmit": true,
    "skipLibCheck": true,
    "esModuleInterop": true,
    "allowSyntheticDefaultImports": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "paths": {
      "@core/*": ["./client/core/*"],
      "@design/*": ["./client/design/*"],
      "@components/*": ["./client/components/*"]
    }
  },
  "include": [
    "client/**/*",
    "scalar/**/*"
  ],
  "exclude": [
    "node_modules",
    "dist"
  ]
}
```

---

## Phase 5: Refactor web/web.go

```go
package web

import (
	"embed"
	"net/http"

	"github.com/JaimeStill/agent-lab/pkg/routes"
	pkgweb "github.com/JaimeStill/agent-lab/pkg/web"
)

//go:embed dist/*
var distFS embed.FS

//go:embed public/*
var publicFS embed.FS

//go:embed server/layouts/*
var layoutFS embed.FS

//go:embed server/pages/*
var pageFS embed.FS

var publicFiles = []string{
	"favicon.ico",
	"favicon-16x16.png",
	"favicon-32x32.png",
	"apple-touch-icon.png",
	"site.webmanifest",
}

var pages = []pkgweb.PageDef{
	{Route: "", Template: "home.html", Title: "Home", Bundle: "app"},
	{Route: "/components", Template: "components.html", Title: "Components", Bundle: "app"},
}

func Dist() http.HandlerFunc {
	return pkgweb.DistServer(distFS, "dist", "/dist/")
}

func Public() []routes.Route {
	return pkgweb.PublicFileRoutes(publicFS, "public", publicFiles...)
}

type Handler struct {
	templates *pkgweb.TemplateSet
}

func NewHandler() (*Handler, error) {
	ts, err := pkgweb.NewTemplateSet(layoutFS, pageFS, "server/layouts/*.html", "server/pages", pages)
	if err != nil {
		return nil, err
	}
	return &Handler{templates: ts}, nil
}

func (h *Handler) Routes() routes.Group {
	return h.templates.PageRoutes("/app", "app.html", pages)
}
```

### 5.1 Update web/server/layouts/app.html

Update asset paths to reflect new URL structure:

1. **Favicon paths** - change from `/app/` to root level:
```html
<link rel="icon" type="image/x-icon" href="/favicon.ico">
<link rel="apple-touch-icon" sizes="180x180" href="/apple-touch-icon.png">
<link rel="icon" type="image/png" sizes="32x32" href="/favicon-32x32.png">
<link rel="icon" type="image/png" sizes="16x16" href="/favicon-16x16.png">
```

2. **Bundle paths** - change from `/static/` to `/dist/`:
```html
<link rel="stylesheet" href="/dist/{{ .Bundle }}.css">
```

```html
<script type="module" src="/dist/{{ .Bundle }}.js"></script>
```

---

## Phase 6: Update web/scalar/scalar.go

```go
package scalar

import (
	_ "embed"
	"net/http"

	"github.com/JaimeStill/agent-lab/pkg/routes"
)

//go:embed index.html
var indexHTML []byte

func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(indexHTML)
	}
}

func Routes() routes.Group {
	return routes.Group{
		Prefix: "/scalar",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: Handler()},
		},
	}
}
```

---

## Phase 7: Update cmd/server/routes.go

Update imports and route registration:

```go
// Change import
import (
    // ... existing imports ...
    "github.com/JaimeStill/agent-lab/pkg/routes"   // for types
    "github.com/JaimeStill/agent-lab/web/scalar"   // was web/docs
)

// In registerRoutes function:

// Remove docsHandler := docs.NewHandler(specBytes) and r.RegisterGroup(docsHandler.Routes())

// Add scalar routes:
r.RegisterGroup(scalar.Routes())

// Change /static/ to /dist/ for Vite-built assets:
r.RegisterRoute(routes.Route{
    Method:  "GET",
    Pattern: "/dist/",
    Handler: web.Dist(),
})

// Add public file routes (favicons, manifest):
for _, route := range web.Public() {
    r.RegisterRoute(route)
}
```

---

## Phase 8: Makefile Infrastructure

Create a Makefile at project root for development and build workflows:

```makefile
.PHONY: dev build web run test vet clean

# Development: build web assets and run server
dev: web run

# Production build
build: web
	go build -o bin/server ./cmd/server

# Build web assets
web:
	cd web && bun run build

# Run the server
run:
	go run ./cmd/server

# Run tests
test:
	go test ./tests/...

# Run go vet
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf web/dist/
```

---

## Appendix A: Web Client Encapsulation

This appendix establishes a fully encapsulated web client that can be mounted at any sub-path (e.g., `/app`). All web client resources (pages, assets, public files) are served from that sub-path, isolating the web client from the raw server.

### A.1 Update pkg/web/pages.go

Add `basePath` to `TemplateSet` (set once at construction) and `BasePath` to `PageData` (passed to templates). This enforces the correct boundary - each web client at a different path gets its own TemplateSet.

```go
type PageData struct {
	Title    string
	Bundle   string
	BasePath string
	Data     any
}

type TemplateSet struct {
	pages    map[string]*template.Template
	basePath string
}

func NewTemplateSet(layoutFS, pageFS embed.FS, layoutGlob, pageSubdir, basePath string, pages []PageDef) (*TemplateSet, error) {
	layouts, err := template.ParseFS(layoutFS, layoutGlob)
	if err != nil {
		return nil, err
	}

	pageSub, err := fs.Sub(pageFS, pageSubdir)
	if err != nil {
		return nil, err
	}

	pageTemplates := make(map[string]*template.Template, len(pages))
	for _, p := range pages {
		t, err := layouts.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone layouts for %s: %w", p.Template, err)
		}
		_, err = t.ParseFS(pageSub, p.Template)
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", p.Template, err)
		}
		pageTemplates[p.Template] = t
	}

	return &TemplateSet{
		pages:    pageTemplates,
		basePath: basePath,
	}, nil
}

// ErrorHandler returns an HTTP handler that renders an error template with the given status code.
func (ts *TemplateSet) ErrorHandler(layout, template string, status int, title string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		data := PageData{Title: title, BasePath: ts.basePath}
		if err := ts.Render(w, layout, template, data); err != nil {
			http.Error(w, http.StatusText(status), status)
		}
	}
}

// PageHandler returns an HTTP handler that renders the given page.
func (ts *TemplateSet) PageHandler(layout string, page PageDef) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:    page.Title,
			Bundle:   page.Bundle,
			BasePath: ts.basePath,
		}
		if err := ts.Render(w, layout, page.Template, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
```

Remove `PageRoutes` method - routing is now handled by the web client's Router.

### A.2 Create pkg/web/router.go

Router wraps `http.ServeMux` with an optional fallback handler for unmatched routes. Other error scenarios (Unauthorized, Forbidden) are handled by middleware, not the Router.

```go
package web

import "net/http"

// Router wraps http.ServeMux with optional fallback handling for unmatched routes.
type Router struct {
	mux      *http.ServeMux
	fallback http.HandlerFunc
}

// NewRouter creates a Router with default behavior (no custom fallback).
func NewRouter() *Router {
	return &Router{mux: http.NewServeMux()}
}

// SetFallback sets the handler for unmatched routes. If not set, the default
// ServeMux behavior applies.
func (r *Router) SetFallback(handler http.HandlerFunc) {
	r.fallback = handler
}

// Handle registers a handler for the given pattern.
func (r *Router) Handle(pattern string, handler http.Handler) {
	r.mux.Handle(pattern, handler)
}

// HandleFunc registers a handler function for the given pattern.
func (r *Router) HandleFunc(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc(pattern, handler)
}

// ServeHTTP implements http.Handler with optional fallback handling.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	_, pattern := r.mux.Handler(req)
	if pattern == "" && r.fallback != nil {
		r.fallback.ServeHTTP(w, req)
		return
	}
	r.mux.ServeHTTP(w, req)
}
```

### A.3 Create web/server/pages/404.html

```html
{{ define "content" }}
<div class="error-page">
  <h1>404</h1>
  <p>The page you're looking for doesn't exist.</p>
  <a href="{{ .BasePath }}" class="btn btn-primary">Go Home</a>
</div>
{{ end }}
```

### A.4 Update web/server/layouts/app.html

Use `{{ .BasePath }}` for all asset and navigation URLs:

```html
<!DOCTYPE html>
<html lang="en">

<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .Title }} - Agent Lab</title>
  <link rel="icon" type="image/x-icon" href="{{ .BasePath }}/favicon.ico">
  <link rel="apple-touch-icon" sizes="180x180" href="{{ .BasePath }}/apple-touch-icon.png">
  <link rel="icon" type="image/png" sizes="32x32" href="{{ .BasePath }}/favicon-32x32.png">
  <link rel="icon" type="image/png" sizes="16x16" href="{{ .BasePath }}/favicon-16x16.png">
  <link rel="stylesheet" href="{{ .BasePath }}/dist/{{ .Bundle }}.css">
</head>

<body>
  <nav class="app-nav">
    <a href="{{ .BasePath }}" class="app-nav-brand">Agent Lab</a>
    <div class="app-nav-links">
      <a href="{{ .BasePath }}/workflows">Workflows</a>
      <a href="{{ .BasePath }}/documents">Documents</a>
      <a href="{{ .BasePath }}/profiles">Profiles</a>
      <a href="{{ .BasePath }}/agents">Agents</a>
      <a href="{{ .BasePath }}/providers">Providers</a>
      <a href="{{ .BasePath }}/components">Components</a>
    </div>
  </nav>
  <main class="app-content">
    {{ block "content" . }}{{ end }}
  </main>

  <script type="module" src="{{ .BasePath }}/dist/{{ .Bundle }}.js"></script>
</body>

</html>
```

### A.5 Update web/web.go

Refactor to expose a fully encapsulated Router. The `basePath` is passed to `NewTemplateSet` and used automatically by all handlers:

```go
package web

import (
	"embed"
	"net/http"

	pkgweb "github.com/JaimeStill/agent-lab/pkg/web"
)

//go:embed dist/*
var distFS embed.FS

//go:embed public/*
var publicFS embed.FS

//go:embed server/layouts/*
var layoutFS embed.FS

//go:embed server/pages/*
var pageFS embed.FS

var publicFiles = []string{
	"favicon.ico",
	"favicon-16x16.png",
	"favicon-32x32.png",
	"apple-touch-icon.png",
	"site.webmanifest",
}

var pages = []pkgweb.PageDef{
	{Route: "/{$}", Template: "home.html", Title: "Home", Bundle: "app"},
	{Route: "/components", Template: "components.html", Title: "Components", Bundle: "app"},
}

var errorPages = []pkgweb.PageDef{
	{Template: "404.html", Title: "Not Found"},
}

type Handler struct {
	templates *pkgweb.TemplateSet
}

func NewHandler(basePath string) (*Handler, error) {
	allPages := append(pages, errorPages...)
	ts, err := pkgweb.NewTemplateSet(
		layoutFS,
		pageFS,
		"server/layouts/*.html",
		"server/pages",
		basePath,
		allPages,
	)
	if err != nil {
		return nil, err
	}
	return &Handler{templates: ts}, nil
}

func (h *Handler) Router() http.Handler {
	r := pkgweb.NewRouter()
	r.SetFallback(h.templates.ErrorHandler("app.html", "404.html", http.StatusNotFound, "Not Found"))

	// Pages
	for _, page := range pages {
		r.HandleFunc("GET "+page.Route, h.templates.PageHandler("app.html", page))
	}

	// Dist assets (URL /dist/app.js matches embed path dist/app.js directly)
	r.Handle("GET /dist/", http.FileServer(http.FS(distFS)))

	// Public files
	for _, route := range pkgweb.PublicFileRoutes(publicFS, "public", publicFiles...) {
		r.HandleFunc(route.Method+" "+route.Pattern, route.Handler)
	}

	return r
}
```

### A.6 Update cmd/server/routes.go

Mount the web client at `/app/`:

```go
// Create web handler with base path
webHandler, err := web.NewHandler("/app")
if err != nil {
    return nil, err
}

// Mount web client at /app/
r.RegisterRoute(routes.Route{
    Method:  "GET",
    Pattern: "/app/",
    Handler: http.StripPrefix("/app", webHandler.Router()).ServeHTTP,
})

// Remove old web routes:
// - web.Dist() route
// - web.Public() routes
// - webHandler.Routes() group
```

### A.7 Architecture Summary

**Raw server mux (root `/`):**
- `GET /api/*` - API routes (JSON)
- `GET /healthz` - Liveness check
- `GET /readyz` - Readiness check
- `GET /scalar` - OpenAPI UI
- `GET /app/*` - Web client (mounted)

**Web client mux (mounted at `/app`):**
- `GET /` → Home page
- `GET /components` → Components page
- `GET /dist/*` → Vite bundles
- `GET /favicon.ico`, etc. → Public files
- `GET /*` (unmatched) → 404 page

The web client is fully portable - change the `basePath` parameter to mount it at any location.

---

## Verification

After completing all phases:

1. `go vet ./...`
2. `cd web && bun run build`
3. `go run ./cmd/server`
4. Test routes:
   - `GET /app` - Home page
   - `GET /app/components` - Components page
   - `GET /scalar` - Scalar OpenAPI UI
   - `GET /dist/app.js` - App bundle
   - `GET /favicon.ico` - Favicon
   - `GET /nonexistent` - 404 page

---

## Appendix B: Isolated Web Clients

This appendix establishes fully isolated web clients where each `web/[client-name]/` directory is completely self-contained and mounts to `/[client-name]`. No shared dependencies between clients.

### Problem Context

During Appendix A implementation, routing issues emerged due to trailing-slash-removal middleware:

1. **Go's ServeMux behavior**: Pattern `/app/` redirects `/app` to `/app/` (301)
2. **Middleware conflict**: Trailing slash removal + redirect = infinite loop
3. **Bandaid solution**: Separate `/app` exact match + root-level `/dist/{path...}` for scalar

The bandaid works but creates friction:
- Multiple route registrations per web client
- Root-level `/dist/` route pollutes main router namespace
- `web.Dist()` method exists solely for scalar's benefit
- Shared `dist/` directory between clients breaks isolation

### Target Directory Structure

```
web/
├── app/                      # Main app - fully self-contained
│   ├── client/               # TypeScript source
│   │   ├── app.ts            # Entry point
│   │   ├── design/           # CSS design system
│   │   ├── core/             # TypeScript utilities
│   │   └── components/       # Web components
│   ├── dist/                 # Vite build output (gitignored)
│   │   ├── app.js
│   │   └── app.css
│   ├── public/               # Static assets (favicons, manifest)
│   ├── server/               # Go templates
│   │   ├── layouts/
│   │   └── pages/
│   └── app.go                # Handler + Mount
├── scalar/                   # Scalar - fully self-contained
│   ├── app.ts                # Entry point
│   ├── index.html            # Mount HTML (serves at /scalar)
│   ├── scalar.js             # Vite build output (gitignored)
│   ├── scalar.css            # Vite build output (gitignored)
│   └── scalar.go             # Handler + Mount
├── vite.config.ts            # Builds all clients to their directories
├── package.json
└── tsconfig.json
```

**Principles:**
- Each client directory contains everything it needs: source, build output, templates, handler
- Vite outputs each client's bundle to its own directory
- Adding a new client = create `web/[client-name]/` with standard structure
- `[client-name].go` implements `Mount()` for self-registration

---

### B.1 Move Files to web/app/

```bash
# Create app directory structure
mkdir -p web/app

# Move client-side code
mv web/client web/app/

# Move server-side templates
mv web/server web/app/

# Move public assets
mv web/public web/app/

# Move dist (will be recreated by build)
mv web/dist web/app/

# Move and rename handler
mv web/web.go web/app/app.go
```

### B.2 Update web/app/app.go

Change package name and add `Mount()` method:

```go
package app

import (
	"embed"
	"net/http"

	"github.com/JaimeStill/agent-lab/pkg/routes"
	pkgweb "github.com/JaimeStill/agent-lab/pkg/web"
)

//go:embed dist/*
var distFS embed.FS

//go:embed public/*
var publicFS embed.FS

//go:embed server/layouts/*
var layoutFS embed.FS

//go:embed server/pages/*
var pageFS embed.FS

var publicFiles = []string{
	"favicon.ico",
	"favicon-16x16.png",
	"favicon-32x32.png",
	"apple-touch-icon.png",
	"site.webmanifest",
}

var pages = []pkgweb.PageDef{
	{Route: "/{$}", Template: "home.html", Title: "Home", Bundle: "app"},
	{Route: "/components", Template: "components.html", Title: "Components", Bundle: "app"},
}

var errorPages = []pkgweb.PageDef{
	{Template: "404.html", Title: "Not Found"},
}

type Handler struct {
	templates *pkgweb.TemplateSet
}

func NewHandler(basePath string) (*Handler, error) {
	allPages := append(pages, errorPages...)
	ts, err := pkgweb.NewTemplateSet(
		layoutFS,
		pageFS,
		"server/layouts/*.html",
		"server/pages",
		basePath,
		allPages,
	)
	if err != nil {
		return nil, err
	}
	return &Handler{templates: ts}, nil
}

func (h *Handler) Router() http.Handler {
	r := pkgweb.NewRouter()
	r.SetFallback(h.templates.ErrorHandler(
		"app.html",
		"404.html",
		http.StatusNotFound,
		"Not Found",
	))

	for _, page := range pages {
		r.HandleFunc("GET "+page.Route, h.templates.PageHandler("app.html", page))
	}

	r.Handle("GET /dist/", http.FileServer(http.FS(distFS)))

	for _, route := range pkgweb.PublicFileRoutes(publicFS, "public", publicFiles...) {
		r.HandleFunc(route.Method+" "+route.Pattern, route.Handler)
	}

	return r
}

// Mount registers the app client at the given prefix.
// Handles both exact-match and wildcard patterns for compatibility
// with trailing-slash-removal middleware.
func (h *Handler) Mount(r routes.System, prefix string) {
	router := h.Router()

	// Exact match for prefix (e.g., /app)
	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: prefix,
		Handler: func(w http.ResponseWriter, req *http.Request) {
			req.URL.Path = "/"
			router.ServeHTTP(w, req)
		},
	})

	// Wildcard for all paths under prefix (e.g., /app/components)
	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: prefix + "/{path...}",
		Handler: http.StripPrefix(prefix, router).ServeHTTP,
	})
}
```

### B.3 Update web/scalar/scalar.go

Replace with Mount-based implementation that uses an internal router (same pattern as app). Using `http.FileServer` directly with path rewriting doesn't work cleanly when mounted via `routes.System` - the FileServer causes unexpected redirects.

```go
// Package scalar provides the interactive API documentation handler using Scalar UI.
// Assets are embedded at compile time for zero-dependency deployment.
package scalar

import (
	"embed"
	"net/http"

	"github.com/JaimeStill/agent-lab/pkg/routes"
)

//go:embed index.html scalar.css scalar.js
var staticFS embed.FS

// Mount registers the scalar client at the given prefix.
func Mount(r routes.System, prefix string) {
	router := newRouter()

	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: prefix,
		Handler: func(w http.ResponseWriter, req *http.Request) {
			req.URL.Path = "/"
			router.ServeHTTP(w, req)
		},
	})

	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: prefix + "/{path...}",
		Handler: http.StripPrefix(prefix, router).ServeHTTP,
	})
}

func newRouter() http.Handler {
	mux := http.NewServeMux()

	// Serve index.html at root
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data, _ := staticFS.ReadFile("index.html")
		w.Write(data)
	})

	// Serve static assets
	mux.Handle("GET /", http.FileServer(http.FS(staticFS)))

	return mux
}
```

### B.4 Update web/scalar/index.html

Use absolute paths with the `/scalar/` prefix. This ensures assets resolve correctly regardless of whether the user accesses `/scalar` or `/scalar/`:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Agent Lab - API Documentation</title>
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <link rel="stylesheet" href="/scalar/scalar.css">
  <style>
    :root {
      --scalar-font: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
      --scalar-font-code: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, Monaco, 'Courier New', monospace;
    }
  </style>
</head>
<body>
  <div id="api-reference"></div>
  <script type="module" src="/scalar/scalar.js"></script>
</body>
</html>
```

### B.5 Create web/vite.client.ts

Create a shared module with types, defaults, and merge function:

```typescript
import { resolve } from 'path'
import type { PreRenderedAsset, PreRenderedChunk, RollupOptions } from 'rollup'
import type { UserConfig } from 'vite'

export interface ClientConfig {
  name: string
  input?: string
  output?: {
    entryFileNames?: string | ((chunk: PreRenderedChunk) => string)
    assetFileNames?: string | ((asset: PreRenderedAsset) => string)
  }
  aliases?: Record<string, string>
}

const root = __dirname

export function merge(clients: ClientConfig[]): UserConfig {
  return {
    build: {
      outDir: '.',
      emptyOutDir: false,
      rollupOptions: mergeRollup(clients),
    },
    resolve: mergeResolve(clients),
  }
}

function mergeRollup(clients: ClientConfig[]): RollupOptions {
  return {
    input: Object.fromEntries(
      clients.map(c => [c.name, c.input ?? defaultInput(c.name)])
    ),
    output: {
      entryFileNames: (chunk) => {
        const client = clients.find(c => c.name === chunk.name)
        const custom = client?.output?.entryFileNames
        if (custom) return typeof custom === 'function' ? custom(chunk) : custom
        return defaultEntry(chunk.name)
      },
      assetFileNames: (asset) => {
        const originalPath = asset.originalFileNames?.[0] ?? ''
        const client = clients.find(c => originalPath.startsWith(`${c.name}/`))
        if (client?.output?.assetFileNames) {
          const custom = client.output.assetFileNames
          return typeof custom === 'function' ? custom(asset) : custom
        }
        return client ? defaultAssets(client.name) : 'app/dist/[name][extname]'
      },
    },
  }
}

function mergeResolve(clients: ClientConfig[]): UserConfig['resolve'] {
  return {
    alias: Object.assign({}, ...clients.map(c => c.aliases ?? {})),
  }
}

function defaultInput(name: string) {
  return resolve(root, `${name}/client/app.ts`)
}

function defaultEntry(name: string) {
  return `${name}/dist/app.js`
}

function defaultAssets(name: string) {
  return `${name}/dist/[name][extname]`
}
```

### B.6 Create Per-Client Vite Configs

Each client defines only what differs from defaults.

**web/app/vite.config.ts** (new file):

```typescript
import { resolve } from 'path'
import type { ClientConfig } from '../vite.client'

const root = __dirname

const config: ClientConfig = {
  name: 'app',
  aliases: {
    '@app/design': resolve(root, 'client/design'),
    '@app/core': resolve(root, 'client/core'),
    '@app/components': resolve(root, 'client/components'),
  },
}

export default config
```

**web/scalar/vite.config.ts** (new file):

```typescript
import { resolve } from 'path'
import type { ClientConfig } from '../vite.client'

const config: ClientConfig = {
  name: 'scalar',
  input: resolve(__dirname, 'app.ts'),
  output: {
    entryFileNames: 'scalar/scalar.js',
    assetFileNames: 'scalar/scalar.css',
  },
}

export default config
```

### B.7 Update web/vite.config.ts

Replace with simple merge call:

```typescript
import { defineConfig } from 'vite'
import { merge } from './vite.client'
import appConfig from './app/vite.config'
import scalarConfig from './scalar/vite.config'

export default defineConfig(merge([appConfig, scalarConfig]))
```

### B.8 Update web/tsconfig.json

Update paths for client-scoped aliases:

```json
{
  "compilerOptions": {
    "target": "ES2024",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "noEmit": true,
    "skipLibCheck": true,
    "esModuleInterop": true,
    "allowSyntheticDefaultImports": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "paths": {
      "@app/core/*": ["./app/client/core/*"],
      "@app/design/*": ["./app/client/design/*"],
      "@app/components/*": ["./app/client/components/*"]
    }
  },
  "include": [
    "app/client/**/*",
    "scalar/**/*"
  ],
  "exclude": [
    "node_modules",
    "app/dist"
  ]
}
```

### B.9 Update web/app/client/app.ts

Change aliases from `@design`, `@core`, `@components` to `@app/*`:

```typescript
import '@app/design/styles.css'
export * from '@app/core/index'
export * from '@app/components/index'
```

### B.10 Update web/.gitignore

Add client-specific build outputs:

```
node_modules/
app/dist/
scalar/scalar.js
scalar/scalar.css
```

### B.11 Update cmd/server/routes.go

Replace web imports and simplify to Mount() calls:

```go
// Change imports
import (
	// ... existing imports ...
	"github.com/JaimeStill/agent-lab/web/app"    // was "web"
	"github.com/JaimeStill/agent-lab/web/scalar"
)

// In registerRoutes function, replace all web client registration with:

// Mount app client at /app
appHandler, err := app.NewHandler("/app")
if err != nil {
	return err
}
appHandler.Mount(r, "/app")

// Mount scalar at /scalar
scalar.Mount(r, "/scalar")

// DELETE these routes (no longer needed):
// - r.RegisterRoute for /dist/{path...}
// - web.Dist()
// - r.RegisterRoute for /app
// - r.RegisterRoute for /app/{path...}
// - r.RegisterGroup(scalar.Routes())
```

### B.12 Update Tests

Move `tests/web/web_test.go` to `tests/web/app/app_test.go` and update imports:

```go
package app_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JaimeStill/agent-lab/web/app"
)

func TestNewHandler(t *testing.T) {
	h, err := app.NewHandler("/app")
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	if h == nil {
		t.Fatal("NewHandler() returned nil")
	}
}

// ... rest of tests with "web" replaced by "app"
```

### B.13 Delete Obsolete Files

```bash
# Remove old web.go (now at web/app/app.go)
# Already moved in B.1

# Remove shared dist (now at web/app/dist/)
# Already moved in B.1

# Delete scalar README if present
rm -f web/scalar/README.md
```

---

### Verification

After completing Appendix B:

1. `cd web && bun run build` - produces:
   - `web/app/dist/app.js` and `web/app/dist/app.css`
   - `web/scalar/scalar.js` and `web/scalar/scalar.css`
2. `go vet ./...` - passes
3. `go test ./tests/...` - passes
4. `go run ./cmd/server` - starts without errors
5. Test routes:
   - `GET /app` → Home page (no redirect loop)
   - `GET /app/components` → Components page
   - `GET /app/dist/app.js` → JS bundle
   - `GET /scalar` → Scalar UI
   - `GET /scalar/scalar.js` → Scalar bundle
6. Verify no root-level `/dist/` route exists
