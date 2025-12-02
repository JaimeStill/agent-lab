# Session 1d: Domain Infrastructure Patterns

**Status**: Implementation Guide
**Milestone**: 01 - Foundation & Infrastructure

## Objective

Establish reusable infrastructure patterns that eliminate repetitive boilerplate across domain systems, then refactor providers to use the new infrastructure.

---

## Phase 1: pkg/repository

Create new package with transaction helpers, generic query executors, and domain-agnostic error mapping.

### File: pkg/repository/repository.go

```go
package repository

import (
	"context"
	"database/sql"
)

type Querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Scanner interface {
	Scan(dest ...any) error
}

type ScanFunc[T any] func(Scanner) (T, error)

func WithTx[T any](ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) (T, error)) (T, error) {
	var zero T

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()

	result, err := fn(tx)
	if err != nil {
		return zero, err
	}

	if err := tx.Commit(); err != nil {
		return zero, err
	}

	return result, nil
}

func QueryOne[T any](ctx context.Context, q Querier, query string, args []any, scan ScanFunc[T]) (T, error) {
	var zero T
	row := q.QueryRowContext(ctx, query, args...)
	result, err := scan(row)
	if err != nil {
		return zero, err
	}
	return result, nil
}

func QueryMany[T any](ctx context.Context, q Querier, query string, args []any, scan ScanFunc[T]) ([]T, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]T, 0)
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func ExecExpectOne(ctx context.Context, e Executor, query string, args ...any) error {
	result, err := e.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}
```

### File: pkg/repository/errors.go

```go
package repository

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const pgDuplicateKeyCode = "23505"

func MapError(err error, notFoundErr, duplicateErr error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return notFoundErr
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgDuplicateKeyCode {
		return duplicateErr
	}

	return err
}
```

---

## Phase 2: pkg/handlers

Create new package with stateless HTTP response utilities.

### File: pkg/handlers/handlers.go

```go
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func RespondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func RespondError(w http.ResponseWriter, logger *slog.Logger, status int, err error) {
	logger.Error("handler error", "error", err, "status", status)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
```

---

## Phase 3: pkg/query - Multi-Column Sorting

Replace single-column sorting with multi-column sorting support.

### Replace: pkg/query/builder.go

