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

func TestHandlerServesApp(t *testing.T) {
	h, err := web.NewHandler()
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	routes := h.Routes()
	handler := routes.Routes[0].Handler

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
