# Session 3e: Sample Workflows and Integration Tests

## Problem Context

The workflow execution infrastructure built in sessions 3a-3d provides the foundation for executing code-defined workflows. However, workflows currently cannot execute agent capabilities (Chat, Vision, Tools, Embed) because these are implemented in the Handler layer rather than the System layer.

This session moves agent execution capabilities from Handler to System, enabling workflows to make LLM calls through `runtime.Agents()`. We then validate the infrastructure with sample workflows that use live agents.

## Architecture Approach

### Agent System Expansion

```
Before:
  Handler.Chat() → constructAgent() → agent.Chat()

After:
  Handler.Chat() → System.Chat() → constructAgent() → agent.Chat()
                           ↑
  Workflow node → runtime.Agents().Chat()
```

### System Prompt Override

Workflows need to provide context-specific prompts for each node. The `opts["system_prompt"]` override allows this while preserving the stored system prompt as fallback.

```go
opts := map[string]any{
    "system_prompt": "You are an analytical assistant.",
}
resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
```

---

## Phase 1: Agent System Capabilities

### Step 1.1: Update System Interface

**File**: `internal/agents/system.go`

Add capability methods to the interface:

```go
package agents

import (
	"context"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/go-agents/pkg/agent"
	"github.com/JaimeStill/go-agents/pkg/response"
	"github.com/google/uuid"
)

type System interface {
	List(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Agent], error)
	Find(ctx context.Context, id uuid.UUID) (*Agent, error)
	Create(ctx context.Context, cmd CreateCommand) (*Agent, error)
	Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Agent, error)
	Delete(ctx context.Context, id uuid.UUID) error

	Chat(ctx context.Context, id uuid.UUID, prompt string, opts map[string]any, token string) (*response.ChatResponse, error)
	ChatStream(ctx context.Context, id uuid.UUID, prompt string, opts map[string]any, token string) (<-chan *response.StreamingChunk, error)
	Vision(ctx context.Context, id uuid.UUID, prompt string, images []string, opts map[string]any, token string) (*response.ChatResponse, error)
	VisionStream(ctx context.Context, id uuid.UUID, prompt string, images []string, opts map[string]any, token string) (<-chan *response.StreamingChunk, error)
	Tools(ctx context.Context, id uuid.UUID, prompt string, tools []agent.Tool, opts map[string]any, token string) (*response.ToolsResponse, error)
	Embed(ctx context.Context, id uuid.UUID, input string, opts map[string]any, token string) (*response.EmbeddingsResponse, error)
}
```

### Step 1.2: Implement Capabilities in Repository

**File**: `internal/agents/repository.go`

Add imports (at top of file):

```go
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
	"github.com/JaimeStill/go-agents/pkg/response"
	"github.com/google/uuid"
)
```

Add capability methods after `validateConfig`:

```go
func (r *repo) Chat(ctx context.Context, id uuid.UUID, prompt string, opts map[string]any, token string) (*response.ChatResponse, error) {
	agt, err := r.constructAgent(ctx, id, token, opts)
	if err != nil {
		return nil, err
	}

	resp, err := agt.Chat(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExecution, err)
	}

	return resp, nil
}

func (r *repo) ChatStream(ctx context.Context, id uuid.UUID, prompt string, opts map[string]any, token string) (<-chan *response.StreamingChunk, error) {
	agt, err := r.constructAgent(ctx, id, token, opts)
	if err != nil {
		return nil, err
	}

	stream, err := agt.ChatStream(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExecution, err)
	}

	return stream, nil
}

func (r *repo) Vision(ctx context.Context, id uuid.UUID, prompt string, images []string, opts map[string]any, token string) (*response.ChatResponse, error) {
	agt, err := r.constructAgent(ctx, id, token, opts)
	if err != nil {
		return nil, err
	}

	resp, err := agt.Vision(ctx, prompt, images)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExecution, err)
	}

	return resp, nil
}

func (r *repo) VisionStream(ctx context.Context, id uuid.UUID, prompt string, images []string, opts map[string]any, token string) (<-chan *response.StreamingChunk, error) {
	agt, err := r.constructAgent(ctx, id, token, opts)
	if err != nil {
		return nil, err
	}

	stream, err := agt.VisionStream(ctx, prompt, images)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExecution, err)
	}

	return stream, nil
}

func (r *repo) Tools(ctx context.Context, id uuid.UUID, prompt string, tools []agent.Tool, opts map[string]any, token string) (*response.ToolsResponse, error) {
	agt, err := r.constructAgent(ctx, id, token, opts)
	if err != nil {
		return nil, err
	}

	resp, err := agt.Tools(ctx, prompt, tools)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExecution, err)
	}

	return resp, nil
}

func (r *repo) Embed(ctx context.Context, id uuid.UUID, input string, opts map[string]any, token string) (*response.EmbeddingsResponse, error) {
	agt, err := r.constructAgent(ctx, id, token, opts)
	if err != nil {
		return nil, err
	}

	resp, err := agt.Embed(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExecution, err)
	}

	return resp, nil
}

func (r *repo) constructAgent(ctx context.Context, id uuid.UUID, token string, opts map[string]any) (agent.Agent, error) {
	record, err := r.Find(ctx, id)
	if err != nil {
		return nil, err
	}

	cfg := agtconfig.DefaultAgentConfig()

	var storedCfg agtconfig.AgentConfig
	if err := json.Unmarshal(record.Config, &storedCfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	cfg.Merge(&storedCfg)

	if systemPrompt, ok := opts["system_prompt"].(string); ok && systemPrompt != "" {
		cfg.SystemPrompt = systemPrompt
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

---

## Phase 2: Handler Refactoring

### Step 2.1: Update Handler to Delegate

**File**: `internal/agents/handler.go`

Remove imports no longer needed directly by handler:
- Remove: `"github.com/JaimeStill/go-agents/pkg/agent"`
- Remove: `agtconfig "github.com/JaimeStill/go-agents/pkg/config"`

Keep these imports:
```go
import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/routes"
	"github.com/JaimeStill/agent-lab/pkg/handlers"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/go-agents/pkg/response"
	"github.com/google/uuid"
)
```

Replace the Chat handler:

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

	resp, err := h.sys.Chat(r.Context(), id, req.Prompt, req.Options, req.Token)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, resp)
}
```

