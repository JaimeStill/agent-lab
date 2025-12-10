package images

import "github.com/JaimeStill/agent-lab/pkg/openapi"

// spec defines OpenAPI operations for image endpoints.
type spec struct {
	List   *openapi.Operation
	Find   *openapi.Operation
	Data   *openapi.Operation
	Render *openapi.Operation
	Delete *openapi.Operation
}

// Spec provides OpenAPI specifications for all image endpoints.
var Spec = spec{
	List: &openapi.Operation{
		Summary:     "List images",
		Description: "List rendered images with optional filters and pagination",
		Parameters: []*openapi.Parameter{
			openapi.QueryParam("document_id", "string", "Filter by document ID", false),
			openapi.QueryParam("page", "integer", "Page number", false),
			openapi.QueryParam("page_size", "integer", "Items per page", false),
			openapi.QueryParam("format", "string", "Filter by format (png or jpg)", false),
			openapi.QueryParam("page_number", "integer", "Filter by page number", false),
		},
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Image list", "ImagePageResult"),
		},
	},
	Find: &openapi.Operation{
		Summary:     "Find image metadata",
		Description: "Find metadata for a rendered image",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Image ID"),
		},
		Responses: map[int]*openapi.Response{
			200: openapi.ResponseJSON("Image metadata", "Image"),
			404: openapi.ResponseRef("NotFound"),
		},
	},
	Data: &openapi.Operation{
		Summary:     "Get image binary",
		Description: "Get the raw binary data for a rendered image",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Image ID"),
		},
		Responses: map[int]*openapi.Response{
			200: {
				Description: "Image binary data",
				Content: map[string]*openapi.MediaType{
					"image/png":  {Schema: &openapi.Schema{Type: "string", Format: "binary"}},
					"image/jpeg": {Schema: &openapi.Schema{Type: "string", Format: "binary"}},
				},
			},
			404: openapi.ResponseRef("NotFound"),
		},
	},
	Render: &openapi.Operation{
		Summary:     "Render document pages",
		Description: "Render document pages to images. Supports batch rendering with page range expressions (e.g., '1-5,10,15-20'). Currently supports PDF files.",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("documentId", "Document ID"),
		},
		RequestBody: openapi.RequestBodyJSON("RenderRequest", false),
		Responses: map[int]*openapi.Response{
			201: openapi.ResponseJSON("Images rendered", "ImageArray"),
			400: openapi.ResponseRef("BadRequest"),
			404: openapi.ResponseRef("NotFound"),
			500: {Description: "Render failed"},
		},
	},
	Delete: &openapi.Operation{
		Summary:     "Delete image",
		Description: "Delete a rendered image from storage and database",
		Parameters: []*openapi.Parameter{
			openapi.PathParam("id", "Image ID"),
		},
		Responses: map[int]*openapi.Response{
			204: {Description: "Image deleted"},
			404: openapi.ResponseRef("NotFound"),
		},
	},
}

// Schemas returns OpenAPI schemas for image-related types.
func (spec) Schemas() map[string]*openapi.Schema {
	return map[string]*openapi.Schema{
		"Image": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"id":          {Type: "string", Format: "uuid"},
				"document_id": {Type: "string", Format: "uuid"},
				"page_number": {Type: "integer", Description: "Page number (1-indexed)"},
				"format":      {Type: "string", Description: "Image format (png or jpg)"},
				"dpi":         {Type: "integer", Description: "Resolution in DPI"},
				"quality":     {Type: "integer", Description: "JPEG quality (1-100)"},
				"brightness":  {Type: "integer", Description: "Brightness adjustment (0-200)"},
				"contrast":    {Type: "integer", Description: "Contrast adjustment (-100 to 100)"},
				"saturation":  {Type: "integer", Description: "Saturation adjustment (0-200)"},
				"rotation":    {Type: "integer", Description: "Rotation in degrees (0-360)"},
				"background":  {Type: "string", Description: "Background color name"},
				"storage_key": {Type: "string", Description: "Storage location key"},
				"size_bytes":  {Type: "integer", Format: "int64", Description: "File size in bytes"},
				"created_at":  {Type: "string", Format: "date-time"},
			},
		},
		"ImageArray": {
			Type:  "array",
			Items: openapi.SchemaRef("Image"),
		},
		"ImagePageResult": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"data":        {Type: "array", Items: openapi.SchemaRef("Image")},
				"total":       {Type: "integer"},
				"page":        {Type: "integer"},
				"page_size":   {Type: "integer"},
				"total_pages": {Type: "integer"},
			},
		},
		"RenderRequest": {
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"pages":      {Type: "string", Description: "Page range expression (e.g., '1-5,10,15-20'). Omit to render all pages."},
				"format":     {Type: "string", Description: "Output format", Enum: []any{"png", "jpg"}, Default: "png"},
				"dpi":        {Type: "integer", Description: "Resolution in DPI (72-1200)", Minimum: floatPtr(72), Maximum: floatPtr(1200), Default: 300},
				"quality":    {Type: "integer", Description: "JPEG quality (1-100, only applies to jpg format)", Minimum: floatPtr(1), Maximum: floatPtr(100), Default: 90},
				"brightness": {Type: "integer", Description: "Brightness adjustment (0-200, 100 is neutral)", Minimum: floatPtr(0), Maximum: floatPtr(200), Default: 100},
				"contrast":   {Type: "integer", Description: "Contrast adjustment (-100 to 100, 0 is neutral)", Minimum: floatPtr(-100), Maximum: floatPtr(100), Default: 0},
				"saturation": {Type: "integer", Description: "Saturation adjustment (0-200, 100 is neutral)", Minimum: floatPtr(0), Maximum: floatPtr(200), Default: 100},
				"rotation":   {Type: "integer", Description: "Rotation in degrees (0-360)", Minimum: floatPtr(0), Maximum: floatPtr(360), Default: 0},
				"background": {Type: "string", Description: "Background color name", Default: "white"},
				"force":      {Type: "boolean", Description: "Re-render even if matching image exists", Default: false},
			},
		},
	}
}

func floatPtr(v float64) *float64 {
	return &v
}
