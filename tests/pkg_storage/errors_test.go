package pkg_storage_test

import (
	"errors"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/storage"
)

func TestErrors_Defined(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrNotFound", storage.ErrNotFound, "storage: key not found"},
		{"ErrPermissionDenied", storage.ErrPermissionDenied, "storage: permission denied"},
		{"ErrInvalidKey", storage.ErrInvalidKey, "storage: invalid key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Error("error is nil")
			}
			if tt.err.Error() != tt.msg {
				t.Errorf("error message = %q, want %q", tt.err.Error(), tt.msg)
			}
		})
	}
}

func TestErrors_IsComparable(t *testing.T) {
	wrapped := errors.New("wrapped: " + storage.ErrNotFound.Error())

	if errors.Is(wrapped, storage.ErrNotFound) {
		t.Error("wrapped error should not match via errors.Is")
	}

	if !errors.Is(storage.ErrNotFound, storage.ErrNotFound) {
		t.Error("ErrNotFound should match itself")
	}
}
