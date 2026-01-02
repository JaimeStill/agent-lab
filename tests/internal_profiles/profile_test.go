package internal_profiles_test

import (
	"testing"

	"github.com/JaimeStill/agent-lab/internal/profiles"
	"github.com/google/uuid"
)

func TestNewProfileWithStages(t *testing.T) {
	prompt1 := "First prompt"
	prompt2 := "Second prompt"

	tests := []struct {
		name       string
		stages     []profiles.ProfileStage
		wantLen    int
		wantStages []string
	}{
		{
			"no stages",
			nil,
			0,
			nil,
		},
		{
			"single stage",
			[]profiles.ProfileStage{
				{StageName: "summarize", SystemPrompt: &prompt1},
			},
			1,
			[]string{"summarize"},
		},
		{
			"multiple stages",
			[]profiles.ProfileStage{
				{StageName: "analyze", SystemPrompt: &prompt1},
				{StageName: "conclude", SystemPrompt: &prompt2},
			},
			2,
			[]string{"analyze", "conclude"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p *profiles.ProfileWithStages
			if len(tt.stages) == 0 {
				p = profiles.NewProfileWithStages()
			} else {
				p = profiles.NewProfileWithStages(tt.stages...)
			}

			if p == nil {
				t.Fatal("NewProfileWithStages() returned nil")
			}

			if len(p.Stages) != tt.wantLen {
				t.Errorf("NewProfileWithStages() stages len = %d, want %d", len(p.Stages), tt.wantLen)
			}

			for i, wantName := range tt.wantStages {
				if p.Stages[i].StageName != wantName {
					t.Errorf("stage[%d].StageName = %q, want %q", i, p.Stages[i].StageName, wantName)
				}
			}
		})
	}
}

func TestProfileWithStages_Stage(t *testing.T) {
	prompt1 := "First prompt"
	prompt2 := "Second prompt"
	agentID := uuid.New()

	p := profiles.NewProfileWithStages(
		profiles.ProfileStage{StageName: "analyze", SystemPrompt: &prompt1},
		profiles.ProfileStage{StageName: "conclude", SystemPrompt: &prompt2, AgentID: &agentID},
	)

	tests := []struct {
		name           string
		stageName      string
		wantNil        bool
		wantPrompt     string
		wantHasAgentID bool
	}{
		{
			"existing stage without agent",
			"analyze",
			false,
			"First prompt",
			false,
		},
		{
			"existing stage with agent",
			"conclude",
			false,
			"Second prompt",
			true,
		},
		{
			"non-existent stage",
			"nonexistent",
			true,
			"",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stage := p.Stage(tt.stageName)

			if tt.wantNil {
				if stage != nil {
					t.Errorf("Stage(%q) = %v, want nil", tt.stageName, stage)
				}
				return
			}

			if stage == nil {
				t.Fatalf("Stage(%q) = nil, want non-nil", tt.stageName)
			}

			if stage.StageName != tt.stageName {
				t.Errorf("Stage(%q).StageName = %q, want %q", tt.stageName, stage.StageName, tt.stageName)
			}

			if stage.SystemPrompt == nil {
				t.Errorf("Stage(%q).SystemPrompt = nil, want non-nil", tt.stageName)
			} else if *stage.SystemPrompt != tt.wantPrompt {
				t.Errorf("Stage(%q).SystemPrompt = %q, want %q", tt.stageName, *stage.SystemPrompt, tt.wantPrompt)
			}

			hasAgentID := stage.AgentID != nil
			if hasAgentID != tt.wantHasAgentID {
				t.Errorf("Stage(%q) hasAgentID = %v, want %v", tt.stageName, hasAgentID, tt.wantHasAgentID)
			}
		})
	}
}

func TestProfileWithStages_Stage_ReturnsPointerToSliceElement(t *testing.T) {
	prompt := "Original prompt"
	p := profiles.NewProfileWithStages(
		profiles.ProfileStage{StageName: "test", SystemPrompt: &prompt},
	)

	stage := p.Stage("test")
	if stage == nil {
		t.Fatal("Stage() returned nil")
	}

	newPrompt := "Modified prompt"
	stage.SystemPrompt = &newPrompt

	stage2 := p.Stage("test")
	if stage2.SystemPrompt == nil || *stage2.SystemPrompt != "Modified prompt" {
		t.Error("Stage() should return pointer to actual slice element, allowing modification")
	}
}
