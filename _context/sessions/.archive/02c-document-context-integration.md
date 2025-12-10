# Session 2c: document-context Integration

## Overview

This session integrates the document-context library for PDF page rendering with images as first-class database entities. The implementation supports batch rendering, persistent storage, and workflow integration.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        API Layer                                │
├─────────────────────────────────────────────────────────────────┤
│  POST /documents/{id}/images/render  → Render pages to images   │
│  GET  /documents/{id}/images         → List rendered images     │
│  GET  /documents/{id}/images/{id}    → Get image metadata       │
│  GET  /documents/{id}/images/{id}/data → Get image binary       │
│  DELETE /documents/{id}/images/{id}  → Delete image             │
├─────────────────────────────────────────────────────────────────┤
│                     Domain Layer                                │
├──────────────────┬──────────────────────────────────────────────┤
│    documents     │              images                          │
│    (existing)    │         (NEW, depends on documents)          │
├──────────────────┴──────────────────────────────────────────────┤
│                    Infrastructure                               │
├─────────────────────────────────────────────────────────────────┤
│  storage.System  │  document-context (render only, no cache)    │
│  PostgreSQL      │                                              │
└─────────────────────────────────────────────────────────────────┘
```

## Prerequisites

- Go 1.25.2+
- ImageMagick 7+ installed (`magick` command available)
- PostgreSQL 17 running
- Existing agent-lab codebase from Session 2b

---

## Phase 1: Infrastructure

### 1.1 Add document-context Dependency

```bash
go get github.com/JaimeStill/document-context@v0.1.0
```

### 1.2 Fix WhereNullable in Query Builder

**File:** `pkg/query/builder.go`

The existing `WhereNullable` method is broken (calls non-existent `b.Where`). Replace the entire method:

```go
func (b *Builder) WhereNullable(field string, val any) *Builder {
	col := b.projection.Column(field)
	if val == nil {
		b.conditions = append(b.conditions, condition{
			clause: col + " IS NULL",
			args:   nil,
		})
	} else {
		b.conditions = append(b.conditions, condition{
			clause: fmt.Sprintf("%s = $%%d", col),
			args:   []any{val},
		})
	}
	return b
}
```

**Remove Build method** (lines 73-84). It was incorrectly added when `BuildPage` or `BuildSingle` should be used instead.

**Add BuildSingleOrNull method** (for queries that may return zero or one row with complex WHERE conditions):

```go
func (b *Builder) BuildSingleOrNull() (string, []any) {
	where, args, _ := b.buildWhere(1)
	sql := fmt.Sprintf(
		"SELECT %s FROM %s%s LIMIT 1",
		b.projection.Columns(),
		b.projection.Table(),
		where,
	)
	return sql, args
}
```

### 1.3 Add Path() to Storage Interface

**File:** `internal/storage/storage.go`

Add to the `System` interface:

```go
Path(ctx context.Context, key string) (string, error)
```

### 1.3 Implement Path() in Filesystem Storage

**File:** `internal/storage/filesystem.go`

Add method:

```go
func (f *filesystem) Path(ctx context.Context, key string) (string, error) {
	path, err := f.fullPath(key)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("stat file: %w", err)
	}

	return path, nil
}
```

---

## Phase 2: Database Migration

### 2.1 Create Migration Files

**File:** `internal/migrations/000005_images.up.sql`

```sql
CREATE TABLE images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    page_number INTEGER NOT NULL,
    format TEXT NOT NULL,
    dpi INTEGER NOT NULL,
    quality INTEGER,
    brightness INTEGER,
    contrast INTEGER,
    saturation INTEGER,
    rotation INTEGER,
    background TEXT,
    storage_key TEXT NOT NULL UNIQUE,
    size_bytes BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(document_id, page_number, format, dpi, quality,
           brightness, contrast, saturation, rotation, background)
);

