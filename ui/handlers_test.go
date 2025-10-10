package ui

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	sqladapter "github.com/preslavrachev/backoffice/adapters/sql"
	"github.com/preslavrachev/backoffice/core"
	"github.com/preslavrachev/backoffice/middleware/auth"

	_ "github.com/mattn/go-sqlite3"
)

type TestUser struct {
	ID        uint      `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func setupHandlerTestDB(t *testing.T) (*sql.DB, *core.BackOffice) {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE test_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Insert 15 test users (more than page size of 10) with specific creation order
	users := []struct {
		name      string
		createdAt string
	}{
		{"Alice", "2024-01-01 10:00:00"}, // Oldest
		{"Bob", "2024-01-02 11:00:00"},
		{"Charlie", "2024-01-03 12:00:00"},
		{"David", "2024-01-04 13:00:00"},
		{"Eve", "2024-01-05 14:00:00"},
		{"Frank", "2024-01-06 15:00:00"},
		{"Grace", "2024-01-07 16:00:00"},
		{"Henry", "2024-01-08 17:00:00"},
		{"Ivy", "2024-01-09 18:00:00"},
		{"Jack", "2024-01-10 19:00:00"}, // 10th user (end of first page)
		{"Kate", "2024-01-11 20:00:00"}, // 11th user (start of second page)
		{"Liam", "2024-01-12 21:00:00"},
		{"Maya", "2024-01-13 22:00:00"},
		{"Noah", "2024-01-14 23:00:00"},
		{"Olivia", "2024-01-15 24:00:00"}, // Newest
	}

	for _, user := range users {
		_, err := db.Exec(`INSERT INTO test_users (name, created_at) VALUES (?, ?)`, user.name, user.createdAt)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// Create BackOffice with SQL adapter (default BasePath is "/admin")
	adapter := sqladapter.New(db)
	admin := core.New(adapter, auth.WithNoAuth())

	// Register resource with derived field that has sort configuration
	admin.RegisterResource(&TestUser{}).
		WithField("Name", func(f *core.FieldBuilder) {
			f.DisplayName("Name")
		}).
		WithField("CreatedAt", func(f *core.FieldBuilder) {
			f.DisplayName("Created At")
		}).
		WithDerivedField("AccountAge", "Account Age", func(user any) string {
			u := user.(*TestUser)
			days := int(time.Since(u.CreatedAt).Hours() / 24)
			return fmt.Sprintf("%d days", days)
		}, func(f *core.FieldBuilder) {
			f.SortBy("CreatedAt", core.SortDesc) // Sort by CreatedAt when AccountAge is requested
		})

	return db, admin
}

// extractLoadMoreURL extracts the "Load More" button URL from HTML response
func extractLoadMoreURL(html string) string {
	// Look for hx-get attribute specifically in Load More button
	re := regexp.MustCompile(`hx-get="([^"]*load_more=true[^"]*)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func TestPaginationPreservesSorting(t *testing.T) {
	tests := []struct {
		name          string
		sortField     string
		sortDirection string
		expectedFirst string // Expected first user on page 2
	}{
		{
			name:          "derived field ASC",
			sortField:     "AccountAge",
			sortDirection: "asc",
			expectedFirst: "Kate", // Should be 11th oldest user
		},
		{
			name:          "derived field DESC",
			sortField:     "AccountAge",
			sortDirection: "desc",
			expectedFirst: "Eve", // Should be 5th newest user (after first 10)
		},
		{
			name:          "regular field ASC",
			sortField:     "Name",
			sortDirection: "asc",
			expectedFirst: "Kate", // Alphabetically after first 10
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, admin := setupHandlerTestDB(t)
			defer db.Close()

			// Use the full Handler function which sets up routing
			handler := Handler(admin, "/admin")

			// Step 1: Request first page with sorting
			url := fmt.Sprintf("/admin/TestUser?sort=%s&direction=%s", tt.sortField, tt.sortDirection)
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected 200, got %d. Response: %s", w.Code, w.Body.String())
			}

			firstPageHTML := w.Body.String()

			// Step 2: Extract "Load More" button URL from HTML response
			loadMoreURL := extractLoadMoreURL(firstPageHTML)
			if loadMoreURL == "" {
				t.Fatal("Could not find Load More URL in response")
			}

			// Step 3: Verify URL contains sort parameters
			if !strings.Contains(loadMoreURL, "sort="+tt.sortField) {
				t.Errorf("Load More URL should contain sort=%s, got: %s", tt.sortField, loadMoreURL)
			}
			if !strings.Contains(loadMoreURL, "direction="+tt.sortDirection) {
				t.Errorf("Load More URL should contain direction=%s, got: %s", tt.sortDirection, loadMoreURL)
			}

			// Step 4: Follow the "Load More" URL to get second page
			req2 := httptest.NewRequest("GET", loadMoreURL, nil)
			w2 := httptest.NewRecorder()

			handler.ServeHTTP(w2, req2)

			if w2.Code != http.StatusOK {
				t.Fatalf("Expected 200 for load more request, got %d", w2.Code)
			}

			secondPageHTML := w2.Body.String()

			// Step 5: Verify second page contains expected first user (maintains sort order)
			if !strings.Contains(secondPageHTML, tt.expectedFirst) {
				t.Errorf("Expected second page to start with %s, but user not found in: %s", tt.expectedFirst, secondPageHTML)
			}
		})
	}
}
