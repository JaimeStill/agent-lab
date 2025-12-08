# Session 2a: Blob Storage Infrastructure

## Problem Context

Milestone 2 requires document upload and storage capabilities. Before implementing the documents domain (Session 2b), we need a storage abstraction that:

1. Provides a consistent interface for blob operations (Store, Retrieve, Delete, Validate)
2. Supports filesystem storage now, with Azure blob storage planned for Milestone 8
3. Integrates with the lifecycle coordinator for directory initialization
4. Follows established patterns from the database system

## Architecture Approach

### Package Structure

```
internal/storage/
  storage.go      # System interface + error types
  filesystem.go   # Filesystem implementation

internal/config/
  storage.go      # StorageConfig section (new file)
```

This mirrors the database pattern where interface and implementation are co-located in `internal/`.

### System Interface Design

The storage `System` interface follows the database pattern:
- `New()` validates configuration and returns a configured instance
- `Start()` registers lifecycle hooks (directory creation)
- Operational methods: `Store`, `Retrieve`, `Delete`, `Validate`

### Lifecycle Integration

- `OnStartup`: Creates base directory with `os.MkdirAll`
- No `OnShutdown` needed (filesystem has no cleanup requirements)

### Key Behavior

- **Validate**: Returns `(true, nil)` if exists, `(false, nil)` if not exists, `(false, error)` for permission/system errors
- **Path Safety**: `fullPath()` helper prevents directory traversal attacks
- **Atomic Writes**: Store writes to temp file then renames for crash safety

---

## Phase 1: Configuration

### 1.1 Create `internal/config/storage.go`

```go
package config

import (
	"fmt"
	"os"
)

const (
	EnvStorageBasePath = "STORAGE_BASE_PATH"
)

type StorageConfig struct {
	BasePath string `toml:"base_path"`
}

func (c *StorageConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()
	return c.validate()
}

func (c *StorageConfig) Merge(overlay *StorageConfig) {
	if overlay.BasePath != "" {
		c.BasePath = overlay.BasePath
	}
}

func (c *StorageConfig) loadDefaults() {
	if c.BasePath == "" {
		c.BasePath = ".data/blobs"
	}
}

func (c *StorageConfig) loadEnv() {
	if v := os.Getenv(EnvStorageBasePath); v != "" {
		c.BasePath = v
	}
}

func (c *StorageConfig) validate() error {
	if c.BasePath == "" {
		return fmt.Errorf("base_path required")
	}
	return nil
}
```

### 1.2 Update `internal/config/config.go`

Add to imports if needed:
```go
// No new imports required
```

Add field to Config struct (after Pagination):
```go
type Config struct {
	Server          ServerConfig      `toml:"server"`
	Database        DatabaseConfig    `toml:"database"`
	Logging         LoggingConfig     `toml:"logging"`
	CORS            CORSConfig        `toml:"cors"`
	Pagination      pagination.Config `toml:"pagination"`
	Storage         StorageConfig     `toml:"storage"`
	Domain          string            `toml:"version"`
	ShutdownTimeout string            `toml:"shutdown_timeout"`
	Version         string            `toml:"version"`
}
```

Add to `finalize()` method (after Pagination.Finalize):
```go
if err := c.Storage.Finalize(); err != nil {
	return fmt.Errorf("storage: %w", err)
}
```

Add to `Merge()` method (after Pagination.Merge):
```go
c.Storage.Merge(&overlay.Storage)
```

### 1.3 Update `config.toml`

Add after `[pagination]` section:
```toml
[storage]
base_path = ".data/blobs"
```

### 1.4 Update `.env`

Add after the Pagination section (before CLI Tools):
```env
# Storage
STORAGE_BASE_PATH=.data/blobs
```

---

## Phase 2: Storage Interface

### 2.1 Create `internal/storage/errors.go`

```go
package storage

import "errors"

var (
	ErrNotFound         = errors.New("storage: key not found")
	ErrPermissionDenied = errors.New("storage: permission denied")
	ErrInvalidKey       = errors.New("storage: invalid key")
)
```

### 2.2 Create `internal/storage/storage.go`

```go
package storage

import (
	"context"

	"github.com/JaimeStill/agent-lab/internal/lifecycle"
)

type System interface {
	Store(ctx context.Context, key string, data []byte) error
	Retrieve(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	Validate(ctx context.Context, key string) (bool, error)
	Start(lc *lifecycle.Coordinator) error
}
```

---

## Phase 3: Filesystem Implementation

### 3.1 Create `internal/storage/filesystem.go`

