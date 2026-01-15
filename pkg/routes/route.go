// Package routes provides types for HTTP route registration.
// These types are used by internal/routes to build the HTTP handler
// and by domain handlers to define their routes.
package routes

import (
	"net/http"

	"github.com/JaimeStill/agent-lab/pkg/openapi"
)

// Route defines an HTTP endpoint with its method, pattern, handler,
// and optional OpenAPI documentation.
type Route struct {
	Method  string
	Pattern string
	Handler http.HandlerFunc
	OpenAPI *openapi.Operation
}
