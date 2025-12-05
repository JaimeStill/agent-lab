package pkg_openapi_test

import (
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/openapi"
)

func TestNewComponents(t *testing.T) {
	components := openapi.NewComponents()

	if components.Schemas == nil {
		t.Fatal("Schemas map is nil")
	}

	if components.Responses == nil {
		t.Fatal("Responses map is nil")
	}

	requiredSchemas := []string{"PageRequest"}
	for _, name := range requiredSchemas {
		if _, ok := components.Schemas[name]; !ok {
			t.Errorf("missing required schema: %s", name)
		}
	}

	requiredResponses := []string{"BadRequest", "NotFound", "Conflict"}
	for _, name := range requiredResponses {
		if _, ok := components.Responses[name]; !ok {
			t.Errorf("missing required response: %s", name)
		}
	}
}

func TestAddSchemas(t *testing.T) {
	components := openapi.NewComponents()

	newSchemas := map[string]*openapi.Schema{
		"User": {
			Type: "object",
			Properties: map[string]*openapi.Property{
				"id":   {Type: "string", Format: "uuid"},
				"name": {Type: "string"},
			},
		},
		"Product": {
			Type: "object",
			Properties: map[string]*openapi.Property{
				"sku":   {Type: "string"},
				"price": {Type: "number"},
			},
		},
	}

	components.AddSchemas(newSchemas)

	for name := range newSchemas {
		if _, ok := components.Schemas[name]; !ok {
			t.Errorf("schema %q not added", name)
		}
	}

	if _, ok := components.Schemas["PageRequest"]; !ok {
		t.Error("original PageRequest schema was overwritten")
	}
}

func TestAddResponses(t *testing.T) {
	components := openapi.NewComponents()

	newResponses := map[string]*openapi.Response{
		"Unauthorized": {
			Description: "Authentication required",
		},
		"Forbidden": {
			Description: "Access denied",
		},
	}

	components.AddResponses(newResponses)

	for name := range newResponses {
		if _, ok := components.Responses[name]; !ok {
			t.Errorf("response %q not added", name)
		}
	}

	if _, ok := components.Responses["BadRequest"]; !ok {
		t.Error("original BadRequest response was overwritten")
	}
}
