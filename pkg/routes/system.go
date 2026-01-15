package routes

import "net/http"

// System defines the interface for route registration and HTTP handler building.
// Implementations handle the actual registration and multiplexer construction.
type System interface {
	RegisterGroup(group Group)
	RegisterRoute(route Route)
	Build() http.Handler
	Groups() []Group
	Routes() []Route
}
