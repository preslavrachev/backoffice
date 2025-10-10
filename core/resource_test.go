package core

import (
	"context"
	"testing"
	"time"

	"backoffice/middleware/auth"
)

// Test struct for field order testing
type TestUser struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	CreatedAt   time.Time `json:"created_at"`
	IsActive    bool      `json:"is_active"`
	Description string    `json:"description"`
}

func TestFieldRegistrationOrder(t *testing.T) {
	// Create a mock adapter for testing
	mockAdapter := &MockAdapter{}

	// Create BackOffice instance
	admin := New(mockAdapter, auth.WithNoAuth())

	// Register resource with fields in specific order
	admin.RegisterResource(&TestUser{}).
		WithField("Name", func(f *FieldBuilder) {
			f.DisplayName("Full Name").Required(true)
		}).
		WithField("Email", func(f *FieldBuilder) {
			f.DisplayName("Email Address").Required(true)
		}).
		WithDerivedField("Status", "Status", func(user any) string {
			return "Active"
		}).
		WithField("Description", func(f *FieldBuilder) {
			f.DisplayName("User Description")
		}).
		WithDerivedField("Summary", "Summary", func(user any) string {
			return "User summary"
		})

	// Get the registered resource
	resource, exists := admin.GetResource("TestUser")
	if !exists {
		t.Fatal("TestUser resource not found")
	}

	// Verify field order matches registration order
	expectedOrder := []string{
		"ID",          // Primary key (always first)
		"Name",        // First WithField
		"Email",       // Second WithField
		"Status",      // First WithComputedField
		"Description", // Third WithField
		"Summary",     // Second WithComputedField
	}

	if len(resource.Fields) != len(expectedOrder) {
		t.Logf("Actual fields:")
		for i, field := range resource.Fields {
			t.Logf("  [%d] %s (computed: %v, relationship: %v)", i, field.Name, field.IsComputed, field.Relationship != nil)
		}
		t.Fatalf("Expected %d fields, got %d", len(expectedOrder), len(resource.Fields))
	}

	for i, expectedField := range expectedOrder {
		if resource.Fields[i].Name != expectedField {
			t.Errorf("Field at position %d: expected %s, got %s", i, expectedField, resource.Fields[i].Name)
		}
	}

	// Verify display names are correctly applied
	nameField := findFieldByName(resource.Fields, "Name")
	if nameField == nil {
		t.Fatal("Name field not found")
	}
	if nameField.DisplayName != "Full Name" {
		t.Errorf("Expected display name 'Full Name', got '%s'", nameField.DisplayName)
	}

	emailField := findFieldByName(resource.Fields, "Email")
	if emailField == nil {
		t.Fatal("Email field not found")
	}
	if emailField.DisplayName != "Email Address" {
		t.Errorf("Expected display name 'Email Address', got '%s'", emailField.DisplayName)
	}

	// Verify computed fields are marked as computed and read-only
	statusField := findFieldByName(resource.Fields, "Status")
	if statusField == nil {
		t.Fatal("Status field not found")
	}
	if !statusField.IsComputed {
		t.Error("Status field should be marked as computed")
	}
	if !statusField.ReadOnly {
		t.Error("Status field should be read-only")
	}
}

func TestFieldOrderWithManyToOneRelationship(t *testing.T) {
	// Test struct with relationship
	type TestCategory struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}

	type TestProduct struct {
		ID         uint          `json:"id"`
		Name       string        `json:"name"`
		Price      float64       `json:"price"`
		CategoryID uint          `json:"category_id"`
		Category   *TestCategory `json:"category"` // Relationship field exists in struct
	}

	mockAdapter := &MockAdapter{}
	admin := New(mockAdapter, auth.WithNoAuth())

	// Register with mixed field types in specific order
	admin.RegisterResource(&TestProduct{}).
		WithField("Name", func(f *FieldBuilder) {
			f.DisplayName("Product Name")
		}).
		WithDerivedField("PriceFormatted", "Formatted Price", func(product any) string {
			return "$99.99"
		}).
		WithManyToOneField("Category", "Category", func(r *RelationshipBuilder) {
			r.DisplayField("Name")
		}).
		WithField("Price", func(f *FieldBuilder) {
			f.DisplayName("Price ($)")
		})

	resource, exists := admin.GetResource("TestProduct")
	if !exists {
		t.Fatal("TestProduct resource not found")
	}

	// Expected order: ID (primary key), then registration order
	expectedOrder := []string{
		"ID",             // Primary key (always first)
		"Name",           // First WithField
		"PriceFormatted", // WithComputedField
		"Category",       // WithManyToOneField (internally calls WithField)
		"Price",          // Second WithField
	}

	if len(resource.Fields) != len(expectedOrder) {
		t.Logf("Actual fields:")
		for i, field := range resource.Fields {
			t.Logf("  [%d] %s (computed: %v, relationship: %v)", i, field.Name, field.IsComputed, field.Relationship != nil)
		}
		t.Fatalf("Expected %d fields, got %d", len(expectedOrder), len(resource.Fields))
	}

	for i, expectedField := range expectedOrder {
		if resource.Fields[i].Name != expectedField {
			t.Errorf("Field at position %d: expected %s, got %s", i, expectedField, resource.Fields[i].Name)
		}
	}

	// Verify relationship field configuration
	categoryField := findFieldByName(resource.Fields, "Category")
	if categoryField == nil {
		t.Fatal("Category field not found")
	}
	if categoryField.Relationship == nil {
		t.Error("Category field should have relationship info")
	}
	if categoryField.Relationship.Type != RelationshipManyToOne {
		t.Error("Category field should be many-to-one relationship")
	}
}

// Helper function to find a field by name
func findFieldByName(fields []FieldInfo, name string) *FieldInfo {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}
	return nil
}

// Mock adapter for testing
type MockAdapter struct{}

func (m *MockAdapter) Find(ctx context.Context, resource *Resource, query *Query) (*Result, error) {
	return &Result{
		Items:      []any{},
		TotalCount: 0,
		HasMore:    false,
		Query:      *query,
	}, nil
}

func (m *MockAdapter) GetAll(ctx context.Context, resource *Resource, filters map[string]any) ([]any, error) {
	return []any{}, nil
}

func (m *MockAdapter) GetByID(ctx context.Context, resource *Resource, id any) (any, error) {
	return nil, nil
}

func (m *MockAdapter) Create(ctx context.Context, resource *Resource, data any) error {
	return nil
}

func (m *MockAdapter) Update(ctx context.Context, resource *Resource, id any, data any) error {
	return nil
}

func (m *MockAdapter) Delete(ctx context.Context, resource *Resource, id any) error {
	return nil
}

func (m *MockAdapter) GetSchema(resource *Resource) (*Schema, error) {
	return &Schema{}, nil
}

func (m *MockAdapter) ValidateData(resource *Resource, data any) error {
	return nil
}

func (m *MockAdapter) Count(ctx context.Context, resource *Resource, filters map[string]any) (int64, error) {
	return 0, nil
}

func (m *MockAdapter) Search(ctx context.Context, resource *Resource, query string) ([]any, error) {
	return []any{}, nil
}
