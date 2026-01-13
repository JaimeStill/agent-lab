# Client Development Architecture

This directory contains all client-side web infrastructure for agent-lab.

## Philosophy

**Build in CI, embed in Go, deploy without Node.js.**

- **CI/CD:** Builds web assets (Bun + Vite), then builds Go binary with embedded assets
- **Container:** Contains only the Go binary - no node_modules, no Node.js runtime
- **Air-gap:** Works in isolated environments with zero external dependencies

The `dist/` directory is **gitignored** - CI builds from source to ensure consistency. Local development requires running the build.

## Directory Structure

```
web/
├── README.md              # This file
├── package.json           # Bun dependencies
├── vite.config.ts         # Vite build configuration
├── tsconfig.json          # TypeScript configuration
├── .gitignore             # Ignores dist/, node_modules/
├── web.go                 # Go embedding and static handler
├── src/                   # TypeScript/CSS source
│   ├── core/              # Foundation layer
│   ├── design/            # Design system (CSS tokens, reset, themes)
│   ├── components/        # Web components
│   └── entries/           # Route-scoped entry points
│       ├── shared.ts      # Common components
│       └── docs.ts        # Scalar API docs
├── templates/             # Go HTML templates
│   └── layouts/
│       └── app.html       # Base app layout
├── docs/                  # API documentation
│   ├── docs.go            # Go handler (serves index.html)
│   └── index.html         # Scalar mount point
└── dist/                  # Build output (gitignored)
    ├── shared.js
    ├── docs.js            # Bundled Scalar
    └── docs.css
```

## Build Pipeline

### CI/CD Flow

```
┌─────────────────────────────────────────────────────────────┐
│                         CI/CD                               │
├─────────────────────────────────────────────────────────────┤
│  1. bun install                                             │
│  2. bun run build        → web/dist/                        │
│  3. go build ./cmd/server → embeds web/dist/                │
│  4. docker build          → container with Go binary only   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Container                              │
├─────────────────────────────────────────────────────────────┤
│  /app/server              (Go binary with embedded assets)  │
│                                                             │
│  No node_modules, no bun, no npm, no Node.js                │
└─────────────────────────────────────────────────────────────┘
```

## Development Workflow

### Initial Setup

```bash
cd web
bun install
bun run build
```

### Development Cycle

```bash
# After changing TypeScript/CSS
cd web && bun run build

# Run Go server
go run ./cmd/server
```

### Watch Mode

```bash
cd web
bun run dev          # Vite rebuilds on source changes
```

Run the Go server separately. Restart server and refresh browser to see changes.

## Embedding Patterns

### Static Assets (web.go)

All Vite-built assets are embedded and served from `/static/`:

```go
//go:embed dist/*
var distFS embed.FS

func Static() http.HandlerFunc {
    sub, _ := fs.Sub(distFS, "dist")
    fileServer := http.StripPrefix("/static/", http.FileServer(http.FS(sub)))
    return func(w http.ResponseWriter, r *http.Request) {
        fileServer.ServeHTTP(w, r)
    }
}
```

### Templates (web.go)

Go HTML templates for server-rendered pages:

```go
//go:embed templates/*
var templateFS embed.FS

func Templates() (*template.Template, error) {
    return template.ParseFS(templateFS, "templates/**/*.html")
}
```

### API Documentation (docs/docs.go)

Serves the Scalar mount point HTML:

```go
//go:embed index.html
var indexHTML []byte
```

## Scalar API Documentation

Scalar is bundled through Vite for automated dependency management.

### Architecture

- `src/entries/docs.ts` imports Scalar's ESM module
- Vite bundles to `dist/docs.js` and `dist/docs.css`
- `docs/index.html` mounts Scalar to a div
- Static handler serves bundled assets at `/static/`

### Initialization

Scalar uses programmatic initialization (not data attributes):

```typescript
// src/entries/docs.ts
import { createApiReference } from '@scalar/api-reference'
import '@scalar/api-reference/style.css'

createApiReference('#api-reference', {
  url: '/api/openapi.json',
})
```

```html
<!-- docs/index.html -->
<div id="api-reference"></div>
<script type="module" src="/static/docs.js"></script>
```

### Updating Scalar

```bash
cd web
bun update @scalar/api-reference
bun run build
```

## Vite Configuration

### Entry Points

Multiple entry points produce route-scoped bundles:

```typescript
lib: {
  entry: {
    shared: resolve(__dirname, 'src/entries/shared.ts'),
    docs: resolve(__dirname, 'src/entries/docs.ts'),
  },
  formats: ['es'],
  fileName: (_, entryName) => `${entryName}.js`,
}
```

### Path Aliases

Clean imports via TypeScript path aliases:

```typescript
resolve: {
  alias: {
    '@core': resolve(__dirname, 'src/core'),
    '@design': resolve(__dirname, 'src/design'),
    '@components': resolve(__dirname, 'src/components'),
  },
}
```

## Design System

Global CSS custom properties in `src/design/styles.css`:

```css
:root {
  --color-primary: #2563eb;
  --color-surface: #ffffff;
  --color-text: #1e293b;
}
```

Entry points import design tokens:

```typescript
import '@design/styles.css'
```

## CI/CD Integration

### Build Stage

```yaml
build:
  steps:
    # Build web assets
    - run: cd web && bun install && bun run build

    # Build Go binary (embeds web assets)
    - run: go build -o server ./cmd/server

    # Build container (Go binary only)
    - run: docker build -t agent-lab .
```

### Dockerfile Pattern

```dockerfile
# Build stage
FROM golang:1.23 AS builder
WORKDIR /app
COPY . .
RUN go build -o server ./cmd/server

# Runtime stage - no Node.js
FROM gcr.io/distroless/base-debian12
COPY --from=builder /app/server /server
ENTRYPOINT ["/server"]
```

Web assets must be built before the Docker build (or use a multi-stage build with Bun).

## Air-Gap Verification

Before deploying to air-gapped environments:

```bash
# Build everything
cd web && bun install && bun run build && cd ..
go build -o server ./cmd/server

# Check for external URLs in built assets
grep -r "http://" web/dist/ || echo "No http:// found"
grep -r "https://" web/dist/ || echo "No https:// found"

# Test with network disabled
# 1. Disconnect network
# 2. Run ./server
# 3. Verify all routes function without external fetches
```

## Troubleshooting

### Scalar UI not rendering

Check browser console for errors. Common issues:
- `dist/` not built: Run `cd web && bun run build`
- Server not restarted after build
- `process is not defined`: Ensure `define` is set in vite.config.ts

### Changes to src/ not reflected

1. Rebuild: `cd web && bun run build`
2. Restart Go server (re-embeds assets)
3. Hard refresh browser (Ctrl+Shift+R)

### go:embed errors during build

Ensure dist/ exists:
```bash
cd web && bun install && bun run build
```
