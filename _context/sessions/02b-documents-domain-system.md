# Session 02b: Documents Domain System

## Summary

Implemented the Documents domain system following established M1 patterns, enabling document upload with PDF metadata extraction and full CRUD operations.

## What Was Implemented

### Documents Domain (`internal/documents/`)
- **document.go**: Document struct, CreateCommand, UpdateCommand
- **errors.go**: ErrNotFound, ErrDuplicate, ErrFileTooLarge, ErrInvalidFile with MapHTTPStatus
- **filters.go**: Filters struct with Name and ContentType, FiltersFromQuery, Apply
- **projection.go**: Query projection map for documents table
- **scanner.go**: scanDocument function for row scanning
- **system.go**: System interface (Create, Update, Delete, GetByID, Search)
- **repository.go**: Implementation with blob storage integration and filename sanitization
- **handler.go**: HTTP handlers with multipart upload, PDF page count extraction
- **openapi.go**: OpenAPI specs and schemas for all endpoints

### Database Migration
- `000004_documents.up.sql`: Creates documents table with indexes
- `000004_documents.down.sql`: Drops table and indexes

### Configuration Updates
- Added `MaxUploadSize` to StorageConfig (human-readable format via docker/go-units)
- Environment variable: `STORAGE_MAX_UPLOAD_SIZE`

### Infrastructure Improvements
- Storage Delete now cleans up empty parent directories
- SortFields type with flexible JSON unmarshaling (accepts string or array)
- OpenAPI infrastructure aligned with OpenAPI 3.1 (Properties are Schemas)

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Update support | Name field only | Documents immutable except display name |
| Storage key format | `documents/{uuid}/{filename}` | Domain constructs keys, not storage system |
| pdfcpu integration | Direct in agent-lab | api.ReadContext for in-memory page count extraction |
| Size config | docker/go-units | Industry-standard parsing for "100MB" format |
| Sort field JSON | Flexible unmarshaling | Accepts both string and array formats for API compatibility |

## Patterns Established

### Multipart Upload with Metadata Extraction
```go
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
    r.ParseMultipartForm(h.maxUploadSize)
    file, header, _ := r.FormFile("file")
    // Read data, detect content-type, extract PDF metadata
    // Call sys.Create with CreateCommand
}
```

### Storage-First Atomicity
```go
// Store blob first
if err := r.storage.Store(ctx, storageKey, cmd.Data); err != nil {
    return nil, err
}
// Insert DB record
doc, err := repository.WithTx(...)
if err != nil {
    // Cleanup blob on DB failure
    r.storage.Delete(ctx, storageKey)
    return nil, err
}
```

### Empty Directory Cleanup
Storage Delete now checks if parent directory is empty after file removal and cleans it up.

## Files Created/Modified

### New Files
- `internal/documents/*.go` (9 files)
- `cmd/migrate/migrations/000004_documents.{up,down}.sql`
- `tests/internal_documents/*.go`
- `_context/openapi-integration.md`

### Modified Files
- `internal/config/storage.go` - MaxUploadSize field
- `internal/storage/filesystem.go` - Empty directory cleanup
- `pkg/pagination/pagination.go` - SortFields type
- `pkg/openapi/types.go` - Properties as Schemas
- `cmd/server/domain.go` - Documents domain wiring
- `cmd/server/routes.go` - Handler and schema registration

## Tests Added
- `tests/internal_documents/errors_test.go`
- `tests/internal_documents/filters_test.go`
- `tests/internal_storage/filesystem_test.go` - Empty directory cleanup tests
- `tests/pkg_pagination/pagination_test.go` - SortFields unmarshal tests

## Workflow Improvements

Updated CLAUDE.md Development Session Workflow:
- Added Step 4: OpenAPI Maintenance (moved from Documentation phase)
- OpenAPI specs now excluded from implementation guides
- Ensures handlers can reference Spec operations without compilation errors
