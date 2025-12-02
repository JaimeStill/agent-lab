package agents

import "github.com/JaimeStill/agent-lab/pkg/repository"

func scanAgent(s repository.Scanner) (Agent, error) {
	var a Agent
	err := s.Scan(&a.ID, &a.Name, &a.Config, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}
