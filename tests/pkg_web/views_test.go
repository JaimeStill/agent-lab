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

//go:embed testdata/layouts/*
var layoutFS embed.FS

//go:embed testdata/pages/*
var pageFS embed.FS

var testPages = []web.ViewDef{
	{Route: "", Template: "home.html", Title: "Home", Bundle: "app"},
	{Route: "/about", Template: "about.html", Title: "About", Bundle: "app"},
}

var errorPages = []web.ViewDef{
	{Template: "404.html", Title: "Not Found"},
}

func TestNewTemplateSet(t *testing.T) {
	ts, err := web.NewTemplateSet(layoutFS, pageFS, "testdata/layouts/*.html", "testdata/pages", "/app", testPages)
	if err != nil {
		t.Fatalf("NewTemplateSet() error = %v", err)
	}
	if ts == nil {
		t.Fatal("NewTemplateSet() returned nil")
	}
}

func TestNewTemplateSetInvalidLayoutGlob(t *testing.T) {
	_, err := web.NewTemplateSet(layoutFS, pageFS, "nonexistent/*.html", "testdata/pages", "/app", testPages)
	if err == nil {
		t.Error("NewTemplateSet() with invalid layout glob should return error")
	}
}

func TestNewTemplateSetInvalidPageSubdir(t *testing.T) {
	_, err := web.NewTemplateSet(layoutFS, pageFS, "testdata/layouts/*.html", "nonexistent", "/app", testPages)
	if err == nil {
		t.Error("NewTemplateSet() with invalid page subdir should return error")
	}
}

func TestNewTemplateSetInvalidTemplate(t *testing.T) {
	invalidPages := []web.ViewDef{
		{Route: "", Template: "nonexistent.html", Title: "Missing", Bundle: "app"},
	}
	_, err := web.NewTemplateSet(layoutFS, pageFS, "testdata/layouts/*.html", "testdata/pages", "/app", invalidPages)
	if err == nil {
		t.Error("NewTemplateSet() with invalid template should return error")
	}
}

func TestRender(t *testing.T) {
	ts, err := web.NewTemplateSet(layoutFS, pageFS, "testdata/layouts/*.html", "testdata/pages", "/app", testPages)
	if err != nil {
		t.Fatalf("NewTemplateSet() error = %v", err)
	}

	w := httptest.NewRecorder()
	data := web.ViewData{Title: "Test", Bundle: "test-bundle", BasePath: "/app"}

	err = ts.Render(w, "test.html", "home.html", data)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	resp := w.Result()
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", contentType)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "<!DOCTYPE html>") {
		t.Error("response does not contain DOCTYPE")
	}
	if !strings.Contains(bodyStr, "<title>Test</title>") {
		t.Error("response does not contain title")
	}
	if !strings.Contains(bodyStr, "Home Page") {
		t.Error("response does not contain page content")
	}
	if !strings.Contains(bodyStr, "test-bundle") {
		t.Error("response does not contain bundle")
	}
	if !strings.Contains(bodyStr, `data-basepath="/app"`) {
		t.Error("response does not contain basepath")
	}
}

func TestRenderNotFound(t *testing.T) {
	ts, err := web.NewTemplateSet(layoutFS, pageFS, "testdata/layouts/*.html", "testdata/pages", "/app", testPages)
	if err != nil {
		t.Fatalf("NewTemplateSet() error = %v", err)
	}

	w := httptest.NewRecorder()
	data := web.ViewData{Title: "Test", Bundle: "app"}

	err = ts.Render(w, "test.html", "nonexistent.html", data)
	if err == nil {
		t.Error("Render() with nonexistent template should return error")
	}
}

func TestPageHandler(t *testing.T) {
	ts, err := web.NewTemplateSet(layoutFS, pageFS, "testdata/layouts/*.html", "testdata/pages", "/app", testPages)
	if err != nil {
		t.Fatalf("NewTemplateSet() error = %v", err)
	}

	handler := ts.PageHandler("test.html", testPages[0])
	if handler == nil {
		t.Fatal("PageHandler() returned nil")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "<title>Home</title>") {
		t.Error("response does not contain page title")
	}
	if !strings.Contains(bodyStr, "app") {
		t.Error("response does not contain bundle name")
	}
	if !strings.Contains(bodyStr, `data-basepath="/app"`) {
		t.Error("response does not contain basepath from TemplateSet")
	}
}

func TestPageHandlerServesPages(t *testing.T) {
	ts, err := web.NewTemplateSet(layoutFS, pageFS, "testdata/layouts/*.html", "testdata/pages", "/test", testPages)
	if err != nil {
		t.Fatalf("NewTemplateSet() error = %v", err)
	}

	tests := []struct {
		name     string
		page     web.ViewDef
		wantBody string
	}{
		{"home", testPages[0], "Home Page"},
		{"about", testPages[1], "About Page"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := ts.PageHandler("test.html", tt.page)

			req := httptest.NewRequest(http.MethodGet, "/test"+tt.page.Route, nil)
			w := httptest.NewRecorder()

			handler(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
			}

			body, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(body), tt.wantBody) {
				t.Errorf("response does not contain %q", tt.wantBody)
			}
			if !strings.Contains(string(body), `data-basepath="/test"`) {
				t.Error("response does not contain basepath")
			}
		})
	}
}

func TestErrorHandler(t *testing.T) {
	allPages := append(testPages, errorPages...)
	ts, err := web.NewTemplateSet(layoutFS, pageFS, "testdata/layouts/*.html", "testdata/pages", "/app", allPages)
	if err != nil {
		t.Fatalf("NewTemplateSet() error = %v", err)
	}

	handler := ts.ErrorHandler("test.html", errorPages[0], http.StatusNotFound)
	if handler == nil {
		t.Fatal("ErrorHandler() returned nil")
	}

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "<title>Not Found</title>") {
		t.Error("response does not contain error title")
	}
	if !strings.Contains(bodyStr, "404 Not Found") {
		t.Error("response does not contain error content")
	}
	if !strings.Contains(bodyStr, `data-basepath="/app"`) {
		t.Error("response does not contain basepath")
	}
}

func TestErrorHandlerDifferentStatus(t *testing.T) {
	allPages := append(testPages, errorPages...)
	ts, err := web.NewTemplateSet(layoutFS, pageFS, "testdata/layouts/*.html", "testdata/pages", "/app", allPages)
	if err != nil {
		t.Fatalf("NewTemplateSet() error = %v", err)
	}

	handler := ts.ErrorHandler("test.html", errorPages[0], http.StatusInternalServerError)

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "<title>Not Found</title>") {
		t.Error("response does not contain error title from PageDef")
	}
}
