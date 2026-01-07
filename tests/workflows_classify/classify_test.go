package workflows_classify_test

import (
	"testing"

	"github.com/JaimeStill/agent-lab/workflows/classify"
	"github.com/google/uuid"
)

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
		Legibility: 0.95,
		Faded:      true,
	}

	if m.Text != "SECRET//NOFORN" {
		t.Errorf("Text = %q, want %q", m.Text, "SECRET//NOFORN")
	}

	if m.Location != "header" {
		t.Errorf("Location = %q, want %q", m.Location, "header")
	}

	if m.Legibility != 0.95 {
		t.Errorf("Legibility = %f, want 0.95", m.Legibility)
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