Replace the ChatStream handler:

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

	stream, err := h.sys.ChatStream(r.Context(), id, req.Prompt, req.Options, req.Token)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	h.writeSSEStream(w, r, stream)
}
```

Replace the Vision handler:

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

	resp, err := h.sys.Vision(r.Context(), id, form.Prompt, form.Images, form.Options, form.Token)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, resp)
}
```

Replace the VisionStream handler:

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

	stream, err := h.sys.VisionStream(r.Context(), id, form.Prompt, form.Images, form.Options, form.Token)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	h.writeSSEStream(w, r, stream)
}
```

Replace the Tools handler:

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

	resp, err := h.sys.Tools(r.Context(), id, req.Prompt, req.Tools, req.Options, req.Token)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, resp)
}
```

Replace the Embed handler:

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

	resp, err := h.sys.Embed(r.Context(), id, req.Input, req.Options, req.Token)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, resp)
}
```

Remove these functions from handler.go (they're now in repository.go):
- `constructAgent`
- `options`

---

## Phase 3: Sample Workflows

### Step 3.1: Create Samples Directory

Create directory: `internal/workflows/samples/`

### Step 3.2: Create Summarize Workflow

**File**: `internal/workflows/samples/summarize.go`

```go
package samples

import (
	"context"
	"fmt"

	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
	"github.com/google/uuid"
)

func init() {
	workflows.Register("summarize", summarizeFactory, "Summarizes input text using an AI agent")
}

func summarizeFactory(ctx context.Context, graph state.StateGraph, runtime *workflows.Runtime, params map[string]any) (state.State, error) {
	summarizeNode := state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		agentIDStr, ok := s.Get("agent_id")
		if !ok {
			return s, fmt.Errorf("agent_id is required")
		}

		agentID, err := uuid.Parse(agentIDStr.(string))
		if err != nil {
			return s, fmt.Errorf("invalid agent_id: %w", err)
		}

		text, ok := s.Get("text")
		if !ok {
			return s, fmt.Errorf("text is required")
		}

		token, _ := s.Get("token")
		tokenStr, _ := token.(string)

		systemPrompt := "You are a concise summarization assistant. Provide clear, brief summaries that capture the key points."
		if sp, ok := s.Get("system_prompt"); ok {
			if spStr, ok := sp.(string); ok && spStr != "" {
				systemPrompt = spStr
			}
		}

		opts := map[string]any{
			"system_prompt": systemPrompt,
		}

		prompt := fmt.Sprintf("Please summarize the following text:\n\n%s", text)

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, tokenStr)
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

### Step 3.3: Create Reasoning Workflow

**File**: `internal/workflows/samples/reasoning.go`

