package workflows

import (
	"context"
	"fmt"

	"github.com/JaimeStill/agent-lab/internal/profiles"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
	"github.com/google/uuid"
)

// ExtractAgentParams extracts the agent ID and token from workflow state.
// If the stage has an AgentID configured, it takes precedence over state params.
// Returns the agent UUID, optional token string, and any error.
func ExtractAgentParams(s state.State, stage *profiles.ProfileStage) (uuid.UUID, string, error) {
	var agentID uuid.UUID

	if stage != nil && stage.AgentID != nil {
		agentID = *stage.AgentID
	} else {
		agentIDStr, ok := s.Get("agent_id")
		if !ok {
			return uuid.Nil, "", fmt.Errorf("agent_id is required")
		}
		var err error
		agentID, err = uuid.Parse(agentIDStr.(string))
		if err != nil {
			return uuid.Nil, "", fmt.Errorf("invalid agent_id: %w", err)
		}
	}

	tkn, _ := s.Get("token")
	token, _ := tkn.(string)

	return agentID, token, nil
}

// LoadProfile resolves the profile configuration for a workflow execution.
// If profile_id is provided in params, loads from database.
// Otherwise, returns the provided default profile.
func LoadProfile(ctx context.Context, rt *Runtime, params map[string]any, defaultProfile *profiles.ProfileWithStages) (*profiles.ProfileWithStages, error) {
	if profileIDStr, ok := params["profile_id"].(string); ok {
		profileID, err := uuid.Parse(profileIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid profile_id: %w", err)
		}
		return rt.Profiles().Find(ctx, profileID)
	}
	return defaultProfile, nil
}
