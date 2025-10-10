package core

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"backoffice/middleware/auth"

	"github.com/iancoleman/strcase"
)

// BackOffice represents the main admin instance
type BackOffice struct {
	adapter       Adapter
	resources     map[string]*Resource
	resourceOrder []string // Track registration order for consistent display
	config        *Config
}

// Config holds configuration for the BackOffice instance
type Config struct {
	BasePath     string                            `json:"base_path"`
	Title        string                            `json:"title"`
	ItemsPerPage int                               `json:"items_per_page"`
	Resources    map[string]*ResourceConfig        `json:"resources"`
	Middleware   []func(http.Handler) http.Handler `json:"-"`
	Auth         *auth.AuthConfig                  `json:"-"`
}

// ResourceConfig holds configuration for individual resources
type ResourceConfig struct {
	DisplayName string `json:"display_name"`
	PluralName  string `json:"plural_name"`
	Hidden      bool   `json:"hidden"`
	ReadOnly    bool   `json:"read_only"`
}

// New creates a new BackOffice instance with the given adapter and auth configuration
func New(adapter Adapter, authConfig auth.AuthConfig) *BackOffice {
	return &BackOffice{
		adapter:       adapter,
		resources:     make(map[string]*Resource),
		resourceOrder: make([]string, 0),
		config: &Config{
			BasePath:     "/admin",
			Title:        "BackOffice Admin",
			ItemsPerPage: 20,
			Resources:    make(map[string]*ResourceConfig),
			Middleware:   []func(http.Handler) http.Handler{},
			Auth:         &authConfig,
		},
	}
}

// RegisterResource registers a new resource with the admin panel
func (bo *BackOffice) RegisterResource(model any) *ResourceBuilder {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() != reflect.Ptr {
		panic("RegisterResource expects a pointer to a struct")
	}

	elemType := modelType.Elem()
	if elemType.Kind() != reflect.Struct {
		panic("RegisterResource expects a pointer to a struct")
	}

	// Generate resource name from type
	resourceName := elemType.Name()

	// Create resource
	resource := &Resource{
		Name:         resourceName,
		DisplayName:  generateDisplayName(resourceName),
		PluralName:   generatePluralName(resourceName),
		Model:        model,
		ModelType:    modelType,
		TableName:    generateTableName(resourceName),
		Hidden:       false,
		ReadOnly:     false,
		FieldConfigs: make(map[string]*FieldConfig),
		FieldOrder:   []string{}, // Initialize empty order slice
	}

	// Discover fields using reflection
	if err := resource.DiscoverFields(); err != nil {
		panic(fmt.Sprintf("Failed to discover fields for %s: %v", resourceName, err))
	}

	// Store resource
	bo.resources[resourceName] = resource

	// Track registration order for consistent display
	bo.resourceOrder = append(bo.resourceOrder, resourceName)

	// Return builder for fluent configuration
	return &ResourceBuilder{
		backoffice: bo,
		resource:   resource,
	}
}

// GetResource retrieves a registered resource by name
func (bo *BackOffice) GetResource(name string) (*Resource, bool) {
	resource, exists := bo.resources[name]
	return resource, exists
}

// GetResources returns all registered resources in registration order
// TODO: Add method to customize resource display order
// Future: admin.SetResourceOrder([]string{"User", "Product", "Category"})
// or admin.RegisterResource(&User{}).WithDisplayOrder(1)
func (bo *BackOffice) GetResources() []*Resource {
	ordered := make([]*Resource, 0, len(bo.resourceOrder))
	for _, name := range bo.resourceOrder {
		if resource, exists := bo.resources[name]; exists {
			ordered = append(ordered, resource)
		}
	}
	return ordered
}

// GetConfig returns the configuration
func (bo *BackOffice) GetConfig() *Config {
	return bo.config
}

// GetAdapter returns the adapter
func (bo *BackOffice) GetAdapter() Adapter {
	return bo.adapter
}

// GetAuth returns the authentication configuration
func (bo *BackOffice) GetAuth() *auth.AuthConfig {
	return bo.config.Auth
}

// ResourceBuilder provides fluent API for resource configuration
type ResourceBuilder struct {
	backoffice *BackOffice
	resource   *Resource
}

// WithName sets a custom display name for the resource
func (rb *ResourceBuilder) WithName(name string) *ResourceBuilder {
	rb.resource.DisplayName = name
	return rb
}

// WithPluralName sets a custom plural name for the resource
func (rb *ResourceBuilder) WithPluralName(name string) *ResourceBuilder {
	rb.resource.PluralName = name
	return rb
}

// WithField configures a specific field
func (rb *ResourceBuilder) WithField(fieldName string, config func(*FieldBuilder)) *ResourceBuilder {
	builder := NewFieldBuilder()
	config(builder)
	rb.resource.FieldConfigs[fieldName] = builder.Build()

	// Track field registration order
	rb.resource.FieldOrder = append(rb.resource.FieldOrder, fieldName)

	// Re-discover fields to apply the configuration
	rb.resource.DiscoverFields()

	return rb
}

