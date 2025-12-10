package agents

import "github.com/JaimeStill/agent-lab/pkg/query"

var projection = query.
	NewProjectionMap("public", "agents", "a").
	Project("id", "Id").
	Project("name", "Name").
	Project("config", "Config").
	Project("created_at", "CreatedAt").
	Project("updated_at", "UpdatedAt")

var defaultSort = query.SortField{Field: "Name"}
