package query

import (
	"fmt"
	"strings"
)

type condition struct {
	clause string
	args   []any
}

// Builder constructs SQL queries using a fluent API with automatic parameter numbering.
type Builder struct {
	projection  *ProjectionMap
	conditions  []condition
	orderBy     string
	descending  bool
	defaultSort string
}

// NewBuilder creates a Builder for the given projection with a default sort field.
func NewBuilder(projection *ProjectionMap, defaultSort string) *Builder {
	return &Builder{
		projection:  projection,
		conditions:  make([]condition, 0),
		defaultSort: defaultSort,
	}
}

// BuildCount returns a COUNT(*) query with the current conditions.
func (b *Builder) BuildCount() (string, []any) {
	where, args, _ := b.buildWhere(1)
	sql := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", b.projection.Table(), where)
	return sql, args
}

// BuildPage returns a paginated SELECT query with ordering, limit, and offset.
func (b *Builder) BuildPage(page, pageSize int) (string, []any) {
	where, args, _ := b.buildWhere(1)
	orderBy := b.buildOrderBy()
	offset := (page - 1) * pageSize

	sql := fmt.Sprintf(
		"SELECT %s FROM %s%s%s LIMIT %d OFFSET %d",
		b.projection.Columns(),
		b.projection.Table(),
		where,
		orderBy,
		pageSize,
		offset,
	)

	return sql, args
}

// BuildSingle returns a SELECT query for a single record by ID.
func (b *Builder) BuildSingle(idField string, id any) (string, []any) {
	col := b.projection.Column(idField)
	sql := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s = $1",
		b.projection.Columns(),
		b.projection.Table(),
		col,
	)
	return sql, []any{id}
}

// OrderBy sets the sort field and direction. Empty field uses the default sort.
func (b *Builder) OrderBy(field string, descending bool) *Builder {
	if field != "" {
		b.orderBy = b.projection.Column(field)
	}
	b.descending = descending
	return b
}

// WhereContains adds a case-insensitive ILIKE condition. Nil or empty values are ignored.
func (b *Builder) WhereContains(field string, value *string) *Builder {
	if value == nil || *value == "" {
		return b
	}
	col := b.projection.Column(field)
	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s ILIKE $%%d", col),
		args:   []any{"%" + *value + "%"},
	})
	return b
}

// WhereEquals adds an equality condition. Nil values are ignored.
func (b *Builder) WhereEquals(field string, value any) *Builder {
	if value == nil {
		return b
	}
	col := b.projection.Column(field)
	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s = $%%d", col),
		args:   []any{value},
	})
	return b
}

// WhereIn adds an IN condition for multiple values. Empty slices are ignored.
func (b *Builder) WhereIn(field string, values []any) *Builder {
	if len(values) == 0 {
		return b
	}
	col := b.projection.Column(field)
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = "$%d"
	}
	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", ")),
		args:   values,
	})
	return b
}

// WhereSearch adds an OR condition across multiple fields with ILIKE. Nil or empty search is ignored.
func (b *Builder) WhereSearch(search *string, fields ...string) *Builder {
	if search == nil || *search == "" || len(fields) == 0 {
		return b
	}

	clauses := make([]string, len(fields))
	args := make([]any, len(fields))
	searchPattern := "%" + *search + "%"

	for i, field := range fields {
		col := b.projection.Column(field)
		clauses[i] = fmt.Sprintf("%s ILIKE $%%d", col)
		args[i] = searchPattern
	}

	b.conditions = append(b.conditions, condition{
		clause: "(" + strings.Join(clauses, " OR ") + ")",
		args:   args,
	})
	return b
}

func (b *Builder) buildOrderBy() string {
	orderCol := b.orderBy
	if orderCol == "" {
		orderCol = b.projection.Column(b.defaultSort)
	}

	dir := "ASC"
	if b.descending {
		dir = "DESC"
	}

	return fmt.Sprintf(" ORDER BY %s %s", orderCol, dir)
}

func (b *Builder) buildWhere(startParam int) (string, []any, int) {
	if len(b.conditions) == 0 {
		return "", nil, startParam
	}

	clauses := make([]string, 0, len(b.conditions))
	args := make([]any, 0)
	paramIdx := startParam

	for _, cond := range b.conditions {
		clause := cond.clause
		for _, arg := range cond.args {
			clause = strings.Replace(clause, "$%d", fmt.Sprintf("$%d", paramIdx), 1)
			args = append(args, arg)
			paramIdx++
		}
		clauses = append(clauses, clause)
	}

	return " WHERE " + strings.Join(clauses, " AND "), args, paramIdx
}
