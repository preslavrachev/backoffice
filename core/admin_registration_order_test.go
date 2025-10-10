package core

import (
	"context"
	"testing"

	"github.com/preslavrachev/backoffice/middleware/auth"
)

// Test structs for registration order testing
type TestEntityA struct {
	ID   uint   `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

type TestEntityB struct {
	ID   uint   `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

type TestEntityC struct {
	ID   uint   `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

type TestEntityD struct {
	ID   uint   `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

// TestResourceRegistrationOrder tests that resources are returned in registration order
func TestResourceRegistrationOrder(t *testing.T) {
	// Create a mock adapter (we only need the BackOffice instance, not actual DB operations)
	mockAdapter := &orderTestMockAdapter{}
	admin := New(mockAdapter, auth.WithNoAuth())

	// Register resources in a specific order
	expectedOrder := []string{"TestEntityA", "TestEntityB", "TestEntityC", "TestEntityD"}

	admin.RegisterResource(&TestEntityA{})
	admin.RegisterResource(&TestEntityB{})
	admin.RegisterResource(&TestEntityC{})
	admin.RegisterResource(&TestEntityD{})

	// Test that GetResources exists and returns resources in registration order
	orderedResources := admin.GetResources()

	if len(orderedResources) != len(expectedOrder) {
		t.Fatalf("Expected %d resources, got %d", len(expectedOrder), len(orderedResources))
	}

	// Verify the order matches registration order
	for i, expectedName := range expectedOrder {
		if orderedResources[i].Name != expectedName {
			t.Errorf("Expected resource at position %d to be %s, got %s",
				i, expectedName, orderedResources[i].Name)
		}
	}
}

// TestResourceOrderConsistency tests that the order is consistent across multiple calls
func TestResourceOrderConsistency(t *testing.T) {
	mockAdapter := &orderTestMockAdapter{}
	admin := New(mockAdapter, auth.WithNoAuth())

	// Register resources
	admin.RegisterResource(&TestEntityA{})
	admin.RegisterResource(&TestEntityB{})
	admin.RegisterResource(&TestEntityC{})
	admin.RegisterResource(&TestEntityD{})

	// Get ordered resources multiple times
	var orders [][]string
	for i := 0; i < 10; i++ {
		orderedResources := admin.GetResources()
		var order []string
		for _, resource := range orderedResources {
			order = append(order, resource.Name)
		}
		orders = append(orders, order)
	}

	// Verify all orders are identical
	expectedOrder := orders[0]
	for i, order := range orders {
		if len(order) != len(expectedOrder) {
			t.Fatalf("Iteration %d: Expected %d resources, got %d", i, len(expectedOrder), len(order))
		}
		for j, name := range order {
			if name != expectedOrder[j] {
				t.Errorf("Iteration %d: Order inconsistency at position %d: expected %s, got %s",
					i, j, expectedOrder[j], name)
			}
		}
	}
}

// TestResourceOrderWithHiddenResources tests that hidden resources are filtered but order is preserved
func TestResourceOrderWithHiddenResources(t *testing.T) {
	mockAdapter := &orderTestMockAdapter{}
	admin := New(mockAdapter, auth.WithNoAuth())

	// Register resources with one hidden
	admin.RegisterResource(&TestEntityA{})
	admin.RegisterResource(&TestEntityB{}).Hidden(true) // This one should be filtered out
	admin.RegisterResource(&TestEntityC{})
	admin.RegisterResource(&TestEntityD{})

	// Get all resources (including hidden)
	allResources := admin.GetResources()
	expectedAll := []string{"TestEntityA", "TestEntityB", "TestEntityC", "TestEntityD"}

	if len(allResources) != len(expectedAll) {
		t.Fatalf("Expected %d total resources, got %d", len(expectedAll), len(allResources))
	}

	for i, expectedName := range expectedAll {
		if allResources[i].Name != expectedName {
			t.Errorf("Expected resource at position %d to be %s, got %s",
				i, expectedName, allResources[i].Name)
		}
	}

	// Verify TestEntityB is marked as hidden
	if !allResources[1].Hidden {
		t.Error("Expected TestEntityB to be hidden")
	}
}

// Mock adapter for testing (minimal implementation)
type orderTestMockAdapter struct{}

func (m *orderTestMockAdapter) Find(ctx context.Context, resource *Resource, query *Query) (*Result, error) {
	return &Result{Items: []any{}, TotalCount: 0, HasMore: false}, nil
}

func (m *orderTestMockAdapter) GetByID(ctx context.Context, resource *Resource, id any) (any, error) {
	return nil, nil
}

func (m *orderTestMockAdapter) Create(ctx context.Context, resource *Resource, data any) error {
	return nil
}

func (m *orderTestMockAdapter) Update(ctx context.Context, resource *Resource, id any, data any) error {
	return nil
}

func (m *orderTestMockAdapter) Delete(ctx context.Context, resource *Resource, id any) error {
	return nil
}

func (m *orderTestMockAdapter) GetSchema(resource *Resource) (*Schema, error) {
	return &Schema{}, nil
}

func (m *orderTestMockAdapter) ValidateData(resource *Resource, data any) error {
	return nil
}

func (m *orderTestMockAdapter) Count(ctx context.Context, resource *Resource, filters map[string]any) (int64, error) {
	return 0, nil
}

func (m *orderTestMockAdapter) Search(ctx context.Context, resource *Resource, query string) ([]any, error) {
	return []any{}, nil
}

func (m *orderTestMockAdapter) GetAll(ctx context.Context, resource *Resource, filters map[string]any) ([]any, error) {
	return []any{}, nil
}
