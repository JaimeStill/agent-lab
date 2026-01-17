package pkg_middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/middleware"
)

func TestNew(t *testing.T) {
	mw := middleware.New()

	if mw == nil {
		t.Fatal("New() returned nil")
	}
}

func TestSystem_Apply_NoMiddleware(t *testing.T) {
	mw := middleware.New()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("handler"))
	})

	wrapped := mw.Apply(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestSystem_Use_SingleMiddleware(t *testing.T) {
	mw := middleware.New()

	mw.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test", "value")
			next.ServeHTTP(w, r)
		})
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Apply(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.Header.Get("X-Test") != "value" {
		t.Error("middleware was not applied")
	}
}

func TestSystem_Use_MiddlewareOrder(t *testing.T) {
	mw := middleware.New()
	var order []string

	mw.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "first-before")
			next.ServeHTTP(w, r)
			order = append(order, "first-after")
		})
	})

	mw.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "second-before")
			next.ServeHTTP(w, r)
			order = append(order, "second-after")
		})
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	wrapped := mw.Apply(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	expected := []string{"first-before", "second-before", "handler", "second-after", "first-after"}
	if len(order) != len(expected) {
		t.Fatalf("order length = %d, want %d", len(order), len(expected))
	}

	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}