// WithDerivedField adds a derived field that calculates its value dynamically from fetched data
// Supports optional field configuration functions for sorting, display settings, etc.
func (rb *ResourceBuilder) WithDerivedField(fieldName, displayName string, computeFunc ComputeFunc, configFuncs ...func(*FieldBuilder)) *ResourceBuilder {
	builder := NewFieldBuilder()
	builder.DisplayName(displayName).ReadOnly(true) // Derived fields are read-only by default
	builder.config.IsComputed = true
	builder.config.ComputeFunc = computeFunc

	// Apply optional configurations
	for _, configFunc := range configFuncs {
		configFunc(builder)
	}

	rb.resource.FieldConfigs[fieldName] = builder.Build()

	// Track field registration order
	rb.resource.FieldOrder = append(rb.resource.FieldOrder, fieldName)

	// Re-discover fields to apply the configuration
	rb.resource.DiscoverFields()

	return rb
}

// Hidden sets whether the resource should be hidden from the admin panel
func (rb *ResourceBuilder) Hidden(hidden bool) *ResourceBuilder {
	rb.resource.Hidden = hidden
	return rb
}

// ReadOnly sets whether the resource should be read-only
func (rb *ResourceBuilder) ReadOnly(readOnly bool) *ResourceBuilder {
	rb.resource.ReadOnly = readOnly
	return rb
}

// WithDefaultSort sets the default sorting for the resource
func (rb *ResourceBuilder) WithDefaultSort(field string, direction SortDirection) *ResourceBuilder {
	rb.resource.DefaultSort = SortField{
		Field:      field,
		Direction:  direction,
		Precedence: SortPrecedenceExplicit,
	}
	return rb
}

// WithManyToOneField configures a many-to-one relationship field
func (rb *ResourceBuilder) WithManyToOneField(fieldName string, relatedModel string, options func(*RelationshipBuilder)) *ResourceBuilder {
	relationshipBuilder := &RelationshipBuilder{
		info: &RelationshipInfo{
			Type:           RelationshipManyToOne,
			RelatedModel:   relatedModel,
			DisplayField:   "Name",    // Default display field
			DisplayPattern: "compact", // Default display pattern
		},
	}

	if options != nil {
		options(relationshipBuilder)
	}

	// Configure the field with relationship info
	rb.WithField(fieldName, func(fb *FieldBuilder) {
		fb.config.Relationship = relationshipBuilder.info
	})

	return rb
}

// RelationshipBuilder provides fluent API for configuring relationships
type RelationshipBuilder struct {
	info *RelationshipInfo
}

// DisplayField sets which field from the related model to display
func (rb *RelationshipBuilder) DisplayField(fieldName string) *RelationshipBuilder {
	rb.info.DisplayField = fieldName
	return rb
}

// ForeignKey sets the foreign key field name
func (rb *RelationshipBuilder) ForeignKey(fkName string) *RelationshipBuilder {
	rb.info.ForeignKey = fkName
	return rb
}

// CompactDisplay sets the relationship to use compact display pattern
func (rb *RelationshipBuilder) CompactDisplay() *RelationshipBuilder {
	rb.info.DisplayPattern = "compact"
	return rb
}

// BadgeDisplay sets the relationship to use badge display pattern
func (rb *RelationshipBuilder) BadgeDisplay() *RelationshipBuilder {
	rb.info.DisplayPattern = "badge"
	return rb
}

// HierarchicalDisplay sets the relationship to use hierarchical display pattern
func (rb *RelationshipBuilder) HierarchicalDisplay() *RelationshipBuilder {
	rb.info.DisplayPattern = "hierarchical"
	return rb
}

// InlineEditor sets the relationship to use inline editor in detail views
func (rb *RelationshipBuilder) InlineEditor() *RelationshipBuilder {
	rb.info.DisplayPattern = "inline"
	return rb
}

// CardDisplay sets the relationship to use card-based display in detail views
func (rb *RelationshipBuilder) CardDisplay() *RelationshipBuilder {
	rb.info.DisplayPattern = "card"
	return rb
}

// SidebarDisplay sets the relationship to use sidebar summary display (default)
func (rb *RelationshipBuilder) SidebarDisplay() *RelationshipBuilder {
	rb.info.DisplayPattern = "sidebar"
	return rb
}

// Helper functions for generating names
func generateDisplayName(name string) string {
	// Convert CamelCase to "Display Name"
	result := ""
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result += " "
		}
		result += string(r)
	}
	return result
}

func generatePluralName(name string) string {
	displayName := generateDisplayName(name)
	return pluralize(displayName)
}

func generateTableName(name string) string {
	// Convert to snake_case and pluralize
	snake := strcase.ToSnake(name)
	return pluralize(snake)
}

// Basic pluralization - can be enhanced later
func pluralize(word string) string {
	if strings.HasSuffix(word, "y") {
		return strings.TrimSuffix(word, "y") + "ies"
	}
	if strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") ||
		strings.HasSuffix(word, "z") || strings.HasSuffix(word, "ch") ||
		strings.HasSuffix(word, "sh") {
		return word + "es"
	}
	return word + "s"
}
