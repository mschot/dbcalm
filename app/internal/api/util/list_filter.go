package util

// ListFilter contains common filtering/pagination options for list endpoints
type ListFilter struct {
	// Filters parsed from query parameter
	Filters []QueryFilter
	// Order by clauses parsed from order parameter
	Order []OrderClause
	// Pagination
	Page    int
	PerPage int
}
