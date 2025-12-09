package pkg_pagination_test

import (
	"net/url"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/agent-lab/pkg/query"
)

func TestPageRequest_Normalize(t *testing.T) {
	cfg := pagination.Config{
		DefaultPageSize: 20,
		MaxPageSize:     100,
	}

	tests := []struct {
		name         string
		request      pagination.PageRequest
		wantPage     int
		wantPageSize int
	}{
		{
			name:         "valid values unchanged",
			request:      pagination.PageRequest{Page: 2, PageSize: 25},
			wantPage:     2,
			wantPageSize: 25,
		},
		{
			name:         "zero page becomes 1",
			request:      pagination.PageRequest{Page: 0, PageSize: 25},
			wantPage:     1,
			wantPageSize: 25,
		},
		{
			name:         "negative page becomes 1",
			request:      pagination.PageRequest{Page: -1, PageSize: 25},
			wantPage:     1,
			wantPageSize: 25,
		},
		{
			name:         "zero page size gets default",
			request:      pagination.PageRequest{Page: 1, PageSize: 0},
			wantPage:     1,
			wantPageSize: 20,
		},
		{
			name:         "negative page size gets default",
			request:      pagination.PageRequest{Page: 1, PageSize: -10},
			wantPage:     1,
			wantPageSize: 20,
		},
		{
			name:         "page size exceeding max gets capped",
			request:      pagination.PageRequest{Page: 1, PageSize: 200},
			wantPage:     1,
			wantPageSize: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.request.Normalize(cfg)

			if tt.request.Page != tt.wantPage {
				t.Errorf("Page = %d, want %d", tt.request.Page, tt.wantPage)
			}

			if tt.request.PageSize != tt.wantPageSize {
				t.Errorf("PageSize = %d, want %d", tt.request.PageSize, tt.wantPageSize)
			}
		})
	}
}

func TestPageRequest_Offset(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		pageSize   int
		wantOffset int
	}{
		{"first page", 1, 20, 0},
		{"second page", 2, 20, 20},
		{"third page", 3, 10, 20},
		{"large page", 10, 25, 225},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := pagination.PageRequest{Page: tt.page, PageSize: tt.pageSize}

			offset := req.Offset()
			if offset != tt.wantOffset {
				t.Errorf("Offset() = %d, want %d", offset, tt.wantOffset)
			}
		})
	}
}

func TestNewPageResult(t *testing.T) {
	tests := []struct {
		name           string
		data           []string
		total          int
		page           int
		pageSize       int
		wantTotalPages int
	}{
		{
			name:           "exact division",
			data:           []string{"a", "b"},
			total:          100,
			page:           1,
			pageSize:       20,
			wantTotalPages: 5,
		},
		{
			name:           "with remainder",
			data:           []string{"a"},
			total:          21,
			page:           1,
			pageSize:       20,
			wantTotalPages: 2,
		},
		{
			name:           "single page",
			data:           []string{"a", "b", "c"},
			total:          3,
			page:           1,
			pageSize:       20,
			wantTotalPages: 1,
		},
		{
			name:           "empty result still has one page",
			data:           []string{},
			total:          0,
			page:           1,
			pageSize:       20,
			wantTotalPages: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pagination.NewPageResult(tt.data, tt.total, tt.page, tt.pageSize)

			if result.Total != tt.total {
				t.Errorf("Total = %d, want %d", result.Total, tt.total)
			}

			if result.Page != tt.page {
				t.Errorf("Page = %d, want %d", result.Page, tt.page)
			}

			if result.PageSize != tt.pageSize {
				t.Errorf("PageSize = %d, want %d", result.PageSize, tt.pageSize)
			}

			if result.TotalPages != tt.wantTotalPages {
				t.Errorf("TotalPages = %d, want %d", result.TotalPages, tt.wantTotalPages)
			}

			if len(result.Data) != len(tt.data) {
				t.Errorf("len(Data) = %d, want %d", len(result.Data), len(tt.data))
			}
		})
	}
}

func TestNewPageResult_NilDataBecomesEmptySlice(t *testing.T) {
	result := pagination.NewPageResult[string](nil, 0, 1, 20)

	if result.Data == nil {
		t.Error("Data is nil, want empty slice")
	}

	if len(result.Data) != 0 {
		t.Errorf("len(Data) = %d, want 0", len(result.Data))
	}
}

