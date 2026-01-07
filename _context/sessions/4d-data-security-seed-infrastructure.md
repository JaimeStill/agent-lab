# Session 4d: Data Security and Seed Infrastructure

## Summary

This session addressed a security issue discovered in Session 4c: the cloud provider token was being captured in workflow state and stored in database tables, checkpoint serialization, and SSE events. The session also established infrastructure for profile experimentation via a new `cmd/seed` sub-command.

## What Was Implemented

### Phase 1: go-agents-orchestration Secrets Support

Added a dedicated `Secrets` field to `state.State` that is explicitly excluded from persistence and observability.

**File**: `pkg/state/state.go`
- Added `Secrets map[string]any` field with `json:"-"` tag
- Updated `New()` to initialize Secrets map
- Updated `Clone()` to clone Secrets map
- Added `GetSecret(key)`, `SetSecret(key, value)`, `DeleteSecret(key)` methods
- Added comprehensive godoc documentation for all secret methods

**File**: `ARCHITECTURE.md`
- Updated State struct documentation to show Secrets field
- Added "Secret Operations" section documenting the API

**File**: `tests/state/state_test.go`
- Added 10 new tests for secrets functionality

**Release**: v0.3.2

### Phase 2: agent-lab Integration

Updated agent-lab to use the new Secrets API.

**File**: `internal/workflows/executor.go`
- Changed `initialState.Set("token", token)` to `initialState.SetSecret("token", token)` in both `Execute()` and `executeStreamAsync()`

**File**: `internal/workflows/profile.go`
- Updated `ExtractAgentParams()` to use `s.GetSecret("token")` instead of `s.Get("token")`
- Updated godoc to reflect secrets usage

**File**: `go.mod`
- Updated go-agents-orchestration dependency to v0.3.2

### Phase 3: cmd/seed Infrastructure

Created new CLI tool for seeding data.

**Files Created**:
- `cmd/seed/main.go` - CLI entry point with flags (-dsn, -all, -profiles, -file, -list)
- `cmd/seed/seeder.go` - Seeder interface, registry, and transaction wrapper
- `cmd/seed/seeds.go` - Shared `embed.FS` for seed files
- `cmd/seed/profiles.go` - ProfileSeeder implementation using `profiles.ProfileWithStages`
- `cmd/seed/seeds/classify_profiles.json` - Default seed data with two profiles

## Key Decisions

### Secrets Field Design
- `json:"-"` tag excludes from checkpoint serialization
- Observer snapshots use `maps.Clone(state.Data)` - Secrets naturally excluded
- Explicit API (`SetSecret` vs `Set`) makes intent clear

### Using Existing Types for Seeding
- Used `profiles.ProfileWithStages` directly instead of defining separate seed types
- Auto-generated fields get zero values during JSON unmarshal, new ones generated during save

### Data Mutation Naming Convention
Standardized terminology across codebase:

| Verb | Semantics |
|------|-----------|
| `Create` | Insert new record (fails if exists) |
| `Update` | Modify existing record (fails if not exists) |
| `Save` | Create or update (idempotent) |
| `Delete` | Remove record |

Updated throughout agent-lab: "upsert" -> "save", "removes" -> "deletes"

## Validation Results

### Token No Longer Emitted
Verified token absent from:
- SSE streamed events
- `/api/workflows/runs/{id}/stages` snapshots
- `/api/workflows/runs/{id}` params and result
- Database tables (checkpoints, stages, runs)

### Profile Seeding
- All stages created with correct `profile_id` linkage
- JSONB `options` field populated correctly
- Re-running seed is idempotent (save semantics)

## Patterns Established

### Secret Management Pattern
```go
// Store sensitive data
initialState = initialState.SetSecret("token", token)

// Retrieve sensitive data
if tokenVal, ok := s.GetSecret("token"); ok {
    token = tokenVal.(string)
}
```

### Seeder Pattern
```go
type Seeder interface {
    Name() string
    Description() string
    Seed(ctx context.Context, tx *sql.Tx) error
}

func init() {
    registerSeeder(&MySeeder{})
}
```

## Files Modified

### go-agents-orchestration
- `pkg/state/state.go`
- `ARCHITECTURE.md`
- `tests/state/state_test.go`
- `CHANGELOG.md`

### agent-lab
- `go.mod`
- `internal/workflows/executor.go`
- `internal/workflows/profile.go`
- Multiple files for naming convention updates
- `ARCHITECTURE.md` - Data Mutation Methods convention
- `CLAUDE.md` - Naming conventions reference
- `PROJECT.md` - Session 4e notes

### agent-lab (New Files)
- `cmd/seed/main.go`
- `cmd/seed/seeder.go`
- `cmd/seed/seeds.go`
- `cmd/seed/profiles.go`
- `cmd/seed/seeds/classify_profiles.json`

## Session 4e Notes

Classification logic refinement identified for next session:
- Detected markings should contribute to classification regardless of fading/confidence
- Fading/confidence should affect ACCEPT/REVIEW/REJECT recommendation, not classification itself
