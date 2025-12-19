package internal_workflows_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
)

func TestNewHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	paginationCfg := pagination.Config{DefaultPageSize: 20, MaxPageSize: 100}

	handler := workflows.NewHandler(nil, logger, paginationCfg)

	if handler == nil {
		t.Fatal("NewHandler() returned nil")
	}
}

func TestHandler_Routes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	paginationCfg := pagination.Config{DefaultPageSize: 20, MaxPageSize: 100}

	handler := workflows.NewHandler(nil, logger, paginationCfg)
	group := handler.Routes()

	if group.Prefix != "/api/workflows" {
		t.Errorf("Prefix = %q, want %q", group.Prefix, "/api/workflows")
	}

	if len(group.Tags) != 1 || group.Tags[0] != "Workflows" {
		t.Errorf("Tags = %v, want [Workflows]", group.Tags)
	}

	expectedRoutes := []struct {
		method  string
		pattern string
	}{
		{"GET", ""},
		{"POST", "/{name}/execute"},
		{"POST", "/{name}/execute/stream"},
	}

	if len(group.Routes) != len(expectedRoutes) {
		t.Fatalf("Routes count = %d, want %d", len(group.Routes), len(expectedRoutes))
	}

	for i, expected := range expectedRoutes {
		if group.Routes[i].Method != expected.method {
			t.Errorf("Routes[%d].Method = %q, want %q", i, group.Routes[i].Method, expected.method)
		}
		if group.Routes[i].Pattern != expected.pattern {
			t.Errorf("Routes[%d].Pattern = %q, want %q", i, group.Routes[i].Pattern, expected.pattern)
		}
		if group.Routes[i].Handler == nil {
			t.Errorf("Routes[%d].Handler is nil", i)
		}
		if group.Routes[i].OpenAPI == nil {
			t.Errorf("Routes[%d].OpenAPI is nil", i)
		}
	}
}

func TestHandler_Routes_Children(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	paginationCfg := pagination.Config{DefaultPageSize: 20, MaxPageSize: 100}

	handler := workflows.NewHandler(nil, logger, paginationCfg)
	group := handler.Routes()

	if len(group.Children) != 1 {
		t.Fatalf("Children count = %d, want 1", len(group.Children))
	}

	runsGroup := group.Children[0]

	if runsGroup.Prefix != "/runs" {
		t.Errorf("Children[0].Prefix = %q, want %q", runsGroup.Prefix, "/runs")
	}

	if len(runsGroup.Tags) != 1 || runsGroup.Tags[0] != "Runs" {
		t.Errorf("Children[0].Tags = %v, want [Runs]", runsGroup.Tags)
	}

	expectedRunsRoutes := []struct {
		method  string
		pattern string
	}{
		{"GET", ""},
		{"GET", "/{id}"},
		{"GET", "/{id}/stages"},
		{"GET", "/{id}/decisions"},
		{"POST", "/{id}/cancel"},
		{"POST", "/{id}/resume"},
	}

	if len(runsGroup.Routes) != len(expectedRunsRoutes) {
		t.Fatalf("Children[0].Routes count = %d, want %d", len(runsGroup.Routes), len(expectedRunsRoutes))
	}

	for i, expected := range expectedRunsRoutes {
		if runsGroup.Routes[i].Method != expected.method {
			t.Errorf("Children[0].Routes[%d].Method = %q, want %q", i, runsGroup.Routes[i].Method, expected.method)
		}
		if runsGroup.Routes[i].Pattern != expected.pattern {
			t.Errorf("Children[0].Routes[%d].Pattern = %q, want %q", i, runsGroup.Routes[i].Pattern, expected.pattern)
		}
		if runsGroup.Routes[i].Handler == nil {
			t.Errorf("Children[0].Routes[%d].Handler is nil", i)
		}
		if runsGroup.Routes[i].OpenAPI == nil {
			t.Errorf("Children[0].Routes[%d].OpenAPI is nil", i)
		}
	}
}
