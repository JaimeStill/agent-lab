package images

import (
	"fmt"
	"time"

	"github.com/JaimeStill/document-context/pkg/config"
	"github.com/JaimeStill/document-context/pkg/document"
	"github.com/google/uuid"
)

// Image represents a rendered document page stored in the system.
type Image struct {
	ID         uuid.UUID            `json:"id"`
	DocumentID uuid.UUID            `json:"document_id"`
	PageNumber int                  `json:"page_number"`
	Format     document.ImageFormat `json:"format"`
	DPI        int                  `json:"dpi"`
	Quality    *int                 `json:"quality,omitempty"`
	Brightness *int                 `json:"brightness,omitempty"`
	Contrast   *int                 `json:"contrast,omitempty"`
	Saturation *int                 `json:"saturation,omitempty"`
	Rotation   *int                 `json:"rotation,omitempty"`
	Background *string              `json:"background,omitempty"`
	StorageKey string               `json:"storage_key"`
	SizeBytes  int64                `json:"size_bytes"`
	CreatedAt  time.Time            `json:"created_at"`
}

// RenderOptions specifies parameters for rendering document pages to images.
type RenderOptions struct {
	Pages      string               `json:"pages"`
	Format     document.ImageFormat `json:"format"`
	DPI        int                  `json:"dpi"`
	Quality    *int                 `json:"quality,omitempty"`
	Brightness *int                 `json:"brightness,omitempty"`
	Contrast   *int                 `json:"contrast,omitempty"`
	Saturation *int                 `json:"saturation,omitempty"`
	Rotation   *int                 `json:"rotation,omitempty"`
	Background *string              `json:"background,omitempty"`
	Force      bool                 `json:"force"`
}

// Validate validates and applies defaults to render options.
// It ensures all values are within acceptable ranges and sets
// default values for unspecified options.
func (o *RenderOptions) Validate() error {
	format, err := document.ParseImageFormat(string(o.Format))
	if err != nil {
		return fmt.Errorf("%w: format must be 'png' or 'jpg'", ErrInvalidRenderOption)
	}
	o.Format = format

	if o.DPI == 0 {
		o.DPI = 300
	} else if o.DPI < 72 || o.DPI > 1200 {
		return fmt.Errorf("%w: dpi must be between 72 and 1200", ErrInvalidRenderOption)
	}

	if o.Quality != nil && (*o.Quality < 1 || *o.Quality > 100) {
		return fmt.Errorf("%w: quality must be between 1 and 100", ErrInvalidRenderOption)
	}

	if o.Brightness != nil && (*o.Brightness < 0 || *o.Brightness > 200) {
		return fmt.Errorf("%w: brightness must be between 0 and 200", ErrInvalidRenderOption)
	}

	if o.Contrast != nil && (*o.Contrast < -100 || *o.Contrast > 100) {
		return fmt.Errorf("%w: contrast must be between -100 and 100", ErrInvalidRenderOption)
	}

	if o.Saturation != nil && (*o.Saturation < 0 || *o.Saturation > 200) {
		return fmt.Errorf("%w: saturation must be between 0 and 200", ErrInvalidRenderOption)
	}

	if o.Rotation != nil && (*o.Rotation < 0 || *o.Rotation > 360) {
		return fmt.Errorf("%w: rotation must be between 0 and 360", ErrInvalidRenderOption)
	}

	if o.Background == nil {
		bg := "white"
		o.Background = &bg
	}

	return nil
}

// ToImage creates an Image record from render options and rendering results.
func (o RenderOptions) ToImage(
	id, documentID uuid.UUID,
	pageNumber int,
	storageKey string,
	sizeBytes int64,
) *Image {
	return &Image{
		ID:         id,
		DocumentID: documentID,
		PageNumber: pageNumber,
		Format:     o.Format,
		DPI:        o.DPI,
		Quality:    o.Quality,
		Brightness: o.Brightness,
		Contrast:   o.Contrast,
		Saturation: o.Saturation,
		Rotation:   o.Rotation,
		Background: o.Background,
		StorageKey: storageKey,
		SizeBytes:  sizeBytes,
	}
}

// ToImageConfig converts render options to document-context ImageConfig.
func (o RenderOptions) ToImageConfig() config.ImageConfig {
	cfg := config.ImageConfig{
		Format:  string(o.Format),
		DPI:     o.DPI,
		Options: make(map[string]any),
	}

	if o.Quality != nil {
		cfg.Quality = *o.Quality
	} else if o.Format == document.JPEG {
		cfg.Quality = 90
	}

	if o.Brightness != nil {
		cfg.Options["brightness"] = *o.Brightness
	}
	if o.Contrast != nil {
		cfg.Options["contrast"] = *o.Contrast
	}
	if o.Saturation != nil {
		cfg.Options["saturation"] = *o.Saturation
	}
	if o.Rotation != nil {
		cfg.Options["rotation"] = *o.Rotation
	}
	if o.Background != nil {
		cfg.Options["background"] = *o.Background
	}

	return cfg
}
