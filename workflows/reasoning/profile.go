package reasoning

import "github.com/JaimeStill/agent-lab/internal/profiles"

func DefaultProfile() *profiles.ProfileWithStages {
	analyzePrompt := "You are an analytical assistant. Break down problems into their components and identify the important elements."
	reasonPrompt := "You are a logical reasoning assistant. Think step-by-step and explain your reasoning clearly."
	concludePrompt := "You are a concise assistant. Provide clear, direct conclusions based on the reasoning provided."

	return profiles.NewProfileWithStages(
		profiles.ProfileStage{StageName: "analyze", SystemPrompt: &analyzePrompt},
		profiles.ProfileStage{StageName: "reason", SystemPrompt: &reasonPrompt},
		profiles.ProfileStage{StageName: "conclude", SystemPrompt: &concludePrompt},
	)
}
