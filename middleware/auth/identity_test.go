package auth

import (
	"testing"
)

func TestIdentityHelpers_GetString(t *testing.T) {
	tests := []struct {
		name     string
		id       Identity
		expected string
	}{
		{"string identity", "test123", "test123"},
		{"integer identity", 42, ""},
		{"nil identity", nil, ""},
		{"empty string", "", ""},
	}

	helper := IdentityHelpers{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helper.GetString(tt.id)
			if result != tt.expected {
				t.Errorf("GetString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIdentityHelpers_GetInt64(t *testing.T) {
	tests := []struct {
		name     string
		id       Identity
		expected int64
	}{
		{"int64 identity", int64(123), 123},
		{"int identity", int(456), 456},
		{"uint identity", uint(789), 789},
		{"string identity", "not_a_number", 0},
		{"nil identity", nil, 0},
	}

	helper := IdentityHelpers{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helper.GetInt64(tt.id)
			if result != tt.expected {
				t.Errorf("GetInt64() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIdentityHelpers_GetUint(t *testing.T) {
	tests := []struct {
		name     string
		id       Identity
		expected uint
	}{
		{"uint identity", uint(123), 123},
		{"uint32 identity", uint32(456), 456},
		{"positive int", int(789), 789},
		{"negative int", int(-10), 0},
		{"string identity", "not_a_number", 0},
		{"nil identity", nil, 0},
	}

	helper := IdentityHelpers{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helper.GetUint(tt.id)
			if result != tt.expected {
				t.Errorf("GetUint() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGlobalHelper(t *testing.T) {
	// Test that the global ID helper works
	stringResult := ID.GetString("hello")
	if stringResult != "hello" {
		t.Errorf("Global ID helper GetString() = %v, want %v", stringResult, "hello")
	}

	intResult := ID.GetInt64(int64(42))
	if intResult != 42 {
		t.Errorf("Global ID helper GetInt64() = %v, want %v", intResult, 42)
	}
}
