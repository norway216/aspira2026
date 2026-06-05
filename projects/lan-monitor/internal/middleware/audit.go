package middleware

import (
	"bytes"
	"io"
	"strings"
	"time"

	"lan-monitor/internal/database"

	"github.com/gin-gonic/gin"
)

func AuditLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/ws") || strings.HasPrefix(path, "/static") {
			c.Next()
			return
		}

		start := time.Now()

		var body string
		if c.Request.Body != nil && c.Request.Method != "GET" {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			body = string(bodyBytes)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			if len(body) > 500 {
				body = body[:500] + "..."
			}
		}

		c.Next()

		method := c.Request.Method
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			return
		}

		userID, _ := c.Get("user_id")
		username, _ := c.Get("username")

		var uid interface{} = nil
		var uname string
		if v, ok := userID.(uint); ok {
			uid = v
		}
		if v, ok := username.(string); ok {
			uname = v
		}

		database.DB.Exec(
			"INSERT INTO audit_logs (user_id, username, action, resource_type, resource_id, ip, user_agent, detail, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			uid, uname, method+" "+c.FullPath(), resourceTypeFromPath(path),
			c.Param("id"), c.ClientIP(), truncateStr(c.GetHeader("User-Agent"), 200), body, start)
	}
}

func resourceTypeFromPath(path string) string {
	switch {
	case strings.Contains(path, "/auth"):
		return "auth"
	case strings.Contains(path, "/devices"):
		return "device"
	case strings.Contains(path, "/scans"):
		return "scan"
	case strings.Contains(path, "/traffic"):
		return "traffic"
	case strings.Contains(path, "/alerts"):
		return "alert"
	case strings.Contains(path, "/users"):
		return "user"
	case strings.Contains(path, "/agents"):
		return "agent"
	default:
		return "other"
	}
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
