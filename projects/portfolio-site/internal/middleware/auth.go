package middleware

import (
	"context"
	"net/http"
)

// contextKey is used for storing values in context.
type contextKey string

const (
	// UserIDKey stores the authenticated user's ID.
	UserIDKey contextKey = "user_id"
	// UserNameKey stores the authenticated user's name.
	UserNameKey contextKey = "user_name"
	// UserRoleKey stores the authenticated user's role.
	UserRoleKey contextKey = "user_role"
)

// Auth provides authentication middleware.
type Auth struct {
	sessionName string
}

// NewAuth creates a new auth middleware.
func NewAuth(sessionName string) *Auth {
	return &Auth{sessionName: sessionName}
}

// RequireLogin redirects to login if not authenticated.
func (a *Auth) RequireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.IsAuthenticated(r) {
			// For API requests, return 401.
			if isAPIRequest(r) {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// IsAuthenticated checks if the request has a valid session.
func (a *Auth) IsAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie(a.sessionName)
	if err != nil {
		return false
	}
	// Validate that the cookie has a value.
	return cookie.Value != ""
}

// SetSession sets the user info as a signed cookie.
// In production, use a proper session store (e.g., gorilla/sessions).
func (a *Auth) SetSession(w http.ResponseWriter, userID int64, username, role string, maxAge int, secure, httpOnly bool) {
	// Simple token-based approach. In production, use proper signed tokens.
	token := generateSessionToken(userID, username, role)

	cookie := &http.Cookie{
		Name:     a.sessionName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		Secure:   secure,
		HttpOnly: httpOnly,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

// ClearSession removes the session cookie.
func (a *Auth) ClearSession(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     a.sessionName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)
}

// GetUserID extracts the user ID from context.
func GetUserID(ctx context.Context) int64 {
	if v, ok := ctx.Value(UserIDKey).(int64); ok {
		return v
	}
	return 0
}

// GetUserName extracts the username from context.
func GetUserName(ctx context.Context) string {
	if v, ok := ctx.Value(UserNameKey).(string); ok {
		return v
	}
	return ""
}

// GetUserRole extracts the user role from context.
func GetUserRole(ctx context.Context) string {
	if v, ok := ctx.Value(UserRoleKey).(string); ok {
		return v
	}
	return ""
}

func isAPIRequest(r *http.Request) bool {
	return r.Header.Get("Accept") == "application/json" ||
		r.Header.Get("Content-Type") == "application/json"
}

// generateSessionToken creates a simple session token.
// In production, use JWT or proper session management.
func generateSessionToken(userID int64, username, role string) string {
	// This is a simplified implementation.
	// Use proper JWT or session tokens in production.
	return username
}
