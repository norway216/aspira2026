package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/secure-gateway/internal/model"
)

// DashboardHandler handles dashboard statistics endpoints.
type DashboardHandler struct {
	db *sql.DB
}

// NewDashboardHandler creates a new dashboard handler.
func NewDashboardHandler(db *sql.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

// GetStats returns aggregated dashboard statistics.
func (h *DashboardHandler) GetStats(c *gin.Context) {
	stats := &model.DashboardStats{}

	// Count online nodes
	h.db.QueryRow("SELECT COUNT(*) FROM nodes WHERE status = 'online'").Scan(&stats.OnlineNodes)

	// Total active connections (from latest metrics)
	var totalConns sql.NullInt64
	h.db.QueryRow(
		`SELECT COALESCE(SUM(active_connections), 0) FROM node_metrics n1
		 WHERE created_at = (SELECT MAX(created_at) FROM node_metrics n2 WHERE n2.node_id = n1.node_id)`,
	).Scan(&totalConns)
	stats.TotalConnections = int(totalConns.Int64)

	// Today's traffic
	var todayBytes sql.NullInt64
	h.db.QueryRow(
		"SELECT COALESCE(SUM(rx_bytes + tx_bytes), 0) FROM traffic_records WHERE created_at >= CURRENT_DATE",
	).Scan(&todayBytes)
	stats.TodayTrafficBytes = todayBytes.Int64

	// Average CPU and memory
	var avgCPU, avgMem sql.NullFloat64
	h.db.QueryRow(
		`SELECT COALESCE(AVG(cpu_usage), 0), COALESCE(AVG(mem_usage), 0)
		 FROM node_metrics n1
		 WHERE created_at = (SELECT MAX(created_at) FROM node_metrics n2 WHERE n2.node_id = n1.node_id)`,
	).Scan(&avgCPU, &avgMem)
	stats.AvgCPUUsage = avgCPU.Float64
	stats.AvgMemUsage = avgMem.Float64

	// Alert count (nodes offline for more than 30 seconds, simplified: just count offline nodes)
	var alertCount sql.NullInt64
	h.db.QueryRow(
		"SELECT COUNT(*) FROM nodes WHERE status = 'offline'",
	).Scan(&alertCount)
	stats.AlertCount = int(alertCount.Int64)

	c.JSON(http.StatusOK, model.APIResponse{Code: 200, Message: "Success", Data: stats})
}

// GetNodeDistribution returns node count by region.
func (h *DashboardHandler) GetNodeDistribution(c *gin.Context) {
	rows, err := h.db.Query("SELECT region, COUNT(*) as count FROM nodes GROUP BY region")
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "Query failed"})
		return
	}
	defer rows.Close()

	var distribution []gin.H
	for rows.Next() {
		var region string
		var count int
		if err := rows.Scan(&region, &count); err != nil {
			continue
		}
		distribution = append(distribution, gin.H{"region": region, "count": count})
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 200, Message: "Success", Data: distribution})
}

// GetTrafficHistory returns traffic data points for charts.
func (h *DashboardHandler) GetTrafficHistory(c *gin.Context) {
	rows, err := h.db.Query(
		`SELECT DATE_TRUNC('hour', created_at) as ts, COALESCE(SUM(rx_bytes + tx_bytes), 0) as total
		 FROM traffic_records
		 WHERE created_at >= NOW() - INTERVAL '24 hours'
		 GROUP BY ts
		 ORDER BY ts`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "Query failed"})
		return
	}
	defer rows.Close()

	var data []gin.H
	for rows.Next() {
		var ts sql.NullTime
		var total sql.NullInt64
		if err := rows.Scan(&ts, &total); err != nil {
			continue
		}
		data = append(data, gin.H{
			"timestamp": ts.Time,
			"bytes":     total.Int64,
		})
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 200, Message: "Success", Data: data})
}

// GetTopUsers returns the top traffic-consuming users.
func (h *DashboardHandler) GetTopUsers(c *gin.Context) {
	rows, err := h.db.Query(
		`SELECT u.id, u.username, COALESCE(SUM(tr.rx_bytes + tr.tx_bytes), 0) as total_bytes
		 FROM users u
		 LEFT JOIN traffic_records tr ON tr.user_id = u.id
		 GROUP BY u.id, u.username
		 ORDER BY total_bytes DESC
		 LIMIT 10`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "Query failed"})
		return
	}
	defer rows.Close()

	var users []gin.H
	for rows.Next() {
		var id int64
		var username string
		var bytes int64
		if err := rows.Scan(&id, &username, &bytes); err != nil {
			continue
		}
		users = append(users, gin.H{"id": id, "username": username, "total_bytes": bytes})
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 200, Message: "Success", Data: users})
}