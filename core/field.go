package core

// RelationshipType defines the type of relationship
type RelationshipType string

const (
	RelationshipNone       RelationshipType = "none"
	RelationshipManyToOne  RelationshipType = "many_to_one"
	RelationshipOneToMany  RelationshipType = "one_to_many"
	RelationshipManyToMany RelationshipType = "many_to_many"
)

// FieldRenderer defines how a field should be rendered in list views
type FieldRenderer string

const (
	RenderText     FieldRenderer = "text"     // Plain text, truncate if needed
	RenderHTML     FieldRenderer = "html"     // Strip HTML tags, show preview
	RenderRichText FieldRenderer = "richtext" // Show formatted preview
	RenderMarkdown FieldRenderer = "markdown" // Render markdown preview
)

// ComputeFunc is a function type for computing field values dynamically
type ComputeFunc func(any) string

// RelationshipInfo holds metadata about field relationships
type RelationshipInfo struct {
	Type           RelationshipType `json:"type"`
	RelatedModel   string           `json:"related_model"`
	DisplayField   string           `json:"display_field"`
	ForeignKey     string           `json:"foreign_key"`
	DisplayPattern string           `json:"display_pattern"` // "compact", "badge", "hierarchical"
}

// FieldInfo represents metadata about a struct field
type FieldInfo struct {
	Name             string            `json:"name"`
	Type             string            `json:"type"`
	JSONName         string            `json:"json_name"`
	DisplayName      string            `json:"display_name"`
	DBColumnName     string            `json:"db_column_name,omitempty"`
	Required         bool              `json:"required"`
	ReadOnly         bool              `json:"read_only"`
	Searchable       bool              `json:"searchable"`
	Unique           bool              `json:"unique"`
	PrimaryKey       bool              `json:"primary_key"`
	Choices          []string          `json:"choices,omitempty"`
	DefaultVal       any               `json:"default_value,omitempty"`
	Relationship     *RelationshipInfo `json:"relationship,omitempty"`
	IsComputed       bool              `json:"is_computed"`
	ComputeFunc      ComputeFunc       `json:"-"`
	SortFields       []SortField       `json:"sort_fields,omitempty"`
	IsSortable       bool              `json:"is_sortable"`
	RenderAs         FieldRenderer     `json:"render_as,omitempty"`
	MaxPreviewLength int               `json:"max_preview_length,omitempty"`
}

// FieldConfig holds configuration for a field
type FieldConfig struct {
	DisplayName      string
	DBColumnName     string
	Required         bool
	ReadOnly         bool
	Searchable       bool
	Unique           bool
	PrimaryKey       bool
	Choices          []string
	DefaultVal       any
	Relationship     *RelationshipInfo
	IsComputed       bool
	ComputeFunc      ComputeFunc
	SortFields       []SortField `json:"sort_fields,omitempty"`
	IsSortable       bool        `json:"is_sortable"`
	RenderAs         FieldRenderer
	MaxPreviewLength int
}

// Apply applies the configuration to a FieldInfo
func (fc *FieldConfig) Apply(info *FieldInfo) {
	if fc.DisplayName != "" {
		info.DisplayName = fc.DisplayName
	}
	if fc.DBColumnName != "" {
		info.DBColumnName = fc.DBColumnName
	}
	info.Required = fc.Required
	info.ReadOnly = fc.ReadOnly
	info.Searchable = fc.Searchable
	info.Unique = fc.Unique
	info.PrimaryKey = fc.PrimaryKey
	if len(fc.Choices) > 0 {
		info.Choices = fc.Choices
	}
	if fc.DefaultVal != nil {
		info.DefaultVal = fc.DefaultVal
	}
	if fc.Relationship != nil {
		info.Relationship = fc.Relationship
	}
	info.IsComputed = fc.IsComputed
	info.ComputeFunc = fc.ComputeFunc
	if len(fc.SortFields) > 0 {
		info.SortFields = fc.SortFields
	}
	info.IsSortable = fc.IsSortable
	if fc.RenderAs != "" {
		info.RenderAs = fc.RenderAs
	}
	if fc.MaxPreviewLength > 0 {
		info.MaxPreviewLength = fc.MaxPreviewLength
	}
}

