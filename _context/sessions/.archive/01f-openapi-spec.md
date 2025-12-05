# Session 01f: OpenAPI Specification & Scalar UI Integration

**Session ID:** 01f
**Status:** Ready for Implementation
**Duration Estimate:** 3-4 hours

## Overview

Implement interactive API documentation using OpenAPI 3.1 specification generation and self-hosted Scalar UI. The implementation uses a hybrid approach: component schemas are manually defined, route metadata extends existing route definitions, and spec generation occurs during server startup.

## Architecture Approach

**OpenAPI Generation (Integrated):**
- Domain-owned schemas (e.g., `internal/providers/schema.go`, `internal/agents/schema.go`)
- Shared infrastructure in `pkg/openapi` (types, helpers, shared responses)
- Route definitions extended with optional `OpenAPI` field
- Spec generated during `cmd/server` startup (no separate CLI tool)
- Environment-specific output: `api/openapi.{env}.json`
- Change detection: generate in memory, compare with file, write only if different
- Docs handler serves spec from memory (no per-request file I/O)

**Scalar Integration:**
- Self-hosted (no CDN dependencies)
- Assets embedded via `go:embed`
- Served at `/docs` endpoint
- Air-gap compatible from day 1

## Implementation Steps

### Phase 1: OpenAPI Infrastructure

#### Step 1.1: Create pkg/openapi/types.go

Define OpenAPI 3.1 type structures:

```go
package openapi

type Spec struct {
	OpenAPI    string                `json:"openapi"`
	Info       *Info                 `json:"info"`
	Servers    []*Server             `json:"servers,omitempty"`
	Paths      map[string]*PathItem  `json:"paths"`
	Components *Components           `json:"components,omitempty"`
}

type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
}

type Operation struct {
	Summary     string              `json:"summary,omitempty"`
	Description string              `json:"description,omitempty"`
	Tags        []string            `json:"tags,omitempty"`
	Parameters  []*Parameter        `json:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[int]*Response   `json:"responses"`
}

type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"`
	Required    bool    `json:"required,omitempty"`
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema"`
}

type RequestBody struct {
	Description string                `json:"description,omitempty"`
	Required    bool                  `json:"required,omitempty"`
	Content     map[string]*MediaType `json:"content"`
}

type Response struct {
	Description string                `json:"description"`
	Content     map[string]*MediaType `json:"content,omitempty"`
	Ref         string                `json:"$ref,omitempty"`
}

type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}

type Schema struct {
	Type        string                `json:"type,omitempty"`
	Format      string                `json:"format,omitempty"`
	Description string                `json:"description,omitempty"`
	Properties  map[string]*Property  `json:"properties,omitempty"`
	Required    []string              `json:"required,omitempty"`
	Items       *Schema               `json:"items,omitempty"`
	Ref         string                `json:"$ref,omitempty"`
}

type Property struct {
	Type        string `json:"type"`
	Format      string `json:"format,omitempty"`
	Description string `json:"description,omitempty"`
	Example     any    `json:"example,omitempty"`
}

type Components struct {
	Schemas   map[string]*Schema   `json:"schemas,omitempty"`
	Responses map[string]*Response `json:"responses,omitempty"`
}

func SchemaRef(name string) *Schema {
	return &Schema{Ref: "#/components/schemas/" + name}
}

func ResponseRef(name string) *Response {
	return &Response{Ref: "#/components/responses/" + name}
}

func RequestBodyJSON(schemaName string, required bool) *RequestBody {
	return &RequestBody{
		Required: required,
		Content: map[string]*MediaType{
			"application/json": {Schema: SchemaRef(schemaName)},
		},
	}
}

func ResponseJSON(description string, schemaName string) *Response {
	return &Response{
		Description: description,
		Content: map[string]*MediaType{
			"application/json": {Schema: SchemaRef(schemaName)},
		},
	}
}

func PathParam(name, description string) *Parameter {
	return &Parameter{
		Name:        name,
		In:          "path",
		Required:    true,
		Description: description,
		Schema:      &Schema{Type: "string", Format: "uuid"},
	}
}

func QueryParam(name, typ, description string, required bool) *Parameter {
	return &Parameter{
		Name:        name,
		In:          "query",
		Required:    required,
		Description: description,
		Schema:      &Schema{Type: typ},
	}
}
```

#### Step 1.2: Extend Configuration for OpenAPI Metadata

Add application version and domain to root config, plus Env() method.

**Update `internal/config/types.go`:**

Add environment constants:

```go
const (
	// ... existing constants ...
	EnvServiceVersion = "SERVICE_VERSION"
	EnvServiceDomain  = "SERVICE_DOMAIN"
	EnvServiceEnv     = "SERVICE_ENV"
)
```

**Update `internal/config/config.go`:**

```go
type Config struct {
	Version         string            `toml:"version"`
	Domain          string            `toml:"domain"`
	ShutdownTimeout string            `toml:"shutdown_timeout"`
	Server          ServerConfig      `toml:"server"`
	Database        DatabaseConfig    `toml:"database"`
	Logging         LoggingConfig     `toml:"logging"`
	CORS            CORSConfig        `toml:"cors"`
	Pagination      pagination.Config `toml:"pagination"`
}

