// Package app provides the web application module with embedded templates and assets.
package app

import (
	"embed"
	"net/http"

	"github.com/JaimeStill/agent-lab/pkg/module"
	"github.com/JaimeStill/agent-lab/pkg/web"
)

//go:embed dist/*
var distFS embed.FS

//go:embed public/*
var publicFS embed.FS

//go:embed server/layouts/*
var layoutFS embed.FS

//go:embed server/pages/*
var pageFS embed.FS

var publicFiles = []string{
	"favicon.ico",
	"favicon-16x16.png",
	"favicon-32x32.png",
	"apple-touch-icon.png",
	"site.webmanifest",
}

var pages = []web.PageDef{
	{Route: "/{$}", Template: "home.html", Title: "Home", Bundle: "app"},
	{Route: "/components/", Template: "components.html", Title: "Components", Bundle: "app"},
}

var errorPages = []web.PageDef{
	{Template: "404.html", Title: "Not Found"},
}

// NewModule creates the app module configured for the given base path.
func NewModule(basePath string) (*module.Module, error) {
	allPages := append(pages, errorPages...)
	ts, err := web.NewTemplateSet(
		layoutFS,
		pageFS,
		"server/layouts/*.html",
		"server/pages",
		basePath,
		allPages,
	)
	if err != nil {
		return nil, err
	}

	router := buildRouter(ts)
	return module.New(basePath, router), nil
}

func buildRouter(ts *web.TemplateSet) http.Handler {
	r := web.NewRouter()
	r.SetFallback(ts.ErrorHandler(
		"app.html",
		"404.html",
		http.StatusNotFound,
		"Not Found",
	))

	for _, page := range pages {
		r.HandleFunc("GET "+page.Route, ts.PageHandler("app.html", page))
	}

	r.Handle("GET /dist/", http.FileServer(http.FS(distFS)))

	for _, route := range web.PublicFileRoutes(publicFS, "public", publicFiles...) {
		r.HandleFunc(route.Method+" "+route.Pattern, route.Handler)
	}

	return r
}
