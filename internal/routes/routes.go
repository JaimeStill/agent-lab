// Package routes provides HTTP route registration and handler building.
package routes

import (
	"log/slog"
	"net/http"
)

// System manages route registration and HTTP handler construction.
type System interface {
	RegisterGroup(group Group)
	RegisterRoute(route Route)
	Build() http.Handler
	Groups() []Group
	Routes() []Route
}

type routes struct {
	routes []Route
	groups []Group
	logger *slog.Logger
}

// New creates a route system with the specified logger.
func New(logger *slog.Logger) System {
	return &routes{
		logger: logger,
		groups: []Group{},
		routes: []Route{},
	}
}

func (r *routes) Groups() []Group {
	return r.groups
}

func (r *routes) Routes() []Route {
	return r.routes
}

// RegisterRoute adds a route to the route system.
func (r *routes) RegisterRoute(route Route) {
	r.routes = append(r.routes, route)
}

// RegisterGroup adds a route group to the route system.
func (r *routes) RegisterGroup(group Group) {
	r.groups = append(r.groups, group)
}

// Build constructs an http.Handler from all registered routes and groups.
func (r *routes) Build() http.Handler {
	mux := http.NewServeMux()

	for _, route := range r.routes {
		mux.HandleFunc(route.Method+" "+route.Pattern, route.Handler)
	}

	for _, group := range r.groups {
		r.registerGroup(mux, group)
	}

	return mux
}

func (r *routes) registerGroup(mux *http.ServeMux, group Group) {
	for _, route := range group.Routes {
		pattern := group.Prefix + route.Pattern
		mux.HandleFunc(route.Method+" "+pattern, route.Handler)
	}
	for _, child := range group.Children {
		r.registerGroup(mux, child)
	}
}
