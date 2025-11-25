package internal_middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/middleware"
)

func TestCORS_Disabled(t *testing.T) {
	cfg := &config.CORSConfig{
		Enabled: false,
		Origins: []string{"http://example.com"},
	}

	corsMiddleware := middleware.CORS(cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := corsMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS headers should not be set when disabled")
	}
}

func TestCORS_NoOrigins(t *testing.T) {
	cfg := &config.CORSConfig{
		Enabled: true,
		Origins: []string{},
	}

	corsMiddleware := middleware.CORS(cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := corsMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS headers should not be set when no origins configured")
	}
}

func TestCORS_AllowedOrigin(t *testing.T) {
	cfg := &config.CORSConfig{
		Enabled:        true,
		Origins:        []string{"http://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
	}

	corsMiddleware := middleware.CORS(cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := corsMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("Expected Access-Control-Allow-Origin %q, got %q", "http://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	}

	if rec.Header().Get("Access-Control-Allow-Methods") != "GET, POST" {
		t.Errorf("Expected Access-Control-Allow-Methods %q, got %q", "GET, POST", rec.Header().Get("Access-Control-Allow-Methods"))
	}

	if rec.Header().Get("Access-Control-Allow-Headers") != "Content-Type" {
		t.Errorf("Expected Access-Control-Allow-Headers %q, got %q", "Content-Type", rec.Header().Get("Access-Control-Allow-Headers"))
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	cfg := &config.CORSConfig{
		Enabled: true,
		Origins: []string{"http://example.com"},
	}

	corsMiddleware := middleware.CORS(cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := corsMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://malicious.com")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS headers should not be set for disallowed origin")
	}
}

func TestCORS_AllowCredentials(t *testing.T) {
	cfg := &config.CORSConfig{
		Enabled:          true,
		Origins:          []string{"http://example.com"},
		AllowedMethods:   []string{"GET"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}

	corsMiddleware := middleware.CORS(cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := corsMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("Expected Access-Control-Allow-Credentials to be set to true")
	}
}

func TestCORS_MaxAge(t *testing.T) {
	cfg := &config.CORSConfig{
		Enabled:        true,
		Origins:        []string{"http://example.com"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
		MaxAge:         3600,
	}

	corsMiddleware := middleware.CORS(cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := corsMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Max-Age") != "3600" {
		t.Errorf("Expected Access-Control-Max-Age %q, got %q", "3600", rec.Header().Get("Access-Control-Max-Age"))
	}
}

func TestCORS_OptionsRequest(t *testing.T) {
	cfg := &config.CORSConfig{
		Enabled:        true,
		Origins:        []string{"http://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
	}

	corsMiddleware := middleware.CORS(cfg)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := corsMiddleware(handler)

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if handlerCalled {
		t.Error("Handler should not be called for OPTIONS preflight request")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d for OPTIONS request, got %d", http.StatusOK, rec.Code)
	}

	if rec.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Error("CORS headers should be set for OPTIONS request")
	}
}

func TestCORS_MultipleOrigins(t *testing.T) {
	cfg := &config.CORSConfig{
		Enabled:        true,
		Origins:        []string{"http://example.com", "http://localhost:3000", "https://app.example.com"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
	}

	corsMiddleware := middleware.CORS(cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := corsMiddleware(handler)

	tests := []struct {
		origin  string
		allowed bool
	}{
		{"http://example.com", true},
		{"http://localhost:3000", true},
		{"https://app.example.com", true},
		{"http://malicious.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", tt.origin)
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)

			corsHeader := rec.Header().Get("Access-Control-Allow-Origin")

			if tt.allowed {
				if corsHeader != tt.origin {
					t.Errorf("Expected Access-Control-Allow-Origin %q, got %q", tt.origin, corsHeader)
				}
			} else {
				if corsHeader != "" {
					t.Errorf("Expected no CORS headers for %q, but got %q", tt.origin, corsHeader)
				}
			}
		})
	}
}
