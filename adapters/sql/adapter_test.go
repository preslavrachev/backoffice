package sql

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/preslavrachev/backoffice/core"

	_ "github.com/mattn/go-sqlite3"
)

// Test entities
type TestUser struct {
	ID        uint         `json:"id" db:"id"`
	Name      string       `json:"name" db:"name"`
	Email     string       `json:"email" db:"email"`
	Age       int          `json:"age" db:"age"`
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt time.Time    `json:"updated_at" db:"updated_at"`
	DeletedAt sql.NullTime `json:"deleted_at" db:"deleted_at"`
}

type TestCategory struct {
	ID       uint   `json:"id" db:"id"`
	Name     string `json:"name" db:"name"`
	Priority int    `json:"priority" db:"priority"`
}

func setupTestDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}

	// Create test tables
	schema := `
	CREATE TABLE test_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL,
		age INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		deleted_at DATETIME
	);

	CREATE TABLE test_categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		priority INTEGER NOT NULL
	);
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	return db, nil
}

func seedTestData(db *sql.DB) error {
	users := []TestUser{
		{Name: "Alice", Email: "alice@example.com", Age: 25, CreatedAt: time.Now().Add(-10 * 24 * time.Hour)},
		{Name: "Bob", Email: "bob@example.com", Age: 30, CreatedAt: time.Now().Add(-5 * 24 * time.Hour)},
		{Name: "Charlie", Email: "charlie@example.com", Age: 35, CreatedAt: time.Now().Add(-2 * 24 * time.Hour)},
		{Name: "David", Email: "david@example.com", Age: 28, CreatedAt: time.Now().Add(-1 * 24 * time.Hour)},
		{Name: "Eve", Email: "eve@example.com", Age: 32, CreatedAt: time.Now().Add(-3 * 24 * time.Hour)},
		{Name: "Frank", Email: "frank@example.com", Age: 27, CreatedAt: time.Now().Add(-7 * 24 * time.Hour)},
		{Name: "Grace", Email: "grace@example.com", Age: 29, CreatedAt: time.Now().Add(-4 * 24 * time.Hour)},
		{Name: "Henry", Email: "henry@example.com", Age: 31, CreatedAt: time.Now().Add(-6 * 24 * time.Hour)},
		{Name: "Ivy", Email: "ivy@example.com", Age: 26, CreatedAt: time.Now().Add(-8 * 24 * time.Hour)},
		{Name: "Jack", Email: "jack@example.com", Age: 33, CreatedAt: time.Now().Add(-9 * 24 * time.Hour)},
		{Name: "Kate", Email: "kate@example.com", Age: 24, CreatedAt: time.Now()},
		{Name: "Liam", Email: "liam@example.com", Age: 36, CreatedAt: time.Now().Add(-11 * 24 * time.Hour)},
	}

	categories := []TestCategory{
		{Name: "Electronics", Priority: 1},
		{Name: "Books", Priority: 2},
		{Name: "Clothing", Priority: 3},
		{Name: "Sports", Priority: 4},
		{Name: "Home", Priority: 5},
	}

	for _, user := range users {
		_, err := db.Exec(`
			INSERT INTO test_users (name, email, age, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?)
		`, user.Name, user.Email, user.Age, user.CreatedAt, user.CreatedAt)
		if err != nil {
			return err
		}
	}

	for _, cat := range categories {
		_, err := db.Exec(`
			INSERT INTO test_categories (name, priority) 
			VALUES (?, ?)
		`, cat.Name, cat.Priority)
		if err != nil {
			return err
		}
	}

	return nil
}

func createTestResource() *core.Resource {
	return &core.Resource{
		Name:        "TestUser",
		DisplayName: "Test User",
		PluralName:  "Test Users",
		Model:       &TestUser{},
		ModelType:   reflect.TypeOf(&TestUser{}),
		Fields: []core.FieldInfo{
			{Name: "ID", Type: "uint", PrimaryKey: true},
			{Name: "Name", Type: "string"},
			{Name: "Email", Type: "string"},
			{Name: "Age", Type: "int"},
			{Name: "created_at", Type: "time.Time"}, // Use snake_case for SQL compatibility
		},
		IDField:     "ID",
		PrimaryKey:  "id",
		TableName:   "test_users",
		DefaultSort: core.SortField{}, // Will test both with and without default sort (zero value means auto-detection)
	}
}

func TestFind_BasicQuery(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	adapter := New(db)
	resource := createTestResource()

	// Test basic query without any filters or sorting
	query := core.NewQuery()
	result, err := adapter.Find(context.Background(), resource, query)

	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}

	if len(result.Items) != core.DefaultPageSize {
		t.Errorf("Expected %d items (page size), got %d", core.DefaultPageSize, len(result.Items))
	}

	if result.TotalCount != 12 {
		t.Errorf("Expected total count of 12, got %d", result.TotalCount)
	}

	if !result.HasMore {
		t.Error("Expected HasMore to be true since we have more than 10 records")
	}
}

func TestFind_WithFilters(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	adapter := New(db)
	resource := createTestResource()

	// Test query with filters
	query := core.NewQuery()
	query.WithFilters(map[string]any{"name": "Alice"})

	result, err := adapter.Find(context.Background(), resource, query)

	if err != nil {
		t.Fatalf("Find with filters failed: %v", err)
	}

	if len(result.Items) != 1 {
		t.Errorf("Expected 1 item with name filter, got %d", len(result.Items))
	}

	if result.TotalCount != 1 {
		t.Errorf("Expected total count of 1, got %d", result.TotalCount)
	}

	if result.HasMore {
		t.Error("Expected HasMore to be false since we only have 1 matching record")
	}

	// Verify the returned item
	user := result.Items[0].(*TestUser)
	if user.Name != "Alice" {
		t.Errorf("Expected user name to be Alice, got %s", user.Name)
	}
}

func TestFind_WithSorting(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	adapter := New(db)
	resource := createTestResource()

	// Test ascending sort by age
	query := core.NewQuery()
	query.WithSort("age", core.SortAsc)
	query.WithPagination(5, 0) // Limit to 5 for easier testing

	result, err := adapter.Find(context.Background(), resource, query)

	if err != nil {
		t.Fatalf("Find with sorting failed: %v", err)
	}

	if len(result.Items) != 5 {
		t.Errorf("Expected 5 items, got %d", len(result.Items))
	}

	// Verify ascending age order
	prevAge := 0
	for i, item := range result.Items {
		user := item.(*TestUser)
		if i > 0 && user.Age < prevAge {
			t.Errorf("Items not sorted by age ascending: user %d has age %d, previous was %d", i, user.Age, prevAge)
		}
		prevAge = user.Age
	}

	// Test descending sort by age
	query2 := core.NewQuery()
	query2.WithSort("age", core.SortDesc)
	query2.WithPagination(5, 0)

	result2, err := adapter.Find(context.Background(), resource, query2)

	if err != nil {
		t.Fatalf("Find with desc sorting failed: %v", err)
	}

	// Verify descending age order
	prevAge = 999
	for i, item := range result2.Items {
		user := item.(*TestUser)
		if i > 0 && user.Age > prevAge {
			t.Errorf("Items not sorted by age descending: user %d has age %d, previous was %d", i, user.Age, prevAge)
		}
		prevAge = user.Age
	}
}

func TestFind_WithPagination(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	adapter := New(db)
	resource := createTestResource()

	// Test first page
	query1 := core.NewQuery()
	query1.WithPagination(5, 0)
	query1.WithSort("id", core.SortAsc) // Sort for predictable results

	result1, err := adapter.Find(context.Background(), resource, query1)

	if err != nil {
		t.Fatalf("Find first page failed: %v", err)
	}

	if len(result1.Items) != 5 {
		t.Errorf("Expected 5 items on first page, got %d", len(result1.Items))
	}

	if result1.TotalCount != 12 {
		t.Errorf("Expected total count of 12, got %d", result1.TotalCount)
	}

	if !result1.HasMore {
		t.Error("Expected HasMore to be true for first page")
	}

	// Test second page
	query2 := core.NewQuery()
	query2.WithPagination(5, 5)
	query2.WithSort("id", core.SortAsc)

	result2, err := adapter.Find(context.Background(), resource, query2)

	if err != nil {
		t.Fatalf("Find second page failed: %v", err)
	}

	if len(result2.Items) != 5 {
		t.Errorf("Expected 5 items on second page, got %d", len(result2.Items))
	}

	if !result2.HasMore {
		t.Error("Expected HasMore to be true for second page")
	}

	// Test third page (partial)
	query3 := core.NewQuery()
	query3.WithPagination(5, 10)
	query3.WithSort("id", core.SortAsc)

	result3, err := adapter.Find(context.Background(), resource, query3)

	if err != nil {
		t.Fatalf("Find third page failed: %v", err)
	}

	if len(result3.Items) != 2 {
		t.Errorf("Expected 2 items on third page, got %d", len(result3.Items))
	}

	if result3.HasMore {
		t.Error("Expected HasMore to be false for last page")
	}

	// Verify different items on each page
	firstPageFirstID := result1.Items[0].(*TestUser).ID
	secondPageFirstID := result2.Items[0].(*TestUser).ID
	thirdPageFirstID := result3.Items[0].(*TestUser).ID

	if firstPageFirstID == secondPageFirstID || secondPageFirstID == thirdPageFirstID {
		t.Error("Pages should contain different items")
	}
}

func TestFind_DefaultSorting(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	adapter := New(db)

	// Test resource with CreatedAt field (should default to CreatedAt DESC)
	resource := createTestResource()

	query := core.NewQuery()
	query.WithPagination(5, 0)

	result, err := adapter.Find(context.Background(), resource, query)

	if err != nil {
		t.Fatalf("Find with default sorting failed: %v", err)
	}

	// Should be sorted by created_at DESC by default
	if len(result.Query.Sort) != 1 {
		t.Errorf("Expected 1 sort field after applying default, got %d", len(result.Query.Sort))
	}

	if result.Query.Sort[0].Field != "created_at" {
		t.Errorf("Expected default sort by created_at, got %s", result.Query.Sort[0].Field)
	}

	if result.Query.Sort[0].Direction != core.SortDesc {
		t.Errorf("Expected default created_at sort to be DESC, got %s", result.Query.Sort[0].Direction)
	}

	// Verify items are in CreatedAt DESC order
	var prevCreatedAt time.Time = time.Now().Add(24 * time.Hour) // Future time for comparison
	for i, item := range result.Items {
		user := item.(*TestUser)
		if i > 0 && user.CreatedAt.After(prevCreatedAt) {
			t.Errorf("Items not sorted by CreatedAt descending at position %d", i)
		}
		prevCreatedAt = user.CreatedAt
	}
}

func TestFind_ConfiguredDefaultSort(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	adapter := New(db)

	// Test resource with configured default sort
	resource := createTestResource()
	resource.DefaultSort = core.SortField{
		Field:      "Age",
		Direction:  core.SortDesc,
		Precedence: core.SortPrecedenceExplicit,
	}

	query := core.NewQuery()
	query.WithPagination(5, 0)

	result, err := adapter.Find(context.Background(), resource, query)

	if err != nil {
		t.Fatalf("Find with configured default sorting failed: %v", err)
	}

	// Should be sorted by Age DESC as configured
	if result.Query.Sort[0].Field != "Age" {
		t.Errorf("Expected configured default sort by Age, got %s", result.Query.Sort[0].Field)
	}

	// Verify items are in Age DESC order
	prevAge := 999
	for i, item := range result.Items {
		user := item.(*TestUser)
		if i > 0 && user.Age > prevAge {
			t.Errorf("Items not sorted by Age descending at position %d: %d > %d", i, user.Age, prevAge)
		}
		prevAge = user.Age
	}
}

func TestCRUD_Operations(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	adapter := New(db)
	resource := createTestResource()

	// Test Create
	newUser := &TestUser{
		Name:      "Test User",
		Email:     "test@example.com",
		Age:       25,
		CreatedAt: time.Now(),
	}

	err = adapter.Create(context.Background(), resource, newUser)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Test GetByID
	retrievedUser, err := adapter.GetByID(context.Background(), resource, 1)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	user := retrievedUser.(*TestUser)
	if user.Name != "Test User" {
		t.Errorf("Expected name 'Test User', got %s", user.Name)
	}

	// Test Update
	updateUser := &TestUser{
		Name:  "Updated User",
		Email: "updated@example.com",
		Age:   30,
	}

	err = adapter.Update(context.Background(), resource, 1, updateUser)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	updatedUser, err := adapter.GetByID(context.Background(), resource, 1)
	if err != nil {
		t.Fatalf("GetByID after update failed: %v", err)
	}

	updated := updatedUser.(*TestUser)
	if updated.Name != "Updated User" {
		t.Errorf("Expected updated name 'Updated User', got %s", updated.Name)
	}

	// Test Delete
	err = adapter.Delete(context.Background(), resource, 1)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err = adapter.GetByID(context.Background(), resource, 1)
	if err == nil {
		t.Error("Expected error when getting deleted record, but got none")
	}
}

func TestFind_ErrorHandling(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	adapter := New(db)
	resource := createTestResource()

	// Test with nil query (should error)
	_, err = adapter.Find(context.Background(), resource, nil)
	if err == nil {
		t.Error("Expected error when passing nil query")
	}
}

func TestFind_CombinedFiltersAndSorting(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	adapter := New(db)
	resource := createTestResource()

	// Test combining filters and sorting
	query := core.NewQuery()
	query.WithFilters(map[string]any{"age": 30}) // This should match Bob and others with age 30
	query.WithSort("name", core.SortAsc)
	query.WithPagination(10, 0)

	// Add a few more users with age 30 for better testing
	moreUsers := []TestUser{
		{Name: "Zoe", Email: "zoe@example.com", Age: 30, CreatedAt: time.Now()},
		{Name: "Adam", Email: "adam@example.com", Age: 30, CreatedAt: time.Now()},
	}

	for _, user := range moreUsers {
		_, err := db.Exec(`
			INSERT INTO test_users (name, email, age, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?)
		`, user.Name, user.Email, user.Age, user.CreatedAt, user.CreatedAt)
		if err != nil {
			t.Fatalf("Failed to add more test users: %v", err)
		}
	}

	result, err := adapter.Find(context.Background(), resource, query)

	if err != nil {
		t.Fatalf("Find with combined filters and sorting failed: %v", err)
	}

	// Should have users with age 30, sorted by name ascending
	for _, item := range result.Items {
		user := item.(*TestUser)
		if user.Age != 30 {
			t.Errorf("Expected all users to have age 30, got %d for user %s", user.Age, user.Name)
		}
	}

	// Verify name ascending order
	var prevName string
	for i, item := range result.Items {
		user := item.(*TestUser)
		if i > 0 && user.Name < prevName {
			t.Errorf("Items not sorted by name ascending: %s < %s", user.Name, prevName)
		}
		prevName = user.Name
	}
}

func TestFind_EmptyResults(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	// Don't seed data - test with empty database
	adapter := New(db)
	resource := createTestResource()

	query := core.NewQuery()
	result, err := adapter.Find(context.Background(), resource, query)

	if err != nil {
		t.Fatalf("Find on empty database failed: %v", err)
	}

	if len(result.Items) != 0 {
		t.Errorf("Expected 0 items from empty database, got %d", len(result.Items))
	}

	if result.TotalCount != 0 {
		t.Errorf("Expected total count of 0, got %d", result.TotalCount)
	}

	if result.HasMore {
		t.Error("Expected HasMore to be false for empty results")
	}
}

func TestDerivedFieldRespectsUserDirection(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	adapter := New(db)
	resource := createTestResource()

	// Add a derived field that can sort by created_at
	resource.FieldConfigs = map[string]*core.FieldConfig{
		"AccountAge": {
			IsComputed:  true,
			ComputeFunc: func(u any) string { return "old" },
			SortFields:  []core.SortField{{Field: "created_at", Direction: core.SortDesc}},
			IsSortable:  true,
		},
	}
	resource.Fields = append(resource.Fields, core.FieldInfo{
		Name: "AccountAge", Type: "string", IsComputed: true, IsSortable: true,
		SortFields: []core.SortField{{Field: "created_at", Direction: core.SortDesc}},
	})

	// User requests ASC - should get oldest first
	queryAsc := core.NewQuery()
	queryAsc.WithSort("AccountAge", core.SortAsc)
	queryAsc.WithPagination(3, 0)
	resultAsc, _ := adapter.Find(context.Background(), resource, queryAsc)

	// User requests DESC - should get newest first
	queryDesc := core.NewQuery()
	queryDesc.WithSort("AccountAge", core.SortDesc)
	queryDesc.WithPagination(3, 0)
	resultDesc, _ := adapter.Find(context.Background(), resource, queryDesc)

	// Should respect user's direction choice
	firstAsc := resultAsc.Items[0].(*TestUser).Name
	firstDesc := resultDesc.Items[0].(*TestUser).Name

	if firstAsc == firstDesc {
		t.Errorf("Expected different first results for ASC vs DESC, both returned: %s", firstAsc)
	}
}
