package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// CSRF provides CSRF protection middleware.
type CSRF struct {
	cookieName string
}

// NewCSRF creates a new CSRF middleware.
func NewCSRF() *CSRF {
	return &CSRF{cookieName: "csrf_token"}
}

// Protect validates CSRF tokens for state-changing requests.
func (c *CSRF) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Auto-generate CSRF token on GET requests so forms have it available.
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			// Ensure a CSRF cookie is always set for browser clients.
			if _, err := r.Cookie(c.cookieName); err != nil {
				c.SetToken(w, r)
			}
			next.ServeHTTP(w, r)
			return
		}

		// For API requests, check the X-CSRF-Token header.
		token := r.Header.Get("X-CSRF-Token")
		if token == "" {
			token = r.FormValue("csrf_token")
		}

		cookie, err := r.Cookie(c.cookieName)
		if err != nil || cookie.Value == "" {
			http.Error(w, "CSRF token missing", http.StatusForbidden)
			return
		}

		if token != cookie.Value {
			http.Error(w, "CSRF token mismatch", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SetToken sets a CSRF token cookie.
func (c *CSRF) SetToken(w http.ResponseWriter, r *http.Request) string {
	token := generateCSRFToken()

	http.SetCookie(w, &http.Cookie{
		Name:     c.cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   86400,
		Secure:   false, // Set to true in production with HTTPS.
		HttpOnly: false, // Must be readable by JS if needed.
		SameSite: http.SameSiteLaxMode,
	})

	return token
}

// GetTokenField returns an HTML hidden input field with the CSRF token.
func (c *CSRF) GetTokenField(r *http.Request) string {
	cookie, err := r.Cookie(c.cookieName)
	if err != nil {
		return `<input type="hidden" name="csrf_token" value="">`
	}
	return `<input type="hidden" name="csrf_token" value="` + cookie.Value + `">`
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
