# Session 01e: Agents System

**Session**: 01e
**Milestone**: 01 - Foundation & Infrastructure

## Overview

Implement the Agents domain system with CRUD operations and execution endpoints for all go-agents protocols (Chat, Vision, Tools, Embed) including SSE streaming.

## Architecture Context

The agents system stores complete `AgentConfig` as JSONB, decoupled from the providers table. Agent instances are constructed per-request using `agent.New(cfg)` - no caching.

---

## Phase 1: Database Migration

### File: `cmd/migrate/migrations/000003_agents.up.sql`

```sql
CREATE TABLE agents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  config JSONB NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_agents_name ON agents(name);
CREATE INDEX idx_agents_created_at ON agents(created_at DESC);
```

### File: `cmd/migrate/migrations/000003_agents.down.sql`

```sql
DROP INDEX IF EXISTS idx_agents_created_at;
DROP INDEX IF EXISTS idx_agents_name;
DROP TABLE IF EXISTS agents;
```

---

## Phase 2: Domain Core

### File: `internal/agents/agent.go`

```go
package agents

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Agent struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Config    json.RawMessage `json:"config"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type CreateCommand struct {
	Name   string          `json:"name"`
	Config json.RawMessage `json:"config"`
}

type UpdateCommand struct {
	Name   string          `json:"name"`
	Config json.RawMessage `json:"config"`
}
```

### File: `internal/agents/errors.go`

```go
package agents

import (
	"errors"
	"net/http"
)

var (
	ErrNotFound      = errors.New("agent not found")
	ErrDuplicate     = errors.New("agent name already exists")
	ErrInvalidConfig = errors.New("invalid agent config")
	ErrExecution     = errors.New("agent execution failed")
)

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
	if errors.Is(err, ErrExecution) {
		return http.StatusBadGateway
	}
	return http.StatusInternalServerError
}
```

### File: `internal/agents/projection.go`

```go
package agents

import "github.com/JaimeStill/agent-lab/pkg/query"

var projection = query.NewProjectionMap("public", "agents", "a").
	Project("id", "Id").
	Project("name", "Name").
	Project("config", "Config").
	Project("created_at", "CreatedAt").
	Project("updated_at", "UpdatedAt")
```

### File: `internal/agents/filters.go`

```go
package agents

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

### File: `internal/agents/scanner.go`

```go
package agents

import "github.com/JaimeStill/agent-lab/pkg/repository"

func scanAgent(s repository.Scanner) (Agent, error) {
	var a Agent
	err := s.Scan(&a.ID, &a.Name, &a.Config, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}
```

### File: `internal/agents/system.go`

```go
package agents

import (
	"context"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

type System interface {
	Create(ctx context.Context, cmd CreateCommand) (*Agent, error)
	Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Agent, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*Agent, error)
	Search(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Agent], error)
}
```

### File: `internal/agents/repository.go`

```go
package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/agent-lab/pkg/repository"
	"github.com/JaimeStill/go-agents/pkg/agent"
	agtconfig "github.com/JaimeStill/go-agents/pkg/config"
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
		logger:     logger.With("system", "agent"),
		pagination: pagination,
	}
}

func (r *repo) Create(ctx context.Context, cmd CreateCommand) (*Agent, error) {
	if err := r.validateConfig(cmd.Config); err != nil {
		return nil, err
	}

	q := `
		INSERT INTO agents (name, config)
		VALUES ($1, $2)
		RETURNING id, name, config, created_at, updated_at`

	a, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Agent, error) {
		return repository.QueryOne(ctx, tx, q, []any{cmd.Name, cmd.Config}, scanAgent)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("agent created", "id", a.ID, "name", a.Name)
	return &a, nil
}

func (r *repo) Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Agent, error) {
	if err := r.validateConfig(cmd.Config); err != nil {
		return nil, err
	}

	q := `
		UPDATE agents
		SET name = $1, config = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, name, config, created_at, updated_at`

	a, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Agent, error) {
		return repository.QueryOne(ctx, tx, q, []any{cmd.Name, cmd.Config, id}, scanAgent)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("agent updated", "id", a.ID, "name", a.Name)
	return &a, nil
}

func (r *repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (struct{}, error) {
		err := repository.ExecExpectOne(ctx, tx, "DELETE FROM agents WHERE id = $1", id)
		return struct{}{}, err
	})

	if err != nil {
		return repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("agent deleted", "id", id)
	return nil
}

func (r *repo) GetByID(ctx context.Context, id uuid.UUID) (*Agent, error) {
	q, args := query.NewBuilder(projection).BuildSingle("Id", id)

	a, err := repository.QueryOne(ctx, r.db, q, args, scanAgent)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	return &a, nil
}

func (r *repo) Search(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Agent], error) {
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
		return nil, fmt.Errorf("count agents: %w", err)
	}

	pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	agents, err := repository.QueryMany(ctx, r.db, pageSQL, pageArgs, scanAgent)
	if err != nil {
		return nil, fmt.Errorf("query agents: %w", err)
	}

	result := pagination.NewPageResult(agents, total, page.Page, page.PageSize)
	return &result, nil
}

func (r *repo) validateConfig(config json.RawMessage) error {
	var cfg agtconfig.AgentConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	if _, err := agent.New(&cfg); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	return nil
}
```

