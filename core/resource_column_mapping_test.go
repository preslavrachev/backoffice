package core

import (
	"context"
	"testing"

	"backoffice/middleware/auth"

	"github.com/iancoleman/strcase"
)

// Test structs for different column mapping scenarios
type TestUserWithDBTags struct {
	ID        uint   `db:"id"`
	Name      string `db:"full_name"`
	Email     string `db:"email_address"`
	CreatedAt string `db:"created_at"`
	Excluded  string `db:"-"`
}

type TestUserWithGormTags struct {
	ID        uint   `gorm:"column:id;primaryKey"`
	Name      string `gorm:"column:user_full_name;not null"`
	Email     string `gorm:"column:user_email"`
	CreatedAt string `gorm:"column:dt_created"`
}

type TestUserWithJSONTags struct {
	ID        uint   `json:"id"`
	Name      string `json:"full_name"`
	Email     string `json:"email_address,omitempty"`
	CreatedAt string `json:"created_at"`
}

type TestUserNoTags struct {
	ID        uint
	Name      string
	Email     string
	CreatedAt string
}

type TestUserMixedTags struct {
	ID        uint   `db:"id"`
	Name      string `gorm:"column:user_name"`
	Email     string `json:"email_addr"`
	CreatedAt string
}

// Mock adapter for testing
type mockAdapter struct{}

func (m *mockAdapter) Find(ctx context.Context, resource *Resource, query *Query) (*Result, error) {
	return nil, nil
}

func (m *mockAdapter) GetAll(ctx context.Context, resource *Resource, filters map[string]any) ([]any, error) {
	return nil, nil
}

func (m *mockAdapter) GetByID(ctx context.Context, resource *Resource, id any) (any, error) {
	return nil, nil
}

func (m *mockAdapter) Create(ctx context.Context, resource *Resource, data any) error {
	return nil
}

func (m *mockAdapter) Update(ctx context.Context, resource *Resource, id any, data any) error {
	return nil
}

func (m *mockAdapter) Delete(ctx context.Context, resource *Resource, id any) error {
	return nil
}
func (m *mockAdapter) GetSchema(resource *Resource) (*Schema, error)   { return nil, nil }
func (m *mockAdapter) ValidateData(resource *Resource, data any) error { return nil }
func (m *mockAdapter) Count(ctx context.Context, resource *Resource, filters map[string]any) (int64, error) {
	return 0, nil
}

func (m *mockAdapter) Search(ctx context.Context, resource *Resource, query string) ([]any, error) {
	return nil, nil
}

func setupBackOffice() *BackOffice {
	return New(&mockAdapter{}, auth.WithNoAuth())
}

