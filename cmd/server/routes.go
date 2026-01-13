package main

import (
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/agents"
	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/documents"
	"github.com/JaimeStill/agent-lab/internal/images"
	"github.com/JaimeStill/agent-lab/internal/lifecycle"
	"github.com/JaimeStill/agent-lab/internal/profiles"
	"github.com/JaimeStill/agent-lab/internal/providers"
	"github.com/JaimeStill/agent-lab/internal/routes"
	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/agent-lab/pkg/openapi"
	"github.com/JaimeStill/agent-lab/web"
	"github.com/JaimeStill/agent-lab/web/docs"
)

// registerRoutes configures all HTTP routes for the service.
func registerRoutes(r routes.System, runtime *Runtime, domain *Domain, cfg *config.Config) error {
	providerHandler := providers.NewHandler(
		domain.Providers,
		runtime.Logger,
		runtime.Pagination,
	)
	r.RegisterGroup(providerHandler.Routes())

	agentHandler := agents.NewHandler(
		domain.Agents,
		runtime.Logger,
		runtime.Pagination,
	)
	r.RegisterGroup(agentHandler.Routes())

	documentHandler := documents.NewHandler(
		domain.Documents,
		runtime.Logger,
		runtime.Pagination,
		cfg.Storage.MaxUploadSizeBytes(),
	)
	r.RegisterGroup(documentHandler.Routes())

	imagesHandler := images.NewHandler(
		domain.Images,
		runtime.Logger,
		runtime.Pagination,
	)
	r.RegisterGroup(imagesHandler.Routes())

	profilesHandler := profiles.NewHandler(
		domain.Profiles,
		runtime.Logger,
		runtime.Pagination,
	)
	r.RegisterGroup(profilesHandler.Routes())

	workflowHandler := workflows.NewHandler(
		domain.Workflows,
		runtime.Logger,
		runtime.Pagination,
	)
	r.RegisterGroup(workflowHandler.Routes())

	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: "/healthz",
		Handler: handleHealthCheck,
		OpenAPI: &openapi.Operation{
			Summary: "Health check endpoint",
			Tags:    []string{"Infrastructure"},
			Responses: map[int]*openapi.Response{
				200: {Description: "Service is healthy"},
			},
		},
	})

	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: "/readyz",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			handleReadinessCheck(w, runtime.Lifecycle)
		},
		OpenAPI: &openapi.Operation{
			Summary: "Readiness check endpoint",
			Tags:    []string{"Infrastructure"},
			Responses: map[int]*openapi.Response{
				200: {Description: "Service is ready"},
				503: {Description: "Service not ready"},
			},
		},
	})

	components := openapi.NewComponents()
	components.AddSchemas(agents.Spec.Schemas())
	components.AddSchemas(providers.Spec.Schemas())
	components.AddSchemas(documents.Spec.Schemas())
	components.AddSchemas(images.Spec.Schemas())
	components.AddSchemas(profiles.Spec.Schemas())
	components.AddSchemas(workflows.Spec.Schemas())

	specBytes, err := loadOrGenerateSpec(cfg, r, components)
	if err != nil {
		return err
	}

	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: "/api/openapi.json",
		Handler: serveOpenAPISpec(specBytes),
	})

	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: "/static/",
		Handler: web.Static(),
	})

	docsHandler := docs.NewHandler(specBytes)
	r.RegisterGroup(docsHandler.Routes())

	return nil
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

func serveOpenAPISpec(spec []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(spec)
	}
}
