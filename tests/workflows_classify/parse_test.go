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
			{"text": "SECRET", "location": "header", "legibility": 0.95, "faded": false}
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
		wantLegibility float64
	}{
		{
			name:           "clamps clarity above 1.0",
			input:          `{"page_number": 1, "markings_found": [{"text": "X", "location": "body", "legibility": 0.5, "faded": false}], "clarity_score": 1.5}`,
			wantClarity:    1.0,
			wantLegibility: 0.5,
		},
		{
			name:           "clamps clarity below 0.0",
			input:          `{"page_number": 1, "markings_found": [{"text": "X", "location": "body", "legibility": 0.5, "faded": false}], "clarity_score": -0.5}`,
			wantClarity:    0.0,
			wantLegibility: 0.5,
		},
		{
			name:           "clamps legibility above 1.0",
			input:          `{"page_number": 1, "markings_found": [{"text": "X", "location": "body", "legibility": 1.5, "faded": false}], "clarity_score": 0.8}`,
			wantClarity:    0.8,
			wantLegibility: 1.0,
		},
		{
			name:           "clamps legibility below 0.0",
			input:          `{"page_number": 1, "markings_found": [{"text": "X", "location": "body", "legibility": -0.5, "faded": false}], "clarity_score": 0.8}`,
			wantClarity:    0.8,
			wantLegibility: 0.0,
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

			if len(result.MarkingsFound) > 0 && result.MarkingsFound[0].Legibility != tt.wantLegibility {
				t.Errorf("Legibility = %f, want %f", result.MarkingsFound[0].Legibility, tt.wantLegibility)
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

func TestParseClassificationResponse_DirectJSON(t *testing.T) {
	input := `{
		"classification": "SECRET",
		"alternative_readings": [
			{"classification": "CONFIDENTIAL", "probability": 0.2, "reason": "Some markings unclear"}
		],
		"marking_summary": ["SECRET", "NOFORN"],
		"rationale": "Consistent SECRET markings across all pages"
	}`

	result, err := classify.ParseClassificationResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Classification != "SECRET" {
		t.Errorf("Classification = %q, want %q", result.Classification, "SECRET")
	}

	if len(result.AlternativeReadings) != 1 {
		t.Fatalf("AlternativeReadings length = %d, want 1", len(result.AlternativeReadings))
	}

	if result.AlternativeReadings[0].Classification != "CONFIDENTIAL" {
		t.Errorf("alternative classification = %q, want %q", result.AlternativeReadings[0].Classification, "CONFIDENTIAL")
	}

	if len(result.MarkingSummary) != 2 {
		t.Errorf("MarkingSummary length = %d, want 2", len(result.MarkingSummary))
	}
}

func TestParseClassificationResponse_MarkdownCodeBlock(t *testing.T) {
	input := "Here is the classification:\n\n```json\n" + `{
		"classification": "TOP SECRET",
		"marking_summary": ["TOP SECRET"],
		"rationale": "Clear TOP SECRET markings"
	}` + "\n```"

	result, err := classify.ParseClassificationResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Classification != "TOP SECRET" {
		t.Errorf("Classification = %q, want %q", result.Classification, "TOP SECRET")
	}
}

func TestParseClassificationResponse_InvalidJSON(t *testing.T) {
	input := "Not valid JSON"

	_, err := classify.ParseClassificationResponse(input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, classify.ErrParseResponse) {
		t.Errorf("error = %v, want ErrParseResponse", err)
	}
}

func TestParseClassificationResponse_ClampsProbability(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantProbability float64
	}{
		{
			name:            "clamps above 1.0",
			input:           `{"classification": "SECRET", "alternative_readings": [{"classification": "X", "probability": 1.5, "reason": "test"}], "marking_summary": [], "rationale": ""}`,
			wantProbability: 1.0,
		},
		{
			name:            "clamps below 0.0",
			input:           `{"classification": "SECRET", "alternative_readings": [{"classification": "X", "probability": -0.5, "reason": "test"}], "marking_summary": [], "rationale": ""}`,
			wantProbability: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := classify.ParseClassificationResponse(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.AlternativeReadings) > 0 && result.AlternativeReadings[0].Probability != tt.wantProbability {
				t.Errorf("Probability = %f, want %f", result.AlternativeReadings[0].Probability, tt.wantProbability)
			}
		})
	}
}

func TestParseScoringResponse_DirectJSON(t *testing.T) {
	input := `{
		"overall_score": 0.85,
		"factors": [
			{"name": "marking_clarity", "score": 0.9, "weight": 0.3, "description": "High clarity"}
		],
		"recommendation": "REVIEW"
	}`

	result, err := classify.ParseScoringResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.OverallScore != 0.85 {
		t.Errorf("OverallScore = %f, want 0.85", result.OverallScore)
	}

	if result.Recommendation != "REVIEW" {
		t.Errorf("Recommendation = %q, want %q", result.Recommendation, "REVIEW")
	}

	if len(result.Factors) != 1 {
		t.Fatalf("Factors length = %d, want 1", len(result.Factors))
	}

	if result.Factors[0].Name != "marking_clarity" {
		t.Errorf("factor name = %q, want %q", result.Factors[0].Name, "marking_clarity")
	}
}

func TestParseScoringResponse_MarkdownCodeBlock(t *testing.T) {
	input := "```json\n" + `{
		"overall_score": 0.92,
		"factors": [],
		"recommendation": "ACCEPT"
	}` + "\n```"

	result, err := classify.ParseScoringResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.OverallScore != 0.92 {
		t.Errorf("OverallScore = %f, want 0.92", result.OverallScore)
	}

	if result.Recommendation != "ACCEPT" {
		t.Errorf("Recommendation = %q, want %q", result.Recommendation, "ACCEPT")
	}
}

func TestParseScoringResponse_InvalidJSON(t *testing.T) {
	input := "Invalid"

	_, err := classify.ParseScoringResponse(input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, classify.ErrParseResponse) {
		t.Errorf("error = %v, want ErrParseResponse", err)
	}
}

func TestParseScoringResponse_ClampsValues(t *testing.T) {
	input := `{
		"overall_score": 1.5,
		"factors": [
			{"name": "test", "score": 2.0, "weight": -0.5, "description": ""}
		],
		"recommendation": "ACCEPT"
	}`

	result, err := classify.ParseScoringResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.OverallScore != 1.0 {
		t.Errorf("OverallScore = %f, want 1.0", result.OverallScore)
	}

	if result.Factors[0].Score != 1.0 {
		t.Errorf("factor score = %f, want 1.0", result.Factors[0].Score)
	}

	if result.Factors[0].Weight != 0.0 {
		t.Errorf("factor weight = %f, want 0.0", result.Factors[0].Weight)
	}
}

func TestParseScoringResponse_ComputesRecommendationWhenInvalid(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		wantRecommendation string
	}{
		{
			name:               "high score gets ACCEPT",
			input:              `{"overall_score": 0.95, "factors": [], "recommendation": "INVALID"}`,
			wantRecommendation: "ACCEPT",
		},
		{
			name:               "medium score gets REVIEW",
			input:              `{"overall_score": 0.80, "factors": [], "recommendation": ""}`,
			wantRecommendation: "REVIEW",
		},
		{
			name:               "low score gets REJECT",
			input:              `{"overall_score": 0.50, "factors": [], "recommendation": "UNKNOWN"}`,
			wantRecommendation: "REJECT",
		},
		{
			name:               "valid ACCEPT preserved",
			input:              `{"overall_score": 0.50, "factors": [], "recommendation": "ACCEPT"}`,
			wantRecommendation: "ACCEPT",
		},
		{
			name:               "valid REVIEW preserved",
			input:              `{"overall_score": 0.95, "factors": [], "recommendation": "REVIEW"}`,
			wantRecommendation: "REVIEW",
		},
		{
			name:               "valid REJECT preserved",
			input:              `{"overall_score": 0.95, "factors": [], "recommendation": "REJECT"}`,
			wantRecommendation: "REJECT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := classify.ParseScoringResponse(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Recommendation != tt.wantRecommendation {
				t.Errorf("Recommendation = %q, want %q", result.Recommendation, tt.wantRecommendation)
			}
		})
	}
}
