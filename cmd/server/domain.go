package main

import "github.com/JaimeStill/agent-lab/internal/providers"

type Domain struct {
	Providers providers.System
}

func NewDomain(runtime *Runtime) *Domain {
	return &Domain{
		Providers: providers.New(
			runtime.Database.Connection(),
			runtime.Logger,
			runtime.Pagination,
		),
	}
}
