package profiles

import (
	"encoding/json"
	"net/url"

	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/agent-lab/pkg/repository"
)

var profileProjection = query.
	NewProjectionMap("public", "profiles", "p").
	Project("id", "ID").
	Project("workflow_name", "WorkflowName").
	Project("name", "Name").
	Project("description", "Description").
	Project("created_at", "CreatedAt").
	Project("updated_at", "UpdatedAt")

var stageProjection = query.
	NewProjectionMap("public", "profile_stages", "ps").
	Project("profile_id", "ProfileID").
	Project("stage_name", "StageName").
	Project("agent_id", "AgentID").
	Project("system_prompt", "SystemPrompt").
	Project("options", "Options")

var defaultSort = query.SortField{Field: "Name"}

func scanProfile(s repository.Scanner) (Profile, error) {
	var p Profile
	err := s.Scan(
		&p.ID, &p.WorkflowName, &p.Name,
		&p.Description, &p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

func scanProfileStage(s repository.Scanner) (ProfileStage, error) {
	var ps ProfileStage
	var opts []byte
	err := s.Scan(
		&ps.ProfileID, &ps.StageName,
		&ps.AgentID, &ps.SystemPrompt, &opts,
	)
	if len(opts) > 0 {
		ps.Options = json.RawMessage(opts)
	}
	return ps, err
}

type Filters struct {
	WorkflowName *string
}

func FiltersFromQuery(values url.Values) Filters {
	var workflowName *string
	if wn := values.Get("workflow_name"); wn != "" {
		workflowName = &wn
	}
	return Filters{
		WorkflowName: workflowName,
	}
}

func (f Filters) Apply(b *query.Builder) *query.Builder {
	return b.WhereEquals("WorkflowName", f.WorkflowName)
}
