# Session 05b: Service Infrastructure - Plan Outline

## Objective

Establish service patterns for domain data access with API client, signal-based state, and SSE streaming utilities.

## Scope

Per PROJECT.md and milestone guide:
- API client utility (fetch wrapper with Result type)
- Service pattern template (context + interface + factory)
- Signal-based state management patterns
- SSE client utility for streaming

## Approach

### Phase 1: Shared Infrastructure

Create `web/app/client/shared/` with foundational utilities:

1. **`api.ts`** - Type-safe fetch wrapper
   - `Result<T>` type for success/error handling
   - JSON request/response handling
   - Base path resolution from `<base href>`
   - Query string builder for pagination

2. **`types.ts`** - Common pagination types
   - `PageResult<T>` matching Go backend
   - `PageRequest` for query parameters
   - Helper to build query strings

3. **`sse.ts`** - EventSource wrapper
   - JSON message parsing
   - `[DONE]` marker detection
   - Cleanup function return
   - Error callback support

### Phase 2: Provider Service (Pattern Validation)

Implement one complete domain service to validate patterns:

1. **`providers/types.ts`** - Domain types matching Go
2. **`providers/service.ts`** - Full service pattern:
   - Interface with Signal state
   - Context creation
   - Factory function
   - CRUD operations

### Phase 3: Validation View

Create minimal view to demonstrate service consumption:

1. **`providers/views/providers-view.ts`** - Uses:
   - `@provide` to inject service
   - `SignalWatcher` for reactivity
   - Fetches and displays provider list

2. **Route registration** in `routes.ts`

## File Structure

```
web/app/client/
├── shared/
│   ├── api.ts          # Fetch wrapper
│   ├── types.ts        # Pagination types
│   └── sse.ts          # SSE client
└── providers/
    ├── types.ts        # Provider domain types
    ├── service.ts      # Context + interface + factory
    └── views/
        ├── providers-view.ts   # Validation view
        └── providers-view.css
```

## Validation Criteria

1. **Build**: `cd web && bun run build` succeeds
2. **Types**: No TypeScript errors
3. **Integration**: Navigate to `/app/providers`, see list from API
4. **Reactivity**: Loading state displays, then data renders

## Dependencies

- Session 05a complete ✅
- Backend `/api/providers` endpoint exists ✅
- Lit dependencies installed ✅

## Questions to Clarify

None identified - approach aligns with milestone guide patterns and PROJECT.md scope.

---

## Context Snapshot (for session resumption)

### Session State

- **Branch**: `05b-service-infrastructure`
- **Development methodology step**: Step 5 (Developer Execution) - implementation guide is complete, developer is implementing
- **Implementation guide**: `_context/05b-service-infrastructure.md`

### Completed This Session

1. **Milestone guide cleanup** (`_context/milestones/m05-workflow-lab-interface.md`):
   - Updated session overview table: dropped "Design System" session, renumbered 05b as Service Infrastructure
   - Changed component prefix from `al-` to `lab-` in all examples
   - Updated CSS layers from `reset, theme, layout, components` to `tokens, reset, theme`
   - Removed `layout.css` from directory structure
   - Removed obsolete "Session 05a Details" and "Existing Infrastructure" sections

2. **Implementation guide created** (`_context/05b-service-infrastructure.md`) with phases:
   - Phase 1: Core infrastructure (pagination.ts, streaming.ts, api.ts)
   - Phase 2: Provider service (types.ts, service.ts)
   - Phase 3: Validation view (providers-view.ts/css)
   - Phase 4: Route registration

### Design Decisions Made

1. **Directory naming**: Developer changed `shared/` to `core/` and organizes files by concern (`pagination.ts`, `streaming.ts`) instead of a monolithic `types.ts`

2. **API client pattern**: Evolved from method-specific wrapper object (`api.get()`, `api.post()`) to a single `request()` function using native `RequestInit`:
   - `request<T>(path, init?, parse?)` - minimal wrapper, caller controls everything
   - `stream(path, init, options)` - separate function for SSE via fetch body reader
   - No EventSource (doesn't support POST) - uses `ReadableStreamDefaultReader` with SSE line parsing
   - Base path is simple `const BASE = '/api'` (not parsed from `<base href>`)
   - FormData works naturally (browser sets Content-Type with boundary automatically)
   - Response parsing controlled by caller callback (defaults to `res.json()`)

3. **SSE streaming**: Uses fetch body reader (not EventSource) because agent streaming endpoints require POST with request bodies. Includes `parseSSE()`, `handleStreamResponse()`, `handleStreamError()` helpers.

### Developer Progress

The developer was partway through Phase 1 (Step 1.2 - api.ts) when the design discussion about `request()` vs method wrappers occurred. The implementation guide has been updated to reflect the final agreed design. Developer should resume implementation from Phase 1 using the updated guide.

### Key Files to Reference

| File | Purpose |
|------|---------|
| `_context/05b-service-infrastructure.md` | Implementation guide (source of truth) |
| `_context/milestones/m05-workflow-lab-interface.md` | Milestone architecture |
| `web/app/client/core/` | Where core infrastructure goes |
| `web/app/client/providers/` | Where provider service + validation view go |
| `/home/jaime/code/go-lit/web/app/client/` | Reference implementation from POC |
