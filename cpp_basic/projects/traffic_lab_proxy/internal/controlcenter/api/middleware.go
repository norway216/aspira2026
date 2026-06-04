package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
)

type contextKey string

const (
	ctxAdminUser contextKey = "admin_user"
	ctxNodeID    contextKey = "node_id"
)

// ────────────────────────────────────────────────────────────
// Logging
// ────────────────────────────────────────────────────────────

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(ww, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.statusCode,
			"duration", time.Since(start).String(),
			"remote", r.RemoteAddr,
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// ────────────────────────────────────────────────────────────
// Recovery
// ────────────────────────────────────────────────────────────

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered",
					"panic", rec,
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)
				writeJSON(w, http.StatusInternalServerError, common.APIResponse{
					Success: false,
					Error:   "internal server error",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// ────────────────────────────────────────────────────────────
// Admin Session Auth
// ────────────────────────────────────────────────────────────

func AdminAuthMiddleware(adminUser, adminPass string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check session cookie first
			cookie, err := r.Cookie("admin_session")
			if err == nil && cookie.Value != "" {
				decoded := decodeSessionCookie(cookie.Value)
				if decoded == adminUser {
					ctx := context.WithValue(r.Context(), ctxAdminUser, adminUser)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// Fall back to Basic auth for API access
			user, pass, ok := r.BasicAuth()
			if ok && user == adminUser && pass == adminPass {
				ctx := context.WithValue(r.Context(), ctxAdminUser, adminUser)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// If it's a browser request (web page), redirect to login
			acceptHeader := r.Header.Get("Accept")
			if acceptHeader != "" && (contains(acceptHeader, "text/html") || r.URL.Path == "/" || r.URL.Path == "/admin/login") == false {
				// For API requests, return JSON error
				writeJSON(w, http.StatusUnauthorized, common.APIResponse{
					Success: false,
					Error:   "unauthorized",
				})
				return
			}

			http.Redirect(w, r, "/admin/login", http.StatusFound)
		})
	}
}

// ────────────────────────────────────────────────────────────
// Node HMAC Auth
// ────────────────────────────────────────────────────────────

type NodeSecretLookup func(nodeID string) (string, error)

func NodeAuthMiddleware(lookupSecret NodeSecretLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nodeID := r.Header.Get("X-Node-Id")
			signature := r.Header.Get("X-Node-Signature")

			if nodeID == "" || signature == "" {
				writeJSON(w, http.StatusUnauthorized, common.APIResponse{
					Success: false,
					Error:   "missing node authentication headers",
				})
				return
			}

			secret, err := lookupSecret(nodeID)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, common.APIResponse{
					Success: false,
					Error:   "unknown node",
				})
				return
			}

			// Read body for HMAC verification
			body, err := io.ReadAll(r.Body)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, common.APIResponse{
					Success: false,
					Error:   "cannot read request body",
				})
				return
			}
			// Replace body for downstream handler
			r.Body = io.NopCloser(bytes.NewReader(body))

			expectedSig := computeHMAC(body, secret)
			if subtle.ConstantTimeCompare([]byte(expectedSig), []byte(signature)) != 1 {
				writeJSON(w, http.StatusUnauthorized, common.APIResponse{
					Success: false,
					Error:   "invalid node signature",
				})
				return
			}

			ctx := context.WithValue(r.Context(), ctxNodeID, nodeID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ────────────────────────────────────────────────────────────
// CORS
// ────────────────────────────────────────────────────────────

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Node-Id, X-Node-Signature")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────

func computeHMAC(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func encodeSessionCookie(username string) string {
	return hex.EncodeToString([]byte(username))
}

func decodeSessionCookie(value string) string {
	b, err := hex.DecodeString(value)
	if err != nil {
		return ""
	}
	return string(b)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