func (c *Config) Env() string {
	if env := os.Getenv(EnvServiceEnv); env != "" {
		return env
	}
	return "local"
}

func (c *Config) loadDefaults() {
	if c.Version == "" {
		c.Version = "0.1.0"
	}
	c.Server.loadDefaults()
	if c.Domain == "" {
		host := c.Server.Host
		if host == "0.0.0.0" {
			host = "localhost"
		}
		c.Domain = fmt.Sprintf("http://%s:%d", host, c.Server.Port)
	}
	// ... remaining defaults ...
}

func (c *Config) loadEnv() {
	if v := os.Getenv(EnvServiceVersion); v != "" {
		c.Version = v
	}
	if v := os.Getenv(EnvServiceDomain); v != "" {
		c.Domain = v
	}
	// ... existing env loading ...
}
```

**Update `config.toml`:**

```toml
version = "0.1.0"
domain = "http://localhost:8080"
shutdown_timeout = "30s"

[server]
host = "0.0.0.0"
port = 8080
# ... existing fields ...
```

**Example overlay `config.prod.toml`:**

```toml
version = "1.0.0"
domain = "https://api.example.com"
```

**Update `.env`:**

```bash
# Service Configuration
SERVICE_ENV=local
SERVICE_VERSION=0.1.0
SERVICE_DOMAIN=http://localhost:8080
SERVICE_SHUTDOWN_TIMEOUT=30s
```

#### Step 1.3: Create pkg/openapi/components.go

Define Components constructor and shared responses (infrastructure only, no domain schemas):

```go
package openapi

import "maps"

func NewComponents() *Components {
	return &Components{
		Schemas: map[string]*Schema{
			"PageRequest": {
				Type: "object",
				Properties: map[string]*Property{
					"page":      {Type: "integer", Description: "Page number (1-indexed)", Example: 1},
					"page_size": {Type: "integer", Description: "Results per page", Example: 20},
					"search":    {Type: "string", Description: "Search query"},
					"sort":      {Type: "string", Description: "Comma-separated sort fields. Prefix with - for descending. Example: name,-created_at"},
				},
			},
		},
		Responses: map[string]*Response{
			"BadRequest": {
				Description: "Invalid request",
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
							Properties: map[string]*Property{
								"error": {Type: "string", Description: "Error message"},
							},
						},
					},
				},
			},
			"NotFound": {
				Description: "Resource not found",
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
							Properties: map[string]*Property{
								"error": {Type: "string", Description: "Error message"},
							},
						},
					},
				},
			},
			"Conflict": {
				Description: "Resource conflict (duplicate name)",
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
							Properties: map[string]*Property{
								"error": {Type: "string", Description: "Error message"},
							},
						},
					},
				},
			},
		},
	}
}

func (c *Components) AddSchemas(schemas map[string]*Schema) {
	maps.Copy(c.Schemas, schemas)
}

func (c *Components) AddResponses(responses map[string]*Response) {
	maps.Copy(c.Responses, responses)
}
```

#### Step 1.4: Create internal/providers/openapi.go

Domain-owned provider schemas and operation definitions:

```go
package providers

import "github.com/JaimeStill/agent-lab/pkg/openapi"

type spec struct {
	Create *openapi.Operation
	List   *openapi.Operation
	Get    *openapi.Operation
	Update *openapi.Operation
	Delete *openapi.Operation
	Search *openapi.Operation
}

var Spec = spec{
	Create: &openapi.Operation{
		Summary:     "Create provider",
		Description: "Validates and stores a new provider configuration",
		RequestBody: openapi.RequestBodyJSON("CreateProviderCommand", true),
		Responses: map[int]*openapi.Response{
			201: openapi.ResponseJSON("Provider created", "Provider"),
			400: openapi.ResponseRef("BadRequest"),
			409: openapi.ResponseRef("Conflict"),
		},
	},
	List: &openapi.Operation{
		Summary:     "List providers",
		Description: "Returns a paginated list of providers with optional filtering and sorting",
		Parameters: []*openapi.Parameter{
			openapi.QueryParam("page", "integer", "Page number (1-indexed)", false),
			openapi.QueryParam("page_size", "integer", "Results per page", false),
			openapi.QueryParam("search", "string", "Search query (matches name)", false),
			openapi.QueryParam("sort", "string", "Comma-separated sort fields. Prefix with - for descending", false),
			openapi.QueryParam("name", "string", "Filter by provider name (contains)", false),
		},
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Paginated list of providers", "ProviderPageResult"),
		},
	},
	Get: &openapi.Operation{
		Summary:     "Get provider by ID",
		Description: "Retrieves a single provider configuration",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Provider UUID"),
		},
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Provider configuration", "Provider"),
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
	Update: &openapi.Operation{
		Summary:     "Update provider",
		Description: "Updates an existing provider configuration",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Provider UUID"),
		},
		RequestBody: openapi.RequestBodyJSON("UpdateProviderCommand", true),
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Provider updated", "Provider"),
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
			409: openapi.ResponseRef("Conflict"),
		},
	},
	Delete: &openapi.Operation{
		Summary:     "Delete provider",
		Description: "Removes a provider configuration",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Provider UUID"),
		},
		Responses: map[int]*openapi.Response{
			204: {Description: "Provider deleted"},
			404: openapi.ResponseRef("NotFound"),
		},
	},
	Search: &openapi.Operation{
		Summary:     "Search providers",
		Description: "Search providers with filters and pagination via POST body",
		Parameters: []*openapi.Parameter{
			openapi.QueryParam("name", "string", "Filter by provider name (contains)", false),
		},
		RequestBody: openapi.RequestBodyJSON("PageRequest", false),
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Paginated search results", "ProviderPageResult"),
			400: openapi.ResponseRef("BadRequest"),
		},
	},
}

