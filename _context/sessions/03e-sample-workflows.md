# Session 3e: Sample Workflows - Summary

## Completed

This session enabled workflows to execute agent capabilities and validated the M3 workflow infrastructure with live agent integration.

## Key Deliverables

### Agent System Capabilities

Expanded `agents.System` interface with capability methods:
- `Chat` / `ChatStream` - Text completion with optional streaming
- `Vision` / `VisionStream` - Image analysis with optional streaming
- `Tools` - Function calling / tool use
- `Embed` - Text embeddings

All methods support:
- `opts["system_prompt"]` - Override stored system prompt
- `token` - Override stored API token

Implementation moved from Handler to System/repository, enabling workflows to access via `runtime.Agents()`.

### Sample Workflows

Created `internal/workflows/samples/` with two workflows demonstrating live agent integration:

**summarize** - Single-node workflow
- Takes `agent_id` and `text` params
- Optional `system_prompt` override
- Returns `summary` in result

**reasoning** - Multi-node workflow (analyze → reason → conclude)
- Takes `agent_id` and `problem` params
- Optional per-node system prompt overrides
- Demonstrates state flow between LLM calls

### Bug Fix: Query Builder nil Pointer Handling

Fixed `WhereEquals` and `WhereNullable` to properly handle nil pointers passed as `any`. Added `isNil()` helper using reflection to detect nil interface values with non-nil types.

### Nullable JSON Scan Fix

Fixed `scanRun` and `scanStage` to handle NULL JSONB columns by scanning into `*[]byte` intermediaries.

### DeleteRun Endpoint

Added `DELETE /api/workflows/runs/{id}` endpoint to remove workflow runs and their related data (stages, decisions, checkpoints cascade via foreign keys).

## Files Changed

| File | Changes |
|------|---------|
| `internal/agents/system.go` | Added capability methods to interface |
| `internal/agents/repository.go` | Implemented capabilities + constructAgent |
| `internal/agents/handler.go` | Delegated to system for capabilities |
| `internal/workflows/system.go` | Added DeleteRun to interface |
| `internal/workflows/repository.go` | Implemented DeleteRun |
| `internal/workflows/executor.go` | Added DeleteRun delegation |
| `internal/workflows/handler.go` | Added DeleteRun handler and route |
| `internal/workflows/openapi.go` | Added DeleteRun operation spec |
| `internal/workflows/samples/summarize.go` | New single-node workflow |
| `internal/workflows/samples/reasoning.go` | New multi-node workflow |
| `internal/workflows/mapping.go` | Fixed nullable JSON scanning |
| `pkg/query/builder.go` | Added isNil() for nil pointer detection |
| `cmd/server/server.go` | Added samples import |
| `README.md` | Added Sample Workflows section |

## Tests Added

- `TestBuilder_WhereEquals_NilPointerIgnored`
- `TestBuilder_WhereNullable_NilPointerIsNull`
- `TestBuilder_WhereNullable_NonNilValue`
- `TestSampleWorkflows_Registered`
- `TestSampleWorkflows_Info`

## Validation

All validation criteria passed:
- Both workflows registered and listed via API
- Summarize workflow executes with live LLM call
- Reasoning workflow executes 3 stages with LLM calls
- Runs table shows execution records
- Stages endpoint shows node execution history

## Patterns Established

1. **System prompt override** - Workflows provide context-specific prompts via `opts["system_prompt"]`, with stored config as fallback

2. **Capability delegation** - Handler delegates to System for execution logic, enabling workflows to access same capabilities

3. **Nullable JSON scanning** - Use `*[]byte` intermediary for nullable JSONB columns

4. **Nil interface detection** - Use reflection to check for nil pointers passed as `any`
