package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"lan-monitor/internal/database"

	"github.com/gin-gonic/gin"
)

type DeviceRow struct {
	ID                    uint       `json:"id"`
	IP                    string     `json:"ip"`
	MAC                   string     `json:"mac"`
	Hostname              string     `json:"hostname"`
	Vendor                string     `json:"vendor"`
	DeviceType            string     `json:"device_type"`
	Label                 string     `json:"label"`
	Owner                 string     `json:"owner"`
	Department            string     `json:"department"`
	Status                string     `json:"status"`
	FirstSeen             time.Time  `json:"first_seen"`
	LastSeen              time.Time  `json:"last_seen"`
	LastOnlineAt          *time.Time `json:"last_online_at"`
	LastOfflineAt         *time.Time `json:"last_offline_at"`
	OnlineDurationSeconds int64      `json:"online_duration_seconds"`
	OfflineCount          int        `json:"offline_count"`
	FailCount             int        `json:"fail_count"`
	ConsecutiveFailures   int        `json:"consecutive_failures"`
	RiskLevel             string     `json:"risk_level"`
	Note                  string     `json:"note"`
	OpenPorts             string     `json:"open_ports"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func scanDevice(row interface{ Scan(...interface{}) error }, d *DeviceRow) error {
	return row.Scan(&d.ID, &d.IP, &d.MAC, &d.Hostname, &d.Vendor, &d.DeviceType,
		&d.Label, &d.Owner, &d.Department, &d.Status, &d.FirstSeen, &d.LastSeen,
		&d.LastOnlineAt, &d.LastOfflineAt, &d.OnlineDurationSeconds, &d.OfflineCount,
		&d.FailCount, &d.ConsecutiveFailures, &d.RiskLevel, &d.Note, &d.OpenPorts,
		&d.CreatedAt, &d.UpdatedAt)
}

func handleDeviceList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	status := c.Query("status")
	search := c.Query("search")
	deviceType := c.Query("type")

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
	if deviceType != "" {
		where += " AND device_type = ?"
		args = append(args, deviceType)
	}
	if search != "" {
		like := "%" + search + "%"
		where += " AND (ip LIKE ? OR mac LIKE ? OR hostname LIKE ? OR label LIKE ?)"
		args = append(args, like, like, like, like)
	}

	var total int64
	database.DB.QueryRow("SELECT COUNT(*) FROM devices "+where, args...).Scan(&total)

	offset := (page - 1) * size
	query := fmt.Sprintf("SELECT id, ip, mac, hostname, vendor, device_type, label, owner, department, status, first_seen, last_seen, last_online_at, last_offline_at, online_duration_seconds, offline_count, fail_count, consecutive_failures, risk_level, note, open_ports, created_at, updated_at FROM devices %s ORDER BY last_seen DESC LIMIT ? OFFSET ?", where)
	args = append(args, size, offset)

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	var devices []DeviceRow
	for rows.Next() {
		var d DeviceRow
		if err := scanDevice(rows, &d); err == nil {
			devices = append(devices, d)
		}
	}
	if devices == nil {
		devices = []DeviceRow{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  devices,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

func handleOnlineDevices(c *gin.Context) {
	rows, err := database.DB.Query("SELECT id, ip, mac, hostname, vendor, device_type, label, owner, department, status, first_seen, last_seen, last_online_at, last_offline_at, online_duration_seconds, offline_count, fail_count, consecutive_failures, risk_level, note, open_ports, created_at, updated_at FROM devices WHERE status = 'online' ORDER BY last_seen DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	var devices []DeviceRow
	for rows.Next() {
		var d DeviceRow
		if scanDevice(rows, &d) == nil {
			devices = append(devices, d)
		}
	}
	if devices == nil {
		devices = []DeviceRow{}
	}
	c.JSON(http.StatusOK, gin.H{"data": devices, "total": len(devices)})
}

func handleDeviceDetail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的设备ID"})
		return
	}

	var d DeviceRow
	row := database.DB.QueryRow(
		"SELECT id, ip, mac, hostname, vendor, device_type, label, owner, department, status, first_seen, last_seen, last_online_at, last_offline_at, online_duration_seconds, offline_count, fail_count, consecutive_failures, risk_level, note, open_ports, created_at, updated_at FROM devices WHERE id = ?", id)
	if err := scanDevice(row, &d); err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "设备不存在"})
		return
	}

	// Events
	type EventRow struct {
		ID          uint      `json:"id"`
		DeviceID    uint      `json:"device_id"`
		DeviceMAC   string    `json:"device_mac"`
		EventType   string    `json:"event_type"`
		OldStatus   string    `json:"old_status"`
		NewStatus   string    `json:"new_status"`
		EventTime   time.Time `json:"event_time"`
		Description string    `json:"description"`
	}
	var events []EventRow
	eRows, _ := database.DB.Query("SELECT id, device_id, device_mac, event_type, old_status, new_status, event_time, description FROM device_events WHERE device_id = ? ORDER BY event_time DESC LIMIT 50", id)
	if eRows != nil {
		defer eRows.Close()
		for eRows.Next() {
			var e EventRow
			eRows.Scan(&e.ID, &e.DeviceID, &e.DeviceMAC, &e.EventType, &e.OldStatus, &e.NewStatus, &e.EventTime, &e.Description)
			events = append(events, e)
		}
	}
	if events == nil {
		events = []EventRow{}
	}

	// Traffic (last 5 min)
	type TrafficRow struct {
		ID        uint      `json:"id"`
		DeviceID  uint      `json:"device_id"`
		RxRate    float64   `json:"rx_rate"`
		TxRate    float64   `json:"tx_rate"`
		TotalRate float64   `json:"total_rate"`
		Timestamp time.Time `json:"timestamp"`
	}
	var traffic []TrafficRow
	since := time.Now().Add(-5 * time.Minute)
	tRows, _ := database.DB.Query("SELECT id, device_id, rx_rate, tx_rate, total_rate, timestamp FROM traffic_records WHERE device_id = ? AND timestamp > ? ORDER BY timestamp ASC LIMIT 60", id, since)
	if tRows != nil {
		defer tRows.Close()
		for tRows.Next() {
			var t TrafficRow
			tRows.Scan(&t.ID, &t.DeviceID, &t.RxRate, &t.TxRate, &t.TotalRate, &t.Timestamp)
			traffic = append(traffic, t)
		}
	}
	if traffic == nil {
		traffic = []TrafficRow{}
	}

	// Alerts
	type AlertRow struct {
		ID        uint      `json:"id"`
		Level     string    `json:"level"`
		Title     string    `json:"title"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"created_at"`
	}
	var alerts []AlertRow
	aRows, _ := database.DB.Query("SELECT id, level, title, status, created_at FROM alerts WHERE device_id = ? ORDER BY created_at DESC LIMIT 10", id)
	if aRows != nil {
		defer aRows.Close()
		for aRows.Next() {
			var a AlertRow
			aRows.Scan(&a.ID, &a.Level, &a.Title, &a.Status, &a.CreatedAt)
			alerts = append(alerts, a)
		}
	}
	if alerts == nil {
		alerts = []AlertRow{}
	}

	// Sessions
	type SessionRow struct {
		ID          uint       `json:"id"`
		OnlineAt    time.Time  `json:"online_at"`
		OfflineAt   *time.Time `json:"offline_at"`
		DurationSec int64      `json:"duration_sec"`
	}
	var sessions []SessionRow
	sRows, _ := database.DB.Query("SELECT id, online_at, offline_at, duration_sec FROM online_sessions WHERE device_id = ? ORDER BY online_at DESC LIMIT 20", id)
	if sRows != nil {
		defer sRows.Close()
		for sRows.Next() {
			var s SessionRow
			sRows.Scan(&s.ID, &s.OnlineAt, &s.OfflineAt, &s.DurationSec)
			sessions = append(sessions, s)
		}
	}
	if sessions == nil {
		sessions = []SessionRow{}
	}

	c.JSON(http.StatusOK, gin.H{
		"device":   d,
		"events":   events,
		"traffic":  traffic,
		"alerts":   alerts,
		"sessions": sessions,
	})
}

type UpdateDeviceRequest struct {
	Hostname   *string `json:"hostname"`
	Label      *string `json:"label"`
	Owner      *string `json:"owner"`
	Department *string `json:"department"`
	DeviceType *string `json:"device_type"`
	RiskLevel  *string `json:"risk_level"`
	Note       *string `json:"note"`
}

func handleDeviceUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的设备ID"})
		return
	}

	var exists int64
	database.DB.QueryRow("SELECT COUNT(*) FROM devices WHERE id = ?", id).Scan(&exists)
	if exists == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "设备不存在"})
		return
	}

	var req UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求数据格式错误"})
		return
	}

	sets := []string{}
	args := []interface{}{}
	if req.Hostname != nil {
		sets = append(sets, "hostname = ?")
		args = append(args, *req.Hostname)
	}
	if req.Label != nil {
		sets = append(sets, "label = ?")
		args = append(args, *req.Label)
	}
	if req.Owner != nil {
		sets = append(sets, "owner = ?")
		args = append(args, *req.Owner)
	}
	if req.Department != nil {
		sets = append(sets, "department = ?")
		args = append(args, *req.Department)
	}
	if req.DeviceType != nil {
		sets = append(sets, "device_type = ?")
		args = append(args, *req.DeviceType)
	}
	if req.RiskLevel != nil {
		sets = append(sets, "risk_level = ?")
		args = append(args, *req.RiskLevel)
	}
	if req.Note != nil {
		sets = append(sets, "note = ?")
		args = append(args, *req.Note)
	}

	if len(sets) > 0 {
		sets = append(sets, "updated_at = ?")
		args = append(args, time.Now())
		query := "UPDATE devices SET " + strings.Join(sets, ", ") + " WHERE id = ?"
		args = append(args, id)
		database.DB.Exec(query, args...)

		// Log event
		database.DB.Exec("INSERT INTO device_events (device_id, event_type, event_time, description) VALUES (?, 'updated', ?, '设备信息已更新')", id, time.Now())
	}

	// Return updated device
	var d DeviceRow
	row := database.DB.QueryRow("SELECT id, ip, mac, hostname, vendor, device_type, label, owner, department, status, first_seen, last_seen, last_online_at, last_offline_at, online_duration_seconds, offline_count, fail_count, consecutive_failures, risk_level, note, open_ports, created_at, updated_at FROM devices WHERE id = ?", id)
	scanDevice(row, &d)

	c.JSON(http.StatusOK, gin.H{"message": "更新成功", "device": d})
}

func handleDeviceDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	database.DB.Exec("DELETE FROM device_events WHERE device_id = ?", id)
	database.DB.Exec("DELETE FROM traffic_records WHERE device_id = ?", id)
	database.DB.Exec("DELETE FROM alerts WHERE device_id = ?", id)
	database.DB.Exec("DELETE FROM online_sessions WHERE device_id = ?", id)
	database.DB.Exec("DELETE FROM devices WHERE id = ?", id)
	c.JSON(http.StatusOK, gin.H{"message": "设备已删除"})
}

func handleDeviceIgnore(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	database.DB.Exec("UPDATE devices SET status = 'ignored' WHERE id = ?", id)
	c.JSON(http.StatusOK, gin.H{"message": "设备已忽略"})
}

func handleDeviceLabel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Label string `json:"label" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入标签"})
		return
	}
	database.DB.Exec("UPDATE devices SET label = ? WHERE id = ?", req.Label, id)
	c.JSON(http.StatusOK, gin.H{"message": "标签已更新"})
}