---

## Phase 3: Request Types

### File: `internal/agents/requests.go`

```go
package agents

import "github.com/JaimeStill/go-agents/pkg/agent"

type ChatRequest struct {
	Prompt  string         `json:"prompt"`
	Options map[string]any `json:"options,omitempty"`
}

type VisionRequest struct {
	Prompt  string         `json:"prompt"`
	Images  []string       `json:"images"`
	Options map[string]any `json:"options,omitempty"`
}

type ToolsRequest struct {
	Prompt  string         `json:"prompt"`
	Tools   []agent.Tool   `json:"tools"`
	Options map[string]any `json:"options,omitempty"`
}

type EmbedRequest struct {
	Input   string         `json:"input"`
	Options map[string]any `json:"options,omitempty"`
}
```

---

## Phase 4: Handler

### File: `internal/agents/handler.go`

```go
package agents

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/routes"
	"github.com/JaimeStill/agent-lab/pkg/handlers"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/go-agents/pkg/agent"
	agtconfig "github.com/JaimeStill/go-agents/pkg/config"
	"github.com/JaimeStill/go-agents/pkg/response"
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
		Prefix:      "/api/agents",
		Tags:        []string{"Agents"},
		Description: "Agent configuration and execution",
		Routes: []routes.Route{
			{Method: "POST", Pattern: "", Handler: h.Create},
			{Method: "GET", Pattern: "", Handler: h.List},
			{Method: "GET", Pattern: "/{id}", Handler: h.GetByID},
			{Method: "PUT", Pattern: "/{id}", Handler: h.Update},
			{Method: "DELETE", Pattern: "/{id}", Handler: h.Delete},
			{Method: "POST", Pattern: "/search", Handler: h.Search},
			{Method: "POST", Pattern: "/{id}/chat", Handler: h.Chat},
			{Method: "POST", Pattern: "/{id}/chat/stream", Handler: h.ChatStream},
			{Method: "POST", Pattern: "/{id}/vision", Handler: h.Vision},
			{Method: "POST", Pattern: "/{id}/vision/stream", Handler: h.VisionStream},
			{Method: "POST", Pattern: "/{id}/tools", Handler: h.Tools},
			{Method: "POST", Pattern: "/{id}/embed", Handler: h.Embed},
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

	result, err := h.sys.GetByID(r.Context(), id)
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

func (h *Handler) constructAgent(ctx context.Context, id uuid.UUID) (agent.Agent, error) {
	record, err := h.sys.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var cfg agtconfig.AgentConfig
	if err := json.Unmarshal(record.Config, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	agt, err := agent.New(&cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	return agt, nil
}

func (h *Handler) writeSSEStream(w http.ResponseWriter, r *http.Request, stream <-chan *response.StreamingChunk) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	for chunk := range stream {
		if chunk.Error != nil {
			data, _ := json.Marshal(map[string]string{"error": chunk.Error.Error()})
			fmt.Fprintf(w, "data: %s\n\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			return
		}

		select {
		case <-r.Context().Done():
			return
		default:
		}

		data, err := json.Marshal(chunk)
		if err != nil {
			h.logger.Error("failed to marshal chunk", "error", err)
			continue
		}

		fmt.Fprintf(w, "data: %s\n\n", data)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	agt, err := h.constructAgent(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	resp, err := agt.Chat(r.Context(), req.Prompt, req.Options)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(fmt.Errorf("%w: %v", ErrExecution, err)), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, resp)
}

func (h *Handler) ChatStream(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	agt, err := h.constructAgent(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	stream, err := agt.ChatStream(r.Context(), req.Prompt, req.Options)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(fmt.Errorf("%w: %v", ErrExecution, err)), err)
		return
	}

	h.writeSSEStream(w, r, stream)
}

func (h *Handler) Vision(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	var req VisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	agt, err := h.constructAgent(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	resp, err := agt.Vision(r.Context(), req.Prompt, req.Images, req.Options)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(fmt.Errorf("%w: %v", ErrExecution, err)), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, resp)
}

func (h *Handler) VisionStream(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	var req VisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	agt, err := h.constructAgent(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	stream, err := agt.VisionStream(r.Context(), req.Prompt, req.Images, req.Options)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(fmt.Errorf("%w: %v", ErrExecution, err)), err)
		return
	}

	h.writeSSEStream(w, r, stream)
}

func (h *Handler) Tools(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	var req ToolsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	agt, err := h.constructAgent(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	resp, err := agt.Tools(r.Context(), req.Prompt, req.Tools, req.Options)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(fmt.Errorf("%w: %v", ErrExecution, err)), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, resp)
}

func (h *Handler) Embed(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	var req EmbedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	agt, err := h.constructAgent(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	resp, err := agt.Embed(r.Context(), req.Input, req.Options)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(fmt.Errorf("%w: %v", ErrExecution, err)), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, resp)
}
```

