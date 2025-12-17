package providers

import (
	"net/url"

	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/agent-lab/pkg/repository"
)

var projection = query.NewProjectionMap("public", "providers", "p").
	Project("id", "ID").
	Project("name", "Name").
	Project("config", "Config").
	Project("created_at", "CreatedAt").
	Project("updated_at", "UpdatedAt")

var defaultSort = query.SortField{Field: "Name"}

func scanProvider(s repository.Scanner) (Provider, error) {
	var p Provider
	err := s.Scan(&p.ID, &p.Name, &p.Config, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

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
