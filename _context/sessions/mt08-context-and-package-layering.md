# mt08: Context Optimization and Package Layering

## Summary

This maintenance session addressed two infrastructure concerns that were blocking Milestone 5 progress:

1. **Context Optimization** - Restructured `.claude/` for Claude Code's native skill triggering
2. **Package Layering Fix** - Eliminated `pkg/` → `internal/` dependency violation

## Changes

### Phase 1: Context Optimization

**Deleted:**
- `.claude/rules/` directory (content consolidated into CLAUDE.md)

**Updated:**
- `.claude/CLAUDE.md` - Consolidated from rules content, reduced to 147 lines (under 200 target)
  - Commands table
  - Session workflow reference
  - Architecture overview
  - Project structure
  - Skills table
  - Closeout checklist
  - Directory conventions

**Updated skill descriptions** (all 12 skills):
- Trigger-optimized format with "REQUIRED for..." pattern
- Improved keyword and file pattern triggers
- Better alignment with Claude Code's automatic skill loading

### Phase 2: Package Layering Fix

**Created:**
- `pkg/logging/logging.go` - Level, Format types with validation and factory
- `pkg/logging/config.go` - Config, Env types with Finalize/Merge pattern

**Moved:**
- `pkg/runtime/` → `internal/infrastructure/`
  - Allows direct `*config.Config` usage without interface abstraction
  - Eliminates pkg/ → internal/ dependency violation

**Updated:**
- `internal/config/config.go` - Added `loggingEnv`, changed `Logging` field to `logging.Config`
- `cmd/server/main.go` - Updated imports to use `internal/infrastructure`

**Deleted:**
- `pkg/runtime/` directory
- `internal/config/types.go` (types moved to pkg/logging)
- `internal/config/logging.go` (moved to pkg/logging)

**Test Infrastructure:**
- Created `tests/pkg_logging/` with logging_test.go and config_test.go
- Deleted `tests/internal_config/logging_test.go` and `tests/internal_config/types_test.go`

## Key Decisions

### Infrastructure as Internal Concern

The `Infrastructure` struct (lifecycle, logger, database, storage) is now in `internal/infrastructure/` rather than `pkg/runtime/`. This reflects its nature as an application-scoped composition that:
- Directly uses `*config.Config` without interface abstraction
- Is specific to this application's infrastructure needs
- Should not be exported as a public API

### Logging Package Pattern

`pkg/logging/` follows the same pattern as `pkg/database/` and `pkg/storage/`:
- Defines `Env` struct for environment variable mapping
- `internal/config/` passes concrete env var names
- Package is self-contained with types, validation, and factory

### Env Struct Pattern

The `Env` struct pattern separates:
- **pkg/** packages: Define `Env` struct shape (which env vars to look for)
- **internal/config/**: Pass actual env var names (application configuration)

This keeps pkg/ packages generic while internal/config/ owns the application-specific mapping.

## Validation

```bash
go vet ./...           # Clean
go test ./tests/...    # All tests passing
wc -l .claude/CLAUDE.md   # 147 lines (under 200)
ls .claude/rules/         # Directory does not exist
grep -r "internal/config" pkg/  # No results (no pkg/ → internal/ imports)
```

## Files Changed

| File | Change |
|------|--------|
| `.claude/CLAUDE.md` | Rewrote with consolidated content |
| `.claude/rules/` | Deleted directory |
| `.claude/skills/*/SKILL.md` | Updated 12 skill descriptions |
| `pkg/logging/logging.go` | New file |
| `pkg/logging/config.go` | New file |
| `internal/infrastructure/infrastructure.go` | Moved from pkg/runtime |
| `internal/config/config.go` | Added loggingEnv, updated Logging type |
| `internal/config/types.go` | Deleted |
| `internal/config/logging.go` | Deleted |
| `cmd/server/main.go` | Updated imports |
| `tests/pkg_logging/logging_test.go` | New file |
| `tests/pkg_logging/config_test.go` | New file |
| `tests/internal_config/logging_test.go` | Deleted |
| `tests/internal_config/types_test.go` | Deleted |

## Phase 3: Deferred

Web development skill rewrite deferred to separate session (part of M5 preparation).
