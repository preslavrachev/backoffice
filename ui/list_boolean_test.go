package ui

import (
	"context"
	"strings"
	"testing"

	"backoffice/core"
)

// TestListBooleanFieldIntegration tests that boolean fields in the list render using the Yes/No component
func TestListBooleanFieldIntegration(t *testing.T) {
	// Test entity with boolean field
	type TestEntity struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		IsActive bool   `json:"is_active"`
	}

	// Create resource with boolean field
	resource := &core.Resource{
		Name:        "TestEntity",
		DisplayName: "Test Entity",
		PluralName:  "Test Entities",
		IDField:     "ID",
		Fields: []core.FieldInfo{
			{
				Name:        "Name",
				DisplayName: "Name",
				Type:        "string",
			},
			{
				Name:        "IsActive",
				DisplayName: "Active",
				Type:        "bool",
			},
		},
	}

	tests := []struct {
		name           string
		boolValue      bool
		expectedOutput string
	}{
		{
			name:           "true boolean renders Yes badge",
			boolValue:      true,
			expectedOutput: renderYesNoComponent(true),
		},
		{
			name:           "false boolean renders No badge",
			boolValue:      false,
			expectedOutput: renderYesNoComponent(false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEntity := &TestEntity{
				ID:       1,
				Name:     "Test Item",
				IsActive: tt.boolValue,
			}

			// Render the full list
			var sb strings.Builder
			ctx := context.Background()
			items := []interface{}{testEntity}

			err := List(resource, items, 1, "").Render(ctx, &sb)
			if err != nil {
				t.Fatalf("Failed to render List: %v", err)
			}

			listHTML := sb.String()

			// Check that the Yes/No component output appears in the list
			if !strings.Contains(listHTML, tt.expectedOutput) {
				t.Errorf("Expected list HTML to contain Yes/No component output:\n%s\nBut got:\n%s", tt.expectedOutput, listHTML)
			}
		})
	}
}

// renderYesNoComponent renders the FormatBooleanField component and returns its HTML
func renderYesNoComponent(value bool) string {
	var sb strings.Builder
	ctx := context.Background()

	err := FormatBooleanField(value).Render(ctx, &sb)
	if err != nil {
		panic("Failed to render FormatBooleanField: " + err.Error())
	}

	return sb.String()
}
