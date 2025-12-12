package images

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/agent-lab/pkg/repository"
	"github.com/JaimeStill/document-context/pkg/document"
	"github.com/google/uuid"
)

// projection maps database columns to Image struct fields for query building.
var projection = query.NewProjectionMap("public", "images", "i").
	Project("id", "ID").
	Project("document_id", "DocumentID").
	Project("page_number", "PageNumber").
	Project("format", "Format").
	Project("dpi", "DPI").
	Project("quality", "Quality").
	Project("brightness", "Brightness").
	Project("contrast", "Contrast").
	Project("saturation", "Saturation").
	Project("rotation", "Rotation").
	Project("background", "Background").
	Project("storage_key", "StorageKey").
	Project("size_bytes", "SizeBytes").
	Project("created_at", "CreatedAt")

// defaultSort orders images by creation time, newest first.
var defaultSort = query.SortField{Field: "CreatedAt", Descending: true}

// scanImage reads an Image from a database row.
func scanImage(s repository.Scanner) (Image, error) {
	var img Image
	err := s.Scan(
		&img.ID,
		&img.DocumentID,
		&img.PageNumber,
		&img.Format,
		&img.DPI,
		&img.Quality,
		&img.Brightness,
		&img.Contrast,
		&img.Saturation,
		&img.Rotation,
		&img.Background,
		&img.StorageKey,
		&img.SizeBytes,
		&img.CreatedAt,
	)
	return img, err
}

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

// ParsePageRange parses a page range expression into a sorted slice of page numbers.
// Supports formats: "1", "1-5", "1,3,5", "1-5,10,15-20", "-3" (start at 1), "5-" (end at maxPage).
// Results are deduplicated and sorted.
func ParsePageRange(expr string, maxPage int) ([]int, error) {
	if expr == "" {
		return nil, fmt.Errorf("%w: empty page range", ErrInvalidPageRange)
	}

	seen := make(map[int]bool)
	parts := strings.SplitSeq(expr, ",")

	for part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			start, end, err := parseRange(part, maxPage)
			if err != nil {
				return nil, err
			}

			for i := start; i <= end; i++ {
				seen[i] = true
			}
		} else {
			page, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid page %q", ErrInvalidPageRange, part)
			}
			if page < 1 || page > maxPage {
				return nil, fmt.Errorf("%w: page %d out of range [1-%d]", ErrPageOutOfRange, page, maxPage)
			}
			seen[page] = true
		}
	}

	if len(seen) == 0 {
		return nil, fmt.Errorf("%w: no valid pages", ErrInvalidPageRange)
	}

	pages := make([]int, 0, len(seen))
	for page := range seen {
		pages = append(pages, page)
	}

	sort.Ints(pages)

	return pages, nil
}

func parseRange(part string, maxPage int) (int, int, error) {
	idx := strings.Index(part, "-")
	if idx == -1 {
		return 0, 0, fmt.Errorf("%w: invalid range %q", ErrInvalidPageRange, part)
	}

	startStr := strings.TrimSpace(part[:idx])
	endStr := strings.TrimSpace(part[idx+1:])

	var start, end int
	var err error

	if startStr == "" {
		start = 1
	} else {
		start, err = strconv.Atoi(startStr)
		if err != nil {
			return 0, 0, fmt.Errorf("%w: invalid start %q", ErrInvalidPageRange, startStr)
		}
	}

	if endStr == "" {
		end = maxPage
	} else {
		end, err = strconv.Atoi(endStr)
		if err != nil {
			return 0, 0, fmt.Errorf("%w: invalid end %q", ErrInvalidPageRange, endStr)
		}
	}

	if start < 1 {
		return 0, 0, fmt.Errorf("%w: start page must be >= 1", ErrInvalidPageRange)
	}
	if end > maxPage {
		return 0, 0, fmt.Errorf("%w: end page %d exceends document pages (%d)", ErrPageOutOfRange, end, maxPage)
	}
	if start > end {
		return 0, 0, fmt.Errorf("%w: start > end in %q", ErrInvalidPageRange, part)
	}

	return start, end, nil
}
