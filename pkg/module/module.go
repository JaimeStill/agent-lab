// Package module provides modular HTTP routing with middleware support.
// Modules are isolated handler groups that can be mounted at path prefixes,
// each with their own middleware chain.
package module

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/JaimeStill/agent-lab/pkg/middleware"
)

// Module represents an isolated HTTP handler group with a path prefix
// and middleware chain. Modules can be mounted onto a Router.
type Module struct {
	prefix     string
	router     http.Handler
	middleware middleware.System
}

// New creates a Module with the given path prefix and HTTP handler.
// Panics if the prefix is invalid (must start with "/" and be single-level).
func New(prefix string, router http.Handler) *Module {
	if err := validatePrefix(prefix); err != nil {
		panic(err)
	}
	return &Module{
		prefix:     prefix,
		router:     router,
		middleware: middleware.New(),
	}
}

// Handler returns the module's handler with all middleware applied.
func (m *Module) Handler() http.Handler {
	return m.middleware.Apply(m.router)
}

// Prefix returns the module's path prefix.
func (m *Module) Prefix() string {
	return m.prefix
}

// Serve handles HTTP requests by stripping the module prefix from the path
// before routing to the module's handler chain.
func (m *Module) Serve(w http.ResponseWriter, req *http.Request) {
	path := extractPath(req.URL.Path, m.prefix)
	request := cloneRequest(req, path)
	m.Handler().ServeHTTP(w, request)
}

// Use adds middleware to the module's chain.
func (m *Module) Use(mw func(http.Handler) http.Handler) {
	m.middleware.Use(mw)
}

func cloneRequest(req *http.Request, path string) *http.Request {
	request := new(http.Request)
	*request = *req
	request.URL = new(url.URL)
	*request.URL = *req.URL
	request.URL.Path = path
	request.URL.RawPath = ""
	return request
}

func extractPath(fullPath, prefix string) string {
	path := fullPath[len(prefix):]
	if path == "" {
		return "/"
	}
	return path
}

func validatePrefix(prefix string) error {
	if prefix == "" {
		return fmt.Errorf("module prefix cannot be empty")
	}
	if !strings.HasPrefix(prefix, "/") {
		return fmt.Errorf("module prefix must start with /: %s", prefix)
	}
	if strings.Count(prefix, "/") != 1 {
		return fmt.Errorf("module prefix must be single-level sub-path: %s", prefix)
	}
	return nil
}