```go
package query

import (
	"fmt"
	"strings"
)

type condition struct {
	clause string
	args   []any
}

type SortField struct {
	Field      string
	Descending bool
}

type Builder struct {
	projection        *ProjectionMap
	conditions        []condition
	orderByFields     []SortField
	defaultSortFields []SortField
}

func NewBuilder(projection *ProjectionMap, defaultSort ...SortField) *Builder {
	return &Builder{
		projection:        projection,
		conditions:        make([]condition, 0),
		defaultSortFields: defaultSort,
	}
}

func ParseSortFields(s string) []SortField {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	fields := make([]SortField, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.HasPrefix(part, "-") {
			fields = append(fields, SortField{
				Field:      strings.TrimPrefix(part, "-"),
				Descending: true,
			})
		} else {
			fields = append(fields, SortField{
				Field:      part,
				Descending: false,
			})
		}
	}

	return fields
}

func (b *Builder) BuildCount() (string, []any) {
	where, args, _ := b.buildWhere(1)
	sql := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", b.projection.Table(), where)
	return sql, args
}

func (b *Builder) BuildPage(page, pageSize int) (string, []any) {
	where, args, _ := b.buildWhere(1)
	orderBy := b.buildOrderBy()
	offset := (page - 1) * pageSize

	sql := fmt.Sprintf(
		"SELECT %s FROM %s%s%s LIMIT %d OFFSET %d",
		b.projection.Columns(),
		b.projection.Table(),
		where,
		orderBy,
		pageSize,
		offset,
	)

	return sql, args
}

func (b *Builder) BuildSingle(idField string, id any) (string, []any) {
	col := b.projection.Column(idField)
	sql := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s = $1",
		b.projection.Columns(),
		b.projection.Table(),
		col,
	)
	return sql, []any{id}
}

func (b *Builder) OrderByFields(fields []SortField) *Builder {
	b.orderByFields = fields
	return b
}

func (b *Builder) WhereContains(field string, value *string) *Builder {
	if value == nil || *value == "" {
		return b
	}
	col := b.projection.Column(field)
	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s ILIKE $%%d", col),
		args:   []any{"%" + *value + "%"},
	})
	return b
}

func (b *Builder) WhereEquals(field string, value any) *Builder {
	if value == nil {
		return b
	}
	col := b.projection.Column(field)
	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s = $%%d", col),
		args:   []any{value},
	})
	return b
}

func (b *Builder) WhereIn(field string, values []any) *Builder {
	if len(values) == 0 {
		return b
	}
	col := b.projection.Column(field)
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = "$%d"
	}
	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", ")),
		args:   values,
	})
	return b
}

func (b *Builder) WhereSearch(search *string, fields ...string) *Builder {
	if search == nil || *search == "" || len(fields) == 0 {
		return b
	}

	clauses := make([]string, len(fields))
	args := make([]any, len(fields))
	searchPattern := "%" + *search + "%"

	for i, field := range fields {
		col := b.projection.Column(field)
		clauses[i] = fmt.Sprintf("%s ILIKE $%%d", col)
		args[i] = searchPattern
	}

	b.conditions = append(b.conditions, condition{
		clause: "(" + strings.Join(clauses, " OR ") + ")",
		args:   args,
	})
	return b
}

func (b *Builder) buildOrderBy() string {
	fields := b.orderByFields
	if len(fields) == 0 {
		fields = b.defaultSortFields
	}

	if len(fields) == 0 {
		return ""
	}

	parts := make([]string, len(fields))
	for i, f := range fields {
		col := b.projection.Column(f.Field)
		dir := "ASC"
		if f.Descending {
			dir = "DESC"
		}
		parts[i] = fmt.Sprintf("%s %s", col, dir)
	}

	return " ORDER BY " + strings.Join(parts, ", ")
}

func (b *Builder) buildWhere(startParam int) (string, []any, int) {
	if len(b.conditions) == 0 {
		return "", nil, startParam
	}

	clauses := make([]string, 0, len(b.conditions))
	args := make([]any, 0)
	paramIdx := startParam

	for _, cond := range b.conditions {
		clause := cond.clause
		for _, arg := range cond.args {
			clause = strings.Replace(clause, "$%d", fmt.Sprintf("$%d", paramIdx), 1)
			args = append(args, arg)
			paramIdx++
		}
		clauses = append(clauses, clause)
	}

	return " WHERE " + strings.Join(clauses, " AND "), args, paramIdx
}
```

---

## Phase 4: pkg/pagination - Query Parameter Parsing

Update PageRequest to use multi-column sorting and add query parameter parsing.

### Replace: pkg/pagination/pagination.go

```go
package pagination

import (
	"net/url"
	"strconv"

	"github.com/JaimeStill/agent-lab/pkg/query"
)

type PageRequest struct {
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	Search   *string           `json:"search,omitempty"`
	Sort     []query.SortField `json:"sort,omitempty"`
}

func (r *PageRequest) Normalize(cfg Config) {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 {
		r.PageSize = cfg.DefaultPageSize
	}
	if r.PageSize > cfg.MaxPageSize {
		r.PageSize = cfg.MaxPageSize
	}
}

func (r *PageRequest) Offset() int {
	return (r.Page - 1) * r.PageSize
}

func PageRequestFromQuery(values url.Values, cfg Config) PageRequest {
	page, _ := strconv.Atoi(values.Get("page"))
	pageSize, _ := strconv.Atoi(values.Get("page_size"))

	var search *string
	if s := values.Get("search"); s != "" {
		search = &s
	}

	sort := query.ParseSortFields(values.Get("sort"))

	req := PageRequest{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
		Sort:     sort,
	}

	req.Normalize(cfg)
	return req
}

type PageResult[T any] struct {
	Data       []T `json:"data"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalPages int `json:"total_pages"`
}

