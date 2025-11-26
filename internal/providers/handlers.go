package providers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

// HandleCreate handles POST /api/providers requests.
func HandleCreate(w http.ResponseWriter, r *http.Request, sys System, logger *slog.Logger) {
	var cmd CreateCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		respondError(w, logger, http.StatusBadRequest, err)
		return
	}

	result, err := sys.Create(r.Context(), cmd)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrInvalidConfig) {
			status = http.StatusBadRequest
		} else if errors.Is(err, ErrDuplicate) {
			status = http.StatusConflict
		}
		respondError(w, logger, status, err)
		return
	}

	respondJSON(w, http.StatusCreated, result)
}

// HandleUpdate handles PUT /api/providers/{id} requests.
func HandleUpdate(w http.ResponseWriter, r *http.Request, sys System, logger *slog.Logger) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, logger, http.StatusBadRequest, err)
		return
	}

	var cmd UpdateCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		respondError(w, logger, http.StatusBadRequest, err)
		return
	}

	result, err := sys.Update(r.Context(), id, cmd)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrInvalidConfig) {
			status = http.StatusBadRequest
		} else if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, ErrDuplicate) {
			status = http.StatusConflict
		}
		respondError(w, logger, status, err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// HandleDelete handles DELETE /api/providers/{id} requests.
func HandleDelete(w http.ResponseWriter, r *http.Request, sys System, logger *slog.Logger) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, logger, http.StatusBadRequest, err)
		return
	}

	if err := sys.Delete(r.Context(), id); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, logger, status, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleGetByID handles GET /api/providers/{id} requests.
func HandleGetByID(w http.ResponseWriter, r *http.Request, sys System, logger *slog.Logger) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, logger, http.StatusBadRequest, err)
		return
	}

	result, err := sys.FindByID(r.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, logger, status, err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// HandleSearch handles POST /api/providers/search requests.
func HandleSearch(w http.ResponseWriter, r *http.Request, sys System, logger *slog.Logger) {
	var page pagination.PageRequest
	if err := json.NewDecoder(r.Body).Decode(&page); err != nil {
		respondError(w, logger, http.StatusBadRequest, err)
		return
	}

	result, err := sys.Search(r.Context(), page)
	if err != nil {
		respondError(w, logger, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, logger *slog.Logger, status int, err error) {
	logger.Error("handler error", "error", err, "status", status)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
