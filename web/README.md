# Web Client Architecture

This directory contains isolated, self-contained web clients for agent-lab.

## Philosophy

**Build in CI, embed in Go, deploy without Node.js.**

- **CI/CD**: Builds web assets (Bun + Vite), then builds Go binary with embedded assets
- **Container**: Contains only the Go binary - no node_modules, no Node.js runtime
- **Air-gap**: Works in isolated environments with zero external dependencies

Build outputs are **gitignored** - CI builds from source to ensure consistency.

## Directory Structure

Each web client is fully isolated in its own directory:

```
web/
├── app/                         # Main app client
│   ├── client/                  # TypeScript source
│   │   ├── app.ts               # Entry point → dist/app.js
│   │   ├── core/                # Utilities (created when needed)
│   │   ├── design/              # CSS architecture (@layers)
│   │   └── components/          # Web components (when needed)
│   ├── dist/                    # Build output (gitignored)
│   ├── public/                  # Static assets (favicons, manifest)
│   ├── server/                  # Go templates (SSR)
│   │   ├── layouts/
│   │   └── pages/
│   ├── app.go                   # Handler + Mount()
│   └── client.config.ts         # Vite client config
├── scalar/                      # Scalar OpenAPI UI
│   ├── app.ts                   # Entry point → scalar.js
│   ├── index.html               # Scalar mount point
│   ├── scalar.go                # Mount()
│   ├── client.config.ts         # Vite client config
│   └── [scalar.js/css]          # Build output (gitignored)
├── vite.client.ts               # Shared config module (ClientConfig, merge)
├── vite.config.ts               # Root config (merges all clients)
├── tsconfig.json
└── package.json
```

## URL Routing

| Path | Client | Description |
|------|--------|-------------|
| `/app/*` | app | Main application (SSR pages, assets, public files) |
| `/scalar/*` | scalar | OpenAPI documentation UI |

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

# Run Go server (embeds fresh assets)
go run ./cmd/server
```

Or use the Makefile from project root:

```bash
make dev    # Build web + run server
make web    # Build web assets only
make run    # Run server only
```

### Watch Mode

```bash
cd web
bun run dev    # Vite rebuilds on source changes
```

Run the Go server separately. Restart server to pick up new embedded assets.

## Vite Configuration

### Per-Client Configs

Each client defines a `client.config.ts` that exports a `ClientConfig`:

```typescript
// web/app/client.config.ts
import { resolve } from 'path'
import type { ClientConfig } from '../vite.client'

const config: ClientConfig = {
  name: 'app',
  aliases: {
    '@app/design': resolve(__dirname, 'client/design'),
    '@app/core': resolve(__dirname, 'client/core'),
    '@app/components': resolve(__dirname, 'client/components'),
  },
}

export default config
```

### ClientConfig Interface

```typescript
interface ClientConfig {
  name: string                    // Client identifier
  input?: string                  // Entry point (default: [name]/client/app.ts)
  output?: {
    entryFileNames?: string       // JS output (default: [name]/dist/app.js)
    assetFileNames?: string       // CSS output (default: [name]/dist/[name][extname])
  }
  aliases?: Record<string, string> // Path aliases
}
```

### Root Config

The root `vite.config.ts` imports and merges all client configs:

```typescript
import { defineConfig } from 'vite'
import { merge } from './vite.client'
import appConfig from './app/client.config.ts'
import scalarConfig from './scalar/client.config.ts'

export default defineConfig(merge([appConfig, scalarConfig]))
```

## Mount Pattern

Web clients implement a `Mount()` function for route registration:

```go
func (h *Handler) Mount(r routes.System, prefix string) {
    router := h.Router()

    // Exact match for /prefix
    r.RegisterRoute(routes.Route{
        Method:  "GET",
        Pattern: prefix,
        Handler: func(w http.ResponseWriter, req *http.Request) {
            req.URL.Path = "/"
            router.ServeHTTP(w, req)
        },
    })

    // Wildcard for /prefix/*
    r.RegisterRoute(routes.Route{
        Method:  "GET",
        Pattern: prefix + "/{path...}",
        Handler: http.StripPrefix(prefix, router).ServeHTTP,
    })
}
```

Each client has its own internal `http.ServeMux` router to handle requests properly.

## Asset Paths

- **SSR templates**: Use `{{ .BasePath }}` template variable for all URLs
- **Static HTML**: Use absolute paths (`/scalar/scalar.js`) - relative paths depend on trailing slash

## Adding a New Client

1. Create `web/[client-name]/` directory
2. Add TypeScript entry point and any source files
3. Implement `[client-name].go` with `Mount()` function
4. Create `client.config.ts` exporting `ClientConfig`
5. Import and add to `web/vite.config.ts` merge array
6. Register in `cmd/server/routes.go`

## CSS Architecture

The app client uses CSS `@layer` for cascade control:

```
client/design/
├── reset.css       # Box-sizing, margin reset, a11y
├── theme.css       # Color tokens, dark/light mode
├── layout.css      # Spacing scale, typography
├── components.css  # Semantic element classes
└── styles.css      # Layer orchestration
```

Layer order: `reset` → `theme` → `layout` → `components`

## CI/CD Integration

### Build Stage

```yaml
build:
  steps:
    - run: cd web && bun install && bun run build
    - run: go build -o server ./cmd/server
    - run: docker build -t agent-lab .
```

### Dockerfile Pattern

```dockerfile
FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
RUN go build -o server ./cmd/server

FROM gcr.io/distroless/base-debian12
COPY --from=builder /app/server /server
ENTRYPOINT ["/server"]
```

Web assets must be built before Docker build (or use multi-stage with Bun).

## Troubleshooting

### Assets not loading (404 or MIME errors)

1. Rebuild: `cd web && bun run build`
2. Restart Go server (re-embeds assets)
3. Hard refresh browser (Ctrl+Shift+R)

### Scalar UI blank page

Check browser console. Common issues:
- Build output missing: Run `bun run build`
- Server not rebuilt after Vite build (Go embeds at compile time)

### go:embed errors

Ensure build outputs exist before Go build:

```bash
cd web && bun install && bun run build
go build ./cmd/server
```
