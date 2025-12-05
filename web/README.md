# Client Development Architecture

This directory contains all client-side web infrastructure for agent-lab.

## Philosophy

**Build in CI, embed in Go, deploy without Node.js.**

- **CI/CD:** Builds web assets (Bun + Vite), then builds Go binary with embedded assets
- **Container:** Contains only the Go binary - no node_modules, no Node.js runtime
- **Air-gap:** Works in isolated environments with zero external dependencies

The `dist/` directory is **committed to git** as a convenience for local development, but CI always rebuilds from source to ensure consistency.

## Directory Structure

```
web/
├── README.md              # This file
├── update-scalar.sh       # Script to update Scalar assets
├── docs/                  # API documentation (Scalar UI)
│   ├── docs.go            # Go handler with go:embed
│   ├── index.html         # HTML template
│   ├── scalar.js          # Scalar standalone bundle (committed)
│   └── scalar.css         # Scalar styles (committed)
├── src/                   # TypeScript/CSS source (future)
│   └── ...
├── dist/                  # Built assets (committed for convenience)
│   └── ...
├── package.json           # Bun dependencies (future)
├── vite.config.ts         # Vite configuration (future)
└── tsconfig.json          # TypeScript configuration (future)
```

## Build Pipeline

### CI/CD Flow

```
┌─────────────────────────────────────────────────────────────┐
│                         CI/CD                               │
├─────────────────────────────────────────────────────────────┤
│  1. bun install                                             │
│  2. bun run build        → web/dist/                        │
│  3. go build ./cmd/server → embeds web/dist/ and web/docs/  │
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

### Why Commit dist/?

Committing built assets provides:
- **Local development** without running build pipeline
- **Quick iteration** - edit Go code, run immediately
- **Fallback** if build tooling isn't available locally

CI rebuilds anyway, so committed `dist/` is a convenience, not a source of truth.

## Current State

The `docs/` subdirectory uses a simple embedding pattern:
- Pre-built Scalar assets downloaded via `update-scalar.sh`
- Assets embedded directly via `go:embed` in `docs.go`
- No Vite build required for Scalar

## Future State

When building custom client UI (Milestone 6+), the full build pipeline activates:

1. **Source files** in `src/` (TypeScript, CSS)
2. **Vite builds** to `dist/` with tree-shaking
3. **Go embeds** from `dist/`
4. **CI rebuilds** both web and Go on every build

## Development Workflow

### Updating Scalar Assets

```bash
./web/update-scalar.sh
```

This downloads the latest Scalar bundle and copies assets to `docs/`.

### Building Custom Client Assets (Future)

```bash
cd web
bun install          # Install dependencies
bun run build        # Build to dist/
```

### Development Watch Mode (Future)

```bash
cd web
bun run dev          # Vite rebuilds on source changes
```

Run the Go server separately. Restart server and refresh browser to see changes.

### Local Development Cycle

```bash
# Option 1: Use committed dist/ (quick)
go run ./cmd/server

# Option 2: Rebuild assets first (ensures latest)
cd web && bun run build && cd ..
go run ./cmd/server
```

## Embedding Patterns

### Pattern 1: Pre-built Third-Party Assets

Used for Scalar and other pre-built libraries:

```go
//go:embed scalar.js
var scalarJS []byte

//go:embed scalar.css
var scalarCSS []byte
```

Assets downloaded via update scripts and embedded directly.

### Pattern 2: Vite-Built Custom Assets (Future)

Used for custom TypeScript/CSS:

```go
//go:embed dist/*
var distFS embed.FS
```

Source lives in `src/`, Vite builds to `dist/`, Go embeds from `dist/`.

## Scalar UI Notes

### Initialization

Scalar uses **data attribute initialization** (not JavaScript API):

```html
<script
  id="api-reference"
  data-url="/docs/openapi.json"
  src="/docs/scalar.js">
</script>
```

The JavaScript API (`Scalar.createApiReference()`) has issues with the standalone bundle. The data attribute approach works reliably.

### Required Files

- `scalar.js` - Standalone JavaScript bundle
- `scalar.css` - Stylesheet (linked in HTML head)
- `index.html` - HTML template with data attributes

### Updating Scalar Version

```bash
./web/update-scalar.sh
# Test at http://localhost:8080/docs
git add web/docs/scalar.js web/docs/scalar.css
git commit -m "chore: update Scalar"
```

## Vite Configuration (Future)

When custom client UI is needed:

```typescript
// web/vite.config.ts
import { defineConfig } from 'vite'
import { resolve } from 'path'

export default defineConfig({
  root: 'src',
  build: {
    outDir: '../dist',
    emptyOutDir: true,
    lib: {
      entry: {
        'app': resolve(__dirname, 'src/app/main.ts'),
      },
      formats: ['es'],
      fileName: (_, entryName) => `${entryName}.js`,
    },
    rollupOptions: {
      output: {
        assetFileNames: '[name][extname]',
      },
    },
    sourcemap: true,
  },
})
```

### Route-Scoped Bundles

Each client route gets its own entry point:

```
src/
├── styles.css              # Global design tokens
├── app/
│   ├── main.ts             # App entry point
│   └── styles.css          # App-specific styles
└── workflows/
    ├── main.ts             # Workflows entry point
    └── styles.css          # Workflows-specific styles
```

This produces separate bundles that only include what each route needs.

## Design Tokens (Future)

Global CSS custom properties in `src/styles.css`:

```css
:root {
  --color-primary: #2563eb;
  --color-surface: #ffffff;
  --color-text: #1e293b;
  --space-4: 1rem;
  --font-sans: system-ui, -apple-system, sans-serif;
}
```

Route-specific styles import and use these tokens:

```css
@import '../styles.css';

.container {
  padding: var(--space-4);
  color: var(--color-text);
}
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

Web assets are built before the Docker build (or in a multi-stage build with Bun).

## Air-Gap Verification

Before deploying to air-gapped environments:

```bash
# Build everything
cd web && bun install && bun run build && cd ..
go build -o server ./cmd/server

# Check for external URLs in built assets
grep -r "http://" web/dist/ web/docs/*.js || echo "No http:// found"
grep -r "https://" web/dist/ web/docs/*.js || echo "No https:// found"

# Test with network disabled
# 1. Disconnect network
# 2. Run ./server
# 3. Verify all routes function without external fetches
```

## Troubleshooting

### Scalar UI shows loading skeleton but never loads content

Check browser console for errors. Common issues:
- Spec not accessible at `/docs/openapi.json`
- Using JavaScript API instead of data attributes
- Missing CSS file

### Changes to src/ not reflected

1. Rebuild: `cd web && bun run build`
2. Restart Go server (re-embeds assets)
3. Hard refresh browser (Ctrl+Shift+R)

### go:embed errors during build

Ensure embedded files exist:
- Run `./web/update-scalar.sh` for Scalar assets
- Run `cd web && bun run build` for custom assets
