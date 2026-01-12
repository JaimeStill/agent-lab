# Session 4a: Workflow Profiles Infrastructure

## Summary

Implemented the profiles domain for storing workflow stage configurations and migrated existing workflows to a top-level `workflows/` directory with profile separation. This enables A/B testing and prompt iteration without code changes.

## What Was Implemented

### Database
- Migration `000007_profiles.up.sql` creating `profiles` and `profile_stages` tables
- Composite primary key `(profile_id, stage_name)` for profile_stages
- Cascade delete for stages when profile is deleted

### Profiles Domain (`internal/profiles/`)
- Full CRUD for profiles with pagination and filtering by `workflow_name`
- Stage management with save semantics (`SetStage`)
- `ProfileWithStages` type with `Stage(name)` method for O(1)-ish lookup
- `NewProfileWithStages(stages...)` variadic constructor

### Workflow Profile Helpers (`internal/workflows/profile.go`)
- `ExtractAgentParams(state, stage)` - extracts agent ID from stage config or params
- `LoadProfile(ctx, runtime, params, default)` - resolves profile from DB or returns default

### Workflow Migration
- Moved workflows from `internal/workflows/samples/` to `workflows/`
- Each workflow has `profile.go` with `DefaultProfile()` returning hardcoded defaults
- Created `workflows/init.go` for single-import aggregation
- Updated `cmd/server/server.go` to import `_ "github.com/JaimeStill/agent-lab/workflows"`

### Runtime Integration
- Added `profiles.System` to workflow `Runtime`
- Updated `Domain` and routes registration

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| No `is_default` column | Profiles are explicitly selected or hardcoded defaults used |
| Composite PK for stages | `(profile_id, stage_name)` - ID never used, simplifies API |
| Full replacement strategy | DB profile completely replaces hardcoded config |
| Stage AgentID precedence | If stage has AgentID, it overrides params agent_id |
| Variadic constructor | `NewProfileWithStages(stages...)` cleaner than slice literal |

## Patterns Established

### Profile Resolution
```go
profile, err := workflows.LoadProfile(ctx, runtime, params, DefaultProfile())
// If profile_id in params → load from DB
// Otherwise → use hardcoded default
```

### Stage Configuration Access
```go
stage := profile.Stage("summarize")
agentID, token, err := workflows.ExtractAgentParams(s, stage)
opts := map[string]any{"system_prompt": *stage.SystemPrompt}
```

### Default Profile Definition
```go
func DefaultProfile() *profiles.ProfileWithStages {
    prompt := "System prompt here"
    return profiles.NewProfileWithStages(
        profiles.ProfileStage{StageName: "stage", SystemPrompt: &prompt},
    )
}
```

## Files Changed

### New Files
- `cmd/migrate/migrations/000007_profiles.up.sql`
- `cmd/migrate/migrations/000007_profiles.down.sql`
- `internal/profiles/profile.go`
- `internal/profiles/system.go`
- `internal/profiles/errors.go`
- `internal/profiles/mapping.go`
- `internal/profiles/repository.go`
- `internal/profiles/handler.go`
- `internal/profiles/openapi.go`
- `internal/workflows/profile.go`
- `workflows/init.go`
- `workflows/summarize/profile.go`
- `workflows/summarize/summarize.go`
- `workflows/reasoning/profile.go`
- `workflows/reasoning/reasoning.go`
- `tests/internal_profiles/errors_test.go`
- `tests/internal_profiles/mapping_test.go`
- `tests/internal_profiles/profile_test.go`

### Modified Files
- `internal/workflows/runtime.go`
- `cmd/server/domain.go`
- `cmd/server/routes.go`
- `cmd/server/server.go`
- `tests/internal_workflows/executor_test.go`
- `tests/internal_workflows/runtime_test.go`

### Deleted
- `internal/workflows/samples/` directory

## Validation Results

All endpoints tested and working:
- Profile CRUD (List, Create, Find, Update, Delete)
- Stage management (SetStage, DeleteStage)
- Filtering by `workflow_name`
- 404 handling for non-existent resources

Both profile scenarios validated:
1. Stage with only `system_prompt` - agent_id from params ✓
2. Stage with `agent_id` configured - no agent_id needed in params ✓

All tests passing (including new profiles domain tests).
