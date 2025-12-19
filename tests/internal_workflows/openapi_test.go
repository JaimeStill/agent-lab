package internal_workflows_test

import (
	"testing"

	"github.com/JaimeStill/agent-lab/internal/workflows"
)

func TestSpec_Operations(t *testing.T) {
	tests := []struct {
		name      string
		operation any
	}{
		{"ListWorkflows", workflows.Spec.ListWorkflows},
		{"Execute", workflows.Spec.Execute},
		{"ExecuteStream", workflows.Spec.ExecuteStream},
		{"ListRuns", workflows.Spec.ListRuns},
		{"FindRun", workflows.Spec.FindRun},
		{"GetStages", workflows.Spec.GetStages},
		{"GetDecisions", workflows.Spec.GetDecisions},
		{"Cancel", workflows.Spec.Cancel},
		{"Resume", workflows.Spec.Resume},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.operation == nil {
				t.Errorf("Spec.%s is nil", tt.name)
			}
		})
	}
}

func TestSpec_Schemas(t *testing.T) {
	schemas := workflows.Spec.Schemas()

	expectedSchemas := []string{
		"WorkflowInfo",
		"WorkflowInfoList",
		"Run",
		"RunPageResult",
		"Stage",
		"StageList",
		"Decision",
		"DecisionList",
		"ExecuteRequest",
		"ExecutionEvent",
	}

	for _, name := range expectedSchemas {
		t.Run(name, func(t *testing.T) {
			if schemas[name] == nil {
				t.Errorf("Schemas()[%q] is nil", name)
			}
		})
	}
}

func TestSpec_ExecuteStream_SSEResponse(t *testing.T) {
	op := workflows.Spec.ExecuteStream

	if op.Responses == nil {
		t.Fatal("ExecuteStream.Responses is nil")
	}

	response200, ok := op.Responses[200]
	if !ok {
		t.Fatal("ExecuteStream.Responses[200] not found")
	}

	if response200.Content == nil {
		t.Fatal("ExecuteStream.Responses[200].Content is nil")
	}

	sseContent, ok := response200.Content["text/event-stream"]
	if !ok {
		t.Error("ExecuteStream.Responses[200].Content[text/event-stream] not found")
	}

	if sseContent.Schema == nil {
		t.Error("SSE content schema is nil")
	}
}

func TestSpec_RunSchema_StatusEnum(t *testing.T) {
	schemas := workflows.Spec.Schemas()

	runSchema := schemas["Run"]
	if runSchema == nil {
		t.Fatal("Run schema not found")
	}

	if runSchema.Properties == nil {
		t.Fatal("Run.Properties is nil")
	}

	statusProp := runSchema.Properties["status"]
	if statusProp == nil {
		t.Fatal("Run.Properties[status] is nil")
	}

	if len(statusProp.Enum) == 0 {
		t.Error("Run.Properties[status].Enum is empty")
	}

	expectedStatuses := map[any]bool{
		"pending":   false,
		"running":   false,
		"completed": false,
		"failed":    false,
		"cancelled": false,
	}

	for _, v := range statusProp.Enum {
		if _, ok := expectedStatuses[v]; ok {
			expectedStatuses[v] = true
		}
	}

	for status, found := range expectedStatuses {
		if !found {
			t.Errorf("Run.Properties[status].Enum missing %q", status)
		}
	}
}

func TestSpec_ExecutionEventSchema_TypeEnum(t *testing.T) {
	schemas := workflows.Spec.Schemas()

	eventSchema := schemas["ExecutionEvent"]
	if eventSchema == nil {
		t.Fatal("ExecutionEvent schema not found")
	}

	if eventSchema.Properties == nil {
		t.Fatal("ExecutionEvent.Properties is nil")
	}

	typeProp := eventSchema.Properties["type"]
	if typeProp == nil {
		t.Fatal("ExecutionEvent.Properties[type] is nil")
	}

	if len(typeProp.Enum) == 0 {
		t.Error("ExecutionEvent.Properties[type].Enum is empty")
	}

	expectedTypes := map[any]bool{
		"stage.start":    false,
		"stage.complete": false,
		"decision":       false,
		"error":          false,
		"complete":       false,
	}

	for _, v := range typeProp.Enum {
		if _, ok := expectedTypes[v]; ok {
			expectedTypes[v] = true
		}
	}

	for eventType, found := range expectedTypes {
		if !found {
			t.Errorf("ExecutionEvent.Properties[type].Enum missing %q", eventType)
		}
	}
}
