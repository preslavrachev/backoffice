package core

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/iancoleman/strcase"
)

// Resource represents a registered resource with its metadata
type Resource struct {
	Name         string                  `json:"name"`
	DisplayName  string                  `json:"display_name"`
	PluralName   string                  `json:"plural_name"`
	Model        any                     `json:"-"`
	ModelType    reflect.Type            `json:"-"`
	Fields       []FieldInfo             `json:"fields"`
	PrimaryKey   string                  `json:"primary_key"`
	IDField      string                  `json:"id_field"`
	TableName    string                  `json:"table_name"`
	Hidden       bool                    `json:"hidden"`
	ReadOnly     bool                    `json:"read_only"`
	FieldConfigs map[string]*FieldConfig `json:"-"`
	FieldOrder   []string                `json:"-"`            // Track order of field registration
	DefaultSort  SortField               `json:"default_sort"` // Default sorting configuration
	Actions      []CustomAction          `json:"-"`            // Custom actions for this resource
}

// ResourceMeta contains basic metadata for templates
type ResourceMeta struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	PluralName  string `json:"plural_name"`
	Hidden      bool   `json:"hidden"`
	ReadOnly    bool   `json:"read_only"`
}

// GetMeta returns basic metadata for templates
func (r *Resource) GetMeta() ResourceMeta {
	return ResourceMeta{
		Name:        r.Name,
		DisplayName: r.DisplayName,
		PluralName:  r.PluralName,
		Hidden:      r.Hidden,
		ReadOnly:    r.ReadOnly,
	}
}

// GetEffectiveDefaultSort returns the effective default sort for this resource
// following the precedence hierarchy: Explicit > CreatedAt > ID
func (r *Resource) GetEffectiveDefaultSort() SortField {
	// Priority 1: Use explicitly configured default sort
	if r.DefaultSort.Precedence == SortPrecedenceExplicit {
		return r.DefaultSort
	}

	// Priority 2: Check if the model has a CreatedAt field
	if hasCreatedAtField(r) {
		// Use the actual field name from the resource
		for _, field := range r.Fields {
			fieldName := strings.ToLower(field.Name)
			if fieldName == "createdat" || fieldName == "created_at" {
				return SortField{
					Field:      field.Name,
					Direction:  SortDesc,
					Precedence: SortPrecedenceAutoCreatedAt,
				}
			}
		}
	}

	// Priority 3: Fallback to ID field
	return SortField{
		Field:      r.IDField,
		Direction:  SortAsc,
		Precedence: SortPrecedenceAutoID,
	}
}

// DiscoverFields extracts field information from the struct using reflection
func (r *Resource) DiscoverFields() error {
	t := r.ModelType
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	r.Fields = make([]FieldInfo, 0)

	// First, discover primary key field (always needed for CRUD operations)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Check if this is a primary key field - always include it
		if isPrimaryKeyField(field) {
			fieldInfo := FieldInfo{
				Name:        field.Name,
				Type:        field.Type.String(),
				JSONName:    getJSONTag(field),
				DisplayName: field.Name,
				Required:    false,
				ReadOnly:    true, // Primary keys are typically read-only
				Searchable:  false,
				Unique:      true,
				PrimaryKey:  true,
				IsComputed:  false,
				ComputeFunc: nil,
			}

			r.PrimaryKey = field.Name
			r.IDField = field.Name
			r.Fields = append(r.Fields, fieldInfo)
			break
		}
	}

	// Second, process only explicitly configured fields in registration order
	for _, fieldName := range r.FieldOrder {
		// Skip if this field is already added (primary key)
		var alreadyExists bool
		for _, existingField := range r.Fields {
			if existingField.Name == fieldName {
				alreadyExists = true
				break
			}
		}
		if alreadyExists {
			continue
		}

		config, exists := r.FieldConfigs[fieldName]
		if !exists {
			continue // Shouldn't happen, but be safe
		}

		var fieldInfo FieldInfo

		if config.IsComputed {
			// Computed field
			fieldInfo = FieldInfo{
				Name:        fieldName,
				Type:        "string", // Computed fields return strings
				JSONName:    fieldName,
				DisplayName: fieldName,
				Required:    false,
				ReadOnly:    true, // Computed fields are read-only
				Searchable:  false,
				Unique:      false,
				PrimaryKey:  false,
				IsComputed:  true,
				ComputeFunc: config.ComputeFunc,
			}
		} else {
			// Find the corresponding struct field
			var structField *reflect.StructField
			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)
				if field.Name == fieldName {
					structField = &field
					break
				}
			}

			if structField == nil {
				return fmt.Errorf("configured field %s not found in struct %s", fieldName, t.Name())
			}

			// Create field info from struct field
			fieldInfo = FieldInfo{
				Name:        structField.Name,
				Type:        structField.Type.String(),
				JSONName:    getJSONTag(*structField),
				DisplayName: structField.Name,
				Required:    false,
				ReadOnly:    false,
				Searchable:  false,
				Unique:      false,
				PrimaryKey:  false,
				IsComputed:  false,
				ComputeFunc: nil,
			}

			// Auto-detect relationships for explicitly configured fields
			if relInfo := detectRelationship(*structField, t); relInfo != nil {
				fieldInfo.Relationship = relInfo
			}
		}

		// Apply user configurations
		config.Apply(&fieldInfo)
		r.Fields = append(r.Fields, fieldInfo)
	}

	return nil
}

