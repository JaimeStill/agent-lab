package documents

import "github.com/JaimeStill/agent-lab/pkg/repository"

func scanDocument(s repository.Scanner) (Document, error) {
	var d Document
	err := s.Scan(
		&d.ID,
		&d.Name,
		&d.Filename,
		&d.ContentType,
		&d.SizeBytes,
		&d.PageCount,
		&d.StorageKey,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	return d, err
}
