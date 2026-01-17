package module

import (
	"net/http"
	"strings"
)

// Router routes requests to mounted modules or native handlers.
type Router struct {
	modules map[string]*Module
	native  *http.ServeMux
}

// NewRouter creates a Router for mounting modules and native handlers.
func NewRouter() *Router {
	return &Router{
		modules: make(map[string]*Module),
		native:  http.NewServeMux(),
	}
}

// HandleNative registers a handler directly with the native ServeMux,
// bypassing module routing. Used for handlers like health checks.
func (r *Router) HandleNative(pattern string, handler http.HandlerFunc) {
	r.native.HandleFunc(pattern, handler)
}

// Mount registers a module at its configured prefix.
func (r *Router) Mount(m *Module) {
	r.modules[m.prefix] = m
}

// ServeHTTP routes requests to the matching module or falls back to native handlers.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	prefix := extractPrefix(req.URL.Path)

	if m, ok := r.modules[prefix]; ok {
		m.Serve(w, req)
		return
	}

	r.native.ServeHTTP(w, req)
}

func extractPrefix(path string) string {
	parts := strings.SplitN(path, "/", 3)
	if len(parts) >= 2 {
		return "/" + parts[1]
	}
	return path
}
