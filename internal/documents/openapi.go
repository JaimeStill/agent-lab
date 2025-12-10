package documents

import "github.com/JaimeStill/agent-lab/pkg/openapi"

type spec struct {
	List   *openapi.Operation
	Find   *openapi.Operation
	Search *openapi.Operation
	Upload *openapi.Operation
	Update *openapi.Operation
	Delete *openapi.Operation
}

var Spec = spec{
	List: &openapi.Operation{
		Summary:     "List documents",
		Description: "List documents with pagination and optional filters",
		Parameters: []*openapi.Parameter{
			openapi.QueryParam("page", "integer", "Page number", false),
			openapi.QueryParam("page_size", "integer", "Items per page", false),
			openapi.QueryParam("search", "string", "Search in name and filename", false),
			openapi.QueryParam("name", "string", "Filter by name (contains)", false),
			openapi.QueryParam("content_type", "string", "Filter by content type (contains)", false),
		},
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Documents list", "DocumentPageResult"),
		},
	},
	Find: &openapi.Operation{
		Summary:     "Find document",
		Description: "Find document by ID",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Document ID"),
		},
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Document details", "Document"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
	Search: &openapi.Operation{
		Summary:     "Search documents",
		Description: "Search documents with pagination in request body",
		Parameters: []*openapi.Parameter{
			openapi.QueryParam("name", "string", "Filter by name (contains)", false),
			openapi.QueryParam("content_type", "string", "Filter by content type (contains)", false),
		},
		RequestBody: openapi.RequestBodyJSON("PageRequest", true),
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Search results", "DocumentPageResult"),
			400: openapi.ResponseRef("BadRequest"),
		},
	},
	Upload: &openapi.Operation{
		Summary:     "Upload document",
		Description: "Upload a document file with optional display name. PDFs have page count extracted automatically.",
		RequestBody: &openapi.RequestBody{
			Required: true,
			Content: map[string]*openapi.MediaType{
				"multipart/form-data": {
					Schema: &openapi.Schema{
						Type: "object",
						Properties: map[string]*openapi.Schema{
							"file": {Type: "string", Description: "Document file to upload"},
							"name": {Type: "string", Description: "Optional display name (defaults to filename)"},
						},
						Required: []string{"file"},
					},
				},
			},
		},
		Responses: map[int]*openapi.Response{
			201: openapi.ResponseJSON("Document uploaded", "Document"),
			400: openapi.ResponseRef("BadRequest"),
			413: {Description: "File too large"},
		},
	},
	Update: &openapi.Operation{
		Summary:     "Update document",
		Description: "Update document display name",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Document ID"),
		},
		RequestBody: openapi.RequestBodyJSON("UpdateDocumentCommand", true),
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Document updated", "Document"),
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
	Delete: &openapi.Operation{
		Summary:     "Delete document",
		Description: "Delete document and its stored file",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Document ID"),
		},
		Responses: map[int]*openapi.Response{
			204: {Description: "Document deleted"},
			404: openapi.ResponseRef("NotFound"),
		},
	},
}

func (spec) Schemas() map[string]*openapi.Schema {
	return map[string]*openapi.Schema{
		"Document": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"id":           {Type: "string", Format: "uuid"},
				"name":         {Type: "string", Description: "Display name"},
				"filename":     {Type: "string", Description: "Original filename"},
				"content_type": {Type: "string", Description: "MIME type"},
				"size_bytes":   {Type: "integer", Format: "int64", Description: "File size in bytes"},
				"page_count":   {Type: "integer", Description: "Page count (PDFs only)"},
				"storage_key":  {Type: "string", Description: "Storage location key"},
				"created_at":   {Type: "string", Format: "date-time"},
				"updated_at":   {Type: "string", Format: "date-time"},
			},
		},
		"UpdateDocumentCommand": {
			Type:     "object",
			Required: []string{"name"},
			Properties: map[string]*openapi.Schema{
				"name": {Type: "string", Description: "New display name"},
			},
		},
		"DocumentPageResult": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"data":        {Type: "array", Items: openapi.SchemaRef("Document")},
				"total":       {Type: "integer"},
				"page":        {Type: "integer"},
				"page_size":   {Type: "integer"},
				"total_pages": {Type: "integer"},
			},
		},
	}
}
