# API Documentation

This package serves the interactive API documentation via Scalar UI at `/docs`.

## Files

- `index.html` - HTML template with Scalar mount point and system font overrides
- `docs.go` - Handler serving documentation endpoint

## Architecture

Scalar is bundled via Vite as part of the unified web build pipeline:

1. `web/src/entries/docs.ts` - Programmatic Scalar initialization
2. `bun run build` - Produces `dist/docs.js` and `dist/docs.css`
3. `web/web.go` - Embeds dist/ and serves at `/static/`
4. `index.html` - References `/static/docs.js` and `/static/docs.css`

## Updating Scalar

Update the dependency in `web/package.json`:

```bash
cd web
bun update @scalar/api-reference
bun run build
```

## Air-Gap Compatibility

External font loading is disabled for air-gap deployments:

- `docs.ts` sets `withDefaultFonts: false`
- `index.html` overrides `--scalar-font` and `--scalar-font-code` with system font stacks

## OpenAPI Specification

The OpenAPI spec is generated on server startup and cached to `api/openapi.{env}.json`.
The `/api/openapi.json` endpoint serves the spec from memory.

**Spec Regeneration:**
- Happens automatically on startup if routes change
- Delete `api/openapi.*.json` to force regeneration
- Environment-specific: `openapi.local.json`, `openapi.prod.json`, etc.

## Scalar Initialization

Scalar uses programmatic initialization via `createApiReference()`:

```typescript
import { createApiReference } from '@scalar/api-reference'
import '@scalar/api-reference/style.css'

createApiReference('#api-reference', {
  url: '/api/openapi.json',
  withDefaultFonts: false,
})
```

See [web/README.md](../README.md) for full client development architecture.
