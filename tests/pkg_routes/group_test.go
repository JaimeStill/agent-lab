package pkg_routes_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/openapi"
	"github.com/JaimeStill/agent-lab/pkg/routes"
)

func TestGroup_AddToSpec(t *testing.T) {
	spec := openapi.NewSpec("Test API", "1.0.0")

	group := routes.Group{
		Prefix: "/users",
		Tags:   []string{"Users"},
		Routes: []routes.Route{
			{
				Method:  "GET",
				Pattern: "",
				Handler: func(w http.ResponseWriter, r *http.Request) {},
				OpenAPI: &openapi.Operation{
					Summary: "List users",
				},
			},
			{
				Method:  "POST",
				Pattern: "",
				Handler: func(w http.ResponseWriter, r *http.Request) {},
				OpenAPI: &openapi.Operation{
					Summary: "Create user",
				},
			},
		},
	}

	group.AddToSpec("/api", spec)

	if spec.Paths["/api/users"] == nil {
		t.Fatal("path /api/users not added to spec")
	}

	if spec.Paths["/api/users"].Get == nil {
		t.Error("GET operation not added")
	}

	if spec.Paths["/api/users"].Post == nil {
		t.Error("POST operation not added")
	}

	if spec.Paths["/api/users"].Get.Summary != "List users" {
		t.Errorf("GET summary = %q, want %q", spec.Paths["/api/users"].Get.Summary, "List users")
	}
}

func TestGroup_AddToSpec_InheritsTags(t *testing.T) {
	spec := openapi.NewSpec("Test API", "1.0.0")

	group := routes.Group{
		Prefix: "/users",
		Tags:   []string{"Users"},
		Routes: []routes.Route{
			{
				Method:  "GET",
				Pattern: "",
				Handler: func(w http.ResponseWriter, r *http.Request) {},
				OpenAPI: &openapi.Operation{
					Summary: "List users",
				},
			},
		},
	}

	group.AddToSpec("", spec)

	if len(spec.Paths["/users"].Get.Tags) != 1 || spec.Paths["/users"].Get.Tags[0] != "Users" {
		t.Errorf("Tags = %v, want [Users]", spec.Paths["/users"].Get.Tags)
	}
}

func TestGroup_AddToSpec_PreservesExplicitTags(t *testing.T) {
	spec := openapi.NewSpec("Test API", "1.0.0")

	group := routes.Group{
		Prefix: "/users",
		Tags:   []string{"Users"},
		Routes: []routes.Route{
			{
				Method:  "GET",
				Pattern: "",
				Handler: func(w http.ResponseWriter, r *http.Request) {},
				OpenAPI: &openapi.Operation{
					Summary: "List users",
					Tags:    []string{"Admin"},
				},
			},
		},
	}

	group.AddToSpec("", spec)

	if len(spec.Paths["/users"].Get.Tags) != 1 || spec.Paths["/users"].Get.Tags[0] != "Admin" {
		t.Errorf("Tags = %v, want [Admin]", spec.Paths["/users"].Get.Tags)
	}
}

func TestGroup_AddToSpec_Children(t *testing.T) {
	spec := openapi.NewSpec("Test API", "1.0.0")

	group := routes.Group{
		Prefix: "/users",
		Tags:   []string{"Users"},
		Routes: []routes.Route{
			{
				Method:  "GET",
				Pattern: "",
				Handler: func(w http.ResponseWriter, r *http.Request) {},
				OpenAPI: &openapi.Operation{Summary: "List users"},
			},
		},
		Children: []routes.Group{
			{
				Prefix: "/posts",
				Tags:   []string{"Posts"},
				Routes: []routes.Route{
					{
						Method:  "GET",
						Pattern: "",
						Handler: func(w http.ResponseWriter, r *http.Request) {},
						OpenAPI: &openapi.Operation{Summary: "List user posts"},
					},
				},
			},
		},
	}

	group.AddToSpec("/api", spec)

	if spec.Paths["/api/users"] == nil {
		t.Error("parent path not added")
	}

	if spec.Paths["/api/users/posts"] == nil {
		t.Error("child path not added")
	}

	if spec.Paths["/api/users/posts"].Get.Summary != "List user posts" {
		t.Errorf("child summary = %q, want %q",
			spec.Paths["/api/users/posts"].Get.Summary, "List user posts")
	}
}

func TestGroup_AddToSpec_Schemas(t *testing.T) {
	spec := openapi.NewSpec("Test API", "1.0.0")

	group := routes.Group{
		Prefix: "/users",
		Schemas: map[string]*openapi.Schema{
			"User": {
				Type: "object",
				Properties: map[string]*openapi.Schema{
					"id":   {Type: "integer"},
					"name": {Type: "string"},
				},
			},
		},
	}

	group.AddToSpec("", spec)

	if spec.Components.Schemas["User"] == nil {
		t.Error("schema not added to spec")
	}

	if spec.Components.Schemas["User"].Type != "object" {
		t.Errorf("schema type = %q, want %q", spec.Components.Schemas["User"].Type, "object")
	}
}