func handleDeviceEvents(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "50"))

	var total int64
	database.DB.QueryRow("SELECT COUNT(*) FROM device_events WHERE device_id = ?", id).Scan(&total)

	offset := (page - 1) * size
	rows, _ := database.DB.Query("SELECT id, device_id, device_mac, event_type, old_status, new_status, event_time, description FROM device_events WHERE device_id = ? ORDER BY event_time DESC LIMIT ? OFFSET ?", id, size, offset)
	if rows != nil {
		defer rows.Close()
	}

	type ER struct {
		ID          uint      `json:"id"`
		DeviceID    uint      `json:"device_id"`
		DeviceMAC   string    `json:"device_mac"`
		EventType   string    `json:"event_type"`
		OldStatus   string    `json:"old_status"`
		NewStatus   string    `json:"new_status"`
		EventTime   time.Time `json:"event_time"`
		Description string    `json:"description"`
	}
	var events []ER
	if rows != nil {
		for rows.Next() {
			var e ER
			rows.Scan(&e.ID, &e.DeviceID, &e.DeviceMAC, &e.EventType, &e.OldStatus, &e.NewStatus, &e.EventTime, &e.Description)
			events = append(events, e)
		}
	}
	if events == nil {
		events = []ER{}
	}
	c.JSON(http.StatusOK, gin.H{"data": events, "total": total, "page": page})
}

