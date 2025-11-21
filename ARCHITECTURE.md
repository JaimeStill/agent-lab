# agent-lab Architecture

**Status**: Living Document - Flexible guideline that adapts as requirements emerge

## Overview

agent-lab is a containerized Go web service platform for building and orchestrating agentic workflows. It builds upon the foundation of go-agents, go-agents-orchestration, and document-context libraries, providing a production-ready HTTP API for intelligent document processing and workflow execution.

## Architectural Principles

### 1. Layered Composition Architecture (LCA)

The system follows LCA principles established in the underlying Go libraries:

- **Data vs Behavior Separation**: Configuration structs (data) are distinct from domain objects (behavior)
- **Explicit Boundaries**: Validation occurs at transformation boundaries
- **Interface-Based APIs**: Public contracts through interfaces, private implementations
- **Immutability Where Possible**: Configuration immutable after initialization
- **Fail-Fast Validation**: Invalid configurations rejected at creation time

### 2. Service Lifecycle Model

**Long-Running Services** (Application-scoped)
- Initialized at server startup
- Live for the application lifetime
- Examples: Database connection pool, logger, configuration
- Owned by the `Application` struct

**Ephemeral Services** (Request-scoped)
- Initialized per HTTP request
- Exist only for request duration
- Composed from: Application state + request context
- Form dependency chains hierarchically
- Examples: ItemService, OrderService

**Key Benefits:**
- No consolidated "Services" struct that becomes brittle
- Service hierarchies emerge naturally based on use cases
- Request-scoped state flows through service chains
- Clear separation of long-running vs ephemeral concerns

### 3. Configuration-Driven Initialization

All services use `New*` constructor functions following a consistent pattern:

```go
func NewItemService(db *sql.DB, logger *slog.Logger, userID string) (*ItemService, error) {
    // 1. Finalize: Apply defaults
    if logger == nil {
        logger = slog.Default()
    }

    // 2. Validate: Check required dependencies
    if db == nil {
        return nil, errors.New("database required")
    }
    if userID == "" {
        return nil, errors.New("user ID required")
    }

    // 3. Transform: Create service instance
    return &ItemService{
        db:     db,
        logger: logger.With("service", "item", "user_id", userID),
        userID: userID,
    }, nil
}
```

**Pattern: Finalize → Validate → Transform**

This ensures:
- Validation at every initialization boundary
- Objects always in valid state
- Clear error reporting at construction time
- Consistent initialization across all components

## System Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      HTTP Requests                          │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                    Middleware                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                   │
│  │  Logger  │→ │ Recovery │→ │   Auth   │ (future)          │
│  └──────────┘  └──────────┘  └──────────┘                   │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                    Handlers                                 │
│  ┌────────────────────────────────────────────────────┐     │
│  │  Initialize Ephemeral Services per Request         │     │
│  │  (ItemService, OrderService, etc.)                 │     │
│  └────────────────────────────────────────────────────┘     │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│              Ephemeral Services                             │
│  ┌──────────────────────────────────────────────────┐       │
│  │  Business Logic (Queries + Commands)             │       │
│  │  - Validation                                    │       │
│  │  - Transaction Management                        │       │
│  │  - Domain Logic                                  │       │
│  └──────────────────────────────────────────────────┘       │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                    Models                                   │
│  ┌──────────────────────────────────────────────────┐       │
│  │  Pure Data Structures                            │       │
│  │  - Entities                                      │       │
│  │  - Commands                                      │       │
│  │  - Filters                                       │       │
│  └──────────────────────────────────────────────────┘       │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                  Database (PostgreSQL)                      │
└─────────────────────────────────────────────────────────────┘
```

### Application Struct (Long-Running State)

```go
type Application struct {
    config *config.Config
    logger *slog.Logger
    db     *sql.DB
}
```

The Application struct:
- Holds long-running dependencies
- Lives for the entire server lifetime
- Passed to handlers at initialization
- Provides accessor methods for dependencies

### Models (Pure Data Structures)

Models define only data structures with no methods:

```go
// Entity
type Item struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// Command
type CreateItemCommand struct {
    Name        string
    Description string
}

// Filter
type ItemFilters struct {
    Name   string
    Search string
}
```

**Responsibilities:**
- Define structure of domain data
- JSON serialization tags
- Database mapping (through query patterns)
- NO business logic
- NO database operations
- NO validation logic

### Services (Ephemeral, Request-Scoped)

Services encapsulate business logic and contain both queries (read operations) and commands (write operations):

```go
type ItemService struct {
    db     *sql.DB
    logger *slog.Logger
    userID string  // Request context
}

// Queries (read-only operations)
func (s *ItemService) Get(ctx context.Context, id string) (*models.Item, error)
func (s *ItemService) List(ctx context.Context, filters models.ItemFilters) ([]*models.Item, error)

// Commands (write operations with transactions)
func (s *ItemService) Create(ctx context.Context, cmd models.CreateItemCommand) (*models.Item, error)
func (s *ItemService) Update(ctx context.Context, id string, cmd models.UpdateItemCommand) (*models.Item, error)
func (s *ItemService) Delete(ctx context.Context, id string) error
```

**Queries vs Commands (Conceptual Distinction):**

- **Queries**: Read operations, no mutations, can be cached
- **Commands**: Write operations, always use transactions, mutate state

This is a **conceptual pattern**, not a structural requirement. Services are not split into separate QueryService/CommandService - they're unified per domain.

**Command Pattern:**
```go
func (s *ItemService) Create(ctx context.Context, cmd models.CreateItemCommand) (*models.Item, error) {
    // Begin transaction
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Validate command
    if err := s.validateCreate(cmd); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    // Execute atomic mutation
    query := `
        INSERT INTO items (id, name, description, created_at, updated_at)
        VALUES (gen_random_uuid(), $1, $2, NOW(), NOW())
        RETURNING id, name, description, created_at, updated_at`

    var item models.Item
    err = tx.QueryRowContext(ctx, query, cmd.Name, cmd.Description).Scan(
        &item.ID, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt)
    if err != nil {
        return nil, fmt.Errorf("insert item: %w", err)
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("commit transaction: %w", err)
    }

    s.logger.Info("item created", "item_id", item.ID, "name", item.Name)
    return &item, nil
}
```

**Services Beyond SQL:**

Not all services interact with databases:
```go
// NotificationService - external API calls
type NotificationService struct {
    emailClient *smtp.Client
    logger      *slog.Logger
}

// CacheService - Redis or in-memory cache
type CacheService struct {
    client *redis.Client
    logger *slog.Logger
}
```

### Handlers (HTTP Layer)

Handlers are dedicated structs per domain resource:

```go
type ItemHandler struct {
    app *server.Application
}

func NewItemHandler(app *server.Application) *ItemHandler {
    return &ItemHandler{app: app}
}

func (h *ItemHandler) Create(w http.ResponseWriter, r *http.Request) {
    var cmd models.CreateItemCommand
    if err := readJSON(r, &cmd); err != nil {
        h.clientError(w, http.StatusBadRequest)
        return
    }

    ctx := r.Context()
    userID := "system" // Will come from auth middleware

    // Initialize ephemeral service for this request
    svc, err := services.NewItemService(h.app.DB(), h.app.Logger(), userID)
    if err != nil {
        h.serverError(w, r, err)
        return
    }

    // Execute command
    item, err := svc.Create(ctx, cmd)
    if err != nil {
        h.serverError(w, r, err)
        return
    }

    respondJSON(w, http.StatusCreated, envelope{"item": item})
}
```

**Benefits:**
- Handlers only access what they need (Application dependencies)
- Clear per-domain organization
- Easier testing (mock Application interface)
- Aligns with single responsibility principle

### Routing (http.ServeMux)

Using Go 1.22+ standard library routing:

```go
func (app *application) routes() http.Handler {
    mux := http.NewServeMux()

    // Health check
    mux.HandleFunc("GET /health", healthHandler)

    // Item endpoints
    itemHandler := handlers.NewItemHandler(app)
    mux.HandleFunc("GET /api/items", itemHandler.List)
    mux.HandleFunc("GET /api/items/{id}", itemHandler.Get)
    mux.HandleFunc("POST /api/items", itemHandler.Create)
    mux.HandleFunc("PUT /api/items/{id}", itemHandler.Update)
    mux.HandleFunc("DELETE /api/items/{id}", itemHandler.Delete)

    // Wrap with middleware
    return app.recoverPanic(app.logRequest(mux))
}
```

**Path Parameters:**
```go
id := r.PathValue("id")  // From URL pattern /api/items/{id}
```

**Why http.ServeMux:**
- Zero dependencies
- Go 1.22+ has method and path parameter support
- Sufficient for RESTful APIs
- Native to Go ecosystem

If blocking limitations emerge, re-evaluate with chi or httprouter.

## Configuration Management

### Layered Configuration Loading

Configuration is loaded in priority order:

1. `config.yaml` - Base defaults
2. `config.{ENV}.yaml` - Environment-specific (ENV=development, production, staging, etc.)
3. `config.local.yaml` - Local overrides (gitignored)
4. Environment variables - Highest priority

### Configuration Structure

```go
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Database DatabaseConfig `yaml:"database"`
    Logging  LoggingConfig  `yaml:"logging"`
}

