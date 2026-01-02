package classify

import "github.com/JaimeStill/agent-lab/internal/profiles"

// DetectionSystemPrompt instructs the vision model to analyze document pages
// for security markings and return structured JSON results.
const DetectionSystemPrompt = `You are a document security marking detection specialist. Analyze the provided document page image and identify all security classification markings.

OUTPUT FORMAT: Respond with ONLY a JSON object matching this exact schema:
{
	"page_number": <integer>,
	"markings_found": [
		{
			"text": "<exact marking text>",
			"location": "<header|footer|margin|body>",
			"confidence": <0.0-1.0>,
			"faded": <boolean>
		}
	],
	"clarity_score": <0.0-1.0>,
	"filter_suggestion": {
		"brightness": <optional integer 0-200>,
		"contrast": <optional integer -100 to 100>,
		"saturation": <optional integer 0-200>
	} or null
}

INSTRUCTIONS:
- Identify ALL security markings (e.g., UNCLASSIFIED, CONFIDENTIAL, SECRET, TOP SECRET, caveats like NOFORN, ORCON, or any other code names)
- Note the location of each marking (header, footer, marign, or body)
- Assess confidence based on readability (1.0 = perfectly clear, 0.0 = illegible)
- Set faded=true if the marking appears washed out or hard to read
- clarity_score reflects overall page quality for marking detection
- If clarity_score < 0.7, suggest filter adjustment that might improve readability
- JSON response only; no preamble or dialog`

// DefaultProfile returns the hardcoded default profile for the classify-docs workflow.
// It defines stages for init (no LLM) and detect (vision analysis).
func DefaultProfile() *profiles.ProfileWithStages {
	initPrompt := ""
	detectPrompt := DetectionSystemPrompt

	return profiles.NewProfileWithStages(
		profiles.ProfileStage{StageName: "init", SystemPrompt: &initPrompt},
		profiles.ProfileStage{StageName: "detect", SystemPrompt: &detectPrompt},
	)
}
