# OpenAPI 3.1 Integration Guide

This guide documents how agent-lab implements OpenAPI 3.1 specification for automatic API documentation generation.

## OpenAPI 3.1 Specification Alignment

OpenAPI 3.1 aligns with JSON Schema Draft 2020-12. The key insight is that **Schema Objects are used everywhere** - properties within an object schema are themselves Schema Objects, not a separate type.

### Schema Object Structure

Per OpenAPI 3.1, a Schema Object follows JSON Schema and includes:

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Data type (string, number, integer, boolean, array, object) |
| `format` | string | Extended format (uuid, date-time, int64, binary, etc.) |
| `description` | string | Human-readable description |
| `properties` | map[string]*Schema | Object properties (each is a Schema) |
| `required` | []string | Required property names |
| `items` | *Schema | Array item schema |
| `$ref` | string | JSON Reference to another schema |
| `example` | any | Example value |
| `default` | any | Default value |
| `enum` | []any | Enumerated values |
| `minimum/maximum` | *float64 | Numeric constraints |
| `minLength/maxLength` | *int | String length constraints |
| `pattern` | string | Regex pattern for strings |

## Implementation Architecture

### Package Structure

```
pkg/openapi/
├── types.go       # Core OpenAPI types (Spec, Schema, Operation, etc.)
├── components.go  # Shared components (PageRequest, error responses)

internal/{domain}/
└── openapi.go     # Domain-specific operations and schemas
```

### Core Types (`pkg/openapi/types.go`)

```go
type Schema struct {
    Type        string             `json:"type,omitempty"`
    Format      string             `json:"format,omitempty"`
    Description string             `json:"description,omitempty"`
    Properties  map[string]*Schema `json:"properties,omitempty"`
    Required    []string           `json:"required,omitempty"`
    Items       *Schema            `json:"items,omitempty"`
    Ref         string             `json:"$ref,omitempty"`
    Example     any                `json:"example,omitempty"`
    // ... additional JSON Schema fields
}
```

### Helper Functions

| Function | Purpose |
|----------|---------|
| `SchemaRef(name)` | Creates `$ref` to `#/components/schemas/{name}` |
| `ResponseRef(name)` | Creates `$ref` to `#/components/responses/{name}` |
| `RequestBodyJSON(schema, required)` | Creates JSON request body referencing a schema |
| `ResponseJSON(desc, schema)` | Creates JSON response referencing a schema |
| `PathParam(name, desc)` | Creates required UUID path parameter |
| `QueryParam(name, type, desc, required)` | Creates query parameter |

## Domain OpenAPI Pattern

Each domain implements an `openapi.go` file with:

### 1. Spec Variable

```go
type spec struct {
    Create *openapi.Operation
    List   *openapi.Operation
    Get    *openapi.Operation
    // ... operations
}

var Spec = spec{
    Create: &openapi.Operation{
        Summary:     "Create resource",
        Description: "Detailed description",
        RequestBody: openapi.RequestBodyJSON("CreateCommand", true),
        Responses: map[int]*openapi.Response{
            201: openapi.ResponseJSON("Resource created", "Resource"),
            400: openapi.ResponseRef("BadRequest"),
        },
    },
    // ... other operations
}
```

### 2. Schemas Method

```go
func (spec) Schemas() map[string]*openapi.Schema {
    return map[string]*openapi.Schema{
        "Resource": {
            Type: "object",
            Properties: map[string]*openapi.Schema{
                "id":         {Type: "string", Format: "uuid"},
                "name":       {Type: "string"},
                "created_at": {Type: "string", Format: "date-time"},
            },
        },
        "CreateCommand": {
            Type:     "object",
            Required: []string{"name"},
            Properties: map[string]*openapi.Schema{
                "name": {Type: "string", Example: "example"},
            },
        },
    }
}
```

### 3. Route Linkage

In `handler.go`, each route references its OpenAPI operation:

```go
func (h *Handler) Routes() routes.Group {
    return routes.Group{
        Prefix: "/api/resources",
        Tags:   []string{"Resources"},
        Routes: []routes.Route{
            {Method: "POST", Pattern: "", Handler: h.Create, OpenAPI: Spec.Create},
            {Method: "GET", Pattern: "/{id}", Handler: h.GetByID, OpenAPI: Spec.Get},
        },
    }
}
```

## Shared Components

`pkg/openapi/components.go` provides reusable definitions:

### Schemas
- **PageRequest**: Standard pagination request (page, page_size, search, sort)

### Responses
- **BadRequest**: 400 error with message
- **NotFound**: 404 error with message
- **Conflict**: 409 error with message (duplicate resources)

## Server Integration

In `cmd/server/routes.go`:

```go
components := openapi.NewComponents()
components.AddSchemas(providers.Spec.Schemas())
components.AddSchemas(agents.Spec.Schemas())
components.AddSchemas(documents.Spec.Schemas())
```

## Common Patterns

### Array with Item Reference

```go
"PageResult": {
    Type: "object",
    Properties: map[string]*openapi.Schema{
        "data": {Type: "array", Items: openapi.SchemaRef("Resource")},
        "total": {Type: "integer"},
    },
}
```

### Multipart Form Data

```go
RequestBody: &openapi.RequestBody{
    Required: true,
    Content: map[string]*openapi.MediaType{
        "multipart/form-data": {
            Schema: &openapi.Schema{
                Type: "object",
                Properties: map[string]*openapi.Schema{
                    "file": {Type: "string", Description: "File to upload"},
                    "name": {Type: "string"},
                },
                Required: []string{"file"},
            },
        },
    },
}
```

**Important**: For binary file upload fields, only use `Type` and `Description`. Do NOT include `Format: "binary"` - this causes Scalar UI to display "BINARY" as a placeholder instead of rendering a file upload button.

### Inline vs Referenced Schemas

Use **references** for:
- Reusable domain entities (Resource, CreateCommand)
- Shared response schemas
- Pagination results

Use **inline schemas** for:
- Multipart form definitions (specific to single endpoint)
- Simple one-off response structures

## Validation

After API changes:

1. Start server - spec auto-generates to `api/openapi.{env}.json`
2. Visit `/docs` to verify Scalar UI renders correctly
3. Test "Try It" functionality for new endpoints
4. Verify schema references resolve correctly
