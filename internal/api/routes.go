package api

import (
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/agents"
	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/documents"
	"github.com/JaimeStill/agent-lab/internal/images"
	"github.com/JaimeStill/agent-lab/internal/profiles"
	"github.com/JaimeStill/agent-lab/internal/providers"
	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/agent-lab/pkg/openapi"
	"github.com/JaimeStill/agent-lab/pkg/routes"
)

func registerRoutes(
	mux *http.ServeMux,
	spec *openapi.Spec,
	runtime *Runtime,
	domain *Domain,
	cfg *config.Config,
) {
	providerHandler := providers.NewHandler(domain.Providers, runtime.Logger, runtime.Pagination)
	agentsHandler := agents.NewHandler(domain.Agents, runtime.Logger, runtime.Pagination)
	documentsHandler := documents.NewHandler(domain.Documents, runtime.Logger, runtime.Pagination, cfg.Storage.MaxUploadSizeBytes())
	imagesHandler := images.NewHandler(domain.Images, runtime.Logger, runtime.Pagination)
	profilesHandler := profiles.NewHandler(domain.Profiles, runtime.Logger, runtime.Pagination)
	workflowsHandler := workflows.NewHandler(domain.Workflows, runtime.Logger, runtime.Pagination)

	routes.Register(
		mux,
		cfg.API.BasePath,
		spec,
		providerHandler.Routes(),
		agentsHandler.Routes(),
		documentsHandler.Routes(),
		imagesHandler.Routes(),
		profilesHandler.Routes(),
		workflowsHandler.Routes(),
	)
}
