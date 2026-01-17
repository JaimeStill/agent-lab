package documents

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/JaimeStill/agent-lab/pkg/handlers"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/agent-lab/pkg/routes"
	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Handler provides HTTP endpoints for document operations.
type Handler struct {
	sys           System
	logger        *slog.Logger
	pagination    pagination.Config
	maxUploadSize int64
}

// NewHandler creates a document handler with the specified configuration.
func NewHandler(sys System, logger *slog.Logger, pagination pagination.Config, maxUploadSize int64) *Handler {
	return &Handler{
		sys:           sys,
		logger:        logger.With("handler", "documents"),
		pagination:    pagination,
		maxUploadSize: maxUploadSize,
	}
}

// Routes returns the document endpoint route group.
func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix:      "/documents",
		Tags:        []string{"Documents"},
		Description: "Document upload and management",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.List, OpenAPI: Spec.List},
			{Method: "GET", Pattern: "/{id}", Handler: h.Find, OpenAPI: Spec.Find},
			{Method: "POST", Pattern: "/search", Handler: h.Search, OpenAPI: Spec.Search},
			{Method: "POST", Pattern: "", Handler: h.Upload, OpenAPI: Spec.Upload},
			{Method: "PUT", Pattern: "/{id}", Handler: h.Update, OpenAPI: Spec.Update},
			{Method: "DELETE", Pattern: "/{id}", Handler: h.Delete, OpenAPI: Spec.Delete},
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

func (h *Handler) Find(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	doc, err := h.sys.Find(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, doc)
}

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

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(h.maxUploadSize); err != nil {
		handlers.RespondError(w, h.logger, http.StatusRequestEntityTooLarge, ErrFileTooLarge)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrInvalidFile)
		return
	}
	defer file.Close()

	if header.Size > h.maxUploadSize {
		handlers.RespondError(w, h.logger, http.StatusRequestEntityTooLarge, ErrFileTooLarge)
		return
	}

	data := make([]byte, header.Size)
	if _, err := file.Read(data); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrInvalidFile)
		return
	}

	contentType := detectContentType(header.Header.Get("Content-Type"), data)

	name := r.FormValue("name")
	if name == "" {
		name = header.Filename
	}

	var pageCount *int
	if contentType == "application/pdf" {
		pc, err := extractPDFPageCount(data)
		if err != nil {
			h.logger.Warn("failed to extract pdf page count", "error", err)
		} else {
			pageCount = pc
		}
	}

	cmd := CreateCommand{
		Name:        name,
		Filename:    header.Filename,
		ContentType: contentType,
		SizeBytes:   header.Size,
		PageCount:   pageCount,
		Data:        data,
	}

	doc, err := h.sys.Create(r.Context(), cmd)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusCreated, doc)
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

	doc, err := h.sys.Update(r.Context(), id, cmd)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, doc)
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

func detectContentType(header string, data []byte) string {
	if header != "" && header != "application/octet-stream" {
		return header
	}
	return http.DetectContentType(data)
}

func extractPDFPageCount(data []byte) (*int, error) {
	count, err := api.PageCount(bytes.NewReader(data), model.NewDefaultConfiguration())
	if err != nil {
		return nil, err
	}
	return &count, nil
}