func (spec) Schemas() map[string]*openapi.Schema {
	return map[string]*openapi.Schema{
		"Provider": {
			Type: "object",
			Properties: map[string]*openapi.Property{
				"id":         {Type: "string", Format: "uuid"},
				"name":       {Type: "string"},
				"config":     {Type: "object", Description: "go-agents ProviderConfig as JSON"},
				"created_at": {Type: "string", Format: "date-time"},
				"updated_at": {Type: "string", Format: "date-time"},
			},
		},
		"CreateProviderCommand": {
			Type:     "object",
			Required: []string{"name", "config"},
			Properties: map[string]*openapi.Property{
				"name":   {Type: "string", Example: "ollama"},
				"config": {Type: "object", Description: "go-agents ProviderConfig as JSON"},
			},
		},
		"UpdateProviderCommand": {
			Type:     "object",
			Required: []string{"name", "config"},
			Properties: map[string]*openapi.Property{
				"name":   {Type: "string"},
				"config": {Type: "object", Description: "go-agents ProviderConfig as JSON"},
			},
		},
		"ProviderPageResult": {
			Type: "object",
			Properties: map[string]*openapi.Property{
				"data":        {Type: "array", Description: "Array of providers"},
				"total":       {Type: "integer", Description: "Total number of results"},
				"page":        {Type: "integer", Description: "Current page number"},
				"page_size":   {Type: "integer", Description: "Results per page"},
				"total_pages": {Type: "integer", Description: "Total number of pages"},
			},
		},
	}
}
```

#### Step 1.5: Create internal/agents/openapi.go

Domain-owned agent schemas and operation definitions:

```go
package agents

import "github.com/JaimeStill/agent-lab/pkg/openapi"

type spec struct {
	Create       *openapi.Operation
	List         *openapi.Operation
	Get          *openapi.Operation
	Update       *openapi.Operation
	Delete       *openapi.Operation
	Search       *openapi.Operation
	Chat         *openapi.Operation
	ChatStream   *openapi.Operation
	Vision       *openapi.Operation
	VisionStream *openapi.Operation
	Tools        *openapi.Operation
	Embed        *openapi.Operation
}

