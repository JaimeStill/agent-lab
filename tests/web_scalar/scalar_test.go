package web_scalar_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JaimeStill/agent-lab/web/scalar"
)

func TestHandler(t *testing.T) {
	handler := scalar.Handler()
	if handler == nil {
		t.Fatal("Handler() returned nil")
	}
}

func TestRoutes(t *testing.T) {
	group := scalar.Routes()

	if group.Prefix != "/scalar" {
		t.Errorf("Prefix = %q, want %q", group.Prefix, "/scalar")
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
	handler := scalar.Handler()

	req := httptest.NewRequest(http.MethodGet, "/scalar", nil)
	w := httptest.NewRecorder()

	handler(w, req)

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
