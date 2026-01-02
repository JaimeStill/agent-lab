package workflows_classify_test

import (
	"errors"
	"testing"

	"github.com/JaimeStill/agent-lab/workflows/classify"
)

func TestParseDetectionResponse_DirectJSON(t *testing.T) {
	input := `{
		"page_number": 1,
		"markings_found": [
			{"text": "SECRET", "location": "header", "confidence": 0.95, "faded": false}
		],
		"clarity_score": 0.85,
		"filter_suggestion": null
	}`

	result, err := classify.ParseDetectionResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PageNumber != 1 {
		t.Errorf("PageNumber = %d, want 1", result.PageNumber)
	}

	if len(result.MarkingsFound) != 1 {
		t.Fatalf("MarkingsFound length = %d, want 1", len(result.MarkingsFound))
	}

	if result.MarkingsFound[0].Text != "SECRET" {
		t.Errorf("marking text = %q, want %q", result.MarkingsFound[0].Text, "SECRET")
	}

	if result.ClarityScore != 0.85 {
		t.Errorf("ClarityScore = %f, want 0.85", result.ClarityScore)
	}
}

func TestParseDetectionResponse_MarkdownCodeBlock(t *testing.T) {
	input := "Here is the analysis:\n\n```json\n" + `{
		"page_number": 2,
		"markings_found": [],
		"clarity_score": 0.9
	}` + "\n```\n\nLet me know if you need more details."

	result, err := classify.ParseDetectionResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PageNumber != 2 {
		t.Errorf("PageNumber = %d, want 2", result.PageNumber)
	}

	if result.ClarityScore != 0.9 {
		t.Errorf("ClarityScore = %f, want 0.9", result.ClarityScore)
	}
}

func TestParseDetectionResponse_MarkdownCodeBlockNoLanguage(t *testing.T) {
	input := "```\n" + `{"page_number": 3, "markings_found": [], "clarity_score": 0.7}` + "\n```"

	result, err := classify.ParseDetectionResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PageNumber != 3 {
		t.Errorf("PageNumber = %d, want 3", result.PageNumber)
	}
}

func TestParseDetectionResponse_InvalidJSON(t *testing.T) {
	input := "This is not JSON at all"

	_, err := classify.ParseDetectionResponse(input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, classify.ErrParseResponse) {
		t.Errorf("error = %v, want ErrParseResponse", err)
	}
}

func TestParseDetectionResponse_ClampsValues(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantClarity    float64
		wantConfidence float64
	}{
		{
			name:           "clamps clarity above 1.0",
			input:          `{"page_number": 1, "markings_found": [{"text": "X", "location": "body", "confidence": 0.5, "faded": false}], "clarity_score": 1.5}`,
			wantClarity:    1.0,
			wantConfidence: 0.5,
		},
		{
			name:           "clamps clarity below 0.0",
			input:          `{"page_number": 1, "markings_found": [{"text": "X", "location": "body", "confidence": 0.5, "faded": false}], "clarity_score": -0.5}`,
			wantClarity:    0.0,
			wantConfidence: 0.5,
		},
		{
			name:           "clamps confidence above 1.0",
			input:          `{"page_number": 1, "markings_found": [{"text": "X", "location": "body", "confidence": 1.5, "faded": false}], "clarity_score": 0.8}`,
			wantClarity:    0.8,
			wantConfidence: 1.0,
		},
		{
			name:           "clamps confidence below 0.0",
			input:          `{"page_number": 1, "markings_found": [{"text": "X", "location": "body", "confidence": -0.5, "faded": false}], "clarity_score": 0.8}`,
			wantClarity:    0.8,
			wantConfidence: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := classify.ParseDetectionResponse(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.ClarityScore != tt.wantClarity {
				t.Errorf("ClarityScore = %f, want %f", result.ClarityScore, tt.wantClarity)
			}

			if len(result.MarkingsFound) > 0 && result.MarkingsFound[0].Confidence != tt.wantConfidence {
				t.Errorf("Confidence = %f, want %f", result.MarkingsFound[0].Confidence, tt.wantConfidence)
			}
		})
	}
}

func TestParseDetectionResponse_WithFilterSuggestion(t *testing.T) {
	input := `{
		"page_number": 1,
		"markings_found": [],
		"clarity_score": 0.5,
		"filter_suggestion": {"brightness": 120, "contrast": 20}
	}`

	result, err := classify.ParseDetectionResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.FilterSuggestion == nil {
		t.Fatal("FilterSuggestion is nil, want non-nil")
	}

	if result.FilterSuggestion.Brightness == nil || *result.FilterSuggestion.Brightness != 120 {
		t.Errorf("Brightness = %v, want 120", result.FilterSuggestion.Brightness)
	}

	if result.FilterSuggestion.Contrast == nil || *result.FilterSuggestion.Contrast != 20 {
		t.Errorf("Contrast = %v, want 20", result.FilterSuggestion.Contrast)
	}
}

func TestParseDetectionResponse_WhitespaceHandling(t *testing.T) {
	input := "   \n\t  " + `{"page_number": 1, "markings_found": [], "clarity_score": 0.8}` + "   \n"

	result, err := classify.ParseDetectionResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PageNumber != 1 {
		t.Errorf("PageNumber = %d, want 1", result.PageNumber)
	}
}
