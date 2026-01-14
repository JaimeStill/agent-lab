package web_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JaimeStill/agent-lab/web"
)

func TestStatic(t *testing.T) {
	handler := web.Static()
	if handler == nil {
		t.Fatal("Static() returned nil")
	}
}

func TestStaticServesJS(t *testing.T) {
	handler := web.Static()

	req := httptest.NewRequest(http.MethodGet, "/static/docs.js", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Error("response body is empty")
	}
}

func TestStaticServesCSS(t *testing.T) {
	handler := web.Static()

	req := httptest.NewRequest(http.MethodGet, "/static/docs.css", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/css") {
		t.Errorf("Content-Type = %q, want text/css", contentType)
	}
}

func TestStaticNotFound(t *testing.T) {
	handler := web.Static()

	req := httptest.NewRequest(http.MethodGet, "/static/nonexistent.js", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestNewHandler(t *testing.T) {
	h, err := web.NewHandler()
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	if h == nil {
		t.Fatal("NewHandler() returned nil")
	}
}

func TestHandlerRoutes(t *testing.T) {
	h, err := web.NewHandler()
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	routes := h.Routes()
	if routes.Prefix != "/app" {
		t.Errorf("Prefix = %q, want %q", routes.Prefix, "/app")
	}
	if len(routes.Routes) == 0 {
		t.Error("Routes() returned empty routes")
	}
}

func TestHandlerServesHome(t *testing.T) {
	h, err := web.NewHandler()
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	routes := h.Routes()

	var handler http.HandlerFunc
	for _, route := range routes.Routes {
		if route.Pattern == "" {
			handler = route.Handler
			break
		}
	}

	if handler == nil {
		t.Fatal("home handler not found")
	}

	req := httptest.NewRequest(http.MethodGet, "/app", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", contentType, "text/html; charset=utf-8")
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "<!DOCTYPE html>") {
		t.Error("response body does not contain DOCTYPE")
	}
}

func TestHandlerServesComponents(t *testing.T) {
	h, err := web.NewHandler()
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	routes := h.Routes()

	var handler http.HandlerFunc
	for _, route := range routes.Routes {
		if route.Pattern == "/components" {
			handler = route.Handler
			break
		}
	}

	if handler == nil {
		t.Fatal("components handler not found")
	}

	req := httptest.NewRequest(http.MethodGet, "/app/components", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", contentType, "text/html; charset=utf-8")
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Component Styles") {
		t.Error("response body does not contain expected content")
	}
}

func TestHandlerServesPublicFiles(t *testing.T) {
	h, err := web.NewHandler()
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	routes := h.Routes()

	var handler http.HandlerFunc
	for _, route := range routes.Routes {
		if route.Pattern == "/favicon.ico" {
			handler = route.Handler
			break
		}
	}

	if handler == nil {
		t.Fatal("favicon handler not found")
	}

	req := httptest.NewRequest(http.MethodGet, "/app/favicon.ico", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Error("favicon response body is empty")
	}
}

func TestHandlerRoutesComplete(t *testing.T) {
	h, err := web.NewHandler()
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	routes := h.Routes()

	expectedPatterns := map[string]string{
		"":                      "GET",
		"/components":           "GET",
		"/favicon.ico":          "GET",
		"/favicon-16x16.png":    "GET",
		"/favicon-32x32.png":    "GET",
		"/apple-touch-icon.png": "GET",
		"/site.webmanifest":     "GET",
	}

	for _, route := range routes.Routes {
		expectedMethod, ok := expectedPatterns[route.Pattern]
		if !ok {
			t.Errorf("unexpected route pattern: %s", route.Pattern)
			continue
		}

		if route.Method != expectedMethod {
			t.Errorf("route %s: Method = %q, want %q", route.Pattern, route.Method, expectedMethod)
		}

		if route.Handler == nil {
			t.Errorf("route %s: Handler is nil", route.Pattern)
		}
	}

	if len(routes.Routes) != len(expectedPatterns) {
		t.Errorf("Routes count = %d, want %d", len(routes.Routes), len(expectedPatterns))
	}
}