type ServerConfig struct {
    Port         int           `yaml:"port"`
    Host         string        `yaml:"host"`
    ReadTimeout  time.Duration `yaml:"read_timeout"`
    WriteTimeout time.Duration `yaml:"write_timeout"`
    IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

type DatabaseConfig struct {
    Host        string        `yaml:"host"`
    Port        int           `yaml:"port"`
    Database    string        `yaml:"database"`
    User        string        `yaml:"user"`
    Password    string        `yaml:"password"`
    MaxConns    int           `yaml:"max_conns"`
    MinConns    int           `yaml:"min_conns"`
    MaxIdleTime time.Duration `yaml:"max_idle_time"`
}

type LoggingConfig struct {
    Level  string `yaml:"level"`   // debug, info, warn, error
    Format string `yaml:"format"`  // text, json
}
```

### Environment Variable Convention

Environment variables mirror the YAML structure using underscores:

**Simple values:**
```bash
server_port=8080
database_host=localhost
database_password=secret
logging_level=debug
```

**Arrays (indexed convention - Kubernetes pattern):**
```bash
cors_origins_0=http://localhost:3000
cors_origins_1=http://localhost:4000
```

**Nested objects in arrays:**
```bash
database_replicas_0_host=db1
database_replicas_0_port=5432
database_replicas_0_weight=1

database_replicas_1_host=db2
database_replicas_1_port=5432
database_replicas_1_weight=2
```

**Rationale:**
- Mirrors YAML structure (intuitive mapping)
- Self-documenting (clear what each variable controls)
- Scales naturally (adding fields is straightforward)
- Standard Kubernetes/cloud-native convention
- Supports complex nested arrays

### External Configuration Sources

Environment variables work implicitly across deployment scenarios:

- **Local Development**: `.env` file or shell export
- **Docker**: Environment variables in docker-compose.yml
- **Kubernetes**: ConfigMaps and Secrets mounted as environment variables
- **Cloud Platforms**: Platform-provided environment variables

No explicit external config loading needed - standard environment variable mechanism handles all cases.

## Database Architecture

### Connection Management

Using `database/sql` with pgx driver (raw SQL approach):

```go
func Open(cfg DatabaseConfig) (*sql.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
        cfg.Host, cfg.Port, cfg.Database, cfg.User, cfg.Password)

    db, err := sql.Open("pgx", dsn)
    if err != nil {
        return nil, fmt.Errorf("open database: %w", err)
    }

    // Configure connection pool
    db.SetMaxOpenConns(cfg.MaxConns)
    db.SetMaxIdleConns(cfg.MinConns)
    db.SetConnMaxIdleTime(cfg.MaxIdleTime)

    // Verify connection
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        return nil, fmt.Errorf("ping database: %w", err)
    }

    return db, nil
}
```

### Query Patterns

**Single Row Query:**
```go
query := `SELECT id, name, description FROM items WHERE id = $1`
var item Item
err := db.QueryRowContext(ctx, query, id).Scan(&item.ID, &item.Name, &item.Description)
if err == sql.ErrNoRows {
    return nil, ErrNotFound
}
```

**Multiple Rows Query:**
```go
query := `SELECT id, name, description FROM items ORDER BY created_at DESC`
rows, err := db.QueryContext(ctx, query)
if err != nil {
    return nil, err
}
defer rows.Close()

items := []*Item{}
for rows.Next() {
    var item Item
    if err := rows.Scan(&item.ID, &item.Name, &item.Description); err != nil {
        return nil, err
    }
    items = append(items, &item)
}
return items, rows.Err()
```

**Transaction Pattern:**
```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

// Execute mutations within transaction
_, err = tx.ExecContext(ctx, query, args...)
if err != nil {
    return err
}

return tx.Commit()
```

### Why Raw SQL?

- **Full control**: See exactly what queries execute
- **Performance**: No ORM overhead
- **Learning**: Understand database/sql patterns at low level
- **Simplicity**: Minimal dependencies (just driver)
- **Flexibility**: Can add query builders (squirrel, goqu) later if needed

Future evolution may introduce query building patterns similar to the .NET ProjectionMap/QueryBuilder approach, but only when complexity justifies it.

### Database Migrations

SQL migrations are stored in `/migrations` directory:

```
migrations/
├── 000001_create_items.up.sql
├── 000001_create_items.down.sql
├── 000002_add_users.up.sql
└── 000002_add_users.down.sql
```

**Migration tools**: Use `golang-migrate/migrate` or similar tool (to be configured during development).

## Document Processing Architecture

agent-lab integrates the **document-context** library to provide PDF processing capabilities for workflow execution. This architecture supports the core use case of document classification and analysis using LLM vision APIs.

### Integration with document-context Library

The document-context library provides low-level primitives for PDF processing:
- **Document/Page Interfaces**: Abstraction over PDF structure
- **ImageMagick Rendering**: High-quality page-to-image conversion
- **Filesystem Caching**: SHA-256 keyed cache for rendered images
- **Enhancement Filters**: Brightness, contrast, saturation, rotation adjustments
- **Data URI Encoding**: Base64 encoding for LLM vision API integration

agent-lab orchestrates these primitives within the web service architecture.

### Service Structure

**BlobStorageService** (Long-Running, Application-Scoped):
- Stores uploaded PDFs and rendered images
- Abstraction layer supporting filesystem (development) and Azure Blob (production)
- Initialized at startup from configuration
- See "Blob Storage Architecture" section for details

**CacheService** (Long-Running, Application-Scoped):
- Wraps document-context filesystem cache
- Configured directory: `.data/cache/images`
- SHA-256 content-based keying for deduplication
- Lifecycle: cache entries persist across application restarts

**DocumentProcessingService** (Ephemeral, Request-Scoped):
- Coordinates PDF upload, storage, and rendering operations
- Dependencies: BlobStorageService, CacheService, Logger
- Initialized per request with request context
- Uses document-context library for PDF processing

### PDF Upload and Processing Flow

**1. Upload**
```go
func (h *DocumentHandler) Upload(w http.ResponseWriter, r *http.Request) {
    file, _, err := r.FormFile("document")
    if err != nil {
        h.clientError(w, http.StatusBadRequest)
        return
    }
    defer file.Close()

    pdfData, err := io.ReadAll(file)
    if err != nil {
        h.serverError(w, r, err)
        return
    }

    ctx := r.Context()
    userID := "system"

    svc, err := services.NewDocumentProcessingService(
        h.app.BlobStorage(),
        h.app.Cache(),
        h.app.Logger(),
        userID,
    )
    if err != nil {
        h.serverError(w, r, err)
        return
    }

    doc, err := svc.ProcessUpload(ctx, pdfData)
    if err != nil {
        h.serverError(w, r, err)
        return
    }

    respondJSON(w, http.StatusCreated, envelope{"document": doc})
}
```

**2. Storage and Metadata Extraction**
```go
func (s *DocumentProcessingService) ProcessUpload(
    ctx context.Context,
    pdfData []byte,
) (*models.Document, error) {
    docID := generateID()

    if err := s.blobStorage.Store(ctx, docID, pdfData, nil); err != nil {
        return nil, fmt.Errorf("store PDF: %w", err)
    }

    doc, err := document.OpenPDF(pdfData)
    if err != nil {
        return nil, fmt.Errorf("open PDF: %w", err)
    }
    defer doc.Close()

    metadata := &models.Document{
        ID:        docID,
        PageCount: doc.PageCount(),
        UploadedAt: time.Now(),
    }

    if err := s.persistMetadata(ctx, metadata); err != nil {
        return nil, fmt.Errorf("persist metadata: %w", err)
    }

    return metadata, nil
}
```

**3. On-Demand Page Rendering**
```go
func (s *DocumentProcessingService) RenderPage(
    ctx context.Context,
    documentID string,
    pageNum int,
    filterOverrides map[string]interface{},
) ([]byte, error) {
    pdfData, err := s.blobStorage.Retrieve(ctx, documentID)
    if err != nil {
        return nil, fmt.Errorf("retrieve PDF: %w", err)
    }

    doc, err := document.OpenPDF(pdfData)
    if err != nil {
        return nil, fmt.Errorf("open PDF: %w", err)
    }
    defer doc.Close()

    page, err := doc.ExtractPage(pageNum)
    if err != nil {
        return nil, fmt.Errorf("extract page: %w", err)
    }

    renderer, err := s.createRenderer(ctx, documentID, filterOverrides)
    if err != nil {
        return nil, fmt.Errorf("create renderer: %w", err)
    }

    imageData, err := page.ToImage(renderer, s.cache)
    if err != nil {
        return nil, fmt.Errorf("render page: %w", err)
    }

    return imageData, nil
}
```

### Enhancement Filter Configuration

Enhancement filters can be configured at two levels:

**Per-Workflow Defaults** (stored in workflow configuration):
```yaml
image_processing:
  format: "png"
  dpi: 300
  quality: 85
  filters:
    brightness: 110
    contrast: 10
    saturation: 100
    rotation: 0
