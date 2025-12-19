# Maintenance Session 03: Native MultiObserver Support

**Type:** Maintenance Session
**Repositories:** go-agents-orchestration, agent-lab
**Release:** go-agents-orchestration v0.3.1 (patch)

## Problem

When executing workflows, it's common to need multiple observers (e.g., PostgresObserver for persistence + StreamingObserver for SSE). Currently, agent-lab implements a `MultiObserver` shim to broadcast events to multiple observers. This is a fundamental pattern that should be supported natively by go-agents-orchestration.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Thread-safe? | No | Constructor-time composition, no runtime modification |
| Filter nil observers? | Yes | Defensive, zero runtime cost |
| Registry registration? | No | Composition utility, not configuration-driven |
| Include Add()? | No | Simpler, avoids race conditions, matches current usage |

---

## Phase 1: go-agents-orchestration

### Step 1.1: Create MultiObserver

**Create file:** `pkg/observability/multi.go`

```go
package observability

import "context"

type MultiObserver struct {
	observers []Observer
}

func NewMultiObserver(observers ...Observer) *MultiObserver {
	filtered := make([]Observer, 0, len(observers))
	for _, obs := range observers {
		if obs != nil {
			filtered = append(filtered, obs)
		}
	}
	return &MultiObserver{observers: filtered}
}

func (m *MultiObserver) OnEvent(ctx context.Context, event Event) {
	for _, obs := range m.observers {
		obs.OnEvent(ctx, event)
	}
}
```

### Step 1.2: Create Tests

**Create file:** `tests/observability/multi_test.go`

```go
package observability_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/JaimeStill/go-agents-orchestration/pkg/observability"
)

type captureObserver struct {
	mu     sync.Mutex
	events []observability.Event
}

func (o *captureObserver) OnEvent(ctx context.Context, event observability.Event) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, event)
}

func (o *captureObserver) getEvents() []observability.Event {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.events
}

func TestMultiObserver_BroadcastsToAllObservers(t *testing.T) {
	obs1 := &captureObserver{}
	obs2 := &captureObserver{}
	obs3 := &captureObserver{}

	multi := observability.NewMultiObserver(obs1, obs2, obs3)

	event := observability.Event{
		Type:      observability.EventNodeStart,
		Timestamp: time.Now(),
		Source:    "test",
		Data:      map[string]any{"key": "value"},
	}

	multi.OnEvent(context.Background(), event)

	observers := []*captureObserver{obs1, obs2, obs3}
	for i, obs := range observers {
		events := obs.getEvents()
		if len(events) != 1 {
			t.Errorf("Observer %d: got %d events, want 1", i, len(events))
		}
		if events[0].Type != observability.EventNodeStart {
			t.Errorf("Observer %d: got type %v, want %v", i, events[0].Type, observability.EventNodeStart)
		}
	}
}

func TestMultiObserver_EmptyObservers(t *testing.T) {
	multi := observability.NewMultiObserver()

	event := observability.Event{
		Type:      observability.EventNodeStart,
		Timestamp: time.Now(),
		Source:    "test",
	}

	multi.OnEvent(context.Background(), event)
}

func TestMultiObserver_FiltersNilObservers(t *testing.T) {
	obs1 := &captureObserver{}
	obs2 := &captureObserver{}

	multi := observability.NewMultiObserver(obs1, nil, obs2, nil)

	event := observability.Event{
		Type:      observability.EventNodeComplete,
		Timestamp: time.Now(),
		Source:    "test",
	}

	multi.OnEvent(context.Background(), event)

	if len(obs1.getEvents()) != 1 {
		t.Errorf("obs1: got %d events, want 1", len(obs1.getEvents()))
	}
	if len(obs2.getEvents()) != 1 {
		t.Errorf("obs2: got %d events, want 1", len(obs2.getEvents()))
	}
}

func TestMultiObserver_SingleObserver(t *testing.T) {
	obs := &captureObserver{}
	multi := observability.NewMultiObserver(obs)

	event := observability.Event{
		Type:      observability.EventGraphStart,
		Timestamp: time.Now(),
		Source:    "graph",
		Data:      map[string]any{"name": "test-graph"},
	}

	multi.OnEvent(context.Background(), event)

	events := obs.getEvents()
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Data["name"] != "test-graph" {
		t.Errorf("got name %v, want test-graph", events[0].Data["name"])
	}
}

func TestMultiObserver_PreservesEventData(t *testing.T) {
	obs := &captureObserver{}
	multi := observability.NewMultiObserver(obs)

	originalData := map[string]any{
		"string": "value",
		"number": 42,
		"nested": map[string]any{"inner": "data"},
	}

	event := observability.Event{
		Type:      observability.EventStateSet,
		Timestamp: time.Now(),
		Source:    "state",
		Data:      originalData,
	}

	multi.OnEvent(context.Background(), event)

	events := obs.getEvents()
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}

	receivedData := events[0].Data
	if receivedData["string"] != "value" {
		t.Errorf("string: got %v, want value", receivedData["string"])
	}
	if receivedData["number"] != 42 {
		t.Errorf("number: got %v, want 42", receivedData["number"])
	}
}

func TestMultiObserver_PropagatesContext(t *testing.T) {
	type ctxKey string
	key := ctxKey("test-key")

	var receivedCtx context.Context
	obs := &captureObserver{}

	wrapper := &contextCapture{
		inner:      obs,
		capturedFn: func(ctx context.Context) { receivedCtx = ctx },
	}

	multi := observability.NewMultiObserver(wrapper)

	ctx := context.WithValue(context.Background(), key, "test-value")
	event := observability.Event{
		Type:      observability.EventNodeStart,
		Timestamp: time.Now(),
		Source:    "test",
	}

	multi.OnEvent(ctx, event)

	if receivedCtx == nil {
		t.Fatal("context was not propagated")
	}
	if receivedCtx.Value(key) != "test-value" {
		t.Errorf("context value: got %v, want test-value", receivedCtx.Value(key))
	}
}

type contextCapture struct {
	inner      observability.Observer
	capturedFn func(ctx context.Context)
}

func (c *contextCapture) OnEvent(ctx context.Context, event observability.Event) {
	c.capturedFn(ctx)
	c.inner.OnEvent(ctx, event)
}

func TestMultiObserver_ConcurrentEvents(t *testing.T) {
	obs := &captureObserver{}
	multi := observability.NewMultiObserver(obs)

	const numGoroutines = 10
	const eventsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				event := observability.Event{
					Type:      observability.EventNodeStart,
					Timestamp: time.Now(),
					Source:    "concurrent-test",
					Data:      map[string]any{"goroutine": id, "event": j},
				}
				multi.OnEvent(context.Background(), event)
			}
		}(i)
	}

	wg.Wait()

	events := obs.getEvents()
	expected := numGoroutines * eventsPerGoroutine
	if len(events) != expected {
		t.Errorf("got %d events, want %d", len(events), expected)
	}
}
```

