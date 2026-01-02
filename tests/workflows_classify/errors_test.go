package workflows_classify_test

import (
	"testing"

	"github.com/JaimeStill/agent-lab/workflows/classify"
)

func TestErrorValues(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{"ErrDocumentNotFound", classify.ErrDocumentNotFound, "document not found"},
		{"ErrNoPages", classify.ErrNoPages, "document has no pages"},
		{"ErrRenderFailed", classify.ErrRenderFailed, "failed to render pages"},
		{"ErrParseResponse", classify.ErrParseResponse, "failed to parse detection response"},
		{"ErrDetectionFailed", classify.ErrDetectionFailed, "detection failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.wantMsg {
				t.Errorf("error message = %q, want %q", tt.err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestErrors_AreDistinct(t *testing.T) {
	errors := []error{
		classify.ErrDocumentNotFound,
		classify.ErrNoPages,
		classify.ErrRenderFailed,
		classify.ErrParseResponse,
		classify.ErrDetectionFailed,
	}

	for i, err1 := range errors {
		for j, err2 := range errors {
			if i != j && err1 == err2 {
				t.Errorf("errors at index %d and %d are the same", i, j)
			}
		}
	}
}
