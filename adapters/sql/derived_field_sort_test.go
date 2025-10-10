package sql

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"backoffice/core"
	"backoffice/middleware/auth"

	_ "github.com/mattn/go-sqlite3"
)

// Test struct for integration testing
type IntegrationTestUser struct {
	ID        uint      `json:"id" db:"id"`
	FirstName string    `json:"first_name" db:"first_name"`
	LastName  string    `json:"last_name" db:"last_name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func setupDerivedFieldTestDB(t *testing.T) (*sql.DB, *Adapter) {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE integration_test_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO integration_test_users (first_name, last_name, created_at) VALUES
		('John', 'Smith', '2024-01-01 10:00:00'),
		('Jane', 'Doe', '2024-01-02 11:00:00'),
		('Alice', 'Johnson', '2024-01-03 12:00:00'),
		('Bob', 'Williams', '2024-01-04 13:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	adapter := New(db)
	return db, adapter
}

func TestDerivedFieldSortingIntegration(t *testing.T) {
	db, adapter := setupDerivedFieldTestDB(t)
	defer db.Close()

	// Create BackOffice instance
	admin := core.New(adapter, auth.WithNoAuth())

	// Register resource with derived field that has sort configuration
	admin.RegisterResource(&IntegrationTestUser{}).
		WithDerivedField("FullName", "Full Name", func(user any) string {
			u := user.(*IntegrationTestUser)
			return u.FirstName + " " + u.LastName
		}, func(f *core.FieldBuilder) {
			f.SortBy("LastName", core.SortAsc).SortBy("FirstName", core.SortAsc)
		})

	resource, exists := admin.GetResource("IntegrationTestUser")
	if !exists {
		t.Fatal("IntegrationTestUser resource not found")
	}

	// Create a query that sorts by the derived field
	query := core.NewQuery()
	query.WithSort("FullName", core.SortAsc)

	// Execute the query
	result, err := adapter.Find(context.Background(), resource, query)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	// Verify that we got results
	if len(result.Items) != 4 {
		t.Errorf("Expected 4 results, got %d", len(result.Items))
	}

	// Verify the sorting order - should be sorted by LastName, then FirstName
	expectedOrder := []struct {
		firstName string
		lastName  string
	}{
		{"Jane", "Doe"},
		{"Alice", "Johnson"},
		{"John", "Smith"},
		{"Bob", "Williams"},
	}

	for i, expected := range expectedOrder {
		user := result.Items[i].(*IntegrationTestUser)
		if user.FirstName != expected.firstName || user.LastName != expected.lastName {
			t.Errorf("Result %d: expected %s %s, got %s %s",
				i, expected.firstName, expected.lastName, user.FirstName, user.LastName)
		}
	}
}

func TestRegularFieldSortingStillWorks(t *testing.T) {
	db, adapter := setupDerivedFieldTestDB(t)
	defer db.Close()

	// Create BackOffice instance
	admin := core.New(adapter, auth.WithNoAuth())

	// Register resource with regular field sorting
	admin.RegisterResource(&IntegrationTestUser{})

	resource, exists := admin.GetResource("IntegrationTestUser")
	if !exists {
		t.Fatal("IntegrationTestUser resource not found")
	}

	// Create a query that sorts by a regular field
	query := core.NewQuery()
	query.WithSort("FirstName", core.SortDesc)

	// Execute the query
	result, err := adapter.Find(context.Background(), resource, query)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	// Verify that we got results
	if len(result.Items) != 4 {
		t.Errorf("Expected 4 results, got %d", len(result.Items))
	}

	// Verify the sorting order - should be sorted by FirstName DESC
	expectedOrder := []string{"John", "Jane", "Bob", "Alice"}

	for i, expectedFirstName := range expectedOrder {
		user := result.Items[i].(*IntegrationTestUser)
		if user.FirstName != expectedFirstName {
			t.Errorf("Result %d: expected FirstName %s, got %s",
				i, expectedFirstName, user.FirstName)
		}
	}
}

func TestDerivedFieldWithoutSortConfigIsSkipped(t *testing.T) {
	db, adapter := setupDerivedFieldTestDB(t)
	defer db.Close()

	// Create BackOffice instance
	admin := core.New(adapter, auth.WithNoAuth())

	// Register resource with derived field without sort configuration
	admin.RegisterResource(&IntegrationTestUser{}).
		WithDerivedField("AccountAge", "Account Age", func(user any) string {
			return "Active"
		}) // No sort configuration

	resource, exists := admin.GetResource("IntegrationTestUser")
	if !exists {
		t.Fatal("IntegrationTestUser resource not found")
	}

	// Create a query that tries to sort by the unsortable derived field
	query := core.NewQuery()
	query.WithSort("AccountAge", core.SortAsc)

	// Execute the query - should not fail but should ignore the sort
	result, err := adapter.Find(context.Background(), resource, query)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	// Verify that we got results (sorting was ignored, default order applied)
	if len(result.Items) != 4 {
		t.Errorf("Expected 4 results, got %d", len(result.Items))
	}

	// Should be in creation order since no valid sorting was applied
	firstUser := result.Items[0].(*IntegrationTestUser)
	if firstUser.FirstName != "John" {
		t.Errorf("Expected first user to be John (default order), got %s", firstUser.FirstName)
	}
}

func TestFieldLevelSortOverridesRegularSorting(t *testing.T) {
	db, adapter := setupDerivedFieldTestDB(t)
	defer db.Close()

	// Create BackOffice instance
	admin := core.New(adapter, auth.WithNoAuth())

	// Register resource with regular field that has custom sort configuration
	admin.RegisterResource(&IntegrationTestUser{}).
		WithField("FirstName", func(f *core.FieldBuilder) {
			f.DisplayName("First Name").SortBy("LastName", core.SortAsc).SortBy("FirstName", core.SortAsc)
		})

	resource, exists := admin.GetResource("IntegrationTestUser")
	if !exists {
		t.Fatal("IntegrationTestUser resource not found")
	}

	// Create a query that sorts by FirstName (which has custom sort configuration)
	query := core.NewQuery()
	query.WithSort("FirstName", core.SortAsc)

	// Execute the query
	result, err := adapter.Find(context.Background(), resource, query)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	// Verify that we got results
	if len(result.Items) != 4 {
		t.Errorf("Expected 4 results, got %d", len(result.Items))
	}

	// Verify the sorting order - should be sorted by LastName, then FirstName (not just FirstName)
	expectedOrder := []struct {
		firstName string
		lastName  string
	}{
		{"Jane", "Doe"},
		{"Alice", "Johnson"},
		{"John", "Smith"},
		{"Bob", "Williams"},
	}

	for i, expected := range expectedOrder {
		user := result.Items[i].(*IntegrationTestUser)
		if user.FirstName != expected.firstName || user.LastName != expected.lastName {
			t.Errorf("Result %d: expected %s %s, got %s %s",
				i, expected.firstName, expected.lastName, user.FirstName, user.LastName)
		}
	}
}

func TestMultiFieldSortConfiguration(t *testing.T) {
	db, adapter := setupDerivedFieldTestDB(t)
	defer db.Close()

	// Insert additional test data with same last names to test multi-field sorting
	_, err := db.Exec(`
		INSERT INTO integration_test_users (first_name, last_name, created_at) VALUES
		('Charlie', 'Smith', '2024-01-05 14:00:00'),
		('David', 'Doe', '2024-01-06 15:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert additional test data: %v", err)
	}

	// Create BackOffice instance
	admin := core.New(adapter, auth.WithNoAuth())

	// Register resource with derived field that has multi-field sort configuration
	admin.RegisterResource(&IntegrationTestUser{}).
		WithDerivedField("DisplayName", "Display Name", func(user any) string {
			u := user.(*IntegrationTestUser)
			return u.LastName + ", " + u.FirstName
		}, func(f *core.FieldBuilder) {
			f.SortBy("LastName", core.SortAsc).SortBy("FirstName", core.SortAsc).SortBy("CreatedAt", core.SortDesc)
		})

	resource, exists := admin.GetResource("IntegrationTestUser")
	if !exists {
		t.Fatal("IntegrationTestUser resource not found")
	}

	// Create a query that sorts by the derived field
	query := core.NewQuery()
	query.WithSort("DisplayName", core.SortAsc)

	// Execute the query
	result, err := adapter.Find(context.Background(), resource, query)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	// Verify that we got results
	if len(result.Items) != 6 {
		t.Errorf("Expected 6 results, got %d", len(result.Items))
	}

	// Verify the sorting order - should be sorted by LastName, FirstName, CreatedAt DESC
	// Expected order: Doe(David first, then Jane), Johnson(Alice), Smith(Charlie first, then John)
	expectedOrder := []struct {
		firstName string
		lastName  string
	}{
		{"David", "Doe"},     // FirstName: David comes before Jane alphabetically
		{"Jane", "Doe"},      // FirstName: Jane comes after David alphabetically
		{"Alice", "Johnson"}, // Only Johnson entry
		{"Charlie", "Smith"}, // FirstName: Charlie comes before John alphabetically
		{"John", "Smith"},    // FirstName: John comes after Charlie alphabetically
	}

	for i, expected := range expectedOrder {
		if i >= len(result.Items) {
			t.Errorf("Missing result at index %d", i)
			continue
		}
		user := result.Items[i].(*IntegrationTestUser)
		if user.FirstName != expected.firstName || user.LastName != expected.lastName {
			t.Errorf("Result %d: expected %s %s, got %s %s",
				i, expected.firstName, expected.lastName, user.FirstName, user.LastName)
		}
	}
}
