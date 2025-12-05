package pkg_openapi_test

import (
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/openapi"
)

func TestSchemaRef(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantRef string
	}{
		{"simple name", "User", "#/components/schemas/User"},
		{"compound name", "CreateUserCommand", "#/components/schemas/CreateUserCommand"},
		{"with numbers", "PageResult2", "#/components/schemas/PageResult2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := openapi.SchemaRef(tt.input)

			if schema.Ref != tt.wantRef {
				t.Errorf("SchemaRef(%q).Ref = %q, want %q", tt.input, schema.Ref, tt.wantRef)
			}
		})
	}
}

func TestResponseRef(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantRef string
	}{
		{"bad request", "BadRequest", "#/components/responses/BadRequest"},
		{"not found", "NotFound", "#/components/responses/NotFound"},
		{"conflict", "Conflict", "#/components/responses/Conflict"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := openapi.ResponseRef(tt.input)

			if resp.Ref != tt.wantRef {
				t.Errorf("ResponseRef(%q).Ref = %q, want %q", tt.input, resp.Ref, tt.wantRef)
			}
		})
	}
}

func TestRequestBodyJSON(t *testing.T) {
	tests := []struct {
		name         string
		schemaName   string
		required     bool
		wantRequired bool
	}{
		{"required body", "CreateUser", true, true},
		{"optional body", "UpdateUser", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := openapi.RequestBodyJSON(tt.schemaName, tt.required)

			if body.Required != tt.wantRequired {
				t.Errorf("Required = %v, want %v", body.Required, tt.wantRequired)
			}

			content, ok := body.Content["application/json"]
			if !ok {
				t.Fatal("Content missing application/json media type")
			}

			wantRef := "#/components/schemas/" + tt.schemaName
			if content.Schema.Ref != wantRef {
				t.Errorf("Schema.Ref = %q, want %q", content.Schema.Ref, wantRef)
			}
		})
	}
}

func TestResponseJSON(t *testing.T) {
	tests := []struct {
		name        string
		description string
		schemaName  string
	}{
		{"user response", "User retrieved", "User"},
		{"list response", "List of items", "ItemList"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := openapi.ResponseJSON(tt.description, tt.schemaName)

			if resp.Description != tt.description {
				t.Errorf("Description = %q, want %q", resp.Description, tt.description)
			}

			content, ok := resp.Content["application/json"]
			if !ok {
				t.Fatal("Content missing application/json media type")
			}

			wantRef := "#/components/schemas/" + tt.schemaName
			if content.Schema.Ref != wantRef {
				t.Errorf("Schema.Ref = %q, want %q", content.Schema.Ref, wantRef)
			}
		})
	}
}

func TestPathParam(t *testing.T) {
	tests := []struct {
		name        string
		paramName   string
		description string
	}{
		{"id parameter", "id", "Resource UUID"},
		{"user_id parameter", "user_id", "User identifier"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			param := openapi.PathParam(tt.paramName, tt.description)

			if param.Name != tt.paramName {
				t.Errorf("Name = %q, want %q", param.Name, tt.paramName)
			}

			if param.In != "path" {
				t.Errorf("In = %q, want %q", param.In, "path")
			}

			if !param.Required {
				t.Error("Required = false, want true")
			}

			if param.Description != tt.description {
				t.Errorf("Description = %q, want %q", param.Description, tt.description)
			}

			if param.Schema.Type != "string" {
				t.Errorf("Schema.Type = %q, want %q", param.Schema.Type, "string")
			}

			if param.Schema.Format != "uuid" {
				t.Errorf("Schema.Format = %q, want %q", param.Schema.Format, "uuid")
			}
		})
	}
}

func TestQueryParam(t *testing.T) {
	tests := []struct {
		name        string
		paramName   string
		typ         string
		description string
		required    bool
	}{
		{"required string", "search", "string", "Search query", true},
		{"optional integer", "page", "integer", "Page number", false},
		{"required boolean", "active", "boolean", "Filter active", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			param := openapi.QueryParam(tt.paramName, tt.typ, tt.description, tt.required)

			if param.Name != tt.paramName {
				t.Errorf("Name = %q, want %q", param.Name, tt.paramName)
			}

			if param.In != "query" {
				t.Errorf("In = %q, want %q", param.In, "query")
			}

			if param.Required != tt.required {
				t.Errorf("Required = %v, want %v", param.Required, tt.required)
			}

			if param.Description != tt.description {
				t.Errorf("Description = %q, want %q", param.Description, tt.description)
			}

			if param.Schema.Type != tt.typ {
				t.Errorf("Schema.Type = %q, want %q", param.Schema.Type, tt.typ)
			}
		})
	}
}