### Step 1.3: Validate

```bash
cd /home/jaime/code/go-agents-orchestration
go vet ./...
go test ./tests/...
```

### Step 1.4: Commit and Release

```bash
git add -A
git commit -m "feat(observability): add MultiObserver for event broadcasting"
git tag v0.3.1
git push origin main --tags
```

---

## Phase 2: agent-lab Migration

### Step 2.1: Update Dependency

```bash
cd /home/jaime/code/agent-lab
go get github.com/JaimeStill/go-agents-orchestration@v0.3.1
go mod tidy
```

### Step 2.2: Update executor.go

**File:** `internal/workflows/executor.go`

**Add import** for observability package:

```go
import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/go-agents-orchestration/pkg/config"
	"github.com/JaimeStill/go-agents-orchestration/pkg/observability"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
	"github.com/google/uuid"
)
```

**Update line 227** from:
```go
multiObs := NewMultiObserver(postgresObs, streamingObs)
```

To:
```go
multiObs := observability.NewMultiObserver(postgresObs, streamingObs)
```

### Step 2.3: Remove Shim from streaming.go

**File:** `internal/workflows/streaming.go`

Remove lines 12-28 (MultiObserver type, NewMultiObserver, and OnEvent method).

Keep StreamingObserver (lines 30-180) - this is domain-specific to agent-lab.

### Step 2.4: Update Tests

**File:** `tests/internal_workflows/streaming_test.go`

Remove the following test functions (now covered by library tests):
- `TestMultiObserver_BroadcastsToAll`
- Any other MultiObserver-specific tests

Keep all StreamingObserver tests.

### Step 2.5: Validate

```bash
go vet ./...
go test ./tests/...
```

---

## Validation Checklist

### go-agents-orchestration
- [ ] `pkg/observability/multi.go` created
- [ ] `tests/observability/multi_test.go` created
- [ ] `go vet ./...` passes
- [ ] `go test ./tests/...` passes
- [ ] v0.3.1 tagged and pushed

### agent-lab
- [ ] go.mod updated to v0.3.1
- [ ] executor.go uses `observability.NewMultiObserver`
- [ ] MultiObserver shim removed from streaming.go
- [ ] MultiObserver tests removed from streaming_test.go
- [ ] `go vet ./...` passes
- [ ] `go test ./tests/...` passes
