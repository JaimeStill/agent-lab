package pkg_database_test

import (
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/database"
)

func TestErrNotReady_Defined(t *testing.T) {
	if database.ErrNotReady == nil {
		t.Error("ErrNotReady is nil")
	}

	if database.ErrNotReady.Error() != "database not ready" {
		t.Errorf("ErrNotReady.Error() = %q, want %q",
			database.ErrNotReady.Error(), "database not ready")
	}
}