func TestResourceBuilder_GetColumnName(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(*BackOffice) *Resource
		fieldName   string
		expected    string
		description string
	}{
		// DB tag tests
		{
			name: "db_tag_simple",
			setupFunc: func(bo *BackOffice) *Resource {
				return bo.RegisterResource(&TestUserWithDBTags{}).
					WithField("Name", func(f *FieldBuilder) {
						f.DisplayName("Full Name")
					}).resource
			},
			fieldName:   "Name",
			expected:    "full_name",
			description: "should use db tag",
		},
		{
			name: "db_tag_excluded",
			setupFunc: func(bo *BackOffice) *Resource {
				return bo.RegisterResource(&TestUserWithDBTags{}).
					WithField("Excluded", func(f *FieldBuilder) {
						f.DisplayName("Excluded Field")
					}).resource
			},
			fieldName:   "Excluded",
			expected:    "excluded",
			description: "db:'-' should fall back to snake_case",
		},

		// GORM tag tests
		{
			name: "gorm_tag_simple",
			setupFunc: func(bo *BackOffice) *Resource {
				return bo.RegisterResource(&TestUserWithGormTags{}).
					WithField("Name", func(f *FieldBuilder) {
						f.DisplayName("User Name")
					}).resource
			},
			fieldName:   "Name",
			expected:    "user_full_name",
			description: "should parse gorm column tag",
		},
		{
			name: "gorm_tag_with_options",
			setupFunc: func(bo *BackOffice) *Resource {
				return bo.RegisterResource(&TestUserWithGormTags{}).
					WithField("ID", func(f *FieldBuilder) {
						f.DisplayName("ID")
					}).resource
			},
			fieldName:   "ID",
			expected:    "id",
			description: "should extract column from gorm tag with options",
		},

		// JSON tag tests
		{
			name: "json_tag_simple",
			setupFunc: func(bo *BackOffice) *Resource {
				return bo.RegisterResource(&TestUserWithJSONTags{}).
					WithField("Name", func(f *FieldBuilder) {
						f.DisplayName("Name")
					}).resource
			},
			fieldName:   "Name",
			expected:    "full_name",
			description: "should use json tag",
		},
		{
			name: "json_tag_with_omitempty",
			setupFunc: func(bo *BackOffice) *Resource {
				return bo.RegisterResource(&TestUserWithJSONTags{}).
					WithField("Email", func(f *FieldBuilder) {
						f.DisplayName("Email")
					}).resource
			},
			fieldName:   "Email",
			expected:    "email_address",
			description: "should extract field name before omitempty",
		},

		// Snake case fallback tests
		{
			name: "snake_case_fallback",
			setupFunc: func(bo *BackOffice) *Resource {
				return bo.RegisterResource(&TestUserNoTags{}).
					WithField("CreatedAt", func(f *FieldBuilder) {
						f.DisplayName("Created At")
					}).resource
			},
			fieldName:   "CreatedAt",
			expected:    "created_at",
			description: "should convert to snake_case when no tags",
		},

		// Priority order tests
		{
			name: "priority_db_over_gorm",
			setupFunc: func(bo *BackOffice) *Resource {
				return bo.RegisterResource(&TestUserMixedTags{}).
					WithField("ID", func(f *FieldBuilder) {
						f.DisplayName("ID")
					}).resource
			},
			fieldName:   "ID",
			expected:    "id",
			description: "db tag should win over other tags",
		},
		{
			name: "priority_gorm_over_json",
			setupFunc: func(bo *BackOffice) *Resource {
				return bo.RegisterResource(&TestUserMixedTags{}).
					WithField("Name", func(f *FieldBuilder) {
						f.DisplayName("Name")
					}).resource
			},
			fieldName:   "Name",
			expected:    "user_name",
			description: "gorm tag should win over json tag",
		},
		{
			name: "priority_json_over_snake_case",
			setupFunc: func(bo *BackOffice) *Resource {
				return bo.RegisterResource(&TestUserMixedTags{}).
					WithField("Email", func(f *FieldBuilder) {
						f.DisplayName("Email")
					}).resource
			},
			fieldName:   "Email",
			expected:    "email_addr",
			description: "json tag should win over snake_case",
		},

		// Explicit override tests (highest priority)
		{
			name: "explicit_override_beats_db_tag",
			setupFunc: func(bo *BackOffice) *Resource {
				return bo.RegisterResource(&TestUserWithDBTags{}).
					WithField("Name", func(f *FieldBuilder) {
						f.DisplayName("Full Name").WithDBColumnName("custom_column_override")
					}).resource
			},
			fieldName:   "Name",
			expected:    "custom_column_override",
			description: "explicit DBColumnName should override db tag",
		},
		{
			name: "explicit_override_beats_gorm_tag",
			setupFunc: func(bo *BackOffice) *Resource {
				return bo.RegisterResource(&TestUserWithGormTags{}).
					WithField("Name", func(f *FieldBuilder) {
						f.DisplayName("Name").WithDBColumnName("explicit_name")
					}).resource
			},
			fieldName:   "Name",
			expected:    "explicit_name",
			description: "explicit DBColumnName should override gorm tag",
		},

		// Complex scenarios
		{
			name: "legacy_database_mapping",
			setupFunc: func(bo *BackOffice) *Resource {
				return bo.RegisterResource(&TestUserNoTags{}).
					WithField("CreatedAt", func(f *FieldBuilder) {
						f.DisplayName("Creation Date").WithDBColumnName("dt_created")
					}).
					WithField("Name", func(f *FieldBuilder) {
						f.DisplayName("User Name").WithDBColumnName("user_full_name")
					}).resource
			},
			fieldName:   "CreatedAt",
			expected:    "dt_created",
			description: "should handle legacy database column names",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bo := setupBackOffice()
			resource := tt.setupFunc(bo)

			result := resource.GetColumnName(tt.fieldName)
			if result != tt.expected {
				t.Errorf("GetColumnName(%s) = %s, expected %s (%s)",
					tt.fieldName, result, tt.expected, tt.description)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Fixed critical cases
		{"ID", "id"},
		{"API", "api"},
		{"URL", "url"},
		{"HTTP", "http"},
		{"HTTPS", "https"},
		{"XML", "xml"},
		{"JSON", "json"},
		{"SQL", "sql"},
		{"UUID", "uuid"},
		{"IO", "io"},
		{"OS", "os"},

		// Basic cases
		{"Name", "name"},
		{"CreatedAt", "created_at"},
		{"UserName", "user_name"},
		{"BlogPost", "blog_post"},

		// Smart acronym handling
		{"XMLHttpRequest", "xml_http_request"},
		{"HTTPSConnection", "https_connection"},
		{"IOError", "io_error"},
		{"APIKey", "api_key"},
		{"URLPath", "url_path"},
		{"JSONData", "json_data"},
		{"SQLQuery", "sql_query"},

		// Mixed case scenarios
		{"iPhone", "i_phone"},
		{"macOS", "mac_os"},
		{"IPv4Address", "i_pv_4_address"},
		{"XMLParser", "xml_parser"},

		// Edge cases
		{"", ""},
		{"A", "a"},
		{"AB", "ab"},
		{"ABc", "a_bc"},
		{"ABC", "abc"},
		{"AbC", "ab_c"},
		{"lowercase", "lowercase"},
		{"UPPERCASE", "uppercase"},

		// Numbers and special chars
		{"User123", "user_123"},
		{"HTTP2Protocol", "http_2_protocol"},
		{"OAuth2Token", "o_auth_2_token"},

		// Complex real-world examples
		{"HTTPSAPIKeyValidator", "httpsapi_key_validator"},
		{"XMLDocumentParser", "xml_document_parser"},
		{"JSONWebTokenAuth", "json_web_token_auth"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := strcase.ToSnake(tt.input)
			if result != tt.expected {
				t.Errorf("ToSnakeCase(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestResourceBuilder_WithDBColumnName_Integration(t *testing.T) {
	// Test the complete integration with WithDBColumnName
	bo := setupBackOffice()

	resource := bo.RegisterResource(&TestUserWithDBTags{}).
		WithField("Name", func(f *FieldBuilder) {
			f.DisplayName("Customer Name").
				Required(true).
				Searchable(true).
				WithDBColumnName("customer_full_name")
		}).
		WithField("Email", func(f *FieldBuilder) {
			f.DisplayName("Email Address").
				Required(true).
				WithDBColumnName("customer_email")
		}).resource

	tests := []struct {
		fieldName string
		expected  string
	}{
		{"Name", "customer_full_name"}, // explicit override
		{"Email", "customer_email"},    // explicit override
		{"CreatedAt", "created_at"},    // db tag (not configured)
		{"ID", "id"},                   // db tag (primary key, auto-detected)
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			result := resource.GetColumnName(tt.fieldName)
			if result != tt.expected {
				t.Errorf("GetColumnName(%s) = %s, expected %s", tt.fieldName, result, tt.expected)
			}
		})
	}
}

func TestResourceBuilder_NonExistentField(t *testing.T) {
	bo := setupBackOffice()
	resource := bo.RegisterResource(&TestUserWithDBTags{}).
		WithField("Name", func(f *FieldBuilder) {
			f.DisplayName("Name")
		}).resource

	// Non-existent field should fall back to snake_case
	result := resource.GetColumnName("NonExistentField")
	expected := "non_existent_field"

	if result != expected {
		t.Errorf("GetColumnName(NonExistentField) = %s, expected %s", result, expected)
	}
}
