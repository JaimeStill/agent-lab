package internal_workflows_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
)

func TestNewSystem(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runtime := workflows.NewRuntime(nil, nil, nil, logger)
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
	runtime := workflows.NewRuntime(nil, nil, nil, logger)
	paginationCfg := pagination.Config{
		DefaultPageSize: 20,
		MaxPageSize:     100,
	}

	var _ workflows.System = workflows.NewSystem(runtime, nil, logger, paginationCfg)
}

func TestExecutor_ListWorkflows(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runtime := workflows.NewRuntime(nil, nil, nil, logger)
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
