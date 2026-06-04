package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/secure-gateway/internal/auth"
	"github.com/secure-gateway/internal/model"
	"github.com/secure-gateway/pkg/limiter"
	"github.com/secure-gateway/pkg/logger"
)

// CORSMiddleware adds CORS headers.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// LoggerMiddleware logs all HTTP requests.
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		clientIP := c.ClientIP()

		logger.Info("HTTP request",
			"method", method,
			"path", path,
			"status", status,
			"latency", latency.String(),
			"ip", clientIP,
		)
	}
}

// AuthMiddleware validates JWT access token.
func AuthMiddleware(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, model.APIResponse{
				Code:    401,
				Message: "Missing authorization header",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, model.APIResponse{
				Code:    401,
				Message: "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		claims, err := jwtManager.ValidateAccessToken(parts[1])
		if err != nil {
			msg := "Invalid token"
			if err == auth.ErrExpiredToken {
				msg = "Token has expired"
			}
			c.JSON(http.StatusUnauthorized, model.APIResponse{
				Code:    401,
				Message: msg,
			})
			c.Abort()
			return
		}

		// Store user info in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// AdminMiddleware restricts access to admin users only.
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != model.RoleAdmin {
			c.JSON(http.StatusForbidden, model.APIResponse{
				Code:    403,
				Message: "Admin access required",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RateLimitMiddleware limits request rate per client IP.
func RateLimitMiddleware(rl *limiter.RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rl == nil {
			c.Next()
			return
		}
		clientIP := c.ClientIP()
		if !rl.Allow(clientIP) {
			c.JSON(http.StatusTooManyRequests, model.APIResponse{
				Code:    429,
				Message: "Too many requests, please slow down",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// AuditLogMiddleware records audit log for sensitive operations.
func AuditLogMiddleware(db interface {
	InsertAuditLog(l *model.AuditLog) error
}, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		status := c.Writer.Status()
		if status >= 200 && status < 300 {
			userID, _ := c.Get("user_id")
			username, _ := c.Get("username")

			var uid *int64
			if id, ok := userID.(int64); ok {
				uid = &id
			}

			_ = db.InsertAuditLog(&model.AuditLog{
				UserID:    uid,
				Action:    action,
				Resource:  c.Request.URL.Path,
				IPAddr:    c.ClientIP(),
				UserAgent: c.Request.UserAgent(),
				Detail:    username.(string),
			})
		}
	}
}