---

## Phase 5: Integration

### File: `cmd/server/domain.go` (modify)

Add import and Agents field:

```go
package main

import (
	"github.com/JaimeStill/agent-lab/internal/agents"
	"github.com/JaimeStill/agent-lab/internal/providers"
)

type Domain struct {
	Providers providers.System
	Agents    agents.System
}

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
	}
}
```

### File: `cmd/server/routes.go` (modify)

Add agents handler registration:

```go
package main

import (
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/agents"
	"github.com/JaimeStill/agent-lab/internal/lifecycle"
	"github.com/JaimeStill/agent-lab/internal/providers"
	"github.com/JaimeStill/agent-lab/internal/routes"
)

func registerRoutes(r routes.System, runtime *Runtime, domain *Domain) {
	providerHandler := providers.NewHandler(domain.Providers, runtime.Logger, runtime.Pagination)
	r.RegisterGroup(providerHandler.Routes())

	agentHandler := agents.NewHandler(domain.Agents, runtime.Logger, runtime.Pagination)
	r.RegisterGroup(agentHandler.Routes())

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

## Phase 6: Execution Enhancements

### Overview

Two enhancements to execution endpoints:
1. **Token support**: Optional `token` field for Azure API key / bearer token authentication
2. **Vision file uploads**: Accept `multipart/form-data` with image files instead of base64 JSON

### File: `internal/agents/requests.go` (replace entire file)

```go
package agents

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/JaimeStill/go-agents/pkg/agent"
)

type ChatRequest struct {
	Prompt  string         `json:"prompt"`
	Options map[string]any `json:"options,omitempty"`
	Token   string         `json:"token,omitempty"`
}

type ToolsRequest struct {
	Prompt  string         `json:"prompt"`
	Tools   []agent.Tool   `json:"tools"`
	Options map[string]any `json:"options,omitempty"`
	Token   string         `json:"token,omitempty"`
}

type EmbedRequest struct {
	Input   string         `json:"input"`
	Options map[string]any `json:"options,omitempty"`
	Token   string         `json:"token,omitempty"`
}

type VisionForm struct {
	Prompt  string
	Images  []string
	Options map[string]any
	Token   string
}

func ParseVisionForm(r *http.Request, maxMemory int64) (*VisionForm, error) {
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		return nil, fmt.Errorf("failed to parse multipart form: %w", err)
	}

	form := &VisionForm{
		Prompt: r.FormValue("prompt"),
		Token:  r.FormValue("token"),
	}

	if form.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	if optStr := r.FormValue("options"); optStr != "" {
		if err := json.Unmarshal([]byte(optStr), &form.Options); err != nil {
			return nil, fmt.Errorf("invalid options JSON: %w", err)
		}
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		return nil, fmt.Errorf("at least one image is required")
	}

	images, err := prepareImages(files)
	if err != nil {
		return nil, err
	}
	form.Images = images

	return form, nil
}

