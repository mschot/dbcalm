package sqlite

import (
	"fmt"
	"strings"
	"time"

	"github.com/martijn/dbcalm/internal/api/util"
)

// datetimeFields defines fields that contain datetime values and need normalization
var datetimeFields = map[string]bool{
	"start_time":       true,
	"end_time":         true,
	"backup_timestamp": true,
	"created_at":       true,
	"updated_at":       true,
}

// isDatetimeField checks if a field is a datetime field
func isDatetimeField(field string) bool {
	return datetimeFields[field]
}

// normalizeDateTime attempts to parse and normalize datetime strings for consistent comparison
// Python's Peewee ORM stores datetime as "2025-11-24 14:00:00" (with space, no timezone)
// Go's modernc/sqlite stores as "2025-11-24T14:00:00.123456789Z" (with T and timezone)
// User input like "2025-11-24T00:00" needs to be normalized to work with both formats.
// We use space separator format since it compares correctly with both stored formats:
// - "2025-11-24 00:00:00" <= "2025-11-24 14:00:00" (Python format) ✓
// - "2025-11-24 00:00:00" <= "2025-11-24T14:00:00.123456789Z" (Go format) ✓
//   (space ASCII 32 < T ASCII 84)
func normalizeDateTime(value string) string {
	// Try various input formats that users might provide
	formats := []string{
		time.RFC3339Nano,          // 2006-01-02T15:04:05.999999999Z07:00
		time.RFC3339,              // 2006-01-02T15:04:05Z07:00
		"2006-01-02T15:04:05",     // Without timezone
		"2006-01-02T15:04",        // Without seconds
		"2006-01-02 15:04:05",     // Space separator with seconds
		"2006-01-02 15:04",        // Space separator without seconds
		"2006-01-02",              // Date only
	}

	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			// Return in space-separated format (like Python's Peewee uses)
			// This compares correctly with both Python and Go stored formats
			return t.UTC().Format("2006-01-02 15:04:05")
		}
	}

	// If parsing fails, return original value
	return value
}

// BuildFilterClause builds a SQL WHERE clause from a QueryFilter
func BuildFilterClause(f util.QueryFilter) (string, []interface{}) {
	// Normalize datetime values for consistent string comparison in SQLite
	value := f.Value
	if isDatetimeField(f.Field) {
		if strVal, ok := value.(string); ok {
			value = normalizeDateTime(strVal)
		}
	}

	switch f.Operator {
	case util.OpEq:
		return fmt.Sprintf("%s = ?", f.Field), []interface{}{value}
	case util.OpNe:
		return fmt.Sprintf("%s != ?", f.Field), []interface{}{value}
	case util.OpGt:
		return fmt.Sprintf("%s > ?", f.Field), []interface{}{value}
	case util.OpGte:
		return fmt.Sprintf("%s >= ?", f.Field), []interface{}{value}
	case util.OpLt:
		return fmt.Sprintf("%s < ?", f.Field), []interface{}{value}
	case util.OpLte:
		return fmt.Sprintf("%s <= ?", f.Field), []interface{}{value}
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
