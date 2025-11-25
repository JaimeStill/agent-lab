package main

import (
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/lifecycle"
	"github.com/JaimeStill/agent-lab/internal/routes"
)

// registerRoutes configures all HTTP routes for the service.
func registerRoutes(r routes.System, ready lifecycle.ReadinessChecker) {
	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: "/healthz",
		Handler: handleHealthCheck,
	})

	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: "/readyz",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			handleReadinessCheck(w, ready)
		},
	})
}

// handleHealthCheck responds with OK status for health monitoring.
func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleReadinessCheck(w http.ResponseWriter, ready lifecycle.ReadinessChecker) {
	if !ready.Ready() {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("NOT READY"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("READY"))
}
