package internal_routes_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/routes"
)

func TestRegisterGroup(t *testing.T) {
	sys := routes.New(testLogger())

	group := routes.Group{
		Prefix:      "/api",
		Description: "API routes",
		Tags:        []string{"api"},
		Routes: []routes.Route{
			{
				Method:  "GET",
				Pattern: "/users",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("users"))
				},
			},
		},
	}

	sys.RegisterGroup(group)
	handler := sys.Build()

	req := httptest.NewRequest("GET", "/api/users", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Body.String() != "users" {
		t.Errorf("Expected body %q, got %q", "users", rec.Body.String())
	}
}

func TestRegisterGroup_MultipleRoutes(t *testing.T) {
	sys := routes.New(testLogger())

	group := routes.Group{
		Prefix: "/api",
		Routes: []routes.Route{
			{
				Method:  "GET",
				Pattern: "/users",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("get users"))
				},
			},
			{
				Method:  "POST",
				Pattern: "/users",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("create user"))
				},
			},
			{
				Method:  "GET",
				Pattern: "/posts",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("get posts"))
				},
			},
		},
	}

	sys.RegisterGroup(group)
	handler := sys.Build()

	tests := []struct {
		method   string
		path     string
		expected string
	}{
		{"GET", "/api/users", "get users"},
		{"POST", "/api/users", "create user"},
		{"GET", "/api/posts", "get posts"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Body.String() != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, rec.Body.String())
			}
		})
	}
}

func TestRegisterGroup_MultipleGroups(t *testing.T) {
	sys := routes.New(testLogger())

	apiGroup := routes.Group{
		Prefix: "/api",
		Routes: []routes.Route{
			{
				Method:  "GET",
				Pattern: "/users",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("api users"))
				},
			},
		},
	}

	adminGroup := routes.Group{
		Prefix: "/admin",
		Routes: []routes.Route{
			{
				Method:  "GET",
				Pattern: "/users",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("admin users"))
				},
			},
		},
	}

	sys.RegisterGroup(apiGroup)
	sys.RegisterGroup(adminGroup)
	handler := sys.Build()

	req1 := httptest.NewRequest("GET", "/api/users", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Body.String() != "api users" {
		t.Errorf("Expected %q, got %q", "api users", rec1.Body.String())
	}

	req2 := httptest.NewRequest("GET", "/admin/users", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Body.String() != "admin users" {
		t.Errorf("Expected %q, got %q", "admin users", rec2.Body.String())
	}
}

func TestRegisterGroup_EmptyPrefix(t *testing.T) {
	sys := routes.New(testLogger())

	group := routes.Group{
		Prefix: "",
		Routes: []routes.Route{
			{
				Method:  "GET",
				Pattern: "/root",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("root route"))
				},
			},
		},
	}

	sys.RegisterGroup(group)
	handler := sys.Build()

	req := httptest.NewRequest("GET", "/root", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "root route" {
		t.Errorf("Expected %q, got %q", "root route", rec.Body.String())
	}
}

func TestRegisterGroup_NestedPaths(t *testing.T) {
	sys := routes.New(testLogger())

	group := routes.Group{
		Prefix: "/api/v1",
		Routes: []routes.Route{
			{
				Method:  "GET",
				Pattern: "/users/{id}",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("user by id"))
				},
			},
		},
	}

	sys.RegisterGroup(group)
	handler := sys.Build()

	req := httptest.NewRequest("GET", "/api/v1/users/123", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "user by id" {
		t.Errorf("Expected %q, got %q", "user by id", rec.Body.String())
	}
}

func TestMixedRoutesAndGroups(t *testing.T) {
	sys := routes.New(testLogger())

	standaloneRoute := routes.Route{
		Method:  "GET",
		Pattern: "/health",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("healthy"))
		},
	}

	apiGroup := routes.Group{
		Prefix: "/api",
		Routes: []routes.Route{
			{
				Method:  "GET",
				Pattern: "/users",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("users"))
				},
			},
		},
	}

	sys.RegisterRoute(standaloneRoute)
	sys.RegisterGroup(apiGroup)
	handler := sys.Build()

	req1 := httptest.NewRequest("GET", "/health", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Body.String() != "healthy" {
		t.Errorf("Expected %q, got %q", "healthy", rec1.Body.String())
	}

	req2 := httptest.NewRequest("GET", "/api/users", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Body.String() != "users" {
		t.Errorf("Expected %q, got %q", "users", rec2.Body.String())
	}
}
