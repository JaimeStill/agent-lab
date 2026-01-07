package internal_workflows_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/workflows"
	_ "github.com/JaimeStill/agent-lab/workflows"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
)

func TestNewSystem(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runtime := workflows.NewRuntime(nil, nil, nil, nil, nil, logger)
	paginationCfg := pagination.Config{
		DefaultPageSize: 20,
		MaxPageSize:     100,
	}

	sys := workflows.NewSystem(runtime, nil, logger, paginationCfg)

	if sys == nil {
		t.Fatal("NewSystem() returned nil")
	}
}

func TestNewSystem_ImplementsInterface(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runtime := workflows.NewRuntime(nil, nil, nil, nil, nil, logger)
	paginationCfg := pagination.Config{
		DefaultPageSize: 20,
		MaxPageSize:     100,
	}

	var _ workflows.System = workflows.NewSystem(runtime, nil, logger, paginationCfg)
}

func TestExecutor_ListWorkflows(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runtime := workflows.NewRuntime(nil, nil, nil, nil, nil, logger)
	paginationCfg := pagination.Config{
		DefaultPageSize: 20,
		MaxPageSize:     100,
	}

	sys := workflows.NewSystem(runtime, nil, logger, paginationCfg)

	infos := sys.ListWorkflows()
	if infos == nil {
		t.Fatal("ListWorkflows() returned nil")
	}
}

func TestWorkflows_Registered(t *testing.T) {
	infos := workflows.List()

	workflowNames := make(map[string]bool)
	for _, info := range infos {
		workflowNames[info.Name] = true
	}

	tests := []struct {
		name        string
		description string
	}{
		{"summarize", "Summarizes input text using an AI agent"},
		{"reasoning", "Multi-step reasoning workflow that analyzes problems"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !workflowNames[tt.name] {
				t.Errorf("workflow %q not registered", tt.name)
			}

			factory, exists := workflows.Get(tt.name)
			if !exists {
				t.Errorf("Get(%q) returned false", tt.name)
			}
			if factory == nil {
				t.Errorf("Get(%q) returned nil factory", tt.name)
			}
		})
	}
}

func TestWorkflows_Info(t *testing.T) {
	infos := workflows.List()

	infoMap := make(map[string]workflows.WorkflowInfo)
	for _, info := range infos {
		infoMap[info.Name] = info
	}

	if info, ok := infoMap["summarize"]; ok {
		if info.Description != "Summarizes input text using an AI agent" {
			t.Errorf("summarize description = %q, want %q", info.Description, "Summarizes input text using an AI agent")
		}
	}

	if info, ok := infoMap["reasoning"]; ok {
		if info.Description != "Multi-step reasoning workflow that analyzes problems" {
			t.Errorf("reasoning description = %q, want %q", info.Description, "Multi-step reasoning workflow that analyzes problems")
		}
	}
}
