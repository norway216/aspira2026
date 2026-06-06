package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/aspira/crossborder-payment-gateway/internal/database"
	"github.com/aspira/crossborder-payment-gateway/internal/models"
	"github.com/gin-gonic/gin"
)

func AuditLog(db database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only log mutating requests
		if c.Request.Method == "GET" || c.Request.Method == "OPTIONS" || c.Request.Method == "HEAD" {
			c.Next()
			return
		}

		// Capture request body
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		c.Next()

		// Determine audit action from path
		action := determineAction(c.Request.Method, c.FullPath())

		userID, _ := c.Get("user_id")
		userIDStr, _ := userID.(string)

		// Try to extract resource ID from path
		resourceID := extractResourceID(c)

		// Build detail JSON
		detail := map[string]interface{}{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"request_ip": c.ClientIP(),
		}
		if len(bodyBytes) > 0 && len(bodyBytes) < 4096 {
			var bodyJSON interface{}
			if json.Unmarshal(bodyBytes, &bodyJSON) == nil {
				detail["body"] = bodyJSON
			}
		}
		detailJSON, _ := json.Marshal(detail)

		log := &models.AuditLog{
			UserID:     userIDStr,
			Action:     action,
			Resource:   determineResource(c.FullPath()),
			ResourceID: resourceID,
			IPAddr:     c.ClientIP(),
			UserAgent:  c.GetHeader("User-Agent"),
			Detail:     string(detailJSON),
			CreatedAt:  time.Now(),
		}

		_ = db.CreateAuditLog(log)
	}
}

func determineAction(method, path string) string {
	switch {
	case path == "/api/v1/auth/login":
		return "user.login"
	case path == "/api/v1/auth/logout":
		return "user.logout"
	case path == "/api/v1/transactions" && method == "POST":
		return "txn.create"
	case path == "/api/v1/transactions/:id/refund":
		return "txn.refund"
	case path == "/api/v1/merchants" && method == "POST":
		return "merchant.create"
	case path == "/api/v1/merchants/:id" && method == "PUT":
		return "merchant.update"
	case path == "/api/v1/merchants/:id" && method == "DELETE":
		return "merchant.delete"
	case path == "/api/v1/exchange-rates" && method == "POST":
		return "rate.update"
	default:
		return "api." + method
	}
}

func determineResource(path string) string {
	switch {
	case contains(path, "transactions"):
		return "transaction"
	case contains(path, "accounts"):
		return "account"
	case contains(path, "merchants"):
		return "merchant"
	case contains(path, "exchange-rates"):
		return "exchange_rate"
	case contains(path, "auth"):
		return "auth"
	default:
		return "unknown"
	}
}

func extractResourceID(c *gin.Context) string {
	if id := c.Param("id"); id != "" {
		return id
	}
	return ""
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
