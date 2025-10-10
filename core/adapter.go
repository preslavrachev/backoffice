package core

import "context"

// Adapter defines the interface for data source adapters
type Adapter interface {
	// Resource operations
	Find(ctx context.Context, resource *Resource, query *Query) (*Result, error)
	GetByID(ctx context.Context, resource *Resource, id any) (any, error)
	Create(ctx context.Context, resource *Resource, data any) error
	Update(ctx context.Context, resource *Resource, id any, data any) error
	Delete(ctx context.Context, resource *Resource, id any) error

	// Metadata operations
	GetSchema(resource *Resource) (*Schema, error)
	ValidateData(resource *Resource, data any) error

	// Legacy operations (deprecated, will be removed)
	GetAll(ctx context.Context, resource *Resource, filters map[string]any) ([]any, error)
	Count(ctx context.Context, resource *Resource, filters map[string]any) (int64, error)
	Search(ctx context.Context, resource *Resource, query string) ([]any, error)
}

// Schema represents the structure of a resource
type Schema struct {
	Fields     []FieldInfo    `json:"fields"`
	PrimaryKey string         `json:"primary_key"`
	TableName  string         `json:"table_name"`
	Metadata   map[string]any `json:"metadata"`
}
