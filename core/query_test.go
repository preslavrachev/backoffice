package core

import (
	"os"
	"testing"
	"time"
)

func TestNewQuery(t *testing.T) {
	query := NewQuery()

	if query == nil {
		t.Fatal("NewQuery() returned nil")
	}

	if query.Filters == nil {
		t.Error("Query.Filters should be initialized")
	}

	if len(query.Sort) != 0 {
		t.Error("Query.Sort should be empty initially")
	}

	if query.Pagination.Limit != DefaultPageSize {
		t.Errorf("Expected default page size %d, got %d", DefaultPageSize, query.Pagination.Limit)
	}

	if query.Pagination.Offset != 0 {
		t.Error("Query.Pagination.Offset should be 0 initially")
	}
}

func TestQueryWithFilters(t *testing.T) {
	query := NewQuery()
	filters := map[string]any{"name": "test", "active": true}

	result := query.WithFilters(filters)

	// Should return same instance for chaining
	if result != query {
		t.Error("WithFilters should return same instance for chaining")
	}

	if len(query.Filters) != 2 {
		t.Errorf("Expected 2 filters, got %d", len(query.Filters))
	}

	if query.Filters["name"] != "test" {
		t.Error("Filter 'name' not set correctly")
	}

	if query.Filters["active"] != true {
		t.Error("Filter 'active' not set correctly")
	}
}

func TestQueryWithSort(t *testing.T) {
	query := NewQuery()

	result := query.WithSort("name", SortAsc)

	// Should return same instance for chaining
	if result != query {
		t.Error("WithSort should return same instance for chaining")
	}

	if len(query.Sort) != 1 {
		t.Errorf("Expected 1 sort field, got %d", len(query.Sort))
	}

	if query.Sort[0].Field != "name" {
		t.Errorf("Expected sort field 'name', got '%s'", query.Sort[0].Field)
	}

	if query.Sort[0].Direction != SortAsc {
		t.Errorf("Expected sort direction ASC, got %s", query.Sort[0].Direction)
	}

	// Test multiple sorts
	query.WithSort("created_at", SortDesc)

	if len(query.Sort) != 2 {
		t.Errorf("Expected 2 sort fields, got %d", len(query.Sort))
	}

	if query.Sort[1].Field != "created_at" {
		t.Errorf("Expected second sort field 'created_at', got '%s'", query.Sort[1].Field)
	}
}

func TestQueryWithPagination(t *testing.T) {
	query := NewQuery()

	result := query.WithPagination(20, 10)

	// Should return same instance for chaining
	if result != query {
		t.Error("WithPagination should return same instance for chaining")
	}

	if query.Pagination.Limit != 20 {
		t.Errorf("Expected limit 20, got %d", query.Pagination.Limit)
	}

	if query.Pagination.Offset != 10 {
		t.Errorf("Expected offset 10, got %d", query.Pagination.Offset)
	}
}

func TestQueryPaginationLimits(t *testing.T) {
	query := NewQuery()

	// Test max limit enforcement
	query.WithPagination(MaxPageSize+10, 0)
	if query.Pagination.Limit != MaxPageSize {
		t.Errorf("Expected limit to be capped at %d, got %d", MaxPageSize, query.Pagination.Limit)
	}

	// Test negative limit
	query.WithPagination(-5, 0)
	if query.Pagination.Limit != DefaultPageSize {
		t.Errorf("Expected negative limit to default to %d, got %d", DefaultPageSize, query.Pagination.Limit)
	}

	// Test negative offset
	query.WithPagination(10, -5)
	if query.Pagination.Offset != 0 {
		t.Errorf("Expected negative offset to be set to 0, got %d", query.Pagination.Offset)
	}
}

func TestQueryNextPage(t *testing.T) {
	query := NewQuery()
	query.WithFilters(map[string]any{"name": "test"})
	query.WithSort("created_at", SortDesc)
	query.WithPagination(10, 0)

	nextQuery := query.NextPage()

	// Should be a new instance
	if nextQuery == query {
		t.Error("NextPage should return a new instance")
	}

	// Should copy filters
	if len(nextQuery.Filters) != 1 || nextQuery.Filters["name"] != "test" {
		t.Error("NextPage should copy filters")
	}

	// Should copy sort
	if len(nextQuery.Sort) != 1 || nextQuery.Sort[0].Field != "created_at" {
		t.Error("NextPage should copy sort fields")
	}

	// Should advance pagination
	if nextQuery.Pagination.Offset != 10 {
		t.Errorf("Expected next page offset to be 10, got %d", nextQuery.Pagination.Offset)
	}

	if nextQuery.Pagination.Limit != 10 {
		t.Errorf("Expected next page limit to remain 10, got %d", nextQuery.Pagination.Limit)
	}
}

