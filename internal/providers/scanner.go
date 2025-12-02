package providers

import "github.com/JaimeStill/agent-lab/pkg/repository"

func scanProvider(s repository.Scanner) (Provider, error) {
	var p Provider
	err := s.Scan(&p.ID, &p.Name, &p.Config, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}
