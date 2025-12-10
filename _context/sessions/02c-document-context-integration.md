# Session 2c: document-context Integration

## Summary

Integrated the document-context library for PDF page rendering with page images as first-class database entities. The implementation supports batch rendering, persistent storage, and flexible querying.

## What Was Implemented

### Database Schema
- Created `000005_images` migration with images table
- Full render options storage (format, dpi, quality, brightness, contrast, saturation, rotation, background)
- Cascade delete from documents to images
- Unique constraint on render parameters for deduplication

### Images Domain (`internal/images/`)
- **System interface** - List, Find, Data, Render, Delete operations
- **Repository** - Database operations with document-context integration
- **Handler** - HTTP endpoints with proper error mapping
- **OpenAPI spec** - Full API documentation

### Key Features
- Page range expressions: `1`, `1-5`, `1,3,5`, `1-5,10,15-20`, `-3`, `5-`
- Deduplication - Same render options returns existing image
- Force re-render option to regenerate images
- Optional parameters with ImageMagick neutral defaults

### API Endpoints
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/images/{documentId}/render` | Render document pages |
| GET | `/api/images` | List images with filters |
| GET | `/api/images/{id}` | Get image metadata |
| GET | `/api/images/{id}/data` | Get raw image binary |
| DELETE | `/api/images/{id}` | Delete image |

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Route structure | `/api/images` prefix | Images are first-class resources, documentId only required for render |
| DocumentID filter | Optional query param | Allows listing all images or filtering by document |
| Pages parameter | Optional, defaults to all | Simplifies "render all pages" use case |
| Request body | Optional | All fields have sensible defaults |
| ImageMagick defaults | brightness=100, contrast=0, saturation=100, rotation=0 | Neutral values that don't modify the image |
| Page count extraction | pdfcpu `api.PageCount()` | More reliable than context-based extraction |

## Patterns Established

### Cross-Domain Dependencies
Images domain depends on documents domain (unidirectional). Domain dependencies are specified before runtime dependencies in constructors:
```go
func New(docs documents.System, db *sql.DB, storage storage.System, ...) System
```

### Naming Conventions
Standardized method names across all domains (providers, agents, documents, images):

| Repository | Handler | OpenAPI | Purpose |
|------------|---------|---------|---------|
| List | List | List | Paginated collection query |
| Find | Find | Find | Single record by ID |
| Create | Create | Create | Create new record |
| Update | Update | Update | Update existing record |
| Delete | Delete | Delete | Remove record |

Images domain adds `Data` for binary retrieval and `Render` for page rendering.

Method ordering: queries (List, Find) before commands (Create, Update, Delete).
Route ordering: GET, POST, PUT, DELETE.

### Render Options Validation
Options struct with `Validate()` method that:
1. Validates and applies defaults
2. Returns wrapped domain errors
3. Mutates receiver to set defaults

### Page Range Parsing
Flexible expression syntax with deduplication and sorting:
```go
ParsePageRange("1-5,10,15-20", maxPage) // → [1,2,3,4,5,10,15,16,17,18,19,20]
```

### Query Builder Patterns
- `BuildPage(page, pageSize)` for paginated results
- `BuildSingle(field, value)` for single record by unique field
- `BuildSingleOrNull()` for optional single record lookup

## Files Created

### Images Domain
- `internal/images/document.go` - Document type support
- `internal/images/errors.go` - Domain errors and HTTP mapping
- `internal/images/filters.go` - Query filters
- `internal/images/handler.go` - HTTP handlers
- `internal/images/image.go` - Image model and RenderOptions
- `internal/images/openapi.go` - API specification
- `internal/images/pagerange.go` - Page range parsing
- `internal/images/projection.go` - Query projection
- `internal/images/repository.go` - Database operations
- `internal/images/scanner.go` - Row scanner
- `internal/images/system.go` - System interface

### Database
- `internal/migrations/000005_images.up.sql`
- `internal/migrations/000005_images.down.sql`

### Tests
- `tests/internal_images/errors_test.go`
- `tests/internal_images/filters_test.go`
- `tests/internal_images/image_test.go`
- `tests/internal_images/pagerange_test.go`

## Files Modified

- `internal/storage/storage.go` - Added `Path()` to interface
- `internal/storage/filesystem.go` - Implemented `Path()`
- `internal/documents/handler.go` - Fixed PDF page count extraction using pdfcpu
- `cmd/server/domain.go` - Added Images system
- `cmd/server/routes.go` - Wired images handler and routes

## Validation

All endpoints manually tested with curl:
- Render single page and page ranges
- List all images and filter by document_id
- Get image metadata and binary data
- Delete images

52 unit tests covering:
- Page range parsing (all expression types, edge cases, errors)
- Render options validation (defaults, bounds checking)
- Filters from query parsing
- Query builder filter application
- Error HTTP status mapping