CREATE INDEX idx_images_document_id ON images(document_id);
CREATE INDEX idx_images_created_at ON images(created_at DESC);
```

**File:** `internal/migrations/000005_images.down.sql`

```sql
DROP TABLE IF EXISTS images;
```

### 2.2 Embed Migration

**File:** `internal/migrations/migrations.go`

Verify the embed directive includes the new files (should work automatically with `*.sql` glob).

---

## Phase 3: Images Domain Package

### 3.1 Image Model

**File:** `internal/images/image.go`

```go
package images

import (
	"fmt"
	"strings"
	"time"

	"github.com/JaimeStill/document-context/pkg/document"
	"github.com/google/uuid"
)

type Image struct {
	ID         uuid.UUID            `json:"id"`
	DocumentID uuid.UUID            `json:"document_id"`
	PageNumber int                  `json:"page_number"`
	Format     document.ImageFormat `json:"format"`
	DPI        int                  `json:"dpi"`
	Quality    *int      `json:"quality,omitempty"`
	Brightness *int      `json:"brightness,omitempty"`
	Contrast   *int      `json:"contrast,omitempty"`
	Saturation *int      `json:"saturation,omitempty"`
	Rotation   *int      `json:"rotation,omitempty"`
	Background *string   `json:"background,omitempty"`
	StorageKey string    `json:"storage_key"`
	SizeBytes  int64     `json:"size_bytes"`
	CreatedAt  time.Time `json:"created_at"`
}

type RenderOptions struct {
	Pages      string               `json:"pages"`
	Format     document.ImageFormat `json:"format"`
	DPI        int                  `json:"dpi"`
	Quality    *int                 `json:"quality,omitempty"`
	Brightness *int                 `json:"brightness,omitempty"`
	Contrast   *int                 `json:"contrast,omitempty"`
	Saturation *int                 `json:"saturation,omitempty"`
	Rotation   *int                 `json:"rotation,omitempty"`
	Background *string              `json:"background,omitempty"`
	Force      bool                 `json:"force"`
}

func (o *RenderOptions) Validate() error {
	if o.Pages == "" {
		return fmt.Errorf("%w: pages is required", ErrInvalidRenderOption)
	}

	format, err := ParseImageFormat(string(o.Format))
	if err != nil {
		return err
	}
	o.Format = format

	if o.DPI == 0 {
		o.DPI = 300
	} else if o.DPI < 72 || o.DPI > 1200 {
		return fmt.Errorf("%w: dpi must be between 72 and 1200", ErrInvalidRenderOption)
	}

	if o.Quality != nil && (*o.Quality < 1 || *o.Quality > 100) {
		return fmt.Errorf("%w: quality must be between 1 and 100", ErrInvalidRenderOption)
	}

	if o.Brightness != nil && (*o.Brightness < 0 || *o.Brightness > 200) {
		return fmt.Errorf("%w: brightness must be between 0 and 200", ErrInvalidRenderOption)
	}

	if o.Contrast != nil && (*o.Contrast < -100 || *o.Contrast > 100) {
		return fmt.Errorf("%w: contrast must be between -100 and 100", ErrInvalidRenderOption)
	}

	if o.Saturation != nil && (*o.Saturation < 0 || *o.Saturation > 200) {
		return fmt.Errorf("%w: saturation must be between 0 and 200", ErrInvalidRenderOption)
	}

	if o.Rotation != nil && (*o.Rotation < 0 || *o.Rotation > 360) {
		return fmt.Errorf("%w: rotation must be between 0 and 360", ErrInvalidRenderOption)
	}

	if o.Background == nil {
		bg := "white"
		o.Background = &bg
	}

	return nil
}

func (o RenderOptions) ToImage(
	id, documentID uuid.UUID,
	pageNumber int,
	storageKey string,
	sizeBytes int64,
) *Image {
	return &Image{
		ID:         id,
		DocumentID: documentID,
		PageNumber: pageNumber,
		Format:     o.Format,
		DPI:        o.DPI,
		Quality:    o.Quality,
		Brightness: o.Brightness,
		Contrast:   o.Contrast,
		Saturation: o.Saturation,
		Rotation:   o.Rotation,
		Background: o.Background,
		StorageKey: storageKey,
		SizeBytes:  sizeBytes,
	}
}

