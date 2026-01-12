---
name: go-core
description: >
  Core Go development patterns. Use when working with file structure,
  error handling, naming conventions, or package organization.
  Triggers: errors.go, domain errors, file structure, constants, interfaces,
  structured logging, slog, naming convention, Err prefix.
  File patterns: internal/**/*.go, pkg/**/*.go
---

# Go Core Development

## When This Skill Applies

- Organizing code within Go files
- Defining package-level errors
- Naming interface methods
- Implementing structured logging
- Creating new packages

## Principles

### 1. File Structure Convention

Go files follow a consistent structural order:

1. **Package declaration and imports**
2. **Constants** - Package-level constants
3. **Global variables** - Package-level variables (including errors)
4. **Interfaces** - Interface definitions
5. **Pure types** - Types without methods (data containers)
6. **Types with methods** - Structs with associated methods
7. **Functions** - Package-level functions

```go
package workflows

import (
    "context"
    "time"
)

const defaultStreamBufferSize = 100

var ErrWorkflowNotFound = errors.New("workflow not found")

type System interface {
    Execute(ctx context.Context, req ExecuteRequest) error
}

type ExecuteRequest struct {
    Params map[string]any `json:"params"`
}

type Handler struct {
    sys    System
    logger *slog.Logger
}

func NewHandler(sys System, logger *slog.Logger) *Handler {
    return &Handler{sys: sys, logger: logger}
}

func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
    // ...
}
```

### 2. Interface Naming Convention

**Getters** (Nouns - Access State):
```go
Id() uuid.UUID
Name() string
Connection() *sql.DB
```

**Commands** (Verbs - Perform Actions):
```go
Start(ctx context.Context) error
Create(ctx context.Context, cmd CreateCommand) (*Provider, error)
Search(ctx context.Context, req SearchRequest) (*SearchResult, error)
```

**Events** (On* - Notifications):
```go
OnShutdown() <-chan struct{}
OnError() <-chan error
```

### 3. Repository Query Methods

| Verb | Returns | Use Case |
|------|---------|----------|
| `List` | `*PageResult[T]` | Paginated browsing/searching |
| `Find` | `*T` | Single item by ID |
| `Get` | `[]T` | All related items (bounded slice) |

```go
ListRuns(ctx, page, filters) (*PageResult[Run], error)  // Paginated
FindRun(ctx, id) (*Run, error)                          // Single by ID
GetStages(ctx, runID) ([]Stage, error)                  // All for parent
```

### 4. Data Mutation Methods

| Verb | Semantics |
|------|-----------|
| `Create` | Insert new (fails if exists) |
| `Update` | Modify existing (fails if not exists) |
| `Save` | Create or update (idempotent) |
| `Delete` | Remove record |

### 5. Encapsulated Package Errors

Each package defines errors in a dedicated `errors.go` file:

```go
// internal/database/errors.go
package database

import "errors"

var ErrNotReady = errors.New("database not ready")
```

**Conventions**:
- Package-level errors in `errors.go`
- Use `Err` prefix for exported error variables
- Error messages are lowercase, no punctuation
- Enables clean usage: `database.ErrNotReady`, `providers.ErrNotFound`

### 6. Error Wrapping

```go
if err := doSomething(); err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

### 7. Structured Logging

```go
logger.Info("operation succeeded", "id", id, "name", name)
logger.Error("operation failed", "error", err, "id", id)
```

## Anti-Patterns

### Unstructured Error Messages

```go
// Bad: Inconsistent casing, punctuation
var ErrNotFound = errors.New("Item Not Found!")

// Good: Lowercase, no punctuation
var ErrNotFound = errors.New("item not found")
```

### Scattered Error Definitions

```go
// Bad: Errors defined inline throughout code
func (r *Repository) Find(id string) (*Item, error) {
    return nil, errors.New("not found")
}

// Good: Centralized in errors.go
var ErrNotFound = errors.New("not found")

func (r *Repository) Find(id string) (*Item, error) {
    return nil, ErrNotFound
}
```