```go
package samples

import (
	"context"
	"fmt"

	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
	"github.com/google/uuid"
)

func init() {
	workflows.Register("reasoning", reasoningFactory, "Multi-step reasoning workflow that analyzes problems")
}

func reasoningFactory(ctx context.Context, graph state.StateGraph, runtime *workflows.Runtime, params map[string]any) (state.State, error) {
	analyzeNode := state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		agentID, token, err := extractAgentParams(s)
		if err != nil {
			return s, err
		}

		problem, ok := s.Get("problem")
		if !ok {
			return s, fmt.Errorf("problem is required")
		}

		systemPrompt := "You are an analytical assistant. Break down problems into their key components and identify the important elements."
		if sp, ok := s.Get("analyze_system_prompt"); ok {
			if spStr, ok := sp.(string); ok && spStr != "" {
				systemPrompt = spStr
			}
		}

		opts := map[string]any{
			"system_prompt": systemPrompt,
		}

		prompt := fmt.Sprintf("Analyze this problem and identify its key components:\n\n%s", problem)

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
		if err != nil {
			return s, fmt.Errorf("analyze failed: %w", err)
		}

		return s.Set("analysis", resp.Content()), nil
	})

	reasonNode := state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		agentID, token, err := extractAgentParams(s)
		if err != nil {
			return s, err
		}

		analysis, ok := s.Get("analysis")
		if !ok {
			return s, fmt.Errorf("analysis not found in state")
		}

		systemPrompt := "You are a logical reasoning assistant. Think step-by-step and explain your reasoning clearly."
		if sp, ok := s.Get("reason_system_prompt"); ok {
			if spStr, ok := sp.(string); ok && spStr != "" {
				systemPrompt = spStr
			}
		}

		opts := map[string]any{
			"system_prompt": systemPrompt,
		}

		prompt := fmt.Sprintf("Given this analysis:\n\n%s\n\nWhat are the logical steps to solve this problem?", analysis)

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
		if err != nil {
			return s, fmt.Errorf("reason failed: %w", err)
		}

		return s.Set("reasoning", resp.Content()), nil
	})

	concludeNode := state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		agentID, token, err := extractAgentParams(s)
		if err != nil {
			return s, err
		}

		reasoning, ok := s.Get("reasoning")
		if !ok {
			return s, fmt.Errorf("reasoning not found in state")
		}

		systemPrompt := "You are a concise assistant. Provide clear, direct conclusions based on the reasoning provided."
		if sp, ok := s.Get("conclude_system_prompt"); ok {
			if spStr, ok := sp.(string); ok && spStr != "" {
				systemPrompt = spStr
			}
		}

		opts := map[string]any{
			"system_prompt": systemPrompt,
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

func extractAgentParams(s state.State) (uuid.UUID, string, error) {
	agentIDStr, ok := s.Get("agent_id")
	if !ok {
		return uuid.Nil, "", fmt.Errorf("agent_id is required")
	}

	agentID, err := uuid.Parse(agentIDStr.(string))
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("invalid agent_id: %w", err)
	}

	token, _ := s.Get("token")
	tokenStr, _ := token.(string)

	return agentID, tokenStr, nil
}
```

---

## Phase 4: Server Integration

### Step 4.1: Add Samples Import

**File**: `cmd/server/server.go`

Add blank import to trigger sample workflow registration:

```go
import (
	// ... existing imports ...

	_ "github.com/JaimeStill/agent-lab/internal/workflows/samples"
)
```

---

## Phase 5: Validation

### Step 5.1: Verify Compilation

```bash
go vet ./...
```

### Step 5.2: Run Existing Tests

```bash
go test ./tests/...
```

### Step 5.3: Start Server and Test

```bash
# Start server
go run ./cmd/server

# In another terminal:

# List agents to get an agent ID
curl http://localhost:8080/api/agents

# List workflows (should show summarize and reasoning)
curl http://localhost:8080/api/workflows

# Execute summarize workflow
curl -X POST http://localhost:8080/api/workflows/summarize/execute \
  -H "Content-Type: application/json" \
  -d '{"params": {"agent_id": "<AGENT_UUID>", "text": "The quick brown fox jumps over the lazy dog. This sentence contains every letter of the alphabet and is commonly used for typing practice."}}'

# Execute reasoning workflow
curl -X POST http://localhost:8080/api/workflows/reasoning/execute \
  -H "Content-Type: application/json" \
  -d '{"params": {"agent_id": "<AGENT_UUID>", "problem": "If all roses are flowers and some flowers fade quickly, can we conclude that some roses fade quickly?"}}'

# Get stages for a run
curl http://localhost:8080/api/workflows/runs/<RUN_ID>/stages
```

---

## Summary

This session:
1. Expands `agents.System` with capability methods (Chat, ChatStream, Vision, VisionStream, Tools, Embed)
2. Implements system prompt override via `opts["system_prompt"]`
3. Refactors Handler to delegate to System
4. Creates two sample workflows demonstrating live agent integration
5. Validates M3 infrastructure with actual LLM calls