func (o RenderOptions) ToImageConfig() config.ImageConfig {
	cfg := config.ImageConfig{
		Format:  string(o.Format),
		DPI:     o.DPI,
		Options: make(map[string]any),
	}

	if o.Quality != nil {
		cfg.Quality = *o.Quality
	} else if o.Format == document.JPEG {
		cfg.Quality = 90
	}

	if o.Brightness != nil {
		cfg.Options["brightness"] = *o.Brightness
	}
	if o.Contrast != nil {
		cfg.Options["contrast"] = *o.Contrast
	}
	if o.Saturation != nil {
		cfg.Options["saturation"] = *o.Saturation
	}
	if o.Rotation != nil {
		cfg.Options["rotation"] = *o.Rotation
	}
	if o.Background != nil {
		cfg.Options["background"] = *o.Background
	}

	return cfg
}
```

Add import for `"github.com/JaimeStill/document-context/pkg/config"` to image.go.

### 3.2 Domain Errors

**File:** `internal/images/errors.go`

```go
package images

import (
	"errors"
	"net/http"
)

var (
	ErrNotFound           = errors.New("image not found")
	ErrDuplicate          = errors.New("image already exists")
	ErrDocumentNotFound   = errors.New("document not found")
	ErrUnsupportedFormat  = errors.New("document format not supported for rendering")
	ErrInvalidPageRange   = errors.New("invalid page range")
	ErrPageOutOfRange     = errors.New("page number out of range")
	ErrInvalidRenderOption = errors.New("invalid render option")
	ErrRenderFailed       = errors.New("render failed")
)

func MapHTTPStatus(err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrDuplicate):
		return http.StatusConflict
	case errors.Is(err, ErrDocumentNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrUnsupportedFormat):
		return http.StatusBadRequest
	case errors.Is(err, ErrInvalidPageRange):
		return http.StatusBadRequest
	case errors.Is(err, ErrPageOutOfRange):
		return http.StatusBadRequest
	case errors.Is(err, ErrInvalidRenderOption):
		return http.StatusBadRequest
	case errors.Is(err, ErrRenderFailed):
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
```

### 3.3 Image Format Parsing

Add to `internal/images/image.go` (uses `document.ImageFormat` from document-context):

```go
func ParseImageFormat(s string) (document.ImageFormat, error) {
	switch strings.ToLower(s) {
	case "png":
		return document.PNG, nil
	case "jpg", "jpeg":
		return document.JPEG, nil
	case "":
		return document.PNG, nil
	default:
		return "", fmt.Errorf("%w: format must be 'png' or 'jpg'", ErrInvalidRenderOption)
	}
}
```

Add imports for `"strings"`, `"fmt"`, and `"github.com/JaimeStill/document-context/pkg/document"` to image.go.

### 3.4 Document Abstraction

**File:** `internal/images/document.go`

This is a shim that will be moved to document-context in Session 2d.
See `_context/document-context-format-support.md` for migration plan.

```go
package images

import (
	"io"

	"github.com/JaimeStill/document-context/pkg/document"
)

var SupportedFormats = map[string]bool{
	"application/pdf": true,
}

type PageExtractor interface {
	ExtractPage(pageNum int) (document.Page, error)
	io.Closer
}

func IsSupported(contentType string) bool {
	return SupportedFormats[contentType]
}

func OpenDocument(path string, contentType string) (PageExtractor, error) {
	switch contentType {
	case "application/pdf":
		return document.OpenPDF(path)
	default:
		return nil, ErrUnsupportedFormat
	}
}
```

### 3.5 Filters

**File:** `internal/images/filters.go`

```go
package images

import (
	"net/url"
	"strconv"

	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/document-context/pkg/document"
	"github.com/google/uuid"
)

type Filters struct {
	DocumentID uuid.UUID
	Format     *document.ImageFormat
	PageNumber *int
}

func FiltersFromQuery(values url.Values, documentID uuid.UUID) Filters {
	f := Filters{DocumentID: documentID}

	if format := values.Get("format"); format != "" {
		parsed, err := ParseImageFormat(format)
		if err == nil {
			f.Format = &parsed
		}
	}

	if pg := values.Get("page_number"); pg != "" {
		if page, err := strconv.Atoi(pg); err == nil {
			f.PageNumber = &page
		}
	}

	return f
}

