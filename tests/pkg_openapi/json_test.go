package pkg_openapi_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/openapi"
)

func TestMarshalJSON(t *testing.T) {
	spec := &openapi.Spec{
		OpenAPI: "3.1.0",
		Info: &openapi.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]*openapi.PathItem{
			"/users": {
				Get: &openapi.Operation{
					Summary: "List users",
					Responses: map[int]*openapi.Response{
						200: {Description: "Success"},
					},
				},
			},
		},
	}

	data, err := openapi.MarshalJSON(spec)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if result["openapi"] != "3.1.0" {
		t.Errorf("openapi = %v, want 3.1.0", result["openapi"])
	}

	info, ok := result["info"].(map[string]any)
	if !ok {
		t.Fatal("info is not an object")
	}

	if info["title"] != "Test API" {
		t.Errorf("info.title = %v, want Test API", info["title"])
	}
}

func TestWriteJSON(t *testing.T) {
	spec := &openapi.Spec{
		OpenAPI: "3.1.0",
		Info: &openapi.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: make(map[string]*openapi.PathItem),
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "openapi.json")

	err := openapi.WriteJSON(spec, filePath)
	if err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("written file is not valid JSON: %v", err)
	}

	if result["openapi"] != "3.1.0" {
		t.Errorf("openapi = %v, want 3.1.0", result["openapi"])
	}
}

func TestWriteJSON_InvalidPath(t *testing.T) {
	spec := &openapi.Spec{
		OpenAPI: "3.1.0",
		Info: &openapi.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: make(map[string]*openapi.PathItem),
	}

	err := openapi.WriteJSON(spec, "/nonexistent/directory/openapi.json")
	if err == nil {
		t.Error("WriteJSON() expected error for invalid path, got nil")
	}
}
