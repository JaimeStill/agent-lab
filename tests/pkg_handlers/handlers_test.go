package pkg_handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/handlers"
)

func TestRespondJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       any
		wantStatus int
		wantBody   string
	}{
		{
			"ok with map",
			http.StatusOK,
			map[string]string{"message": "hello"},
			http.StatusOK,
			`{"message":"hello"}`,
		},
		{
			"created with struct",
			http.StatusCreated,
			struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}{1, "test"},
			http.StatusCreated,
			`{"id":1,"name":"test"}`,
		},
		{
			"ok with slice",
			http.StatusOK,
			[]int{1, 2, 3},
			http.StatusOK,
			`[1,2,3]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			handlers.RespondJSON(w, tt.status, tt.data)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			contentType := resp.Header.Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
			}

			body, _ := io.ReadAll(resp.Body)
			var got, want any
			json.Unmarshal(body, &got)
			json.Unmarshal([]byte(tt.wantBody), &want)

			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("body = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}

func TestRespondError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		err        error
		wantStatus int
		wantError  string
	}{
		{
			"bad request",
			http.StatusBadRequest,
			errors.New("invalid input"),
			http.StatusBadRequest,
			"invalid input",
		},
		{
			"not found",
			http.StatusNotFound,
			errors.New("resource not found"),
			http.StatusNotFound,
			"resource not found",
		},
		{
			"internal error",
			http.StatusInternalServerError,
			errors.New("something went wrong"),
			http.StatusInternalServerError,
			"something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			handlers.RespondError(w, logger, tt.status, tt.err)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			contentType := resp.Header.Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
			}

			body, _ := io.ReadAll(resp.Body)
			var result map[string]string
			json.Unmarshal(body, &result)

			if result["error"] != tt.wantError {
				t.Errorf("error = %q, want %q", result["error"], tt.wantError)
			}
		})
	}
}