func (f Filters) Apply(b *query.Builder) *query.Builder {
	b.WhereEquals("DocumentID", f.DocumentID)

	if f.Format != nil {
		b.WhereEquals("Format", *f.Format)
	}
	if f.PageNumber != nil {
		b.WhereEquals("PageNumber", *f.PageNumber)
	}

	return b
}
```

### 3.6 System Interface

**File:** `internal/images/system.go`

```go
package images

import (
	"context"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

type System interface {
	Render(ctx context.Context, documentID uuid.UUID, opts RenderOptions) ([]Image, error)
	Search(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Image], error)
	GetByID(ctx context.Context, id uuid.UUID) (*Image, error)
	GetData(ctx context.Context, id uuid.UUID) ([]byte, string, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
```

### 3.7 Projection Map

**File:** `internal/images/projection.go`

```go
package images

import "github.com/JaimeStill/agent-lab/pkg/query"

var projection = query.NewProjectionMap("public", "images", "i").
	Project("id", "ID").
	Project("document_id", "DocumentID").
	Project("page_number", "PageNumber").
	Project("format", "Format").
	Project("dpi", "DPI").
	Project("quality", "Quality").
	Project("brightness", "Brightness").
	Project("contrast", "Contrast").
	Project("saturation", "Saturation").
	Project("rotation", "Rotation").
	Project("background", "Background").
	Project("storage_key", "StorageKey").
	Project("size_bytes", "SizeBytes").
	Project("created_at", "CreatedAt")

var defaultSort = query.SortField{Field: "CreatedAt", Descending: true}
```

**Note:** This `defaultSort` pattern was applied across all domain packages (`agents`, `documents`, `images`, `providers`) to standardize default sort configuration. Single-field sorts use `query.SortField` directly; multi-field sorts use `[]query.SortField` with spread (`defaultSort...`).

### 3.8 Scanner

**File:** `internal/images/scanner.go`

```go
package images

import "github.com/JaimeStill/agent-lab/pkg/repository"

func scanImage(s repository.Scanner) (Image, error) {
	var img Image
	err := s.Scan(
		&img.ID,
		&img.DocumentID,
		&img.PageNumber,
		&img.Format,
		&img.DPI,
		&img.Quality,
		&img.Brightness,
		&img.Contrast,
		&img.Saturation,
		&img.Rotation,
		&img.Background,
		&img.StorageKey,
		&img.SizeBytes,
		&img.CreatedAt,
	)
	return img, err
}
```

### 3.9 Page Range Parser

**File:** `internal/images/pagerange.go`

```go
package images

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func ParsePageRange(expr string, maxPage int) ([]int, error) {
	if expr == "" {
		return nil, fmt.Errorf("%w: empty page range", ErrInvalidPageRange)
	}

	seen := make(map[int]bool)
	parts := strings.Split(expr, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			start, end, err := parseRange(part, maxPage)
			if err != nil {
				return nil, err
			}

			for i := start; i <= end; i++ {
				seen[i] = true
			}
		} else {
			page, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid page %q", ErrInvalidPageRange, part)
			}
			if page < 1 || page > maxPage {
				return nil, fmt.Errorf("%w: page %d out of range [1-%d]", ErrPageOutOfRange, page, maxPage)
			}
			seen[page] = true
		}
	}

	if len(seen) == 0 {
		return nil, fmt.Errorf("%w: no valid pages", ErrInvalidPageRange)
	}

	pages := make([]int, 0, len(seen))
	for page := range seen {
		pages = append(pages, page)
	}
	sort.Ints(pages)

	return pages, nil
}

