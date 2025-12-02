package agents

import (
	"net/url"

	"github.com/JaimeStill/agent-lab/pkg/query"
)

// Filters contains optional filtering criteria for agent queries.
type Filters struct {
	Name *string
}

// FiltersFromQuery extracts filter values from URL query parameters.
func FiltersFromQuery(values url.Values) Filters {
	var name *string
	if n := values.Get("name"); n != "" {
		name = &n
	}

	return Filters{
		Name: name,
	}
}

// Apply adds filter conditions to a query builder.
func (f Filters) Apply(b *query.Builder) *query.Builder {
	return b.WhereContains("Name", f.Name)
}
