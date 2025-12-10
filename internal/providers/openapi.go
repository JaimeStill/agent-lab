package providers

import "github.com/JaimeStill/agent-lab/pkg/openapi"

// spec holds OpenAPI operation definitions for the providers domain.
type spec struct {
	List   *openapi.Operation
	Find   *openapi.Operation
	Search *openapi.Operation
	Create *openapi.Operation
	Update *openapi.Operation
	Delete *openapi.Operation
}

// Spec contains OpenAPI operation definitions for all provider endpoints.
var Spec = spec{
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
	Find: &openapi.Operation{
		Summary:     "Find provider by ID",
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
}

// Schemas returns the provider domain schemas for OpenAPI components.
func (spec) Schemas() map[string]*openapi.Schema {
	return map[string]*openapi.Schema{
		"Provider": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
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
			Properties: map[string]*openapi.Schema{
				"name":   {Type: "string", Example: "ollama"},
				"config": {Type: "object", Description: "go-agents ProviderConfig as JSON"},
			},
		},
		"UpdateProviderCommand": {
			Type:     "object",
			Required: []string{"name", "config"},
			Properties: map[string]*openapi.Schema{
				"name":   {Type: "string"},
				"config": {Type: "object", Description: "go-agents ProviderConfig as JSON"},
			},
		},
		"ProviderPageResult": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"data":        {Type: "array", Items: openapi.SchemaRef("Provider")},
				"total":       {Type: "integer", Description: "Total number of results"},
				"page":        {Type: "integer", Description: "Current page number"},
				"page_size":   {Type: "integer", Description: "Results per page"},
				"total_pages": {Type: "integer", Description: "Total number of pages"},
			},
		},
	}
}
