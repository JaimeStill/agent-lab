package internal_workflows_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/workflows"
)

func TestNewRuntime(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	runtime := workflows.NewRuntime(nil, nil, nil, nil, logger)

	if runtime == nil {
		t.Fatal("NewRuntime() returned nil")
	}
}

func TestRuntime_Getters(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	runtime := workflows.NewRuntime(nil, nil, nil, nil, logger)

	if runtime.Agents() != nil {
		t.Error("Agents() should return nil when initialized with nil")
	}

	if runtime.Documents() != nil {
		t.Error("Documents() should return nil when initialized with nil")
	}

	if runtime.Images() != nil {
		t.Error("Images() should return nil when initialized with nil")
	}

	if runtime.Profiles() != nil {
		t.Error("Profiles() should return nil when initialized with nil")
	}

	if runtime.Logger() != logger {
		t.Error("Logger() should return the logger passed to NewRuntime")
	}
}
