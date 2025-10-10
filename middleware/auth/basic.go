package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/preslavrachev/backoffice/config"
)

// BasicAuthUser represents a user configured for basic authentication
type BasicAuthUser struct {
	Username string
	Password string
	User     AuthUser
}

// WithBasicAuth creates an AuthConfig that uses HTTP Basic Authentication
// Users are provided as a map of username -> BasicAuthUser
func WithBasicAuth(users map[string]BasicAuthUser) AuthConfig {
	sessionStore := NewMemorySessionStore()

	authenticator := func(ctx context.Context, username, password string) (*AuthUser, error) {
		fmt.Printf("🔐 DEBUG: BasicAuth - Checking username: '%s'\n", username)
		fmt.Printf("🔐 DEBUG: BasicAuth - Available users: %v\n", func() []string {
			var usernames []string
			for u := range users {
				usernames = append(usernames, u)
			}
			return usernames
		}())

		user, exists := users[username]
		if !exists {
			fmt.Printf("❌ DEBUG: BasicAuth - User '%s' not found\n", username)
			return nil, errors.New("user not found")
		}
		fmt.Printf("✅ DEBUG: BasicAuth - User '%s' found\n", username)

		// Use constant time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(password), []byte(user.Password)) != 1 {
			fmt.Printf("❌ DEBUG: BasicAuth - Password mismatch for user '%s' (expected: '%s', got: '%s')\n", username, user.Password, password)
			return nil, errors.New("invalid password")
		}
		fmt.Printf("✅ DEBUG: BasicAuth - Password correct for user '%s'\n", username)

		// Return the configured AuthUser
		return &user.User, nil
	}

	return AuthConfig{
		Enabled:        true,
		LoginPath:      "/login",
		LogoutPath:     "/logout",
		Authenticator:  authenticator,
		SessionStore:   sessionStore,
		RequireAuth:    true,
		LoginRedirect:  "/admin",
		LogoutRedirect: "/admin",
	}
}

// WithBasicAuthAndTimeout creates an AuthConfig with custom session timeout
func WithBasicAuthAndTimeout(users map[string]BasicAuthUser, sessionTimeout time.Duration) AuthConfig {
	sessionStore := NewMemorySessionStoreWithTimeout(sessionTimeout)

	authenticator := func(ctx context.Context, username, password string) (*AuthUser, error) {
		user, exists := users[username]
		if !exists {
			return nil, errors.New("user not found")
		}

		// Use constant time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(password), []byte(user.Password)) != 1 {
			return nil, errors.New("invalid password")
		}

		// Return the configured AuthUser
		return &user.User, nil
	}

	return AuthConfig{
		Enabled:        true,
		LoginPath:      "/login",
		LogoutPath:     "/logout",
		Authenticator:  authenticator,
		SessionStore:   sessionStore,
		RequireAuth:    true,
		LoginRedirect:  "/admin",
		LogoutRedirect: "/admin",
	}
}

// NewBasicAuthUser creates a BasicAuthUser with the provided details
// This is a helper function to make it easier to create basic auth users
func NewBasicAuthUser(username, password string, id Identity, email string, roles []string) BasicAuthUser {
	return BasicAuthUser{
		Username: username,
		Password: password,
		User: AuthUser{
			ID:       id,
			Username: username,
			Email:    email,
			Roles:    roles,
		},
	}
}

// WithBasicAuthFromConfig creates an AuthConfig using centralized configuration
func WithBasicAuthFromConfig() AuthConfig {
	cfg := config.LoadConfig()

	// Create a single admin user from config
	users := map[string]BasicAuthUser{
		cfg.Auth.BasicAuthUser: NewBasicAuthUser(
			cfg.Auth.BasicAuthUser,
			cfg.Auth.BasicAuthPass,
			"admin001",
			"admin@example.com",
			[]string{"admin"},
		),
	}

	return WithBasicAuth(users)
}
