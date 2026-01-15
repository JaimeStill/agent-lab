package app

import (
	"embed"
	"net/http"

	"github.com/JaimeStill/agent-lab/pkg/routes"
	pkgweb "github.com/JaimeStill/agent-lab/pkg/web"
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

var pages = []pkgweb.PageDef{
	{Route: "/{$}", Template: "home.html", Title: "Home", Bundle: "app"},
	{Route: "/components", Template: "components.html", Title: "Components", Bundle: "app"},
}

var errorPages = []pkgweb.PageDef{
	{Template: "404.html", Title: "Not Found"},
}

type Handler struct {
	templates *pkgweb.TemplateSet
}

func NewHandler(basePath string) (*Handler, error) {
	allPages := append(pages, errorPages...)
	ts, err := pkgweb.NewTemplateSet(
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
	return &Handler{templates: ts}, nil
}

func (h *Handler) Mount(r routes.System, prefix string) {
	router := h.Router()

	// Exact match for prefix (e.g., /app)
	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: prefix,
		Handler: func(w http.ResponseWriter, req *http.Request) {
			req.URL.Path = "/"
			router.ServeHTTP(w, req)
		},
	})

	// Wildcard for all paths under prefix (e.g., /app/components)
	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: prefix + "/{path...}",
		Handler: http.StripPrefix(prefix, router).ServeHTTP,
	})
}

func (h *Handler) Router() http.Handler {
	r := pkgweb.NewRouter()
	r.SetFallback(h.templates.ErrorHandler(
		"app.html",
		"404.html",
		http.StatusNotFound,
		"Not Found",
	))

	for _, page := range pages {
		r.HandleFunc("GET "+page.Route, h.templates.PageHandler("app.html", page))
	}

	r.Handle("GET /dist/", http.FileServer(http.FS(distFS)))

	for _, route := range pkgweb.PublicFileRoutes(publicFS, "public", publicFiles...) {
		r.HandleFunc(route.Method+" "+route.Pattern, route.Handler)
	}

	return r
}
