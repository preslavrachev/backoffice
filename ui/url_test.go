package ui

import (
	"net/http"
	"strings"
	"testing"
)

func TestNewAdminURL(t *testing.T) {
	tests := []struct {
		name         string
		resourceName string
		expected     string
	}{
		{"simple resource", "User", "/admin/User"},
		{"resource with spaces", "User Profile", "/admin/User%20Profile"},
		{"resource with special chars", "User&Data", "/admin/User&Data"},
		{"empty resource", "", "/admin/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewAdminURL(tt.resourceName)
			result := builder.String()

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestWithSort(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		direction string
		expected  string
	}{
		{"field and direction", "Name", "asc", "/admin/User?direction=asc&sort=Name"},
		{"field only", "Name", "", "/admin/User?sort=Name"},
		{"empty field", "", "asc", "/admin/User"},
		{"both empty", "", "", "/admin/User"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewAdminURL("User").WithSort(tt.field, tt.direction).String()

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestWithPagination(t *testing.T) {
	tests := []struct {
		name     string
		offset   int
		limit    int
		expected string
	}{
		{"normal pagination", 10, 20, "/admin/User?limit=20&offset=10"},
		{"zero offset", 0, 10, "/admin/User?limit=10&offset=0"},
		{"negative values", -5, -10, "/admin/User?limit=-10&offset=-5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewAdminURL("User").WithPagination(tt.offset, tt.limit).String()

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestChainedMethods(t *testing.T) {
	result := NewAdminURL("User").
		WithSort("Name", "desc").
		WithPagination(20, 10).
		WithFilter("status", "active").
		WithLoadMore().
		String()

	// Verify all parameters are present (order may vary due to url.Values.Encode())
	expectedParams := []string{"sort=Name", "direction=desc", "offset=20", "limit=10", "status=active", "load_more=true"}

	for _, param := range expectedParams {
		if !strings.Contains(result, param) {
			t.Errorf("Expected URL to contain %s, got %s", param, result)
		}
	}

	if !strings.HasPrefix(result, "/admin/User?") {
		t.Errorf("Expected URL to start with /admin/User?, got %s", result)
	}
}

func TestPreserveFromRequest(t *testing.T) {
	tests := []struct {
		name           string
		requestURL     string
		expectedParams []string
		excludedParams []string
	}{
		{
			name:           "preserve user params",
			requestURL:     "/admin/User?sort=Name&direction=asc&status=active",
			expectedParams: []string{"sort=Name", "direction=asc", "status=active"},
			excludedParams: []string{},
		},
		{
			name:           "exclude internal params",
			requestURL:     "/admin/User?sort=Name&load_more=true&success=delete",
			expectedParams: []string{"sort=Name"},
			excludedParams: []string{"load_more", "success"},
		},
		{
			name:           "mixed params",
			requestURL:     "/admin/User?sort=Name&direction=desc&load_more=true&status=active&resource=User",
			expectedParams: []string{"sort=Name", "direction=desc", "status=active"},
			excludedParams: []string{"load_more", "resource"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock request
			req, err := http.NewRequest("GET", tt.requestURL, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			result := NewAdminURL("User").PreserveFromRequest(req).String()

			// Check expected parameters are present
			for _, param := range tt.expectedParams {
				if !strings.Contains(result, param) {
					t.Errorf("Expected URL to contain %s, got %s", param, result)
				}
			}

			// Check excluded parameters are not present
			for _, param := range tt.excludedParams {
				if strings.Contains(result, param) {
					t.Errorf("Expected URL to NOT contain %s, got %s", param, result)
				}
			}
		})
	}
}

func TestWithParam(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{"normal param", "filter", "active", "/admin/User?filter=active"},
		{"empty key", "", "value", "/admin/User"},
		{"empty value", "key", "", "/admin/User?key="},
		{"both empty", "", "", "/admin/User"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewAdminURL("User").WithParam(tt.key, tt.value).String()

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRemoveParam(t *testing.T) {
	builder := NewAdminURL("User").
		WithSort("Name", "asc").
		WithPagination(10, 20).
		WithFilter("status", "active")

	// Remove sorting
	result := builder.RemoveParam("sort").RemoveParam("direction").String()

	if strings.Contains(result, "sort=") || strings.Contains(result, "direction=") {
		t.Errorf("Expected sort params to be removed, got %s", result)
	}

	// Should still contain other params
	if !strings.Contains(result, "offset=10") || !strings.Contains(result, "status=active") {
		t.Errorf("Expected other params to remain, got %s", result)
	}
}

func TestIsInternalParam(t *testing.T) {
	tests := []struct {
		param    string
		expected bool
	}{
		{"load_more", true},
		{"success", true},
		{"resource", true},
		{"highlighted", true},
		{"LOAD_MORE", true}, // Case insensitive
		{"sort", false},
		{"direction", false},
		{"status", false},
		{"offset", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.param, func(t *testing.T) {
			result := isInternalParam(tt.param)
			if result != tt.expected {
				t.Errorf("isInternalParam(%s) = %v, expected %v", tt.param, result, tt.expected)
			}
		})
	}
}

func TestURLEncoding(t *testing.T) {
	result := NewAdminURL("Special Resource").
		WithSort("field with spaces", "asc").
		WithFilter("name", "John & Jane").
		String()

	// Resource name should be URL encoded
	if !strings.Contains(result, "/admin/Special%20Resource") {
		t.Errorf("Expected resource name to be URL encoded, got %s", result)
	}

	// Parameter values should be URL encoded
	if !strings.Contains(result, "name=John+%26+Jane") && !strings.Contains(result, "name=John%20%26%20Jane") {
		t.Errorf("Expected parameter values to be URL encoded, got %s", result)
	}
}

func TestEmptyBuilder(t *testing.T) {
	result := NewAdminURL("User").String()
	expected := "/admin/User"

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// Example of how the URL builder fixes the pagination bug
func TestLoadMoreURLPreservesSort(t *testing.T) {
	// Simulate current request with sorting
	req, _ := http.NewRequest("GET", "/admin/User?sort=AccountAge&direction=asc&status=active", nil)

	// Build load more URL - automatically preserves all user params
	loadMoreURL := NewAdminURL("User").
		PreserveFromRequest(req).
		WithPagination(10, 10).
		WithLoadMore().
		String()

	// Verify sorting is preserved
	if !strings.Contains(loadMoreURL, "sort=AccountAge") {
		t.Error("Load more URL should preserve sort parameter")
	}
	if !strings.Contains(loadMoreURL, "direction=asc") {
		t.Error("Load more URL should preserve direction parameter")
	}
	if !strings.Contains(loadMoreURL, "status=active") {
		t.Error("Load more URL should preserve filter parameters")
	}
	if !strings.Contains(loadMoreURL, "load_more=true") {
		t.Error("Load more URL should include load_more parameter")
	}
	if !strings.Contains(loadMoreURL, "offset=10") {
		t.Error("Load more URL should include pagination")
	}
}
