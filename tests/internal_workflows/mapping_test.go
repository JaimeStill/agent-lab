package internal_workflows_test

import (
	"net/url"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/workflows"
)

func TestRunFiltersFromQuery(t *testing.T) {
	tests := []struct {
		name             string
		query            string
		wantWorkflowName *string
		wantStatus       *string
	}{
		{
			"empty query",
			"",
			nil,
			nil,
		},
		{
			"workflow_name only",
			"workflow_name=classify-docs",
			strPtr("classify-docs"),
			nil,
		},
		{
			"status only",
			"status=running",
			nil,
			strPtr("running"),
		},
		{
			"both filters",
			"workflow_name=classify-docs&status=completed",
			strPtr("classify-docs"),
			strPtr("completed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.query)
			got := workflows.RunFiltersFromQuery(values)

			if !strPtrEqual(got.WorkflowName, tt.wantWorkflowName) {
				t.Errorf("WorkflowName = %v, want %v", strPtrVal(got.WorkflowName), strPtrVal(tt.wantWorkflowName))
			}

			if !strPtrEqual(got.Status, tt.wantStatus) {
				t.Errorf("Status = %v, want %v", strPtrVal(got.Status), strPtrVal(tt.wantStatus))
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func strPtrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func strPtrVal(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
