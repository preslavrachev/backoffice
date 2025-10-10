package auth

import (
	"context"
	"net/http"
)

// AuthUser represents an authenticated user in the system
type AuthUser struct {
	ID       Identity `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
}

// AuthenticatorFunc defines the interface for authentication functions
// It takes a username and password and returns an AuthUser or an error
type AuthenticatorFunc func(ctx context.Context, username, password string) (*AuthUser, error)

// AuthConfig holds the complete authentication configuration
type AuthConfig struct {
	// Enabled determines if authentication is active
	Enabled bool

	// LoginPath is the URL path for the login page (default: "/login")
	LoginPath string

	// LogoutPath is the URL path for logout (default: "/logout")
	LogoutPath string

	// Authenticator is the function used to validate user credentials
	Authenticator AuthenticatorFunc

	// SessionStore handles session persistence
	SessionStore SessionStore

	// RequireAuth determines if all admin routes require authentication
	// If false, authentication is optional and users can access without logging in
	RequireAuth bool

	// LoginRedirect is the path to redirect to after successful login
	LoginRedirect string

	// LogoutRedirect is the path to redirect to after logout
	LogoutRedirect string
}

// SessionStore defines the interface for session management
type SessionStore interface {
	// GetSession retrieves a user session by session ID
	GetSession(ctx context.Context, sessionID string) (*AuthUser, error)

	// CreateSession creates a new session for the user and returns the session ID
	CreateSession(ctx context.Context, user *AuthUser) (sessionID string, err error)

	// DeleteSession removes a session by session ID
	DeleteSession(ctx context.Context, sessionID string) error

	// CleanExpiredSessions removes expired sessions (called periodically)
	CleanExpiredSessions(ctx context.Context) error
}

// AuthMiddleware wraps HTTP handlers to provide authentication
type AuthMiddleware func(http.Handler) http.Handler
