package internal_profiles_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/profiles"
	"github.com/JaimeStill/agent-lab/pkg/query"
)

func TestFiltersFromQuery(t *testing.T) {
	tests := []struct {
		name             string
		query            string
		wantWorkflowName bool
		workflowNameVal  string
	}{
		{
			"empty query",
			"",
			false,
			"",
		},
		{
			"with workflow_name filter",
			"workflow_name=summarize",
			true,
			"summarize",
		},
		{
			"with empty workflow_name",
			"workflow_name=",
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
			"with workflow_name and other params",
			"workflow_name=reasoning&page=1&pageSize=10",
			true,
			"reasoning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.query)
			filters := profiles.FiltersFromQuery(values)

			if tt.wantWorkflowName {
				if filters.WorkflowName == nil {
					t.Error("FiltersFromQuery() WorkflowName = nil, want non-nil")
				} else if *filters.WorkflowName != tt.workflowNameVal {
					t.Errorf("FiltersFromQuery() WorkflowName = %q, want %q", *filters.WorkflowName, tt.workflowNameVal)
				}
			} else {
				if filters.WorkflowName != nil {
					t.Errorf("FiltersFromQuery() WorkflowName = %q, want nil", *filters.WorkflowName)
				}
			}
		})
	}
}

func newTestProjection() *query.ProjectionMap {
	return query.NewProjectionMap("public", "profiles", "p").
		Project("id", "ID").
		Project("workflow_name", "WorkflowName").
		Project("name", "Name")
}

func TestFilters_Apply(t *testing.T) {
	tests := []struct {
		name               string
		workflowNameFilter *string
		wantWhere          bool
	}{
		{
			"no filter",
			nil,
			false,
		},
		{
			"with workflow_name filter",
			strPtr("summarize"),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := newTestProjection()
			b := query.NewBuilder(pm, query.SortField{Field: "Name"})

			filters := profiles.Filters{WorkflowName: tt.workflowNameFilter}
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
