package core

import (
	"testing"
	"time"

	"backoffice/middleware/auth"
)

// Test struct for sort testing
type SortTestUser struct {
	ID        uint      `json:"id" db:"id"`
	FirstName string    `json:"first_name" db:"first_name"`
	LastName  string    `json:"last_name" db:"last_name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

func TestFieldBuilderSortBy(t *testing.T) {
	builder := NewFieldBuilder()

	// Test single SortBy call
	builder.SortBy("CreatedAt", SortDesc)

	config := builder.Build()

	if len(config.SortFields) != 1 {
		t.Errorf("Expected 1 sort field, got %d", len(config.SortFields))
	}

	if config.SortFields[0].Field != "CreatedAt" {
		t.Errorf("Expected sort field 'CreatedAt', got '%s'", config.SortFields[0].Field)
	}

	if config.SortFields[0].Direction != SortDesc {
		t.Errorf("Expected sort direction 'desc', got '%s'", config.SortFields[0].Direction)
	}

	if !config.IsSortable {
		t.Error("Expected IsSortable to be true")
	}
}

func TestFieldBuilderMultipleSortBy(t *testing.T) {
	builder := NewFieldBuilder()

	// Test multiple SortBy calls (additive behavior)
	builder.SortBy("LastName", SortAsc).SortBy("FirstName", SortAsc).SortBy("CreatedAt", SortDesc)

	config := builder.Build()

	if len(config.SortFields) != 3 {
		t.Errorf("Expected 3 sort fields, got %d", len(config.SortFields))
	}

	// Verify order and values
	expectedFields := []struct {
		field     string
		direction SortDirection
	}{
		{"LastName", SortAsc},
		{"FirstName", SortAsc},
		{"CreatedAt", SortDesc},
	}

	for i, expected := range expectedFields {
		if config.SortFields[i].Field != expected.field {
			t.Errorf("Sort field %d: expected '%s', got '%s'", i, expected.field, config.SortFields[i].Field)
		}
		if config.SortFields[i].Direction != expected.direction {
			t.Errorf("Sort field %d direction: expected '%s', got '%s'", i, expected.direction, config.SortFields[i].Direction)
		}
	}

	if !config.IsSortable {
		t.Error("Expected IsSortable to be true")
	}
}

func TestWithDerivedFieldSortConfiguration(t *testing.T) {
	// Create a mock adapter for testing
	mockAdapter := &MockAdapter{}

	// Create BackOffice instance
	admin := New(mockAdapter, auth.WithNoAuth())

	// Register resource with derived field that has sort configuration
	admin.RegisterResource(&SortTestUser{}).
		WithDerivedField("DisplayName", "Full Name", func(user any) string {
			u := user.(*SortTestUser)
			return u.FirstName + " " + u.LastName
		}, func(f *FieldBuilder) {
			f.SortBy("LastName", SortAsc).SortBy("FirstName", SortAsc)
		})

	// Get the registered resource
	resource, exists := admin.GetResource("SortTestUser")
	if !exists {
		t.Fatal("SortTestUser resource not found")
	}

	// Find the derived field
	var displayNameField *FieldInfo
	for i, field := range resource.Fields {
		if field.Name == "DisplayName" {
			displayNameField = &resource.Fields[i]
			break
		}
	}

	if displayNameField == nil {
		t.Fatal("DisplayName field not found")
	}

	// Verify field configuration
	if !displayNameField.IsComputed {
		t.Error("Expected DisplayName to be computed")
	}

	if !displayNameField.IsSortable {
		t.Error("Expected DisplayName to be sortable")
	}

	if len(displayNameField.SortFields) != 2 {
		t.Errorf("Expected 2 sort fields, got %d", len(displayNameField.SortFields))
	}

	// Verify sort configuration
	expectedSortFields := []struct {
		field     string
		direction SortDirection
	}{
		{"LastName", SortAsc},
		{"FirstName", SortAsc},
	}

	for i, expected := range expectedSortFields {
		if displayNameField.SortFields[i].Field != expected.field {
			t.Errorf("Sort field %d: expected '%s', got '%s'", i, expected.field, displayNameField.SortFields[i].Field)
		}
		if displayNameField.SortFields[i].Direction != expected.direction {
			t.Errorf("Sort field %d direction: expected '%s', got '%s'", i, expected.direction, displayNameField.SortFields[i].Direction)
		}
	}
}

func TestResourceGetFieldSortConfiguration(t *testing.T) {
	// Create a mock adapter for testing
	mockAdapter := &MockAdapter{}

	// Create BackOffice instance
	admin := New(mockAdapter, auth.WithNoAuth())

	// Register resource with different field types
	admin.RegisterResource(&SortTestUser{}).
		WithField("FirstName", func(f *FieldBuilder) {
			f.DisplayName("First Name")
		}).
		WithDerivedField("FullName", "Full Name", func(user any) string {
			u := user.(*SortTestUser)
			return u.FirstName + " " + u.LastName
		}, func(f *FieldBuilder) {
			f.SortBy("LastName", SortAsc).SortBy("FirstName", SortAsc)
		}).
		WithDerivedField("AccountAge", "Account Age", func(user any) string {
			return "Active"
		}) // No sort configuration

	// Get the registered resource
	resource, exists := admin.GetResource("SortTestUser")
	if !exists {
		t.Fatal("SortTestUser resource not found")
	}

	// Test regular field - should return nil (no custom sort config)
	sortConfig := resource.GetFieldSortConfiguration("FirstName")
	if sortConfig != nil {
		t.Error("Expected nil sort configuration for regular field")
	}

	// Test derived field with sort configuration
	sortConfig = resource.GetFieldSortConfiguration("FullName")
	if sortConfig == nil {
		t.Fatal("Expected sort configuration for FullName field")
	}
	if len(sortConfig) != 2 {
		t.Errorf("Expected 2 sort fields for FullName, got %d", len(sortConfig))
	}

	// Test derived field without sort configuration
	sortConfig = resource.GetFieldSortConfiguration("AccountAge")
	if sortConfig != nil {
		t.Error("Expected nil sort configuration for AccountAge field (no sort config)")
	}
}

func TestResourceIsFieldSortable(t *testing.T) {
	// Create a mock adapter for testing
	mockAdapter := &MockAdapter{}

	// Create BackOffice instance
	admin := New(mockAdapter, auth.WithNoAuth())

	// Register resource with different field types
	admin.RegisterResource(&SortTestUser{}).
		WithField("FirstName", func(f *FieldBuilder) {
			f.DisplayName("First Name")
		}).
		WithDerivedField("FullName", "Full Name", func(user any) string {
			u := user.(*SortTestUser)
			return u.FirstName + " " + u.LastName
		}, func(f *FieldBuilder) {
			f.SortBy("LastName", SortAsc)
		}).
		WithDerivedField("AccountAge", "Account Age", func(user any) string {
			return "Active"
		}) // No sort configuration - should not be sortable

	// Get the registered resource
	resource, exists := admin.GetResource("SortTestUser")
	if !exists {
		t.Fatal("SortTestUser resource not found")
	}

	// Test regular field - should be sortable
	if !resource.IsFieldSortable("FirstName") {
		t.Error("Expected FirstName to be sortable")
	}

	// Test derived field with sort configuration - should be sortable
	if !resource.IsFieldSortable("FullName") {
		t.Error("Expected FullName to be sortable")
	}

	// Test derived field without sort configuration - should not be sortable
	if resource.IsFieldSortable("AccountAge") {
		t.Error("Expected AccountAge to not be sortable (no sort config)")
	}

	// Test non-existent field - should be sortable (default behavior)
	if !resource.IsFieldSortable("NonExistentField") {
		t.Error("Expected non-existent field to be sortable (default)")
	}
}

func TestDerivedFieldRegistration(t *testing.T) {
	// Create a mock adapter for testing
	mockAdapter := &MockAdapter{}

	// Create BackOffice instance
	admin := New(mockAdapter, auth.WithNoAuth())

	// Register resource using WithDerivedField method
	admin.RegisterResource(&SortTestUser{}).
		WithDerivedField("Status", "Status", func(user any) string {
			return "Active"
		})

	// Get the registered resource
	resource, exists := admin.GetResource("SortTestUser")
	if !exists {
		t.Fatal("SortTestUser resource not found")
	}

	// Find the computed field
	var statusField *FieldInfo
	for i, field := range resource.Fields {
		if field.Name == "Status" {
			statusField = &resource.Fields[i]
			break
		}
	}

	if statusField == nil {
		t.Fatal("Status field not found")
	}

	// Verify field configuration (should work same as before)
	if !statusField.IsComputed {
		t.Error("Expected Status to be computed")
	}

	if !statusField.ReadOnly {
		t.Error("Expected Status to be read-only")
	}

	// Should not be sortable by default (no sort configuration)
	if resource.IsFieldSortable("Status") {
		t.Error("Expected Status to not be sortable by default")
	}
}