// FieldBuilder provides fluent API for configuring fields
type FieldBuilder struct {
	config *FieldConfig
}

// NewFieldBuilder creates a new FieldBuilder
func NewFieldBuilder() *FieldBuilder {
	return &FieldBuilder{
		config: &FieldConfig{},
	}
}

// DisplayName sets the display name for the field
func (fb *FieldBuilder) DisplayName(name string) *FieldBuilder {
	fb.config.DisplayName = name
	return fb
}

// WithDBColumnName sets the database column name for the field
func (fb *FieldBuilder) WithDBColumnName(columnName string) *FieldBuilder {
	fb.config.DBColumnName = columnName
	return fb
}

// Required marks the field as required
func (fb *FieldBuilder) Required(required bool) *FieldBuilder {
	fb.config.Required = required
	return fb
}

// ReadOnly marks the field as read-only
func (fb *FieldBuilder) ReadOnly(readOnly bool) *FieldBuilder {
	fb.config.ReadOnly = readOnly
	return fb
}

// Searchable marks the field as searchable
func (fb *FieldBuilder) Searchable(searchable bool) *FieldBuilder {
	fb.config.Searchable = searchable
	return fb
}

// Unique marks the field as unique
func (fb *FieldBuilder) Unique(unique bool) *FieldBuilder {
	fb.config.Unique = unique
	return fb
}

// PrimaryKey marks the field as a primary key
func (fb *FieldBuilder) PrimaryKey(pk bool) *FieldBuilder {
	fb.config.PrimaryKey = pk
	return fb
}

// Choices sets available choices for the field
func (fb *FieldBuilder) Choices(choices []string) *FieldBuilder {
	fb.config.Choices = choices
	return fb
}

// Default sets the default value for the field
func (fb *FieldBuilder) Default(value any) *FieldBuilder {
	fb.config.DefaultVal = value
	return fb
}

// ManyToOne configures the field as a many-to-one relationship
func (fb *FieldBuilder) ManyToOne(relatedModel string, displayField string) *FieldBuilder {
	fb.config.Relationship = &RelationshipInfo{
		Type:         RelationshipManyToOne,
		RelatedModel: relatedModel,
		DisplayField: displayField,
	}
	return fb
}

// DisplayPattern sets the display pattern for relationship fields
func (fb *FieldBuilder) DisplayPattern(pattern string) *FieldBuilder {
	if fb.config.Relationship == nil {
		fb.config.Relationship = &RelationshipInfo{}
	}
	fb.config.Relationship.DisplayPattern = pattern
	return fb
}

// ForeignKey sets the foreign key field name for relationships
func (fb *FieldBuilder) ForeignKey(fk string) *FieldBuilder {
	if fb.config.Relationship == nil {
		fb.config.Relationship = &RelationshipInfo{}
	}
	fb.config.Relationship.ForeignKey = fk
	return fb
}

// SortBy adds a sort field configuration (additive - can be called multiple times)
func (fb *FieldBuilder) SortBy(fieldName string, direction SortDirection) *FieldBuilder {
	if fb.config.SortFields == nil {
		fb.config.SortFields = []SortField{}
	}
	fb.config.SortFields = append(fb.config.SortFields, SortField{
		Field:     fieldName, // Admin field name, not DB column
		Direction: direction,
	})
	fb.config.IsSortable = true
	return fb
}

// RenderAsHTML configures the field to strip HTML tags and show preview
func (fb *FieldBuilder) RenderAsHTML() *FieldBuilder {
	fb.config.RenderAs = RenderHTML
	return fb
}

// RenderAsRichText configures the field to show formatted preview
func (fb *FieldBuilder) RenderAsRichText() *FieldBuilder {
	fb.config.RenderAs = RenderRichText
	return fb
}

// RenderAsMarkdown configures the field to render markdown preview
func (fb *FieldBuilder) RenderAsMarkdown() *FieldBuilder {
	fb.config.RenderAs = RenderMarkdown
	return fb
}

// MaxPreviewLength sets the maximum length for field preview in list views
func (fb *FieldBuilder) MaxPreviewLength(length int) *FieldBuilder {
	fb.config.MaxPreviewLength = length
	return fb
}

// Build returns the final FieldConfig
func (fb *FieldBuilder) Build() *FieldConfig {
	return fb.config
}