func parseRange(part string, maxPage int) (int, int, error) {
	idx := strings.Index(part, "-")
	if idx == -1 {
		return 0, 0, fmt.Errorf("%w: invalid range %q", ErrInvalidPageRange, part)
	}

	startStr := strings.TrimSpace(part[:idx])
	endStr := strings.TrimSpace(part[idx+1:])

	var start, end int
	var err error

	if startStr == "" {
		start = 1
	} else {
		start, err = strconv.Atoi(startStr)
		if err != nil {
			return 0, 0, fmt.Errorf("%w: invalid start %q", ErrInvalidPageRange, startStr)
		}
	}

	if endStr == "" {
		end = maxPage
	} else {
		end, err = strconv.Atoi(endStr)
		if err != nil {
			return 0, 0, fmt.Errorf("%w: invalid end %q", ErrInvalidPageRange, endStr)
		}
	}

	if start < 1 {
		return 0, 0, fmt.Errorf("%w: start page must be >= 1", ErrInvalidPageRange)
	}
	if end > maxPage {
		return 0, 0, fmt.Errorf("%w: end page %d exceeds document pages (%d)", ErrPageOutOfRange, end, maxPage)
	}
	if start > end {
		return 0, 0, fmt.Errorf("%w: start > end in %q", ErrInvalidPageRange, part)
	}

	return start, end, nil
}
```

### 3.10 Repository

**File:** `internal/images/repository.go`

**Design Notes:**
- Injects `documents.System` for cross-domain access (unidirectional dependency: images → documents)
- Injects `pagination.Config` following established documents pattern
- Uses `RenderOptions.ToImageConfig()` method for renderer creation
- Private methods use `create`/`update` terminology to match API patterns
- `IsSupported()` replaces `IsDocumentSupported()` (package context is clear)
- `Search` follows documents pattern with PageRequest, Filters, and PageResult

```go
package images

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/JaimeStill/agent-lab/internal/documents"
	"github.com/JaimeStill/agent-lab/internal/storage"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/agent-lab/pkg/repository"
	"github.com/JaimeStill/document-context/pkg/image"
	"github.com/google/uuid"
)

type repo struct {
	db         *sql.DB
	documents  documents.System
	storage    storage.System
	logger     *slog.Logger
	pagination pagination.Config
}

func New(db *sql.DB, docs documents.System, storage storage.System, logger *slog.Logger, pagination pagination.Config) System {
	return &repo{
		db:         db,
		documents:  docs,
		storage:    storage,
		logger:     logger.With("system", "images"),
		pagination: pagination,
	}
}

func (r *repo) Render(ctx context.Context, documentID uuid.UUID, opts RenderOptions) ([]Image, error) {
	doc, err := r.documents.GetByID(ctx, documentID)
	if err != nil {
		if errors.Is(err, documents.ErrNotFound) {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}

	if !IsSupported(doc.ContentType) {
		return nil, ErrUnsupportedFormat
	}

	if doc.PageCount == nil {
		return nil, fmt.Errorf("%w: document has no page count", ErrRenderFailed)
	}

	pages, err := ParsePageRange(opts.Pages, *doc.PageCount)
	if err != nil {
		return nil, err
	}

	docPath, err := r.storage.Path(ctx, doc.StorageKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRenderFailed, err)
	}

	openDoc, err := OpenDocument(docPath, doc.ContentType)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRenderFailed, err)
	}
	defer openDoc.Close()

	renderer, err := image.NewImageMagickRenderer(opts.ToImageConfig())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRenderFailed, err)
	}

	var results []Image
	for _, pageNum := range pages {
		img, err := r.renderPage(ctx, documentID, openDoc, renderer, pageNum, opts)
		if err != nil {
			return nil, err
		}
		results = append(results, *img)
	}

	return results, nil
}

