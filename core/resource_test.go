package core

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/preslavrachev/backoffice/middleware/auth"
)

// Mock adapter for testing
type MockAdapter struct{}

func (m *MockAdapter) Find(ctx context.Context, resource *Resource, query *Query) (*Result, error) {
	return &Result{Items: []any{}, TotalCount: 0, HasMore: false, Query: *query}, nil
}

func (m *MockAdapter) GetAll(ctx context.Context, resource *Resource, filters map[string]any) ([]any, error) {
	return []any{}, nil
}

func (m *MockAdapter) GetByID(ctx context.Context, resource *Resource, id any) (any, error) {
	return nil, nil
}

func (m *MockAdapter) Create(ctx context.Context, resource *Resource, data any) error {
	return nil
}

func (m *MockAdapter) Update(ctx context.Context, resource *Resource, id any, data any) error {
	return nil
}

func (m *MockAdapter) Delete(ctx context.Context, resource *Resource, id any) error {
	return nil
}

func (m *MockAdapter) GetSchema(resource *Resource) (*Schema, error) {
	return &Schema{}, nil
}

func (m *MockAdapter) ValidateData(resource *Resource, data any) error {
	return nil
}

func (m *MockAdapter) Count(ctx context.Context, resource *Resource, filters map[string]any) (int64, error) {
	return 0, nil
}

func (m *MockAdapter) Search(ctx context.Context, resource *Resource, query string) ([]any, error) {
	return []any{}, nil
}

// Test model types for ParseID testing
type ModelWithUintID struct{ ID uint }
type ModelWithUint8ID struct{ ID uint8 }
type ModelWithUint16ID struct{ ID uint16 }
type ModelWithUint32ID struct{ ID uint32 }
type ModelWithUint64ID struct{ ID uint64 }
type ModelWithIntID struct{ ID int }
type ModelWithInt8ID struct{ ID int8 }
type ModelWithInt16ID struct{ ID int16 }
type ModelWithInt32ID struct{ ID int32 }
type ModelWithInt64ID struct{ ID int64 }
type ModelWithStringID struct{ ID string }
type ModelWithUUIDID struct{ ID uuid.UUID }

