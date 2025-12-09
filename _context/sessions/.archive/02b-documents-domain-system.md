# Session 02b: Documents Domain System

## Problem Context

Session 2b implements document upload and management, building on the storage infrastructure from Session 2a. This session establishes the Documents domain following Milestone 1 patterns, enabling PDF upload with automatic page count extraction.

## Architecture Approach

### Domain Pattern
Follow the established domain structure from `internal/providers/`:
- System interface defines the contract
- Repository implements persistence + storage coordination
- Handler provides HTTP layer
- OpenAPI spec is domain-owned

### Key Design Decisions
- **Storage coordination**: Repository handles both blob storage and database operations
- **Fail-fast**: Extract PDF metadata before storage, fail early on invalid files
- **Simple atomicity**: Store blob first, delete on DB failure (defer self-healing)
- **Human-readable config**: Size limits as "100MB" strings, parsed via docker/go-units

---

## Phase 1: Configuration Update

### 1.1 Update `internal/config/storage.go`

Add imports:
```go
import (
	"fmt"
	"os"

	"github.com/docker/go-units"
)
```

Add constant:
```go
const (
	EnvStorageBasePath      = "STORAGE_BASE_PATH"
	EnvStorageMaxUploadSize = "STORAGE_MAX_UPLOAD_SIZE"
)
```

Update struct:
```go
type StorageConfig struct {
	BasePath         string `toml:"base_path"`
	MaxUploadSize    string `toml:"max_upload_size"`
	maxUploadSizeVal int64
}
```

Add method:
```go
func (c *StorageConfig) MaxUploadSizeBytes() int64 {
	return c.maxUploadSizeVal
}
```

Update `loadDefaults`:
```go
func (c *StorageConfig) loadDefaults() {
	if c.BasePath == "" {
		c.BasePath = ".data/blobs"
	}
	if c.MaxUploadSize == "" {
		c.MaxUploadSize = "100MB"
	}
}
```

Update `loadEnv`:
```go
func (c *StorageConfig) loadEnv() {
	if v := os.Getenv(EnvStorageBasePath); v != "" {
		c.BasePath = v
	}
	if v := os.Getenv(EnvStorageMaxUploadSize); v != "" {
		c.MaxUploadSize = v
	}
}
```

Update `validate`:
```go
func (c *StorageConfig) validate() error {
	if c.BasePath == "" {
		return fmt.Errorf("base_path required")
	}

	size, err := units.FromHumanSize(c.MaxUploadSize)
	if err != nil {
		return fmt.Errorf("invalid max_upload_size: %w", err)
	}
	if size <= 0 {
		return fmt.Errorf("max_upload_size must be positive")
	}
	c.maxUploadSizeVal = size

	return nil
}
```

Update `Merge`:
```go
func (c *StorageConfig) Merge(overlay *StorageConfig) {
	if overlay.BasePath != "" {
		c.BasePath = overlay.BasePath
	}

	if size, err := units.FromHumanSize(overlay.MaxUploadSize); err == nil {
		c.MaxUploadSize = overlay.MaxUploadSize
		c.maxUploadSizeVal = size
	}
}
```

---

## Phase 2: Database Migration

### 2.1 Create `cmd/migrate/migrations/000004_documents.up.sql`

```sql
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    page_count INTEGER,
    storage_key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_documents_name ON documents(name);
CREATE INDEX idx_documents_content_type ON documents(content_type);
CREATE INDEX idx_documents_created_at ON documents(created_at DESC);
```

### 2.2 Create `cmd/migrate/migrations/000004_documents.down.sql`

```sql
DROP INDEX IF EXISTS idx_documents_name;
DROP INDEX IF EXISTS idx_documents_content_type;
DROP INDEX IF EXISTS idx_documents_created_at;
DROP TABLE IF EXISTS documents;
```

---

## Phase 3: Documents Domain System

### 3.1 Create `internal/documents/document.go`

```go
package documents

import (
	"time"

	"github.com/google/uuid"
)

type Document struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	PageCount   *int      `json:"page_count,omitempty"`
	StorageKey  string    `json:"storage_key"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateCommand struct {
	Name        string
	Filename    string
	ContentType string
	SizeBytes   int64
	PageCount   *int
	Data        []byte
}

