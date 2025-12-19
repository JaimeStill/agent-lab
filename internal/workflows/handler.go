package workflows

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/routes"
	"github.com/JaimeStill/agent-lab/pkg/handlers"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

// ExecuteRequest represents the request body for workflow execution.
type ExecuteRequest struct {
	Params map[string]any `json:"params,omitempty"`
}

// Handler provides HTTP handlers for workflow operations.
type Handler struct {
	sys        System
	logger     *slog.Logger
	pagination pagination.Config
}

// NewHandler creates a Handler with the provided dependencies.
func NewHandler(sys System, logger *slog.Logger, pagination pagination.Config) *Handler {
	return &Handler{
		sys:        sys,
		logger:     logger,
		pagination: pagination,
	}
}

// Routes returns the route group for workflow endpoints.
func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix:      "/api/workflows",
		Tags:        []string{"Workflows"},
		Description: "Workflow execution and management",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.ListWorkflows, OpenAPI: Spec.ListWorkflows},
			{Method: "POST", Pattern: "/{name}/execute", Handler: h.Execute, OpenAPI: Spec.Execute},
			{Method: "POST", Pattern: "/{name}/execute/stream", Handler: h.ExecuteStream, OpenAPI: Spec.ExecuteStream},
		},
		Children: []routes.Group{
			{
				Prefix:      "/runs",
				Tags:        []string{"Runs"},
				Description: "Workflow run inspection and control",
				Routes: []routes.Route{
					{Method: "GET", Pattern: "", Handler: h.ListRuns, OpenAPI: Spec.ListRuns},
					{Method: "GET", Pattern: "/{id}", Handler: h.FindRun, OpenAPI: Spec.FindRun},
					{Method: "GET", Pattern: "/{id}/stages", Handler: h.GetStages, OpenAPI: Spec.GetStages},
					{Method: "GET", Pattern: "/{id}/decisions", Handler: h.GetDecisions, OpenAPI: Spec.GetDecisions},
					{Method: "DELETE", Pattern: "/{id}", Handler: h.DeleteRun, OpenAPI: Spec.DeleteRun},
					{Method: "POST", Pattern: "/{id}/cancel", Handler: h.Cancel, OpenAPI: Spec.Cancel},
					{Method: "POST", Pattern: "/{id}/resume", Handler: h.Resume, OpenAPI: Spec.Resume},
				},
			},
		},
	}
}

func (h *Handler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	workflows := h.sys.ListWorkflows()
	handlers.RespondJSON(w, http.StatusOK, workflows)
}

func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
	}

	run, err := h.sys.Execute(r.Context(), name, req.Params)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, run)
}

func (h *Handler) ExecuteStream(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	events, run, err := h.sys.ExecuteStream(r.Context(), name, req.Params)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Run-ID", run.ID.String())
	w.WriteHeader(http.StatusOK)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	for event := range events {
		select {
		case <-r.Context().Done():
			return
		default:
		}

		data, err := json.Marshal(event)
		if err != nil {
			h.logger.Error("failed to marshal event", "error", err)
			continue
		}

		fmt.Fprintf(w, "event: %s\n", event.Type)
		fmt.Fprintf(w, "data: %s\n\n", data)

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

func (h *Handler) ListRuns(w http.ResponseWriter, r *http.Request) {
	page := pagination.PageRequestFromQuery(r.URL.Query(), h.pagination)
	filters := RunFiltersFromQuery(r.URL.Query())

	result, err := h.sys.ListRuns(r.Context(), page, filters)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusInternalServerError, err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, result)
}

func (h *Handler) FindRun(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	run, err := h.sys.FindRun(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, run)
}

func (h *Handler) GetStages(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	stages, err := h.sys.GetStages(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, stages)
}

func (h *Handler) GetDecisions(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	decisions, err := h.sys.GetDecisions(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, decisions)
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	if err := h.sys.Cancel(r.Context(), id); err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Resume(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	run, err := h.sys.Resume(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, run)
}

// DeleteRun removes a workflow run and its related data.
func (h *Handler) DeleteRun(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	if err := h.sys.DeleteRun(r.Context(), id); err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
