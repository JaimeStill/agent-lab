package pkg_web_test

import (
	"embed"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/web"
)

//go:embed testdata/static/*
var staticFS embed.FS

func TestDistServer(t *testing.T) {
	handler := web.DistServer(staticFS, "testdata/static", "/dist/")
	if handler == nil {
		t.Fatal("DistServer() returned nil")
	}
}

func TestDistServerServesFile(t *testing.T) {
	handler := web.DistServer(staticFS, "testdata/static", "/dist/")

	req := httptest.NewRequest(http.MethodGet, "/dist/app.js", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "console.log") {
		t.Error("response body does not contain expected content")
	}
}

func TestDistServerNotFound(t *testing.T) {
	handler := web.DistServer(staticFS, "testdata/static", "/dist/")

	req := httptest.NewRequest(http.MethodGet, "/dist/nonexistent.js", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestPublicFile(t *testing.T) {
	handler := web.PublicFile(staticFS, "testdata/static", "test.txt")
	if handler == nil {
		t.Fatal("PublicFile() returned nil")
	}
}

func TestPublicFileServesFile(t *testing.T) {
	handler := web.PublicFile(staticFS, "testdata/static", "test.txt")

	req := httptest.NewRequest(http.MethodGet, "/test.txt", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "test content") {
		t.Error("response body does not contain expected content")
	}
}

func TestPublicFileNotFound(t *testing.T) {
	handler := web.PublicFile(staticFS, "testdata/static", "nonexistent.txt")

	req := httptest.NewRequest(http.MethodGet, "/nonexistent.txt", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestPublicFileRoutes(t *testing.T) {
	routes := web.PublicFileRoutes(staticFS, "testdata/static", "test.txt", "app.js")

	if len(routes) != 2 {
		t.Fatalf("PublicFileRoutes() returned %d routes, want 2", len(routes))
	}

	expectedPatterns := []string{"/test.txt", "/app.js"}
	for i, route := range routes {
		if route.Method != "GET" {
			t.Errorf("route %d: Method = %q, want GET", i, route.Method)
		}
		if route.Pattern != expectedPatterns[i] {
			t.Errorf("route %d: Pattern = %q, want %q", i, route.Pattern, expectedPatterns[i])
		}
		if route.Handler == nil {
			t.Errorf("route %d: Handler is nil", i)
		}
	}
}

func TestServeEmbeddedFile(t *testing.T) {
	data := []byte("hello world")
	handler := web.ServeEmbeddedFile(data, "text/plain")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Content-Type = %q, want text/plain", contentType)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello world" {
		t.Errorf("body = %q, want %q", string(body), "hello world")
	}
}
