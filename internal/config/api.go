package config

import (
	"fmt"
	"os"

	"github.com/JaimeStill/agent-lab/pkg/middleware"
	"github.com/JaimeStill/agent-lab/pkg/openapi"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
)

var corsEnv = &middleware.CORSEnv{
	Enabled:          "API_CORS_ENABLED",
	Origins:          "API_CORS_ORIGINS",
	AllowedMethods:   "API_CORS_ALLOWED_METHODS",
	AllowedHeaders:   "API_CORS_ALLOWED_HEADERS",
	AllowCredentials: "API_CORS_ALLOW_CREDENTIALS",
	MaxAge:           "API_CORS_MAX_AGE",
}

var openAPIEnv = &openapi.ConfigEnv{
	Title:       "API_OPENAPI_TITLE",
	Description: "API_OPENAPI_DESCRIPTION",
}

var paginationEnv = &pagination.ConfigEnv{
	DefaultPageSize: "API_PAGINATION_DEFAULT_PAGE_SIZE",
	MaxPageSize:     "API_PAGINATION_MAX_PAGE_SIZE",
}

type APIConfig struct {
	BasePath   string                `toml:"base_path"`
	CORS       middleware.CORSConfig `toml:"cors"`
	Pagination pagination.Config     `toml:"pagination"`
	OpenAPI    openapi.Config        `toml:"openapi"`
}

func (c *APIConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()

	if err := c.CORS.Finalize(corsEnv); err != nil {
		return fmt.Errorf("cors: %w", err)
	}
	if err := c.Pagination.Finalize(paginationEnv); err != nil {
		return fmt.Errorf("pagination: %w", err)
	}
	if err := c.OpenAPI.Finalize(openAPIEnv); err != nil {
		return fmt.Errorf("openapi: %w", err)
	}
	return nil
}

func (c *APIConfig) Merge(overlay *APIConfig) {
	if overlay.BasePath != "" {
		c.BasePath = overlay.BasePath
	}
	c.CORS.Merge(&overlay.CORS)
	c.Pagination.Merge(&overlay.Pagination)
	c.OpenAPI.Merge(&overlay.OpenAPI)
}

func (c *APIConfig) loadDefaults() {
	if c.BasePath == "" {
		c.BasePath = "/api"
	}
}

func (c *APIConfig) loadEnv() {
	if v := os.Getenv("API_BASE_PATH"); v != "" {
		c.BasePath = v
	}
}
