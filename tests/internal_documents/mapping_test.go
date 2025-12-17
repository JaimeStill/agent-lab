package internal_documents_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/documents"
	"github.com/JaimeStill/agent-lab/pkg/query"
)

func TestFiltersFromQuery(t *testing.T) {
	tests := []struct {
		name            string
		query           string
		wantName        bool
		nameVal         string
		wantContentType bool
		contentTypeVal  string
	}{
		{
			"empty query",
			"",
			false, "",
			false, "",
		},
		{
			"with name filter",
			"name=report",
			true, "report",
			false, "",
		},
		{
			"with content_type filter",
			"content_type=application/pdf",
			false, "",
			true, "application/pdf",
		},
		{
			"with both filters",
			"name=invoice&content_type=pdf",
			true, "invoice",
			true, "pdf",
		},
		{
			"with empty name",
			"name=",
			false, "",
			false, "",
		},
		{
			"with empty content_type",
			"content_type=",
			false, "",
			false, "",
		},
		{
			"with other params only",
			"page=1&page_size=10",
			false, "",
			false, "",
		},
		{
			"with filters and other params",
			"name=doc&content_type=image&page=1",
			true, "doc",
			true, "image",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.query)
			filters := documents.FiltersFromQuery(values)

			if tt.wantName {
				if filters.Name == nil {
					t.Error("FiltersFromQuery() Name = nil, want non-nil")
				} else if *filters.Name != tt.nameVal {
					t.Errorf("FiltersFromQuery() Name = %q, want %q", *filters.Name, tt.nameVal)
				}
			} else {
				if filters.Name != nil {
					t.Errorf("FiltersFromQuery() Name = %q, want nil", *filters.Name)
				}
			}

			if tt.wantContentType {
				if filters.ContentType == nil {
					t.Error("FiltersFromQuery() ContentType = nil, want non-nil")
				} else if *filters.ContentType != tt.contentTypeVal {
					t.Errorf("FiltersFromQuery() ContentType = %q, want %q", *filters.ContentType, tt.contentTypeVal)
				}
			} else {
				if filters.ContentType != nil {
					t.Errorf("FiltersFromQuery() ContentType = %q, want nil", *filters.ContentType)
				}
			}
		})
	}
}

func newTestProjection() *query.ProjectionMap {
	return query.NewProjectionMap("public", "documents", "d").
		Project("id", "ID").
		Project("name", "Name").
		Project("content_type", "ContentType")
}

func TestFilters_Apply(t *testing.T) {
	tests := []struct {
		name              string
		nameFilter        *string
		contentTypeFilter *string
		wantWhere         bool
		wantArgCount      int
	}{
		{
			"no filters",
			nil, nil,
			false, 0,
		},
		{
			"with name filter only",
			strPtr("report"),
			nil,
			true, 1,
		},
		{
			"with content_type filter only",
			nil,
			strPtr("application/pdf"),
			true, 1,
		},
		{
			"with both filters",
			strPtr("invoice"),
			strPtr("pdf"),
			true, 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := newTestProjection()
			b := query.NewBuilder(pm, query.SortField{Field: "Name"})

			filters := documents.Filters{
				Name:        tt.nameFilter,
				ContentType: tt.contentTypeFilter,
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

func strPtr(s string) *string {
	return &s
}
