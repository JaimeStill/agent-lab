package providers

import (
	"net/url"

	"github.com/JaimeStill/agent-lab/pkg/query"
)

type Filters struct {
	Name *string
}

func FiltersFromQuery(values url.Values) Filters {
	var name *string
	if n := values.Get("name"); n != "" {
		name = &n
	}

	return Filters{
		Name: name,
	}
}

func (f Filters) Apply(b *query.Builder) *query.Builder {
	return b.WhereContains("Name", f.Name)
}
