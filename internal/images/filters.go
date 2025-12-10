package images

import (
	"net/url"
	"strconv"

	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/document-context/pkg/document"
	"github.com/google/uuid"
)

// Filters defines optional criteria for querying images.
type Filters struct {
	DocumentID *uuid.UUID
	Format     *document.ImageFormat
	PageNumber *int
}

// FiltersFromQuery extracts image filters from URL query parameters.
func FiltersFromQuery(values url.Values) Filters {
	var f Filters

	if docID := values.Get("document_id"); docID != "" {
		if parsed, err := uuid.Parse(docID); err == nil {
			f.DocumentID = &parsed
		}
	}

	if format := values.Get("format"); format != "" {
		if parsed, err := ParseImageFormat(format); err == nil {
			f.Format = &parsed
		}
	}

	if pg := values.Get("page_number"); pg != "" {
		if page, err := strconv.Atoi(pg); err == nil {
			f.PageNumber = &page
		}
	}

	return f
}

// Apply adds filter conditions to a query builder.
func (f Filters) Apply(b *query.Builder) *query.Builder {
	if f.DocumentID != nil {
		b.WhereEquals("DocumentID", *f.DocumentID)
	}

	if f.Format != nil {
		b.WhereEquals("Format", *f.Format)
	}

	if f.PageNumber != nil {
		b.WhereEquals("PageNumber", *f.PageNumber)
	}

	return b
}
