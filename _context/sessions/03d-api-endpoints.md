# Session 3d: API Endpoints

**Status**: Completed (2025-12-19)

## Objective

Implement HTTP handlers and OpenAPI specifications for workflow execution and inspection.

## Implemented

### HTTP Endpoints

| Method | Endpoint | Handler | Description |
|--------|----------|---------|-------------|
| GET | `/api/workflows` | ListWorkflows | List registered workflows |
| POST | `/api/workflows/{name}/execute` | Execute | Execute workflow (sync) |
| POST | `/api/workflows/{name}/execute/stream` | ExecuteStream | Execute with SSE progress |
| GET | `/api/workflows/runs` | ListRuns | List runs with filters |
| GET | `/api/workflows/runs/{id}` | FindRun | Get run details |
| GET | `/api/workflows/runs/{id}/stages` | GetStages | Get execution stages |
| GET | `/api/workflows/runs/{id}/decisions` | GetDecisions | Get routing decisions |
| POST | `/api/workflows/runs/{id}/cancel` | Cancel | Cancel running workflow |
| POST | `/api/workflows/runs/{id}/resume` | Resume | Resume from checkpoint |

### Streaming Infrastructure

- **StreamingObserver**: Converts graph events to ExecutionEvents, sends to buffered channel
- **MultiObserver**: Broadcasts events to multiple observers (PostgresObserver + StreamingObserver)
- **ExecutionEvent types**: stage.start, stage.complete, decision, error, complete
- **SSE format**: `event: {type}\ndata: {json}\n\n` with real-time flush

### New Files

| File | Purpose |
|------|---------|
| `pkg/decode/decode.go` | Generic map[string]any to struct decoder via JSON roundtrip |
| `internal/workflows/streaming.go` | StreamingObserver, MultiObserver implementations |
| `internal/workflows/handler.go` | HTTP handlers for all workflow endpoints |
| `internal/workflows/openapi.go` | OpenAPI operations and schemas |

### Modified Files

| File | Changes |
|------|---------|
| `internal/workflows/run.go` | Added ExecutionEvent types and event data structs |
| `internal/workflows/observer.go` | Refactored to use decode.FromMap pattern |
| `internal/workflows/executor.go` | Added ExecuteStream method |
| `internal/workflows/system.go` | Added ExecuteStream to interface |
| `internal/routes/group.go` | Added Children field for nested route groups |
| `internal/routes/routes.go` | Updated registerGroup for prefix concatenation |
| `cmd/server/openapi.go` | Updated processGroup for prefix concatenation |
| `cmd/server/routes.go` | Added workflow handler registration |

## Architectural Additions

### Route Children Pattern

Route groups now support hierarchical nesting with prefix concatenation:

```go
routes.Group{
    Prefix: "/api/workflows",
    Routes: []routes.Route{...},
    Children: []routes.Group{
        {
            Prefix: "/runs",  // Results in /api/workflows/runs
            Routes: []routes.Route{...},
        },
    },
}
```

### decode.FromMap Pattern

Generic helper for converting observability event data to typed structs:

```go
data, err := decode.FromMap[NodeStartData](event.Data)
```

Uses JSON roundtrip (marshal map to JSON, unmarshal to struct) - avoids reflection-based libraries.

### Go File Structure Convention

Established ordering convention for Go files:
1. Package declaration and imports
2. Constants (directly beneath imports)
3. Global variables (directly beneath constants)
4. Pure types (types without methods)
5. Types with methods
6. Functions

Added to CLAUDE.md (design instruction) and ARCHITECTURE.md (architectural principle #7).

### SSE Streaming Architecture

```
Handler.ExecuteStream()
        │
        ▼
System.ExecuteStream() → creates StreamingObserver
        │
        ▼
executeStreamAsync() [goroutine]
        │
        ├── PostgresObserver (persists)
        │
        └── StreamingObserver (streams)
                │
                ▼
        chan ExecutionEvent
                │
                ▼
        Handler SSE loop
```

## Testing

### New Test Files

| File | Tests |
|------|-------|
| `tests/pkg_decode/decode_test.go` | 6 tests for FromMap function |
| `tests/internal_workflows/streaming_test.go` | 12 tests for StreamingObserver, MultiObserver |
| `tests/internal_workflows/handler_test.go` | 3 tests for Handler construction and routes |
| `tests/internal_workflows/openapi_test.go` | 5 tests for OpenAPI operations and schemas |

### Updated Test Files

| File | Tests Added |
|------|-------------|
| `tests/internal_workflows/run_test.go` | 6 tests for ExecutionEvent types |
| `tests/internal_routes/routes_test.go` | 2 tests for Children group routing |

## Validation

- All 18 test packages pass
- `go vet ./...` clean
- SSE streaming architecture validated

## Maintenance Notes

Documented future maintenance task to add native MultiObserver support to go-agents-orchestration:
- Location: `_context/multi-observer-support.md`
- Scope: Patch release to library, then migrate shim from agent-lab

## Key Decisions

1. **Children over SubGroups**: Cleaner naming for nested route groups
2. **JSON roundtrip for decoding**: Avoids archived/unmaintained mapstructure library
3. **Handler function pattern**: Each event type has dedicated handler returning `*ExecutionEvent`
4. **Relative prefixes in Children**: `/runs` not `/api/workflows/runs`
5. **X-Run-ID header**: SSE response includes run ID for client tracking
