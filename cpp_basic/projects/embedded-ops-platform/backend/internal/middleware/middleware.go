package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/embedded-ops-platform/backend/internal/model"
	"github.com/embedded-ops-platform/backend/pkg/logger"
	"github.com/embedded-ops-platform/backend/pkg/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	// Context keys
	CtxUserID   = "user_id"
	CtxUsername = "username"
	CtxUserRole = "user_role"
	CtxRequestID = "request_id"
)

// CORSMiddleware handles CORS for frontend communication.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Device-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// RequestIDMiddleware adds a unique request ID to each request.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set(CtxRequestID, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// AuthMiddleware validates JWT tokens for user API access.
func AuthMiddleware(tokenManager *token.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}

		claims, err := tokenManager.ValidateToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxUsername, claims.Username)
		c.Set(CtxUserRole, claims.Role)
		c.Next()
	}
}

// AgentAuthMiddleware validates agent tokens for agent API access.
func AgentAuthMiddleware(validateAgentFn func(deviceID, tokenStr string) bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.GetHeader("X-Device-ID")
		authHeader := c.GetHeader("Authorization")

		if deviceID == "" || authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing X-Device-ID or Authorization header"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}

		if !validateAgentFn(deviceID, tokenStr) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid agent credentials"})
			return
		}

		c.Set("device_id", deviceID)
		c.Next()
	}
}

// RoleMiddleware checks that the user has at least one of the required roles.
func RoleMiddleware(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get(CtxUserRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "no role information"})
			return
		}

		roleStr := userRole.(string)
		for _, role := range roles {
			if roleStr == role {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}

// LoggerMiddleware logs incoming requests.
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		requestID, _ := c.Get(CtxRequestID)

		logger.Info("HTTP Request",
			logger.String("method", c.Request.Method),
			logger.String("path", path),
			logger.String("query", query),
			logger.Int("status", status),
			logger.Duration("latency", latency),
			logger.String("request_id", requestID.(string)),
			logger.String("client_ip", c.ClientIP()),
		)
	}
}

// RateLimitMiddleware provides simple in-memory rate limiting.
type RateLimitMiddleware struct {
	mu       sync.Mutex
	visitors map[string]*rateLimitEntry
}

type rateLimitEntry struct {
	count    int
	resetAt  time.Time
}

func NewRateLimitMiddleware() *RateLimitMiddleware {
	m := &RateLimitMiddleware{
		visitors: make(map[string]*rateLimitEntry),
	}
	// Cleanup old entries periodically
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			m.mu.Lock()
			now := time.Now()
			for k, v := range m.visitors {
				if now.After(v.resetAt) {
					delete(m.visitors, k)
				}
			}
			m.mu.Unlock()
		}
	}()
	return m
}

func (m *RateLimitMiddleware) Limit(requests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP() + ":" + c.Request.URL.Path

		m.mu.Lock()
		entry, exists := m.visitors[key]
		now := time.Now()

		if !exists || now.After(entry.resetAt) {
			m.visitors[key] = &rateLimitEntry{
				count:   1,
				resetAt: now.Add(window),
			}
			m.mu.Unlock()
			c.Next()
			return
		}

		entry.count++
		if entry.count > requests {
			m.mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
				"retry_after": entry.resetAt.Sub(now).Seconds(),
			})
			return
		}
		m.mu.Unlock()
		c.Next()
	}
}

// VerifyPassword checks a plain-text password against a bcrypt hash.
// This is a placeholder — in production, load the hash from the database.
func VerifyPassword(hashedPwd, plainPwd string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(plainPwd))
	return err == nil
}

// HashPassword creates a bcrypt hash of the password.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// GetUserFromContext extracts user info from the Gin context.
func GetUserFromContext(c *gin.Context) *model.User {
	_, _ = c.Get(CtxUserID) // userID available if needed
	username, _ := c.Get(CtxUsername)
	role, _ := c.Get(CtxUserRole)
	return &model.User{
		Username: username.(string),
		Role:     role.(string),
	}
}

