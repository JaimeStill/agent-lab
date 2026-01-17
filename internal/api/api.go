package api

import (
	"log/slog"
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/pkg/database"
	"github.com/JaimeStill/agent-lab/pkg/lifecycle"
	"github.com/JaimeStill/agent-lab/pkg/middleware"
	"github.com/JaimeStill/agent-lab/pkg/module"
	"github.com/JaimeStill/agent-lab/pkg/openapi"
	"github.com/JaimeStill/agent-lab/pkg/storage"
)

func NewModule(
	cfg *config.Config,
	logger *slog.Logger,
	db database.System,
	store storage.System,
	lc *lifecycle.Coordinator,
) (*module.Module, error) {
	runtime := NewRuntime(cfg, logger, db, store, lc)
	domain := NewDomain(runtime)

	spec := openapi.NewSpec(cfg.API.OpenAPI.Title, cfg.Version)
	spec.SetDescription(cfg.API.OpenAPI.Description)
	spec.AddServer(cfg.Domain)

	mux := http.NewServeMux()
	registerRoutes(mux, spec, runtime, domain, cfg)

	specBytes, err := openapi.MarshalJSON(spec)
	if err != nil {
		return nil, err
	}
	mux.HandleFunc("GET /openapi.json", openapi.ServeSpec(specBytes))

	m := module.New(cfg.API.BasePath, mux)
	m.Use(middleware.TrimSlash())
	m.Use(middleware.CORS(&cfg.API.CORS))
	m.Use(middleware.Logger(runtime.Logger))

	return m, nil
}
