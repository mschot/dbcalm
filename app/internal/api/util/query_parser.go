package util

import (
	"fmt"
	"strings"
)

// QueryOperator represents a filter operator
type QueryOperator string

const (
	OpEq        QueryOperator = "eq"
	OpNe        QueryOperator = "ne"
	OpGt        QueryOperator = "gt"
	OpGte       QueryOperator = "gte"
	OpLt        QueryOperator = "lt"
	OpLte       QueryOperator = "lte"
	OpIn        QueryOperator = "in"
	OpNin       QueryOperator = "nin"
	OpIsNull    QueryOperator = "isnull"
	OpIsNotNull QueryOperator = "isnotnull"
)

// QueryFilter represents a single filter condition
type QueryFilter struct {
	Field    string
	Operator QueryOperator
	Value    interface{} // string or []string for in/nin
}

// OrderDirection represents sort direction
type OrderDirection string

const (
	OrderAsc  OrderDirection = "asc"
	OrderDesc OrderDirection = "desc"
)

// OrderClause represents a single order by clause
type OrderClause struct {
	Field     string
	Direction OrderDirection
}

var validOperators = map[string]QueryOperator{
	"eq":        OpEq,
	"ne":        OpNe,
	"gt":        OpGt,
	"gte":       OpGte,
	"lt":        OpLt,
	"lte":       OpLte,
	"in":        OpIn,
	"nin":       OpNin,
	"isnull":    OpIsNull,
	"isnotnull": OpIsNotNull,
}

// ParseQueryString parses a query string into filter conditions.
// Supports formats:
//   - field|value (defaults to eq operator)
//   - field|isnull or field|isnotnull (null checks)
//   - field|operator|value (explicit operator)
//
// Multiple conditions are comma-separated.
func ParseQueryString(queryStr string) ([]QueryFilter, error) {
	if queryStr == "" {
		return nil, nil
	}

	var filters []QueryFilter

	for _, pair := range strings.Split(queryStr, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.Split(pair, "|")

		switch len(parts) {
		case 2:
			// Could be field|value (eq) or field|isnull/isnotnull
			potentialOp := strings.ToLower(parts[1])
			if potentialOp == "isnull" || potentialOp == "isnotnull" {
				filters = append(filters, QueryFilter{
					Field:    parts[0],
					Operator: QueryOperator(potentialOp),
					Value:    nil,
				})
			} else {
				// Default to equality
				filters = append(filters, QueryFilter{
					Field:    parts[0],
					Operator: OpEq,
					Value:    parts[1],
				})
			}

		case 3:
			// field|operator|value
			opStr := strings.ToLower(parts[1])
			op, valid := validOperators[opStr]
			if !valid {
				return nil, fmt.Errorf("invalid operator: %s", opStr)
			}

			var value interface{}
			if op == OpIn || op == OpNin {
				// Split value by comma for list operators
				value = strings.Split(parts[2], ",")
			} else {
				value = parts[2]
			}

			filters = append(filters, QueryFilter{
				Field:    parts[0],
				Operator: op,
				Value:    value,
			})

		default:
			return nil, fmt.Errorf("invalid query format: %s (expected field|value or field|operator|value)", pair)
		}
	}

	return filters, nil
}

// ParseOrderString parses an order string into order clauses.
// Format: field|direction (direction is asc or desc)
// Multiple clauses are comma-separated.
func ParseOrderString(orderStr string) ([]OrderClause, error) {
	if orderStr == "" {
		return nil, nil
	}

	var orders []OrderClause

	for _, pair := range strings.Split(orderStr, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.Split(pair, "|")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid order format: %s (expected field|direction)", pair)
		}

		direction := strings.ToLower(parts[1])
		if direction != "asc" && direction != "desc" {
			return nil, fmt.Errorf("invalid order direction: %s (expected asc or desc)", direction)
		}

		orders = append(orders, OrderClause{
			Field:     parts[0],
			Direction: OrderDirection(direction),
		})
	}

	return orders, nil
}

// ValidateFilterFields validates that all filter fields are in the allowed set
func ValidateFilterFields(filters []QueryFilter, allowedFields []string) error {
	allowed := make(map[string]bool)
	for _, f := range allowedFields {
		allowed[f] = true
	}

	for _, filter := range filters {
		if !allowed[filter.Field] {
			return fmt.Errorf("invalid query field: %s (valid fields: %s)", filter.Field, strings.Join(allowedFields, ", "))
		}
	}

	return nil
}

// ValidateOrderFields validates that all order fields are in the allowed set
func ValidateOrderFields(orders []OrderClause, allowedFields []string) error {
	allowed := make(map[string]bool)
	for _, f := range allowedFields {
		allowed[f] = true
	}

	for _, order := range orders {
		if !allowed[order.Field] {
			return fmt.Errorf("invalid order field: %s (valid fields: %s)", order.Field, strings.Join(allowedFields, ", "))
		}
	}

	return nil
}
