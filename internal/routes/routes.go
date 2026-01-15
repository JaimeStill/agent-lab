// Package routes provides HTTP route registration and handler building.
package routes

import (
	"log/slog"
	"net/http"

	pkgroutes "github.com/JaimeStill/agent-lab/pkg/routes"
)

type routes struct {
	routes []pkgroutes.Route
	groups []pkgroutes.Group
	logger *slog.Logger
}

// New creates a route system with the specified logger.
func New(logger *slog.Logger) pkgroutes.System {
	return &routes{
		logger: logger,
		groups: []pkgroutes.Group{},
		routes: []pkgroutes.Route{},
	}
}

func (r *routes) Groups() []pkgroutes.Group {
	return r.groups
}

func (r *routes) Routes() []pkgroutes.Route {
	return r.routes
}

// RegisterRoute adds a route to the route system.
func (r *routes) RegisterRoute(route pkgroutes.Route) {
	r.routes = append(r.routes, route)
}

// RegisterGroup adds a route group to the route system.
func (r *routes) RegisterGroup(group pkgroutes.Group) {
	r.groups = append(r.groups, group)
}

// Build constructs an http.Handler from all registered routes and groups.
func (r *routes) Build() http.Handler {
	mux := http.NewServeMux()

	for _, route := range r.routes {
		mux.HandleFunc(route.Method+" "+route.Pattern, route.Handler)
	}

	for _, group := range r.groups {
		r.registerGroup(mux, "", group)
	}

	return mux
}

func (r *routes) registerGroup(mux *http.ServeMux, parentPrefix string, group pkgroutes.Group) {
	fullPrefix := parentPrefix + group.Prefix
	for _, route := range group.Routes {
		pattern := fullPrefix + route.Pattern
		mux.HandleFunc(route.Method+" "+pattern, route.Handler)
	}
	for _, child := range group.Children {
		r.registerGroup(mux, fullPrefix, child)
	}
}
