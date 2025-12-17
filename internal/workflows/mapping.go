package workflows

import (
	"net/url"

	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/agent-lab/pkg/repository"
)

var runProjection = query.NewProjectionMap("public", "runs", "r").
	Project("id", "ID").
	Project("workflow_name", "WorkflowName").
	Project("status", "Status").
	Project("params", "Params").
	Project("result", "Result").
	Project("error_message", "ErrorMessage").
	Project("started_at", "StartedAt").
	Project("completed_at", "CompletedAt").
	Project("created_at", "CreatedAt").
	Project("updated_at", "UpdatedAt")

var runDefaultSort = query.SortField{Field: "CreatedAt", Descending: true}
var stageDefaultSort = query.SortField{Field: "CreatedAt", Descending: false}
var decisionDefaultSort = query.SortField{Field: "CreatedAt", Descending: false}

func scanRun(s repository.Scanner) (Run, error) {
	var r Run
	err := s.Scan(
		&r.ID,
		&r.WorkflowName,
		&r.Status,
		&r.Params,
		&r.Result,
		&r.ErrorMessage,
		&r.StartedAt,
		&r.CompletedAt,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	return r, err
}

var stageProjection = query.NewProjectionMap("public", "stages", "s").
	Project("id", "ID").
	Project("run_id", "RunID").
	Project("node_name", "NodeName").
	Project("iteration", "Iteration").
	Project("status", "Status").
	Project("input_snapshot", "InputSnapshot").
	Project("output_snapshot", "OutputSnapshot").
	Project("duration_ms", "DurationMs").
	Project("error_message", "ErrorMessage").
	Project("created_at", "CreatedAt")

func scanStage(s repository.Scanner) (Stage, error) {
	var st Stage
	err := s.Scan(
		&st.ID,
		&st.RunID,
		&st.NodeName,
		&st.Iteration,
		&st.Status,
		&st.InputSnapshot,
		&st.OutputSnapshot,
		&st.DurationMs,
		&st.ErrorMessage,
		&st.CreatedAt,
	)
	return st, err
}

var decisionProjection = query.NewProjectionMap("public", "decisions", "d").
	Project("id", "ID").
	Project("run_id", "RunID").
	Project("from_node", "FromNode").
	Project("to_node", "ToNode").
	Project("predicate_name", "PredicateName").
	Project("predicate_result", "PredicateResult").
	Project("reason", "Reason").
	Project("created_at", "CreatedAt")

func scanDecision(s repository.Scanner) (Decision, error) {
	var d Decision
	err := s.Scan(
		&d.ID,
		&d.RunID,
		&d.FromNode,
		&d.ToNode,
		&d.PredicateName,
		&d.PredicateResult,
		&d.Reason,
		&d.CreatedAt,
	)
	return d, err
}

// RunFilters contains optional criteria for filtering run queries.
type RunFilters struct {
	WorkflowName *string
	Status       *string
}

// RunFiltersFromQuery extracts run filters from URL query parameters.
func RunFiltersFromQuery(values url.Values) RunFilters {
	var f RunFilters

	if wn := values.Get("workflow_name"); wn != "" {
		f.WorkflowName = &wn
	}

	if s := values.Get("status"); s != "" {
		f.Status = &s
	}

	return f
}

// Apply adds filter conditions to the query builder.
func (f RunFilters) Apply(b *query.Builder) *query.Builder {
	return b.
		WhereEquals("WorkflowName", f.WorkflowName).
		WhereEquals("Status", f.Status)
}
