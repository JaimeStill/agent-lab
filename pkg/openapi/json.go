package openapi

import (
	"encoding/json"
	"os"
)

// MarshalJSON serializes a Spec to formatted JSON bytes.
func MarshalJSON(spec *Spec) ([]byte, error) {
	return json.MarshalIndent(spec, "", "  ")
}

// WriteJSON serializes a Spec to formatted JSON and writes it to a file.
func WriteJSON(spec *Spec, filename string) error {
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
