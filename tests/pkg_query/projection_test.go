package pkg_query_test

import (
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/query"
)

func TestNewProjectionMap(t *testing.T) {
	pm := query.NewProjectionMap("public", "users", "u")

	if pm.Alias() != "u" {
		t.Errorf("Alias() = %q, want %q", pm.Alias(), "u")
	}

	if pm.Table() != "public.users u" {
		t.Errorf("Table() = %q, want %q", pm.Table(), "public.users u")
	}
}

func TestProjectionMap_Project(t *testing.T) {
	pm := query.NewProjectionMap("public", "users", "u").
		Project("id", "ID").
		Project("email", "Email").
		Project("created_at", "CreatedAt")

	tests := []struct {
		viewName string
		wantCol  string
	}{
		{"ID", "u.id"},
		{"Email", "u.email"},
		{"CreatedAt", "u.created_at"},
	}

	for _, tt := range tests {
		t.Run(tt.viewName, func(t *testing.T) {
			col := pm.Column(tt.viewName)
			if col != tt.wantCol {
				t.Errorf("Column(%q) = %q, want %q", tt.viewName, col, tt.wantCol)
			}
		})
	}
}

func TestProjectionMap_Column_UnknownReturnsInput(t *testing.T) {
	pm := query.NewProjectionMap("public", "users", "u").
		Project("id", "ID")

	col := pm.Column("Unknown")
	if col != "Unknown" {
		t.Errorf("Column(%q) = %q, want %q", "Unknown", col, "Unknown")
	}
}

func TestProjectionMap_Columns(t *testing.T) {
	pm := query.NewProjectionMap("public", "users", "u").
		Project("id", "ID").
		Project("email", "Email")

	cols := pm.Columns()
	want := "u.id, u.email"

	if cols != want {
		t.Errorf("Columns() = %q, want %q", cols, want)
	}
}

func TestProjectionMap_ColumnList(t *testing.T) {
	pm := query.NewProjectionMap("public", "users", "u").
		Project("id", "ID").
		Project("email", "Email")

	list := pm.ColumnList()
	if len(list) != 2 {
		t.Fatalf("len(ColumnList()) = %d, want 2", len(list))
	}

	if list[0] != "u.id" {
		t.Errorf("ColumnList()[0] = %q, want %q", list[0], "u.id")
	}

	if list[1] != "u.email" {
		t.Errorf("ColumnList()[1] = %q, want %q", list[1], "u.email")
	}
}
