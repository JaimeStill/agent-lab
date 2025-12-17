package workflows

import (
	"context"
	"sync"

	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
)

type WorkflowFactory func(ctx context.Context, systems *Systems, params map[string]any) (state.StateGraph, state.State, error)

type workflowRegistry struct {
	factories map[string]WorkflowFactory
	info      map[string]WorkflowInfo
	mu        sync.RWMutex
}

var registry = &workflowRegistry{
	factories: make(map[string]WorkflowFactory),
	info:      make(map[string]WorkflowInfo),
}

func Register(name string, factory WorkflowFactory, description string) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.factories[name] = factory
	registry.info[name] = WorkflowInfo{Name: name, Description: description}
}

func Get(name string) (WorkflowFactory, bool) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	factory, exists := registry.factories[name]
	return factory, exists
}

func List() []WorkflowInfo {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	result := make([]WorkflowInfo, 0, len(registry.info))
	for _, info := range registry.info {
		result = append(result, info)
	}
	return result
}