func TestQueryGetCurrentPage(t *testing.T) {
	query := NewQuery()

	// First page
	query.WithPagination(10, 0)
	if page := query.GetCurrentPage(); page != 1 {
		t.Errorf("Expected page 1, got %d", page)
	}

	// Second page
	query.WithPagination(10, 10)
	if page := query.GetCurrentPage(); page != 2 {
		t.Errorf("Expected page 2, got %d", page)
	}

	// Third page
	query.WithPagination(10, 20)
	if page := query.GetCurrentPage(); page != 3 {
		t.Errorf("Expected page 3, got %d", page)
	}

	// Edge case: zero limit should return page 1
	query.WithPagination(0, 0)
	if page := query.GetCurrentPage(); page != 1 {
		t.Errorf("Expected page 1 for zero limit, got %d", page)
	}
}

func TestQueryHasFilters(t *testing.T) {
	query := NewQuery()

	if query.HasFilters() {
		t.Error("Empty query should not have filters")
	}

	query.WithFilters(map[string]any{"name": "test"})

	if !query.HasFilters() {
		t.Error("Query with filters should return true for HasFilters()")
	}
}

func TestQueryHasSort(t *testing.T) {
	query := NewQuery()

	if query.HasSort() {
		t.Error("Empty query should not have sort")
	}

	query.WithSort("name", SortAsc)

	if !query.HasSort() {
		t.Error("Query with sort should return true for HasSort()")
	}
}

func TestQueryGetPrimarySort(t *testing.T) {
	query := NewQuery()

	// No sort
	if sort := query.GetPrimarySort(); sort != nil {
		t.Error("Query without sort should return nil for GetPrimarySort()")
	}

	// With sort
	query.WithSort("name", SortAsc)
	query.WithSort("created_at", SortDesc)

	primarySort := query.GetPrimarySort()
	if primarySort == nil {
		t.Error("Query with sort should return non-nil for GetPrimarySort()")
	}

	if primarySort.Field != "name" {
		t.Errorf("Expected primary sort field 'name', got '%s'", primarySort.Field)
	}

	if primarySort.Direction != SortAsc {
		t.Errorf("Expected primary sort direction ASC, got %s", primarySort.Direction)
	}
}

func TestSortDirection(t *testing.T) {
	// Test string representation
	if SortAsc.String() != "asc" {
		t.Errorf("Expected SortAsc.String() to be 'asc', got '%s'", SortAsc.String())
	}

	if SortDesc.String() != "desc" {
		t.Errorf("Expected SortDesc.String() to be 'desc', got '%s'", SortDesc.String())
	}

	// Test IsValid
	if !SortAsc.IsValid() {
		t.Error("SortAsc should be valid")
	}

	if !SortDesc.IsValid() {
		t.Error("SortDesc should be valid")
	}

	if SortDirection("invalid").IsValid() {
		t.Error("Invalid sort direction should not be valid")
	}

	// Test Opposite
	if SortAsc.Opposite() != SortDesc {
		t.Error("SortAsc.Opposite() should be SortDesc")
	}

	if SortDesc.Opposite() != SortAsc {
		t.Error("SortDesc.Opposite() should be SortAsc")
	}
}

