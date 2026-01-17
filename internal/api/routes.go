package api

import (
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/pkg/openapi"
	"github.com/JaimeStill/agent-lab/pkg/routes"
)

func registerRoutes(
	mux *http.ServeMux,
	spec *openapi.Spec,
	domain *Domain,
	cfg *config.Config,
) {
	routes.Register(
		mux,
		cfg.API.BasePath,
		spec,
		domain.Agents.Handler().Routes(),
		domain.Documents.Handler(cfg.Storage.MaxUploadSizeBytes()).Routes(),
		domain.Images.Handler().Routes(),
		domain.Profiles.Handler().Routes(),
		domain.Providers.Handler().Routes(),
		domain.Workflows.Handler().Routes(),
	)
}
