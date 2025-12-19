package decode_test

import (
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/decode"
)

type testStruct struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type nestedStruct struct {
	ID   string     `json:"id"`
	Data testStruct `json:"data"`
}

func TestFromMap_SimpleStruct(t *testing.T) {
	input := map[string]any{
		"name":  "test",
		"value": 42,
	}

	result, err := decode.FromMap[testStruct](input)
	if err != nil {
		t.Fatalf("FromMap() error = %v", err)
	}

	if result.Name != "test" {
		t.Errorf("Name = %q, want %q", result.Name, "test")
	}

	if result.Value != 42 {
		t.Errorf("Value = %d, want %d", result.Value, 42)
	}
}

func TestFromMap_NestedStruct(t *testing.T) {
	input := map[string]any{
		"id": "abc123",
		"data": map[string]any{
			"name":  "nested",
			"value": 100,
		},
	}

	result, err := decode.FromMap[nestedStruct](input)
	if err != nil {
		t.Fatalf("FromMap() error = %v", err)
	}

	if result.ID != "abc123" {
		t.Errorf("ID = %q, want %q", result.ID, "abc123")
	}

	if result.Data.Name != "nested" {
		t.Errorf("Data.Name = %q, want %q", result.Data.Name, "nested")
	}

	if result.Data.Value != 100 {
		t.Errorf("Data.Value = %d, want %d", result.Data.Value, 100)
	}
}

func TestFromMap_EmptyMap(t *testing.T) {
	input := map[string]any{}

	result, err := decode.FromMap[testStruct](input)
	if err != nil {
		t.Fatalf("FromMap() error = %v", err)
	}

	if result.Name != "" {
		t.Errorf("Name = %q, want empty string", result.Name)
	}

	if result.Value != 0 {
		t.Errorf("Value = %d, want 0", result.Value)
	}
}

func TestFromMap_ExtraFields(t *testing.T) {
	input := map[string]any{
		"name":  "test",
		"value": 42,
		"extra": "ignored",
	}

	result, err := decode.FromMap[testStruct](input)
	if err != nil {
		t.Fatalf("FromMap() error = %v", err)
	}

	if result.Name != "test" {
		t.Errorf("Name = %q, want %q", result.Name, "test")
	}
}

func TestFromMap_NilMap(t *testing.T) {
	var input map[string]any

	result, err := decode.FromMap[testStruct](input)
	if err != nil {
		t.Fatalf("FromMap() error = %v", err)
	}

	if result.Name != "" {
		t.Errorf("Name = %q, want empty string", result.Name)
	}
}

func TestFromMap_TypeMismatch(t *testing.T) {
	input := map[string]any{
		"name":  123,
		"value": "not a number",
	}

	_, err := decode.FromMap[testStruct](input)
	if err == nil {
		t.Error("FromMap() expected error for type mismatch, got nil")
	}
}