```

**Per-Execution Overrides** (provided in execution request):
```json
{
  "workflow_id": "classify-docs",
  "document_id": "abc123",
  "filter_overrides": {
    "page_2": {
      "brightness": 130,
      "contrast": 25
    }
  }
}
```

**Use Case**: Re-analyze specific pages with adjusted filters to clarify faded stamps or low-contrast markings (e.g., NOFORN stamp).

### Image Cache Lifecycle

**Cache Key**: SHA-256 hash of (PDF content + page number + render settings)

**Lifecycle**:
- Rendered images cached on first render
- Cache persists across application restarts
- Cache tied to execution run lifecycle
- When execution run deleted, associated cached images deleted
- Managed through `execution_cache_entries` table linking runs to cache keys

**Cache Directory Structure**:
```
.data/cache/images/
├── ab/
│   └── cd/
│       └── abcd1234...ef56.png
└── 12/
    └── 34/
        └── 1234abcd...ef78.png
```

### ImageMagick Deployment

**Requirement**: ImageMagick 7+ must be available in container.

**Container Setup** (Dockerfile):
```dockerfile
FROM golang:1.25.2-alpine AS builder
# ... build steps ...

FROM alpine:latest
RUN apk add --no-cache imagemagick
COPY --from=builder /app/agent-lab /app/agent-lab
CMD ["/app/agent-lab"]
```

**Startup Verification**:
```go
func verifyImageMagick() error {
    path, err := exec.LookPath("magick")
    if err != nil {
        return fmt.Errorf("ImageMagick not found: %w", err)
    }

    cmd := exec.Command(path, "-version")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("ImageMagick version check failed: %w", err)
    }

    log.Printf("ImageMagick found at %s:\n%s", path, output)
    return nil
}
```

**Version Check**: Application fails fast at startup if ImageMagick unavailable.

### Data URI Encoding for Vision APIs

```go
func (s *WorkflowExecutionService) analyzePageWithVision(
    ctx context.Context,
    imageData []byte,
    prompt string,
) (string, error) {
    dataURI, err := encoding.EncodeImageDataURI(imageData, document.PNG)
    if err != nil {
        return "", fmt.Errorf("encode data URI: %w", err)
    }

    response, err := s.agent.Vision(ctx, []string{dataURI}, prompt, nil)
    if err != nil {
        return "", fmt.Errorf("vision API call: %w", err)
    }

    return response.Content(), nil
}
```

## Blob Storage Architecture

agent-lab requires storage for uploaded PDFs. The blob storage architecture provides a simple abstraction layer supporting both local filesystem (development) and Azure Blob Storage (production).

### Storage Strategy

**PDFs** (Persistent):
- Stored in blob storage until user explicitly deletes document
- Enables re-execution with different parameters
- Supports iterative workflow refinement

**Rendered Images** (Ephemeral, Cached):
- Managed by document-context FilesystemCache
- SHA-256 keyed cache with structure: `<cache_root>/<key>/<filename>`
- Example: `.data/cache/images/a3b5c7.../document.1.png`
- Cache keys tracked in database, tied to execution runs
- When execution run deleted, associated cache keys invalidated

**Agent Configurations** (Database):
- No blob storage needed - stored as JSON in database
- Efficient querying and filtering
- Versioning through database records

**No Metadata Files**: Blob storage stores only binary data (PDFs). All metadata stored in database.

### BlobStorage Interface

```go
type BlobStorage interface {
    Store(ctx context.Context, key string, data []byte) error
    Retrieve(ctx context.Context, key string) ([]byte, error)
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
}
```

**Methods**:
- `Store`: Persist blob (replaces existing)
- `Retrieve`: Fetch blob data
- `Delete`: Remove blob permanently
- `Exists`: Check blob existence without retrieval

**Simplified Design**: No metadata parameter - all metadata stored in database.

### FilesystemBlobStorage Implementation

**Development and On-Premises Deployment**

```go
type FilesystemBlobStorage struct {
    baseDir string
    logger  *slog.Logger
}

func NewFilesystemBlobStorage(cfg FilesystemConfig) (*FilesystemBlobStorage, error) {
    if cfg.Directory == "" {
        return nil, errors.New("directory required")
    }

    if err := os.MkdirAll(cfg.Directory, 0755); err != nil {
        return nil, fmt.Errorf("create directory: %w", err)
    }

    return &FilesystemBlobStorage{
        baseDir: cfg.Directory,
        logger:  cfg.Logger,
    }, nil
}

func (s *FilesystemBlobStorage) Store(ctx context.Context, key string, data []byte) error {
    blobPath := filepath.Join(s.baseDir, key)

    if err := os.MkdirAll(filepath.Dir(blobPath), 0755); err != nil {
        return fmt.Errorf("create directory: %w", err)
    }

    tempPath := blobPath + ".tmp"
    if err := os.WriteFile(tempPath, data, 0644); err != nil {
        return fmt.Errorf("write temp file: %w", err)
    }

    if err := os.Rename(tempPath, blobPath); err != nil {
        os.Remove(tempPath)
        return fmt.Errorf("atomic rename: %w", err)
    }

    s.logger.Debug("blob stored", "key", key, "size", len(data))
    return nil
}

func (s *FilesystemBlobStorage) Retrieve(ctx context.Context, key string) ([]byte, error) {
    blobPath := filepath.Join(s.baseDir, key)

    data, err := os.ReadFile(blobPath)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            return nil, ErrBlobNotFound
        }
        return nil, fmt.Errorf("read blob: %w", err)
    }

    s.logger.Debug("blob retrieved", "key", key, "size", len(data))
    return data, nil
}

func (s *FilesystemBlobStorage) Delete(ctx context.Context, key string) error {
    blobPath := filepath.Join(s.baseDir, key)

    err := os.Remove(blobPath)
    if err != nil && !errors.Is(err, os.ErrNotExist) {
        return fmt.Errorf("delete blob: %w", err)
    }

    s.logger.Debug("blob deleted", "key", key)
    return nil
}

func (s *FilesystemBlobStorage) Exists(ctx context.Context, key string) (bool, error) {
    blobPath := filepath.Join(s.baseDir, key)

    _, err := os.Stat(blobPath)
    if err == nil {
        return true, nil
    }
    if errors.Is(err, os.ErrNotExist) {
        return false, nil
    }

    return false, fmt.Errorf("stat blob: %w", err)
}
```

**Directory Structure**:
```
.data/blobs/
└── documents/
    ├── abc123
    ├── def456
    └── ghi789
```

**Key Pattern**: `documents/{documentID}`

**Atomic Writes**: Temp file + rename ensures consistency.

**Sentinel Error**:
```go
var ErrBlobNotFound = errors.New("blob not found")
```

### AzureBlobStorage Implementation

**Production Cloud Deployment (Phase 8)**

```go
type AzureBlobStorage struct {
    client    *azblob.Client
    container string
    logger    *slog.Logger
}

func NewAzureBlobStorage(cfg AzureConfig) (*AzureBlobStorage, error) {
    var client *azblob.Client
    var err error

    switch cfg.AuthType {
    case "managed_identity":
        cred, err := azidentity.NewDefaultAzureCredential(nil)
        if err != nil {
            return nil, fmt.Errorf("managed identity: %w", err)
        }

        serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net", cfg.Account)
        client, err = azblob.NewClient(serviceURL, cred, nil)
        if err != nil {
            return nil, fmt.Errorf("create client: %w", err)
        }

    case "connection_string":
        client, err = azblob.NewClientFromConnectionString(cfg.ConnectionString, nil)
        if err != nil {
            return nil, fmt.Errorf("connection string: %w", err)
        }

    default:
        return nil, fmt.Errorf("unknown auth type: %s", cfg.AuthType)
    }

    return &AzureBlobStorage{
        client:    client,
        container: cfg.Container,
        logger:    cfg.Logger,
    }, nil
}