// TestParseID_ComprehensiveTypes tests ParseID with all supported ID types
func TestParseID_ComprehensiveTypes(t *testing.T) {
	tests := []struct {
		name         string
		model        any    // Model type to register
		resourceName string // Expected resource name
		input        string // Input string to parse
		wantValue    any    // Expected parsed value
		wantError    bool   // Whether an error is expected
	}{
		// Unsigned integer types
		{
			name:         "uint ID - valid",
			model:        &ModelWithUintID{},
			resourceName: "ModelWithUintID",
			input:        "42",
			wantValue:    uint(42),
			wantError:    false,
		},
		{
			name:         "uint ID - zero",
			model:        &ModelWithUintID{},
			resourceName: "ModelWithUintID",
			input:        "0",
			wantValue:    uint(0),
			wantError:    false,
		},
		{
			name:         "uint ID - invalid negative",
			model:        &ModelWithUintID{},
			resourceName: "ModelWithUintID",
			input:        "-1",
			wantValue:    nil,
			wantError:    true,
		},
		{
			name:         "uint ID - invalid string",
			model:        &ModelWithUintID{},
			resourceName: "ModelWithUintID",
			input:        "not-a-number",
			wantValue:    nil,
			wantError:    true,
		},
		{
			name:         "uint8 ID - valid",
			model:        &ModelWithUint8ID{},
			resourceName: "ModelWithUint8ID",
			input:        "255",
			wantValue:    uint8(255),
			wantError:    false,
		},
		{
			name:         "uint8 ID - overflow",
			model:        &ModelWithUint8ID{},
			resourceName: "ModelWithUint8ID",
			input:        "256",
			wantValue:    nil,
			wantError:    true,
		},
		{
			name:         "uint16 ID - valid",
			model:        &ModelWithUint16ID{},
			resourceName: "ModelWithUint16ID",
			input:        "65535",
			wantValue:    uint16(65535),
			wantError:    false,
		},
		{
			name:         "uint32 ID - valid",
			model:        &ModelWithUint32ID{},
			resourceName: "ModelWithUint32ID",
			input:        "4294967295",
			wantValue:    uint32(4294967295),
			wantError:    false,
		},
		{
			name:         "uint64 ID - valid large",
			model:        &ModelWithUint64ID{},
			resourceName: "ModelWithUint64ID",
			input:        "18446744073709551615",
			wantValue:    uint64(18446744073709551615),
			wantError:    false,
		},

		// Signed integer types
		{
			name:         "int ID - positive",
			model:        &ModelWithIntID{},
			resourceName: "ModelWithIntID",
			input:        "42",
			wantValue:    int(42),
			wantError:    false,
		},
		{
			name:         "int ID - negative",
			model:        &ModelWithIntID{},
			resourceName: "ModelWithIntID",
			input:        "-42",
			wantValue:    int(-42),
			wantError:    false,
		},
		{
			name:         "int ID - zero",
			model:        &ModelWithIntID{},
			resourceName: "ModelWithIntID",
			input:        "0",
			wantValue:    int(0),
			wantError:    false,
		},
		{
			name:         "int8 ID - valid negative",
			model:        &ModelWithInt8ID{},
			resourceName: "ModelWithInt8ID",
			input:        "-128",
			wantValue:    int8(-128),
			wantError:    false,
		},
		{
			name:         "int8 ID - valid positive",
			model:        &ModelWithInt8ID{},
			resourceName: "ModelWithInt8ID",
			input:        "127",
			wantValue:    int8(127),
			wantError:    false,
		},
		{
			name:         "int8 ID - overflow",
			model:        &ModelWithInt8ID{},
			resourceName: "ModelWithInt8ID",
			input:        "128",
			wantValue:    nil,
			wantError:    true,
		},
		{
			name:         "int16 ID - valid",
			model:        &ModelWithInt16ID{},
			resourceName: "ModelWithInt16ID",
			input:        "-32768",
			wantValue:    int16(-32768),
			wantError:    false,
		},
		{
			name:         "int32 ID - valid",
			model:        &ModelWithInt32ID{},
			resourceName: "ModelWithInt32ID",
			input:        "2147483647",
			wantValue:    int32(2147483647),
			wantError:    false,
		},
		{
			name:         "int64 ID - valid large negative",
			model:        &ModelWithInt64ID{},
			resourceName: "ModelWithInt64ID",
			input:        "-9223372036854775808",
			wantValue:    int64(-9223372036854775808),
			wantError:    false,
		},
		{
			name:         "int64 ID - valid large positive",
			model:        &ModelWithInt64ID{},
			resourceName: "ModelWithInt64ID",
			input:        "9223372036854775807",
			wantValue:    int64(9223372036854775807),
			wantError:    false,
		},

		// String types (UUID strings and other string IDs)
		{
			name:         "string ID - UUID v4",
			model:        &ModelWithStringID{},
			resourceName: "ModelWithStringID",
			input:        "550e8400-e29b-11d4-a716-446655440000",
			wantValue:    "550e8400-e29b-11d4-a716-446655440000",
			wantError:    false,
		},
		{
			name:         "string ID - UUID v1",
			model:        &ModelWithStringID{},
			resourceName: "ModelWithStringID",
			input:        "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			wantValue:    "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			wantError:    false,
		},
		{
			name:         "string ID - custom string",
			model:        &ModelWithStringID{},
			resourceName: "ModelWithStringID",
			input:        "user_abc123",
			wantValue:    "user_abc123",
			wantError:    false,
		},
		{
			name:         "string ID - alphanumeric",
			model:        &ModelWithStringID{},
			resourceName: "ModelWithStringID",
			input:        "ABC123XYZ",
			wantValue:    "ABC123XYZ",
			wantError:    false,
		},
		{
			name:         "string ID - empty string",
			model:        &ModelWithStringID{},
			resourceName: "ModelWithStringID",
			input:        "",
			wantValue:    "",
			wantError:    false,
		},
		{
			name:         "string ID - with special chars",
			model:        &ModelWithStringID{},
			resourceName: "ModelWithStringID",
			input:        "id-with-dashes_and_underscores",
			wantValue:    "id-with-dashes_and_underscores",
			wantError:    false,
		},
		{
			name:         "string ID - numeric string",
			model:        &ModelWithStringID{},
			resourceName: "ModelWithStringID",
			input:        "123456",
			wantValue:    "123456",
			wantError:    false,
		},
		{
			name:         "string ID - ULID",
			model:        &ModelWithStringID{},
			resourceName: "ModelWithStringID",
			input:        "01FXMRV3NY2J9Y8K7Z2J9Z2J9Z",
			wantValue:    "01FXMRV3NY2J9Y8K7Z2J9Z2J9Z",
			wantError:    false,
		},
		{
			name:         "string ID - nanoid",
			model:        &ModelWithStringID{},
			resourceName: "ModelWithStringID",
			input:        "V1StGXR8_Z5jdHi6B-myT",
			wantValue:    "V1StGXR8_Z5jdHi6B-myT",
			wantError:    false,
		},
		{
			name:         "string ID - unicode",
			model:        &ModelWithStringID{},
			resourceName: "ModelWithStringID",
			input:        "用户_123",
			wantValue:    "用户_123",
			wantError:    false,
		},

		// uuid.UUID type (actual struct type from github.com/google/uuid)
		{
			name:         "uuid.UUID - valid v4",
			model:        &ModelWithUUIDID{},
			resourceName: "ModelWithUUIDID",
			input:        "123e4567-e89b-12d3-a456-426614174000",
			wantValue:    uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			wantError:    false,
		},
		{
			name:         "uuid.UUID - valid v1",
			model:        &ModelWithUUIDID{},
			resourceName: "ModelWithUUIDID",
			input:        "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			wantValue:    uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
			wantError:    false,
		},
		{
			name:         "uuid.UUID - invalid format",
			model:        &ModelWithUUIDID{},
			resourceName: "ModelWithUUIDID",
			input:        "not-a-uuid",
			wantValue:    nil,
			wantError:    true,
		},
		{
			name:         "uuid.UUID - empty string",
			model:        &ModelWithUUIDID{},
			resourceName: "ModelWithUUIDID",
			input:        "",
			wantValue:    nil,
			wantError:    true,
		},

		// Edge cases
		{
			name:         "uint ID - leading zeros",
			model:        &ModelWithUintID{},
			resourceName: "ModelWithUintID",
			input:        "00042",
			wantValue:    uint(42),
			wantError:    false,
		},
		{
			name:         "int ID - leading plus sign",
			model:        &ModelWithIntID{},
			resourceName: "ModelWithIntID",
			input:        "+42",
			wantValue:    int(42),
			wantError:    false,
		},
		{
			name:         "uint ID - whitespace (should fail)",
			model:        &ModelWithUintID{},
			resourceName: "ModelWithUintID",
			input:        " 42 ",
			wantValue:    nil,
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh BackOffice instance for each test
			mockAdapter := &MockAdapter{}
			admin := New(mockAdapter, auth.WithNoAuth())

			// Register the model
			admin.RegisterResource(tt.model)

			// Get the registered resource
			resource, exists := admin.GetResource(tt.resourceName)
			if !exists {
				t.Fatalf("Resource %q not registered", tt.resourceName)
			}

			// Parse the ID
			got, err := resource.ParseID(tt.input)

			// Check error expectation
			if tt.wantError {
				if err == nil {
					t.Errorf("ParseID() expected error but got none, returned value: %v (%T)", got, got)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseID() unexpected error: %v", err)
				return
			}

			// Compare values
			if !reflect.DeepEqual(got, tt.wantValue) {
				t.Errorf("ParseID() = %v (%T), want %v (%T)", got, got, tt.wantValue, tt.wantValue)
			}
		})
	}
}

// TestParseID_UninitializedIDFieldType tests error handling when IDFieldType is nil
func TestParseID_UninitializedIDFieldType(t *testing.T) {
	resource := &Resource{
		Name:        "TestResource",
		IDFieldType: nil, // Intentionally nil
	}

	_, err := resource.ParseID("123")
	if err == nil {
		t.Error("Expected error when IDFieldType is nil, got none")
	}

	if err != nil && !contains(err.Error(), "not initialized") {
		t.Errorf("Expected error message to contain 'not initialized', got: %v", err)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