func TestApplyDefaultSort(t *testing.T) {
	// Test with resource that has CreatedAt field
	resourceWithCreatedAt := &Resource{
		Fields: []FieldInfo{
			{Name: "ID", Type: "uint"},
			{Name: "Name", Type: "string"},
			{Name: "CreatedAt", Type: "time.Time"},
		},
		IDField: "ID",
	}

	query := NewQuery()
	query.ApplyDefaultSort(resourceWithCreatedAt)

	if !query.HasSort() {
		t.Error("Query should have sort after ApplyDefaultSort")
	}

	primarySort := query.GetPrimarySort()
	if primarySort.Field != "CreatedAt" {
		t.Errorf("Expected default sort by CreatedAt, got %s", primarySort.Field)
	}

	if primarySort.Direction != SortDesc {
		t.Errorf("Expected default CreatedAt sort to be DESC, got %s", primarySort.Direction)
	}

	// Test with resource without CreatedAt field
	resourceWithoutCreatedAt := &Resource{
		Fields: []FieldInfo{
			{Name: "ID", Type: "uint"},
			{Name: "Name", Type: "string"},
		},
		IDField: "ID",
	}

	query2 := NewQuery()
	query2.ApplyDefaultSort(resourceWithoutCreatedAt)

	if !query2.HasSort() {
		t.Error("Query should have sort after ApplyDefaultSort")
	}

	primarySort2 := query2.GetPrimarySort()
	if primarySort2.Field != "ID" {
		t.Errorf("Expected default sort by ID, got %s", primarySort2.Field)
	}

	if primarySort2.Direction != SortAsc {
		t.Errorf("Expected default ID sort to be ASC, got %s", primarySort2.Direction)
	}

	// Test with resource that has configured default sort
	resourceWithDefaultSort := &Resource{
		Fields: []FieldInfo{
			{Name: "ID", Type: "uint"},
			{Name: "Name", Type: "string"},
			{Name: "Price", Type: "float64"},
		},
		IDField: "ID",
		DefaultSort: SortField{
			Field:      "Price",
			Direction:  SortDesc,
			Precedence: SortPrecedenceExplicit,
		},
	}

	query3 := NewQuery()
	query3.ApplyDefaultSort(resourceWithDefaultSort)

	primarySort3 := query3.GetPrimarySort()
	if primarySort3.Field != "Price" {
		t.Errorf("Expected configured default sort by Price, got %s", primarySort3.Field)
	}

	if primarySort3.Direction != SortDesc {
		t.Errorf("Expected configured default sort to be DESC, got %s", primarySort3.Direction)
	}

	// Test that existing sort is not overridden
	query4 := NewQuery()
	query4.WithSort("Name", SortAsc)
	query4.ApplyDefaultSort(resourceWithDefaultSort)

	if len(query4.Sort) != 1 {
		t.Error("ApplyDefaultSort should not override existing sort")
	}

	primarySort4 := query4.GetPrimarySort()
	if primarySort4.Field != "Name" {
		t.Error("ApplyDefaultSort should not override existing sort field")
	}
}

func TestGetPageSizeFromEnv(t *testing.T) {
	// Test default
	if size := getPageSizeFromEnv(); size != DefaultPageSize {
		t.Errorf("Expected default page size %d, got %d", DefaultPageSize, size)
	}

	// Test with valid env var
	os.Setenv("BACKOFFICE_PAGE_SIZE", "25")
	defer os.Unsetenv("BACKOFFICE_PAGE_SIZE")

	if size := getPageSizeFromEnv(); size != 25 {
		t.Errorf("Expected page size from env var 25, got %d", size)
	}

	// Test with invalid env var
	os.Setenv("BACKOFFICE_PAGE_SIZE", "invalid")
	if size := getPageSizeFromEnv(); size != DefaultPageSize {
		t.Errorf("Expected default page size for invalid env var, got %d", size)
	}

	// Test with too large env var
	os.Setenv("BACKOFFICE_PAGE_SIZE", "1000")
	if size := getPageSizeFromEnv(); size != DefaultPageSize {
		t.Errorf("Expected default page size for too large env var, got %d", size)
	}
}

func TestResultStruct(t *testing.T) {
	query := NewQuery()
	query.WithFilters(map[string]any{"name": "test"})

	items := []any{"item1", "item2", "item3"}
	result := &Result{
		Items:      items,
		TotalCount: 10,
		HasMore:    true,
		Query:      *query,
	}

	if len(result.Items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result.Items))
	}

	if result.TotalCount != 10 {
		t.Errorf("Expected total count 10, got %d", result.TotalCount)
	}

	if !result.HasMore {
		t.Error("Expected HasMore to be true")
	}

	if !result.Query.HasFilters() {
		t.Error("Expected query to have filters")
	}
}

// MockEntity for testing CreatedAtProvider interface
type MockEntity struct {
	CreatedAt time.Time
}

// Implement the CreatedAtProvider interface
func (m MockEntity) GetCreatedAt() time.Time {
	return m.CreatedAt
}

func TestCreatedAtProvider(t *testing.T) {
	entity := MockEntity{CreatedAt: time.Now()}

	// Verify interface implementation
	var _ CreatedAtProvider = entity

	// Test the method
	createdAt := entity.GetCreatedAt()
	if createdAt.IsZero() {
		t.Error("GetCreatedAt should return non-zero time")
	}
}
