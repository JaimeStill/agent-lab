package images

import "github.com/JaimeStill/agent-lab/pkg/repository"

// scanImage reads an Image from a database row.
func scanImage(s repository.Scanner) (Image, error) {
	var img Image
	err := s.Scan(
		&img.ID,
		&img.DocumentID,
		&img.PageNumber,
		&img.Format,
		&img.DPI,
		&img.Quality,
		&img.Brightness,
		&img.Contrast,
		&img.Saturation,
		&img.Rotation,
		&img.Background,
		&img.StorageKey,
		&img.SizeBytes,
		&img.CreatedAt,
	)
	return img, err
}