func TestPageRequestFromQuery(t *testing.T) {
	cfg := pagination.Config{
		DefaultPageSize: 20,
		MaxPageSize:     100,
	}

	tests := []struct {
		name         string
		query        string
		wantPage     int
		wantPageSize int
		wantSearch   *string
		wantSort     []query.SortField
	}{
		{
			name:         "empty query uses defaults",
			query:        "",
			wantPage:     1,
			wantPageSize: 20,
			wantSearch:   nil,
			wantSort:     nil,
		},
		{
			name:         "page and page_size parsed",
			query:        "page=2&page_size=50",
			wantPage:     2,
			wantPageSize: 50,
			wantSearch:   nil,
			wantSort:     nil,
		},
		{
			name:         "search parsed",
			query:        "search=test",
			wantPage:     1,
			wantPageSize: 20,
			wantSearch:   strPtr("test"),
			wantSort:     nil,
		},
		{
			name:         "sort parsed ascending",
			query:        "sort=name",
			wantPage:     1,
			wantPageSize: 20,
			wantSearch:   nil,
			wantSort:     []query.SortField{{Field: "name", Descending: false}},
		},
		{
			name:         "sort parsed descending",
			query:        "sort=-createdAt",
			wantPage:     1,
			wantPageSize: 20,
			wantSearch:   nil,
			wantSort:     []query.SortField{{Field: "createdAt", Descending: true}},
		},
		{
			name:         "multi-column sort",
			query:        "sort=name,-createdAt",
			wantPage:     1,
			wantPageSize: 20,
			wantSearch:   nil,
			wantSort: []query.SortField{
				{Field: "name", Descending: false},
				{Field: "createdAt", Descending: true},
			},
		},
		{
			name:         "all params combined",
			query:        "page=3&page_size=25&search=foo&sort=-name",
			wantPage:     3,
			wantPageSize: 25,
			wantSearch:   strPtr("foo"),
			wantSort:     []query.SortField{{Field: "name", Descending: true}},
		},
		{
			name:         "invalid page defaults to 1",
			query:        "page=invalid",
			wantPage:     1,
			wantPageSize: 20,
			wantSearch:   nil,
			wantSort:     nil,
		},
		{
			name:         "page_size exceeding max gets capped",
			query:        "page_size=500",
			wantPage:     1,
			wantPageSize: 100,
			wantSearch:   nil,
			wantSort:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.query)
			req := pagination.PageRequestFromQuery(values, cfg)

			if req.Page != tt.wantPage {
				t.Errorf("Page = %d, want %d", req.Page, tt.wantPage)
			}

			if req.PageSize != tt.wantPageSize {
				t.Errorf("PageSize = %d, want %d", req.PageSize, tt.wantPageSize)
			}

			if tt.wantSearch == nil {
				if req.Search != nil {
					t.Errorf("Search = %v, want nil", *req.Search)
				}
			} else {
				if req.Search == nil {
					t.Errorf("Search = nil, want %v", *tt.wantSearch)
				} else if *req.Search != *tt.wantSearch {
					t.Errorf("Search = %v, want %v", *req.Search, *tt.wantSearch)
				}
			}

			if tt.wantSort == nil {
				if req.Sort != nil {
					t.Errorf("Sort = %v, want nil", req.Sort)
				}
			} else {
				if len(req.Sort) != len(tt.wantSort) {
					t.Fatalf("len(Sort) = %d, want %d", len(req.Sort), len(tt.wantSort))
				}
				for i, want := range tt.wantSort {
					if req.Sort[i].Field != want.Field {
						t.Errorf("Sort[%d].Field = %q, want %q", i, req.Sort[i].Field, want.Field)
					}
					if req.Sort[i].Descending != want.Descending {
						t.Errorf("Sort[%d].Descending = %v, want %v", i, req.Sort[i].Descending, want.Descending)
					}
				}
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func TestSortFields_UnmarshalJSON_String(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantSort []query.SortField
	}{
		{
			name:     "single ascending field",
			json:     `"name"`,
			wantSort: []query.SortField{{Field: "name", Descending: false}},
		},
		{
			name:     "single descending field",
			json:     `"-created_at"`,
			wantSort: []query.SortField{{Field: "created_at", Descending: true}},
		},
		{
			name: "multiple fields",
			json: `"name,-created_at,updated_at"`,
			wantSort: []query.SortField{
				{Field: "name", Descending: false},
				{Field: "created_at", Descending: true},
				{Field: "updated_at", Descending: false},
			},
		},
		{
			name:     "empty string",
			json:     `""`,
			wantSort: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sf pagination.SortFields
			if err := sf.UnmarshalJSON([]byte(tt.json)); err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}

			if tt.wantSort == nil {
				if len(sf) != 0 {
					t.Errorf("SortFields = %v, want empty", sf)
				}
				return
			}

			if len(sf) != len(tt.wantSort) {
				t.Fatalf("len(SortFields) = %d, want %d", len(sf), len(tt.wantSort))
			}

			for i, want := range tt.wantSort {
				if sf[i].Field != want.Field {
					t.Errorf("SortFields[%d].Field = %q, want %q", i, sf[i].Field, want.Field)
				}
				if sf[i].Descending != want.Descending {
					t.Errorf("SortFields[%d].Descending = %v, want %v", i, sf[i].Descending, want.Descending)
				}
			}
		})
	}
}

func TestSortFields_UnmarshalJSON_Array(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantSort []query.SortField
	}{
		{
			name:     "single field object",
			json:     `[{"Field":"name","Descending":false}]`,
			wantSort: []query.SortField{{Field: "name", Descending: false}},
		},
		{
			name: "multiple field objects",
			json: `[{"Field":"name","Descending":false},{"Field":"created_at","Descending":true}]`,
			wantSort: []query.SortField{
				{Field: "name", Descending: false},
				{Field: "created_at", Descending: true},
			},
		},
		{
			name:     "empty array",
			json:     `[]`,
			wantSort: []query.SortField{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sf pagination.SortFields
			if err := sf.UnmarshalJSON([]byte(tt.json)); err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}

			if len(sf) != len(tt.wantSort) {
				t.Fatalf("len(SortFields) = %d, want %d", len(sf), len(tt.wantSort))
			}

			for i, want := range tt.wantSort {
				if sf[i].Field != want.Field {
					t.Errorf("SortFields[%d].Field = %q, want %q", i, sf[i].Field, want.Field)
				}
				if sf[i].Descending != want.Descending {
					t.Errorf("SortFields[%d].Descending = %v, want %v", i, sf[i].Descending, want.Descending)
				}
			}
		})
	}
}

func TestSortFields_UnmarshalJSON_InvalidInput(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"invalid type number", `123`},
		{"invalid type boolean", `true`},
		{"malformed json", `{invalid`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sf pagination.SortFields
			if err := sf.UnmarshalJSON([]byte(tt.json)); err == nil {
				t.Error("UnmarshalJSON() expected error, got nil")
			}
		})
	}
}
