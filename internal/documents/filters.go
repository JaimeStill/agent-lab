package documents

import (
	"net/url"

	"github.com/JaimeStill/agent-lab/pkg/query"
)

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
