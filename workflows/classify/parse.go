package classify

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
)

var jsonBlockRegex = regexp.MustCompile(`(?s)` + "```" + `(?:json)?\s*\n?(.*?)\n?` + "```")

// ParseClassificationResponse parses an LLM response into a ClassificationResult.
// It first attempts direct JSON unmarshaling, then falls back to extracting
// JSON from markdown code blocks. Probability values are clamped to [0,1].
func ParseClassificationResponse(content string) (ClassificationResult, error) {
	return parseResponse(content, validateClassification, "could not parse classification JSON")
}

// ParseDetectionResponse parses an LLM response into a PageDetection struct.
// It first attempts direct JSON unmarshaling, then falls back to extracting
// JSON from markdown code blocks. Values are clamped to valid ranges.
func ParseDetectionResponse(content string) (PageDetection, error) {
	return parseResponse(content, validateDetection, "could not parse detection JSON")
}

// ParseScoringResponse parses an LLM response into a ConfidenceAssessment.
// It first attempts direct JSON unmarshaling, then falls back to extracting
// JSON from markdown code blocks. Scores and weights are clamped to [0,1].
// Invalid recommendations are replaced based on overall score thresholds.
func ParseScoringResponse(content string) (ConfidenceAssessment, error) {
	return parseResponse(content, validateScoring, "could not parse scoring JSON")
}

func clamp(value float64, min float64, max float64) float64 {
	return math.Max(min, math.Min(max, value))
}

func computeRecommendation(score float64) string {
	switch {
	case score >= 0.90:
		return "ACCEPT"
	case score >= 0.70:
		return "REVIEW"
	default:
		return "REJECT"
	}
}

func parseResponse[T any](content string, validate func(T) T, errMsg string) (T, error) {
	var result T
	content = strings.TrimSpace(content)
	if err := json.Unmarshal([]byte(content), &result); err == nil {
		return validate(result), nil
	}

	matches := jsonBlockRegex.FindStringSubmatch(content)
	if len(matches) >= 2 {
		cleaned := strings.TrimSpace(matches[1])
		if err := json.Unmarshal([]byte(cleaned), &result); err == nil {
			return validate(result), nil
		}
	}

	return result, fmt.Errorf("%w: %s", ErrParseResponse, errMsg)
}

func validateClassification(c ClassificationResult) ClassificationResult {
	for i := range c.AlternativeReadings {
		c.AlternativeReadings[i].Probability = clamp(c.AlternativeReadings[i].Probability, 0.0, 1.0)
	}
	return c
}

func validateDetection(d PageDetection) PageDetection {
	d.ClarityScore = clamp(d.ClarityScore, 0.0, 1.0)

	for i := range d.MarkingsFound {
		d.MarkingsFound[i].Confidence = clamp(d.MarkingsFound[i].Confidence, 0.0, 1.0)
	}

	return d
}

func validateScoring(a ConfidenceAssessment) ConfidenceAssessment {
	a.OverallScore = clamp(a.OverallScore, 0.0, 1.0)

	for i := range a.Factors {
		a.Factors[i].Score = clamp(a.Factors[i].Score, 0.0, 1.0)
		a.Factors[i].Weight = clamp(a.Factors[i].Weight, 0.0, 1.0)
	}

	switch a.Recommendation {
	case "ACCEPT", "REVIEW", "REJECT":
	default:
		a.Recommendation = computeRecommendation(a.OverallScore)
	}

	return a
}
