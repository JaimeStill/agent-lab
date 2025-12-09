package agents

import "github.com/JaimeStill/agent-lab/pkg/openapi"

// spec holds OpenAPI operation definitions for the agents domain.
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

// Spec contains OpenAPI operation definitions for all agent endpoints.
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
						Properties: map[string]*openapi.Schema{
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
						Properties: map[string]*openapi.Schema{
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

// Schemas returns the agent domain schemas for OpenAPI components.
func (spec) Schemas() map[string]*openapi.Schema {
	return map[string]*openapi.Schema{
		"Agent": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
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
			Properties: map[string]*openapi.Schema{
				"name":   {Type: "string", Example: "gpt-4o"},
				"config": {Type: "object", Description: "go-agents AgentConfig as JSON"},
			},
		},
		"UpdateAgentCommand": {
			Type:     "object",
			Required: []string{"name", "config"},
			Properties: map[string]*openapi.Schema{
				"name":   {Type: "string"},
				"config": {Type: "object", Description: "go-agents AgentConfig as JSON"},
			},
		},
		"AgentPageResult": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"data":        {Type: "array", Items: openapi.SchemaRef("Agent")},
				"total":       {Type: "integer", Description: "Total number of results"},
				"page":        {Type: "integer", Description: "Current page number"},
				"page_size":   {Type: "integer", Description: "Results per page"},
				"total_pages": {Type: "integer", Description: "Total number of pages"},
			},
		},
		"ChatRequest": {
			Type:     "object",
			Required: []string{"prompt"},
			Properties: map[string]*openapi.Schema{
				"prompt":  {Type: "string", Description: "User prompt"},
				"token":   {Type: "string", Description: "Optional authentication token (for Azure providers)"},
				"options": {Type: "object", Description: "Optional agent options override"},
			},
		},
		"ChatResponse": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"response": {Type: "string", Description: "Agent response text"},
			},
		},
		"ToolsRequest": {
			Type:     "object",
			Required: []string{"prompt", "tools"},
			Properties: map[string]*openapi.Schema{
				"prompt":  {Type: "string", Description: "User prompt"},
				"tools":   {Type: "array", Description: "Available tools"},
				"token":   {Type: "string", Description: "Optional authentication token"},
				"options": {Type: "object", Description: "Optional agent options override"},
			},
		},
		"EmbedRequest": {
			Type:     "object",
			Required: []string{"input"},
			Properties: map[string]*openapi.Schema{
				"prompt":  {Type: "string", Description: "Text to embed"},
				"token":   {Type: "string", Description: "Optional authentication token"},
				"options": {Type: "object", Description: "Optional embedding options"},
			},
		},
		"EmbedResponse": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"embedding": {Type: "array", Description: "Embedding vector"},
			},
		},
	}
}

