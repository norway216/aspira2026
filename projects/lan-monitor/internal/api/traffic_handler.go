package api

import (
	"net/http"
	"strconv"
	"time"

	"lan-monitor/internal/database"

	"github.com/gin-gonic/gin"
)

func handleDeviceRealtimeTraffic(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	since := time.Now().Add(-5 * time.Minute)

	rows, _ := database.DB.Query(
		"SELECT rx_rate, tx_rate, total_rate, timestamp FROM traffic_records WHERE device_id = ? AND timestamp > ? ORDER BY timestamp ASC LIMIT 30",
		id, since)
	if rows != nil {
		defer rows.Close()
	}

	type TP struct {
		RxRate    float64   `json:"rx_rate"`
		TxRate    float64   `json:"tx_rate"`
		TotalRate float64   `json:"total_rate"`
		Timestamp time.Time `json:"timestamp"`
	}
	var data []TP
	var lastRx, lastTx float64
	if rows != nil {
		for rows.Next() {
			var t TP
			rows.Scan(&t.RxRate, &t.TxRate, &t.TotalRate, &t.Timestamp)
			data = append(data, t)
			lastRx, lastTx = t.RxRate, t.TxRate
		}
	}
	if data == nil {
		data = []TP{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":            data,
		"current_rx_rate": lastRx,
		"current_tx_rate": lastTx,
	})
}

func handleDeviceTrafficHistory(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	duration := c.DefaultQuery("duration", "1h")

	var since time.Time
	switch duration {
	case "15m":
		since = time.Now().Add(-15 * time.Minute)
	case "30m":
		since = time.Now().Add(-30 * time.Minute)
	case "1h":
		since = time.Now().Add(-1 * time.Hour)
	case "6h":
		since = time.Now().Add(-6 * time.Hour)
	case "24h":
		since = time.Now().Add(-24 * time.Hour)
	default:
		since = time.Now().Add(-1 * time.Hour)
	}

	rows, _ := database.DB.Query(
		"SELECT rx_rate, tx_rate, total_rate, timestamp FROM traffic_records WHERE device_id = ? AND timestamp > ? ORDER BY timestamp ASC",
		id, since)
	if rows != nil {
		defer rows.Close()
	}

	type TP struct {
		RxRate    float64   `json:"rx_rate"`
		TxRate    float64   `json:"tx_rate"`
		TotalRate float64   `json:"total_rate"`
		Timestamp time.Time `json:"timestamp"`
	}
	var data []TP
	if rows != nil {
		for rows.Next() {
			var t TP
			rows.Scan(&t.RxRate, &t.TxRate, &t.TotalRate, &t.Timestamp)
			data = append(data, t)
		}
	}
	if data == nil {
		data = []TP{}
	}
	c.JSON(http.StatusOK, gin.H{"data": data, "duration": duration})
}

func handleTrafficRank(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	since := time.Now().Add(-5 * time.Minute)

	rows, _ := database.DB.Query(`
		SELECT t.device_id, d.ip, d.mac, d.hostname, COALESCE(AVG(t.total_rate), 0) as total_rate
		FROM traffic_records t JOIN devices d ON d.id = t.device_id
		WHERE t.timestamp > ? GROUP BY t.device_id ORDER BY total_rate DESC LIMIT ?`, since, limit)
	if rows != nil {
		defer rows.Close()
	}

	type Rank struct {
		DeviceID  uint    `json:"device_id"`
		IP        string  `json:"ip"`
		MAC       string  `json:"mac"`
		Hostname  string  `json:"hostname"`
		TotalRate float64 `json:"total_rate"`
	}
	var rank []Rank
	if rows != nil {
		for rows.Next() {
			var r Rank
			rows.Scan(&r.DeviceID, &r.IP, &r.MAC, &r.Hostname, &r.TotalRate)
			rank = append(rank, r)
		}
	}
	if rank == nil {
		rank = []Rank{}
	}
	c.JSON(http.StatusOK, gin.H{"data": rank})
}

func handleTrafficSummary(c *gin.Context) {
	since := time.Now().Add(-5 * time.Minute)
	var avgRx, avgTx float64
	database.DB.QueryRow(
		"SELECT COALESCE(AVG(rx_rate), 0), COALESCE(AVG(tx_rate), 0) FROM system_traffics WHERE timestamp > ?", since,
	).Scan(&avgRx, &avgTx)

	var devicesWithTraffic int64
	database.DB.QueryRow(
		"SELECT COUNT(DISTINCT device_id) FROM traffic_records WHERE timestamp > ?", since,
	).Scan(&devicesWithTraffic)

	var peakRx, peakTx float64
	database.DB.QueryRow(
		"SELECT COALESCE(MAX(rx_rate), 0), COALESCE(MAX(tx_rate), 0) FROM system_traffics WHERE timestamp > ?", since,
	).Scan(&peakRx, &peakTx)

	c.JSON(http.StatusOK, gin.H{
		"avg_rx_rate":          avgRx,
		"avg_tx_rate":          avgTx,
		"peak_rx_rate":         peakRx,
		"peak_tx_rate":         peakTx,
		"devices_with_traffic": devicesWithTraffic,
	})
}
