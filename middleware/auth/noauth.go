package auth

// WithNoAuth creates an AuthConfig that disables authentication
// This is the default configuration for BackOffice instances that don't need auth
func WithNoAuth() AuthConfig {
	return AuthConfig{
		Enabled:        false,
		LoginPath:      "/login",
		LogoutPath:     "/logout",
		Authenticator:  nil,
		SessionStore:   nil,
		RequireAuth:    false,
		LoginRedirect:  "/admin",
		LogoutRedirect: "/admin",
	}
}
