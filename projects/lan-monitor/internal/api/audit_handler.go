package api

import (
	"net/http"
	"runtime"
	"strconv"
	"time"

	"lan-monitor/internal/database"

	"github.com/gin-gonic/gin"
)

var startTime = time.Now()

func handleAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "50"))
	action := c.Query("action")
	userID := c.Query("user_id")

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 50
	}

	where := "WHERE 1=1"
	args := []interface{}{}
	if action != "" {
		where += " AND action LIKE ?"
		args = append(args, "%"+action+"%")
	}
	if userID != "" {
		where += " AND user_id = ?"
		args = append(args, userID)
	}

	var total int64
	database.DB.QueryRow("SELECT COUNT(*) FROM audit_logs "+where, args...).Scan(&total)

	offset := (page - 1) * size
	queryArgs := append(args, size, offset)
	rows, _ := database.DB.Query(
		"SELECT id, user_id, username, action, resource_type, resource_id, ip, user_agent, detail, created_at FROM audit_logs "+where+" ORDER BY created_at DESC LIMIT ? OFFSET ?",
		queryArgs...)
	if rows != nil {
		defer rows.Close()
	}

	type AR struct {
		ID           uint      `json:"id"`
		UserID       *uint     `json:"user_id"`
		Username     string    `json:"username"`
		Action       string    `json:"action"`
		ResourceType string    `json:"resource_type"`
		ResourceID   string    `json:"resource_id"`
		IP           string    `json:"ip"`
		UserAgent    string    `json:"user_agent"`
		Detail       string    `json:"detail"`
		CreatedAt    time.Time `json:"created_at"`
	}
	var logs []AR
	if rows != nil {
		for rows.Next() {
			var l AR
			rows.Scan(&l.ID, &l.UserID, &l.Username, &l.Action, &l.ResourceType, &l.ResourceID, &l.IP, &l.UserAgent, &l.Detail, &l.CreatedAt)
			logs = append(logs, l)
		}
	}
	if logs == nil {
		logs = []AR{}
	}
	c.JSON(http.StatusOK, gin.H{"data": logs, "total": total, "page": page})
}

func handleSystemInfo(c *gin.Context) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	c.JSON(http.StatusOK, gin.H{
		"go_version":     runtime.Version(),
		"goroutines":     runtime.NumGoroutine(),
		"cpu_cores":      runtime.NumCPU(),
		"memory_alloc":   memStats.Alloc / 1024 / 1024,
		"memory_total":   memStats.Sys / 1024 / 1024,
		"uptime_seconds": int64(time.Since(startTime).Seconds()),
		"server_time":    time.Now().Format("2006-01-02 15:04:05"),
	})
}