func (s *AzureBlobStorage) Store(ctx context.Context, key string, data []byte) error {
    blobClient := s.client.ServiceClient().NewContainerClient(s.container).NewBlockBlobClient(key)

    _, err := blobClient.UploadBuffer(ctx, data, nil)
    if err != nil {
        return fmt.Errorf("upload blob: %w", err)
    }

    s.logger.Debug("blob stored", "key", key, "size", len(data), "container", s.container)
    return nil
}

func (s *AzureBlobStorage) Retrieve(ctx context.Context, key string) ([]byte, error) {
    blobClient := s.client.ServiceClient().NewContainerClient(s.container).NewBlockBlobClient(key)

    resp, err := blobClient.DownloadStream(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("download blob: %w", err)
    }
    defer resp.Body.Close()

    data, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("read blob data: %w", err)
    }

    s.logger.Debug("blob retrieved", "key", key, "size", len(data), "container", s.container)
    return data, nil
}
```

**Azure Features**:
- **Managed Identity**: No credentials in configuration
- **Container-Based Organization**: Single container, key-based paths
- **Lifecycle Policies**: Auto-archival/deletion via Azure policies (Phase 8)

### Metadata Management in Database

All blob metadata stored in database, not blob storage:

**documents table**:
```sql
CREATE TABLE documents (
    id TEXT PRIMARY KEY,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    page_count INTEGER NOT NULL,
    blob_key TEXT NOT NULL,
    uploaded_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP
);
```

**execution_cache_entries table**:
```sql
CREATE TABLE execution_cache_entries (
    run_id TEXT NOT NULL,
    cache_key TEXT NOT NULL,
    page_num INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    PRIMARY KEY (run_id, cache_key)
);
```

**Rationale**:
- Database provides rich querying and filtering
- Consistent metadata access across storage implementations
- Simplified cache lifecycle management
- No need to parse/store metadata in blob properties

### Configuration Pattern

```yaml
blob_storage:
  type: "filesystem"
  filesystem:
    directory: "./.data/blobs"
  azure:
    account: "agentlabstorage"
    container: "agent-lab"
    auth_type: "managed_identity"
```

**Environment Variable Override**:
```bash
blob_storage_type=azure
blob_storage_azure_account=agentlabstorage
blob_storage_azure_container=agent-lab
blob_storage_azure_auth_type=managed_identity
```

**Initialization**:
```go
func NewBlobStorage(cfg BlobStorageConfig) (BlobStorage, error) {
    switch cfg.Type {
    case "filesystem":
        return NewFilesystemBlobStorage(cfg.Filesystem)
    case "azure":
        return NewAzureBlobStorage(cfg.Azure)
    default:
        return nil, fmt.Errorf("unknown blob storage type: %s", cfg.Type)
    }
}
```

### Integration with document-context Cache

**PDF Upload and Storage**:
```go
func (s *DocumentProcessingService) ProcessUpload(
    ctx context.Context,
    pdfData []byte,
    filename string,
) (*models.Document, error) {
    docID := generateID()
    blobKey := fmt.Sprintf("documents/%s", docID)

    if err := s.blobStorage.Store(ctx, blobKey, pdfData); err != nil {
        return nil, fmt.Errorf("store PDF: %w", err)
    }

    doc, err := document.OpenPDF(pdfData)
    if err != nil {
        return nil, fmt.Errorf("open PDF: %w", err)
    }
    defer doc.Close()

    metadata := &models.Document{
        ID:          docID,
        Filename:    filename,
        ContentType: "application/pdf",
        SizeBytes:   len(pdfData),
        PageCount:   doc.PageCount(),
        BlobKey:     blobKey,
        UploadedAt:  time.Now(),
    }

    if err := s.persistMetadata(ctx, metadata); err != nil {
        return nil, fmt.Errorf("persist metadata: %w", err)
    }

    return metadata, nil
}
```

**Image Rendering with Cache Tracking**:
```go
func (s *WorkflowExecutionService) RenderPageWithCache(
    ctx context.Context,
    runID string,
    documentID string,
    pageNum int,
    renderer image.Renderer,
) ([]byte, error) {
    pdfData, err := s.blobStorage.Retrieve(ctx, fmt.Sprintf("documents/%s", documentID))
    if err != nil {
        return nil, fmt.Errorf("retrieve PDF: %w", err)
    }

    doc, err := document.OpenPDF(pdfData)
    if err != nil {
        return nil, fmt.Errorf("open PDF: %w", err)
    }
    defer doc.Close()

    page, err := doc.ExtractPage(pageNum)
    if err != nil {
        return nil, fmt.Errorf("extract page: %w", err)
    }

    imageData, err := page.ToImage(renderer, s.cache)
    if err != nil {
        return nil, fmt.Errorf("render page: %w", err)
    }

    cacheKey, err := buildCacheKey(documentID, pageNum, renderer.Settings())
    if err != nil {
        return nil, fmt.Errorf("build cache key: %w", err)
    }

    if err := s.trackCacheEntry(ctx, runID, cacheKey, pageNum); err != nil {
        s.logger.Error("failed to track cache entry", "error", err)
    }

    return imageData, nil
}

func (s *WorkflowExecutionService) trackCacheEntry(
    ctx context.Context,
    runID string,
    cacheKey string,
    pageNum int,
) error {
    query := `
        INSERT INTO execution_cache_entries (run_id, cache_key, page_num, created_at)
        VALUES ($1, $2, $3, NOW())
        ON CONFLICT (run_id, cache_key) DO NOTHING`

    _, err := s.db.ExecContext(ctx, query, runID, cacheKey, pageNum)
    return err
}
```

### Lifecycle Management

**Document Deletion (Soft Delete)**:
```go
func (s *DocumentService) Delete(ctx context.Context, documentID string) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback()

    var blobKey string
    query := `
        UPDATE documents
        SET deleted_at = NOW()
        WHERE id = $1 AND deleted_at IS NULL
        RETURNING blob_key`

    err = tx.QueryRowContext(ctx, query, documentID).Scan(&blobKey)
    if err == sql.ErrNoRows {
        return ErrNotFound
    }
    if err != nil {
        return fmt.Errorf("mark deleted: %w", err)
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("commit transaction: %w", err)
    }

    if err := s.blobStorage.Delete(ctx, blobKey); err != nil {
        s.logger.Error("failed to delete blob", "key", blobKey, "error", err)
    }

    return nil
}
```

**Cache Cleanup on Run Deletion**:
```go
func (s *ExecutionService) DeleteRun(ctx context.Context, runID string) error {
    var cacheKeys []string
    query := `SELECT DISTINCT cache_key FROM execution_cache_entries WHERE run_id = $1`

    rows, err := s.db.QueryContext(ctx, query, runID)
    if err != nil {
        return fmt.Errorf("query cache entries: %w", err)
    }
    defer rows.Close()

    for rows.Next() {
        var key string
        if err := rows.Scan(&key); err != nil {
            return fmt.Errorf("scan cache key: %w", err)
        }
        cacheKeys = append(cacheKeys, key)
    }

    for _, key := range cacheKeys {
        if err := s.cache.Invalidate(key); err != nil {
            s.logger.Error("failed to invalidate cache", "key", key, "error", err)
        }
    }

    query = `DELETE FROM execution_runs WHERE id = $1`
    if _, err := s.db.ExecContext(ctx, query, runID); err != nil {
        return fmt.Errorf("delete run: %w", err)
    }

    s.logger.Info("run deleted", "run_id", runID, "cache_keys_invalidated", len(cacheKeys))
    return nil
}
```

**Asynchronous Blob Deletion**: Soft delete marks record immediately, actual blob deletion is best-effort asynchronous.

## Real-Time Event Streaming

agent-lab provides real-time execution monitoring through Server-Sent Events (SSE). This architecture enables clients to observe workflow execution as it progresses without polling.

### EventBus Architecture

**EventBus** (Long-Running, Application-Scoped):
- Channel-based pub/sub messaging
- Topic-based routing: `execution:{runID}`
- Multiple subscribers per topic
- Thread-safe using sync.RWMutex

```go
type EventBus struct {
    mu          sync.RWMutex
    subscribers map[string][]chan Event
    logger      *slog.Logger
}

type Event struct {
    Type      string
    Timestamp time.Time
    RunID     string
    Data      map[string]interface{}
}

func NewEventBus(logger *slog.Logger) *EventBus {
    return &EventBus{
        subscribers: make(map[string][]chan Event),
        logger:      logger,
    }
}

