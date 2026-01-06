package classify

import (
	"encoding/json"

	"github.com/JaimeStill/agent-lab/internal/profiles"
)

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
- Note the location of each marking (header, footer, margin, or body)
- Assess confidence based on readability (1.0 = perfectly clear, 0.0 = illegible)
- Set faded=true if the marking appears washed out or hard to read
- clarity_score reflects overall page quality for marking detection
- If clarity_score < 0.7, suggest filter adjustment that might improve readability
- JSON response only; no preamble or dialog`

const ClassificationSystemPrompt = `You are a document classification specialist. Analyze security marking detections across all pages to determine the overall document classification.

OUTPUT FORMAT: Respond with ONLY a JSON object matching this exact schema:
{
	"classification": "<overall classification level>",
	"alternative_readings": [
		{
			"classification": "<alternative classification>",
			"probability": <0.0-1.0>,
			"reason": "<why this could be the correct classification>"
		}
	],
	"marking_summary": ["<list of unique markings found>"],
	"rationale": "<explanation of classification decision>"
}

INSTRUCTIONS:
- Analyze all marking detections provided
- Determine the HIGHEST classification level present
- If markings are inconsistent or ambiguous, list alternative readings
- marking_summary should list unique marking texts (deduplicated)
- rationale should explain how you determined the classification
- Consider marking confidence and consistency across pages
- JSON response only; no preamble or dialog`

const ScoringSystemPrompt = `You are a confidence scoring specialist. Evaluate the quality and reliability of document classification results.

OUTPUT FORMAT: Respond with ONLY a JSON object matching this exact schema:
{
	"overall_score": <0.0-1.0>,
	"factors": [
		{
			"name": "<factor name>",
			"score": <0.0-1.0>,
			"weight": <weight>,
			"description": "<explanation of score>"
		}
	],
	"recommendation": "<ACCEPT|REVIEW|REJECT>"
}

FACTORS TO EVALUATE:
- marking_clarity (weight: 0.30): Average clarity across pages
- marking_consistency (weight: 0.25): Marking agreement across pages
- spatial_coverage (weight: 0.15): Markings in expected locations (header/footer)
- enhancement_impact (weight: 0.10): Value added by enhancement (if applied)
- alternative_count (weight: 0.10): Fewer alternatives = higher confidence
- detection_confidence (weight: 0.10): Average marking confidence

THRESHOLDS:
- >= 0.90: ACCEPT - Classification is reliable
- 0.70-0.89: REVIEW - Human verification recommended
- < 0.70: REJECT - Insufficient confidence

JSON response only; no preamble or dialog`

// DefaultProfile returns the hardcoded default profile for the classify-docs workflow.
// It defines stages for init (no LLM) and detect (vision analysis).
func DefaultProfile() *profiles.ProfileWithStages {
	initPrompt := ""
	detectPrompt := DetectionSystemPrompt
	enhancePrompt := DetectionSystemPrompt
	classifyPrompt := ClassificationSystemPrompt
	scorePrompt := ScoringSystemPrompt

	enhanceOpts, _ := json.Marshal(DefaultEnhanceOptions())

	return profiles.NewProfileWithStages(
		profiles.ProfileStage{StageName: "init", SystemPrompt: &initPrompt},
		profiles.ProfileStage{StageName: "detect", SystemPrompt: &detectPrompt},
		profiles.ProfileStage{StageName: "enhance", SystemPrompt: &enhancePrompt, Options: enhanceOpts},
		profiles.ProfileStage{StageName: "classify", SystemPrompt: &classifyPrompt},
		profiles.ProfileStage{StageName: "score", SystemPrompt: &scorePrompt},
	)
}
