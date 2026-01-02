package classify

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
)

var jsonBlockRegex = regexp.MustCompile(`(?s)` + "```" + `(?:json)?\s*\n?(.*?)\n?` + "```")

// ParseDetectionResponse parses an LLM response into a PageDetection struct.
// It first attempts direct JSON unmarshaling, then falls back to extracting
// JSON from markdown code blocks. Values are clamped to valid ranges.
func ParseDetectionResponse(content string) (PageDetection, error) {
	var detection PageDetection

	content = strings.TrimSpace(content)
	if err := json.Unmarshal([]byte(content), &detection); err == nil {
		return validateDetection(detection)
	}

	matches := jsonBlockRegex.FindStringSubmatch(content)
	if len(matches) >= 2 {
		cleaned := strings.TrimSpace(matches[1])
		if err := json.Unmarshal([]byte(cleaned), &detection); err == nil {
			return validateDetection(detection)
		}
	}

	return PageDetection{}, fmt.Errorf("%w: could not parse JSON from response", ErrParseResponse)
}

func validateDetection(d PageDetection) (PageDetection, error) {
	d.ClarityScore = math.Max(0.0, math.Min(1.0, d.ClarityScore))

	for i := range d.MarkingsFound {
		d.MarkingsFound[i].Confidence = math.Max(0.0, math.Min(1.0, d.MarkingsFound[i].Confidence))
	}

	return d, nil
}
