package internal_workflows_test

import (
	"context"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
)

func TestRegister_And_Get(t *testing.T) {
	factory := func(ctx context.Context, graph state.StateGraph, runtime *workflows.Runtime, params map[string]any) (state.State, error) {
		return state.State{}, nil
	}

	workflows.Register("test-workflow", factory, "A test workflow")

	got, exists := workflows.Get("test-workflow")
	if !exists {
		t.Fatal("Get() returned exists=false, want true")
	}

	if got == nil {
		t.Fatal("Get() returned nil factory")
	}
}

func TestGet_NotFound(t *testing.T) {
	_, exists := workflows.Get("nonexistent-workflow")
	if exists {
		t.Error("Get() returned exists=true for nonexistent workflow, want false")
	}
}

func TestList(t *testing.T) {
	workflows.Register("list-test-1", nil, "First test workflow")
	workflows.Register("list-test-2", nil, "Second test workflow")

	infos := workflows.List()

	if len(infos) < 2 {
		t.Fatalf("List() returned %d items, want at least 2", len(infos))
	}

	found1, found2 := false, false
	for _, info := range infos {
		if info.Name == "list-test-1" && info.Description == "First test workflow" {
			found1 = true
		}
		if info.Name == "list-test-2" && info.Description == "Second test workflow" {
			found2 = true
		}
	}

	if !found1 {
		t.Error("List() missing list-test-1")
	}

	if !found2 {
		t.Error("List() missing list-test-2")
	}
}
