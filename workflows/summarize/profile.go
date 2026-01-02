package summarize

import "github.com/JaimeStill/agent-lab/internal/profiles"

func DefaultProfile() *profiles.ProfileWithStages {
	summarizePrompt := "You are a concise summarization assistant. Provide clear, brief summaries that capture the key points."

	return profiles.NewProfileWithStages(
		profiles.ProfileStage{
			StageName:    "summarize",
			SystemPrompt: &summarizePrompt,
		},
	)
}
