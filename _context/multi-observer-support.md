# Maintenance: Native MultiObserver Support in go-agents-orchestration

## Problem

When executing workflows, it's common to need multiple observers:
- **PostgresObserver** - Persist events to database for historical analysis
- **StreamingObserver** - Stream events to SSE clients in real-time
- **SlogObserver** - Log events for debugging (already in library)

Currently, agent-lab implements a `MultiObserver` shim to broadcast events to multiple observers:

```go
type MultiObserver struct {
    observers []observability.Observer
}

func NewMultiObserver(observers ...observability.Observer) *MultiObserver {
    return &MultiObserver{observers: observers}
}

func (m *MultiObserver) OnEvent(ctx context.Context, event observability.Event) {
    for _, obs := range m.observers {
        obs.OnEvent(ctx, event)
    }
}
```

This is a fundamental pattern that should be supported natively by go-agents-orchestration.

## Proposed Solution

Add `MultiObserver` to `pkg/observability/` in go-agents-orchestration:

**File:** `pkg/observability/multi.go`

```go
package observability

import "context"

type MultiObserver struct {
    observers []Observer
}

func NewMultiObserver(observers ...Observer) *MultiObserver {
    return &MultiObserver{observers: observers}
}

func (m *MultiObserver) OnEvent(ctx context.Context, event Event) {
    for _, obs := range m.observers {
        obs.OnEvent(ctx, event)
    }
}

func (m *MultiObserver) Add(obs Observer) {
    m.observers = append(m.observers, obs)
}
```

## Migration Path

1. Release go-agents-orchestration patch (e.g., v0.3.1) with `MultiObserver`
2. Update agent-lab to use `observability.NewMultiObserver`
3. Remove shim from `internal/workflows/streaming.go`

## Scope

- **Type:** Maintenance session
- **Repository:** go-agents-orchestration
- **Release:** Patch (backward compatible addition)
- **Effort:** Small - straightforward addition

## Related

- Session 3d introduced the shim in `internal/workflows/streaming.go`
- Used by `executor.go` in `ExecuteStream` to combine PostgresObserver + StreamingObserver
