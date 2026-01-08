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

func TestProfileWithStages_Merge_NilOther(t *testing.T) {
	prompt := "Base prompt"
	base := profiles.NewProfileWithStages(
		profiles.ProfileStage{StageName: "detect", SystemPrompt: &prompt},
	)

	result := base.Merge(nil)

	if result != base {
		t.Error("Merge(nil) should return original profile")
	}
}

func TestProfileWithStages_Merge_EmptyOther(t *testing.T) {
	basePrompt := "Base prompt"
	base := profiles.NewProfileWithStages(
		profiles.ProfileStage{StageName: "detect", SystemPrompt: &basePrompt},
		profiles.ProfileStage{StageName: "classify", SystemPrompt: &basePrompt},
	)

	other := profiles.NewProfileWithStages()

	result := base.Merge(other)

	if len(result.Stages) != 2 {
		t.Errorf("Merge() stages len = %d, want 2", len(result.Stages))
	}

	if result.Stage("detect") == nil || result.Stage("classify") == nil {
		t.Error("Merge() should preserve base stages when other is empty")
	}
}

func TestProfileWithStages_Merge_OverrideStages(t *testing.T) {
	basePrompt := "Base prompt"
	overridePrompt := "Override prompt"

	base := profiles.NewProfileWithStages(
		profiles.ProfileStage{StageName: "detect", SystemPrompt: &basePrompt},
		profiles.ProfileStage{StageName: "classify", SystemPrompt: &basePrompt},
	)

	other := profiles.NewProfileWithStages(
		profiles.ProfileStage{StageName: "detect", SystemPrompt: &overridePrompt},
	)

	result := base.Merge(other)

	if len(result.Stages) != 2 {
		t.Errorf("Merge() stages len = %d, want 2", len(result.Stages))
	}

	detectStage := result.Stage("detect")
	if detectStage == nil {
		t.Fatal("detect stage is nil")
	}
	if *detectStage.SystemPrompt != "Override prompt" {
		t.Errorf("detect stage prompt = %q, want %q", *detectStage.SystemPrompt, "Override prompt")
	}

	classifyStage := result.Stage("classify")
	if classifyStage == nil {
		t.Fatal("classify stage is nil")
	}
	if *classifyStage.SystemPrompt != "Base prompt" {
		t.Errorf("classify stage prompt = %q, want %q", *classifyStage.SystemPrompt, "Base prompt")
	}
}

func TestProfileWithStages_Merge_AddsNewStages(t *testing.T) {
	basePrompt := "Base prompt"
	newPrompt := "New prompt"

	base := profiles.NewProfileWithStages(
		profiles.ProfileStage{StageName: "detect", SystemPrompt: &basePrompt},
	)

	other := profiles.NewProfileWithStages(
		profiles.ProfileStage{StageName: "score", SystemPrompt: &newPrompt},
	)

	result := base.Merge(other)

	if len(result.Stages) != 2 {
		t.Errorf("Merge() stages len = %d, want 2", len(result.Stages))
	}

	if result.Stage("detect") == nil {
		t.Error("detect stage should be preserved from base")
	}

	if result.Stage("score") == nil {
		t.Error("score stage should be added from other")
	}
}

func TestProfileWithStages_Merge_UsesOtherMetadata(t *testing.T) {
	basePrompt := "Base prompt"

	base := &profiles.ProfileWithStages{
		Profile: profiles.Profile{
			ID:           uuid.New(),
			WorkflowName: "base-workflow",
			Name:         "base-profile",
		},
		Stages: []profiles.ProfileStage{
			{StageName: "detect", SystemPrompt: &basePrompt},
		},
	}

	otherID := uuid.New()
	other := &profiles.ProfileWithStages{
		Profile: profiles.Profile{
			ID:           otherID,
			WorkflowName: "other-workflow",
			Name:         "other-profile",
		},
		Stages: []profiles.ProfileStage{},
	}

	result := base.Merge(other)

	if result.ID != otherID {
		t.Error("Merge() should use other's ID")
	}

	if result.WorkflowName != "other-workflow" {
		t.Errorf("Merge() WorkflowName = %q, want %q", result.WorkflowName, "other-workflow")
	}

	if result.Name != "other-profile" {
		t.Errorf("Merge() Name = %q, want %q", result.Name, "other-profile")
	}
}