// Helper function to get JSON tag
func getJSONTag(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}
	// Handle json:",omitempty" cases
	if tag == "-" {
		return ""
	}
	// Extract just the field name part before any options
	if idx := len(tag); idx > 0 {
		for i, r := range tag {
			if r == ',' {
				return tag[:i]
			}
		}
		return tag
	}
	return field.Name
}

// Helper function to detect relationships based on field structure
func detectRelationship(field reflect.StructField, structType reflect.Type) *RelationshipInfo {
	fieldType := field.Type
	fieldName := field.Name

	// Check for many-to-one relationships
	// Pattern 1: Field name ends with "ID" and there's a corresponding struct field
	if strings.HasSuffix(fieldName, "ID") && isNumericType(fieldType) {
		// Look for corresponding relationship field (remove ID suffix)
		relatedFieldName := strings.TrimSuffix(fieldName, "ID")
		if relatedField, hasRelatedField := structType.FieldByName(relatedFieldName); hasRelatedField {
			// Check if the related field is a pointer to a struct
			if relatedField.Type.Kind() == reflect.Ptr && relatedField.Type.Elem().Kind() == reflect.Struct {
				return &RelationshipInfo{
					Type:           RelationshipManyToOne,
					RelatedModel:   relatedField.Type.Elem().Name(),
					DisplayField:   "Name", // Default display field
					ForeignKey:     fieldName,
					DisplayPattern: "compact", // Default display pattern
				}
			}
		}
	}

	// Pattern 2: Field is a pointer to a struct (database association)
	if fieldType.Kind() == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct {
		// Check if there's a corresponding ID field
		foreignKeyName := fieldName + "ID"
		if _, hasForeignKey := structType.FieldByName(foreignKeyName); hasForeignKey {
			return &RelationshipInfo{
				Type:           RelationshipManyToOne,
				RelatedModel:   fieldType.Elem().Name(),
				DisplayField:   "Name", // Default display field
				ForeignKey:     foreignKeyName,
				DisplayPattern: "compact", // Default display pattern
			}
		}
	}

	return nil
}

// Helper function to check if field is a primary key
func isPrimaryKeyField(field reflect.StructField) bool {
	// Check db tag for primary key indicators
	dbTag := field.Tag.Get("db")
	if dbTag != "" && (dbTag == "id" || strings.Contains(dbTag, "primary")) {
		return true
	}

	// Common primary key field names
	return field.Name == "ID" && field.Type.String() == "uint"
}

// Helper function to check if a type is numeric (suitable for foreign keys)
func isNumericType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	}
	return false
}

