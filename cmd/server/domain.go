package main

import (
	"github.com/JaimeStill/agent-lab/internal/agents"
	"github.com/JaimeStill/agent-lab/internal/providers"
)

type Domain struct {
	Providers providers.System
	Agents    agents.System
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
	}
}
