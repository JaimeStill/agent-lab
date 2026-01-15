package web_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JaimeStill/agent-lab/web"
)

func TestNewHandler(t *testing.T) {
	h, err := web.NewHandler("/app")
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	if h == nil {
		t.Fatal("NewHandler() returned nil")
	}
}

func TestHandlerRouter(t *testing.T) {
	h, err := web.NewHandler("/app")
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	router := h.Router()
	if router == nil {
		t.Fatal("Router() returned nil")
	}
}

func TestRouterServesHome(t *testing.T) {
	h, err := web.NewHandler("/app")
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	router := h.Router()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

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
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "<!DOCTYPE html>") {
		t.Error("response body does not contain DOCTYPE")
	}
}

func TestRouterServesComponents(t *testing.T) {
	h, err := web.NewHandler("/app")
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	router := h.Router()

	req := httptest.NewRequest(http.MethodGet, "/components", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Component Styles") {
		t.Error("response body does not contain expected content")
	}
}

func TestRouterServes404(t *testing.T) {
	h, err := web.NewHandler("/app")
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	router := h.Router()

	req := httptest.NewRequest(http.MethodGet, "/nonexistent-page", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "<!DOCTYPE html>") {
		t.Error("404 response should be HTML page")
	}
}

func TestRouterServesDistAssets(t *testing.T) {
	h, err := web.NewHandler("/app")
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	router := h.Router()

	req := httptest.NewRequest(http.MethodGet, "/dist/app.js", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

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

func TestRouterServesPublicFiles(t *testing.T) {
	h, err := web.NewHandler("/app")
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	router := h.Router()

	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

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

func TestRouterBasePathInTemplates(t *testing.T) {
	h, err := web.NewHandler("/myapp")
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	router := h.Router()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "/myapp") {
		t.Error("response does not contain basePath in template output")
	}
}

func TestRouterDistNotFound(t *testing.T) {
	h, err := web.NewHandler("/app")
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	router := h.Router()

	req := httptest.NewRequest(http.MethodGet, "/dist/nonexistent.js", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}
