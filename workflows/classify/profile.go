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
			"legibility": <0.0-1.0>,
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
- LEGIBILITY measures readability: 1.0 = text is perfectly readable, 0.0 = text is illegible
- FADED indicates visual appearance: true if marking appears washed out or pale
- IMPORTANT: A faded marking can still have high legibility if the text is readable
- clarity_score reflects overall page quality for marking detection
- Only suggest filter_suggestion if legibility < 0.4 AND you believe enhancement would improve readability
- JSON response only; no preamble or dialog`

const ClassificationSystemPrompt = `You are a document classification specialist. Analyze security marking detections across all pages to determine the overall document classification.

OUTPUT FORMAT: Respond with ONLY a JSON object matching this exact schema:
{
	"classification": "<overall classification level>",
	"alternative_readings": [
		{
			"classification": "<alternative classification>",
			"probability": <0.0-1.0>,
			"reason": "<brief phrase, max 15 words>"
		}
	],
	"marking_summary": ["<list of unique markings found>"],
	"rationale": "<1-2 sentences, max 40 words>"
}

INSTRUCTIONS:
- Analyze all marking detections provided
- Determine the HIGHEST classification level present
- IMPORTANT: ALL detected markings contribute to classification regardless of legibility or fading
- A faded or low-legibility marking is still a valid marking - include it in your classification decision
- Include detected caveats (NOFORN, ORCON, REL TO, etc.) in the primary classification (e.g., SECRET//NOFORN)
- Legibility and fading affect confidence scoring, NOT the classification itself
- Only list alternative readings if there is genuine ambiguity about what marking text says
- marking_summary should list unique marking texts (deduplicated)
- Keep rationale brief: 1-2 sentences explaining the key deciding factor
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
			"description": "<brief phrase, max 10 words>"
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
- detection_legibility (weight: 0.10): Average marking legibility (low legibility reduces confidence)

THRESHOLDS:
- >= 0.90: ACCEPT - Classification is reliable
- 0.70-0.89: REVIEW - Human verification recommended
- < 0.70: REJECT - Insufficient confidence

Keep factor descriptions brief (max 10 words each). JSON response only; no preamble or dialog`

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
