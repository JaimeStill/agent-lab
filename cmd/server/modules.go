package main

import (
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/api"
	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/pkg/middleware"
	"github.com/JaimeStill/agent-lab/pkg/module"
	"github.com/JaimeStill/agent-lab/pkg/runtime"
	"github.com/JaimeStill/agent-lab/web/app"
	"github.com/JaimeStill/agent-lab/web/scalar"
)

// Modules holds all application modules that are mounted to the router.
type Modules struct {
	API    *module.Module
	App    *module.Module
	Scalar *module.Module
}

// NewModules creates and configures all application modules.
func NewModules(infra *runtime.Infrastructure, cfg *config.Config) (*Modules, error) {
	apiModule, err := api.NewModule(cfg, infra)
	if err != nil {
		return nil, err
	}

	appModule, err := app.NewModule("/app")
	if err != nil {
		return nil, err
	}
	appModule.Use(middleware.Logger(infra.Logger))

	scalarModule := scalar.NewModule("/scalar")

	return &Modules{
		API:    apiModule,
		App:    appModule,
		Scalar: scalarModule,
	}, nil
}

// Mount registers all modules with the router.
func (m *Modules) Mount(router *module.Router) {
	router.Mount(m.API)
	router.Mount(m.App)
	router.Mount(m.Scalar)
}

func buildRouter(infra *runtime.Infrastructure) *module.Router {
	router := module.NewRouter()

	router.HandleNative("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	router.HandleNative("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if !infra.Lifecycle.Ready() {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("NOT READY"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("READY"))
	})

	return router
}
