package internal_routes_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/routes"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
}

func TestNew(t *testing.T) {
	sys := routes.New(testLogger())
	if sys == nil {
		t.Fatal("New() returned nil")
	}
}

func TestRegisterRoute(t *testing.T) {
	sys := routes.New(testLogger())

	route := routes.Route{
		Method:  "GET",
		Pattern: "/test",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test response"))
		},
	}

	sys.RegisterRoute(route)

	handler := sys.Build()
	if handler == nil {
		t.Fatal("Build() returned nil handler")
	}

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	expected := "test response"
	if rec.Body.String() != expected {
		t.Errorf("Expected body %q, got %q", expected, rec.Body.String())
	}
}

func TestRegisterRoute_MultipleMethods(t *testing.T) {
	tests := []struct {
		name   string
		method string
	}{
		{"GET", "GET"},
		{"POST", "POST"},
		{"PUT", "PUT"},
		{"DELETE", "DELETE"},
		{"PATCH", "PATCH"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sys := routes.New(testLogger())

			route := routes.Route{
				Method:  tt.method,
				Pattern: "/test",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
			}

			sys.RegisterRoute(route)
			handler := sys.Build()

			req := httptest.NewRequest(tt.method, "/test", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("Expected status %d for %s, got %d", http.StatusOK, tt.method, rec.Code)
			}
		})
	}
}

func TestRegisterRoute_MultipleRoutes(t *testing.T) {
	sys := routes.New(testLogger())

	route1 := routes.Route{
		Method:  "GET",
		Pattern: "/route1",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("route1"))
		},
	}

	route2 := routes.Route{
		Method:  "POST",
		Pattern: "/route2",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("route2"))
		},
	}

	sys.RegisterRoute(route1)
	sys.RegisterRoute(route2)

	handler := sys.Build()

	req1 := httptest.NewRequest("GET", "/route1", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Body.String() != "route1" {
		t.Errorf("Expected route1 response, got %q", rec1.Body.String())
	}

	req2 := httptest.NewRequest("POST", "/route2", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Body.String() != "route2" {
		t.Errorf("Expected route2 response, got %q", rec2.Body.String())
	}
}

func TestBuild_EmptySystem(t *testing.T) {
	sys := routes.New(testLogger())
	handler := sys.Build()

	if handler == nil {
		t.Fatal("Build() should return non-nil handler even with no routes")
	}

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status %d for nonexistent route, got %d", http.StatusNotFound, rec.Code)
	}
}