var Spec = spec{
	Create: &openapi.Operation{
		Summary:     "Create agent",
		Description: "Validates and stores a new agent configuration",
		RequestBody: openapi.RequestBodyJSON("CreateAgentCommand", true),
		Responses: map[int]*openapi.Response{
			201: openapi.ResponseJSON("Agent created", "Agent"),
			400: openapi.ResponseRef("BadRequest"),
			409: openapi.ResponseRef("Conflict"),
		},
	},
	List: &openapi.Operation{
		Summary:     "List agents",
		Description: "Returns a paginated list of agents with optional filtering and sorting",
		Parameters: []*openapi.Parameter{
			openapi.QueryParam("page", "integer", "Page number (1-indexed)", false),
			openapi.QueryParam("page_size", "integer", "Results per page", false),
			openapi.QueryParam("search", "string", "Search query (matches name)", false),
			openapi.QueryParam("sort", "string", "Comma-separated sort fields. Prefix with - for descending", false),
			openapi.QueryParam("name", "string", "Filter by agent name (contains)", false),
		},
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Paginated list of agents", "AgentPageResult"),
		},
	},
	Get: &openapi.Operation{
		Summary:     "Get agent by ID",
		Description: "Retrieves a single agent configuration",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Agent UUID"),
		},
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Agent configuration", "Agent"),
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
	Update: &openapi.Operation{
		Summary:     "Update agent",
		Description: "Updates an existing agent configuration",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Agent UUID"),
		},
		RequestBody: openapi.RequestBodyJSON("UpdateAgentCommand", true),
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Agent updated", "Agent"),
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
			409: openapi.ResponseRef("Conflict"),
		},
	},
	Delete: &openapi.Operation{
		Summary:     "Delete agent",
		Description: "Removes an agent configuration",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Agent UUID"),
		},
		Responses: map[int]*openapi.Response{
			204: {Description: "Agent deleted"},
			404: openapi.ResponseRef("NotFound"),
		},
	},
	Search: &openapi.Operation{
		Summary:     "Search agents",
		Description: "Search agents with filters and pagination via POST body",
		Parameters: []*openapi.Parameter{
			openapi.QueryParam("name", "string", "Filter by agent name (contains)", false),
		},
		RequestBody: openapi.RequestBodyJSON("PageRequest", false),
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Paginated search results", "AgentPageResult"),
			400: openapi.ResponseRef("BadRequest"),
		},
	},
	Chat: &openapi.Operation{
		Summary:     "Chat with agent",
		Description: "Execute agent chat completion (synchronous)",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Agent UUID"),
		},
		RequestBody: openapi.RequestBodyJSON("ChatRequest", true),
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Chat response", "ChatResponse"),
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
	ChatStream: &openapi.Operation{
		Summary:     "Chat with agent (streaming)",
		Description: "Execute agent chat completion with SSE streaming",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Agent UUID"),
		},
		RequestBody: openapi.RequestBodyJSON("ChatRequest", true),
		Responses: map[int]*openapi.Response{
			200: {Description: "SSE stream of chat response chunks"},
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
	Vision: &openapi.Operation{
		Summary:     "Vision analysis",
		Description: "Execute agent vision analysis (synchronous, multipart/form-data)",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Agent UUID"),
		},
		RequestBody: &openapi.RequestBody{
			Required: true,
			Content: map[string]*openapi.MediaType{
				"multipart/form-data": {
					Schema: &openapi.Schema{
						Type: "object",
						Properties: map[string]*openapi.Property{
							"prompt": {Type: "string", Description: "Analysis prompt"},
							"images": {Type: "string", Description: "Image file (multiple supported via repeated field)"},
							"token":  {Type: "string", Description: "Optional authentication token"},
						},
					},
				},
			},
		},
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Vision analysis response", "ChatResponse"),
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
	VisionStream: &openapi.Operation{
		Summary:     "Vision analysis (streaming)",
		Description: "Execute agent vision analysis with SSE streaming",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Agent UUID"),
		},
		RequestBody: &openapi.RequestBody{
			Required: true,
			Content: map[string]*openapi.MediaType{
				"multipart/form-data": {
					Schema: &openapi.Schema{
						Type: "object",
						Properties: map[string]*openapi.Property{
							"prompt": {Type: "string", Description: "Analysis prompt"},
							"images": {Type: "string", Description: "Image file (multiple supported via repeated field)"},
							"token":  {Type: "string", Description: "Optional authentication token"},
						},
					},
				},
			},
		},
		Responses: map[int]*openapi.Response{
			200: {Description: "SSE stream of vision analysis chunks"},
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
	Tools: &openapi.Operation{
		Summary:     "Execute with tools",
		Description: "Execute agent with tool calling capabilities",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Agent UUID"),
		},
		RequestBody: openapi.RequestBodyJSON("ToolsRequest", true),
		Responses: map[int]*openapi.Response{
			200: {Description: "Tool execution response"},
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
	Embed: &openapi.Operation{
		Summary:     "Generate embeddings",
		Description: "Generate text embeddings using agent",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Agent UUID"),
		},
		RequestBody: openapi.RequestBodyJSON("EmbedRequest", true),
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Embedding vector", "EmbedResponse"),
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
}

func (spec) Schemas() map[string]*openapi.Schema {
	return map[string]*openapi.Schema{
		"Agent": {
			Type: "object",
			Properties: map[string]*openapi.Property{
				"id":         {Type: "string", Format: "uuid"},
				"name":       {Type: "string"},
				"config":     {Type: "object", Description: "go-agents AgentConfig as JSON (includes embedded provider config)"},
				"created_at": {Type: "string", Format: "date-time"},
				"updated_at": {Type: "string", Format: "date-time"},
			},
		},
		"CreateAgentCommand": {
			Type:     "object",
			Required: []string{"name", "config"},
			Properties: map[string]*openapi.Property{
				"name":   {Type: "string", Example: "gpt-4o"},
				"config": {Type: "object", Description: "go-agents AgentConfig as JSON"},
			},
		},
		"UpdateAgentCommand": {
			Type:     "object",
			Required: []string{"name", "config"},
			Properties: map[string]*openapi.Property{
				"name":   {Type: "string"},
				"config": {Type: "object", Description: "go-agents AgentConfig as JSON"},
			},
		},
		"AgentPageResult": {
			Type: "object",
			Properties: map[string]*openapi.Property{
				"data":        {Type: "array", Description: "Array of agents"},
				"total":       {Type: "integer", Description: "Total number of results"},
				"page":        {Type: "integer", Description: "Current page number"},
				"page_size":   {Type: "integer", Description: "Results per page"},
				"total_pages": {Type: "integer", Description: "Total number of pages"},
			},
		},
		"ChatRequest": {
			Type:     "object",
			Required: []string{"prompt"},
			Properties: map[string]*openapi.Property{
				"prompt":  {Type: "string", Description: "User prompt"},
				"token":   {Type: "string", Description: "Optional authentication token (for Azure providers)"},
				"options": {Type: "object", Description: "Optional agent options override"},
			},
		},
		"ChatResponse": {
			Type: "object",
			Properties: map[string]*openapi.Property{
				"response": {Type: "string", Description: "Agent response text"},
			},
		},
		"ToolsRequest": {
			Type:     "object",
			Required: []string{"prompt", "tools"},
			Properties: map[string]*openapi.Property{
				"prompt":  {Type: "string", Description: "User prompt"},
				"tools":   {Type: "array", Description: "Available tools"},
				"token":   {Type: "string", Description: "Optional authentication token"},
				"options": {Type: "object", Description: "Optional agent options override"},
			},
		},
		"EmbedRequest": {
			Type:     "object",
			Required: []string{"input"},
			Properties: map[string]*openapi.Property{
				"input":   {Type: "string", Description: "Text to embed"},
				"token":   {Type: "string", Description: "Optional authentication token"},
				"options": {Type: "object", Description: "Optional embedding options"},
			},
		},
		"EmbedResponse": {
			Type: "object",
			Properties: map[string]*openapi.Property{
				"embedding": {Type: "array", Description: "Embedding vector"},
			},
		},
	}
}
```

#### Step 1.6: Create pkg/openapi/json.go

Marshal spec to JSON (uses stdlib, no external dependency):

```go
package openapi

import (
	"encoding/json"
	"os"
)

func MarshalJSON(spec *Spec) ([]byte, error) {
	return json.MarshalIndent(spec, "", "  ")
}

func WriteJSON(spec *Spec, filename string) error {
	data, err := MarshalJSON(spec)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
```

#### Step 1.7: Extend internal/routes for OpenAPI

**Update `internal/routes/group.go`** - Add OpenAPI field to Route struct:

```go
package routes

import (
	"net/http"

	"github.com/JaimeStill/agent-lab/pkg/openapi"
)

type Group struct {
	Prefix      string
	Tags        []string
	Description string
	Routes      []Route
}

type Route struct {
	Method  string
	Pattern string
	Handler http.HandlerFunc
	OpenAPI *openapi.Operation
}
```

**Update `internal/routes/routes.go`** - Add Groups() and Routes() methods to System interface and implementation:

```go
type System interface {
	RegisterGroup(group Group)
	RegisterRoute(route Route)
	Build() http.Handler
	Groups() []Group
	Routes() []Route
}

func (r *routes) Groups() []Group {
	return r.groups
}

func (r *routes) Routes() []Route {
	return r.routes
}
```

### Phase 2: Integrated OpenAPI Generation

#### Step 2.1: Create cmd/server/openapi.go

Generator logic integrated into server startup:

```go
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
		return nil, fmt.Errorf("write spec: %w", err)
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
			Title:       "agent-lab API",
			Version:     cfg.Version,
			Description: "REST API for managing LLM provider configurations, agent configurations, and executing agent operations.",
		},
		Servers:    []*openapi.Server{{URL: cfg.Domain}},
		Components: components,
		Paths:      make(map[string]*openapi.PathItem),
	}

	for _, group := range rs.Groups() {
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
	}

	for _, route := range rs.Routes() {
		if route.OpenAPI == nil {
			continue
		}

		addOperation(spec, route.Pattern, route.Method, route.OpenAPI)
	}

	return spec
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
```

### Phase 3: Self-Hosted Scalar Setup

#### Step 3.1: Create Scalar Update Infrastructure

Create the `scalar/` directory and update script:

```bash
mkdir -p scalar
```

**Create `scalar/update.sh`:**

```bash
#!/bin/bash
set -e

cd "$(dirname "$0")"

npm i @scalar/api-reference

mkdir -p ../web/docs

cp node_modules/@scalar/api-reference/dist/browser/standalone.js ../web/docs/scalar.js
cp node_modules/@scalar/api-reference/dist/style.css ../web/docs/scalar.css

rm -rf node_modules package-lock.json

echo "Scalar assets updated successfully"
```

Make executable:

```bash
chmod +x scalar/update.sh
```

#### Step 3.2: Download Scalar Assets

Run the update script:

```bash
./scalar/update.sh
```

This installs `@scalar/api-reference`, copies the assets to `web/docs/`, and cleans up.

The `scalar/package.json` tracks the installed version.

#### Step 3.3: Create web/docs/index.html

HTML template for Scalar UI:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>agent-lab API Documentation</title>
  <link rel="stylesheet" href="/docs/scalar.css">
</head>
<body>
  <script
    id="api-reference"
    data-url="/api/openapi.json"
    src="/docs/scalar.js">
  </script>
</body>
</html>
```

#### Step 3.4: Create web/docs/docs.go

Handler serving Scalar UI assets:

```go
package docs

import (
	_ "embed"
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/routes"
)

//go:embed index.html
var indexHTML []byte

//go:embed scalar.js
var scalarJS []byte

//go:embed scalar.css
var scalarCSS []byte

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix:      "/docs",
		Tags:        []string{"Documentation"},
		Description: "Interactive API documentation powered by Scalar",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.serveIndex},
			{Method: "GET", Pattern: "/scalar.js", Handler: h.serveJS},
			{Method: "GET", Pattern: "/scalar.css", Handler: h.serveCSS},
		},
	}
}

func (h *Handler) serveIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(indexHTML)
}

func (h *Handler) serveJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(scalarJS)
}

func (h *Handler) serveCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(scalarCSS)
}
```

#### Step 3.5: Create web/docs/README.md

Maintenance documentation:

````markdown
# API Documentation

This package serves the interactive API documentation via Scalar UI.

## Scalar Version

Check `scalar/package.json` for the installed version.

## Files

- `index.html` - HTML template that loads Scalar
- `scalar.js` - Scalar standalone JavaScript bundle (embedded)
- `scalar.css` - Scalar stylesheet (embedded)
- `docs.go` - Handler serving documentation at /docs endpoint

## Updating Scalar

Run the update script:

```bash
cd scalar && ./update.sh
```

Then test the documentation endpoint:

```bash
go run ./cmd/server
# Navigate to http://localhost:8080/docs
```

## OpenAPI Specification

The OpenAPI spec is generated on server startup and cached to `api/openapi.{env}.json`.
The docs handler serves the spec from memory (no file I/O per request).

**Spec Regeneration:**
- Happens automatically on startup if routes change
- Delete `api/openapi.*.json` to force regeneration
- Environment-specific: `openapi.local.json`, `openapi.prod.json`, etc.
````

### Phase 4: Add Routes with OpenAPI References

#### Step 4.1: Update internal/providers/handler.go

Update Routes() to reference operations via `Spec`:

```go
func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix:      "/api/providers",
		Tags:        []string{"Providers"},
		Description: "Provider configuration management",
		Routes: []routes.Route{
			{Method: "POST", Pattern: "", Handler: h.Create, OpenAPI: Spec.Create},
			{Method: "GET", Pattern: "", Handler: h.List, OpenAPI: Spec.List},
			{Method: "GET", Pattern: "/{id}", Handler: h.GetByID, OpenAPI: Spec.Get},
			{Method: "PUT", Pattern: "/{id}", Handler: h.Update, OpenAPI: Spec.Update},
			{Method: "DELETE", Pattern: "/{id}", Handler: h.Delete, OpenAPI: Spec.Delete},
			{Method: "POST", Pattern: "/search", Handler: h.Search, OpenAPI: Spec.Search},
		},
	}
}
```

#### Step 4.2: Update internal/agents/handler.go

Update Routes() to reference operations via `Spec`:

```go
func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix:      "/api/agents",
		Tags:        []string{"Agents"},
		Description: "Agent configuration and execution",
		Routes: []routes.Route{
			{Method: "POST", Pattern: "", Handler: h.Create, OpenAPI: Spec.Create},
			{Method: "GET", Pattern: "", Handler: h.List, OpenAPI: Spec.List},
			{Method: "GET", Pattern: "/{id}", Handler: h.GetByID, OpenAPI: Spec.Get},
			{Method: "PUT", Pattern: "/{id}", Handler: h.Update, OpenAPI: Spec.Update},
			{Method: "DELETE", Pattern: "/{id}", Handler: h.Delete, OpenAPI: Spec.Delete},
			{Method: "POST", Pattern: "/search", Handler: h.Search, OpenAPI: Spec.Search},
			{Method: "POST", Pattern: "/{id}/chat", Handler: h.Chat, OpenAPI: Spec.Chat},
			{Method: "POST", Pattern: "/{id}/chat/stream", Handler: h.ChatStream, OpenAPI: Spec.ChatStream},
			{Method: "POST", Pattern: "/{id}/vision", Handler: h.Vision, OpenAPI: Spec.Vision},
			{Method: "POST", Pattern: "/{id}/vision/stream", Handler: h.VisionStream, OpenAPI: Spec.VisionStream},
			{Method: "POST", Pattern: "/{id}/tools", Handler: h.Tools, OpenAPI: Spec.Tools},
			{Method: "POST", Pattern: "/{id}/embed", Handler: h.Embed, OpenAPI: Spec.Embed},
		},
	}
}
```

### Phase 5: Wire Up Server Integration

#### Step 5.1: Update cmd/server/routes.go

Wire up spec generation, API spec route, and docs handler:

```go
import (
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/agents"
	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/providers"
	"github.com/JaimeStill/agent-lab/internal/routes"
	"github.com/JaimeStill/agent-lab/pkg/openapi"
	"github.com/JaimeStill/agent-lab/web/docs"
)

func registerRoutes(r routes.System, runtime *Runtime, domain *Domain, cfg *config.Config) error {
	providerHandler := providers.NewHandler(domain.Providers, runtime.Logger, runtime.Pagination)
	r.RegisterGroup(providerHandler.Routes())

	agentHandler := agents.NewHandler(domain.Agents, runtime.Logger, runtime.Pagination)
	r.RegisterGroup(agentHandler.Routes())

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
	components.AddSchemas(providers.Spec.Schemas())
	components.AddSchemas(agents.Spec.Schemas())

	specBytes, err := loadOrGenerateSpec(cfg, r, components)
	if err != nil {
		return err
	}

	r.RegisterRoute(routes.Route{
		Method:  "GET",
		Pattern: "/api/openapi.json",
		Handler: serveOpenAPISpec(specBytes),
	})

	docsHandler := docs.NewHandler()
	r.RegisterGroup(docsHandler.Routes())

	return nil
}

func serveOpenAPISpec(spec []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(spec)
	}
}
```

#### Step 5.2: Update cmd/server/server.go

Adjust NewServer to handle spec generation:

```go
func NewServer(cfg *config.Config) (*Server, error) {
	runtime, err := NewRuntime(cfg)
	if err != nil {
		return nil, fmt.Errorf("create runtime: %w", err)
	}

	domain := NewDomain(runtime)

	routeSys := routes.New(runtime.Logger)

	if err := registerRoutes(routeSys, runtime, domain, cfg); err != nil {
		return nil, fmt.Errorf("register routes: %w", err)
	}

	runtime.Logger.Info("openapi spec loaded",
		"path", specFilePath(cfg.Env()),
		"version", cfg.Version)

	// Continue with middleware and HTTP server setup...
}
```

### Phase 6: Testing & Validation

#### Step 6.1: Install Dependencies

```bash
go mod tidy
```

#### Step 6.2: Start Service

```bash
go run ./cmd/server
```

Verify:
- OpenAPI spec generated at `api/openapi.local.json`
- Log message: `openapi spec loaded path=api/openapi.local.json version=0.1.0`

#### Step 6.3: Test Documentation

1. Navigate to `http://localhost:8080/docs`
2. Verify Scalar UI loads (no console errors)
3. Verify OpenAPI spec loads
4. Verify all endpoints are listed and grouped correctly
5. Check that request/response schemas render

#### Step 6.4: Test "Try It" Functionality

**Prerequisites:** Need at least one provider and agent configured

1. Test Provider CRUD:
   - Create provider
   - List providers
   - Get provider by ID
   - Update provider
   - Search providers

2. Test Agent CRUD:
   - Create agent
   - List agents
   - Get agent by ID

3. Test Agent Execution (if agent configured):
   - Chat endpoint
   - Verify request/response format

#### Step 6.5: Test Environment-Specific Specs

```bash
# Generate prod spec
SERVICE_ENV=prod go run ./cmd/server
# Verify api/openapi.prod.json created with prod domain

# Restart with local
SERVICE_ENV=local go run ./cmd/server
# Verify api/openapi.local.json used
```

### Phase 7: Documentation Updates

#### Step 7.1: Update README.md

Add after Quick Start section:

````markdown
### API Documentation

Interactive API documentation is available at:
```
http://localhost:8080/docs
```

The documentation is powered by [Scalar](https://github.com/scalar/scalar) and uses the OpenAPI 3.1 specification.

**OpenAPI Spec:**
- Generated automatically on server startup
- Cached to `api/openapi.{env}.json` (only writes when changed)
- Environment-specific: `openapi.local.json`, `openapi.prod.json`, etc.

**Force Spec Regeneration:**
```bash
rm api/openapi.*.json && go run ./cmd/server
```
```

#### Step 7.2: Update ARCHITECTURE.md

Add section under "HTTP Patterns":

```markdown
### API Documentation

**OpenAPI Specification:**
- Generated during server startup (no separate CLI tool)
- Location: `api/openapi.{env}.json` (environment-specific)
- Format: OpenAPI 3.1 (JSON)
- Change detection: only writes to disk when spec changes

**Config-Driven Metadata:**
- `version`: API version (used in OpenAPI info, logging, metrics)
- `domain`: External URL where clients access the service (used in OpenAPI servers)
- Environment overlays (`config.prod.toml`) customize per deployment

**Schema Ownership:**
- **Infrastructure** (`pkg/openapi/components.go`): Shared responses (BadRequest, NotFound, Conflict), PageRequest
- **Domains** (`internal/<domain>/openapi.go`): Domain-specific schemas and operation definitions

**Generation Flow:**
1. Server startup calls `registerRoutes()`
2. Domain handlers registered with OpenAPI metadata
3. `loadOrGenerateSpec()` generates spec in memory
4. Compares with existing file, writes only if changed
5. Docs handler receives spec bytes, serves from memory

**Import Hierarchy:**
- `pkg/openapi`: Types and helpers only (no internal imports)
- `cmd/server/openapi.go`: Generator logic (imports from both pkg and internal)

**Scalar UI:**
- Self-hosted (no CDN dependencies)
- Served at `/docs` endpoint
- Assets embedded via `go:embed`
- Maintenance: See `web/docs/README.md`

**Workflow:**
After API changes:
1. Update operation in domain openapi.go
2. Update domain schemas if entity changed
3. Restart server (spec auto-regenerates)
4. Commit updated `api/openapi.{env}.json`
5. Verify documentation at `/docs`
````

#### Step 7.3: Update PROJECT.md

Mark OpenAPI session complete in Current Status section.

### Phase 8: Infrastructure Consolidation

#### Step 8.1: Consolidate scalar/ into web/

Move the Scalar update script from `scalar/` to `web/`:

```bash
mv scalar/update.sh web/update-scalar.sh
rm -rf scalar/
```

Update the script paths in `web/update-scalar.sh`:

```bash
#!/bin/bash
set -e

cd "$(dirname "$0")"

npm i @scalar/api-reference

cp node_modules/@scalar/api-reference/dist/browser/standalone.js docs/scalar.js
cp node_modules/@scalar/api-reference/dist/style.css docs/scalar.css

rm -rf node_modules package-lock.json

echo "Scalar assets updated successfully"
```

Make executable:

```bash
chmod +x web/update-scalar.sh
```

#### Step 8.2: Add Trailing Slash Middleware

Create `internal/middleware/slash.go`:

```go
package middleware

import (
	"net/http"
	"strings"
)

func TrimSlash() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(r.URL.Path) > 1 && strings.HasSuffix(r.URL.Path, "/") {
				target := strings.TrimSuffix(r.URL.Path, "/")
				if r.URL.RawQuery != "" {
					target += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, target, http.StatusMovedPermanently)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

#### Step 8.3: Update Middleware Chain

Update `cmd/server/middleware.go` to include TrimSlash (first in chain):

```go
func buildMiddleware(runtime *Runtime, cfg *config.Config) middleware.System {
	middlewareSys := middleware.New()
	middlewareSys.Use(middleware.TrimSlash())
	middlewareSys.Use(middleware.Logger(runtime.Logger))
	middlewareSys.Use(middleware.CORS(&cfg.CORS))
	return middlewareSys
}
```

#### Step 8.4: Create web/README.md

Create comprehensive client development documentation at `web/README.md`. See implementation below.

## Validation Checklist

- [ ] `pkg/openapi` package compiles
- [ ] `cmd/server/openapi.go` compiles
- [ ] Service starts without errors
- [ ] `api/openapi.local.json` generated on startup
- [ ] Spec only writes when content changes (check log on restart)
- [ ] `web/docs` package compiles
- [ ] `/docs` endpoint serves Scalar UI
- [ ] `/api/openapi.json` serves specification
- [ ] Scalar loads without CDN errors (check browser console)
- [ ] All endpoint groups visible (Providers, Agents, Infrastructure)
- [ ] Request/response schemas render correctly
- [ ] "Try It" functionality works for test endpoints
- [ ] Environment-specific specs work (`SERVICE_ENV=prod`)
- [ ] `scalar/` directory removed
- [ ] `web/update-scalar.sh` works correctly
- [ ] Trailing slash redirect works (`/docs/` → `/docs`)
- [ ] `web/README.md` created
- [ ] README.md updated
- [ ] ARCHITECTURE.md updated
- [ ] PROJECT.md updated

## Notes

**Dependencies Added:**
- None (uses stdlib `encoding/json`)

**Files Created:**
- `pkg/openapi/types.go` - OpenAPI 3.1 type definitions
- `pkg/openapi/components.go` - NewComponents constructor, AddSchemas, shared responses
- `pkg/openapi/json.go` - JSON marshaling (stdlib)
- `internal/providers/openapi.go` - Domain-owned provider schemas and operations
- `internal/agents/openapi.go` - Domain-owned agent schemas and operations
- `internal/middleware/slash.go` - Trailing slash redirect middleware
- `cmd/server/openapi.go` - Spec generation logic (integrated)
- `web/docs/docs.go` - Documentation handler
- `web/docs/index.html` - Scalar UI template (data attribute initialization)
- `web/docs/scalar.js` (downloaded via web/update-scalar.sh)
- `web/docs/scalar.css` (downloaded via web/update-scalar.sh)
- `web/update-scalar.sh` - Script to update Scalar assets
- `web/README.md` - Client development architecture documentation
- `api/openapi.{env}.json` (generated on startup)

**Files Modified:**
- `internal/config/config.go` (added Version, Domain, Env() method)
- `internal/config/types.go` (added EnvServiceVersion, EnvServiceDomain, EnvServiceEnv)
- `internal/routes/group.go` (added OpenAPI field to Route)
- `internal/providers/handler.go` (Routes() references openapi.go operations)
- `internal/agents/handler.go` (Routes() references openapi.go operations)
- `cmd/server/routes.go` (spec generation, docs handler registration)
- `cmd/server/server.go` (logging spec load)
- `cmd/server/middleware.go` (added TrimSlash to middleware chain)
- `config.toml` (added version, domain)
- `.env` (added SERVICE_VERSION, SERVICE_DOMAIN, SERVICE_ENV)
- `README.md`
- `ARCHITECTURE.md`
- `PROJECT.md`

**Files Removed:**
- `scalar/` directory (consolidated into `web/`)

## Future Enhancements

**Not in this session:**
- OpenAPI validation in CI
- Request/response examples in schemas
- Client SDK generation from spec
- Pre-commit hook for spec validation
