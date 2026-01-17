package main

import (
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/api"
	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/pkg/middleware"
	"github.com/JaimeStill/agent-lab/pkg/module"
	"github.com/JaimeStill/agent-lab/web/app"
	"github.com/JaimeStill/agent-lab/web/scalar"
)

type Modules struct {
	API    *module.Module
	App    *module.Module
	Scalar *module.Module
}

func NewModules(runtime *Runtime, cfg *config.Config) (*Modules, error) {
	apiModule, err := api.NewModule(
		cfg,
		runtime.Logger,
		runtime.Database,
		runtime.Storage,
		runtime.Lifecycle,
	)
	if err != nil {
		return nil, err
	}

	appModule, err := app.NewModule("/app")
	if err != nil {
		return nil, err
	}
	appModule.Use(middleware.AddSlash())
	appModule.Use(middleware.Logger(runtime.Logger))

	scalarModule := scalar.NewModule("/scalar")
	scalarModule.Use(middleware.AddSlash())

	return &Modules{
		API:    apiModule,
		App:    appModule,
		Scalar: scalarModule,
	}, nil
}

func (m *Modules) Mount(router *module.Router) {
	router.Mount(m.API)
	router.Mount(m.App)
	router.Mount(m.Scalar)
}

func buildRouter(runtime *Runtime) *module.Router {
	router := module.NewRouter()

	router.HandleNative("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	router.HandleNative("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if !runtime.Lifecycle.Ready() {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("NOT READY"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("READY"))
	})

	return router
}
