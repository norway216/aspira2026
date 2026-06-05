package api

import (
	"net/http"
	"time"

	"lan-monitor/internal/database"
	"lan-monitor/internal/websocket"

	"github.com/gin-gonic/gin"
)

var agentHub *websocket.Hub

func SetAgentHub(hub *websocket.Hub) {
	agentHub = hub
}

type AgentRegisterRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	IP       string `json:"ip"`
	MAC      string `json:"mac"`
	Hostname string `json:"hostname"`
}

func handleAgentRegister(c *gin.Context) {
	var req AgentRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供设备ID"})
		return
	}

	// Find existing by MAC or label
	var deviceID uint
	if req.MAC != "" {
		database.DB.QueryRow("SELECT id FROM devices WHERE mac = ?", req.MAC).Scan(&deviceID)
	}
	if deviceID == 0 {
		database.DB.QueryRow("SELECT id FROM devices WHERE label = ?", req.DeviceID).Scan(&deviceID)
	}

	now := time.Now()
	if deviceID == 0 {
		result, _ := database.DB.Exec(
			"INSERT INTO devices (ip, mac, hostname, label, status, first_seen, last_seen, last_online_at) VALUES (?, ?, ?, ?, 'online', ?, ?, ?)",
			req.IP, req.MAC, req.Hostname, req.DeviceID, now, now, now)
		id, _ := result.LastInsertId()
		deviceID = uint(id)
	} else {
		database.DB.Exec("UPDATE devices SET ip = ?, hostname = ?, status = 'online', last_seen = ?, last_online_at = ? WHERE id = ?",
			req.IP, req.Hostname, now, now, deviceID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "注册成功", "device_id": deviceID})
}

type AgentHeartbeatRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	IP       string `json:"ip"`
}

func handleAgentHeartbeat(c *gin.Context) {
	var req AgentHeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供设备ID"})
		return
	}
	now := time.Now()
	database.DB.Exec("UPDATE devices SET status = 'online', last_seen = ? WHERE label = ? OR mac = ?", now, req.DeviceID, req.DeviceID)
	c.JSON(http.StatusOK, gin.H{"message": "心跳已接收"})
}

type AgentMetricsRequest struct {
	DeviceID    string  `json:"device_id" binding:"required"`
	IP          string  `json:"ip"`
	MAC         string  `json:"mac"`
	RxBytes     int64   `json:"rx_bytes"`
	TxBytes     int64   `json:"tx_bytes"`
	RxRate      float64 `json:"rx_rate"`
	TxRate      float64 `json:"tx_rate"`
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	Temperature float64 `json:"temperature"`
}

func handleAgentMetrics(c *gin.Context) {
	var req AgentMetricsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供指标数据"})
		return
	}

	// Find device
	var deviceID uint
	var deviceMAC string
	if req.MAC != "" {
		database.DB.QueryRow("SELECT id, mac FROM devices WHERE mac = ?", req.MAC).Scan(&deviceID, &deviceMAC)
	}
	if deviceID == 0 {
		database.DB.QueryRow("SELECT id, mac FROM devices WHERE label = ?", req.DeviceID).Scan(&deviceID, &deviceMAC)
	}

	now := time.Now()
	if deviceID > 0 {
		totalRate := req.RxRate + req.TxRate
		database.DB.Exec(
			"INSERT INTO traffic_records (device_id, device_mac, rx_bytes, tx_bytes, rx_rate, tx_rate, total_rate, timestamp) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			deviceID, deviceMAC, req.RxBytes, req.TxBytes, req.RxRate, req.TxRate, totalRate, now)

		database.DB.Exec("UPDATE devices SET status = 'online', last_seen = ? WHERE id = ?", now, deviceID)

		if agentHub != nil {
			agentHub.BroadcastTraffic(deviceID, gin.H{
				"rx_rate": req.RxRate, "tx_rate": req.TxRate, "total_rate": totalRate,
				"cpu_usage": req.CPUUsage, "memory_usage": req.MemoryUsage, "timestamp": now.Unix(),
			})
		}
	}

	database.DB.Exec(
		"INSERT INTO agent_reports (device_id, ip, mac, rx_bytes, tx_bytes, cpu_usage, memory_usage, disk_usage, temperature, report_time) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		req.DeviceID, req.IP, req.MAC, req.RxBytes, req.TxBytes, req.CPUUsage, req.MemoryUsage, req.DiskUsage, req.Temperature, now)

	c.JSON(http.StatusOK, gin.H{"message": "指标已接收"})
}
