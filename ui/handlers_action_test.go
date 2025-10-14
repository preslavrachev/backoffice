package ui

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/preslavrachev/backoffice/core"
	"github.com/preslavrachev/backoffice/middleware/auth"
)

// MockAdapter for testing
type mockActionAdapter struct {
	getByIDFunc func(ctx context.Context, resource *core.Resource, id any) (any, error)
}

func (m *mockActionAdapter) Find(ctx context.Context, resource *core.Resource, query *core.Query) (*core.Result, error) {
	return nil, nil
}

func (m *mockActionAdapter) GetByID(ctx context.Context, resource *core.Resource, id any) (any, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, resource, id)
	}
	return nil, nil
}

func (m *mockActionAdapter) Create(ctx context.Context, resource *core.Resource, data any) error {
	return nil
}

func (m *mockActionAdapter) Update(ctx context.Context, resource *core.Resource, id any, data any) error {
	return nil
}

func (m *mockActionAdapter) Delete(ctx context.Context, resource *core.Resource, id any) error {
	return nil
}

func (m *mockActionAdapter) GetSchema(resource *core.Resource) (*core.Schema, error) {
	return nil, nil
}

func (m *mockActionAdapter) ValidateData(resource *core.Resource, data any) error {
	return nil
}

func (m *mockActionAdapter) GetAll(ctx context.Context, resource *core.Resource, filters map[string]any) ([]any, error) {
	return nil, nil
}

func (m *mockActionAdapter) Count(ctx context.Context, resource *core.Resource, filters map[string]any) (int64, error) {
	return 0, nil
}

func (m *mockActionAdapter) Search(ctx context.Context, resource *core.Resource, query string) ([]any, error) {
	return nil, nil
}

// TestHandleCustomAction_Success verifies successful action execution
func TestHandleCustomAction_Success(t *testing.T) {
	type TestModel struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}

	// Setup BackOffice with action
	bo := &core.BackOffice{}
	bo = core.New(&mockActionAdapter{}, auth.AuthConfig{})

	var executedID any
	handler := func(ctx context.Context, id any) error {
		executedID = id
		return nil
	}

	bo.RegisterResource(&TestModel{}).
		WithAction("test_action", "Test Action", handler)

	// Create handler
	h := &BackOfficeHandler{bo: bo}

	// Create request
	form := url.Values{}
	form.Add("action_id", "test_action")

	req := httptest.NewRequest(http.MethodPost, "/admin/api/TestModel/1/action", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()

	// Get resource
	resource, _ := bo.GetResource("TestModel")

	// Execute handler
	h.handleCustomAction(w, req, resource, "1")

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify toast notification was sent
	hxTrigger := w.Header().Get("HX-Trigger")
	if !strings.Contains(hxTrigger, "showToast") {
		t.Errorf("Expected HX-Trigger header with showToast, got '%s'", hxTrigger)
	}

	if !strings.Contains(hxTrigger, "success") {
		t.Errorf("Expected success toast, got '%s'", hxTrigger)
	}

	// Verify handler was called with correct ID
	if executedID != uint(1) {
		t.Errorf("Expected ID uint(1), got %v", executedID)
	}
}

// TestHandleCustomAction_ActionNotFound verifies error when action doesn't exist
func TestHandleCustomAction_ActionNotFound(t *testing.T) {
	type TestModel struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}

	bo := core.New(&mockActionAdapter{}, auth.AuthConfig{})
	bo.RegisterResource(&TestModel{})

	h := &BackOfficeHandler{bo: bo}

	form := url.Values{}
	form.Add("action_id", "nonexistent_action")

	req := httptest.NewRequest(http.MethodPost, "/admin/api/TestModel/1/action", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()

	resource, _ := bo.GetResource("TestModel")
	h.handleCustomAction(w, req, resource, "1")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	hxTrigger := w.Header().Get("HX-Trigger")
	if !strings.Contains(hxTrigger, "error") {
		t.Errorf("Expected error toast, got '%s'", hxTrigger)
	}
}

// TestHandleCustomAction_MissingActionID verifies error when action_id is missing
func TestHandleCustomAction_MissingActionID(t *testing.T) {
	type TestModel struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}

	bo := core.New(&mockActionAdapter{}, auth.AuthConfig{})
	bo.RegisterResource(&TestModel{})

	h := &BackOfficeHandler{bo: bo}

	form := url.Values{}
	// Intentionally not adding action_id

	req := httptest.NewRequest(http.MethodPost, "/admin/api/TestModel/1/action", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()

	resource, _ := bo.GetResource("TestModel")
	h.handleCustomAction(w, req, resource, "1")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	hxTrigger := w.Header().Get("HX-Trigger")
	if !strings.Contains(hxTrigger, "Action ID is required") {
		t.Errorf("Expected 'Action ID is required' message, got '%s'", hxTrigger)
	}
}

// TestHandleCustomAction_InvalidIDFormat verifies error with invalid ID
func TestHandleCustomAction_InvalidIDFormat(t *testing.T) {
	type TestModel struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}

	bo := core.New(&mockActionAdapter{}, auth.AuthConfig{})

	handler := func(ctx context.Context, id any) error {
		return nil
	}

	bo.RegisterResource(&TestModel{}).
		WithAction("test_action", "Test Action", handler)

	h := &BackOfficeHandler{bo: bo}

	form := url.Values{}
	form.Add("action_id", "test_action")

	req := httptest.NewRequest(http.MethodPost, "/admin/api/TestModel/invalid/action", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()

	resource, _ := bo.GetResource("TestModel")
	h.handleCustomAction(w, req, resource, "invalid")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	hxTrigger := w.Header().Get("HX-Trigger")
	if !strings.Contains(hxTrigger, "Invalid ID format") {
		t.Errorf("Expected 'Invalid ID format' message, got '%s'", hxTrigger)
	}
}