// GetFieldValue extracts field value from a struct using reflection
func GetFieldValue(item any, fieldName string) any {
	if item == nil {
		return nil
	}

	val := reflect.ValueOf(item)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	field := val.FieldByName(fieldName)
	if !field.IsValid() {
		return nil
	}

	return field.Interface()
}

// GetFieldValueWithResource extracts field value, handling computed fields
func GetFieldValueWithResource(item any, fieldInfo *FieldInfo, resource *Resource) any {
	if item == nil {
		return nil
	}

	// Handle computed fields
	if fieldInfo.IsComputed && fieldInfo.ComputeFunc != nil {
		return fieldInfo.ComputeFunc(item)
	}

	// Regular field extraction
	return GetFieldValue(item, fieldInfo.Name)
}

// FormatFieldValueForDisplay formats field values for display in UI templates
// Handles special cases like slices (shows count) and relationships
// stripHTMLTags removes HTML tags from a string and returns plain text
func stripHTMLTags(html string) string {
	// Simple regex-based HTML tag removal
	// This handles most common HTML cases without external dependencies
	result := html

	// Replace block-level tags with space to preserve sentence boundaries
	blockTags := []string{"</p>", "</div>", "</h1>", "</h2>", "</h3>", "</h4>", "</h5>", "</h6>", "</li>", "</tr>", "</td>", "<br>", "<br/>"}
	for _, tag := range blockTags {
		result = strings.ReplaceAll(result, tag, " ")
	}

	// Remove all HTML tags
	tagPattern := `<[^>]*>`
	result = regexp.MustCompile(tagPattern).ReplaceAllString(result, "")

	// Decode common HTML entities
	result = strings.ReplaceAll(result, "&nbsp;", " ")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&quot;", "\"")
	result = strings.ReplaceAll(result, "&#39;", "'")

	// Collapse multiple spaces and trim
	spacePattern := `\s+`
	result = regexp.MustCompile(spacePattern).ReplaceAllString(result, " ")
	result = strings.TrimSpace(result)

	return result
}

// truncateText truncates text to maxLength and adds ellipsis if needed
func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	// Try to break at word boundary
	truncated := text[:maxLength]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLength/2 {
		truncated = text[:lastSpace]
	}

	return truncated + "..."
}

func FormatFieldValueForDisplay(item any, field *FieldInfo) string {
	// Handle computed fields
	if field.IsComputed && field.ComputeFunc != nil {
		return field.ComputeFunc(item)
	}

	value := GetFieldValue(item, field.Name)
	if value == nil {
		return ""
	}

	// Handle slice/array fields - show count instead of raw slice
	reflectVal := reflect.ValueOf(value)
	if reflectVal.Kind() == reflect.Slice || reflectVal.Kind() == reflect.Array {
		count := reflectVal.Len()
		if count == 0 {
			return "0"
		}
		return fmt.Sprintf("%d", count)
	}

	// Handle pointer fields that might be nil
	if reflectVal.Kind() == reflect.Ptr {
		if reflectVal.IsNil() {
			return ""
		}
		// For non-nil pointers, get the underlying value
		value = reflectVal.Elem().Interface()
	}

	// Convert to string
	strValue := fmt.Sprintf("%v", value)

	// Check if field should render as HTML preview
	if field.RenderAs == RenderHTML || field.RenderAs == RenderRichText {
		strValue = stripHTMLTags(strValue)
	}

	// Apply max preview length if configured
	if field.MaxPreviewLength > 0 {
		strValue = truncateText(strValue, field.MaxPreviewLength)
	}

	return strValue
}

// FormatFieldValueForDisplayWithResource formats field values for display in UI templates
// Using resource context for better handling of computed fields
func FormatFieldValueForDisplayWithResource(item any, field *FieldInfo, resource *Resource) string {
	value := GetFieldValueWithResource(item, field, resource)
	if value == nil {
		return ""
	}

	// For computed fields, the value is already a string
	if field.IsComputed {
		if str, ok := value.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", value)
	}

	// Handle slice/array fields - show count instead of raw slice
	reflectVal := reflect.ValueOf(value)
	if reflectVal.Kind() == reflect.Slice || reflectVal.Kind() == reflect.Array {
		count := reflectVal.Len()
		if count == 0 {
			return "0"
		}
		return fmt.Sprintf("%d", count)
	}

	// Handle pointer fields that might be nil
	if reflectVal.Kind() == reflect.Ptr {
		if reflectVal.IsNil() {
			return ""
		}
		// For non-nil pointers, get the underlying value
		value = reflectVal.Elem().Interface()
	}

	return fmt.Sprintf("%v", value)
}

