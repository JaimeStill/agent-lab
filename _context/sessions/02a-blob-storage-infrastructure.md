# Session 02a: Blob Storage Infrastructure

**Status**: Completed
**Date**: 2025-12-08

## Summary

Implemented blob storage abstraction with filesystem implementation, establishing the foundation for document upload and processing in Milestone 2.

## What Was Implemented

### New Files

| File | Purpose |
|------|---------|
| `internal/storage/errors.go` | Storage error types (ErrNotFound, ErrPermissionDenied, ErrInvalidKey) |
| `internal/storage/storage.go` | System interface definition |
| `internal/storage/filesystem.go` | Filesystem implementation |
| `internal/config/storage.go` | StorageConfig section |
| `tests/internal_storage/filesystem_test.go` | Filesystem storage tests |
| `tests/internal_storage/errors_test.go` | Error type tests |
| `tests/internal_config/storage_test.go` | Config tests |

### Modified Files

| File | Changes |
|------|---------|
| `internal/config/config.go` | Added Storage field, Finalize(), Merge() integration |
| `cmd/server/runtime.go` | Added Storage to Runtime struct |
| `config.toml` | Added [storage] section |
| `.env` | Added STORAGE_BASE_PATH |
| `CLAUDE.md` | Updated testing conventions, added Go commands section |

## Key Decisions

### Package Structure
- **Decision**: Interface and implementation both in `internal/storage/`
- **Rationale**: Mirrors database pattern; avoids import boundary issues with lifecycle coordinator

### Interface Design
- **Methods**: Store, Retrieve, Delete, Validate, Start
- **Data format**: `[]byte` (no streaming - defer until needed)
- **Validate signature**: `(bool, error)` - false/nil for not exists, false/err for permission issues

### Key Handling
- Full key string passed to interface
- Implementation handles path construction internally
- Build/parse key methods are internal, not interface methods

### Testing Standard
- Updated CLAUDE.md to remove arbitrary percentage requirements
- Success measured by coverage of critical paths:
  - Happy paths
  - Security paths (path traversal prevention)
  - Error types
  - Integration points

## Patterns Established

### Storage System Pattern
```go
type System interface {
    Store(ctx, key, data) error
    Retrieve(ctx, key) ([]byte, error)
    Delete(ctx, key) error
    Validate(ctx, key) (bool, error)
    Start(lc *lifecycle.Coordinator) error
}
```

### Atomic File Writes
Store uses temp file + rename pattern for crash safety:
```go
tmpPath := path + ".tmp"
os.WriteFile(tmpPath, data, 0644)
os.Rename(tmpPath, path)
```

### Path Traversal Protection
`fullPath()` helper validates keys and prevents directory traversal:
- Rejects empty keys
- Rejects `..` prefixes
- Rejects absolute paths
- Validates final path stays within base directory

## Validation Results

- All 21 storage tests passing
- Critical paths covered:
  - ✅ Happy paths (Store/Retrieve/Delete/Validate round-trips)
  - ✅ Security (path traversal prevention)
  - ✅ Error types (defined and distinguishable)
  - ✅ Lifecycle integration (directory creation on startup)

## Dependencies for Next Session

Session 2b (Documents Domain System) can now:
- Use `storage.System` for document blob storage
- Store documents at keys like `documents/{id}/{filename}`
- Leverage Validate for existence checks before retrieval
