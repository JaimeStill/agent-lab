package internal_images_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/images"
	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/document-context/pkg/document"
	"github.com/google/uuid"
)

func TestFiltersFromQuery(t *testing.T) {
	testDocID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name           string
		query          string
		wantDocID      bool
		docIDVal       uuid.UUID
		wantFormat     bool
		formatVal      document.ImageFormat
		wantPageNumber bool
		pageNumberVal  int
	}{
		{
			"empty query",
			"",
			false, uuid.Nil,
			false, "",
			false, 0,
		},
		{
			"with document_id filter",
			"document_id=11111111-1111-1111-1111-111111111111",
			true, testDocID,
			false, "",
			false, 0,
		},
		{
			"with format filter png",
			"format=png",
			false, uuid.Nil,
			true, document.PNG,
			false, 0,
		},
		{
			"with format filter jpg",
			"format=jpg",
			false, uuid.Nil,
			true, document.JPEG,
			false, 0,
		},
		{
			"with page_number filter",
			"page_number=5",
			false, uuid.Nil,
			false, "",
			true, 5,
		},
		{
			"with all filters",
			"document_id=11111111-1111-1111-1111-111111111111&format=png&page_number=3",
			true, testDocID,
			true, document.PNG,
			true, 3,
		},
		{
			"with invalid document_id",
			"document_id=invalid",
			false, uuid.Nil,
			false, "",
			false, 0,
		},
		{
			"with invalid format",
			"format=gif",
			false, uuid.Nil,
			false, "",
			false, 0,
		},
		{
			"with invalid page_number",
			"page_number=abc",
			false, uuid.Nil,
			false, "",
			false, 0,
		},
		{
			"with empty values",
			"document_id=&format=&page_number=",
			false, uuid.Nil,
			false, "",
			false, 0,
		},
		{
			"with pagination params only",
			"page=1&page_size=10",
			false, uuid.Nil,
			false, "",
			false, 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.query)
			filters := images.FiltersFromQuery(values)

			if tt.wantDocID {
				if filters.DocumentID == nil {
					t.Error("FiltersFromQuery() DocumentID = nil, want non-nil")
				} else if *filters.DocumentID != tt.docIDVal {
					t.Errorf("FiltersFromQuery() DocumentID = %v, want %v", *filters.DocumentID, tt.docIDVal)
				}
			} else {
				if filters.DocumentID != nil {
					t.Errorf("FiltersFromQuery() DocumentID = %v, want nil", *filters.DocumentID)
				}
			}

			if tt.wantFormat {
				if filters.Format == nil {
					t.Error("FiltersFromQuery() Format = nil, want non-nil")
				} else if *filters.Format != tt.formatVal {
					t.Errorf("FiltersFromQuery() Format = %q, want %q", *filters.Format, tt.formatVal)
				}
			} else {
				if filters.Format != nil {
					t.Errorf("FiltersFromQuery() Format = %q, want nil", *filters.Format)
				}
			}

			if tt.wantPageNumber {
				if filters.PageNumber == nil {
					t.Error("FiltersFromQuery() PageNumber = nil, want non-nil")
				} else if *filters.PageNumber != tt.pageNumberVal {
					t.Errorf("FiltersFromQuery() PageNumber = %d, want %d", *filters.PageNumber, tt.pageNumberVal)
				}
			} else {
				if filters.PageNumber != nil {
					t.Errorf("FiltersFromQuery() PageNumber = %d, want nil", *filters.PageNumber)
				}
			}
		})
	}
}

func newTestProjection() *query.ProjectionMap {
	return query.NewProjectionMap("public", "images", "i").
		Project("id", "ID").
		Project("document_id", "DocumentID").
		Project("format", "Format").
		Project("page_number", "PageNumber")
}

func TestFilters_Apply(t *testing.T) {
	testDocID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	pngFormat := document.PNG

	tests := []struct {
		name         string
		docID        *uuid.UUID
		format       *document.ImageFormat
		pageNumber   *int
		wantWhere    bool
		wantArgCount int
	}{
		{
			"no filters",
			nil, nil, nil,
			false, 0,
		},
		{
			"with document_id only",
			&testDocID, nil, nil,
			true, 1,
		},
		{
			"with format only",
			nil, &pngFormat, nil,
			true, 1,
		},
		{
			"with page_number only",
			nil, nil, intPtr(5),
			true, 1,
		},
		{
			"with all filters",
			&testDocID, &pngFormat, intPtr(3),
			true, 3,
		},
		{
			"with document_id and format",
			&testDocID, &pngFormat, nil,
			true, 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := newTestProjection()
			b := query.NewBuilder(pm, query.SortField{Field: "ID"})

			filters := images.Filters{
				DocumentID: tt.docID,
				Format:     tt.format,
				PageNumber: tt.pageNumber,
			}
			filters.Apply(b)

			sql, args := b.BuildCount()

			if tt.wantWhere {
				if !strings.Contains(sql, "WHERE") {
					t.Errorf("Apply() expected WHERE clause, got %q", sql)
				}
				if len(args) != tt.wantArgCount {
					t.Errorf("Apply() args count = %d, want %d", len(args), tt.wantArgCount)
				}
			} else {
				if strings.Contains(sql, "WHERE") {
					t.Errorf("Apply() unexpected WHERE clause, got %q", sql)
				}
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}
