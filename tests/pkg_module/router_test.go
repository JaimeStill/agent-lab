package pkg_module_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/module"
)

func TestNewRouter(t *testing.T) {
	r := module.NewRouter()

	if r == nil {
		t.Fatal("NewRouter() returned nil")
	}
}

func TestRouter_HandleNative(t *testing.T) {
	r := module.NewRouter()

	r.HandleNative("GET /healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Errorf("body = %q, want %q", string(body), "ok")
	}
}

func TestRouter_Mount(t *testing.T) {
	r := module.NewRouter()

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("api response"))
	})

	m := module.New("/api", handler)
	r.Mount(m)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "api response" {
		t.Errorf("body = %q, want %q", string(body), "api response")
	}
}

func TestRouter_MultipleModules(t *testing.T) {
	r := module.NewRouter()

	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("api"))
	})

	appHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("app"))
	})

	r.Mount(module.New("/api", apiHandler))
	r.Mount(module.New("/app", appHandler))

	tests := []struct {
		path string
		want string
	}{
		{"/api/users", "api"},
		{"/app/home", "app"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
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

func TestRouter_ModulePrefixStripping(t *testing.T) {
	r := module.NewRouter()

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(req.URL.Path))
	})

	r.Mount(module.New("/api", handler))

	tests := []struct {
		name     string
		path     string
		wantPath string
	}{
		{"module root", "/api", "/"},
		{"single segment", "/api/users", "/users"},
		{"multiple segments", "/api/users/123/posts", "/users/123/posts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			if string(body) != tt.wantPath {
				t.Errorf("stripped path = %q, want %q", string(body), tt.wantPath)
			}
		})
	}
}

func TestRouter_FallbackToNative(t *testing.T) {
	r := module.NewRouter()

	r.HandleNative("GET /healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("healthy"))
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("api"))
	})

	r.Mount(module.New("/api", handler))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "healthy" {
		t.Errorf("body = %q, want %q", string(body), "healthy")
	}
}

func TestRouter_UnmatchedPath(t *testing.T) {
	r := module.NewRouter()

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("api"))
	})

	r.Mount(module.New("/api", handler))

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestRouter_PathNormalization(t *testing.T) {
	r := module.NewRouter()

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(req.URL.Path))
	})

	r.Mount(module.New("/api", handler))

	tests := []struct {
		name       string
		inputPath  string
		wantPath   string
		wantStatus int
	}{
		{"strips trailing slash", "/api/users/", "/users", http.StatusOK},
		{"no change for path without slash", "/api/users", "/users", http.StatusOK},
		{"root path unchanged", "/", "/", http.StatusNotFound},
		{"module root with slash normalized", "/api/", "/", http.StatusOK},
		{"deep path trailing slash", "/api/users/123/posts/", "/users/123/posts", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.inputPath, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				if string(body) != tt.wantPath {
					t.Errorf("path = %q, want %q", string(body), tt.wantPath)
				}
			}
		})
	}
}
