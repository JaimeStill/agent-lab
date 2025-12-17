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
