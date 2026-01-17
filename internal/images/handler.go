package images

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/JaimeStill/agent-lab/pkg/handlers"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/agent-lab/pkg/routes"
	"github.com/google/uuid"
)

// Handler provides HTTP endpoints for image management.
type Handler struct {
	sys        System
	logger     *slog.Logger
	pagination pagination.Config
}

// NewHandler creates a new images HTTP handler.
func NewHandler(sys System, logger *slog.Logger, pagination pagination.Config) *Handler {
	return &Handler{
		sys:        sys,
		logger:     logger.With("handler", "images"),
		pagination: pagination,
	}
}

// Routes returns the route configuration for image endpoints.
func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix:      "/images",
		Tags:        []string{"Images"},
		Description: "Document page image rendering and management",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.List, OpenAPI: Spec.List},
			{Method: "GET", Pattern: "/{id}", Handler: h.Find, OpenAPI: Spec.Find},
			{Method: "GET", Pattern: "/{id}/data", Handler: h.Data, OpenAPI: Spec.Data},
			{Method: "POST", Pattern: "/{documentId}/render", Handler: h.Render, OpenAPI: Spec.Render},
			{Method: "DELETE", Pattern: "/{id}", Handler: h.Delete, OpenAPI: Spec.Delete},
		},
	}
}

// List handles GET / - returns paginated images with optional filters.
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

// Find handles GET /{id} - returns image metadata.
func (h *Handler) Find(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	img, err := h.sys.Find(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, img)
}

// Data handles GET /{id}/data - returns raw image bytes.
func (h *Handler) Data(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	data, contentType, err := h.sys.Data(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// Render handles POST /{documentId}/render - renders document pages to images.
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

// Delete handles DELETE /{id} - deletes an image.
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
