package pkg_middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/middleware"
)

func TestAddSlash_RedirectsWithoutSlash(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.AddSlash()(handler)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMovedPermanently {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusMovedPermanently)
	}

	location := resp.Header.Get("Location")
	if location != "/users/" {
		t.Errorf("Location = %q, want %q", location, "/users/")
	}
}

func TestAddSlash_PreservesQueryString(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.AddSlash()(handler)

	req := httptest.NewRequest(http.MethodGet, "/users?page=1", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	location := resp.Header.Get("Location")
	if location != "/users/?page=1" {
		t.Errorf("Location = %q, want %q", location, "/users/?page=1")
	}
}

func TestAddSlash_PassesWithSlash(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	wrapped := middleware.AddSlash()(handler)

	req := httptest.NewRequest(http.MethodGet, "/users/", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestAddSlash_PassesFileExtensions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("file"))
	})

	wrapped := middleware.AddSlash()(handler)

	tests := []string{
		"/app.js",
		"/styles.css",
		"/image.png",
		"/dist/bundle.min.js",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			wrapped.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("status = %d, want %d (should not redirect files)", resp.StatusCode, http.StatusOK)
			}
		})
	}
}

func TestTrimSlash_RedirectsWithSlash(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.TrimSlash()(handler)

	req := httptest.NewRequest(http.MethodGet, "/users/", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMovedPermanently {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusMovedPermanently)
	}

	location := resp.Header.Get("Location")
	if location != "/users" {
		t.Errorf("Location = %q, want %q", location, "/users")
	}
}

func TestTrimSlash_PreservesQueryString(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.TrimSlash()(handler)

	req := httptest.NewRequest(http.MethodGet, "/users/?page=1", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	location := resp.Header.Get("Location")
	if location != "/users?page=1" {
		t.Errorf("Location = %q, want %q", location, "/users?page=1")
	}
}

func TestTrimSlash_PreservesRoot(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("root"))
	})

	wrapped := middleware.TrimSlash()(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d (root should not redirect)", resp.StatusCode, http.StatusOK)
	}
}

func TestTrimSlash_PassesWithoutSlash(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	wrapped := middleware.TrimSlash()(handler)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
