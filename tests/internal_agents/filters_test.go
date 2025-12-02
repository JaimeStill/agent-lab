package internal_agents_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/agents"
	"github.com/JaimeStill/agent-lab/pkg/query"
)

func TestFiltersFromQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantName bool
		nameVal  string
	}{
		{
			"empty query",
			"",
			false,
			"",
		},
		{
			"with name filter",
			"name=ollama",
			true,
			"ollama",
		},
		{
			"with empty name",
			"name=",
			false,
			"",
		},
		{
			"with other params only",
			"page=1&pageSize=10",
			false,
			"",
		},
		{
			"with name and other params",
			"name=azure&page=1&pageSize=10",
			true,
			"azure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.query)
			filters := agents.FiltersFromQuery(values)

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
		})
	}
}

func newTestProjection() *query.ProjectionMap {
	return query.NewProjectionMap("public", "agents", "a").
		Project("id", "Id").
		Project("name", "Name").
		Project("config", "Config")
}

func TestFilters_Apply(t *testing.T) {
	tests := []struct {
		name       string
		nameFilter *string
		wantWhere  bool
	}{
		{
			"no filter",
			nil,
			false,
		},
		{
			"with name filter",
			strPtr("ollama"),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := newTestProjection()
			b := query.NewBuilder(pm, query.SortField{Field: "Name"})

			filters := agents.Filters{Name: tt.nameFilter}
			filters.Apply(b)

			sql, args := b.BuildCount()

			if tt.wantWhere {
				if !strings.Contains(sql, "WHERE") {
					t.Errorf("Apply() expected WHERE clause, got %q", sql)
				}
				if len(args) == 0 {
					t.Error("Apply() expected args, got none")
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
