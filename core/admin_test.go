package core

import (
	"context"
	"testing"

	"github.com/preslavrachev/backoffice/middleware/auth"
)

type ResourceStatus int

const (
	ResourceNotFound ResourceStatus = iota
	ResourceFound
)

// DummyAdapter is a minimal stub for testing.
type DummyAdapter struct{}

// Create implements Adapter.
func (d *DummyAdapter) Create(ctx context.Context, resource *Resource, data any) error {
	panic("unimplemented")
}

// Delete implements Adapter.
func (d *DummyAdapter) Delete(ctx context.Context, resource *Resource, id any) error {
	panic("unimplemented")
}

// Find implements Adapter.
func (d *DummyAdapter) Find(ctx context.Context, resource *Resource, query *Query) (*Result, error) {
	panic("unimplemented")
}

// GetAll implements Adapter.
func (d *DummyAdapter) GetAll(ctx context.Context, resource *Resource, filters map[string]any) ([]any, error) {
	panic("unimplemented")
}

// GetByID implements Adapter.
func (d *DummyAdapter) GetByID(ctx context.Context, resource *Resource, id any) (any, error) {
	panic("unimplemented")
}

// GetSchema implements Adapter.
func (d *DummyAdapter) GetSchema(resource *Resource) (*Schema, error) {
	panic("unimplemented")
}

// Search implements Adapter.
func (d *DummyAdapter) Search(ctx context.Context, resource *Resource, query string) ([]any, error) {
	panic("unimplemented")
}

// Update implements Adapter.
func (d *DummyAdapter) Update(ctx context.Context, resource *Resource, id any, data any) error {
	panic("unimplemented")
}

// ValidateData implements Adapter.
func (d *DummyAdapter) ValidateData(resource *Resource, data any) error {
	panic("unimplemented")
}

// Count is a stub implementation to satisfy the Adapter interface.

func (d *DummyAdapter) Count(ctx context.Context, resource *Resource, filters map[string]any) (int64, error) {
	return 0, nil
}

func TestBackOffice_GetResource(t *testing.T) {
	adapter := &DummyAdapter{}
	authConfig := auth.AuthConfig{}
	bo := New(adapter, authConfig)

	// Register a dummy resource
	type User struct{}
	bo.RegisterResource(&User{})

	tests := []struct {
		name             string
		resourceName     string
		status           ResourceStatus
		wantResourceName string
	}{
		{
			name:             "resource exists",
			resourceName:     "User",
			status:           ResourceFound,
			wantResourceName: "User",
		},
		{
			name:             "resource does not exist",
			resourceName:     "NonExistent",
			status:           ResourceNotFound,
			wantResourceName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, exists := bo.GetResource(tt.resourceName)
			switch tt.status {
			case ResourceFound:
				if !exists {
					t.Errorf("expected resource to exist, got exists=%v", exists)
				}
				if resource == nil {
					t.Errorf("expected resource to be not nil")
				}
				if resource != nil && resource.Name != tt.wantResourceName {
					t.Errorf("expected resource name %q, got %q", tt.wantResourceName, resource.Name)
				}
			case ResourceNotFound:
				if exists {
					t.Errorf("expected resource to not exist, got exists=%v", exists)
				}
				if resource != nil {
					t.Errorf("expected resource to be nil, got %v", resource)
				}
			}
		})
	}
}
