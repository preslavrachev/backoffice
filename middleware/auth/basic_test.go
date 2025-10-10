package auth

import (
	"context"
	"testing"
)

func TestWithBasicAuth(t *testing.T) {
	// Create test users
	users := map[string]BasicAuthUser{
		"admin": NewBasicAuthUser("admin", "password123", "admin001", "admin@test.com", []string{"admin"}),
		"user":  NewBasicAuthUser("user", "userpass", "user001", "user@test.com", []string{"user"}),
	}

	config := WithBasicAuth(users)

	// Test that config is properly set up
	if !config.Enabled {
		t.Error("Expected auth to be enabled")
	}

	if config.Authenticator == nil {
		t.Error("Expected authenticator to be set")
	}

	if config.SessionStore == nil {
		t.Error("Expected session store to be set")
	}

	// Test valid authentication
	ctx := context.Background()
	user, err := config.Authenticator(ctx, "admin", "password123")
	if err != nil {
		t.Errorf("Expected successful authentication, got error: %v", err)
	}

	if user == nil {
		t.Error("Expected user to be returned")
	}

	if user.Username != "admin" {
		t.Errorf("Expected username 'admin', got '%s'", user.Username)
	}

	if user.Email != "admin@test.com" {
		t.Errorf("Expected email 'admin@test.com', got '%s'", user.Email)
	}

	// Test invalid username
	_, err = config.Authenticator(ctx, "nonexistent", "password123")
	if err == nil {
		t.Error("Expected authentication to fail for non-existent user")
	}

	// Test invalid password
	_, err = config.Authenticator(ctx, "admin", "wrongpassword")
	if err == nil {
		t.Error("Expected authentication to fail for wrong password")
	}
}

func TestNewBasicAuthUser(t *testing.T) {
	user := NewBasicAuthUser("testuser", "testpass", "test001", "test@example.com", []string{"admin", "user"})

	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.Username)
	}

	if user.Password != "testpass" {
		t.Errorf("Expected password 'testpass', got '%s'", user.Password)
	}

	if user.User.Username != "testuser" {
		t.Errorf("Expected User.Username 'testuser', got '%s'", user.User.Username)
	}

	if user.User.Email != "test@example.com" {
		t.Errorf("Expected User.Email 'test@example.com', got '%s'", user.User.Email)
	}

	expectedRoles := []string{"admin", "user"}
	if len(user.User.Roles) != len(expectedRoles) {
		t.Errorf("Expected %d roles, got %d", len(expectedRoles), len(user.User.Roles))
	}

	for i, role := range expectedRoles {
		if user.User.Roles[i] != role {
			t.Errorf("Expected role '%s' at index %d, got '%s'", role, i, user.User.Roles[i])
		}
	}
}
