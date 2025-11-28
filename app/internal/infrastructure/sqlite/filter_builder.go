package sqlite

import (
	"fmt"
	"strings"

	"github.com/martijn/dbcalm/internal/api/util"
)

// BuildFilterClause builds a SQL WHERE clause from a QueryFilter
func BuildFilterClause(f util.QueryFilter) (string, []interface{}) {
	switch f.Operator {
	case util.OpEq:
		return fmt.Sprintf("%s = ?", f.Field), []interface{}{f.Value}
	case util.OpNe:
		return fmt.Sprintf("%s != ?", f.Field), []interface{}{f.Value}
	case util.OpGt:
		return fmt.Sprintf("%s > ?", f.Field), []interface{}{f.Value}
	case util.OpGte:
		return fmt.Sprintf("%s >= ?", f.Field), []interface{}{f.Value}
	case util.OpLt:
		return fmt.Sprintf("%s < ?", f.Field), []interface{}{f.Value}
	case util.OpLte:
		return fmt.Sprintf("%s <= ?", f.Field), []interface{}{f.Value}
	case util.OpIsNull:
		return fmt.Sprintf("%s IS NULL", f.Field), nil
	case util.OpIsNotNull:
		return fmt.Sprintf("%s IS NOT NULL", f.Field), nil
	case util.OpIn:
		if values, ok := f.Value.([]string); ok && len(values) > 0 {
			placeholders := make([]string, len(values))
			args := make([]interface{}, len(values))
			for i, v := range values {
				placeholders[i] = "?"
				args[i] = v
			}
			return fmt.Sprintf("%s IN (%s)", f.Field, strings.Join(placeholders, ", ")), args
		}
		return "", nil
	case util.OpNin:
		if values, ok := f.Value.([]string); ok && len(values) > 0 {
			placeholders := make([]string, len(values))
			args := make([]interface{}, len(values))
			for i, v := range values {
				placeholders[i] = "?"
				args[i] = v
			}
			return fmt.Sprintf("%s NOT IN (%s)", f.Field, strings.Join(placeholders, ", ")), args
		}
		return "", nil
	default:
		return "", nil
	}
}

// ApplyFilters applies QueryFilters to a query and returns the modified query and args
func ApplyFilters(query string, args []interface{}, filters []util.QueryFilter) (string, []interface{}) {
	for _, f := range filters {
		clause, filterArgs := BuildFilterClause(f)
		if clause != "" {
			query += " AND " + clause
			args = append(args, filterArgs...)
		}
	}
	return query, args
}

// ApplyOrdering applies OrderClauses to a query
func ApplyOrdering(query string, orders []util.OrderClause, defaultOrder string) string {
	if len(orders) > 0 {
		orderClauses := make([]string, 0, len(orders))
		for _, o := range orders {
			direction := "ASC"
			if o.Direction == util.OrderDesc {
				direction = "DESC"
			}
			orderClauses = append(orderClauses, fmt.Sprintf("%s %s", o.Field, direction))
		}
		return query + " ORDER BY " + strings.Join(orderClauses, ", ")
	}
	return query + " ORDER BY " + defaultOrder
}

// ApplyPagination applies page/perPage to a query
func ApplyPagination(query string, args []interface{}, page, perPage int) (string, []interface{}) {
	if perPage > 0 {
		query += " LIMIT ?"
		args = append(args, perPage)

		if page > 1 {
			offset := (page - 1) * perPage
			query += " OFFSET ?"
			args = append(args, offset)
		}
	}
	return query, args
}
