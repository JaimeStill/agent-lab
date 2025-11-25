package pkg_query_test

import (
	"strings"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/query"
)

func newTestProjection() *query.ProjectionMap {
	return query.NewProjectionMap("public", "users", "u").
		Project("id", "Id").
		Project("name", "Name").
		Project("email", "Email")
}

func TestBuilder_BuildCount_NoConditions(t *testing.T) {
	pm := newTestProjection()
	b := query.NewBuilder(pm, "Name")

	sql, args := b.BuildCount()

	wantSQL := "SELECT COUNT(*) FROM public.users u"
	if sql != wantSQL {
		t.Errorf("BuildCount() sql = %q, want %q", sql, wantSQL)
	}

	if len(args) != 0 {
		t.Errorf("BuildCount() args = %v, want empty", args)
	}
}

func TestBuilder_BuildPage_NoConditions(t *testing.T) {
	pm := newTestProjection()
	b := query.NewBuilder(pm, "Name")

	sql, args := b.BuildPage(1, 20)

	if !strings.Contains(sql, "SELECT u.id, u.name, u.email FROM public.users u") {
		t.Errorf("BuildPage() missing select clause, got %q", sql)
	}

	if !strings.Contains(sql, "ORDER BY u.name ASC") {
		t.Errorf("BuildPage() missing order by, got %q", sql)
	}

	if !strings.Contains(sql, "LIMIT 20 OFFSET 0") {
		t.Errorf("BuildPage() missing limit/offset, got %q", sql)
	}

	if len(args) != 0 {
		t.Errorf("BuildPage() args = %v, want empty", args)
	}
}

func TestBuilder_BuildPage_Pagination(t *testing.T) {
	pm := newTestProjection()

	tests := []struct {
		name       string
		page       int
		pageSize   int
		wantLimit  string
		wantOffset string
	}{
		{"first page", 1, 20, "LIMIT 20", "OFFSET 0"},
		{"second page", 2, 20, "LIMIT 20", "OFFSET 20"},
		{"third page", 3, 10, "LIMIT 10", "OFFSET 20"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := query.NewBuilder(pm, "Name")
			sql, _ := b.BuildPage(tt.page, tt.pageSize)

			if !strings.Contains(sql, tt.wantLimit) {
				t.Errorf("BuildPage() missing %q, got %q", tt.wantLimit, sql)
			}

			if !strings.Contains(sql, tt.wantOffset) {
				t.Errorf("BuildPage() missing %q, got %q", tt.wantOffset, sql)
			}
		})
	}
}

func TestBuilder_BuildSingle(t *testing.T) {
	pm := newTestProjection()
	b := query.NewBuilder(pm, "Name")

	sql, args := b.BuildSingle("Id", 123)

	if !strings.Contains(sql, "WHERE u.id = $1") {
		t.Errorf("BuildSingle() missing where clause, got %q", sql)
	}

	if len(args) != 1 {
		t.Fatalf("BuildSingle() len(args) = %d, want 1", len(args))
	}

	if args[0] != 123 {
		t.Errorf("BuildSingle() args[0] = %v, want 123", args[0])
	}
}

func TestBuilder_OrderBy(t *testing.T) {
	pm := newTestProjection()

	tests := []struct {
		name       string
		field      string
		descending bool
		wantOrder  string
	}{
		{"ascending by name", "Name", false, "ORDER BY u.name ASC"},
		{"descending by name", "Name", true, "ORDER BY u.name DESC"},
		{"ascending by email", "Email", false, "ORDER BY u.email ASC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := query.NewBuilder(pm, "Name").OrderBy(tt.field, tt.descending)
			sql, _ := b.BuildPage(1, 20)

			if !strings.Contains(sql, tt.wantOrder) {
				t.Errorf("BuildPage() missing %q, got %q", tt.wantOrder, sql)
			}
		})
	}
}

func TestBuilder_OrderBy_EmptyFieldUsesDefault(t *testing.T) {
	pm := newTestProjection()
	b := query.NewBuilder(pm, "Name").OrderBy("", false)

	sql, _ := b.BuildPage(1, 20)

	if !strings.Contains(sql, "ORDER BY u.name ASC") {
		t.Errorf("BuildPage() should use default sort, got %q", sql)
	}
}

func TestBuilder_WhereEquals(t *testing.T) {
	pm := newTestProjection()
	b := query.NewBuilder(pm, "Name").WhereEquals("Id", 5)

	sql, args := b.BuildCount()

	if !strings.Contains(sql, "WHERE u.id = $1") {
		t.Errorf("BuildCount() missing where clause, got %q", sql)
	}

	if len(args) != 1 || args[0] != 5 {
		t.Errorf("BuildCount() args = %v, want [5]", args)
	}
}

func TestBuilder_WhereEquals_NilIgnored(t *testing.T) {
	pm := newTestProjection()
	b := query.NewBuilder(pm, "Name").WhereEquals("Id", nil)

	sql, args := b.BuildCount()

	if strings.Contains(sql, "WHERE") {
		t.Errorf("BuildCount() should not have WHERE for nil, got %q", sql)
	}

	if len(args) != 0 {
		t.Errorf("BuildCount() args = %v, want empty", args)
	}
}

