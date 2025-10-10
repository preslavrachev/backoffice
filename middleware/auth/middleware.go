package auth

import (
	"net/http"
	"strings"
)

const sessionCookieName = "backoffice_session"

// CreateAuthMiddleware creates HTTP middleware for authentication
func CreateAuthMiddleware(authConfig *AuthConfig) func(http.Handler) http.Handler {
	if authConfig == nil || !authConfig.Enabled {
		// Return no-op middleware if auth is disabled
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if this is a login/logout endpoint
			if isAuthEndpoint(r.URL.Path, authConfig) {
				// Let auth endpoints handle themselves
				next.ServeHTTP(w, r)
				return
			}

			// Try to get user from session
			user, err := getUserFromSession(r, authConfig)
			if err != nil && authConfig.RequireAuth {
				// Redirect to login page if authentication is required
				redirectToLogin(w, r, authConfig)
				return
			}

			// Add user to context if authenticated
			ctx := r.Context()
			if user != nil {
				ctx = WithAuthUser(ctx, user)
			}

			// Continue with the request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// isAuthEndpoint checks if the path is an authentication endpoint
func isAuthEndpoint(path string, authConfig *AuthConfig) bool {
	basePath := getBasePath(path)
	loginPath := basePath + authConfig.LoginPath
	logoutPath := basePath + authConfig.LogoutPath
	return path == loginPath || path == logoutPath
}

// getUserFromSession retrieves the user from the session cookie
func getUserFromSession(r *http.Request, authConfig *AuthConfig) (*AuthUser, error) {
	// Get session cookie
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, err
	}

	// Get user from session store
	return authConfig.SessionStore.GetSession(r.Context(), cookie.Value)
}

// redirectToLogin redirects the user to the login page
func redirectToLogin(w http.ResponseWriter, r *http.Request, authConfig *AuthConfig) {
	// Store the original URL to redirect back after login
	returnURL := r.URL.Path
	if r.URL.RawQuery != "" {
		returnURL += "?" + r.URL.RawQuery
	}

	// Use the base path from the current request to construct the login URL
	basePath := getBasePath(r.URL.Path)
	loginURL := basePath + authConfig.LoginPath
	if returnURL != loginURL {
		loginURL += "?return=" + returnURL
	}

	http.Redirect(w, r, loginURL, http.StatusSeeOther)
}

// getBasePath extracts the base path from the current request path
func getBasePath(requestPath string) string {
	// For paths like "/admin/something", extract "/admin"
	if strings.HasPrefix(requestPath, "/admin") {
		return "/admin"
	}
	return ""
}

// CreateSessionCookie creates a session cookie for the authenticated user
func CreateSessionCookie(sessionID string) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		// MaxAge is controlled by the session store timeout
	}
}

// DeleteSessionCookie creates a cookie that deletes the session
func DeleteSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   -1, // Delete the cookie immediately
	}
}
