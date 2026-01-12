# Session 4e: Performance and Accuracy

**Status**: Completed
**Date**: 2026-01-07

## Summary

Optimized the classify-docs workflow for multi-page PDF processing, reducing execution time from 3m19s (timeout) to 1m40s for a 27-page document while validating ~96% marking detection accuracy.

## Key Changes

### Phase 1-5: Terminology and Prompt Engineering (Pre-session)
- Renamed clarity → legibility (per-marking quality)
- Added new clarity score (per-page overall quality)
- Enhanced system prompts for better marking detection
- Added FilterSuggestion for enhancement recommendations

### Phase 6: Performance Optimizations

**Step 6.1: WriteTimeout Increase**
- Increased HTTP WriteTimeout from 3m to 15m in `cmd/server/server.go`
- Prevents timeout for long-running multi-page workflows

**Step 6.2: Parallel PDF Rendering**
- Added worker pool pattern to `internal/images/repository.go`
- Dynamic worker count: `max(min(runtime.NumCPU(), pageCount), 1)`
- Reduced 27-page init time from 1m28s to 13.4s

**Step 6.3: Dynamic Worker Detection**
- Changed from hardcoded `WorkerCap: 4` to `DefaultParallelConfig()`
- Uses `min(NumCPU*2, 16, itemCount)` from go-agents-orchestration

**Step 6.4: Lifecycle Context for Long-Running Processes**
- Execute endpoint uses `runtime.Lifecycle().Context()` instead of HTTP context
- Workflows survive HTTP disconnection but respect server shutdown
- Consolidated Execute + ExecuteStream into single streaming endpoint

## Architectural Patterns Established

### HTTP-Initiated Long-Running Process Context
```go
func (e *executor) Execute(name string, params map[string]any, token string) (<-chan ExecutionEvent, *Run, error) {
    ctx := e.runtime.Lifecycle().Context()
    // ...
    go e.executeAsync(ctx, run.ID, factory, params, token, streamingObs)
}

func (e *executor) executeAsync(ctx context.Context, ...) {
    execCtx, cancel := context.WithCancel(ctx)
    e.trackRun(runID, cancel)
    // ...
}
```

**When to use**: Processes that should:
- Continue after HTTP client disconnects
- Support explicit cancellation via API
- Respect server shutdown signals

### Parallel Worker Pool Pattern
```go
workerCount := renderWorkerCount(len(pages))
tasks := make(chan int, len(pages))
results := make(chan renderTask, len(pages))

var wg sync.WaitGroup
for range workerCount {
    wg.Go(func() {
        r.renderWorker(ctx, ...)
    })
}
```

## Performance Results

| Stage | Before | After | Improvement |
|-------|--------|-------|-------------|
| init | 1m28s | 13.4s | 6.6x |
| detect | 1m13s | 28.6s | 2.6x |
| enhance | - | 15.5s | N/A |
| classify | timeout | 25.5s | N/A |
| score | timeout | 17s | N/A |
| **Total** | **3m19s (timeout)** | **1m40s** | **2x** |

## Accuracy Validation

Tested against 27-page multi-document PDF:
- **Detection accuracy**: ~96%
- **Final classification**: SECRET//NOFORN (correct)
- **Minor OCR variances**: WNINTEL→WINITEL, CONFIDENTIAL→CONFIDENTIAI

## Files Modified

| File | Change |
|------|--------|
| `cmd/server/server.go` | WriteTimeout: 3m → 15m |
| `internal/images/repository.go` | Parallel rendering with worker pool |
| `internal/workflows/executor.go` | Lifecycle context, single Execute method |
| `internal/workflows/runtime.go` | Added Lifecycle field and getter |
| `internal/workflows/handler.go` | Removed ExecuteStream, Execute uses SSE |
| `internal/workflows/openapi.go` | Consolidated Execute operation |
| `ARCHITECTURE.md` | Documented lifecycle context pattern |
| `tests/internal_workflows/*.go` | Updated for new signatures |

## Validation

- All 20 test packages passing
- go vet clean
- 27-page PDF completes in 1m40s
- Marking detection accuracy ~96%