type UpdateCommand struct {
	Name string
}
```

### 3.2 Create `internal/documents/errors.go`

```go
package documents

import (
	"errors"
	"net/http"
)

var (
	ErrNotFound     = errors.New("document not found")
	ErrDuplicate    = errors.New("document storage key already exists")
	ErrFileTooLarge = errors.New("file exceeds maximum upload size")
	ErrInvalidFile  = errors.New("invalid file")
)

func MapHTTPStatus(err error) int {
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrDuplicate) {
		return http.StatusConflict
	}
	if errors.Is(err, ErrFileTooLarge) {
		return http.StatusRequestEntityTooLarge
	}
	if errors.Is(err, ErrInvalidFile) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
```

### 3.3 Create `internal/documents/projection.go`

```go
package documents

import "github.com/JaimeStill/agent-lab/pkg/query"

var projection = query.NewProjectionMap("public", "documents", "d").
	Project("id", "Id").
	Project("name", "Name").
	Project("filename", "Filename").
	Project("content_type", "ContentType").
	Project("size_bytes", "SizeBytes").
	Project("page_count", "PageCount").
	Project("storage_key", "StorageKey").
	Project("created_at", "CreatedAt").
	Project("updated_at", "UpdatedAt")
```

### 3.4 Create `internal/documents/scanner.go`

```go
package documents

import "github.com/JaimeStill/agent-lab/pkg/repository"

func scanDocument(s repository.Scanner) (Document, error) {
	var d Document
	err := s.Scan(
		&d.ID,
		&d.Name,
		&d.Filename,
		&d.ContentType,
		&d.SizeBytes,
		&d.PageCount,
		&d.StorageKey,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	return d, err
}
```

### 3.5 Create `internal/documents/filters.go`

```go
package documents

import (
	"net/url"

	"github.com/JaimeStill/agent-lab/pkg/query"
)

type Filters struct {
	Name        *string
	ContentType *string
}

func FiltersFromQuery(values url.Values) Filters {
	var f Filters

	if n := values.Get("name"); n != "" {
		f.Name = &n
	}

	if ct := values.Get("content_type"); ct != "" {
		f.ContentType = &ct
	}

	return f
}

func (f Filters) Apply(b *query.Builder) *query.Builder {
	return b.
		WhereContains("Name", f.Name).
		WhereContains("ContentType", f.ContentType)
}
```

### 3.6 Create `internal/documents/system.go`

```go
package documents

import (
	"context"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

type System interface {
	Create(ctx context.Context, cmd CreateCommand) (*Document, error)
	Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Document, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*Document, error)
	Search(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Document], error)
}
```

### 3.7 Create `internal/documents/repository.go`

```go
package documents

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/JaimeStill/agent-lab/internal/storage"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/agent-lab/pkg/repository"
	"github.com/google/uuid"
)

type repo struct {
	db         *sql.DB
	storage    storage.System
	logger     *slog.Logger
	pagination pagination.Config
}

func New(db *sql.DB, storage storage.System, logger *slog.Logger, pagination pagination.Config) System {
	return &repo{
		db:         db,
		storage:    storage,
		logger:     logger.With("system", "documents"),
		pagination: pagination,
	}
}

func (r *repo) Create(ctx context.Context, cmd CreateCommand) (*Document, error) {
	id := uuid.New()
	storageKey := buildStorageKey(id, cmd.Filename)

	if err := r.storage.Store(ctx, storageKey, cmd.Data); err != nil {
		return nil, fmt.Errorf("store file: %w", err)
	}

	q := `INSERT INTO documents (id, name, filename, content_type, size_bytes, page_count, storage_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, filename, content_type, size_bytes, page_count, storage_key, created_at, updated_at`

	doc, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Document, error) {
		return repository.QueryOne(ctx, tx, q, []any{
			id, cmd.Name, cmd.Filename, cmd.ContentType, cmd.SizeBytes, cmd.PageCount, storageKey,
		}, scanDocument)
	})

	if err != nil {
		if delErr := r.storage.Delete(ctx, storageKey); delErr != nil {
			r.logger.Error("cleanup failed after db error", "storage_key", storageKey, "error", delErr)
		}
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("document created", "id", doc.ID, "name", doc.Name, "storage_key", storageKey)
	return &doc, nil
}

