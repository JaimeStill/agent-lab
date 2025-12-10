package images

import (
	"io"

	"github.com/JaimeStill/document-context/pkg/document"
)

// SupportedFormats lists content types that can be rendered to images.
var SupportedFormats = map[string]bool{
	"application/pdf": true,
}

// PageExtractor provides page-by-page access to a document.
type PageExtractor interface {
	ExtractPage(pageNum int) (document.Page, error)
	io.Closer
}

// IsSupported returns whether a content type can be rendered to images.
func IsSupported(contentType string) bool {
	return SupportedFormats[contentType]
}

// OpenDocument opens a document for page extraction based on its content type.
func OpenDocument(path string, contentType string) (PageExtractor, error) {
	switch contentType {
	case "application/pdf":
		return document.OpenPDF(path)
	default:
		return nil, ErrUnsupportedFormat
	}
}
