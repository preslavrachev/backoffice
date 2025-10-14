package core

import (
	"context"
	"errors"
	"testing"
)

// TestResourceBuilder_WithAction verifies that actions can be registered on resources
func TestResourceBuilder_WithAction(t *testing.T) {
	type TestModel struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}

	bo := &BackOffice{
		resources:     make(map[string]*Resource),
		resourceOrder: []string{},
		config:        &Config{},
	}

	handler := func(ctx context.Context, id any) error {
		return nil
	}

	bo.RegisterResource(&TestModel{}).
		WithAction("approve", "Approve Item", handler).
		WithAction("reject", "Reject Item", handler)

	resource, exists := bo.GetResource("TestModel")
	if !exists {
		t.Fatal("Resource not registered")
	}

	if len(resource.Actions) != 2 {
		t.Fatalf("Expected 2 actions, got %d", len(resource.Actions))
	}

	if resource.Actions[0].ID != "approve" {
		t.Errorf("Expected first action ID 'approve', got '%s'", resource.Actions[0].ID)
	}

	if resource.Actions[0].Title != "Approve Item" {
		t.Errorf("Expected first action title 'Approve Item', got '%s'", resource.Actions[0].Title)
	}

	if resource.Actions[1].ID != "reject" {
		t.Errorf("Expected second action ID 'reject', got '%s'", resource.Actions[1].ID)
	}

	if resource.Actions[1].Title != "Reject Item" {
		t.Errorf("Expected second action title 'Reject Item', got '%s'", resource.Actions[1].Title)
	}
}

// TestResourceBuilder_WithAction_EmptyActions verifies resources without actions have empty slice
func TestResourceBuilder_WithAction_EmptyActions(t *testing.T) {
	type TestModel struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}

	bo := &BackOffice{
		resources:     make(map[string]*Resource),
		resourceOrder: []string{},
		config:        &Config{},
	}

	bo.RegisterResource(&TestModel{})

	resource, exists := bo.GetResource("TestModel")
	if !exists {
		t.Fatal("Resource not registered")
	}

	if resource.Actions == nil {
		t.Error("Actions should be initialized, not nil")
	}

	if len(resource.Actions) != 0 {
		t.Errorf("Expected 0 actions, got %d", len(resource.Actions))
	}
}

// TestResourceBuilder_WithAction_MultipleResources verifies actions are isolated per resource
func TestResourceBuilder_WithAction_MultipleResources(t *testing.T) {
	type User struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}

	type Product struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}

	bo := &BackOffice{
		resources:     make(map[string]*Resource),
		resourceOrder: []string{},
		config:        &Config{},
	}

	userHandler := func(ctx context.Context, id any) error {
		return nil
	}

	productHandler := func(ctx context.Context, id any) error {
		return nil
	}

	bo.RegisterResource(&User{}).
		WithAction("activate", "Activate User", userHandler).
		WithAction("deactivate", "Deactivate User", userHandler)

	bo.RegisterResource(&Product{}).
		WithAction("publish", "Publish Product", productHandler)

	userResource, exists := bo.GetResource("User")
	if !exists {
		t.Fatal("User resource not registered")
	}

	productResource, exists := bo.GetResource("Product")
	if !exists {
		t.Fatal("Product resource not registered")
	}

	if len(userResource.Actions) != 2 {
		t.Errorf("Expected 2 user actions, got %d", len(userResource.Actions))
	}

	if len(productResource.Actions) != 1 {
		t.Errorf("Expected 1 product action, got %d", len(productResource.Actions))
	}

	// Verify user actions
	if userResource.Actions[0].ID != "activate" {
		t.Errorf("Expected user action 'activate', got '%s'", userResource.Actions[0].ID)
	}

	// Verify product actions
	if productResource.Actions[0].ID != "publish" {
		t.Errorf("Expected product action 'publish', got '%s'", productResource.Actions[0].ID)
	}
}

