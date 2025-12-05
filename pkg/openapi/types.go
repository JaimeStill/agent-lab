// Package openapi provides types and utilities for generating OpenAPI 3.1 specifications.
// It offers a programmatic approach to building API documentation that integrates
// with the routes system to auto-generate specifications at server startup.
package openapi

// Spec represents a complete OpenAPI 3.1 specification document.
type Spec struct {
	OpenAPI    string               `json:"openapi"`
	Info       *Info                `json:"info"`
	Servers    []*Server            `json:"servers,omitempty"`
	Paths      map[string]*PathItem `json:"paths"`
	Components *Components          `json:"components,omitempty"`
}

// Info provides metadata about the API.
type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// Server represents a server URL for the API.
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// PathItem describes operations available on a single path.
type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
}

// Operation describes a single API operation on a path.
type Operation struct {
	Summary     string            `json:"summary,omitempty"`
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Parameters  []*Parameter      `json:"parameters,omitempty"`
	RequestBody *RequestBody      `json:"requestBody,omitempty"`
	Responses   map[int]*Response `json:"responses"`
}

// Parameter describes a single operation parameter (path, query, header, or cookie).
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"`
	Required    bool    `json:"required,omitempty"`
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema"`
}

// RequestBody describes a single request body.
type RequestBody struct {
	Description string                `json:"description,omitempty"`
	Required    bool                  `json:"required,omitempty"`
	Content     map[string]*MediaType `json:"content"`
}

// Response describes a single response from an API operation.
type Response struct {
	Description string                `json:"description"`
	Content     map[string]*MediaType `json:"content,omitempty"`
	Ref         string                `json:"$ref,omitempty"`
}

// MediaType provides schema and examples for a media type.
type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}

// Schema defines the structure of input and output data.
type Schema struct {
	Type        string               `json:"type,omitempty"`
	Format      string               `json:"format,omitempty"`
	Description string               `json:"description,omitempty"`
	Properties  map[string]*Property `json:"properties,omitempty"`
	Required    []string             `json:"required,omitempty"`
	Items       *Schema              `json:"items,omitempty"`
	Ref         string               `json:"$ref,omitempty"`
}

// Property defines a single property within a schema.
type Property struct {
	Type        string `json:"type"`
	Format      string `json:"format,omitempty"`
	Description string `json:"description,omitempty"`
	Example     any    `json:"example,omitempty"`
}

// Components holds reusable schema and response definitions.
type Components struct {
	Schemas   map[string]*Schema   `json:"schemas,omitempty"`
	Responses map[string]*Response `json:"responses,omitempty"`
}

// SchemaRef creates a JSON reference to a schema in components/schemas.
func SchemaRef(name string) *Schema {
	return &Schema{Ref: "#/components/schemas/" + name}
}

// ResponseRef creates a JSON reference to a response in components/responses.
func ResponseRef(name string) *Response {
	return &Response{Ref: "#/components/responses/" + name}
}

// RequestBodyJSON creates a request body with JSON content type referencing a schema.
func RequestBodyJSON(schemaName string, required bool) *RequestBody {
	return &RequestBody{
		Required: required,
		Content: map[string]*MediaType{
			"application/json": {Schema: SchemaRef(schemaName)},
		},
	}
}

// ResponseJSON creates a response with JSON content type referencing a schema.
func ResponseJSON(description, schemaName string) *Response {
	return &Response{
		Description: description,
		Content: map[string]*MediaType{
			"application/json": {Schema: SchemaRef(schemaName)},
		},
	}
}

// PathParam creates a required path parameter with UUID format.
func PathParam(name, description string) *Parameter {
	return &Parameter{
		Name:        name,
		In:          "path",
		Required:    true,
		Description: description,
		Schema:      &Schema{Type: "string", Format: "uuid"},
	}
}

// QueryParam creates a query parameter with the specified type.
func QueryParam(name, typ, description string, required bool) *Parameter {
	return &Parameter{
		Name:        name,
		In:          "query",
		Required:    required,
		Description: description,
		Schema:      &Schema{Type: typ},
	}
}
