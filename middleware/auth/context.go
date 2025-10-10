package auth

import "context"

// Context key for storing authenticated user in request context
type contextKey string

const authUserKey contextKey = "authUser"

// GetAuthUser retrieves the authenticated user from the request context
// Returns the user and true if authenticated, nil and false otherwise
func GetAuthUser(ctx context.Context) (*AuthUser, bool) {
	user, ok := ctx.Value(authUserKey).(*AuthUser)
	return user, ok
}

// WithAuthUser adds an authenticated user to the request context
func WithAuthUser(ctx context.Context, user *AuthUser) context.Context {
	return context.WithValue(ctx, authUserKey, user)
}

// IsAuthenticated checks if the request context contains an authenticated user
func IsAuthenticated(ctx context.Context) bool {
	_, ok := GetAuthUser(ctx)
	return ok
}

// RequireAuth returns the authenticated user or panics if not authenticated
// This should only be used in handlers where authentication is guaranteed
func RequireAuth(ctx context.Context) *AuthUser {
	user, ok := GetAuthUser(ctx)
	if !ok {
		panic("authentication required but no user found in context")
	}
	return user
}
