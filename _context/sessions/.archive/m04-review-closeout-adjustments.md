# Milestone 4 Closeout Adjustments

## Overview

This guide addresses an issue identified during the Milestone 4 review: partial profiles loaded from the database cause nil pointer dereferences in workflow nodes.

## Problem

The `aggressive-enhancement` seed profile only defines the `enhance` stage with a modified `legibility_threshold`. When this profile is loaded via `profile_id`, other stages return `nil` from `profile.Stage()`, causing panics:

```go
stage := profile.Stage("detect")  // Returns nil for aggressive-enhancement

if stage.SystemPrompt != nil {    // PANIC: nil pointer dereference
    opts["system_prompt"] = *stage.SystemPrompt
}
```

## Solution

Implement profile merging so that DB profiles overlay on top of hardcoded defaults. The default profile provides all stages, and the DB profile only needs to specify overrides.

---

## Phase 1: Add Merge Method

**File**: `internal/profiles/profile.go`

Add after the `Stage()` method:

```go
func (p *ProfileWithStages) Merge(other *ProfileWithStages) *ProfileWithStages {
	if other == nil {
		return p
	}

	result := &ProfileWithStages{
		Profile: other.Profile,
		Stages:  make([]ProfileStage, 0, len(p.Stages)),
	}

	otherStages := make(map[string]ProfileStage)
	for _, s := range other.Stages {
		otherStages[s.StageName] = s
	}

	seen := make(map[string]bool)
	for _, s := range p.Stages {
		if override, ok := otherStages[s.StageName]; ok {
			result.Stages = append(result.Stages, override)
		} else {
			result.Stages = append(result.Stages, s)
		}
		seen[s.StageName] = true
	}

	for _, s := range other.Stages {
		if !seen[s.StageName] {
			result.Stages = append(result.Stages, s)
		}
	}

	return result
}
```

---

## Phase 2: Update LoadProfile

**File**: `internal/workflows/profile.go`

Replace the existing `LoadProfile` function:

```go
func LoadProfile(ctx context.Context, rt *Runtime, params map[string]any, defaultProfile *profiles.ProfileWithStages) (*profiles.ProfileWithStages, error) {
	if profileIDStr, ok := params["profile_id"].(string); ok {
		profileID, err := uuid.Parse(profileIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid profile_id: %w", err)
		}
		dbProfile, err := rt.Profiles().Find(ctx, profileID)
		if err != nil {
			return nil, err
		}
		return defaultProfile.Merge(dbProfile), nil
	}
	return defaultProfile, nil
}
```

---

## Verification

### Validate Code

```bash
go vet ./...
go test ./tests/...
```

### Test Workflow Execution

**Prerequisites**:
- Server running (`go run ./cmd/server`)
- Database seeded (`go run ./cmd/seed -all`)
- Document uploaded (note `document_id`)
- Agent configured (note `agent_id`)

#### Test 1: Hardcoded Default (no profile_id)

```bash
curl -X POST http://localhost:8080/api/workflows/classify-docs/execute \
  -H "Content-Type: application/json" \
  -d '{
    "params": {
      "document_id": "<document-id>",
      "agent_id": "<agent-id>"
    },
    "token": "<api-key>"
  }'
```

**Expected**: Workflow completes, returns classification result.

#### Test 2: Aggressive-Enhancement Profile (partial)

Get profile ID:
```bash
curl 'http://localhost:8080/api/profiles?workflow_name=classify-docs' | jq '.data[] | select(.name == "aggressive-enhancement") | .id'
```

Execute with profile:
```bash
curl -X POST http://localhost:8080/api/workflows/classify-docs/execute \
  -H "Content-Type: application/json" \
  -d '{
    "params": {
      "document_id": "<document-uuid>",
      "agent_id": "<agent-uuid>",
      "profile_id": "<aggressive-enhancement-uuid>"
    },
    "token": "<api-token>"
  }'
```

**Expected**: Workflow completes without panic. Enhancement triggers at 0.3 threshold instead of 0.4.

#### Test 3: Default Profile from DB

Get profile ID:
```bash
curl 'http://localhost:8080/api/profiles?workflow_name=classify-docs' | jq '.data[] | select(.name == "default") | .id'
```

Execute with profile:
```bash
curl -X POST http://localhost:8080/api/workflows/classify-docs/execute \
  -H "Content-Type: application/json" \
  -d '{
    "params": {
      "document_id": "<document-uuid>",
      "agent_id": "<agent-uuid>",
      "profile_id": "<default-uuid>"
    },
    "token": "<api-token>"
  }'
```

**Expected**: Workflow completes. Behavior identical to Test 1 (DB default matches hardcoded default).

---

## Success Criteria

- [x] `go vet ./...` clean
- [x] All tests passing
- [x] Test 1 completes (hardcoded default)
- [x] Test 2 completes without panic (partial profile)
- [x] Test 3 completes (DB-loaded default)