func (r *repo) renderPage(ctx context.Context, documentID uuid.UUID, doc PageExtractor, renderer image.Renderer, pageNum int, opts RenderOptions) (*Image, error) {
	existing, err := r.findExisting(ctx, documentID, pageNum, opts)
	if err != nil {
		return nil, err
	}

	if existing != nil && !opts.Force {
		return existing, nil
	}

	page, err := doc.ExtractPage(pageNum)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRenderFailed, err)
	}

	data, err := page.ToImage(renderer, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRenderFailed, err)
	}

	storageKey := fmt.Sprintf("images/%s/%s.%s", documentID, uuid.New(), opts.Format)

	if err := r.storage.Store(ctx, storageKey, data); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRenderFailed, err)
	}

	if existing != nil {
		if err := r.update(ctx, existing.ID, storageKey, int64(len(data))); err != nil {
			r.storage.Delete(ctx, storageKey)
			return nil, err
		}
		return r.GetByID(ctx, existing.ID)
	}

	img := opts.ToImage(uuid.New(), documentID, pageNum, storageKey, int64(len(data)))

	if err := r.create(ctx, img); err != nil {
		r.storage.Delete(ctx, storageKey)
		return nil, err
	}

	return r.GetByID(ctx, img.ID)
}

func (r *repo) Search(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Image], error) {
	page.Normalize(r.pagination)

	qb := query.NewBuilder(projection, defaultSort)
	filters.Apply(qb)

	if len(page.Sort) > 0 {
		qb.OrderByFields(page.Sort)
	}

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count images: %w", err)
	}

	pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	imgs, err := repository.QueryMany(ctx, r.db, pageSQL, pageArgs, scanImage)
	if err != nil {
		return nil, fmt.Errorf("query images: %w", err)
	}

	result := pagination.NewPageResult(imgs, total, page.Page, page.PageSize)
	return &result, nil
}

func (r *repo) GetByID(ctx context.Context, id uuid.UUID) (*Image, error) {
	q, args := query.NewBuilder(projection).BuildSingle("ID", id)
	img, err := repository.QueryOne(ctx, r.db, q, args, scanImage)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}
	return &img, nil
}

func (r *repo) GetData(ctx context.Context, id uuid.UUID) ([]byte, string, error) {
	img, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, "", err
	}

	data, err := r.storage.Retrieve(ctx, img.StorageKey)
	if err != nil {
		return nil, "", fmt.Errorf("retrieve image: %w", err)
	}

	contentType, err := img.Format.MimeType()
	if err != nil {
		contentType = http.DetectContentType(data)
	}

	return data, contentType, nil
}

func (r *repo) Delete(ctx context.Context, id uuid.UUID) error {
	img, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	q := `DELETE FROM images WHERE id = $1`
	_, err = repository.WithTx(ctx, r.db, func(tx *sql.Tx) (struct{}, error) {
		return struct{}{}, repository.ExecExpectOne(ctx, tx, q, id)
	})

	if err != nil {
		return repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	if err := r.storage.Delete(ctx, img.StorageKey); err != nil {
		r.logger.Warn("failed to delete image file", "key", img.StorageKey, "error", err)
	}

	return nil
}

func (r *repo) findExisting(ctx context.Context, documentID uuid.UUID, pageNum int, opts RenderOptions) (*Image, error) {
	q, args := query.NewBuilder(projection).
		WhereEquals("DocumentID", documentID).
		WhereEquals("PageNumber", pageNum).
		WhereEquals("Format", opts.Format).
		WhereEquals("DPI", opts.DPI).
		WhereNullable("Quality", opts.Quality).
		WhereNullable("Brightness", opts.Brightness).
		WhereNullable("Contrast", opts.Contrast).
		WhereNullable("Saturation", opts.Saturation).
		WhereNullable("Rotation", opts.Rotation).
		WhereNullable("Background", opts.Background).
		BuildSingleOrNull()

	img, err := repository.QueryOne(ctx, r.db, q, args, scanImage)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &img, nil
}

func (r *repo) create(ctx context.Context, img *Image) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO images (id, document_id, page_number, format, dpi, quality,
			brightness, contrast, saturation, rotation, background, storage_key, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		img.ID, img.DocumentID, img.PageNumber, img.Format, img.DPI, img.Quality,
		img.Brightness, img.Contrast, img.Saturation, img.Rotation, img.Background,
		img.StorageKey, img.SizeBytes)
	return err
}

