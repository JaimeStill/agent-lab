package pkg_web_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/web"
)

func TestNewRouter(t *testing.T) {
	r := web.NewRouter()
	if r == nil {
		t.Fatal("NewRouter() returned nil")
	}
}

func TestRouterHandle(t *testing.T) {
	r := web.NewRouter()

	called := false
	r.Handle("GET /test", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if !called {
		t.Error("handler was not called")
	}

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestRouterHandleFunc(t *testing.T) {
	r := web.NewRouter()

	called := false
	r.HandleFunc("GET /test", func(w http.ResponseWriter, req *http.Request) {
		called = true
		w.Write([]byte("hello"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if !called {
		t.Error("handler was not called")
	}

	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello" {
		t.Errorf("body = %q, want %q", string(body), "hello")
	}
}

func TestRouterWithoutFallback(t *testing.T) {
	r := web.NewRouter()
	r.HandleFunc("GET /exists", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("exists"))
	})

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestRouterSetFallback(t *testing.T) {
	r := web.NewRouter()
	r.HandleFunc("GET /exists", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("exists"))
	})

	fallbackCalled := false
	r.SetFallback(func(w http.ResponseWriter, req *http.Request) {
		fallbackCalled = true
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("custom 404"))
	})

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if !fallbackCalled {
		t.Error("fallback handler was not called")
	}

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "custom 404" {
		t.Errorf("body = %q, want %q", string(body), "custom 404")
	}
}

func TestRouterFallbackNotCalledForMatchedRoutes(t *testing.T) {
	r := web.NewRouter()

	routeHandlerCalled := false
	r.HandleFunc("GET /exists", func(w http.ResponseWriter, req *http.Request) {
		routeHandlerCalled = true
		w.Write([]byte("exists"))
	})

	fallbackCalled := false
	r.SetFallback(func(w http.ResponseWriter, req *http.Request) {
		fallbackCalled = true
		w.WriteHeader(http.StatusNotFound)
	})

	req := httptest.NewRequest(http.MethodGet, "/exists", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if !routeHandlerCalled {
		t.Error("route handler was not called")
	}
	if fallbackCalled {
		t.Error("fallback should not be called for matched routes")
	}

	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "exists" {
		t.Errorf("body = %q, want %q", string(body), "exists")
	}
}

func TestRouterMultipleRoutes(t *testing.T) {
	r := web.NewRouter()

	r.HandleFunc("GET /one", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("one"))
	})
	r.HandleFunc("GET /two", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("two"))
	})
	r.HandleFunc("POST /submit", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("submitted"))
	})

	tests := []struct {
		method string
		path   string
		want   string
	}{
		{http.MethodGet, "/one", "one"},
		{http.MethodGet, "/two", "two"},
		{http.MethodPost, "/submit", "submitted"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			if string(body) != tt.want {
				t.Errorf("body = %q, want %q", string(body), tt.want)
			}
		})
	}
}
