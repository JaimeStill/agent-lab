# Session 05a: Web Infrastructure

## Problem Context

Milestone 5 requires a hybrid web architecture combining Go templates with web component islands. Before building UI components, we need the foundational build pipeline:

- **Vite + Bun + TypeScript** for client-side asset compilation
- **Go embedding** for zero-dependency deployment
- **Scalar API docs** integration into the unified pipeline

Currently, `web/docs/` uses a manual pattern with `update-scalar.sh` downloading standalone Scalar bundles. This session unifies all web assets under a single Vite build pipeline.

## Architecture Approach

### Build Pipeline

```
web/src/           TypeScript/CSS source
    ↓
bun run build      Vite compilation
    ↓
web/dist/          ES modules + CSS bundles
    ↓
go:embed           Embedded in Go binary
    ↓
/static/           Served via http.FileServer
```

### Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Scalar integration | Vite ESM bundle | Automates dependency management, potential optimization |
| dist/ commit | .gitignore | Standard practice, CI builds from source |
| Static path | `/static/` | Clear separation from API routes |

### Directory Structure

```
web/
├── package.json
├── vite.config.ts
├── tsconfig.json
├── .gitignore
├── src/
│   ├── core/
│   │   └── index.ts
│   ├── design/
│   │   └── styles.css
│   ├── components/
│   │   └── index.ts
│   └── entries/
│       ├── shared.ts
│       └── docs.ts
├── templates/
│   └── layouts/
│       └── app.html
├── docs/
│   ├── docs.go
│   └── index.html
├── dist/                    (gitignored)
└── web.go
```

## Implementation Steps

### Phase 1: Build Infrastructure

#### Step 1.1: Create package.json

**File**: `web/package.json`

```json
{
  "name": "agent-lab-web",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite build --watch",
    "build": "vite build",
    "preview": "vite preview"
  },
  "devDependencies": {
    "vite": "^6.0.0",
    "typescript": "^5.7.0"
  },
  "dependencies": {
    "@scalar/api-reference": "^1.28.0"
  }
}
```

#### Step 1.2: Create vite.config.ts

**File**: `web/vite.config.ts`

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
        shared: resolve(__dirname, 'src/entries/shared.ts'),
        docs: resolve(__dirname, 'src/entries/docs.ts'),
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
      '@core': resolve(__dirname, 'src/core'),
      '@design': resolve(__dirname, 'src/design'),
      '@components': resolve(__dirname, 'src/components'),
    },
  },
})
```

#### Step 1.3: Create tsconfig.json

**File**: `web/tsconfig.json`

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
      "@core/*": ["./src/core/*"],
      "@design/*": ["./src/design/*"],
      "@components/*": ["./src/components/*"]
    }
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist"]
}
```

#### Step 1.4: Create .gitignore

**File**: `web/.gitignore`

```
node_modules/
dist/
```

### Phase 2: Source Structure

#### Step 2.1: Create core placeholder

**File**: `web/src/core/index.ts`

```typescript
export {}
```

#### Step 2.2: Create design placeholder

**File**: `web/src/design/styles.css`

```css
:root {
  --color-primary: #2563eb;
  --color-surface: #ffffff;
  --color-text: #1e293b;
}
```

#### Step 2.3: Create components placeholder

**File**: `web/src/components/index.ts`

```typescript
export {}
```

#### Step 2.4: Create shared entry point

**File**: `web/src/entries/shared.ts`

```typescript
import '@design/styles.css'
export * from '@core/index'
export * from '@components/index'
```

#### Step 2.5: Create docs entry point

**File**: `web/src/entries/docs.ts`

```typescript
import { createApiReference } from '@scalar/api-reference'
import '@scalar/api-reference/style.css'

createApiReference('#api-reference', {
  url: '/api/openapi.json',
  withDefaultFonts: false,
})
```

Note: `withDefaultFonts: false` disables Scalar's external font loading from fonts.scalar.com for air-gap compatibility.

#### Step 2.6: Create layout template placeholder

**File**: `web/templates/layouts/app.html`

```html
{{/* Base layout template for app pages */}}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .Title }} - agent-lab</title>
  <link rel="stylesheet" href="/static/{{ .Bundle }}.css">
</head>
<body>
  <nav class="app-nav">
    <a href="/app/workflows">Workflows</a>
    <a href="/app/documents">Documents</a>
    <a href="/app/profiles">Profiles</a>
    <a href="/app/agents">Agents</a>
    <a href="/app/providers">Providers</a>
  </nav>

  <main class="app-content">
    {{ block "content" . }}{{ end }}
  </main>

  <script type="module" src="/static/{{ .Bundle }}.js"></script>
</body>
</html>
```

### Phase 3: Go Integration

#### Step 3.1: Create web/web.go

**File**: `web/web.go`

```go
package web

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
)

//go:embed dist/*
var distFS embed.FS

//go:embed templates/*
var templateFS embed.FS

func Static() http.HandlerFunc {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("failed to create dist sub-filesystem: " + err.Error())
	}
	fileServer := http.StripPrefix("/static/", http.FileServer(http.FS(sub)))
	return func(w http.ResponseWriter, r *http.Request) {
		fileServer.ServeHTTP(w, r)
	}
}

func Templates() (*template.Template, error) {
	return template.ParseFS(templateFS, "templates/**/*.html")
}
```