func NewPageResult[T any](data []T, total, page, pageSize int) PageResult[T] {
	totalPages := total / pageSize
	if total%pageSize != 0 {
		totalPages++
	}
	if totalPages < 1 {
		totalPages = 1
	}

	if data == nil {
		data = []T{}
	}

	return PageResult[T]{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}
```

---

## Phase 5: Domain Filter Pattern

Create filters for providers domain.

### File: internal/providers/filters.go

```go
package providers

import (
	"net/url"

	"github.com/JaimeStill/agent-lab/pkg/query"
)

type Filters struct {
	Name *string
}

func FiltersFromQuery(values url.Values) Filters {
	var name *string
	if n := values.Get("name"); n != "" {
		name = &n
	}

	return Filters{
		Name: name,
	}
}

func (f Filters) Apply(b *query.Builder) *query.Builder {
	return b.WhereContains("Name", f.Name)
}
```

---

## Phase 6: Providers Refactoring

Complete refactoring of providers to use new infrastructure.

### File: internal/providers/scanner.go

```go
package providers

import "github.com/JaimeStill/agent-lab/pkg/repository"

func scanProvider(s repository.Scanner) (Provider, error) {
	var p Provider
	err := s.Scan(&p.ID, &p.Name, &p.Config, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}
```

### Additions to: internal/providers/errors.go

Add HTTP status mapping function:

```go
func MapHTTPStatus(err error) int {
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrDuplicate) {
		return http.StatusConflict
	}
	if errors.Is(err, ErrInvalidConfig) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
```

Update imports to include `net/http`.

### Replace: internal/providers/system.go

```go
package providers

import (
	"context"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

type System interface {
	Create(ctx context.Context, cmd CreateCommand) (*Provider, error)
	Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Provider, error)
	Delete(ctx context.Context, id uuid.UUID) error
	FindByID(ctx context.Context, id uuid.UUID) (*Provider, error)
	Search(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Provider], error)
}
```

### Replace: internal/providers/repository.go

```go
package providers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/agent-lab/pkg/repository"
	agtconfig "github.com/JaimeStill/go-agents/pkg/config"
	agtproviders "github.com/JaimeStill/go-agents/pkg/providers"
	"github.com/google/uuid"
)

type repo struct {
	db         *sql.DB
	logger     *slog.Logger
	pagination pagination.Config
}

func New(db *sql.DB, logger *slog.Logger, pagination pagination.Config) System {
	return &repo{
		db:         db,
		logger:     logger.With("system", "provider"),
		pagination: pagination,
	}
}

func (r *repo) Create(ctx context.Context, cmd CreateCommand) (*Provider, error) {
	if err := r.validateConfig(cmd.Config); err != nil {
		return nil, err
	}

	q := `
		INSERT INTO providers (name, config)
		VALUES ($1, $2)
		RETURNING id, name, config, created_at, updated_at`

	p, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Provider, error) {
		return repository.QueryOne(ctx, tx, q, []any{cmd.Name, cmd.Config}, scanProvider)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("provider created", "id", p.ID, "name", p.Name)
	return &p, nil
}

func (r *repo) Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Provider, error) {
	if err := r.validateConfig(cmd.Config); err != nil {
		return nil, err
	}

	q := `
		UPDATE providers
		SET name = $1, config = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, name, config, created_at, updated_at`

	p, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Provider, error) {
		return repository.QueryOne(ctx, tx, q, []any{cmd.Name, cmd.Config, id}, scanProvider)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("provider updated", "id", p.ID, "name", p.Name)
	return &p, nil
}

