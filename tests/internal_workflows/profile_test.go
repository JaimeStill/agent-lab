package internal_workflows_test

import (
	"testing"

	"github.com/JaimeStill/agent-lab/internal/profiles"
	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
	"github.com/google/uuid"
)

func TestExtractAgentParams_FromStage(t *testing.T) {
	agentID := uuid.New()
	stage := &profiles.ProfileStage{
		StageName: "detect",
		AgentID:   &agentID,
	}

	s := state.New(nil)

	resultID, token, err := workflows.ExtractAgentParams(s, stage)

	if err != nil {
		t.Errorf("ExtractAgentParams() error = %v, want nil", err)
	}

	if resultID != agentID {
		t.Errorf("ExtractAgentParams() agentID = %v, want %v", resultID, agentID)
	}

	if token != "" {
		t.Errorf("ExtractAgentParams() token = %q, want empty", token)
	}
}

func TestExtractAgentParams_FromState(t *testing.T) {
	agentID := uuid.New()
	s := state.New(nil).Set("agent_id", agentID.String())

	resultID, token, err := workflows.ExtractAgentParams(s, nil)

	if err != nil {
		t.Errorf("ExtractAgentParams() error = %v, want nil", err)
	}

	if resultID != agentID {
		t.Errorf("ExtractAgentParams() agentID = %v, want %v", resultID, agentID)
	}

	if token != "" {
		t.Errorf("ExtractAgentParams() token = %q, want empty", token)
	}
}

func TestExtractAgentParams_StageOverridesState(t *testing.T) {
	stateAgentID := uuid.New()
	stageAgentID := uuid.New()

	stage := &profiles.ProfileStage{
		StageName: "detect",
		AgentID:   &stageAgentID,
	}

	s := state.New(nil).Set("agent_id", stateAgentID.String())

	resultID, _, err := workflows.ExtractAgentParams(s, stage)

	if err != nil {
		t.Errorf("ExtractAgentParams() error = %v, want nil", err)
	}

	if resultID != stageAgentID {
		t.Errorf("ExtractAgentParams() agentID = %v, want stage agentID %v", resultID, stageAgentID)
	}
}

func TestExtractAgentParams_StageWithoutAgentID_FallsBackToState(t *testing.T) {
	stateAgentID := uuid.New()

	stage := &profiles.ProfileStage{
		StageName: "detect",
		AgentID:   nil,
	}

	s := state.New(nil).Set("agent_id", stateAgentID.String())

	resultID, _, err := workflows.ExtractAgentParams(s, stage)

	if err != nil {
		t.Errorf("ExtractAgentParams() error = %v, want nil", err)
	}

	if resultID != stateAgentID {
		t.Errorf("ExtractAgentParams() agentID = %v, want state agentID %v", resultID, stateAgentID)
	}
}

func TestExtractAgentParams_NoAgentID_ReturnsError(t *testing.T) {
	s := state.New(nil)

	_, _, err := workflows.ExtractAgentParams(s, nil)

	if err == nil {
		t.Error("ExtractAgentParams() error = nil, want error")
	}
}

func TestExtractAgentParams_InvalidAgentID_ReturnsError(t *testing.T) {
	s := state.New(nil).Set("agent_id", "not-a-uuid")

	_, _, err := workflows.ExtractAgentParams(s, nil)

	if err == nil {
		t.Error("ExtractAgentParams() error = nil, want error")
	}
}

func TestExtractAgentParams_WithToken(t *testing.T) {
	agentID := uuid.New()
	s := state.New(nil).
		Set("agent_id", agentID.String()).
		SetSecret("token", "test-token")

	_, token, err := workflows.ExtractAgentParams(s, nil)

	if err != nil {
		t.Errorf("ExtractAgentParams() error = %v, want nil", err)
	}

	if token != "test-token" {
		t.Errorf("ExtractAgentParams() token = %q, want %q", token, "test-token")
	}
}