func prepareImages(files []*multipart.FileHeader) ([]string, error) {
	prepared := make([]string, len(files))

	for i, fh := range files {
		file, err := fh.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %w", fh.Filename, err)
		}

		data, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", fh.Filename, err)
		}

		mimeType := http.DetectContentType(data)
		if !strings.HasPrefix(mimeType, "image/") {
			return nil, fmt.Errorf("file %s is not an image (detected: %s)", fh.Filename, mimeType)
		}

		encoded := base64.StdEncoding.EncodeToString(data)
		prepared[i] = fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
	}

	return prepared, nil
}
```

### File: `internal/agents/handler.go` (updates)

#### Update `constructAgent` to accept token parameter

```go
func (h *Handler) constructAgent(ctx context.Context, id uuid.UUID, token string) (agent.Agent, error) {
	record, err := h.sys.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var cfg agtconfig.AgentConfig
	if err := json.Unmarshal(record.Config, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	if token != "" {
		if cfg.Provider.Options == nil {
			cfg.Provider.Options = make(map[string]any)
		}
		cfg.Provider.Options["token"] = token
	}

	agt, err := agent.New(&cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	return agt, nil
}
```

#### Update Chat handler to pass token

```go
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	agt, err := h.constructAgent(r.Context(), id, req.Token)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	resp, err := agt.Chat(r.Context(), req.Prompt, req.Options)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(fmt.Errorf("%w: %v", ErrExecution, err)), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, resp)
}
```

#### Update ChatStream handler to pass token

```go
func (h *Handler) ChatStream(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	agt, err := h.constructAgent(r.Context(), id, req.Token)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	stream, err := agt.ChatStream(r.Context(), req.Prompt, req.Options)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(fmt.Errorf("%w: %v", ErrExecution, err)), err)
		return
	}

	h.writeSSEStream(w, r, stream)
}
```

#### Update Vision handler for multipart/form-data

```go
func (h *Handler) Vision(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	form, err := ParseVisionForm(r, 32<<20)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	agt, err := h.constructAgent(r.Context(), id, form.Token)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	resp, err := agt.Vision(r.Context(), form.Prompt, form.Images, form.Options)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(fmt.Errorf("%w: %v", ErrExecution, err)), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, resp)
}
```

#### Update VisionStream handler for multipart/form-data

```go
func (h *Handler) VisionStream(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	form, err := ParseVisionForm(r, 32<<20)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	agt, err := h.constructAgent(r.Context(), id, form.Token)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	stream, err := agt.VisionStream(r.Context(), form.Prompt, form.Images, form.Options)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(fmt.Errorf("%w: %v", ErrExecution, err)), err)
		return
	}

	h.writeSSEStream(w, r, stream)
}
```

#### Update Tools handler to pass token

```go
func (h *Handler) Tools(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	var req ToolsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	agt, err := h.constructAgent(r.Context(), id, req.Token)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	resp, err := agt.Tools(r.Context(), req.Prompt, req.Tools, req.Options)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(fmt.Errorf("%w: %v", ErrExecution, err)), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, resp)
}
```

#### Update Embed handler to pass token

```go
func (h *Handler) Embed(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	var req EmbedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	agt, err := h.constructAgent(r.Context(), id, req.Token)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	resp, err := agt.Embed(r.Context(), req.Input, req.Options)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(fmt.Errorf("%w: %v", ErrExecution, err)), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, resp)
}
```

#### Additional imports note

No additional imports needed in `handler.go` - the multipart parsing and base64 encoding are handled in `requests.go`.

---

## Validation Checklist

After implementation:

1. Run migrations: `go run ./cmd/migrate -dsn "$DATABASE_DSN" -up`
2. Start server: `go run ./cmd/server`
3. Test CRUD endpoints via curl or API client
4. Test execution endpoints with valid AgentConfig
5. Verify SSE streaming format with `/chat/stream` endpoint
6. Test vision endpoints with file upload:
   ```bash
   curl -X POST http://localhost:8080/api/agents/{id}/vision \
     -F "prompt=What do you see?" \
     -F "images=@/path/to/image.png"
   ```
7. Test token authentication with Azure agent config
