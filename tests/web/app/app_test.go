package app_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JaimeStill/agent-lab/web/app"
)

func TestNewModule(t *testing.T) {
	m, err := app.NewModule("/app")
	if err != nil {
		t.Fatalf("NewModule() error = %v", err)
	}
	if m == nil {
		t.Fatal("NewModule() returned nil")
	}
}

func TestModuleHandler(t *testing.T) {
	m, err := app.NewModule("/app")
	if err != nil {
		t.Fatalf("NewModule() error = %v", err)
	}

	handler := m.Handler()
	if handler == nil {
		t.Fatal("Handler() returned nil")
	}
}

func TestModuleServesHome(t *testing.T) {
	m, err := app.NewModule("/app")
	if err != nil {
		t.Fatalf("NewModule() error = %v", err)
	}

	handler := m.Handler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

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

func TestModuleServesComponents(t *testing.T) {
	m, err := app.NewModule("/app")
	if err != nil {
		t.Fatalf("NewModule() error = %v", err)
	}

	handler := m.Handler()

	req := httptest.NewRequest(http.MethodGet, "/components", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

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

func TestModuleServes404(t *testing.T) {
	m, err := app.NewModule("/app")
	if err != nil {
		t.Fatalf("NewModule() error = %v", err)
	}

	handler := m.Handler()

	req := httptest.NewRequest(http.MethodGet, "/nonexistent-page", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

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

func TestModuleServesDistAssets(t *testing.T) {
	m, err := app.NewModule("/app")
	if err != nil {
		t.Fatalf("NewModule() error = %v", err)
	}

	handler := m.Handler()

	req := httptest.NewRequest(http.MethodGet, "/dist/app.js", nil)
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
}

func TestModuleServesPublicFiles(t *testing.T) {
	m, err := app.NewModule("/app")
	if err != nil {
		t.Fatalf("NewModule() error = %v", err)
	}

	handler := m.Handler()

	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

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

func TestModuleBasePathInTemplates(t *testing.T) {
	m, err := app.NewModule("/myapp")
	if err != nil {
		t.Fatalf("NewModule() error = %v", err)
	}

	handler := m.Handler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "/myapp") {
		t.Error("response does not contain basePath in template output")
	}
}

func TestModuleDistNotFound(t *testing.T) {
	m, err := app.NewModule("/app")
	if err != nil {
		t.Fatalf("NewModule() error = %v", err)
	}

	handler := m.Handler()

	req := httptest.NewRequest(http.MethodGet, "/dist/nonexistent.js", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}
