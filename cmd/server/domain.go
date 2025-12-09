package main

import (
	"github.com/JaimeStill/agent-lab/internal/agents"
	"github.com/JaimeStill/agent-lab/internal/documents"
	"github.com/JaimeStill/agent-lab/internal/providers"
)

type Domain struct {
	Providers providers.System
	Agents    agents.System
	Documents documents.System
}

func NewDomain(runtime *Runtime) *Domain {
	return &Domain{
		Providers: providers.New(
			runtime.Database.Connection(),
			runtime.Logger,
			runtime.Pagination,
		),
		Agents: agents.New(
			runtime.Database.Connection(),
			runtime.Logger,
			runtime.Pagination,
		),
		Documents: documents.New(
			runtime.Database.Connection(),
			runtime.Storage,
			runtime.Logger,
			runtime.Pagination,
		),
	}
}
