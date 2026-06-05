package api

import (
	"net/http"
	"time"

	"lan-monitor/internal/database"

	"github.com/gin-gonic/gin"
)

func handleDashboard(c *gin.Context) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var onlineCount, totalDevices, todayNew, todayOffline, highRisk, offlineCount, openAlerts, scansToday int64

	database.DB.QueryRow("SELECT COUNT(*) FROM devices WHERE status = 'online'").Scan(&onlineCount)
	database.DB.QueryRow("SELECT COUNT(*) FROM devices").Scan(&totalDevices)
	database.DB.QueryRow("SELECT COUNT(*) FROM devices WHERE first_seen >= ?", todayStart).Scan(&todayNew)
	database.DB.QueryRow("SELECT COUNT(*) FROM device_events WHERE event_type = 'offline' AND event_time >= ?", todayStart).Scan(&todayOffline)
	database.DB.QueryRow("SELECT COUNT(*) FROM devices WHERE risk_level IN ('warning', 'critical')").Scan(&highRisk)
	database.DB.QueryRow("SELECT COUNT(*) FROM devices WHERE status = 'offline'").Scan(&offlineCount)
	database.DB.QueryRow("SELECT COUNT(*) FROM alerts WHERE status = 'open'").Scan(&openAlerts)
	database.DB.QueryRow("SELECT COUNT(*) FROM scan_tasks WHERE started_at >= ?", todayStart).Scan(&scansToday)

	// Recent alerts
	rows, _ := database.DB.Query("SELECT id, device_id, device_mac, alert_type, level, title, content, status, created_at FROM alerts ORDER BY created_at DESC LIMIT 10")
	var recentAlerts []gin.H
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var id uint
			var deviceID *uint
			var deviceMAC, alertType, level, title, content, status string
			var createdAt time.Time
			rows.Scan(&id, &deviceID, &deviceMAC, &alertType, &level, &title, &content, &status, &createdAt)
			recentAlerts = append(recentAlerts, gin.H{
				"id": id, "device_id": deviceID, "device_mac": deviceMAC,
				"alert_type": alertType, "level": level, "title": title,
				"status": status, "created_at": createdAt,
			})
		}
	}
	if recentAlerts == nil {
		recentAlerts = []gin.H{}
	}

	// Device type distribution
	tRows, _ := database.DB.Query("SELECT device_type, COUNT(*) as count FROM devices GROUP BY device_type ORDER BY count DESC")
	var typeDist []gin.H
	if tRows != nil {
		defer tRows.Close()
		for tRows.Next() {
			var dt string
			var count int64
			tRows.Scan(&dt, &count)
			typeDist = append(typeDist, gin.H{"device_type": dt, "count": count})
		}
	}
	if typeDist == nil {
		typeDist = []gin.H{}
	}

	// Status distribution
	sRows, _ := database.DB.Query("SELECT status, COUNT(*) as count FROM devices GROUP BY status ORDER BY count DESC")
	var statusDist []gin.H
	if sRows != nil {
		defer sRows.Close()
		for sRows.Next() {
			var st string
			var count int64
			sRows.Scan(&st, &count)
			statusDist = append(statusDist, gin.H{"status": st, "count": count})
		}
	}
	if statusDist == nil {
		statusDist = []gin.H{}
	}

	// Recent events
	eRows, _ := database.DB.Query("SELECT id, device_id, device_mac, event_type, old_status, new_status, event_time, description FROM device_events ORDER BY event_time DESC LIMIT 20")
	var recentEvents []gin.H
	if eRows != nil {
		defer eRows.Close()
		for eRows.Next() {
			var id, deviceID uint
			var deviceMAC, eventType, oldStatus, newStatus, desc string
			var eventTime time.Time
			eRows.Scan(&id, &deviceID, &deviceMAC, &eventType, &oldStatus, &newStatus, &eventTime, &desc)
			recentEvents = append(recentEvents, gin.H{
				"id": id, "device_id": deviceID, "device_mac": deviceMAC,
				"event_type": eventType, "old_status": oldStatus, "new_status": newStatus,
				"event_time": eventTime, "description": desc,
			})
		}
	}
	if recentEvents == nil {
		recentEvents = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"online_count":        onlineCount,
		"total_devices":       totalDevices,
		"today_new":           todayNew,
		"today_offline":       todayOffline,
		"high_risk_count":     highRisk,
		"offline_count":       offlineCount,
		"open_alerts":         openAlerts,
		"scans_today":         scansToday,
		"recent_alerts":       recentAlerts,
		"type_distribution":   typeDist,
		"status_distribution": statusDist,
		"recent_events":       recentEvents,
	})
}