// TestResourceBuilder_WithAction_ChainedCalls verifies fluent API chaining works correctly
func TestResourceBuilder_WithAction_ChainedCalls(t *testing.T) {
	type TestModel struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}

	bo := &BackOffice{
		resources:     make(map[string]*Resource),
		resourceOrder: []string{},
		config:        &Config{},
	}

	handler := func(ctx context.Context, id any) error {
		return nil
	}

	// Test that WithAction returns ResourceBuilder for chaining
	builder := bo.RegisterResource(&TestModel{}).
		WithName("Test Item").
		WithAction("action1", "Action 1", handler).
		WithAction("action2", "Action 2", handler).
		WithDefaultSort("ID", SortAsc)

	// Verify we can still chain other methods after WithAction
	if builder == nil {
		t.Fatal("Builder should not be nil")
	}

	resource, exists := bo.GetResource("TestModel")
	if !exists {
		t.Fatal("Resource not registered")
	}

	// Verify all configurations were applied
	if resource.DisplayName != "Test Item" {
		t.Errorf("Expected display name 'Test Item', got '%s'", resource.DisplayName)
	}

	if len(resource.Actions) != 2 {
		t.Errorf("Expected 2 actions, got %d", len(resource.Actions))
	}

	if resource.DefaultSort.Field != "ID" {
		t.Errorf("Expected default sort field 'ID', got '%s'", resource.DefaultSort.Field)
	}
}

// TestResourceBuilder_WithAction_HandlerExecution verifies handlers execute with correct parameters
func TestResourceBuilder_WithAction_HandlerExecution(t *testing.T) {
	type TestModel struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}

	bo := &BackOffice{
		resources:     make(map[string]*Resource),
		resourceOrder: []string{},
		config:        &Config{},
	}

	var executedID any
	var executionCount int

	handler := func(ctx context.Context, id any) error {
		executedID = id
		executionCount++
		return nil
	}

	bo.RegisterResource(&TestModel{}).
		WithAction("test", "Test Action", handler)

	resource, _ := bo.GetResource("TestModel")

	// Execute the action
	testID := uint(42)
	err := resource.Actions[0].Handler(context.Background(), testID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if executionCount != 1 {
		t.Errorf("Expected handler to be called once, got %d", executionCount)
	}

	if executedID != testID {
		t.Errorf("Expected ID %v, got %v", testID, executedID)
	}
}

// TestResourceBuilder_WithAction_ErrorHandling verifies error propagation from handlers
func TestResourceBuilder_WithAction_ErrorHandling(t *testing.T) {
	type TestModel struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}

	bo := &BackOffice{
		resources:     make(map[string]*Resource),
		resourceOrder: []string{},
		config:        &Config{},
	}

	expectedErr := errors.New("operation failed")

	handler := func(ctx context.Context, id any) error {
		return expectedErr
	}

	bo.RegisterResource(&TestModel{}).
		WithAction("failing_action", "Failing Action", handler)

	resource, _ := bo.GetResource("TestModel")

	err := resource.Actions[0].Handler(context.Background(), 1)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("Expected error '%v', got '%v'", expectedErr, err)
	}
}

// TestResourceBuilder_WithAction_OrderPreservation verifies actions maintain registration order
func TestResourceBuilder_WithAction_OrderPreservation(t *testing.T) {
	type TestModel struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}

	bo := &BackOffice{
		resources:     make(map[string]*Resource),
		resourceOrder: []string{},
		config:        &Config{},
	}

	handler := func(ctx context.Context, id any) error {
		return nil
	}

	bo.RegisterResource(&TestModel{}).
		WithAction("first", "First Action", handler).
		WithAction("second", "Second Action", handler).
		WithAction("third", "Third Action", handler).
		WithAction("fourth", "Fourth Action", handler)

	resource, _ := bo.GetResource("TestModel")

	expectedOrder := []string{"first", "second", "third", "fourth"}

	if len(resource.Actions) != len(expectedOrder) {
		t.Fatalf("Expected %d actions, got %d", len(expectedOrder), len(resource.Actions))
	}

	for i, expectedID := range expectedOrder {
		if resource.Actions[i].ID != expectedID {
			t.Errorf("Action at position %d: expected ID '%s', got '%s'", i, expectedID, resource.Actions[i].ID)
		}
	}
}
