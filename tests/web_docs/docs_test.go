package web_docs_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JaimeStill/agent-lab/web/docs"
)

func TestNewHandler(t *testing.T) {
	handler := docs.NewHandler(nil)

	if handler == nil {
		t.Fatal("NewHandler() returned nil")
	}
}

func TestRoutes(t *testing.T) {
	handler := docs.NewHandler(nil)
	group := handler.Routes()

	if group.Prefix != "/docs" {
		t.Errorf("Prefix = %q, want %q", group.Prefix, "/docs")
	}

	if len(group.Tags) == 0 {
		t.Error("Tags is empty")
	}

	if group.Description == "" {
		t.Error("Description is empty")
	}

	expectedPatterns := map[string]string{
		"": "GET",
	}

	for _, route := range group.Routes {
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

	if len(group.Routes) != len(expectedPatterns) {
		t.Errorf("Routes count = %d, want %d", len(group.Routes), len(expectedPatterns))
	}
}

func TestServeIndex(t *testing.T) {
	handler := docs.NewHandler(nil)
	group := handler.Routes()

	var indexHandler http.HandlerFunc
	for _, route := range group.Routes {
		if route.Pattern == "" {
			indexHandler = route.Handler
			break
		}
	}

	if indexHandler == nil {
		t.Fatal("index handler not found")
	}

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	w := httptest.NewRecorder()

	indexHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", contentType)
	}
}