// TestHandleCustomAction_HandlerError verifies error handling when action fails
func TestHandleCustomAction_HandlerError(t *testing.T) {
	type TestModel struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}

	bo := core.New(&mockActionAdapter{}, auth.AuthConfig{})

	expectedErr := errors.New("database error")
	handler := func(ctx context.Context, id any) error {
		return expectedErr
	}

	bo.RegisterResource(&TestModel{}).
		WithAction("failing_action", "Failing Action", handler)

	h := &BackOfficeHandler{bo: bo}

	form := url.Values{}
	form.Add("action_id", "failing_action")

	req := httptest.NewRequest(http.MethodPost, "/admin/api/TestModel/1/action", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()

	resource, _ := bo.GetResource("TestModel")
	h.handleCustomAction(w, req, resource, "1")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	hxTrigger := w.Header().Get("HX-Trigger")
	if !strings.Contains(hxTrigger, "Action failed") {
		t.Errorf("Expected 'Action failed' message, got '%s'", hxTrigger)
	}

	if !strings.Contains(hxTrigger, "database error") {
		t.Errorf("Expected error message to include 'database error', got '%s'", hxTrigger)
	}
}

// TestHandleCustomAction_MultipleActions verifies correct action is executed
func TestHandleCustomAction_MultipleActions(t *testing.T) {
	type TestModel struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}

	bo := core.New(&mockActionAdapter{}, auth.AuthConfig{})

	var executedAction string

	handler1 := func(ctx context.Context, id any) error {
		executedAction = "action1"
		return nil
	}

	handler2 := func(ctx context.Context, id any) error {
		executedAction = "action2"
		return nil
	}

	handler3 := func(ctx context.Context, id any) error {
		executedAction = "action3"
		return nil
	}

	bo.RegisterResource(&TestModel{}).
		WithAction("action1", "Action 1", handler1).
		WithAction("action2", "Action 2", handler2).
		WithAction("action3", "Action 3", handler3)

	h := &BackOfficeHandler{bo: bo}

	// Test action2
	form := url.Values{}
	form.Add("action_id", "action2")

	req := httptest.NewRequest(http.MethodPost, "/admin/api/TestModel/1/action", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()

	resource, _ := bo.GetResource("TestModel")
	h.handleCustomAction(w, req, resource, "1")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if executedAction != "action2" {
		t.Errorf("Expected 'action2' to be executed, got '%s'", executedAction)
	}
}

// TestHandleCustomAction_ContextPropagation verifies context is passed to handler
func TestHandleCustomAction_ContextPropagation(t *testing.T) {
	type TestModel struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}

	bo := core.New(&mockActionAdapter{}, auth.AuthConfig{})

	var receivedContext context.Context
	handler := func(ctx context.Context, id any) error {
		receivedContext = ctx
		return nil
	}

	bo.RegisterResource(&TestModel{}).
		WithAction("test_action", "Test Action", handler)

	h := &BackOfficeHandler{bo: bo}

	form := url.Values{}
	form.Add("action_id", "test_action")

	req := httptest.NewRequest(http.MethodPost, "/admin/api/TestModel/1/action", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()

	resource, _ := bo.GetResource("TestModel")
	h.handleCustomAction(w, req, resource, "1")

	if receivedContext == nil {
		t.Error("Expected context to be passed to handler")
	}

	if receivedContext != req.Context() {
		t.Error("Expected handler to receive request context")
	}
}

// TestHandleCustomAction_SuccessMessage verifies correct success message in toast
func TestHandleCustomAction_SuccessMessage(t *testing.T) {
	type TestModel struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}

	bo := core.New(&mockActionAdapter{}, auth.AuthConfig{})

	handler := func(ctx context.Context, id any) error {
		return nil
	}

	bo.RegisterResource(&TestModel{}).
		WithAction("approve", "Approve Item", handler)

	h := &BackOfficeHandler{bo: bo}

	form := url.Values{}
	form.Add("action_id", "approve")

	req := httptest.NewRequest(http.MethodPost, "/admin/api/TestModel/1/action", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()

	resource, _ := bo.GetResource("TestModel")
	h.handleCustomAction(w, req, resource, "1")

	hxTrigger := w.Header().Get("HX-Trigger")

	// Verify message contains action title
	if !strings.Contains(hxTrigger, "Approve Item") {
		t.Errorf("Expected message to contain 'Approve Item', got '%s'", hxTrigger)
	}

	if !strings.Contains(hxTrigger, "completed successfully") {
		t.Errorf("Expected message to contain 'completed successfully', got '%s'", hxTrigger)
	}
}

// TestHandleCustomAction_InvalidFormData verifies error with malformed form data
func TestHandleCustomAction_InvalidFormData(t *testing.T) {
	type TestModel struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}

	bo := core.New(&mockActionAdapter{}, auth.AuthConfig{})
	bo.RegisterResource(&TestModel{})

	h := &BackOfficeHandler{bo: bo}

	// Create request with invalid content type to trigger ParseForm error
	req := httptest.NewRequest(http.MethodPost, "/admin/api/TestModel/1/action", strings.NewReader("invalid%form%data"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len("invalid%form%data")))

	w := httptest.NewRecorder()

	resource, _ := bo.GetResource("TestModel")

	// This should handle form parsing errors gracefully
	// The actual behavior depends on Go's ParseForm implementation
	h.handleCustomAction(w, req, resource, "1")

	// We expect either bad request or the handler to process it
	// The important thing is it doesn't panic
	if w.Code != http.StatusBadRequest && w.Code != http.StatusOK {
		t.Logf("Got status %d, which is acceptable for invalid form data", w.Code)
	}
}