func handleDeviceTraffic(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	duration := c.DefaultQuery("duration", "1h")
	var since time.Time
	switch duration {
	case "5m":
		since = time.Now().Add(-5 * time.Minute)
	case "15m":
		since = time.Now().Add(-15 * time.Minute)
	case "30m":
		since = time.Now().Add(-30 * time.Minute)
	case "6h":
		since = time.Now().Add(-6 * time.Hour)
	case "24h":
		since = time.Now().Add(-24 * time.Hour)
	default:
		since = time.Now().Add(-1 * time.Hour)
	}

	rows, _ := database.DB.Query("SELECT id, device_id, rx_rate, tx_rate, total_rate, timestamp FROM traffic_records WHERE device_id = ? AND timestamp > ? ORDER BY timestamp ASC", id, since)
	if rows != nil {
		defer rows.Close()
	}
	type TR struct {
		ID        uint      `json:"id"`
		DeviceID  uint      `json:"device_id"`
		RxRate    float64   `json:"rx_rate"`
		TxRate    float64   `json:"tx_rate"`
		TotalRate float64   `json:"total_rate"`
		Timestamp time.Time `json:"timestamp"`
	}
	var traffic []TR
	if rows != nil {
		for rows.Next() {
			var t TR
			rows.Scan(&t.ID, &t.DeviceID, &t.RxRate, &t.TxRate, &t.TotalRate, &t.Timestamp)
			traffic = append(traffic, t)
		}
	}
	if traffic == nil {
		traffic = []TR{}
	}
	c.JSON(http.StatusOK, gin.H{"data": traffic, "duration": duration})
}
