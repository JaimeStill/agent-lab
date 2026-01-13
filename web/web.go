// Package web provides embedded static assets and templates for the web interface.
// Assets are compiled by Vite and embedded at build time for zero-dependency deployment.
package web

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
)

//go:embed dist/*
var distFS embed.FS

//go:embed templates/*
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

// Templates parses and returns all HTML templates from the embedded templates/
// directory. Templates are organized in subdirectories (e.g., templates/layouts/)
// and can be used for server-rendered pages.
func Templates() (*template.Template, error) {
	return template.ParseFS(templateFS, "templates/**/*.html")
}
