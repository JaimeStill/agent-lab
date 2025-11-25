package pagination

// PageRequest represents a client request for a page of data with optional search and sorting.
type PageRequest struct {
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	Search     *string `json:"search,omitempty"`
	SortBy     string  `json:"sort_by,omitempty"`
	Descending bool    `json:"descending,omitempty"`
}

// Normalize adjusts the request to ensure valid pagination values based on the config.
func (r *PageRequest) Normalize(cfg Config) {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 {
		r.PageSize = cfg.DefaultPageSize
	}
	if r.PageSize > cfg.MaxPageSize {
		r.PageSize = cfg.MaxPageSize
	}
}

// Offset calculates the number of records to skip based on page and page size.
func (r *PageRequest) Offset() int {
	return (r.Page - 1) * r.PageSize
}

// PageResult holds a page of data along with pagination metadata.
type PageResult[T any] struct {
	Data       []T `json:"data"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalPages int `json:"total_pages"`
}

// NewPageResult creates a PageResult with calculated total pages.
func NewPageResult[T any](data []T, total, page, pageSize int) PageResult[T] {
	totalPages := total / pageSize
	if total%pageSize != 0 {
		totalPages++
	}
	if totalPages < 1 {
		totalPages = 1
	}

	if data == nil {
		data = []T{}
	}

	return PageResult[T]{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}
