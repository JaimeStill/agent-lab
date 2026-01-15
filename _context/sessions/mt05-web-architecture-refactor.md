# mt05 - Web Architecture Refactor

**Type:** Maintenance Session
**Status:** Complete
**Branch:** `mt05-refine-web-client-architecture`

## Summary

Refactored the web client architecture to reduce organizational friction and establish fully isolated, mountable web clients. Each client (`web/app/`, `web/scalar/`) is now completely self-contained with its own source, build output, templates, and handler.

## Key Changes

### New Packages

- **`pkg/routes`** - Extracted reusable route types (`Route`, `Group`, `System` interface)
- **`pkg/web`** - Web infrastructure (`TemplateSet`, `Router`, `PageDef`, static file utilities)

### Directory Restructure

```
web/
├── app/                      # Main app client
│   ├── client/               # TypeScript source
│   ├── dist/                 # Build output (gitignored)
│   ├── public/               # Static assets
│   ├── server/               # Go templates
│   └── app.go                # Handler + Mount()
├── scalar/                   # Scalar OpenAPI UI client
│   ├── app.ts                # Entry point
│   ├── index.html            # Mount HTML
│   ├── scalar.go             # Handler + Mount()
│   └── [scalar.js/css]       # Build output (gitignored)
├── vite.client.ts            # Shared Vite config module
├── vite.config.ts            # Root config (merges clients)
└── [per-client configs]
```

### Mount Pattern

Web clients implement a `Mount(routes.System, prefix)` function that registers:
1. **Exact match** (`/prefix`) - Rewrites path and serves via internal router
2. **Wildcard** (`/prefix/{path...}`) - Strips prefix and serves via internal router

This pattern works with trailing-slash-removal middleware and encapsulates all routing logic within the client.

### Vite Configuration System

- **`ClientConfig` interface** - Defines name, optional input/output/aliases overrides
- **Per-client configs** - `web/app/client.config.ts`, `web/scalar/client.config.ts`
- **Merge function** - Combines all client configs into single Vite config
- **Convention**: `[client]/dist/` for app, root for scalar (customizable via output overrides)

## Patterns Established

### Mountable Web Client Convention

Each `web/[client-name]/` directory:
- Is fully self-contained (source, build output, templates, handler)
- Mounts to `/[client-name]` via `Mount()` function
- Has its own internal router for proper path handling
- Uses absolute asset paths (`/client-name/asset.ext`) for reliability

### Asset Path Resolution

- **App client**: Uses `{{ .BasePath }}` template variable for all URLs
- **Scalar client**: Uses absolute paths (`/scalar/scalar.js`) since it's static HTML

## Files Changed

| Category | Files |
|----------|-------|
| New packages | `pkg/routes/*.go`, `pkg/web/*.go` |
| Moved | `web/web.go` → `web/app/app.go` |
| Renamed | `web/docs/` → `web/scalar/` |
| Restructured | `web/client/`, `web/server/`, `web/public/` → `web/app/` |
| Updated | `cmd/server/routes.go`, `internal/routes/routes.go` |
| Tests | `tests/web/app/`, `tests/web_scalar/`, `tests/pkg_web/` |

## Lessons Learned

1. **http.FileServer path rewriting** - Doesn't work cleanly when mounted via routes.System; use internal router instead
2. **Browser relative URL resolution** - Depends on trailing slash; use absolute paths for reliability
3. **Go embed at compile time** - Rebuild server after Vite build to embed new assets
