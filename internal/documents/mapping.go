package documents

import (
	"net/url"

	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/agent-lab/pkg/repository"
)

var projection = query.NewProjectionMap("public", "documents", "d").
	Project("id", "Id").
	Project("name", "Name").
	Project("filename", "Filename").
	Project("content_type", "ContentType").
	Project("size_bytes", "SizeBytes").
	Project("page_count", "PageCount").
	Project("storage_key", "StorageKey").
	Project("created_at", "CreatedAt").
	Project("updated_at", "UpdatedAt")

var defaultSort = query.SortField{Field: "CreatedAt", Descending: true}

func scanDocument(s repository.Scanner) (Document, error) {
	var d Document
	err := s.Scan(
		&d.ID,
		&d.Name,
		&d.Filename,
		&d.ContentType,
		&d.SizeBytes,
		&d.PageCount,
		&d.StorageKey,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	return d, err
}

// Filters contains optional criteria for filtering document queries.
type Filters struct {
	Name        *string
	ContentType *string
}

// FiltersFromQuery extracts document filters from URL query parameters.
func FiltersFromQuery(values url.Values) Filters {
	var f Filters

	if n := values.Get("name"); n != "" {
		f.Name = &n
	}

	if ct := values.Get("content_type"); ct != "" {
		f.ContentType = &ct
	}

	return f
}

// Apply adds filter conditions to the query builder.
func (f Filters) Apply(b *query.Builder) *query.Builder {
	return b.
		WhereContains("Name", f.Name).
		WhereContains("ContentType", f.ContentType)
}
