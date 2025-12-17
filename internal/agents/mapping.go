package agents

import (
	"net/url"

	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/agent-lab/pkg/repository"
)

var projection = query.
	NewProjectionMap("public", "agents", "a").
	Project("id", "ID").
	Project("name", "Name").
	Project("config", "Config").
	Project("created_at", "CreatedAt").
	Project("updated_at", "UpdatedAt")

var defaultSort = query.SortField{Field: "Name"}

func scanAgent(s repository.Scanner) (Agent, error) {
	var a Agent
	err := s.Scan(&a.ID, &a.Name, &a.Config, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

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
