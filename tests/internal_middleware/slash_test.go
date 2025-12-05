package internal_middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/middleware"
)

func TestTrimSlash(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name           string
		path           string
		wantStatus     int
		wantLocation   string
		shouldRedirect bool
	}{
		{
			name:           "root path preserved",
			path:           "/",
			wantStatus:     http.StatusOK,
			shouldRedirect: false,
		},
		{
			name:           "path without trailing slash",
			path:           "/docs",
			wantStatus:     http.StatusOK,
			shouldRedirect: false,
		},
		{
			name:           "path with trailing slash redirects",
			path:           "/docs/",
			wantStatus:     http.StatusMovedPermanently,
			wantLocation:   "/docs",
			shouldRedirect: true,
		},
		{
			name:           "nested path with trailing slash redirects",
			path:           "/api/users/",
			wantStatus:     http.StatusMovedPermanently,
			wantLocation:   "/api/users",
			shouldRedirect: true,
		},
		{
			name:           "deeply nested path",
			path:           "/api/v1/users/123/",
			wantStatus:     http.StatusMovedPermanently,
			wantLocation:   "/api/v1/users/123",
			shouldRedirect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := middleware.TrimSlash()
			wrapped := mw(handler)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			wrapped.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			if tt.shouldRedirect {
				location := resp.Header.Get("Location")
				if location != tt.wantLocation {
					t.Errorf("Location = %q, want %q", location, tt.wantLocation)
				}
			}
		})
	}
}

func TestTrimSlash_PreservesQueryString(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.TrimSlash()
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/users/?page=1&size=10", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMovedPermanently {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusMovedPermanently)
	}

	location := resp.Header.Get("Location")
	want := "/users?page=1&size=10"
	if location != want {
		t.Errorf("Location = %q, want %q", location, want)
	}
}

func TestTrimSlash_AllHTTPMethods(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
	}

	mw := middleware.TrimSlash()
	wrapped := mw(handler)

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/", nil)
			w := httptest.NewRecorder()

			wrapped.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusMovedPermanently {
				t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusMovedPermanently)
			}

			location := resp.Header.Get("Location")
			if location != "/api" {
				t.Errorf("Location = %q, want %q", location, "/api")
			}
		})
	}
}
