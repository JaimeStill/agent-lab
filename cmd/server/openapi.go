package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/routes"
	"github.com/JaimeStill/agent-lab/pkg/openapi"
)

func specFilePath(env string) string {
	if env == "" {
		env = "local"
	}
	return filepath.Join("api", fmt.Sprintf("openapi.%s.json", env))
}

func loadOrGenerateSpec(
	cfg *config.Config,
	routeSys routes.System,
	components *openapi.Components,
) ([]byte, error) {
	path := specFilePath(cfg.Env())

	spec := generateSpec(routeSys, components, cfg)
	generated, err := openapi.MarshalJSON(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal spec: %w", err)
	}

	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, generated) {
		return generated, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create api directory: %w", err)
	}

	if err := os.WriteFile(path, generated, 0644); err != nil {
		return nil, fmt.Errorf("Write spec: %w", err)
	}

	return generated, nil
}

func generateSpec(
	rs routes.System,
	components *openapi.Components,
	cfg *config.Config,
) *openapi.Spec {
	spec := &openapi.Spec{
		OpenAPI: "3.1.0",
		Info: &openapi.Info{
			Title:       "Agent Lab API",
			Version:     cfg.Version,
			Description: "Containerized web service platform for building and orchestrating agentic workflows.",
		},
		Servers:    []*openapi.Server{{URL: cfg.Domain}},
		Components: components,
		Paths:      make(map[string]*openapi.PathItem),
	}

	for _, group := range rs.Groups() {
		processGroup(spec, group)
	}

	for _, route := range rs.Routes() {
		if route.OpenAPI == nil {
			continue
		}

		addOperation(spec, route.Pattern, route.Method, route.OpenAPI)
	}

	return spec
}

func processGroup(spec *openapi.Spec, group routes.Group) {
	for _, route := range group.Routes {
		if route.OpenAPI == nil {
			continue
		}

		path := group.Prefix + route.Pattern
		op := route.OpenAPI

		if len(op.Tags) == 0 {
			op.Tags = group.Tags
		}

		addOperation(spec, path, route.Method, op)
	}

	for _, child := range group.Children {
		processGroup(spec, child)
	}
}

func addOperation(spec *openapi.Spec, path, method string, op *openapi.Operation) {
	if spec.Paths[path] == nil {
		spec.Paths[path] = &openapi.PathItem{}
	}

	switch method {
	case "GET":
		spec.Paths[path].Get = op
	case "POST":
		spec.Paths[path].Post = op
	case "PUT":
		spec.Paths[path].Put = op
	case "DELETE":
		spec.Paths[path].Delete = op
	}
}