func TestBuilder_WhereContains(t *testing.T) {
	pm := newTestProjection()
	name := "test"
	b := query.NewBuilder(pm, "Name").WhereContains("Name", &name)

	sql, args := b.BuildCount()

	if !strings.Contains(sql, "WHERE u.name ILIKE $1") {
		t.Errorf("BuildCount() missing ILIKE clause, got %q", sql)
	}

	if len(args) != 1 || args[0] != "%test%" {
		t.Errorf("BuildCount() args = %v, want [%%test%%]", args)
	}
}

func TestBuilder_WhereContains_NilIgnored(t *testing.T) {
	pm := newTestProjection()
	b := query.NewBuilder(pm, "Name").WhereContains("Name", nil)

	sql, args := b.BuildCount()

	if strings.Contains(sql, "WHERE") {
		t.Errorf("BuildCount() should not have WHERE for nil, got %q", sql)
	}

	if len(args) != 0 {
		t.Errorf("BuildCount() args = %v, want empty", args)
	}
}

func TestBuilder_WhereContains_EmptyStringIgnored(t *testing.T) {
	pm := newTestProjection()
	empty := ""
	b := query.NewBuilder(pm, "Name").WhereContains("Name", &empty)

	sql, args := b.BuildCount()

	if strings.Contains(sql, "WHERE") {
		t.Errorf("BuildCount() should not have WHERE for empty string, got %q", sql)
	}

	if len(args) != 0 {
		t.Errorf("BuildCount() args = %v, want empty", args)
	}
}

func TestBuilder_WhereIn(t *testing.T) {
	pm := newTestProjection()
	b := query.NewBuilder(pm, "Name").WhereIn("Id", []any{1, 2, 3})

	sql, args := b.BuildCount()

	if !strings.Contains(sql, "WHERE u.id IN ($1, $2, $3)") {
		t.Errorf("BuildCount() missing IN clause, got %q", sql)
	}

	if len(args) != 3 {
		t.Errorf("BuildCount() len(args) = %d, want 3", len(args))
	}
}

func TestBuilder_WhereIn_EmptyIgnored(t *testing.T) {
	pm := newTestProjection()
	b := query.NewBuilder(pm, "Name").WhereIn("Id", []any{})

	sql, args := b.BuildCount()

	if strings.Contains(sql, "WHERE") {
		t.Errorf("BuildCount() should not have WHERE for empty slice, got %q", sql)
	}

	if len(args) != 0 {
		t.Errorf("BuildCount() args = %v, want empty", args)
	}
}

func TestBuilder_WhereSearch(t *testing.T) {
	pm := newTestProjection()
	search := "test"
	b := query.NewBuilder(pm, "Name").WhereSearch(&search, "Name", "Email")

	sql, args := b.BuildCount()

	if !strings.Contains(sql, "u.name ILIKE $1") {
		t.Errorf("BuildCount() missing first search field, got %q", sql)
	}

	if !strings.Contains(sql, "u.email ILIKE $2") {
		t.Errorf("BuildCount() missing second search field, got %q", sql)
	}

	if !strings.Contains(sql, " OR ") {
		t.Errorf("BuildCount() missing OR connector, got %q", sql)
	}

	if len(args) != 2 {
		t.Errorf("BuildCount() len(args) = %d, want 2", len(args))
	}
}

func TestBuilder_WhereSearch_NilIgnored(t *testing.T) {
	pm := newTestProjection()
	b := query.NewBuilder(pm, "Name").WhereSearch(nil, "Name", "Email")

	sql, args := b.BuildCount()

	if strings.Contains(sql, "WHERE") {
		t.Errorf("BuildCount() should not have WHERE for nil search, got %q", sql)
	}

	if len(args) != 0 {
		t.Errorf("BuildCount() args = %v, want empty", args)
	}
}

func TestBuilder_MultipleConditions(t *testing.T) {
	pm := newTestProjection()
	name := "john"
	b := query.NewBuilder(pm, "Name").
		WhereEquals("Id", 5).
		WhereContains("Name", &name)

	sql, args := b.BuildCount()

	if !strings.Contains(sql, "u.id = $1") {
		t.Errorf("BuildCount() missing first condition, got %q", sql)
	}

	if !strings.Contains(sql, "u.name ILIKE $2") {
		t.Errorf("BuildCount() missing second condition, got %q", sql)
	}

	if !strings.Contains(sql, " AND ") {
		t.Errorf("BuildCount() missing AND connector, got %q", sql)
	}

	if len(args) != 2 {
		t.Errorf("BuildCount() len(args) = %d, want 2", len(args))
	}

	if args[0] != 5 {
		t.Errorf("BuildCount() args[0] = %v, want 5", args[0])
	}

	if args[1] != "%john%" {
		t.Errorf("BuildCount() args[1] = %v, want %%john%%", args[1])
	}
}