#### Step 3.2: Update web/docs/index.html

**File**: `web/docs/index.html` (modify existing)

Update the script and stylesheet references to use the new static paths, and add system font overrides for air-gap compatibility:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Agent Lab API Documentation</title>
  <link rel="stylesheet" href="/static/docs.css">
  <style>
    :root {
      --scalar-font: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
      --scalar-font-code: ui-monospace, 'Cascadia Code', 'SF Mono', Menlo, Monaco, Consolas, monospace;
    }
  </style>
</head>
<body>
  <div id="api-reference"></div>
  <script type="module" src="/static/docs.js"></script>
</body>
</html>
```

The inline CSS overrides Scalar's `--scalar-font` and `--scalar-font-code` variables with system font stacks, ensuring no external font requests.

#### Step 3.3: Update web/docs/docs.go

**File**: `web/docs/docs.go` (modify existing)

Remove the JS and CSS embeds and routes, keeping only index.html:

```go
package docs

import (
	_ "embed"
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/routes"
)

//go:embed index.html
var indexHTML []byte

type Handler struct{}

func NewHandler(spec []byte) *Handler {
	return &Handler{}
}

func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix:      "/docs",
		Tags:        []string{"Documentation"},
		Description: "Interactive API documentation powered by Scalar",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.serveIndex},
		},
	}
}

func (h *Handler) serveIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(indexHTML)
}
```

#### Step 3.4: Update cmd/server/routes.go

**File**: `cmd/server/routes.go` (add static route)

Add import for web package and register static handler after domain handlers:

Add to imports:
```go
"github.com/JaimeStill/agent-lab/web"
```

Add after `docsHandler` registration (before `return nil`):
```go
r.RegisterRoute(routes.Route{
	Method:  "GET",
	Pattern: "/static/",
	Handler: web.Static(),
})
```

### Phase 4: Cleanup

#### Step 4.1: Remove obsolete files

Delete the following files:
- `web/update-scalar.sh`
- `web/docs/scalar.js`
- `web/docs/scalar.css`

### Phase 5: Build and Verify

#### Step 5.1: Measure current Scalar size (before deletion)

```bash
ls -lh web/docs/scalar.js web/docs/scalar.css
```

Record sizes for comparison.

#### Step 5.2: Install dependencies and build

```bash
cd web && bun install && bun run build
```

Expected output in `dist/`:
- `shared.js`
- `docs.js`
- `docs.css`

#### Step 5.3: Measure new bundle sizes

```bash
ls -lh web/dist/docs.js web/dist/docs.css
```

Compare to original sizes.

#### Step 5.4: Verify Go compilation

```bash
go vet ./...
```

#### Step 5.5: Start server and verify endpoints

```bash
go run ./cmd/server
```

Test:
- `GET http://localhost:8080/docs` - Scalar UI renders
- `GET http://localhost:8080/api/openapi.json` - Returns spec
- `GET http://localhost:8080/static/shared.js` - Returns JavaScript
- `GET http://localhost:8080/static/docs.js` - Returns Scalar bundle

#### Step 5.6: Functional verification

- Open `/docs` in browser
- Verify Scalar UI loads and displays API documentation
- Test "Try it" functionality on an endpoint

### Phase 6: Update Tests

#### Step 6.1: Update web_docs tests

**File**: `tests/web_docs/docs_test.go` (modify existing)

Remove tests for `/scalar.js` and `/scalar.css` routes. Keep:
- `TestNewHandler`
- `TestRoutes` (update expected patterns)
- `TestServeIndex`

#### Step 6.2: Add static handler tests

**File**: `tests/web/static_test.go` (new file)

Test coverage:
- Static() returns handler
- Handler serves files from dist/
- 404 for non-existent files
- Correct Content-Type headers

### Phase 7: Documentation

#### Step 7.1: Update web/README.md

Update to reflect:
- New directory structure with src/, templates/, dist/
- Vite build pipeline replacing update-scalar.sh
- .gitignore for dist/ and node_modules/
- Development workflow with bun install/build
- Scalar integration via Vite

## Validation Checklist

- [ ] `bun install` completes without errors
- [ ] `bun run build` produces dist/ with expected files
- [ ] `go vet ./...` passes
- [ ] Server starts without errors
- [ ] `/docs` renders Scalar UI correctly
- [ ] `/static/shared.js` returns content
- [ ] `/static/docs.js` returns Scalar bundle
- [ ] OpenAPI "Try it" functionality works
- [ ] Tests pass: `go test ./tests/...`

## Risk Mitigation

If Scalar ESM bundling doesn't work (initialization issues, tree-shaking problems):

**Fallback**: Use vite-plugin-static-copy to copy pre-built Scalar files:

```typescript
import { viteStaticCopy } from 'vite-plugin-static-copy'

plugins: [
  viteStaticCopy({
    targets: [
      {
        src: 'node_modules/@scalar/api-reference/dist/browser/standalone.js',
        dest: '',
        rename: 'docs.js'
      },
      {
        src: 'node_modules/@scalar/api-reference/dist/style.css',
        dest: '',
        rename: 'docs.css'
      }
    ]
  })
]
```

This still achieves automation while preserving working Scalar behavior.