func (r *repo) Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Document, error) {
	q := `UPDATE documents SET name = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, name, filename, content_type, size_bytes, page_count, storage_key, created_at, updated_at`

	doc, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Document, error) {
		return repository.QueryOne(ctx, tx, q, []any{cmd.Name, id}, scanDocument)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("document updated", "id", doc.ID, "name", doc.Name)
	return &doc, nil
}

func (r *repo) Delete(ctx context.Context, id uuid.UUID) error {
	doc, err := r.GetByID(ctx, id)
	if err != nil {
		if err == ErrNotFound {
			return nil
		}
		return err
	}

	q := `DELETE FROM documents WHERE id = $1`
	err = repository.WithTx(ctx, r.db, func(tx *sql.Tx) (struct{}, error) {
		return struct{}{}, repository.ExecExpectOne(ctx, tx, q, []any{id})
	})

	if err != nil {
		return repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	if err := r.storage.Delete(ctx, doc.StorageKey); err != nil {
		r.logger.Error("storage cleanup failed", "storage_key", doc.StorageKey, "error", err)
	}

	r.logger.Info("document deleted", "id", id)
	return nil
}

func (r *repo) GetByID(ctx context.Context, id uuid.UUID) (*Document, error) {
	q, args := query.NewBuilder(projection).BuildSingle("Id", id)
	doc, err := repository.QueryOne(ctx, r.db, q, args, scanDocument)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}
	return &doc, nil
}

func (r *repo) Search(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Document], error) {
	page.Normalize(r.pagination)

	qb := query.NewBuilder(projection, query.SortField{Field: "CreatedAt", Desc: true}).
		WhereSearch(page.Search, "Name", "Filename")

	filters.Apply(qb)

	if len(page.Sort) > 0 {
		qb.OrderByFields(page.Sort)
	}

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count documents: %w", err)
	}

	pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	docs, err := repository.QueryMany(ctx, r.db, pageSQL, pageArgs, scanDocument)
	if err != nil {
		return nil, fmt.Errorf("query documents: %w", err)
	}

	result := pagination.NewPageResult(docs, total, page.Page, page.PageSize)
	return &result, nil
}

func buildStorageKey(id uuid.UUID, filename string) string {
	return fmt.Sprintf("documents/%s/%s", id.String(), sanitizeFilename(filename))
}

func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	replacer := strings.NewReplacer(
		" ", "_",
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}
```

### 3.8 Create `internal/documents/handler.go`

```go
package documents

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/routes"
	"github.com/JaimeStill/agent-lab/pkg/handlers"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type Handler struct {
	sys           System
	logger        *slog.Logger
	pagination    pagination.Config
	maxUploadSize int64
}

func NewHandler(sys System, logger *slog.Logger, pagination pagination.Config, maxUploadSize int64) *Handler {
	return &Handler{
		sys:           sys,
		logger:        logger.With("handler", "documents"),
		pagination:    pagination,
		maxUploadSize: maxUploadSize,
	}
}

