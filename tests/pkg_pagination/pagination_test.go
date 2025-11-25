package pkg_pagination_test

import (
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
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