func (r *repo) update(ctx context.Context, id uuid.UUID, storageKey string, sizeBytes int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE images SET storage_key = $1, size_bytes = $2 WHERE id = $3`,
		storageKey, sizeBytes, id)
	return err
}
```

### 3.11 Handler

**File:** `internal/images/handler.go`

```go
package images

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/JaimeStill/agent-lab/internal/routes"
	"github.com/JaimeStill/agent-lab/pkg/handlers"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

type Handler struct {
	sys        System
	logger     *slog.Logger
	pagination pagination.Config
}

func NewHandler(sys System, logger *slog.Logger, pagination pagination.Config) *Handler {
	return &Handler{
		sys:        sys,
		logger:     logger.With("handler", "images"),
		pagination: pagination,
	}
}

func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix:      "/api/documents/{documentId}/images",
		Tags:        []string{"Images"},
		Description: "Document page image rendering and management",
		Routes: []routes.Route{
			{Method: "POST", Pattern: "/render", Handler: h.Render, OpenAPI: Spec.Render},
			{Method: "GET", Pattern: "", Handler: h.List, OpenAPI: Spec.List},
			{Method: "GET", Pattern: "/{id}", Handler: h.GetByID, OpenAPI: Spec.Get},
			{Method: "GET", Pattern: "/{id}/data", Handler: h.GetData, OpenAPI: Spec.GetData},
			{Method: "DELETE", Pattern: "/{id}", Handler: h.Delete, OpenAPI: Spec.Delete},
		},
	}
}

func (h *Handler) Render(w http.ResponseWriter, r *http.Request) {
	documentID, err := uuid.Parse(r.PathValue("documentId"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	var opts RenderOptions
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	if err := opts.Validate(); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	images, err := h.sys.Render(r.Context(), documentID, opts)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusCreated, images)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	documentID, err := uuid.Parse(r.PathValue("documentId"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	page := pagination.PageRequestFromQuery(r.URL.Query(), h.pagination)
	filters := FiltersFromQuery(r.URL.Query(), documentID)

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

	img, err := h.sys.GetByID(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, img)
}

func (h *Handler) GetData(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	data, contentType, err := h.sys.GetData(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
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
```

---

## Phase 4: Server Wiring

### 4.1 Update Domain

**File:** `cmd/server/domain.go`

Add images system:

```go
import "github.com/JaimeStill/agent-lab/internal/images"

type Domain struct {
	Providers providers.System
	Agents    agents.System
	Documents documents.System
	Images    images.System
}
```

In NewDomain() (note: Documents must be initialized before Images due to dependency):

```go
docs := documents.New(
	runtime.Database.Connection(),
	runtime.Storage,
	runtime.Logger,
	runtime.Pagination,
)

return &Domain{
	Providers: providers.New(runtime.Database.Connection(), runtime.Logger),
	Agents:    agents.New(runtime.Database.Connection(), runtime.Logger),
	Documents: docs,
	Images: images.New(
		runtime.Database.Connection(),
		docs,
		runtime.Storage,
		runtime.Logger,
		runtime.Pagination,
	),
}
```

### 4.2 Update Routes

**File:** `cmd/server/routes.go`

Add images handler:

```go
imageHandler := images.NewHandler(domain.Images, runtime.Logger, runtime.Pagination)
r.RegisterGroup(imageHandler.Routes())
```

Add images schemas:

```go
components.AddSchemas(images.Spec.Schemas())
```

---

## Phase 5: Run Migration

```bash
go run cmd/migrate/main.go up
```

---

## Validation Criteria

After implementation, verify:

1. **Render single page**: `POST /api/documents/{id}/images/render` with `{"pages": "1"}` creates image record
2. **Render page range**: `{"pages": "1-5"}` creates 5 image records
3. **List images**: `GET /api/documents/{id}/images` returns paginated images
4. **Get image metadata**: `GET /api/documents/{id}/images/{imageId}` returns JSON metadata
5. **Get image binary**: `GET /api/documents/{id}/images/{imageId}/data` returns PNG/JPEG
6. **Delete image**: Removes from storage and database
7. **Duplicate handling**: Same settings without `force` returns existing record
8. **Force re-render**: `force: true` re-renders even if exists
9. **Cascade delete**: Deleting document removes all page images
10. **Invalid page range**: `{"pages": "999"}` on 3-page doc returns 400