func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix:      "/api/documents",
		Tags:        []string{"Documents"},
		Description: "Document upload and management",
		Routes: []routes.Route{
			{Method: "POST", Pattern: "", Handler: h.Upload, OpenAPI: Spec.Upload},
			{Method: "GET", Pattern: "", Handler: h.List, OpenAPI: Spec.List},
			{Method: "GET", Pattern: "/{id}", Handler: h.GetByID, OpenAPI: Spec.Get},
			{Method: "PUT", Pattern: "/{id}", Handler: h.Update, OpenAPI: Spec.Update},
			{Method: "DELETE", Pattern: "/{id}", Handler: h.Delete, OpenAPI: Spec.Delete},
			{Method: "POST", Pattern: "/search", Handler: h.Search, OpenAPI: Spec.Search},
		},
	}
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(h.maxUploadSize); err != nil {
		handlers.RespondError(w, h.logger, http.StatusRequestEntityTooLarge, ErrFileTooLarge)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrInvalidFile)
		return
	}
	defer file.Close()

	if header.Size > h.maxUploadSize {
		handlers.RespondError(w, h.logger, http.StatusRequestEntityTooLarge, ErrFileTooLarge)
		return
	}

	data := make([]byte, header.Size)
	if _, err := file.Read(data); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrInvalidFile)
		return
	}

	contentType := detectContentType(header.Header.Get("Content-Type"), data)

	name := r.FormValue("name")
	if name == "" {
		name = header.Filename
	}

	var pageCount *int
	if contentType == "application/pdf" {
		pc, err := extractPDFPageCount(data)
		if err != nil {
			h.logger.Warn("failed to extract pdf page count", "error", err)
		} else {
			pageCount = pc
		}
	}

	cmd := CreateCommand{
		Name:        name,
		Filename:    header.Filename,
		ContentType: contentType,
		SizeBytes:   header.Size,
		PageCount:   pageCount,
		Data:        data,
	}

	doc, err := h.sys.Create(r.Context(), cmd)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusCreated, doc)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page := pagination.PageRequestFromQuery(r.URL.Query())
	filters := FiltersFromQuery(r.URL.Query())

	result, err := h.sys.Search(r.Context(), page, filters)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusInternalServerError, err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, result)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	doc, err := h.sys.GetByID(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, doc)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	var cmd UpdateCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	doc, err := h.sys.Update(r.Context(), id, cmd)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, doc)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	if err := h.sys.Delete(r.Context(), id); err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var page pagination.PageRequest
	if err := json.NewDecoder(r.Body).Decode(&page); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	filters := FiltersFromQuery(r.URL.Query())

	result, err := h.sys.Search(r.Context(), page, filters)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusInternalServerError, err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, result)
}

func detectContentType(header string, data []byte) string {
	if header != "" && header != "application/octet-stream" {
		return header
	}
	return http.DetectContentType(data)
}

func extractPDFPageCount(data []byte) (*int, error) {
	ctx, err := api.ReadContext(bytes.NewReader(data), model.NewDefaultConfiguration())
	if err != nil {
		return nil, err
	}
	count := ctx.PageCount
	return &count, nil
}
```

---

## Phase 4: Server Integration

### 4.1 Update `cmd/server/domain.go`

Add import:
```go
import (
	// ... existing imports
	"github.com/JaimeStill/agent-lab/internal/documents"
)
```

Update Domain struct:
```go
type Domain struct {
	Providers providers.System
	Agents    agents.System
	Documents documents.System
}
```

Update NewDomain:
```go
func NewDomain(runtime *Runtime) *Domain {
	return &Domain{
		Providers: providers.New(
			runtime.Database.Connection(),
			runtime.Logger,
			runtime.Pagination,
		),
		Agents: agents.New(
			runtime.Database.Connection(),
			runtime.Logger,
			runtime.Pagination,
		),
		Documents: documents.New(
			runtime.Database.Connection(),
			runtime.Storage,
			runtime.Logger,
			runtime.Pagination,
		),
	}
}
```

### 4.2 Update `cmd/server/routes.go`

Add import:
```go
import (
	// ... existing imports
	"github.com/JaimeStill/agent-lab/internal/documents"
)
```

In `registerRoutes`, add handler registration:
```go
documentHandler := documents.NewHandler(
	domain.Documents,
	runtime.Logger,
	runtime.Pagination,
	cfg.Storage.MaxUploadSizeBytes(),
)
r.RegisterGroup(documentHandler.Routes())
```

In OpenAPI components setup, add schemas:
```go
components.AddSchemas(documents.Spec.Schemas())
```

---

## Phase 5: Add Dependencies

Run:
```bash
go get github.com/docker/go-units
go get github.com/pdfcpu/pdfcpu
```

---

## Validation Checklist

After implementation, verify:

- [ ] `go vet ./...` passes
- [ ] Server starts without errors
- [ ] Upload PDF via `/api/documents` returns document with page_count
- [ ] Upload non-PDF returns document with page_count null
- [ ] GET `/api/documents` lists documents with pagination
- [ ] GET `/api/documents/{id}` returns single document
- [ ] PUT `/api/documents/{id}` updates name only
- [ ] DELETE `/api/documents/{id}` removes blob and record
- [ ] Scalar UI at `/docs` shows Documents endpoints
- [ ] "Try It" functionality works for all endpoints
