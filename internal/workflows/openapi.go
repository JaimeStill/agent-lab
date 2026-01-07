package workflows

import "github.com/JaimeStill/agent-lab/pkg/openapi"

type spec struct {
	ListWorkflows *openapi.Operation
	Execute       *openapi.Operation
	ListRuns      *openapi.Operation
	FindRun       *openapi.Operation
	GetStages     *openapi.Operation
	GetDecisions  *openapi.Operation
	DeleteRun     *openapi.Operation
	Cancel        *openapi.Operation
	Resume        *openapi.Operation
}

var Spec = spec{
	ListWorkflows: &openapi.Operation{
		Summary:     "List registered workflows",
		Description: "Returns all workflows registered in the system",
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("List of workflows", "WorkflowInfoList"),
		},
	},
	Execute: &openapi.Operation{
		Summary:     "Execute workflow",
		Description: "Executes a workflow and streams progress events via SSE",
		Parameters: []*openapi.Parameter{
			{
				Name:        "name",
				In:          "path",
				Required:    true,
				Description: "Workflow name",
				Schema:      &openapi.Schema{Type: "string"},
			},
		},
		RequestBody: openapi.RequestBodyJSON("ExecuteRequest", false),
		Responses: map[int]*openapi.Response{
			200: {
				Description: "SSE event stream",
				Content: map[string]*openapi.MediaType{
					"text/event-stream": {
						Schema: openapi.SchemaRef("ExecutionEvent"),
					},
				},
			},
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
	ListRuns: &openapi.Operation{
		Summary:     "List workflow runs",
		Description: "Returns paginated list of workflow runs with optional filters",
		Parameters: []*openapi.Parameter{
			openapi.QueryParam("page", "integer", "Page number", false),
			openapi.QueryParam("page_size", "integer", "Items per page", false),
			openapi.QueryParam("workflow_name", "string", "Filter by workflow name", false),
			openapi.QueryParam("status", "string", "Filter by status", false),
		},
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Paginated runs", "RunPageResult"),
		},
	},
	FindRun: &openapi.Operation{
		Summary:     "Get run details",
		Description: "Returns details for a specific workflow run",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Run ID"),
		},
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Run details", "Run"),
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
	GetStages: &openapi.Operation{
		Summary:     "Get run stages",
		Description: "Returns execution stages for a workflow run",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Run ID"),
		},
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Stage list", "StageList"),
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
	GetDecisions: &openapi.Operation{
		Summary:     "Get run decisions",
		Description: "Returns routing decisions for a workflow run",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Run ID"),
		},
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Decision list", "DecisionList"),
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
	DeleteRun: &openapi.Operation{
		Summary:     "Delete workflow run",
		Description: "Deletes a workflow run and its related data (stages, decisions, checkpoints)",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Run ID"),
		},
		Responses: map[int]*openapi.Response{
			204: {Description: "Run deleted"},
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
	Cancel: &openapi.Operation{
		Summary:     "Cancel workflow run",
		Description: "Cancels an active workflow run",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Run ID"),
		},
		Responses: map[int]*openapi.Response{
			204: {Description: "Run cancelled"},
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
			409: openapi.ResponseRef("Conflict"),
		},
	},
	Resume: &openapi.Operation{
		Summary:     "Resume workflow run",
		Description: "Resumes a failed or cancelled workflow run from checkpoint",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Run ID"),
		},
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Resumed run", "Run"),
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
			409: openapi.ResponseRef("Conflict"),
		},
	},
}

func (spec) Schemas() map[string]*openapi.Schema {
	return map[string]*openapi.Schema{
		"WorkflowInfo": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"name":        {Type: "string"},
				"description": {Type: "string"},
			},
		},
		"WorkflowInfoList": {
			Type:  "array",
			Items: openapi.SchemaRef("WorkflowInfo"),
		},
		"Run": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"id":            {Type: "string", Format: "uuid"},
				"workflow_name": {Type: "string"},
				"status":        {Type: "string", Enum: []any{"pending", "running", "completed", "failed", "cancelled"}},
				"params":        {Type: "object"},
				"result":        {Type: "object"},
				"error_message": {Type: "string"},
				"started_at":    {Type: "string", Format: "date-time"},
				"completed_at":  {Type: "string", Format: "date-time"},
				"created_at":    {Type: "string", Format: "date-time"},
				"updated_at":    {Type: "string", Format: "date-time"},
			},
		},
		"RunPageResult": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"items":       {Type: "array", Items: openapi.SchemaRef("Run")},
				"total":       {Type: "integer"},
				"page":        {Type: "integer"},
				"page_size":   {Type: "integer"},
				"total_pages": {Type: "integer"},
			},
		},
		"Stage": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"id":              {Type: "string", Format: "uuid"},
				"run_id":          {Type: "string", Format: "uuid"},
				"node_name":       {Type: "string"},
				"iteration":       {Type: "integer"},
				"status":          {Type: "string", Enum: []any{"started", "completed", "failed"}},
				"input_snapshot":  {Type: "object"},
				"output_snapshot": {Type: "object"},
				"duration_ms":     {Type: "integer"},
				"error_message":   {Type: "string"},
				"created_at":      {Type: "string", Format: "date-time"},
			},
		},
		"StageList": {
			Type:  "array",
			Items: openapi.SchemaRef("Stage"),
		},
		"Decision": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"id":               {Type: "string", Format: "uuid"},
				"run_id":           {Type: "string", Format: "uuid"},
				"from_node":        {Type: "string"},
				"to_node":          {Type: "string"},
				"predicate_name":   {Type: "string"},
				"predicate_result": {Type: "boolean"},
				"reason":           {Type: "string"},
				"created_at":       {Type: "string", Format: "date-time"},
			},
		},
		"DecisionList": {
			Type:  "array",
			Items: openapi.SchemaRef("Decision"),
		},
		"ExecuteRequest": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"params": {Type: "object", Description: "Workflow parameters"},
				"token":  {Type: "string", Description: "Auth token for agent API calls (not persisted)"},
			},
		},
		"ExecutionEvent": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"type":      {Type: "string", Enum: []any{"stage.start", "stage.complete", "decision", "error", "complete"}},
				"timestamp": {Type: "string", Format: "date-time"},
				"data":      {Type: "object"},
			},
		},
	}
}
