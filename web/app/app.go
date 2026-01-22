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

//go:embed server/views/*
var viewFS embed.FS

var publicFiles = []string{
	"favicon.ico",
	"favicon-16x16.png",
	"favicon-32x32.png",
	"apple-touch-icon.png",
	"site.webmanifest",
}

var views = []web.ViewDef{
	{Route: "/{$}", Template: "home.html", Title: "Home", Bundle: "app"},
	{Route: "/components", Template: "components.html", Title: "Components", Bundle: "app"},
}

var errorViews = []web.ViewDef{
	{Template: "404.html", Title: "Not Found", Bundle: "app"},
}

// NewModule creates the app module configured for the given base path.
func NewModule(basePath string) (*module.Module, error) {
	allViews := append(views, errorViews...)
	ts, err := web.NewTemplateSet(
		layoutFS,
		viewFS,
		"server/layouts/*.html",
		"server/views",
		basePath,
		allViews,
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
		errorViews[0],
		http.StatusNotFound,
	))

	for _, view := range views {
		r.HandleFunc("GET "+view.Route, ts.PageHandler("app.html", view))
	}

	r.Handle("GET /dist/", http.FileServer(http.FS(distFS)))

	for _, route := range web.PublicFileRoutes(publicFS, "public", publicFiles...) {
		r.HandleFunc(route.Method+" "+route.Pattern, route.Handler)
	}

	return r
}
