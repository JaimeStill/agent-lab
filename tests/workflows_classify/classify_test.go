package workflows_classify_test

import (
	"testing"

	"github.com/JaimeStill/agent-lab/workflows/classify"
	"github.com/google/uuid"
)

func TestPageDetection_NeedsEnhancement(t *testing.T) {
	brightness := 120
	contrast := 20

	tests := []struct {
		name      string
		detection classify.PageDetection
		threshold float64
		want      bool
	}{
		{
			name: "below threshold with suggestion",
			detection: classify.PageDetection{
				ClarityScore:     0.5,
				FilterSuggestion: &classify.FilterSuggestion{Brightness: &brightness},
			},
			threshold: 0.7,
			want:      true,
		},
		{
			name: "below threshold without suggestion",
			detection: classify.PageDetection{
				ClarityScore:     0.5,
				FilterSuggestion: nil,
			},
			threshold: 0.7,
			want:      false,
		},
		{
			name: "at threshold with suggestion",
			detection: classify.PageDetection{
				ClarityScore:     0.7,
				FilterSuggestion: &classify.FilterSuggestion{Contrast: &contrast},
			},
			threshold: 0.7,
			want:      false,
		},
		{
			name: "above threshold with suggestion",
			detection: classify.PageDetection{
				ClarityScore:     0.9,
				FilterSuggestion: &classify.FilterSuggestion{Brightness: &brightness},
			},
			threshold: 0.7,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.detection.NeedsEnhancement(tt.threshold)
			if got != tt.want {
				t.Errorf("NeedsEnhancement(%f) = %v, want %v", tt.threshold, got, tt.want)
			}
		})
	}
}

func TestPageImage_Fields(t *testing.T) {
	id := uuid.New()
	img := classify.PageImage{
		PageNumber: 5,
		ImageID:    id,
	}

	if img.PageNumber != 5 {
		t.Errorf("PageNumber = %d, want 5", img.PageNumber)
	}

	if img.ImageID != id {
		t.Errorf("ImageID = %v, want %v", img.ImageID, id)
	}
}

func TestMarkingInfo_Fields(t *testing.T) {
	m := classify.MarkingInfo{
		Text:       "SECRET//NOFORN",
		Location:   "header",
		Confidence: 0.95,
		Faded:      true,
	}

	if m.Text != "SECRET//NOFORN" {
		t.Errorf("Text = %q, want %q", m.Text, "SECRET//NOFORN")
	}

	if m.Location != "header" {
		t.Errorf("Location = %q, want %q", m.Location, "header")
	}

	if m.Confidence != 0.95 {
		t.Errorf("Confidence = %f, want 0.95", m.Confidence)
	}

	if !m.Faded {
		t.Error("Faded = false, want true")
	}
}

func TestFilterSuggestion_OptionalFields(t *testing.T) {
	brightness := 110
	contrast := 15

	fs := classify.FilterSuggestion{
		Brightness: &brightness,
		Contrast:   &contrast,
		Saturation: nil,
	}

	if fs.Brightness == nil || *fs.Brightness != 110 {
		t.Errorf("Brightness = %v, want 110", fs.Brightness)
	}

	if fs.Contrast == nil || *fs.Contrast != 15 {
		t.Errorf("Contrast = %v, want 15", fs.Contrast)
	}

	if fs.Saturation != nil {
		t.Errorf("Saturation = %v, want nil", fs.Saturation)
	}
}
