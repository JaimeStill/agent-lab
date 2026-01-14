// Package web provides embedded static assets and templates for the web interface.
// Assets are compiled by Vite and embedded at build time for zero-dependency deployment.
package web

import (
	"bytes"
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"time"

	"github.com/JaimeStill/agent-lab/internal/routes"
)

//go:embed dist/*
var distFS embed.FS

//go:embed public/*
var publicFS embed.FS

//go:embed all:templates
var templateFS embed.FS

// Static returns an HTTP handler that serves Vite-built assets from the embedded
// dist/ directory. The handler is designed to be mounted at /static/ and strips
// that prefix before serving files.
func Static() http.HandlerFunc {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("failed to created dist sub-filesystem: " + err.Error())
	}
	fileServer := http.StripPrefix("/static/", http.FileServer(http.FS(sub)))
	return func(w http.ResponseWriter, r *http.Request) {
		fileServer.ServeHTTP(w, r)
	}
}

// Handler serves the web client application pages using Go templates.
// It manages layout templates and renders pages by cloning the base layout
// and parsing page-specific templates to isolate template block definitions.
type Handler struct {
	layouts *template.Template
}

// NewHandler creates a new web client handler by parsing embedded templates.
// Returns an error if template parsing fails.
func NewHandler() (*Handler, error) {
	layouts, err := template.ParseFS(templateFS, "templates/layouts/*.html")
	if err != nil {
		return nil, err
	}
	return &Handler{layouts: layouts}, nil
}

// Routes returns the route group for web client endpoints.
// All routes are prefixed with /app.
func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix: "/app",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.serveHome},
			{Method: "GET", Pattern: "/components", Handler: h.serveComponents},
			{Method: "GET", Pattern: "/favicon.ico", Handler: publicFile("favicon.ico")},
			{Method: "GET", Pattern: "/favicon-16x16.png", Handler: publicFile("favicon-16x16.png")},
			{Method: "GET", Pattern: "/favicon-32x32.png", Handler: publicFile("favicon-32x32.png")},
			{Method: "GET", Pattern: "/apple-touch-icon.png", Handler: publicFile("apple-touch-icon.png")},
			{Method: "GET", Pattern: "/site.webmanifest", Handler: publicFile("site.webmanifest")},
		},
	}
}

func (h *Handler) serveHome(w http.ResponseWriter, r *http.Request) {
	tmpl, err := h.page("home/home.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.ExecuteTemplate(w, "app.html", map[string]string{
		"Title":  "Home",
		"Bundle": "shared",
	})
}

func (h *Handler) serveComponents(w http.ResponseWriter, r *http.Request) {
	tmpl, err := h.page("components/components.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.ExecuteTemplate(w, "app.html", map[string]string{
		"Title":  "Components",
		"Bundle": "shared",
	})
}

func (h *Handler) page(name string) (*template.Template, error) {
	t, err := h.layouts.Clone()
	if err != nil {
		return nil, err
	}
	return t.ParseFS(templateFS, "templates/pages/"+name)
}

func publicFile(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := publicFS.ReadFile("public/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
	}
}
