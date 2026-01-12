# Session 4a: Workflow Profiles Infrastructure

## Problem Context

Session 4a establishes workflow profiles for configurable stage-level settings. This enables A/B testing and prompt iteration without code changes by storing agent and prompt configurations in the database.

Currently, workflows have hardcoded system prompts embedded in node functions. This session:
1. Creates the profiles domain for database-stored configurations
2. Moves workflows to top-level `workflows/` directory with profile separation
3. Integrates profiles into the workflow Runtime
4. Establishes the pattern: explicit `profile_id` → DB profile; otherwise → hardcoded defaults

## Architecture Approach

### Profile Resolution Flow

```
Workflow Execution
       │
       ▼
┌─────────────────────────┐
│ profile_id in params?   │
└─────────────────────────┘
       │
  yes  │  no
       ▼
┌──────────────┐  ┌────────────────────┐
│ Load from DB │  │ Use DefaultProfile │
│              │  │ (from profile.go)  │
└──────────────┘  └────────────────────┘
       │                    │
       └────────┬───────────┘
                ▼
         Use Profile for
         stage configs
```

### Key Design Decisions

- **No default profiles**: Either explicit `profile_id` or hardcoded fallback
- **Full replacement**: DB profile completely replaces hardcoded config
- **Profile per workflow package**: Each workflow has `profile.go` with types + defaults
- **Upsert for stages**: `SetStage` creates or updates by `(profile_id, stage_name)`

---

## Phase 1: Database Migration

### 1.1 Create `cmd/migrate/migrations/000007_profiles.up.sql`

```sql
CREATE TABLE profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_name TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(workflow_name, name)
);

CREATE INDEX idx_profiles_workflow_name ON profiles(workflow_name);

CREATE TABLE profile_stages (
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    stage_name TEXT NOT NULL,
    agent_id UUID REFERENCES agents(id),
    system_prompt TEXT,
    options JSONB,
    PRIMARY KEY (profile_id, stage_name)
);
```

### 1.2 Create `cmd/migrate/migrations/000007_profiles.down.sql`

```sql
DROP TABLE IF EXISTS profile_stages;
DROP TABLE IF EXISTS profiles;
```

---

## Phase 2: Profile Domain Types

### 2.1 Create `internal/profiles/profile.go`

```go
package profiles

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	ID           uuid.UUID `json:"id"`
	WorkflowName string    `json:"workflow_name"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ProfileStage struct {
	ProfileID    uuid.UUID       `json:"profile_id"`
	StageName    string          `json:"stage_name"`
	AgentID      *uuid.UUID      `json:"agent_id,omitempty"`
	SystemPrompt *string         `json:"system_prompt,omitempty"`
	Options      json.RawMessage `json:"options,omitempty"`
}

type ProfileWithStages struct {
	Profile
	Stages []ProfileStage `json:"stages"`
}

func (p *ProfileWithStages) Stage(name string) *ProfileStage {
	for i, s := range p.Stages {
		if s.StageName == name {
			return &p.Stages[i]
		}
	}
	return nil
}

func NewProfileWithStages(stages ...ProfileStage) *ProfileWithStages {
	return &ProfileWithStages{
		Stages: stages,
	}
}

type CreateProfileCommand struct {
	WorkflowName string  `json:"workflow_name"`
	Name         string  `json:"name"`
	Description  *string `json:"description,omitempty"`
}

type UpdateProfileCommand struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

type SetProfileStageCommand struct {
	StageName    string          `json:"stage_name"`
	AgentID      *uuid.UUID      `json:"agent_id,omitempty"`
	SystemPrompt *string         `json:"system_prompt,omitempty"`
	Options      json.RawMessage `json:"options,omitempty"`
}
```

### 2.2 Create `internal/profiles/errors.go`

```go
package profiles

import (
	"errors"
	"net/http"
)

var (
	ErrNotFound      = errors.New("profile not found")
	ErrDuplicate     = errors.New("profile name already exists for workflow")
	ErrStageNotFound = errors.New("stage not found")
)

func MapHTTPStatus(err error) int {
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrStageNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrDuplicate) {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}
```

---

## Phase 3: Domain Infrastructure

### 3.1 Create `internal/profiles/mapping.go`

```go
package profiles

import (
	"encoding/json"
	"net/url"

	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/agent-lab/pkg/repository"
	"github.com/google/uuid"
)

