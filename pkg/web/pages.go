// Package web provides infrastructure for serving web pages with Go templates.
// It supports pre-parsed templates for zero per-request overhead and
// declarative page definitions for simplified route generation.
package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
)

// PageDef defines a page with its route, template file, title, and bundle name.
type PageDef struct {
	Route    string
	Template string
	Title    string
	Bundle   string
}

// PageData contains the data passed to page templates during rendering.
// BasePath enables portable URL generation in templates via {{ .BasePath }}.
type PageData struct {
	Title    string
	Bundle   string
	BasePath string
	Data     any
}

// TemplateSet holds pre-parsed templates and a base path for URL generation.
// Templates are parsed once at startup, avoiding per-request overhead.
// The basePath is automatically included in PageData for all handlers.
type TemplateSet struct {
	pages    map[string]*template.Template
	basePath string
}

// NewTemplateSet creates a TemplateSet by parsing layout templates and cloning them
// for each page. The basePath is stored and automatically included in PageData
// for all handlers, enabling portable URL generation in templates.
// This pre-parsing at startup enables fail-fast behavior and eliminates
// per-request template parsing overhead.
func NewTemplateSet(layoutFS, pageFS embed.FS, layoutGlob, pageSubdir, basePath string, pages []PageDef) (*TemplateSet, error) {
	layouts, err := template.ParseFS(layoutFS, layoutGlob)
	if err != nil {
		return nil, err
	}

	pageSub, err := fs.Sub(pageFS, pageSubdir)
	if err != nil {
		return nil, err
	}

	pageTemplates := make(map[string]*template.Template, len(pages))
	for _, p := range pages {
		t, err := layouts.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone layouts for %s: %w", p.Template, err)
		}
		_, err = t.ParseFS(pageSub, p.Template)
		if err != nil {
			return nil, fmt.Errorf("parse template: %s: %w", p.Template, err)
		}
		pageTemplates[p.Template] = t
	}

	return &TemplateSet{
		pages:    pageTemplates,
		basePath: basePath,
	}, nil
}

// ErrorHandler returns an HTTP handler that renders an error template with the
// given status code.
func (ts *TemplateSet) ErrorHandler(layout, template string, status int, title string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		data := PageData{Title: title, BasePath: ts.basePath}
		if err := ts.Render(w, layout, template, data); err != nil {
			http.Error(w, http.StatusText(status), status)
		}
	}
}

// PageHandler returns an HTTP handler that renders the given page.
func (ts *TemplateSet) PageHandler(layout string, page PageDef) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:    page.Title,
			Bundle:   page.Bundle,
			BasePath: ts.basePath,
		}
		if err := ts.Render(w, layout, page.Template, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// Render executes the named layout template with the given page data.
// It sets the Content-Type header to text/html.
func (ts *TemplateSet) Render(w http.ResponseWriter, layoutName, pagePath string, data PageData) error {
	t, ok := ts.pages[pagePath]
	if !ok {
		return fmt.Errorf("template not found: %s", pagePath)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return t.ExecuteTemplate(w, layoutName, data)
}
