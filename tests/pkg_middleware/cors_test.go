package pkg_middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/middleware"
)

func TestCORS_Disabled(t *testing.T) {
	cfg := &middleware.CORSConfig{
		Enabled: false,
		Origins: []string{"http://localhost:3000"},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.CORS(cfg)(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS headers should not be set when disabled")
	}
}

func TestCORS_NoOrigins(t *testing.T) {
	cfg := &middleware.CORSConfig{
		Enabled: true,
		Origins: []string{},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.CORS(cfg)(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS headers should not be set with no origins configured")
	}
}

func TestCORS_AllowedOrigin(t *testing.T) {
	cfg := &middleware.CORSConfig{
		Enabled:        true,
		Origins:        []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.CORS(cfg)(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.Header.Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q",
			resp.Header.Get("Access-Control-Allow-Origin"), "http://localhost:3000")
	}

	if resp.Header.Get("Access-Control-Allow-Methods") != "GET, POST" {
		t.Errorf("Access-Control-Allow-Methods = %q, want %q",
			resp.Header.Get("Access-Control-Allow-Methods"), "GET, POST")
	}

	if resp.Header.Get("Access-Control-Allow-Headers") != "Content-Type" {
		t.Errorf("Access-Control-Allow-Headers = %q, want %q",
			resp.Header.Get("Access-Control-Allow-Headers"), "Content-Type")
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	cfg := &middleware.CORSConfig{
		Enabled: true,
		Origins: []string{"http://localhost:3000"},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.CORS(cfg)(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://evil.com")
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS headers should not be set for disallowed origin")
	}
}

func TestCORS_Credentials(t *testing.T) {
	cfg := &middleware.CORSConfig{
		Enabled:          true,
		Origins:          []string{"http://localhost:3000"},
		AllowCredentials: true,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.CORS(cfg)(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.Header.Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("Access-Control-Allow-Credentials should be set")
	}
}

func TestCORS_MaxAge(t *testing.T) {
	cfg := &middleware.CORSConfig{
		Enabled: true,
		Origins: []string{"http://localhost:3000"},
		MaxAge:  7200,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.CORS(cfg)(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.Header.Get("Access-Control-Max-Age") != "7200" {
		t.Errorf("Access-Control-Max-Age = %q, want %q",
			resp.Header.Get("Access-Control-Max-Age"), "7200")
	}
}

func TestCORS_Preflight(t *testing.T) {
	cfg := &middleware.CORSConfig{
		Enabled: true,
		Origins: []string{"http://localhost:3000"},
	}

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.CORS(cfg)(handler)

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if handlerCalled {
		t.Error("handler should not be called for OPTIONS preflight")
	}
}

func TestCORS_MultipleOrigins(t *testing.T) {
	cfg := &middleware.CORSConfig{
		Enabled: true,
		Origins: []string{"http://localhost:3000", "http://localhost:8080"},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.CORS(cfg)(handler)

	tests := []struct {
		origin string
		want   string
	}{
		{"http://localhost:3000", "http://localhost:3000"},
		{"http://localhost:8080", "http://localhost:8080"},
		{"http://evil.com", ""},
	}

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()

			wrapped.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			got := resp.Header.Get("Access-Control-Allow-Origin")
			if got != tt.want {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tt.want)
			}
		})
	}
}