var profileProjection = query.
	NewProjectionMap("public", "profiles", "p").
	Project("id", "ID").
	Project("workflow_name", "WorkflowName").
	Project("name", "Name").
	Project("description", "Description").
	Project("created_at", "CreatedAt").
	Project("updated_at", "UpdatedAt")

var stageProjection = query.
	NewProjectionMap("public", "profile_stages", "ps").
	Project("profile_id", "ProfileID").
	Project("stage_name", "StageName").
	Project("agent_id", "AgentID").
	Project("system_prompt", "SystemPrompt").
	Project("options", "Options")

var defaultSort = query.SortField{Field: "Name"}

func scanProfile(s repository.Scanner) (Profile, error) {
	var p Profile
	err := s.Scan(
		&p.ID, &p.WorkflowName, &p.Name,
		&p.Description, &p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

func scanProfileStage(s repository.Scanner) (ProfileStage, error) {
	var ps ProfileStage
	var opts []byte
	err := s.Scan(
		&ps.ProfileID, &ps.StageName,
		&ps.AgentID, &ps.SystemPrompt, &opts,
	)
	if len(opts) > 0 {
		ps.Options = json.RawMessage(opts)
	}
	return ps, err
}

type Filters struct {
	WorkflowName *string
}

func FiltersFromQuery(values url.Values) Filters {
	var workflowName *string
	if wn := values.Get("workflow_name"); wn != "" {
		workflowName = &wn
	}
	return Filters{
		WorkflowName: workflowName,
	}
}

func (f Filters) Apply(b *query.Builder) *query.Builder {
	return b.WhereEqual("WorkflowName", f.WorkflowName)
}
```

### 3.2 Create `internal/profiles/system.go`

```go
package profiles

import (
	"context"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

type System interface {
	List(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Profile], error)
	Find(ctx context.Context, id uuid.UUID) (*ProfileWithStages, error)
	Create(ctx context.Context, cmd CreateProfileCommand) (*Profile, error)
	Update(ctx context.Context, id uuid.UUID, cmd UpdateProfileCommand) (*Profile, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SetStage(ctx context.Context, profileID uuid.UUID, cmd SetProfileStageCommand) (*ProfileStage, error)
	DeleteStage(ctx context.Context, profileID uuid.UUID, stageName string) error
}
```

### 3.3 Create `internal/profiles/repository.go`

```go
package profiles

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/agent-lab/pkg/repository"
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
		logger:     logger.With("system", "profiles"),
		pagination: pagination,
	}
}

func (r *repo) List(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Profile], error) {
	page.Normalize(r.pagination)

	qb := query.
		NewBuilder(profileProjection, defaultSort).
		WhereSearch(page.Search, "Name")

	filters.Apply(qb)

	if len(page.Sort) > 0 {
		qb.OrderByFields(page.Sort)
	}

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count profiles: %w", err)
	}

	pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	profiles, err := repository.QueryMany(ctx, r.db, pageSQL, pageArgs, scanProfile)
	if err != nil {
		return nil, fmt.Errorf("query profiles: %w", err)
	}

	result := pagination.NewPageResult(profiles, total, page.Page, page.PageSize)
	return &result, nil
}

func (r *repo) Find(ctx context.Context, id uuid.UUID) (*ProfileWithStages, error) {
	profileSQL, profileArgs := query.NewBuilder(profileProjection, defaultSort).
		WhereEqual("ID", &id).
		Build()

	profile, err := repository.QueryOne(ctx, r.db, profileSQL, profileArgs, scanProfile)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	stageSQL, stageArgs := query.NewBuilder(stageProjection, query.SortField{Field: "StageName"}).
		WhereEqual("ProfileID", &id).
		Build()

	stages, err := repository.QueryMany(ctx, r.db, stageSQL, stageArgs, scanProfileStage)
	if err != nil {
		return nil, fmt.Errorf("query stages: %w", err)
	}

	return &ProfileWithStages{
		Profile: profile,
		Stages:  stages,
	}, nil
}

func (r *repo) Create(ctx context.Context, cmd CreateProfileCommand) (*Profile, error) {
	q := `
		INSERT INTO profiles (workflow_name, name, description)
		VALUES ($1, $2, $3)
		RETURNING id, workflow_name, name, description, created_at, updated_at`

	profile, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Profile, error) {
		return repository.QueryOne(ctx, tx, q, []any{cmd.WorkflowName, cmd.Name, cmd.Description}, scanProfile)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("profile created", "id", profile.ID, "workflow", profile.WorkflowName, "name", profile.Name)
	return &profile, nil
}