func (r *repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (struct{}, error) {
		err := repository.ExecExpectOne(ctx, tx, "DELETE FROM providers WHERE id = $1", id)
		return struct{}{}, err
	})

	if err != nil {
		return repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("provider deleted", "id", id)
	return nil
}

func (r *repo) FindByID(ctx context.Context, id uuid.UUID) (*Provider, error) {
	q, args := query.NewBuilder(projection).BuildSingle("Id", id)

	p, err := repository.QueryOne(ctx, r.db, q, args, scanProvider)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	return &p, nil
}

func (r *repo) Search(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Provider], error) {
	page.Normalize(r.pagination)

	qb := query.NewBuilder(projection, query.SortField{Field: "Name"}).
		WhereSearch(page.Search, "Name")

	filters.Apply(qb)

	if len(page.Sort) > 0 {
		qb.OrderByFields(page.Sort)
	}

	countSql, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRowContext(ctx, countSql, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count providers: %w", err)
	}

	pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	providers, err := repository.QueryMany(ctx, r.db, pageSQL, pageArgs, scanProvider)
	if err != nil {
		return nil, fmt.Errorf("query providers: %w", err)
	}

	result := pagination.NewPageResult(providers, total, page.Page, page.PageSize)
	return &result, nil
}

func (r *repo) validateConfig(config json.RawMessage) error {
	var cfg agtconfig.ProviderConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	if _, err := agtproviders.Create(&cfg); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	return nil
}
```

### File: internal/providers/handler.go

```go
package providers

import (
	"encoding/json"
	"log/slog"
	"net/http"

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
		logger:     logger,
		pagination: pagination,
	}
}

func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix:      "/api/providers",
		Tags:        []string{"Providers"},
		Description: "Provider configuration management",
		Routes: []routes.Route{
			{Method: "POST", Pattern: "", Handler: h.Create},
			{Method: "GET", Pattern: "", Handler: h.List},
			{Method: "GET", Pattern: "/{id}", Handler: h.GetByID},
			{Method: "PUT", Pattern: "/{id}", Handler: h.Update},
			{Method: "DELETE", Pattern: "/{id}", Handler: h.Delete},
			{Method: "POST", Pattern: "/search", Handler: h.Search},
		},
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var cmd CreateCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	result, err := h.sys.Create(r.Context(), cmd)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusCreated, result)
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

	result, err := h.sys.Update(r.Context(), id, cmd)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, result)
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

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	result, err := h.sys.FindByID(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, result)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page := pagination.PageRequestFromQuery(r.URL.Query(), h.pagination)
	filters := FiltersFromQuery(r.URL.Query())

	result, err := h.sys.Search(r.Context(), page, filters)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusInternalServerError, err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, result)
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
```

### Delete Files

- `internal/providers/handlers.go`
- `internal/providers/routes.go`

### Replace: cmd/server/routes.go

```go
package main

import (
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/lifecycle"
	"github.com/JaimeStill/agent-lab/internal/providers"
	"github.com/JaimeStill/agent-lab/internal/routes"
)

func registerRoutes(r routes.System, runtime *Runtime, domain *Domain) {
	providerHandler := providers.NewHandler(domain.Providers, runtime.Logger, runtime.Pagination)
	r.RegisterGroup(providerHandler.Routes())

	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: "/healthz",
		Handler: handleHealthCheck,
	})

	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: "/readyz",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			handleReadinessCheck(w, runtime.Lifecycle)
		},
	})
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleReadinessCheck(w http.ResponseWriter, ready lifecycle.ReadinessChecker) {
	if !ready.Ready() {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("NOT READY"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("READY"))
}
```

---

## Validation Checklist

- [ ] pkg/repository helpers working
- [ ] pkg/handlers utilities working
- [ ] Multi-column sorting generates correct SQL
- [ ] PageRequestFromQuery parses query params correctly
- [ ] Providers uses new infrastructure
- [ ] GET /api/providers works with filters + sort
- [ ] POST /api/providers/search still works
- [ ] All tests passing
