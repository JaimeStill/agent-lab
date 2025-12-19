// Package decode provides utilities for decoding unstructured data into typed structures.
package decode

import "encoding/json"

// FromMap decodes a map[string]any into a typed struct using JSON round-trip.
// This is useful for converting observability event data into domain-specific types.
func FromMap[T any](data map[string]any) (T, error) {
	var result T
	b, err := json.Marshal(data)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(b, &result)
	return result, err
}
