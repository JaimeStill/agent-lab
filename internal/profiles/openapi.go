package profiles

import "github.com/JaimeStill/agent-lab/pkg/openapi"

type spec struct {
	List        *openapi.Operation
	Create      *openapi.Operation
	Find        *openapi.Operation
	Update      *openapi.Operation
	Delete      *openapi.Operation
	SetStage    *openapi.Operation
	DeleteStage *openapi.Operation
}

var Spec = spec{
	List: &openapi.Operation{
		Summary:     "List profiles",
		Description: "Returns a paginated list of workflow profiles with optional filtering",
		Parameters: []*openapi.Parameter{
			openapi.QueryParam("page", "integer", "Page number (1-indexed)", false),
			openapi.QueryParam("page_size", "integer", "Results per page", false),
			openapi.QueryParam("search", "string", "Search query (matches name)", false),
			openapi.QueryParam("sort", "string", "Comma-separated sort fields. Prefix with - for descending", false),
			openapi.QueryParam("workflow_name", "string", "Filter by workflow name", false),
		},
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Paginated list of profiles", "ProfilePageResult"),
		},
	},
	Create: &openapi.Operation{
		Summary:     "Create profile",
		Description: "Creates a new workflow profile",
		RequestBody: openapi.RequestBodyJSON("CreateProfileCommand", true),
		Responses: map[int]*openapi.Response{
			201: openapi.ResponseJSON("Created profile", "Profile"),
			400: openapi.ResponseRef("BadRequest"),
			409: openapi.ResponseRef("Conflict"),
		},
	},
	Find: &openapi.Operation{
		Summary:     "Get profile",
		Description: "Returns a profile with all its stage configurations",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Profile UUID"),
		},
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Profile with stages", "ProfileWithStages"),
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
	Update: &openapi.Operation{
		Summary:     "Update profile",
		Description: "Updates profile metadata (name, description)",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Profile UUID"),
		},
		RequestBody: openapi.RequestBodyJSON("UpdateProfileCommand", true),
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Updated profile", "Profile"),
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
			409: openapi.ResponseRef("Conflict"),
		},
	},
	Delete: &openapi.Operation{
		Summary:     "Delete profile",
		Description: "Deletes a profile and all its stage configurations",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Profile UUID"),
		},
		Responses: map[int]*openapi.Response{
			204: {Description: "Profile deleted"},
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
	SetStage: &openapi.Operation{
		Summary:     "Set stage configuration",
		Description: "Creates or updates a stage configuration for a profile (save)",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Profile UUID"),
		},
		RequestBody: openapi.RequestBodyJSON("SetProfileStageCommand", true),
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Stage configuration", "ProfileStage"),
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
	DeleteStage: &openapi.Operation{
		Summary:     "Delete stage configuration",
		Description: "Removes a stage configuration from a profile",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Profile UUID"),
			openapi.PathParam("stage", "Stage name"),
		},
		Responses: map[int]*openapi.Response{
			204: {Description: "Stage deleted"},
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
}

func (spec) Schemas() map[string]*openapi.Schema {
	return map[string]*openapi.Schema{
		"Profile": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"id":            {Type: "string", Format: "uuid"},
				"workflow_name": {Type: "string"},
				"name":          {Type: "string"},
				"description":   {Type: "string"},
				"created_at":    {Type: "string", Format: "date-time"},
				"updated_at":    {Type: "string", Format: "date-time"},
			},
		},
		"ProfileStage": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"profile_id":    {Type: "string", Format: "uuid"},
				"stage_name":    {Type: "string"},
				"agent_id":      {Type: "string", Format: "uuid"},
				"system_prompt": {Type: "string"},
				"options":       {Type: "object", Description: "Stage-specific options (JSON)"},
			},
		},
		"ProfileWithStages": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"id":            {Type: "string", Format: "uuid"},
				"workflow_name": {Type: "string"},
				"name":          {Type: "string"},
				"description":   {Type: "string"},
				"created_at":    {Type: "string", Format: "date-time"},
				"updated_at":    {Type: "string", Format: "date-time"},
				"stages":        {Type: "array", Items: openapi.SchemaRef("ProfileStage")},
			},
		},
		"CreateProfileCommand": {
			Type:     "object",
			Required: []string{"workflow_name", "name"},
			Properties: map[string]*openapi.Schema{
				"workflow_name": {Type: "string", Example: "summarize"},
				"name":          {Type: "string", Example: "concise-v1"},
				"description":   {Type: "string", Example: "Optimized for brief summaries"},
			},
		},
		"UpdateProfileCommand": {
			Type:     "object",
			Required: []string{"name"},
			Properties: map[string]*openapi.Schema{
				"name":        {Type: "string"},
				"description": {Type: "string"},
			},
		},
		"SetProfileStageCommand": {
			Type:     "object",
			Required: []string{"stage_name"},
			Properties: map[string]*openapi.Schema{
				"stage_name":    {Type: "string", Example: "summarize"},
				"agent_id":      {Type: "string", Format: "uuid", Description: "Override agent for this stage"},
				"system_prompt": {Type: "string", Description: "System prompt for this stage"},
				"options":       {Type: "object", Description: "Additional stage options"},
			},
		},
		"ProfilePageResult": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"data":        {Type: "array", Items: openapi.SchemaRef("Profile")},
				"page":        {Type: "integer"},
				"page_size":   {Type: "integer"},
				"total_count": {Type: "integer"},
				"total_pages": {Type: "integer"},
			},
		},
	}
}
