// Package docs provides the interactive API documentation handler using Scalar UI.
// Assets are embedded at compile time for zero-dependency deployment.
package scalar

import (
	"embed"
	"net/http"

	"github.com/JaimeStill/agent-lab/pkg/routes"
)

//go:embed index.html scalar.css scalar.js
var staticFS embed.FS

func Mount(r routes.System, prefix string) {
	router := newRouter()

	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: prefix,
		Handler: func(w http.ResponseWriter, req *http.Request) {
			req.URL.Path = "/"
			router.ServeHTTP(w, req)
		},
	})

	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: prefix + "/{path...}",
		Handler: http.StripPrefix(prefix, router).ServeHTTP,
	})
}

func newRouter() http.Handler {
	mux := http.NewServeMux()

	// Serve index.html at root
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data, _ := staticFS.ReadFile("index.html")
		w.Write(data)
	})

	// Serve static assets
	mux.Handle("GET /", http.FileServer(http.FS(staticFS)))

	return mux
}
