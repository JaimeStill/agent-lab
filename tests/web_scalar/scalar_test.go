package web_scalar_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/routes"
	"github.com/JaimeStill/agent-lab/web/scalar"
)

func TestMount(t *testing.T) {
	logger := slog.Default()
	r := routes.New(logger)

	scalar.Mount(r, "/scalar")

	registered := r.Routes()
	if len(registered) != 2 {
		t.Errorf("Routes count = %d, want 2", len(registered))
	}

	patterns := make(map[string]bool)
	for _, route := range registered {
		patterns[route.Pattern] = true
		if route.Method != "GET" {
			t.Errorf("route %s: Method = %q, want GET", route.Pattern, route.Method)
		}
		if route.Handler == nil {
			t.Errorf("route %s: Handler is nil", route.Pattern)
		}
	}

	if !patterns["/scalar"] {
		t.Error("missing exact match route /scalar")
	}
	if !patterns["/scalar/{path...}"] {
		t.Error("missing wildcard route /scalar/{path...}")
	}
}

func TestServeIndex(t *testing.T) {
	logger := slog.Default()
	r := routes.New(logger)
	scalar.Mount(r, "/scalar")
	handler := r.Build()

	// Test with trailing slash (how browsers typically access after redirect)
	req := httptest.NewRequest(http.MethodGet, "/scalar/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", contentType)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "<!DOCTYPE html>") {
		t.Error("response body does not contain DOCTYPE")
	}
}

func TestServeAssets(t *testing.T) {
	logger := slog.Default()
	r := routes.New(logger)
	scalar.Mount(r, "/scalar")
	handler := r.Build()

	tests := []struct {
		path        string
		contentType string
	}{
		{"/scalar/scalar.js", "text/javascript"},
		{"/scalar/scalar.css", "text/css"},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
			}

			body, _ := io.ReadAll(resp.Body)
			if len(body) == 0 {
				t.Error("response body is empty")
			}
		})
	}
}
