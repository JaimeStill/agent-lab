package pkg_middleware_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/middleware"
)

func TestLogger_LogsRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.Logger(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	logOutput := buf.String()

	if !strings.Contains(logOutput, "request") {
		t.Error("log should contain 'request' message")
	}

	if !strings.Contains(logOutput, "GET") {
		t.Error("log should contain method")
	}

	if !strings.Contains(logOutput, "/api/users") {
		t.Error("log should contain URI")
	}

	if !strings.Contains(logOutput, "duration") {
		t.Error("log should contain duration")
	}
}

func TestLogger_CallsNextHandler(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.Write([]byte("response"))
	})

	wrapped := middleware.Logger(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("next handler was not called")
	}

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestLogger_LogsAfterHandler(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if buf.Len() > 0 {
			t.Error("log was written before handler completed")
		}
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.Logger(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if buf.Len() == 0 {
		t.Error("log was not written after handler completed")
	}
}
