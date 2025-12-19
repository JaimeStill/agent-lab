package internal_workflows_test

import (
	"encoding/json"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/workflows"
)

func TestRunStatus_Values(t *testing.T) {
	tests := []struct {
		status workflows.RunStatus
		want   string
	}{
		{workflows.StatusPending, "pending"},
		{workflows.StatusRunning, "running"},
		{workflows.StatusCompleted, "completed"},
		{workflows.StatusFailed, "failed"},
		{workflows.StatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("RunStatus = %q, want %q", tt.status, tt.want)
			}
		})
	}
}

func TestStageStatus_Values(t *testing.T) {
	tests := []struct {
		status workflows.StageStatus
		want   string
	}{
		{workflows.StageStarted, "started"},
		{workflows.StageCompleted, "completed"},
		{workflows.StageFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("StageStatus = %q, want %q", tt.status, tt.want)
			}
		})
	}
}

func TestWorkflowInfo_JSON(t *testing.T) {
	info := workflows.WorkflowInfo{
		Name:        "test-workflow",
		Description: "A test workflow",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got workflows.WorkflowInfo
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got.Name != info.Name {
		t.Errorf("Name = %q, want %q", got.Name, info.Name)
	}

	if got.Description != info.Description {
		t.Errorf("Description = %q, want %q", got.Description, info.Description)
	}
}

func TestExecutionEventType_Values(t *testing.T) {
	tests := []struct {
		eventType workflows.ExecutionEventType
		want      string
	}{
		{workflows.EventStageStart, "stage.start"},
		{workflows.EventStageComplete, "stage.complete"},
		{workflows.EventDecision, "decision"},
		{workflows.EventError, "error"},
		{workflows.EventComplete, "complete"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.eventType) != tt.want {
				t.Errorf("ExecutionEventType = %q, want %q", tt.eventType, tt.want)
			}
		})
	}
}

func TestExecutionEvent_JSON(t *testing.T) {
	event := workflows.ExecutionEvent{
		Type: workflows.EventStageStart,
		Data: map[string]any{
			"node_name": "test-node",
			"iteration": 0,
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got workflows.ExecutionEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got.Type != event.Type {
		t.Errorf("Type = %q, want %q", got.Type, event.Type)
	}

	if got.Data["node_name"] != "test-node" {
		t.Errorf("Data[node_name] = %v, want %q", got.Data["node_name"], "test-node")
	}
}

func TestNodeStartData_JSON(t *testing.T) {
	data := workflows.NodeStartData{
		Node:      "test-node",
		Iteration: 1,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got workflows.NodeStartData
	if err := json.Unmarshal(jsonData, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got.Node != data.Node {
		t.Errorf("Node = %q, want %q", got.Node, data.Node)
	}

	if got.Iteration != data.Iteration {
		t.Errorf("Iteration = %d, want %d", got.Iteration, data.Iteration)
	}
}

func TestNodeCompleteData_JSON(t *testing.T) {
	data := workflows.NodeCompleteData{
		Node:         "test-node",
		Iteration:    2,
		Error:        true,
		ErrorMessage: "node failed",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got workflows.NodeCompleteData
	if err := json.Unmarshal(jsonData, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got.Node != data.Node {
		t.Errorf("Node = %q, want %q", got.Node, data.Node)
	}

	if got.Error != data.Error {
		t.Errorf("Error = %v, want %v", got.Error, data.Error)
	}

	if got.ErrorMessage != data.ErrorMessage {
		t.Errorf("ErrorMessage = %q, want %q", got.ErrorMessage, data.ErrorMessage)
	}
}

func TestEdgeTransitionData_JSON(t *testing.T) {
	data := workflows.EdgeTransitionData{
		From:            "node-a",
		To:              "node-b",
		PredicateName:   "is-valid",
		PredicateResult: true,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got workflows.EdgeTransitionData
	if err := json.Unmarshal(jsonData, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got.From != data.From {
		t.Errorf("From = %q, want %q", got.From, data.From)
	}

	if got.To != data.To {
		t.Errorf("To = %q, want %q", got.To, data.To)
	}

	if got.PredicateResult != data.PredicateResult {
		t.Errorf("PredicateResult = %v, want %v", got.PredicateResult, data.PredicateResult)
	}
}
