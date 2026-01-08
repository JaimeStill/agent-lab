package agents

import (
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

const visionSize int64 = 32 << 20

// Handler provides HTTP handlers for agent CRUD operations and execution endpoints.
type Handler struct {
	sys        System
	logger     *slog.Logger
	pagination pagination.Config
}

// NewHandler creates a new agents HTTP handler.
func NewHandler(sys System, logger *slog.Logger, pagination pagination.Config) *Handler {
	return &Handler{
		sys:        sys,
		logger:     logger,
		pagination: pagination,
	}
}

// Routes returns the route group configuration for agent endpoints.
func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix:      "/api/agents",
		Tags:        []string{"Agents"},
		Description: "Agent configuration and execution",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.List, OpenAPI: Spec.List},
			{Method: "GET", Pattern: "/{id}", Handler: h.Find, OpenAPI: Spec.Find},
			{Method: "POST", Pattern: "/search", Handler: h.Search, OpenAPI: Spec.Search},
			{Method: "POST", Pattern: "", Handler: h.Create, OpenAPI: Spec.Create},
			{Method: "PUT", Pattern: "/{id}", Handler: h.Update, OpenAPI: Spec.Update},
			{Method: "DELETE", Pattern: "/{id}", Handler: h.Delete, OpenAPI: Spec.Delete},
			{Method: "POST", Pattern: "/{id}/chat", Handler: h.Chat, OpenAPI: Spec.Chat},
			{Method: "POST", Pattern: "/{id}/chat/stream", Handler: h.ChatStream, OpenAPI: Spec.ChatStream},
			{Method: "POST", Pattern: "/{id}/vision", Handler: h.Vision, OpenAPI: Spec.Vision},
			{Method: "POST", Pattern: "/{id}/vision/stream", Handler: h.VisionStream, OpenAPI: Spec.VisionStream},
			{Method: "POST", Pattern: "/{id}/tools", Handler: h.Tools, OpenAPI: Spec.Tools},
			{Method: "POST", Pattern: "/{id}/embed", Handler: h.Embed, OpenAPI: Spec.Embed},
		},
	}
}

// List handles GET /api/agents to retrieve a paginated list of agents.
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

// Find handles GET /api/agents/{id} to retrieve a single agent.
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

// Search handles POST /api/agents/search to search agents with request body parameters.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var page pagination.PageRequest
	if err := json.NewDecoder(r.Body).Decode(&page); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	filters := FiltersFromQuery(r.URL.Query())

	result, err := h.sys.List(r.Context(), page, filters)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusInternalServerError, err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, result)
}

// Create handles POST /api/agents to create a new agent.
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

// Update handles PUT /api/agents/{id} to update an existing agent.
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

// Delete handles DELETE /api/agents/{id} to delete an agent.
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

// Chat handles POST /api/agents/{id}/chat to execute a chat completion.
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

// ChatStream handles POST /api/agents/{id}/chat/stream to execute a streaming chat completion.
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

// Vision handles POST /api/agents/{id}/vision to execute vision analysis on uploaded images.
func (h *Handler) Vision(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	form, err := ParseVisionForm(r, visionSize)
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

// VisionStream handles POST /api/agents/{id}/vision/stream to execute streaming vision analysis.
func (h *Handler) VisionStream(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	form, err := ParseVisionForm(r, visionSize)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	stream, err := h.sys.VisionStream(r.Context(), id, form.Prompt, form.Images, form.Options, form.Token)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(fmt.Errorf("%w: %v", ErrExecution, err)), err)
		return
	}

	h.writeSSEStream(w, r, stream)
}

// Tools handles POST /api/agents/{id}/tools to execute tool-calling with provided tool definitions.
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
		handlers.RespondError(w, h.logger, MapHTTPStatus(fmt.Errorf("%w: %v", ErrExecution, err)), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, resp)
}

// Embed handles POST /api/agents/{id}/embed to generate text embeddings.
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
		handlers.RespondError(w, h.logger, MapHTTPStatus(fmt.Errorf("%w: %v", ErrExecution, err)), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, resp)
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