```go
package storage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/lifecycle"
)

type filesystem struct {
	basePath string
	logger   *slog.Logger
}

func New(cfg *config.StorageConfig, logger *slog.Logger) (System, error) {
	if cfg.BasePath == "" {
		return nil, fmt.Errorf("base_path required")
	}

	absPath, err := filepath.Abs(cfg.BasePath)
	if err != nil {
		return nil, fmt.Errorf("resolve base_path: %w", err)
	}

	return &filesystem{
		basePath: absPath,
		logger:   logger.With("system", "storage"),
	}, nil
}

func (f *filesystem) Start(lc *lifecycle.Coordinator) error {
	f.logger.Info("starting storage system", "base_path", f.basePath)

	lc.OnStartup(func() {
		if err := os.MkdirAll(f.basePath, 0755); err != nil {
			f.logger.Error("storage initialization failed", "error", err)
			return
		}
		f.logger.Info("storage directory initialized")
	})

	return nil
}

func (f *filesystem) Store(ctx context.Context, key string, data []byte) error {
	path, err := f.fullPath(key)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

func (f *filesystem) Retrieve(ctx context.Context, key string) ([]byte, error) {
	path, err := f.fullPath(key)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrNotFound
		}
		if errors.Is(err, fs.ErrPermission) {
			return nil, ErrPermissionDenied
		}
		return nil, fmt.Errorf("read file: %w", err)
	}

	return data, nil
}

func (f *filesystem) Delete(ctx context.Context, key string) error {
	path, err := f.fullPath(key)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if errors.Is(err, fs.ErrPermission) {
			return ErrPermissionDenied
		}
		return fmt.Errorf("remove file: %w", err)
	}

	return nil
}

func (f *filesystem) Validate(ctx context.Context, key string) (bool, error) {
	path, err := f.fullPath(key)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		if errors.Is(err, fs.ErrPermission) {
			return false, ErrPermissionDenied
		}
		return false, fmt.Errorf("stat file: %w", err)
	}

	return true, nil
}

func (f *filesystem) fullPath(key string) (string, error) {
	if key == "" {
		return "", ErrInvalidKey
	}

	cleaned := filepath.Clean(key)
	if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return "", ErrInvalidKey
	}

	fullPath := filepath.Join(f.basePath, cleaned)

	if !strings.HasPrefix(fullPath, f.basePath) {
		return "", ErrInvalidKey
	}

	return fullPath, nil
}
```

---

## Phase 4: Runtime Integration

### 4.1 Update `cmd/server/runtime.go`

Add import:
```go
"github.com/JaimeStill/agent-lab/internal/storage"
```

Update Runtime struct:
```go
type Runtime struct {
	Lifecycle  *lifecycle.Coordinator
	Logger     *slog.Logger
	Database   database.System
	Storage    storage.System
	Pagination pagination.Config
}
```

Update NewRuntime function (add after database initialization):
```go
func NewRuntime(cfg *config.Config) (*Runtime, error) {
	lc := lifecycle.New()
	logger := newLogger(&cfg.Logging)

	dbSys, err := database.New(&cfg.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("database init failed: %w", err)
	}

	storageSys, err := storage.New(&cfg.Storage, logger)
	if err != nil {
		return nil, fmt.Errorf("storage init failed: %w", err)
	}

	return &Runtime{
		Lifecycle:  lc,
		Logger:     logger,
		Database:   dbSys,
		Storage:    storageSys,
		Pagination: cfg.Pagination,
	}, nil
}
```

Update Start method:
```go
func (r *Runtime) Start() error {
	if err := r.Database.Start(r.Lifecycle); err != nil {
		return fmt.Errorf("database start failed: %w", err)
	}

	if err := r.Storage.Start(r.Lifecycle); err != nil {
		return fmt.Errorf("storage start failed: %w", err)
	}

	return nil
}
```

---

## Validation Checklist

After implementation, verify:

1. [ ] Service starts without errors
2. [ ] `.data/blobs/` directory is created on startup
3. [ ] Log message: "storage directory initialized"
4. [ ] Configuration loads from `config.toml`
5. [ ] Environment variable `STORAGE_BASE_PATH` overrides config

---

## File Summary

| File | Action |
|------|--------|
| `internal/config/storage.go` | Create |
| `internal/config/config.go` | Modify (add Storage field, finalize, merge) |
| `internal/storage/errors.go` | Create |
| `internal/storage/storage.go` | Create |
| `internal/storage/filesystem.go` | Create |
| `cmd/server/runtime.go` | Modify (add Storage to Runtime) |
| `config.toml` | Modify (add [storage] section) |
| `.env` | Modify (add STORAGE_BASE_PATH) |
