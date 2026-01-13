# Session 05a: Web Infrastructure

**Status**: Completed
**Date**: 2026-01-12

## Summary

Established Vite + Bun + TypeScript build pipeline with Go embedding for Milestone 5. Integrated Scalar API docs into unified Vite pipeline, replacing the manual `update-scalar.sh` approach with automated dependency management.

## Key Changes

### Phase 1: Build Infrastructure

- Created `web/package.json` with Bun, Vite, TypeScript, and @scalar/api-reference dependencies
- Created `web/vite.config.ts` with library mode, multiple entry points, and path aliases
- Created `web/tsconfig.json` with ES2024 target and bundler module resolution
- Created `web/.gitignore` for node_modules/ and dist/

### Phase 2: Source Structure

- Created `web/src/core/index.ts` placeholder for foundation layer
- Created `web/src/design/styles.css` with CSS custom properties (design tokens)
- Created `web/src/components/index.ts` placeholder for web components
- Created `web/src/entries/shared.ts` entry point importing design tokens and exports
- Created `web/src/entries/docs.ts` with Scalar programmatic initialization
- Created `web/templates/layouts/app.html` base template

### Phase 3: Go Integration

- Created `web/web.go` with embed.FS for dist/ and templates/, Static() and Templates() functions
- Updated `web/docs/index.html` to reference bundled assets at /static/
- Updated `web/docs/docs.go` to serve only index.html (removed JS/CSS embeds)
- Updated `cmd/server/routes.go` to mount static handler at /static/

### Phase 4: Cleanup

- Removed `web/update-scalar.sh` (replaced by Vite pipeline)
- Removed `web/docs/scalar.js` and `web/docs/scalar.css` (now built by Vite)

## Architectural Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Scalar integration | Vite ESM bundle | Automates dependency management, single build pipeline |
| dist/ commit | .gitignore | Standard practice, CI builds from source |
| Static asset path | /static/ | Clear separation from API routes |
| Design directory | design/ (not tokens/) | Better reflects design system scope |
| Scalar initialization | Programmatic createApiReference() | Cleaner than data attributes, explicit configuration |
| Scalar fonts | System fonts via withDefaultFonts: false | Air-gap compatible, no external CDN requests |

## Key Technical Decisions

### Vite Configuration

```typescript
define: {
  'process.env.NODE_ENV': JSON.stringify('production'),
}
```

Required because Scalar's Vue-based ESM bundle expects `process.env.NODE_ENV` to be replaced at build time.

### Scalar Initialization Pattern

```typescript
import { createApiReference } from '@scalar/api-reference'
import '@scalar/api-reference/style.css'

createApiReference('#api-reference', {
  url: '/api/openapi.json',
  withDefaultFonts: false,
})
```

Direct programmatic initialization rather than window globals or data attributes. `withDefaultFonts: false` disables external font loading.

### System Fonts for Air-Gap Compatibility

```css
:root {
  --scalar-font: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
  --scalar-font-code: ui-monospace, 'Cascadia Code', 'SF Mono', Menlo, Monaco, Consolas, monospace;
}
```

Inline CSS in `web/docs/index.html` overrides Scalar's font variables with system font stacks, eliminating external requests to fonts.scalar.com.

### Go Embedding Pattern

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

Wraps FileServer in HandlerFunc for compatibility with routes system.

## Files Created

| File | Purpose |
|------|---------|
| `web/package.json` | Bun dependencies and scripts |
| `web/vite.config.ts` | Vite build configuration |
| `web/tsconfig.json` | TypeScript configuration |
| `web/.gitignore` | Ignore dist/, node_modules/ |
| `web/src/core/index.ts` | Core exports placeholder |
| `web/src/design/styles.css` | CSS design tokens |
| `web/src/components/index.ts` | Components exports placeholder |
| `web/src/entries/shared.ts` | Shared components entry point |
| `web/src/entries/docs.ts` | Scalar API docs entry point |
| `web/templates/layouts/app.html` | Base layout template |
| `web/web.go` | Go embedding and static handler |
| `tests/web/static_test.go` | Static handler tests |

## Files Modified

| File | Change |
|------|--------|
| `web/docs/index.html` | Updated asset paths to /static/ |
| `web/docs/docs.go` | Removed JS/CSS embeds, kept HTML only |
| `cmd/server/routes.go` | Added static handler registration |
| `tests/web_docs/docs_test.go` | Removed JS/CSS route tests |
| `web/README.md` | Comprehensive rewrite for new architecture |
| `_context/milestones/m05-workflow-lab-interface.md` | tokens/ → design/ rename |

## Files Removed

| File | Reason |
|------|--------|
| `web/update-scalar.sh` | Replaced by Vite pipeline |
| `web/docs/scalar.js` | Now built by Vite |
| `web/docs/scalar.css` | Now built by Vite |

## Validation

- `bun install` completes without errors
- `bun run build` produces dist/ with shared.js, docs.js, docs.css
- `go vet ./...` passes
- `go test ./tests/...` all tests passing
- `/docs` renders Scalar UI correctly
- `/static/shared.js` returns content
- `/static/docs.js` returns Scalar bundle
- OpenAPI "Try it" functionality works
- No external requests (all assets served from localhost, no fonts.scalar.com)
