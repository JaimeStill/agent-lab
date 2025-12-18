package main

import (
	"github.com/JaimeStill/agent-lab/internal/agents"
	"github.com/JaimeStill/agent-lab/internal/documents"
	"github.com/JaimeStill/agent-lab/internal/images"
	"github.com/JaimeStill/agent-lab/internal/providers"
	"github.com/JaimeStill/agent-lab/internal/workflows"
)

type Domain struct {
	Providers providers.System
	Agents    agents.System
	Documents documents.System
	Images    images.System
	Workflows workflows.System
}

func NewDomain(runtime *Runtime) *Domain {
	providersSys := providers.New(
		runtime.Database.Connection(),
		runtime.Logger,
		runtime.Pagination,
	)

	agentsSys := agents.New(
		runtime.Database.Connection(),
		runtime.Logger,
		runtime.Pagination,
	)

	documentsSys := documents.New(
		runtime.Database.Connection(),
		runtime.Storage,
		runtime.Logger,
		runtime.Pagination,
	)

	imagesSys := images.New(
		documentsSys,
		runtime.Database.Connection(),
		runtime.Storage,
		runtime.Logger,
		runtime.Pagination,
	)

	workflowRuntime := workflows.NewRuntime(
		agentsSys,
		documentsSys,
		imagesSys,
		runtime.Logger,
	)

	workflowsSys := workflows.NewSystem(
		workflowRuntime,
		runtime.Database.Connection(),
		runtime.Logger,
		runtime.Pagination,
	)

	return &Domain{
		Providers: providersSys,
		Agents:    agentsSys,
		Documents: documentsSys,
		Images:    imagesSys,
		Workflows: workflowsSys,
	}
}
