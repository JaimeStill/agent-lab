# API Documentation

This package serves the interactive API documentation via Scalar UI.

## Files

- `index.html` - HTML template that loads Scalar (data attribute initialization)
- `scalar.js` - Scalar standalone JavaScript bundle (embedded)
- `scalar.css` - Scalar stylesheet (embedded)
- `docs.go` - Handler serving documentation at `/docs` endpoint

## Updating Scalar

Run the update script from project root:

```bash
./web/update-scalar.sh
```

Then test the documentation endpoint:

```bash
go run ./cmd/server
# Navigate to http://localhost:8080/docs
```

## OpenAPI Specification

The OpenAPI spec is generated on server startup and cached to `api/openapi.{env}.json`.
The docs handler serves the spec from memory (no file I/O per request).

**Spec Regeneration:**
- Happens automatically on startup if routes change
- Delete `api/openapi.*.json` to force regeneration
- Environment-specific: `openapi.local.json`, `openapi.prod.json`, etc.

## Scalar Initialization

Scalar uses data attribute initialization (not JavaScript API):

```html
<script
  id="api-reference"
  data-url="/docs/openapi.json"
  src="/docs/scalar.js">
</script>
```

See [web/README.md](../README.md) for full client development architecture.