func (eb *EventBus) Subscribe(ctx context.Context, topic string) <-chan Event {
    eb.mu.Lock()
    defer eb.mu.Unlock()

    ch := make(chan Event, 10)
    eb.subscribers[topic] = append(eb.subscribers[topic], ch)

    go func() {
        <-ctx.Done()
        eb.Unsubscribe(topic, ch)
    }()

    return ch
}

func (eb *EventBus) Publish(ctx context.Context, topic string, event Event) {
    eb.mu.RLock()
    defer eb.mu.RUnlock()

    for _, ch := range eb.subscribers[topic] {
        select {
        case ch <- event:
        case <-time.After(100 * time.Millisecond):
            eb.logger.Warn("event dropped", "topic", topic, "type", event.Type)
        }
    }
}
```

**Buffer Size**: 10 events per subscriber prevents slow clients from blocking publishers.

**Timeout**: 100ms timeout prevents single slow subscriber from delaying event delivery.

### SSE Endpoint Pattern

```go
func (h *ExecutionHandler) Stream(w http.ResponseWriter, r *http.Request) {
    runID := r.PathValue("id")

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
        return
    }

    topic := fmt.Sprintf("execution:%s", runID)
    events := h.app.EventBus().Subscribe(r.Context(), topic)

    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case event := <-events:
            data, err := json.Marshal(event.Data)
            if err != nil {
                h.app.Logger().Error("marshal event", "error", err)
                continue
            }

            fmt.Fprintf(w, "event: %s\n", event.Type)
            fmt.Fprintf(w, "data: %s\n\n", data)
            flusher.Flush()

            if event.Type == "execution_completed" || event.Type == "execution_failed" {
                return
            }

        case <-ticker.C:
            fmt.Fprintf(w, ": heartbeat\n\n")
            flusher.Flush()

        case <-r.Context().Done():
            return
        }
    }
}
```

**Headers**:
- `text/event-stream`: SSE content type
- `no-cache`: Prevent caching
- `keep-alive`: Maintain connection
- `X-Accel-Buffering: no`: Disable nginx buffering

**Heartbeat**: 30-second interval prevents connection timeout.

**Termination**: Stream closes on execution completion or client disconnect.

### Event Types

**Execution Lifecycle**:
- `execution_started`: Run initiated
- `execution_completed`: Run finished successfully
- `execution_failed`: Run terminated with error
- `execution_cancelled`: Run cancelled by user

**Step Progress**:
- `step_started`: Workflow step initiated
- `step_completed`: Step finished successfully
- `step_failed`: Step encountered error

**Classification-Specific**:
- `page_analyzed`: Page classification complete
- `confidence_scored`: Confidence calculation updated
- `marking_detected`: Classification marking found

### Selective Event Persistence

Not all events need database persistence. Balance observability with storage efficiency:

**Always Persist** (execution_events table):
- Execution lifecycle events (started, completed, failed, cancelled)
- Critical decision points (classification determined, confidence thresholds)
- Final results and outcomes

**Never Persist**:
- Heartbeats (connection maintenance only)
- Progress percentages (ephemeral progress indicators)
- Transient state updates (intermediate calculations)

**Configurable Per-Workflow**:
- Step transitions (detailed execution trace)
- Intermediate state snapshots (checkpoint data)
- Performance metrics (execution timing)

```go
func (s *WorkflowExecutionService) publishEvent(
    ctx context.Context,
    runID string,
    eventType string,
    data map[string]interface{},
    persist bool,
) {
    event := Event{
        Type:      eventType,
        Timestamp: time.Now(),
        RunID:     runID,
        Data:      data,
    }

    topic := fmt.Sprintf("execution:%s", runID)
    s.eventBus.Publish(ctx, topic, event)

    if persist {
        if err := s.persistEvent(ctx, event); err != nil {
            s.logger.Error("failed to persist event", "error", err)
        }
    }
}