func TestGroup_AddToSpec_SkipsNilOpenAPI(t *testing.T) {
	spec := openapi.NewSpec("Test API", "1.0.0")

	group := routes.Group{
		Prefix: "/users",
		Routes: []routes.Route{
			{
				Method:  "GET",
				Pattern: "",
				Handler: func(w http.ResponseWriter, r *http.Request) {},
				OpenAPI: nil,
			},
		},
	}

	group.AddToSpec("", spec)

	if spec.Paths["/users"] != nil {
		t.Error("path should not be added for route without OpenAPI")
	}
}

func TestGroup_AddToSpec_AllMethods(t *testing.T) {
	spec := openapi.NewSpec("Test API", "1.0.0")

	group := routes.Group{
		Prefix: "/resource",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: func(w http.ResponseWriter, r *http.Request) {}, OpenAPI: &openapi.Operation{Summary: "Get"}},
			{Method: "POST", Pattern: "", Handler: func(w http.ResponseWriter, r *http.Request) {}, OpenAPI: &openapi.Operation{Summary: "Create"}},
			{Method: "PUT", Pattern: "", Handler: func(w http.ResponseWriter, r *http.Request) {}, OpenAPI: &openapi.Operation{Summary: "Update"}},
			{Method: "DELETE", Pattern: "", Handler: func(w http.ResponseWriter, r *http.Request) {}, OpenAPI: &openapi.Operation{Summary: "Delete"}},
		},
	}

	group.AddToSpec("", spec)

	pathItem := spec.Paths["/resource"]
	if pathItem == nil {
		t.Fatal("path not added")
	}

	if pathItem.Get == nil || pathItem.Get.Summary != "Get" {
		t.Error("GET operation incorrect")
	}

	if pathItem.Post == nil || pathItem.Post.Summary != "Create" {
		t.Error("POST operation incorrect")
	}

	if pathItem.Put == nil || pathItem.Put.Summary != "Update" {
		t.Error("PUT operation incorrect")
	}

	if pathItem.Delete == nil || pathItem.Delete.Summary != "Delete" {
		t.Error("DELETE operation incorrect")
	}
}

func TestRegister(t *testing.T) {
	mux := http.NewServeMux()
	spec := openapi.NewSpec("Test API", "1.0.0")

	group := routes.Group{
		Prefix: "/users",
		Routes: []routes.Route{
			{
				Method: "GET",
				Pattern: "",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("users list"))
				},
				OpenAPI: &openapi.Operation{Summary: "List users"},
			},
			{
				Method: "GET",
				Pattern: "/{id}",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("user detail"))
				},
				OpenAPI: &openapi.Operation{Summary: "Get user"},
			},
		},
	}

	routes.Register(mux, "/api", spec, group)

	tests := []struct {
		path string
		want string
	}{
		{"/users", "users list"},
		{"/users/123", "user detail"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			if string(body) != tt.want {
				t.Errorf("body = %q, want %q", string(body), tt.want)
			}
		})
	}

	if spec.Paths["/api/users"] == nil {
		t.Error("spec path /api/users not added")
	}

	if spec.Paths["/api/users/{id}"] == nil {
		t.Error("spec path /api/users/{id} not added")
	}
}

func TestRegister_MultipleGroups(t *testing.T) {
	mux := http.NewServeMux()
	spec := openapi.NewSpec("Test API", "1.0.0")

	usersGroup := routes.Group{
		Prefix: "/users",
		Routes: []routes.Route{
			{
				Method: "GET",
				Pattern: "",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("users"))
				},
				OpenAPI: &openapi.Operation{Summary: "List users"},
			},
		},
	}

	postsGroup := routes.Group{
		Prefix: "/posts",
		Routes: []routes.Route{
			{
				Method: "GET",
				Pattern: "",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("posts"))
				},
				OpenAPI: &openapi.Operation{Summary: "List posts"},
			},
		},
	}

	routes.Register(mux, "/api", spec, usersGroup, postsGroup)

	tests := []struct {
		path string
		want string
	}{
		{"/users", "users"},
		{"/posts", "posts"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			if string(body) != tt.want {
				t.Errorf("body = %q, want %q", string(body), tt.want)
			}
		})
	}

	if spec.Paths["/api/users"] == nil {
		t.Error("users spec path not added")
	}

	if spec.Paths["/api/posts"] == nil {
		t.Error("posts spec path not added")
	}
}

func TestRegister_NestedChildren(t *testing.T) {
	mux := http.NewServeMux()
	spec := openapi.NewSpec("Test API", "1.0.0")

	group := routes.Group{
		Prefix: "/workflows",
		Routes: []routes.Route{
			{
				Method: "GET",
				Pattern: "",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("workflows"))
				},
				OpenAPI: &openapi.Operation{Summary: "List workflows"},
			},
		},
		Children: []routes.Group{
			{
				Prefix: "/runs",
				Routes: []routes.Route{
					{
						Method: "GET",
						Pattern: "",
						Handler: func(w http.ResponseWriter, r *http.Request) {
							w.Write([]byte("runs"))
						},
						OpenAPI: &openapi.Operation{Summary: "List runs"},
					},
				},
			},
		},
	}

	routes.Register(mux, "/api", spec, group)

	tests := []struct {
		path string
		want string
	}{
		{"/workflows", "workflows"},
		{"/workflows/runs", "runs"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			if string(body) != tt.want {
				t.Errorf("body = %q, want %q", string(body), tt.want)
			}
		})
	}

	if spec.Paths["/api/workflows/runs"] == nil {
		t.Error("nested spec path not added")
	}
}
