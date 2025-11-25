package internal_middleware_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/middleware"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
}

func TestLogger_AppliesMiddleware(t *testing.T) {
	logger := testLogger()
	loggerMiddleware := middleware.Logger(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	wrapped := loggerMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Body.String() != "test response" {
		t.Errorf("Expected body %q, got %q", "test response", rec.Body.String())
	}
}

func TestLogger_DoesNotInterfere(t *testing.T) {
	logger := testLogger()
	loggerMiddleware := middleware.Logger(logger)

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("created"))
	})

	wrapped := loggerMiddleware(handler)

	req := httptest.NewRequest("POST", "/create", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if !called {
		t.Error("Handler was not called")
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	if rec.Body.String() != "created" {
		t.Errorf("Expected body %q, got %q", "created", rec.Body.String())
	}
}

func TestLogger_AllHTTPMethods(t *testing.T) {
	logger := testLogger()
	loggerMiddleware := middleware.Logger(logger)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			wrapped := loggerMiddleware(handler)

			req := httptest.NewRequest(method, "/test", nil)
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("Expected status %d for method %s, got %d", http.StatusOK, method, rec.Code)
			}
		})
	}
}
