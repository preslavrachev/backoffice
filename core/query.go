package core

import (
	"os"
	"strconv"
	"time"
)

// Constants for pagination
const (
	DefaultPageSize = 10
	MaxPageSize     = 100
)

// SortDirection represents the sort order
type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

// SortPrecedence represents the precedence of sort configuration
type SortPrecedence int

const (
	SortPrecedenceNone          SortPrecedence = iota // Not configured (use auto-detection)
	SortPrecedenceExplicit                            // Explicitly configured via WithDefaultSort
	SortPrecedenceAutoCreatedAt                       // Auto-detected CreatedAt field
	SortPrecedenceAutoID                              // Fallback to ID field
)

// SortField represents a field to sort by with precedence tracking
type SortField struct {
	Field      string         `json:"field"`
	Direction  SortDirection  `json:"direction"`
	Precedence SortPrecedence `json:"precedence"`
}

// Pagination represents pagination parameters
type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// Query represents a comprehensive query with filters, sorting, and pagination
type Query struct {
	Filters    map[string]any `json:"filters"`
	Sort       []SortField    `json:"sort"`
	Pagination Pagination     `json:"pagination"`
}

// Result represents paginated query results
type Result struct {
	Items      []any `json:"items"`
	TotalCount int64 `json:"total_count"`
	HasMore    bool  `json:"has_more"`
	Query      Query `json:"query"`
}

// CreatedAtProvider interface for entities that have a CreatedAt field
type CreatedAtProvider interface {
	GetCreatedAt() time.Time
}

// NewQuery creates a new Query with default pagination
func NewQuery() *Query {
	pageSize := getPageSizeFromEnv()
	return &Query{
		Filters: make(map[string]any),
		Sort:    []SortField{},
		Pagination: Pagination{
			Limit:  pageSize,
			Offset: 0,
		},
	}
}

// WithFilters adds filters to the query
func (q *Query) WithFilters(filters map[string]any) *Query {
	for k, v := range filters {
		q.Filters[k] = v
	}
	return q
}

// WithSort adds a sort field to the query
func (q *Query) WithSort(field string, direction SortDirection) *Query {
	q.Sort = append(q.Sort, SortField{
		Field:     field,
		Direction: direction,
	})
	return q
}

// WithPagination sets pagination parameters
func (q *Query) WithPagination(limit, offset int) *Query {
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	if limit <= 0 {
		limit = getPageSizeFromEnv()
	}
	if offset < 0 {
		offset = 0
	}

	q.Pagination.Limit = limit
	q.Pagination.Offset = offset
	return q
}

// NextPage creates a new query for the next page
func (q *Query) NextPage() *Query {
	nextQuery := &Query{
		Filters:    make(map[string]any),
		Sort:       make([]SortField, len(q.Sort)),
		Pagination: q.Pagination,
	}

	// Copy filters
	for k, v := range q.Filters {
		nextQuery.Filters[k] = v
	}

	// Copy sort fields
	copy(nextQuery.Sort, q.Sort)

	// Advance pagination
	nextQuery.Pagination.Offset += nextQuery.Pagination.Limit

	return nextQuery
}

// GetCurrentPage returns the current page number (1-indexed)
func (q *Query) GetCurrentPage() int {
	if q.Pagination.Limit <= 0 {
		return 1
	}
	return (q.Pagination.Offset / q.Pagination.Limit) + 1
}

// HasFilters returns true if the query has any filters
func (q *Query) HasFilters() bool {
	return len(q.Filters) > 0
}

// HasSort returns true if the query has sorting
func (q *Query) HasSort() bool {
	return len(q.Sort) > 0
}

// GetPrimarySort returns the first sort field, or nil if none
func (q *Query) GetPrimarySort() *SortField {
	if len(q.Sort) > 0 {
		return &q.Sort[0]
	}
	return nil
}

// ApplyDefaultSort applies default sorting if no sort is specified
func (q *Query) ApplyDefaultSort(resource *Resource) {
	if q.HasSort() {
		return // Already has sorting
	}

	// Get the effective default sort for this resource
	defaultSort := resource.GetEffectiveDefaultSort()
	q.WithSort(defaultSort.Field, defaultSort.Direction)
}

// getPageSizeFromEnv gets page size from environment variable or default
func getPageSizeFromEnv() int {
	if envSize := os.Getenv("BACKOFFICE_PAGE_SIZE"); envSize != "" {
		if size, err := strconv.Atoi(envSize); err == nil && size > 0 && size <= MaxPageSize {
			return size
		}
	}
	return DefaultPageSize
}

// String returns a string representation of the sort direction
func (sd SortDirection) String() string {
	return string(sd)
}

// IsValid checks if the sort direction is valid
func (sd SortDirection) IsValid() bool {
	return sd == SortAsc || sd == SortDesc
}

// Opposite returns the opposite sort direction
func (sd SortDirection) Opposite() SortDirection {
	if sd == SortAsc {
		return SortDesc
	}
	return SortAsc
}
