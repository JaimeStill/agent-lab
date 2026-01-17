package web_scalar_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JaimeStill/agent-lab/web/scalar"
)

func TestNewModule(t *testing.T) {
	m := scalar.NewModule("/scalar")
	if m == nil {
		t.Fatal("NewModule() returned nil")
	}
}

func TestModulePrefix(t *testing.T) {
	m := scalar.NewModule("/scalar")
	if m.Prefix() != "/scalar" {
		t.Errorf("Prefix() = %q, want %q", m.Prefix(), "/scalar")
	}
}

func TestModuleHandler(t *testing.T) {
	m := scalar.NewModule("/scalar")
	handler := m.Handler()
	if handler == nil {
		t.Fatal("Handler() returned nil")
	}
}

func TestServeIndex(t *testing.T) {
	m := scalar.NewModule("/scalar")
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
	if !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", contentType)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "<!DOCTYPE html>") {
		t.Error("response body does not contain DOCTYPE")
	}
}

func TestServeIndexContainsBasePath(t *testing.T) {
	m := scalar.NewModule("/docs")
	handler := m.Handler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "/docs") {
		t.Error("response body does not contain basePath")
	}
}

func TestServeAssets(t *testing.T) {
	m := scalar.NewModule("/scalar")
	handler := m.Handler()

	tests := []struct {
		path string
	}{
		{"/scalar.js"},
		{"/scalar.css"},
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