func (s *WorkflowExecutionService) persistEvent(ctx context.Context, event Event) error {
    dataJSON, err := json.Marshal(event.Data)
    if err != nil {
        return fmt.Errorf("marshal event data: %w", err)
    }

    query := `
        INSERT INTO execution_events (run_id, event_type, event_data, created_at)
        VALUES ($1, $2, $3, $4)`

    _, err = s.db.ExecContext(ctx, query, event.RunID, event.Type, dataJSON, event.Timestamp)
    return err
}
```

**execution_events table**:
```sql
CREATE TABLE execution_events (
    id SERIAL PRIMARY KEY,
    run_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    event_data JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_execution_events_run_id ON execution_events(run_id);
CREATE INDEX idx_execution_events_created_at ON execution_events(created_at);
```

### Client-Side Integration

**Vanilla JavaScript**:
```javascript
const source = new EventSource(`/api/runs/${runID}/stream`);

source.addEventListener('step_completed', (e) => {
    const data = JSON.parse(e.data);
    updateProgress(data.step, data.status);
});

source.addEventListener('confidence_scored', (e) => {
    const data = JSON.parse(e.data);
    updateConfidenceChart(data.confidence);
});

source.addEventListener('execution_completed', (e) => {
    const data = JSON.parse(e.data);
    showResults(data.result);
    source.close();
});

source.addEventListener('error', (e) => {
    if (source.readyState === EventSource.CLOSED) {
        console.log('Connection closed');
    } else {
        console.error('SSE error', e);
        source.close();
    }
});
```

**Reconnection**: EventSource automatically reconnects on connection drop (configurable retry interval).

**Close on Completion**: Explicitly close connection when execution finishes to free resources.

## Workflow Execution Model

agent-lab uses an **async-first** execution architecture where all workflow executions are non-blocking and support real-time monitoring via SSE. This model naturally extends to bulk processing through queue-based execution.

### Async-First Execution

**Design Principle**: All executions return immediately with a `run_id`. Client monitors progress via SSE streaming or polls status endpoint.

**HTTP API**:
```go
POST /api/workflows/{id}/execute
{
  "document_id": "abc123",
  "filter_overrides": {
    "page_2": {"brightness": 130}
  }
}

Response (202 Accepted):
{
  "run_id": "run-xyz789",
  "status": "pending",
  "stream_url": "/api/runs/run-xyz789/stream"
}
```

**Status Codes**:
- `202 Accepted`: Execution enqueued successfully
- `400 Bad Request`: Invalid workflow or document ID
- `404 Not Found`: Workflow or document not found
- `503 Service Unavailable`: Queue full (backpressure)

**Why Async-First**:
- Enables real-time monitoring via SSE (core value proposition)
- Prevents HTTP timeout on long workflows
- Naturally supports bulk processing
- Allows cancellation without complex request handling
- No need for separate sync/async code paths

### Service Architecture

**ExecutionQueueService** (Long-Running, Application-Scoped):
- Background job queue (channel-based)
- Configurable queue depth (default: 1000)
- Backpressure when queue full

```go
type ExecutionQueueService struct {
    queue  chan *ExecutionJob
    logger *slog.Logger
}

type ExecutionJob struct {
    RunID      string
    WorkflowID string
    DocumentID string
    Overrides  map[string]interface{}
    Context    context.Context
}

func NewExecutionQueueService(queueDepth int, logger *slog.Logger) *ExecutionQueueService {
    return &ExecutionQueueService{
        queue:  make(chan *ExecutionJob, queueDepth),
        logger: logger,
    }
}

func (s *ExecutionQueueService) Enqueue(ctx context.Context, job *ExecutionJob) error {
    select {
    case s.queue <- job:
        s.logger.Info("job enqueued", "run_id", job.RunID)
        return nil
    case <-time.After(1 * time.Second):
        return errors.New("queue full")
    }
}

func (s *ExecutionQueueService) Dequeue(ctx context.Context) (*ExecutionJob, error) {
    select {
    case job := <-s.queue:
        return job, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

**WorkerPoolService** (Long-Running, Application-Scoped):
- Pool of goroutines consuming from queue
- Configurable worker count (default: runtime.NumCPU())
- Graceful shutdown on SIGTERM

```go
type WorkerPoolService struct {
    queue     *ExecutionQueueService
    executor  *WorkflowExecutionService
    workerCount int
    logger    *slog.Logger
    wg        sync.WaitGroup
}

func (s *WorkerPoolService) Start(ctx context.Context) {
    for i := 0; i < s.workerCount; i++ {
        s.wg.Add(1)
        go s.worker(ctx, i)
    }
}

func (s *WorkerPoolService) worker(ctx context.Context, id int) {
    defer s.wg.Done()

    s.logger.Info("worker started", "worker_id", id)
    defer s.logger.Info("worker stopped", "worker_id", id)

    for {
        job, err := s.queue.Dequeue(ctx)
        if err != nil {
            return
        }

        s.logger.Info("processing job", "worker_id", id, "run_id", job.RunID)

        if err := s.executor.Execute(job.Context, job); err != nil {
            s.logger.Error("execution failed", "run_id", job.RunID, "error", err)
        }
    }
}

func (s *WorkerPoolService) Shutdown(timeout time.Duration) error {
    done := make(chan struct{})

    go func() {
        s.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        return nil
    case <-time.After(timeout):
        return errors.New("shutdown timeout")
    }
}
```

**WorkflowExecutionService** (Ephemeral, Request-Scoped):
- Coordinates go-agents-orchestration primitives
- Publishes events to EventBus during execution
- Manages state transitions

```go
type WorkflowExecutionService struct {
    db              *sql.DB
    blobStorage     BlobStorage
    cache           cache.Cache
    eventBus        *EventBus
    agentService    *AgentService
    documentService *DocumentProcessingService
    logger          *slog.Logger
}

func (s *WorkflowExecutionService) Execute(ctx context.Context, job *ExecutionJob) error {
    if err := s.updateStatus(ctx, job.RunID, StatusRunning); err != nil {
        return fmt.Errorf("update status: %w", err)
    }

    s.publishEvent(ctx, job.RunID, "execution_started", map[string]interface{}{
        "workflow_id": job.WorkflowID,
        "document_id": job.DocumentID,
    }, true)

    result, err := s.executeWorkflow(ctx, job)
    if err != nil {
        s.updateStatus(ctx, job.RunID, StatusFailed)
        s.publishEvent(ctx, job.RunID, "execution_failed", map[string]interface{}{
            "error": err.Error(),
        }, true)
        return err
    }

    if err := s.storeResult(ctx, job.RunID, result); err != nil {
        return fmt.Errorf("store result: %w", err)
    }

    s.updateStatus(ctx, job.RunID, StatusCompleted)
    s.publishEvent(ctx, job.RunID, "execution_completed", map[string]interface{}{
        "result": result,
    }, true)

    return nil
}
```

### State Management

**ExecutionStatus** (enum):
```go
type ExecutionStatus string

const (
    StatusPending   ExecutionStatus = "pending"
    StatusRunning   ExecutionStatus = "running"
    StatusCompleted ExecutionStatus = "completed"
    StatusFailed    ExecutionStatus = "failed"
    StatusCancelled ExecutionStatus = "cancelled"
)
```

**State Transitions**:
```
pending → running → completed
pending → running → failed
pending → running → cancelled
pending → cancelled (before start)
```

**execution_runs table**:
```sql
CREATE TABLE execution_runs (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL,
    document_id TEXT NOT NULL,
    status TEXT NOT NULL,
    input JSONB,
    result JSONB,
    error TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY (workflow_id) REFERENCES workflows(id),
    FOREIGN KEY (document_id) REFERENCES documents(id)
);

CREATE INDEX idx_execution_runs_status ON execution_runs(status);
CREATE INDEX idx_execution_runs_created_at ON execution_runs(created_at DESC);
CREATE INDEX idx_execution_runs_workflow_id ON execution_runs(workflow_id);
```

### Context Cancellation

**User-Initiated Cancellation**:
```go
DELETE /api/runs/{id}

func (h *ExecutionHandler) Cancel(w http.ResponseWriter, r *http.Request) {
    runID := r.PathValue("id")

    if err := h.executionService.Cancel(r.Context(), runID); err != nil {
        if errors.Is(err, ErrNotFound) {
            h.notFound(w)
            return
        }
        h.serverError(w, r, err)
        return
    }

    respondJSON(w, http.StatusNoContent, nil)
}

func (s *ExecutionService) Cancel(ctx context.Context, runID string) error {
    // Cancel context for this run
    s.cancelRun(runID)

    // Update status
    query := `
        UPDATE execution_runs
        SET status = $1, completed_at = NOW()
        WHERE id = $2 AND status IN ($3, $4)`

    result, err := s.db.ExecContext(ctx, query, StatusCancelled, runID, StatusPending, StatusRunning)
    if err != nil {
        return fmt.Errorf("update status: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("rows affected: %w", err)
    }

    if rows == 0 {
        return ErrNotFound
    }

    s.publishEvent(ctx, runID, "execution_cancelled", nil, true)
    return nil
}
```

**Context Propagation**:
- Each execution job carries context
- Context cancellation propagates through workflow steps
- go-agents-orchestration respects context cancellation
- Cleanup on cancellation (partial results discarded)

### Bulk Processing

**Natural Extension** of async architecture:
```go
POST /api/workflows/{id}/execute/bulk
{
  "documents": ["doc1", "doc2", "doc3"],
  "overrides": {}
}

Response (202 Accepted):
{
  "batch_id": "batch-abc123",
  "run_ids": ["run-1", "run-2", "run-3"],
  "status_url": "/api/batches/batch-abc123"
}
```

**Implementation**: Multiple `Enqueue` calls, one per document. No separate bulk processing logic needed.

### Future: Checkpointing (Phase 6+)

go-agents-orchestration state graphs support checkpointing for long workflows:

- Checkpoint states stored in `execution_checkpoints` table
- Enables pause/resume for long workflows
- Recovery from worker failures
- Partial execution replay

**Not in MVP**: Simple workflows (classification) don't require checkpointing.

## Integration with go-agents Ecosystem

agent-lab integrates three libraries to provide flexible workflow orchestration capabilities: go-agents (LLM integration), go-agents-orchestration (workflow patterns), and document-context (PDF processing). The platform provides building blocks for experimentation rather than prescribing specific workflow implementations.

### go-agents Integration

**Agent Configuration and Caching**:

Agent configurations stored as JSON in database with in-memory caching.

```go
type AgentService struct {
    db     *sql.DB
    logger *slog.Logger
    mu     sync.RWMutex
    cache  map[string]*agent.Agent
}

func (s *AgentService) GetAgent(ctx context.Context, agentID string) (*agent.Agent, error) {
    s.mu.RLock()
    if cached, ok := s.cache[agentID]; ok {
        s.mu.RUnlock()
        return cached, nil
    }
    s.mu.RUnlock()

    var configJSON []byte
    query := `SELECT config FROM agents WHERE id = $1`
    err := s.db.QueryRowContext(ctx, query, agentID).Scan(&configJSON)
    if err != nil {
        return nil, fmt.Errorf("load agent config: %w", err)
    }

    var cfg agent.Config
    if err := json.Unmarshal(configJSON, &cfg); err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }

    a, err := agent.New(&cfg)
    if err != nil {
        return nil, fmt.Errorf("create agent: %w", err)
    }

    s.mu.Lock()
    s.cache[agentID] = a
    s.mu.Unlock()

    return a, nil
}
```

**Runtime Option Overrides**:

Agent configs provide baseline defaults. Execution requests override per-run:

```go
POST /api/workflows/{id}/execute
{
  "document_id": "abc123",
  "agent_options": {
    "temperature": 0.2,
    "max_tokens": 1500
  },
  "filter_overrides": {
    "page_2": {"brightness": 130, "contrast": 25}
  }
}
```

**Flexible Agent Invocation**:
```go
response, err := agent.Vision(ctx, []string{imageDataURI}, prompt, executionOptions)
```

Options merge: agent defaults + execution overrides → go-agents handles merging automatically.

**Multi-Provider Support**: Ollama (development), Azure AI Foundry (production), or custom providers.

### go-agents-orchestration Integration

**Workflow Pattern Primitives**:

The orchestration library provides composable patterns for building workflows.

**Sequential Processing** (context accumulation):
```go
type WorkflowContext struct {
    State map[string]interface{}
    Steps []StepResult
}

processor := func(ctx context.Context, input string, context WorkflowContext) (WorkflowContext, error) {
    result, err := performStep(ctx, input, context.State)
    if err != nil {
        return context, err
    }

    context.Steps = append(context.Steps, result)
    context.State["last_result"] = result
    return context, nil
}

cfg := config.DefaultSequentialConfig()
final, err := workflows.ProcessChain(ctx, cfg, inputs, initialContext, processor, nil)
```

**Conditional Routing** (decision-based flow):
```go
predicate := func(s WorkflowContext) (string, error) {
    confidence := s.State["confidence"].(float64)
    if confidence >= 0.95 {
        return "high_confidence", nil
    }
    if confidence >= 0.80 {
        return "medium_confidence", nil
    }
    return "low_confidence", nil
}

routes := workflows.Routes[WorkflowContext]{
    Handlers: map[string]workflows.RouteHandler[WorkflowContext]{
        "high_confidence": acceptHandler,
        "medium_confidence": reviewHandler,
        "low_confidence": reprocessHandler,
    },
}

result, err := workflows.ProcessConditional(ctx, cfg, initialState, predicate, routes)
```

**Parallel Execution** (concurrent processing):
```go
processor := func(ctx context.Context, page int) (PageResult, error) {
    return analyzePage(ctx, page)
}

cfg := config.DefaultParallelConfig()
results, err := workflows.ProcessParallel(ctx, cfg, pages, processor, nil)
```

**Composable Workflow Example** (multi-stage with loops):

Workflows can combine patterns for complex flows:

```go
// Stage 1: Identify markings (sequential per page)
markings, err := workflows.ProcessChain(ctx, cfg, pages, initialCtx, identifyMarkings, nil)

// Stage 2: Conditional routing based on clarity
for _, marking := range markings.Final.Markings {
    if marking.Clarity < 0.7 {
        // Request re-render with adjusted filters
        newFilters := calculateFilters(marking.IssueType)
        rerendered, err := rerenderWithFilters(ctx, marking.Page, newFilters)
        // Re-analyze with clearer image
        updatedMarking, err := analyzeMarking(ctx, rerendered)
    }
}

// Stage 3: Classification analysis (aggregates marking results)
classification, err := analyzeMarkings(ctx, markings.Final)

// Stage 4: QA validation with feedback loop
for {
    qaResult, err := qaAgent.Validate(ctx, classification)
    if qaResult.Approved {
        break
    }

    // QA rejected - loop back to earlier stage
    markings, err = workflows.ProcessChain(ctx, cfg, qaResult.PagesThatNeedReanalysis, markings.Final, identifyMarkings, nil)
    classification, err = analyzeMarkings(ctx, markings.Final)
}
```

**Key Capabilities**:
- State flows between stages
- Agents can request adaptive re-processing
- Conditional routing based on confidence/quality thresholds
- Feedback loops for iterative refinement
- Event publishing at each stage for observability

### document-context Integration

**Adaptive Image Processing**:

Agents can request re-renders with adjusted filters for problematic pages.

```go
func (s *WorkflowExecutionService) RenderPageWithOptions(
    ctx context.Context,
    documentID string,
    pageNum int,
    workflowDefaults config.ImageConfig,
    runtimeOverrides map[string]interface{},
) ([]byte, error) {
    cfg := workflowDefaults

    // Apply runtime overrides (agent-requested adjustments)
    if brightness, ok := runtimeOverrides["brightness"].(float64); ok {
        b := int(brightness)
        cfg.Options["brightness"] = &b
    }
    if contrast, ok := runtimeOverrides["contrast"].(float64); ok {
        c := int(contrast)
        cfg.Options["contrast"] = &c
    }

    renderer, err := image.NewImageMagickRenderer(cfg)
    if err != nil {
        return nil, err
    }

    // Retrieve PDF from blob storage
    pdfData, err := s.blobStorage.Retrieve(ctx, fmt.Sprintf("documents/%s", documentID))
    if err != nil {
        return nil, err
    }

    doc, err := document.OpenPDF(pdfData)
    if err != nil {
        return nil, err
    }
    defer doc.Close()

    page, err := doc.ExtractPage(pageNum)
    if err != nil {
        return nil, err
    }

    // Render with caching (cache key includes filter settings)
    return page.ToImage(renderer, s.cache)
}
```

**Agent-Driven Filter Adjustment**:

Agent analyzes image, identifies faded marking, requests enhanced version:

```json
{
  "marking_type": "classification_stamp",
  "clarity": 0.4,
  "issue": "faded_text",
  "request_rerender": true,
  "suggested_filters": {
    "brightness": 90,
    "contrast": 30
  }
}
```

Workflow executor re-renders with suggested filters, provides enhanced image for re-analysis.

**Cache Lifecycle**: Each render configuration generates unique cache key. Multiple filter combinations coexist in cache, tracked per execution run.

### Iterative Workflow Development

**Core Value Proposition**: Rapid experimentation on workflow designs.

**Iteration Cycle**:
1. **Design**: Configure workflow (agents, stages, routing rules, thresholds)
2. **Execute**: Run workflow against test documents
3. **Analyze**: Review execution trace, confidence scores, agent decisions
4. **Adjust**: Modify prompts, thresholds, routing logic, filter defaults
5. **Re-Execute**: Test adjustments without redeployment
6. **Refine**: Iterate until reliability achieved

**Workflow Configuration Evolution**:

Initial attempt (single-agent):
```json
{
  "stages": [
    {"agent_id": "classifier", "prompt": "Analyze and classify"}
  ]
}
```

Evolved design (multi-stage with QA):
```json
{
  "stages": [
    {
      "name": "marking_identification",
      "agent_id": "marking_detector",
      "prompt": "Identify all classification markings",
      "adaptive_reprocessing": true
    },
    {
      "name": "classification_analysis",
      "agent_id": "classifier",
      "prompt": "Analyze markings and determine classification",
      "confidence_factors": ["marking_clarity", "marking_consistency", "spatial_distribution"]
    },
    {
      "name": "quality_assurance",
      "agent_id": "qa_validator",
      "prompt": "Validate analysis for two-person integrity",
      "loop_on_rejection": "marking_identification",
      "max_iterations": 3
    }
  ]
}
```

**No Prescribed Implementation**: agent-lab provides primitives (agents, orchestration patterns, adaptive processing). Specific workflow designs emerge through experimentation.

**Observability Enables Iteration**: Real-time SSE streaming + persistent execution events provide visibility into agent decisions, enabling informed adjustments.

### Configuration Persistence

**Unified Storage**:

```sql
CREATE TABLE agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    config JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE workflows (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    stages JSONB NOT NULL,
    image_settings JSONB NOT NULL,
    observability_config JSONB,
    created_at TIMESTAMP NOT NULL
);
```

**Workflow Definition Example**:
```json
{
  "stages": [...],
  "image_settings": {
    "format": "png",
    "dpi": 300,
    "options": {"brightness": 110, "contrast": 10}
  },
  "observability_config": {
    "persist_steps": true,
    "persist_agent_reasoning": true
  }
}
```

Configuration validated at execution time when transforming to domain objects.

## Middleware Patterns

### Request Logging

```go
func (app *application) logRequest(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        app.logger.Info("request",
            "method", r.Method,
            "uri", r.URL.RequestURI(),
            "addr", r.RemoteAddr)
        next.ServeHTTP(w, r)
    })
}
```

### Panic Recovery

```go
func (app *application) recoverPanic(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                w.Header().Set("Connection", "close")
                app.logger.Error("panic recovered",
                    "error", err,
                    "method", r.Method,
                    "uri", r.URL.RequestURI())
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

### Middleware Chaining

```go
func (app *application) routes() http.Handler {
    mux := http.NewServeMux()
    // ... register routes

    // Chain middleware (innermost to outermost)
    return app.recoverPanic(app.logRequest(mux))
}
```

For more complex chaining, consider `justinas/alice` or similar composition library.

## Error Handling

### Error Response Pattern

```go
type envelope map[string]any

func respondJSON(w http.ResponseWriter, status int, data any) error {
    js, err := json.MarshalIndent(data, "", "  ")
    if err != nil {
        return err
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    w.Write(js)
    w.Write([]byte("\n"))

    return nil
}

func respondError(w http.ResponseWriter, status int, message string) {
    respondJSON(w, status, envelope{"error": message})
}
```

### Service-Level Errors

```go
var (
    ErrNotFound     = errors.New("resource not found")
    ErrInvalidInput = errors.New("invalid input")
    ErrUnauthorized = errors.New("unauthorized")
)
```

### Handler Error Helpers

```go
func (h *ItemHandler) serverError(w http.ResponseWriter, r *http.Request, err error) {
    h.app.Logger().Error("server error",
        "error", err,
        "method", r.Method,
        "uri", r.URL.RequestURI())
    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

func (h *ItemHandler) notFound(w http.ResponseWriter) {
    respondError(w, http.StatusNotFound, "resource not found")
}
```

## Logging

### Structured Logging with slog

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

logger.Info("server starting", "port", 8080, "env", "production")
logger.Error("database error", "error", err, "query", query)
```

### Contextual Logging

Services receive loggers with context:

```go
func NewItemService(db *sql.DB, logger *slog.Logger, userID string) (*ItemService, error) {
    return &ItemService{
        logger: logger.With("service", "item", "user_id", userID),
    }, nil
}

// All logs from this service include service and user_id attributes
s.logger.Info("creating item", "name", name)
// Output: {"level":"info","service":"item","user_id":"abc123","name":"laptop"}
```

## Testing Strategy

### Unit Tests

Test services with mocked database:

```go
func TestItemService_Create(t *testing.T) {
    // Mock database
    mockDB := &MockDB{
        execFunc: func(query string, args ...interface{}) (sql.Result, error) {
            return mockResult{}, nil
        },
    }

    svc, _ := NewItemService(mockDB, slog.Default(), "test-user")

    item, err := svc.Create(context.Background(), CreateItemCommand{
        Name: "Test Item",
    })

    require.NoError(t, err)
    assert.Equal(t, "Test Item", item.Name)
}
```

### Integration Tests

Test with real database (Docker test containers):

```go
func TestItemService_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    db := setupTestDatabase(t)
    defer cleanupTestDatabase(t, db)

    svc, _ := NewItemService(db, slog.Default(), "test-user")

    // Test with real database
    item, err := svc.Create(context.Background(), CreateItemCommand{
        Name: "Integration Test Item",
    })

    require.NoError(t, err)

    // Verify in database
    retrieved, err := svc.Get(context.Background(), item.ID)
    require.NoError(t, err)
    assert.Equal(t, item.Name, retrieved.Name)
}
```

### API Tests

Test handlers with httptest:

```go
func TestItemHandler_Create(t *testing.T) {
    app := setupTestApplication(t)
    handler := NewItemHandler(app)

    body := `{"name":"API Test Item"}`
    req := httptest.NewRequest("POST", "/api/items", strings.NewReader(body))
    w := httptest.NewRecorder()

    handler.Create(w, req)

    assert.Equal(t, http.StatusCreated, w.Code)

    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    assert.Equal(t, "API Test Item", response["item"].(map[string]interface{})["name"])
}
```

## Server Lifecycle

### Graceful Shutdown

```go
func run() error {
    // ... setup ...

    srv := &http.Server{
        Addr:    ":8080",
        Handler: app.routes(),
    }

    // Channel for shutdown signal
    shutdown := make(chan error, 1)

    // Start server
    go func() {
        if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            shutdown <- err
        }
    }()

    // Listen for interrupt
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

    // Block until signal or error
    select {
    case err := <-shutdown:
        return err
    case sig := <-quit:
        logger.Info("shutting down", "signal", sig)
    }

    // Graceful shutdown with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    return srv.Shutdown(ctx)
}
```

## Deployment Considerations

### Container Architecture

**agent-lab Service**:
- Go binary with embedded assets (go:embed)
- Requires ImageMagick 7+ for document processing
- Mounts: `.data/blobs` (PDF storage), `.data/cache` (rendered images)

**PostgreSQL 17**:
- Container: `postgres:17-alpine`
- Database and user: `agent_lab`
- Volume: `.data/postgres` (persistent data)

### PostgreSQL 17 Container Setup

**docker-compose.yml**:
```yaml
services:
  postgres:
    image: postgres:17-alpine
    container_name: agent-lab-postgres
    restart: unless-stopped
    ports:
      - "5432:5432"
    volumes:
      - ./.data/postgres:/var/lib/postgresql/data
      - ./sql:/opt/sql:ro
    environment:
      POSTGRES_PASSWORD: postgres
```

**Database Initialization**:
```bash
# Connect as superuser
docker exec -it agent-lab-postgres psql -U postgres

# Create database and user
CREATE DATABASE agent_lab;
CREATE USER agent_lab WITH PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE agent_lab TO agent_lab;

# Run migrations
\c agent_lab
\i /opt/sql/001_create_tables.sql
```

**Connection String**:
```
postgres://agent_lab:secure_password@localhost:5432/agent_lab?sslmode=disable
```

**Connection Pool Configuration**:
- MaxOpenConns: 25
- MaxIdleConns: 5
- ConnMaxIdleTime: 5 minutes

### ImageMagick Requirement

**Dockerfile**:
```dockerfile
FROM golang:1.25.2-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o agent-lab ./cmd/server

FROM alpine:latest

RUN apk add --no-cache imagemagick

WORKDIR /app
COPY --from=builder /app/agent-lab .

# Create directories for data
RUN mkdir -p /app/.data/blobs /app/.data/cache/images

EXPOSE 8080

CMD ["./agent-lab"]
```

**Startup Verification**:
```go
func verifyDependencies() error {
    magickPath, err := exec.LookPath("magick")
    if err != nil {
        return fmt.Errorf("ImageMagick not found: %w", err)
    }

    cmd := exec.Command(magickPath, "-version")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("ImageMagick version check failed: %w", err)
    }

    log.Printf("ImageMagick found at %s:\n%s", magickPath, output)
    return nil
}
```

### Environment Configuration

**Environment Variable Convention**:
```bash
# Database
database_host=localhost
database_port=5432
database_name=agent_lab
database_user=agent_lab
database_password=secure_password

# Blob Storage
blob_storage_type=filesystem
blob_storage_filesystem_directory=./.data/blobs

# Cache
cache_directory=./.data/cache/images

# Server
server_port=8080
server_host=0.0.0.0

# Logging
logging_level=info
logging_format=json
```

**Environment Selection**:
```bash
ENV=development go run ./cmd/server  # Loads config.development.yaml
ENV=production go run ./cmd/server   # Loads config.production.yaml
```

### Docker Compose Deployment

**Development Setup**:
```yaml
services:
  postgres:
    image: postgres:17-alpine
    container_name: agent-lab-postgres
    restart: unless-stopped
    ports:
      - "5432:5432"
    volumes:
      - ./.data/postgres:/var/lib/postgresql/data
    environment:
      POSTGRES_PASSWORD: postgres

  agent-lab:
    build: .
    container_name: agent-lab-server
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./.data/blobs:/app/.data/blobs
      - ./.data/cache:/app/.data/cache
    environment:
      ENV: development
      database_host: postgres
      database_port: 5432
      database_name: agent_lab
      database_user: agent_lab
      database_password: secure_password
      blob_storage_type: filesystem
      blob_storage_filesystem_directory: /app/.data/blobs
      cache_directory: /app/.data/cache/images
    depends_on:
      - postgres
```

### Kubernetes Deployment (Phase 8)

**ConfigMap**:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: agent-lab-config
data:
  ENV: "production"
  database_host: "agent-lab-postgres.default.svc.cluster.local"
  database_port: "5432"
  database_name: "agent_lab"
  database_user: "agent_lab"
  blob_storage_type: "azure"
  blob_storage_azure_account: "agentlabstorage"
  blob_storage_azure_container: "agent-lab"
  blob_storage_azure_auth_type: "managed_identity"
```

**Secret**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: agent-lab-secrets
type: Opaque
stringData:
  database_password: "secure_password"
```

**Deployment**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agent-lab
spec:
  replicas: 3
  selector:
    matchLabels:
      app: agent-lab
  template:
    metadata:
      labels:
        app: agent-lab
    spec:
      containers:
      - name: agent-lab
        image: agentlab.azurecr.io/agent-lab:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: agent-lab-config
        - secretRef:
            name: agent-lab-secrets
        volumeMounts:
        - name: cache
          mountPath: /app/.data/cache
      volumes:
      - name: cache
        emptyDir: {}
```

**Azure Integration** (Phase 8):
- Blob Storage: Managed identity for service authentication
- Azure AI Foundry: Entra ID authentication for LLM access
- Key Vault: Sensitive configuration (connection strings, API keys)
- Application Insights: Observability and monitoring

### Volume Management

**Development**:
- `.data/postgres`: PostgreSQL data (gitignored)
- `.data/blobs`: PDF documents (gitignored)
- `.data/cache`: Rendered images (gitignored)

**Production (Kubernetes)**:
- PostgreSQL: PersistentVolumeClaim (retain policy)
- Blob Storage: Azure Blob Storage (no local volume)
- Cache: emptyDir (ephemeral, per-pod cache)

## Future Evolution

This architecture is designed to evolve based on emerging requirements:

### Event-Driven Side Effects

When cross-service reactions become necessary, introduce event system:
- Event bus for publishing domain events
- Event handlers for side effects
- Async processing for non-critical operations

### Query Builder Patterns

If dynamic query building becomes repetitive:
- Introduce squirrel or goqu for programmatic SQL construction
- Consider ProjectionMap pattern from .NET architecture
- Keep raw SQL for complex queries

### Authentication & Authorization

When multi-user support is needed:
- JWT-based authentication middleware
- User context extraction from tokens
- Role-based authorization
- Integration with identity providers

### Observability

As system complexity grows:
- Distributed tracing (OpenTelemetry)
- Metrics collection (Prometheus)
- Enhanced structured logging
- Request correlation IDs

## Conclusion

This architecture provides a solid foundation for building agent-lab while remaining flexible enough to adapt as requirements emerge. Key principles:

- Start with standard library, add dependencies when justified
- Learn low-level patterns before introducing abstractions
- Separate long-running and ephemeral concerns
- Validate at boundaries, fail fast
- Configuration-driven initialization
- Clear separation of data and behavior

The architecture will evolve through practical experience building features, not through premature optimization or over-engineering.