// hasCreatedAtField checks if the resource has a CreatedAt field
func hasCreatedAtField(resource *Resource) bool {
	for _, field := range resource.Fields {
		fieldName := strings.ToLower(field.Name)
		if fieldName == "createdat" || fieldName == "created_at" {
			return true
		}
	}
	return false
}

// GetFieldSortConfiguration returns the sort configuration for a field if it has one
// Returns nil if the field has no custom sort configuration
func (r *Resource) GetFieldSortConfiguration(fieldName string) []SortField {
	// Find the field in the resource
	for _, field := range r.Fields {
		if field.Name == fieldName {
			// Check if field has custom sort configuration
			if len(field.SortFields) > 0 {
				return field.SortFields
			}
			// Check if field is a computed/derived field without sort config
			if field.IsComputed && !field.IsSortable {
				return nil // Return nil to indicate sorting should be disabled
			}
			break
		}
	}
	// No custom configuration found, use default behavior
	return nil
}

// IsFieldSortable checks if a field can be sorted
// Returns false for computed/derived fields without explicit sort configuration
func (r *Resource) IsFieldSortable(fieldName string) bool {
	// Find the field in the resource
	for _, field := range r.Fields {
		if field.Name == fieldName {
			// If field has custom sort configuration, it's sortable
			if len(field.SortFields) > 0 {
				return true
			}
			// If field is computed/derived without sort config, not sortable
			if field.IsComputed && !field.IsSortable {
				return false
			}
			// Regular fields are sortable by default
			return true
		}
	}
	// Field not found, assume sortable (default behavior)
	return true
}

// GetColumnName resolves the database column name for a field following priority order:
// 1. Explicit override (DBColumnName in FieldConfig/FieldInfo)
// 2. Struct tag parsing (db, gorm, json)
// 3. Snake_case fallback
func (r *Resource) GetColumnName(fieldName string) string {
	// Find the field in the resource
	for _, field := range r.Fields {
		if field.Name == fieldName {
			// 1. Check for explicit override
			if field.DBColumnName != "" {
				return field.DBColumnName
			}
			break
		}
	}

	// 2. Parse struct tags from the model type
	if columnName := r.parseStructTags(fieldName); columnName != "" {
		return columnName
	}

	// 3. Fallback to snake_case conversion
	return strcase.ToSnake(fieldName)
}

// parseStructTags extracts database column name from struct tags
// Priority order: db -> gorm -> json
func (r *Resource) parseStructTags(fieldName string) string {
	t := r.ModelType
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Find the struct field
	field, exists := t.FieldByName(fieldName)
	if !exists {
		return ""
	}

	// Priority 1: db tag
	if dbTag := field.Tag.Get("db"); dbTag != "" && dbTag != "-" {
		return dbTag
	}

	// Priority 2: gorm tag (format: gorm:"column:name")
	if gormTag := field.Tag.Get("gorm"); gormTag != "" {
		if strings.Contains(gormTag, "column:") {
			parts := strings.Split(gormTag, "column:")
			if len(parts) > 1 {
				// Extract column name, handling comma-separated options
				columnPart := strings.TrimSpace(parts[1])
				if idx := strings.Index(columnPart, ";"); idx != -1 {
					columnPart = columnPart[:idx]
				}
				if idx := strings.Index(columnPart, ","); idx != -1 {
					columnPart = columnPart[:idx]
				}
				return strings.TrimSpace(columnPart)
			}
		}
	}

	// Priority 3: json tag as fallback (extract field name before options)
	if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
		if idx := strings.Index(jsonTag, ","); idx != -1 {
			return jsonTag[:idx]
		}
		return jsonTag
	}

	return ""
}
