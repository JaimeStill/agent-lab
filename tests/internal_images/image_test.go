package internal_images_test

import (
	"errors"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/images"
	"github.com/JaimeStill/document-context/pkg/document"
	"github.com/google/uuid"
)

func TestParseImageFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    document.ImageFormat
		wantErr bool
	}{
		{"png lowercase", "png", document.PNG, false},
		{"PNG uppercase", "PNG", document.PNG, false},
		{"Png mixed case", "Png", document.PNG, false},
		{"jpg lowercase", "jpg", document.JPEG, false},
		{"JPG uppercase", "JPG", document.JPEG, false},
		{"jpeg lowercase", "jpeg", document.JPEG, false},
		{"JPEG uppercase", "JPEG", document.JPEG, false},
		{"empty string defaults to png", "", document.PNG, false},
		{"invalid format gif", "gif", "", true},
		{"invalid format bmp", "bmp", "", true},
		{"invalid format tiff", "tiff", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := images.ParseImageFormat(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseImageFormat() error = nil, want error")
				}
				if !errors.Is(err, images.ErrInvalidRenderOption) {
					t.Errorf("ParseImageFormat() error = %v, want ErrInvalidRenderOption", err)
				}
			} else {
				if err != nil {
					t.Errorf("ParseImageFormat() error = %v, want nil", err)
				}
				if got != tt.want {
					t.Errorf("ParseImageFormat() = %q, want %q", got, tt.want)
				}
			}
		})
	}
}

func TestRenderOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    images.RenderOptions
		wantErr bool
		errType error
	}{
		{
			"valid defaults",
			images.RenderOptions{},
			false, nil,
		},
		{
			"valid with pages",
			images.RenderOptions{Pages: "1-5"},
			false, nil,
		},
		{
			"valid png format",
			images.RenderOptions{Format: "png"},
			false, nil,
		},
		{
			"valid jpg format",
			images.RenderOptions{Format: "jpg"},
			false, nil,
		},
		{
			"valid dpi 72",
			images.RenderOptions{DPI: 72},
			false, nil,
		},
		{
			"valid dpi 1200",
			images.RenderOptions{DPI: 1200},
			false, nil,
		},
		{
			"invalid format",
			images.RenderOptions{Format: "gif"},
			true, images.ErrInvalidRenderOption,
		},
		{
			"dpi too low",
			images.RenderOptions{DPI: 50},
			true, images.ErrInvalidRenderOption,
		},
		{
			"dpi too high",
			images.RenderOptions{DPI: 1500},
			true, images.ErrInvalidRenderOption,
		},
		{
			"quality too low",
			images.RenderOptions{Quality: intPtr(0)},
			true, images.ErrInvalidRenderOption,
		},
		{
			"quality too high",
			images.RenderOptions{Quality: intPtr(101)},
			true, images.ErrInvalidRenderOption,
		},
		{
			"valid quality",
			images.RenderOptions{Quality: intPtr(90)},
			false, nil,
		},
		{
			"brightness too low",
			images.RenderOptions{Brightness: intPtr(-1)},
			true, images.ErrInvalidRenderOption,
		},
		{
			"brightness too high",
			images.RenderOptions{Brightness: intPtr(201)},
			true, images.ErrInvalidRenderOption,
		},
		{
			"valid brightness",
			images.RenderOptions{Brightness: intPtr(100)},
			false, nil,
		},
		{
			"contrast too low",
			images.RenderOptions{Contrast: intPtr(-101)},
			true, images.ErrInvalidRenderOption,
		},
		{
			"contrast too high",
			images.RenderOptions{Contrast: intPtr(101)},
			true, images.ErrInvalidRenderOption,
		},
		{
			"valid contrast",
			images.RenderOptions{Contrast: intPtr(0)},
			false, nil,
		},
		{
			"saturation too low",
			images.RenderOptions{Saturation: intPtr(-1)},
			true, images.ErrInvalidRenderOption,
		},
		{
			"saturation too high",
			images.RenderOptions{Saturation: intPtr(201)},
			true, images.ErrInvalidRenderOption,
		},
		{
			"valid saturation",
			images.RenderOptions{Saturation: intPtr(100)},
			false, nil,
		},
		{
			"rotation too low",
			images.RenderOptions{Rotation: intPtr(-1)},
			true, images.ErrInvalidRenderOption,
		},
		{
			"rotation too high",
			images.RenderOptions{Rotation: intPtr(361)},
			true, images.ErrInvalidRenderOption,
		},
		{
			"valid rotation",
			images.RenderOptions{Rotation: intPtr(90)},
			false, nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() error = nil, want error")
				}
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("Validate() error = %v, want %v", err, tt.errType)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestRenderOptions_Validate_Defaults(t *testing.T) {
	opts := images.RenderOptions{}
	err := opts.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if opts.Format != document.PNG {
		t.Errorf("Validate() Format = %q, want %q", opts.Format, document.PNG)
	}

	if opts.DPI != 300 {
		t.Errorf("Validate() DPI = %d, want 300", opts.DPI)
	}

	if opts.Background == nil {
		t.Error("Validate() Background = nil, want non-nil")
	} else if *opts.Background != "white" {
		t.Errorf("Validate() Background = %q, want %q", *opts.Background, "white")
	}
}

func TestRenderOptions_ToImage(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	docID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	bg := "white"
	quality := 90

	opts := images.RenderOptions{
		Format:     document.PNG,
		DPI:        300,
		Quality:    &quality,
		Background: &bg,
	}

	img := opts.ToImage(id, docID, 5, "images/test/image.png", 12345)

	if img.ID != id {
		t.Errorf("ToImage() ID = %v, want %v", img.ID, id)
	}
	if img.DocumentID != docID {
		t.Errorf("ToImage() DocumentID = %v, want %v", img.DocumentID, docID)
	}
	if img.PageNumber != 5 {
		t.Errorf("ToImage() PageNumber = %d, want 5", img.PageNumber)
	}
	if img.Format != document.PNG {
		t.Errorf("ToImage() Format = %q, want %q", img.Format, document.PNG)
	}
	if img.DPI != 300 {
		t.Errorf("ToImage() DPI = %d, want 300", img.DPI)
	}
	if img.Quality == nil || *img.Quality != 90 {
		t.Errorf("ToImage() Quality = %v, want 90", img.Quality)
	}
	if img.Background == nil || *img.Background != "white" {
		t.Errorf("ToImage() Background = %v, want white", img.Background)
	}
	if img.StorageKey != "images/test/image.png" {
		t.Errorf("ToImage() StorageKey = %q, want %q", img.StorageKey, "images/test/image.png")
	}
	if img.SizeBytes != 12345 {
		t.Errorf("ToImage() SizeBytes = %d, want 12345", img.SizeBytes)
	}
}

func TestRenderOptions_ToImageConfig(t *testing.T) {
	bg := "blue"
	brightness := 110
	contrast := 10
	saturation := 120
	rotation := 90
	quality := 85

	opts := images.RenderOptions{
		Format:     document.JPEG,
		DPI:        150,
		Quality:    &quality,
		Brightness: &brightness,
		Contrast:   &contrast,
		Saturation: &saturation,
		Rotation:   &rotation,
		Background: &bg,
	}

	cfg := opts.ToImageConfig()

	if cfg.Format != "jpg" {
		t.Errorf("ToImageConfig() Format = %q, want %q", cfg.Format, "jpg")
	}
	if cfg.DPI != 150 {
		t.Errorf("ToImageConfig() DPI = %d, want 150", cfg.DPI)
	}
	if cfg.Quality != 85 {
		t.Errorf("ToImageConfig() Quality = %d, want 85", cfg.Quality)
	}
	if cfg.Options["brightness"] != 110 {
		t.Errorf("ToImageConfig() Options[brightness] = %v, want 110", cfg.Options["brightness"])
	}
	if cfg.Options["contrast"] != 10 {
		t.Errorf("ToImageConfig() Options[contrast] = %v, want 10", cfg.Options["contrast"])
	}
	if cfg.Options["saturation"] != 120 {
		t.Errorf("ToImageConfig() Options[saturation] = %v, want 120", cfg.Options["saturation"])
	}
	if cfg.Options["rotation"] != 90 {
		t.Errorf("ToImageConfig() Options[rotation] = %v, want 90", cfg.Options["rotation"])
	}
	if cfg.Options["background"] != "blue" {
		t.Errorf("ToImageConfig() Options[background] = %v, want blue", cfg.Options["background"])
	}
}

func TestRenderOptions_ToImageConfig_JpegDefaultQuality(t *testing.T) {
	opts := images.RenderOptions{
		Format: document.JPEG,
		DPI:    300,
	}

	cfg := opts.ToImageConfig()

	if cfg.Quality != 90 {
		t.Errorf("ToImageConfig() JPEG default Quality = %d, want 90", cfg.Quality)
	}
}

func TestRenderOptions_ToImageConfig_PngNoQuality(t *testing.T) {
	opts := images.RenderOptions{
		Format: document.PNG,
		DPI:    300,
	}

	cfg := opts.ToImageConfig()

	if cfg.Quality != 0 {
		t.Errorf("ToImageConfig() PNG Quality = %d, want 0", cfg.Quality)
	}
}
