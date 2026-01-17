package pkg_module_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/module"
)

func TestNew(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	m := module.New("/api", handler)

	if m == nil {
		t.Fatal("New() returned nil")
	}

	if m.Prefix() != "/api" {
		t.Errorf("Prefix() = %q, want %q", m.Prefix(), "/api")
	}
}

func TestNew_InvalidPrefix_Empty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("New() with empty prefix did not panic")
		}
	}()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	module.New("", handler)
}

func TestNew_InvalidPrefix_NoLeadingSlash(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("New() without leading slash did not panic")
		}
	}()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	module.New("api", handler)
}

func TestNew_InvalidPrefix_MultiLevel(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("New() with multi-level prefix did not panic")
		}
	}()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	module.New("/api/v1", handler)
}

func TestModule_Handler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})

	m := module.New("/api", handler)
	h := m.Handler()

	if h == nil {
		t.Fatal("Handler() returned nil")
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello" {
		t.Errorf("body = %q, want %q", string(body), "hello")
	}
}

func TestModule_Use(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("handler"))
	})

	m := module.New("/api", handler)

	m.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	m.Handler().ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.Header.Get("X-Middleware") != "applied" {
		t.Error("middleware was not applied")
	}
}

func TestModule_Serve(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.URL.Path))
	})

	m := module.New("/api", handler)

	tests := []struct {
		name     string
		path     string
		wantPath string
	}{
		{"root path", "/api", "/"},
		{"sub path", "/api/users", "/users"},
		{"nested path", "/api/users/123", "/users/123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			m.Serve(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			if string(body) != tt.wantPath {
				t.Errorf("path = %q, want %q", string(body), tt.wantPath)
			}
		})
	}
}

func TestModule_MiddlewareOrder(t *testing.T) {
	var order []string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	m := module.New("/api", handler)

	m.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "first")
			next.ServeHTTP(w, r)
		})
	})

	m.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "second")
			next.ServeHTTP(w, r)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	m.Handler().ServeHTTP(w, req)

	expected := []string{"first", "second", "handler"}
	if len(order) != len(expected) {
		t.Fatalf("order length = %d, want %d", len(order), len(expected))
	}

	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}
