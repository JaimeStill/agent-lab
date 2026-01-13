// Package docs provides the interactive API documentation handler using Scalar UI.
// Assets are embedded at compile time for zero-dependency deployment.
package docs

import (
	_ "embed"
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/routes"
)

//go:embed index.html
var indexHTML []byte

// Handler serves the Scalar API documentation interface.
type Handler struct{}

// NewHandler creates a new documentation handler.
func NewHandler(spec []byte) *Handler {
	return &Handler{}
}

// Routes returns the route group for documentation endpoints.
func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix:      "/docs",
		Tags:        []string{"Documentation"},
		Description: "Interactive API documentation powered by Scalar",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.serveIndex},
		},
	}
}

func (h *Handler) serveIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(indexHTML)
}
