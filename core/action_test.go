package core

import (
	"context"
	"errors"
	"testing"
)

// TestNewAction verifies that NewAction creates a valid action with all fields set
func TestNewAction(t *testing.T) {
	called := false
	handler := func(ctx context.Context, id any) error {
		called = true
		return nil
	}

	action := NewAction("test_action", "Test Action", handler).Build()

	if action.ID != "test_action" {
		t.Errorf("Expected ID 'test_action', got '%s'", action.ID)
	}

	if action.Title != "Test Action" {
		t.Errorf("Expected Title 'Test Action', got '%s'", action.Title)
	}

	if action.Handler == nil {
		t.Fatal("Handler should not be nil")
	}

	// Test that handler works
	err := action.Handler(context.Background(), 1)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !called {
		t.Error("Handler was not called")
	}
}

// TestActionHandler_Success verifies that action handlers execute successfully
func TestActionHandler_Success(t *testing.T) {
	var capturedID any
	handler := func(ctx context.Context, id any) error {
		capturedID = id
		return nil
	}

	action := CustomAction{
		ID:      "approve",
		Title:   "Approve",
		Handler: handler,
	}

	err := action.Handler(context.Background(), uint(123))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if capturedID != uint(123) {
		t.Errorf("Expected ID uint(123), got %v", capturedID)
	}
}

// TestActionHandler_Error verifies that action handlers properly return errors
func TestActionHandler_Error(t *testing.T) {
	expectedErr := errors.New("action failed")
	handler := func(ctx context.Context, id any) error {
		return expectedErr
	}

	action := CustomAction{
		ID:      "reject",
		Title:   "Reject",
		Handler: handler,
	}

	err := action.Handler(context.Background(), 1)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("Expected error '%v', got '%v'", expectedErr, err)
	}
}

// TestActionHandler_ContextPropagation verifies that context is properly passed to handlers
func TestActionHandler_ContextPropagation(t *testing.T) {
	type ctxKey string
	testKey := ctxKey("test")
	testValue := "test-value"

	handler := func(ctx context.Context, id any) error {
		value := ctx.Value(testKey)
		if value == nil {
			return errors.New("context value not found")
		}
		if value != testValue {
			return errors.New("context value mismatch")
		}
		return nil
	}

	action := CustomAction{
		ID:      "check_context",
		Title:   "Check Context",
		Handler: handler,
	}

	ctx := context.WithValue(context.Background(), testKey, testValue)
	err := action.Handler(ctx, 1)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

// TestActionHandler_DifferentIDTypes verifies that handlers work with various ID types
func TestActionHandler_DifferentIDTypes(t *testing.T) {
	tests := []struct {
		name string
		id   any
	}{
		{"uint", uint(42)},
		{"int", 42},
		{"string", "abc123"},
		{"uint64", uint64(9999999999)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedID any
			handler := func(ctx context.Context, id any) error {
				capturedID = id
				return nil
			}

			action := CustomAction{
				ID:      "type_test",
				Title:   "Type Test",
				Handler: handler,
			}

			err := action.Handler(context.Background(), tt.id)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if capturedID != tt.id {
				t.Errorf("Expected ID %v, got %v", tt.id, capturedID)
			}
		})
	}
}
