package main

import (
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/routes"
)

// registerRoutes configures all HTTP routes for the service.
func registerRoutes(r routes.System) {
	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: "/healthz",
		Handler: handleHealthCheck,
	})
}

// handleHealthCheck responds with OK status for health monitoring.
func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
