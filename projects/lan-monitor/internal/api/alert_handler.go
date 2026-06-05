package api

import (
	"net/http"
	"strconv"
	"time"

	"lan-monitor/internal/database"

	"github.com/gin-gonic/gin"
)

func handleAlertList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	status := c.Query("status")
	level := c.Query("level")
	alertType := c.Query("type")

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	where := "WHERE 1=1"
	args := []interface{}{}
	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}
	if level != "" {
		where += " AND level = ?"
		args = append(args, level)
	}
	if alertType != "" {
		where += " AND alert_type = ?"
		args = append(args, alertType)
	}

	var total int64
	database.DB.QueryRow("SELECT COUNT(*) FROM alerts "+where, args...).Scan(&total)

	offset := (page - 1) * size
	queryArgs := append(args, size, offset)
	rows, _ := database.DB.Query(
		"SELECT id, device_id, device_mac, alert_type, level, title, content, status, created_at, resolved_at, resolved_by FROM alerts "+where+" ORDER BY created_at DESC LIMIT ? OFFSET ?",
		queryArgs...)
	if rows != nil {
		defer rows.Close()
	}

	type AR struct {
		ID         uint       `json:"id"`
		DeviceID   *uint      `json:"device_id"`
		DeviceMAC  string     `json:"device_mac"`
		AlertType  string     `json:"alert_type"`
		Level      string     `json:"level"`
		Title      string     `json:"title"`
		Content    string     `json:"content"`
		Status     string     `json:"status"`
		CreatedAt  time.Time  `json:"created_at"`
		ResolvedAt *time.Time `json:"resolved_at"`
		ResolvedBy *uint      `json:"resolved_by"`
	}
	var alerts []AR
	if rows != nil {
		for rows.Next() {
			var a AR
			rows.Scan(&a.ID, &a.DeviceID, &a.DeviceMAC, &a.AlertType, &a.Level, &a.Title, &a.Content, &a.Status, &a.CreatedAt, &a.ResolvedAt, &a.ResolvedBy)
			alerts = append(alerts, a)
		}
	}
	if alerts == nil {
		alerts = []AR{}
	}
	c.JSON(http.StatusOK, gin.H{"data": alerts, "total": total, "page": page})
}

func handleAlertDetail(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var a struct {
		ID         uint       `json:"id"`
		DeviceID   *uint      `json:"device_id"`
		DeviceMAC  string     `json:"device_mac"`
		AlertType  string     `json:"alert_type"`
		Level      string     `json:"level"`
		Title      string     `json:"title"`
		Content    string     `json:"content"`
		Status     string     `json:"status"`
		CreatedAt  time.Time  `json:"created_at"`
		ResolvedAt *time.Time `json:"resolved_at"`
	}
	err := database.DB.QueryRow(
		"SELECT id, device_id, device_mac, alert_type, level, title, content, status, created_at, resolved_at FROM alerts WHERE id = ?", id,
	).Scan(&a.ID, &a.DeviceID, &a.DeviceMAC, &a.AlertType, &a.Level, &a.Title, &a.Content, &a.Status, &a.CreatedAt, &a.ResolvedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "告警不存在"})
		return
	}
	c.JSON(http.StatusOK, a)
}

func handleAlertUpdate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Level *string `json:"level"`
		Title *string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求数据格式错误"})
		return
	}
	if req.Level != nil {
		database.DB.Exec("UPDATE alerts SET level = ? WHERE id = ?", *req.Level, id)
	}
	if req.Title != nil {
		database.DB.Exec("UPDATE alerts SET title = ? WHERE id = ?", *req.Title, id)
	}
	c.JSON(http.StatusOK, gin.H{"message": "告警已更新"})
}

func handleAlertResolve(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint("user_id")
	now := time.Now()
	database.DB.Exec("UPDATE alerts SET status = 'resolved', resolved_at = ?, resolved_by = ? WHERE id = ?", now, userID, id)
	c.JSON(http.StatusOK, gin.H{"message": "告警已解决"})
}

func handleAlertStats(c *gin.Context) {
	var totalOpen, totalResolved, totalCritical int64
	database.DB.QueryRow("SELECT COUNT(*) FROM alerts WHERE status = 'open'").Scan(&totalOpen)
	database.DB.QueryRow("SELECT COUNT(*) FROM alerts WHERE status = 'resolved'").Scan(&totalResolved)
	database.DB.QueryRow("SELECT COUNT(*) FROM alerts WHERE level = 'critical' AND status = 'open'").Scan(&totalCritical)

	type TypeCount struct {
		AlertType string `json:"alert_type"`
		Count     int64  `json:"count"`
	}
	var typeCounts []TypeCount
	rows, _ := database.DB.Query("SELECT alert_type, COUNT(*) as count FROM alerts WHERE status = 'open' GROUP BY alert_type ORDER BY count DESC")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var tc TypeCount
			rows.Scan(&tc.AlertType, &tc.Count)
			typeCounts = append(typeCounts, tc)
		}
	}
	if typeCounts == nil {
		typeCounts = []TypeCount{}
	}
	c.JSON(http.StatusOK, gin.H{"total_open": totalOpen, "total_resolved": totalResolved, "total_critical": totalCritical, "by_type": typeCounts})
}