func (r *repo) Update(ctx context.Context, id uuid.UUID, cmd UpdateProfileCommand) (*Profile, error) {
	q := `
		UPDATE profiles
		SET name = $2, description = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING id, workflow_name, name, description, created_at, updated_at`

	profile, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Profile, error) {
		return repository.QueryOne(ctx, tx, q, []any{id, cmd.Name, cmd.Description}, scanProfile)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("profile updated", "id", profile.ID, "name", profile.Name)
	return &profile, nil
}

func (r *repo) Delete(ctx context.Context, id uuid.UUID) error {
	q := `DELETE FROM profiles WHERE id = $1`

	if err := repository.ExecExpectOne(ctx, r.db, q, id); err != nil {
		return repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("profile deleted", "id", id)
	return nil
}

func (r *repo) SetStage(ctx context.Context, profileID uuid.UUID, cmd SetProfileStageCommand) (*ProfileStage, error) {
	if _, err := r.Find(ctx, profileID); err != nil {
		return nil, err
	}

	q := `
		INSERT INTO profile_stages (profile_id, stage_name, agent_id, system_prompt, options)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (profile_id, stage_name)
		DO UPDATE SET agent_id = $3, system_prompt = $4, options = $5
		RETURNING profile_id, stage_name, agent_id, system_prompt, options`

	stage, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (ProfileStage, error) {
		return repository.QueryOne(ctx, tx, q, []any{
			profileID, cmd.StageName, cmd.AgentID, cmd.SystemPrompt, cmd.Options,
		}, scanProfileStage)
	})

	if err != nil {
		return nil, fmt.Errorf("set stage: %w", err)
	}

	r.logger.Info("stage set", "profile_id", profileID, "stage", cmd.StageName)
	return &stage, nil
}

func (r *repo) DeleteStage(ctx context.Context, profileID uuid.UUID, stageName string) error {
	q := `DELETE FROM profile_stages WHERE profile_id = $1 AND stage_name = $2`

	if err := repository.ExecExpectOne(ctx, r.db, q, profileID, stageName); err != nil {
		return repository.MapError(err, ErrStageNotFound, ErrDuplicate)
	}

	r.logger.Info("stage deleted", "profile_id", profileID, "stage", stageName)
	return nil
}
```

---

## Phase 4: HTTP Layer

### 4.1 Create `internal/profiles/handler.go`

Handler uses OpenAPI spec references prepared by AI:

```go
package profiles

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
		Prefix:      "/api/profiles",
		Tags:        []string{"Profiles"},
		Description: "Workflow profile configuration management",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.List, OpenAPI: Spec.List},
			{Method: "POST", Pattern: "", Handler: h.Create, OpenAPI: Spec.Create},
			{Method: "GET", Pattern: "/{id}", Handler: h.Find, OpenAPI: Spec.Find},
			{Method: "PUT", Pattern: "/{id}", Handler: h.Update, OpenAPI: Spec.Update},
			{Method: "DELETE", Pattern: "/{id}", Handler: h.Delete, OpenAPI: Spec.Delete},
			{Method: "POST", Pattern: "/{id}/stages", Handler: h.SetStage, OpenAPI: Spec.SetStage},
			{Method: "DELETE", Pattern: "/{id}/stages/{stage}", Handler: h.DeleteStage, OpenAPI: Spec.DeleteStage},
		},
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page := pagination.PageRequestFromQuery(r.URL.Query(), h.pagination)
	filters := FiltersFromQuery(r.URL.Query())

	result, err := h.sys.List(r.Context(), page, filters)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusInternalServerError, err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, result)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var cmd CreateProfileCommand
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

func (h *Handler) Find(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	result, err := h.sys.Find(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, result)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	var cmd UpdateProfileCommand
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

func (h *Handler) SetStage(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	var cmd SetProfileStageCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	result, err := h.sys.SetStage(r.Context(), id, cmd)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, result)
}

func (h *Handler) DeleteStage(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	stageName := r.PathValue("stage")
	if stageName == "" {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrStageNotFound)
		return
	}

	if err := h.sys.DeleteStage(r.Context(), id, stageName); err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

---

## Phase 5: Workflow Migration

### 5.1 Create `internal/workflows/profile.go`

Shared helpers for workflow profile loading:

```go
package workflows

import (
	"context"
	"fmt"

	"github.com/JaimeStill/agent-lab/internal/profiles"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
	"github.com/google/uuid"
)

func ExtractAgentParams(s state.State, stage *profiles.ProfileStage) (uuid.UUID, string, error) {
	var agentID uuid.UUID

	if stage != nil && stage.AgentID != nil {
		agentID = *stage.AgentID
	} else {
		agentIDStr, ok := s.Get("agent_id")
		if !ok {
			return uuid.Nil, "", fmt.Errorf("agent_id is required")
		}
		var err error
		agentID, err = uuid.Parse(agentIDStr.(string))
		if err != nil {
			return uuid.Nil, "", fmt.Errorf("invalid agent_id: %w", err)
		}
	}

	tkn, _ := s.Get("token")
	token, _ := tkn.(string)

	return agentID, token, nil
}

func LoadProfile(ctx context.Context, rt *Runtime, params map[string]any, defaultProfile *profiles.ProfileWithStages) (*profiles.ProfileWithStages, error) {
	if profileIDStr, ok := params["profile_id"].(string); ok {
		profileID, err := uuid.Parse(profileIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid profile_id: %w", err)
		}
		return rt.Profiles().Find(ctx, profileID)
	}
	return defaultProfile, nil
}
```

### 5.2 Create `workflows/summarize/profile.go`

```go
package summarize

import "github.com/JaimeStill/agent-lab/internal/profiles"

func DefaultProfile() *profiles.ProfileWithStages {
	summarizePrompt := "You are a concise summarization assistant. Provide clear, brief summaries that capture the key points."

	return profiles.NewProfileWithStages(
		profiles.ProfileStage{
			StageName:    "summarize",
			SystemPrompt: &summarizePrompt,
		},
	)
}
```

### 5.3 Create `workflows/summarize/summarize.go`

Move from `internal/workflows/samples/summarize.go` and refactor:

```go
package summarize

import (
	"context"
	"fmt"

	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
)

func init() {
	workflows.Register("summarize", factory, "Summarizes input text using an AI agent")
}

func factory(ctx context.Context, graph state.StateGraph, runtime *workflows.Runtime, params map[string]any) (state.State, error) {
	profile, err := workflows.LoadProfile(ctx, runtime, params, DefaultProfile())
	if err != nil {
		return state.State{}, err
	}

	summarizeNode := state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		stage := profile.Stage("summarize")

		agentID, token, err := workflows.ExtractAgentParams(s, stage)
		if err != nil {
			return s, err
		}

		text, ok := s.Get("text")
		if !ok {
			return s, fmt.Errorf("text is required")
		}

		opts := map[string]any{
			"system_prompt": *stage.SystemPrompt,
		}

		prompt := fmt.Sprintf("Please summarize the following text:\n\n%s", text)

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
		if err != nil {
			return s, fmt.Errorf("chat failed: %w", err)
		}

		return s.Set("summary", resp.Content()), nil
	})

	if err := graph.AddNode("summarize", summarizeNode); err != nil {
		return state.State{}, err
	}

	if err := graph.SetEntryPoint("summarize"); err != nil {
		return state.State{}, err
	}

	if err := graph.SetExitPoint("summarize"); err != nil {
		return state.State{}, err
	}

	initialState := state.New(nil)
	for k, v := range params {
		initialState = initialState.Set(k, v)
	}

	return initialState, nil
}
```

### 5.4 Create `workflows/reasoning/profile.go`

```go
package reasoning

import "github.com/JaimeStill/agent-lab/internal/profiles"

func DefaultProfile() *profiles.ProfileWithStages {
	analyzePrompt := "You are an analytical assistant. Break down problems into their key components and identify the important elements."
	reasonPrompt := "You are a logical reasoning assistant. Think step-by-step and explain your reasoning clearly."
	concludePrompt := "You are a concise assistant. Provide clear, direct conclusions based on the reasoning provided."

	return profiles.NewProfileWithStages(
		profiles.ProfileStage{StageName: "analyze", SystemPrompt: &analyzePrompt},
		profiles.ProfileStage{StageName: "reason", SystemPrompt: &reasonPrompt},
		profiles.ProfileStage{StageName: "conclude", SystemPrompt: &concludePrompt},
	)
}
```

### 5.5 Create `workflows/reasoning/reasoning.go`

Move from `internal/workflows/samples/reasoning.go` and refactor:

```go
package reasoning

import (
	"context"
	"fmt"

	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
)

func init() {
	workflows.Register("reasoning", factory, "Multi-step reasoning workflow that analyzes problems")
}

func factory(ctx context.Context, graph state.StateGraph, runtime *workflows.Runtime, params map[string]any) (state.State, error) {
	profile, err := workflows.LoadProfile(ctx, runtime, params, DefaultProfile())
	if err != nil {
		return state.State{}, err
	}

	analyzeNode := state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		stage := profile.Stage("analyze")

		agentID, token, err := workflows.ExtractAgentParams(s, stage)
		if err != nil {
			return s, err
		}

		problem, ok := s.Get("problem")
		if !ok {
			return s, fmt.Errorf("problem is required")
		}

		opts := map[string]any{
			"system_prompt": *stage.SystemPrompt,
		}

		prompt := fmt.Sprintf("Analyze this problem and identify its key components:\n\n%s", problem)

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
		if err != nil {
			return s, fmt.Errorf("analyze failed: %w", err)
		}

		return s.Set("analysis", resp.Content()), nil
	})

	reasonNode := state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		stage := profile.Stage("reason")

		agentID, token, err := workflows.ExtractAgentParams(s, stage)
		if err != nil {
			return s, err
		}

		analysis, ok := s.Get("analysis")
		if !ok {
			return s, fmt.Errorf("analysis not found in state")
		}

		opts := map[string]any{
			"system_prompt": *stage.SystemPrompt,
		}

		prompt := fmt.Sprintf("Given this analysis:\n\n%s\n\nWhat are the logical steps to solve this problem?", analysis)

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
		if err != nil {
			return s, fmt.Errorf("reason failed: %w", err)
		}

		return s.Set("reasoning", resp.Content()), nil
	})

	concludeNode := state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		stage := profile.Stage("conclude")

		agentID, token, err := workflows.ExtractAgentParams(s, stage)
		if err != nil {
			return s, err
		}

		reasoning, ok := s.Get("reasoning")
		if !ok {
			return s, fmt.Errorf("reasoning not found in state")
		}

		opts := map[string]any{
			"system_prompt": *stage.SystemPrompt,
		}

		prompt := fmt.Sprintf("Based on this reasoning:\n\n%s\n\nWhat is the conclusion?", reasoning)

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
		if err != nil {
			return s, fmt.Errorf("conclude failed: %w", err)
		}

		return s.Set("conclusion", resp.Content()), nil
	})

	if err := graph.AddNode("analyze", analyzeNode); err != nil {
		return state.State{}, err
	}

	if err := graph.AddNode("reason", reasonNode); err != nil {
		return state.State{}, err
	}

	if err := graph.AddNode("conclude", concludeNode); err != nil {
		return state.State{}, err
	}

	if err := graph.AddEdge("analyze", "reason", nil); err != nil {
		return state.State{}, err
	}

	if err := graph.AddEdge("reason", "conclude", nil); err != nil {
		return state.State{}, err
	}

	if err := graph.SetEntryPoint("analyze"); err != nil {
		return state.State{}, err
	}

	if err := graph.SetExitPoint("conclude"); err != nil {
		return state.State{}, err
	}

	initialState := state.New(nil)
	for k, v := range params {
		initialState = initialState.Set(k, v)
	}

	return initialState, nil
}
```

### 5.6 Create `workflows/init.go`

Aggregates workflow package inits for single import:

```go
package workflows

import (
	_ "github.com/JaimeStill/agent-lab/workflows/reasoning"
	_ "github.com/JaimeStill/agent-lab/workflows/summarize"
)
```

### 5.7 Delete `internal/workflows/samples/` Directory

After creating the new workflow files, delete:
- `internal/workflows/samples/summarize.go`
- `internal/workflows/samples/reasoning.go`
- `internal/workflows/samples/` directory

---

## Phase 6: Integration

### 6.1 Update `internal/workflows/runtime.go`

Add profiles to Runtime:

```go
package workflows

import (
	"log/slog"

	"github.com/JaimeStill/agent-lab/internal/agents"
	"github.com/JaimeStill/agent-lab/internal/documents"
	"github.com/JaimeStill/agent-lab/internal/images"
	"github.com/JaimeStill/agent-lab/internal/profiles"
)

type Runtime struct {
	agents    agents.System
	documents documents.System
	images    images.System
	profiles  profiles.System
	logger    *slog.Logger
}

func NewRuntime(
	agents agents.System,
	documents documents.System,
	images images.System,
	profiles profiles.System,
	logger *slog.Logger,
) *Runtime {
	return &Runtime{
		agents:    agents,
		documents: documents,
		images:    images,
		profiles:  profiles,
		logger:    logger,
	}
}

func (r *Runtime) Agents() agents.System { return r.agents }

func (r *Runtime) Documents() documents.System { return r.documents }

func (r *Runtime) Images() images.System { return r.images }

func (r *Runtime) Profiles() profiles.System { return r.profiles }

func (r *Runtime) Logger() *slog.Logger { return r.logger }
```

### 6.2 Update `cmd/server/domain.go`

Add profiles to imports and Domain struct:

In imports, add:
```go
"github.com/JaimeStill/agent-lab/internal/profiles"
```

Update Domain struct:
```go
type Domain struct {
	Providers providers.System
	Agents    agents.System
	Documents documents.System
	Images    images.System
	Profiles  profiles.System
	Workflows workflows.System
}
```

In NewDomain, after imagesSys and before workflowRuntime:
```go
profilesSys := profiles.New(
	runtime.Database.Connection(),
	runtime.Logger,
	runtime.Pagination,
)
```

Update workflowRuntime creation:
```go
workflowRuntime := workflows.NewRuntime(
	agentsSys,
	documentsSys,
	imagesSys,
	profilesSys,
	runtime.Logger,
)
```

Update return statement:
```go
return &Domain{
	Providers: providersSys,
	Agents:    agentsSys,
	Documents: documentsSys,
	Images:    imagesSys,
	Profiles:  profilesSys,
	Workflows: workflowsSys,
}
```

### 6.3 Update `cmd/server/routes.go`

Add profiles to imports:
```go
"github.com/JaimeStill/agent-lab/internal/profiles"
```

After images handler registration, add:
```go
profilesHandler := profiles.NewHandler(
	domain.Profiles,
	runtime.Logger,
	runtime.Pagination,
)
r.RegisterGroup(profilesHandler.Routes())
```

In AddSchemas, add:
```go
components.AddSchemas(profiles.Spec.Schemas())
```

### 6.4 Update `cmd/server/server.go`

Update blank import for workflows:

```go
// Before
_ "github.com/JaimeStill/agent-lab/internal/workflows/samples"

// After
_ "github.com/JaimeStill/agent-lab/workflows"
```

---

## Phase 7: Validation

### 7.1 Verify Compilation

```bash
go vet ./...
```

### 7.2 Run Database Migration

```bash
go run ./cmd/migrate up
```

### 7.3 Start Server and Test

```bash
# Start server
go run ./cmd/server

# List profiles (should be empty)
curl http://localhost:8080/api/profiles

# Create a profile
curl -X POST http://localhost:8080/api/profiles \
  -H "Content-Type: application/json" \
  -d '{"workflow_name": "summarize", "name": "verbose-v1", "description": "More detailed summaries"}'

# Set a stage for the profile
curl -X POST http://localhost:8080/api/profiles/{profile_id}/stages \
  -H "Content-Type: application/json" \
  -d '{"stage_name": "summarize", "system_prompt": "You are a detailed summarization assistant. Provide comprehensive summaries that include context and nuance."}'

# Get profile with stages
curl http://localhost:8080/api/profiles/{profile_id}

# Execute workflow with default profile
curl -X POST http://localhost:8080/api/workflows/summarize/execute \
  -H "Content-Type: application/json" \
  -d '{"params": {"agent_id": "<AGENT_UUID>", "text": "The quick brown fox..."}}'

# Execute workflow with custom profile
curl -X POST http://localhost:8080/api/workflows/summarize/execute \
  -H "Content-Type: application/json" \
  -d '{"params": {"agent_id": "<AGENT_UUID>", "profile_id": "{profile_id}", "text": "The quick brown fox..."}}'

# List workflows (should show summarize and reasoning)
curl http://localhost:8080/api/workflows
```

---

## Summary

This session:
1. Creates database tables for workflow profiles and stage configurations
2. Implements the profiles domain with full CRUD + stage management
3. Moves workflows to top-level `workflows/` directory with profile separation
4. Establishes the profile resolution pattern: explicit `profile_id` → DB profile; otherwise → hardcoded defaults
5. Integrates profiles into the workflow Runtime for A/B testing capability
