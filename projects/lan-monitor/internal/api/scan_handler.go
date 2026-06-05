package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"lan-monitor/internal/database"
	"lan-monitor/internal/scanner"

	"github.com/gin-gonic/gin"
)

func handleScanStart(c *gin.Context, scannerSvc *scanner.ScannerService) {
	if scannerSvc.IsRunning() {
		c.JSON(http.StatusConflict, gin.H{"error": "扫描正在进行中"})
		return
	}
	go scannerSvc.RunScan()
	c.JSON(http.StatusOK, gin.H{"message": "扫描任务已启动"})
}

func handleScanTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	var total int64
	database.DB.QueryRow("SELECT COUNT(*) FROM scan_tasks").Scan(&total)

	offset := (page - 1) * size
	rows, _ := database.DB.Query(
		"SELECT id, subnet, scan_type, status, started_at, finished_at, total_ips, online_count, error_msg FROM scan_tasks ORDER BY id DESC LIMIT ? OFFSET ?",
		size, offset)
	if rows != nil {
		defer rows.Close()
	}

	type ST struct {
		ID          uint       `json:"id"`
		Subnet      string     `json:"subnet"`
		ScanType    string     `json:"scan_type"`
		Status      string     `json:"status"`
		StartedAt   *time.Time `json:"started_at"`
		FinishedAt  *time.Time `json:"finished_at"`
		TotalIPs    int        `json:"total_ips"`
		OnlineCount int        `json:"online_count"`
		ErrorMsg    string     `json:"error_message"`
	}
	var tasks []ST
	if rows != nil {
		for rows.Next() {
			var t ST
			rows.Scan(&t.ID, &t.Subnet, &t.ScanType, &t.Status, &t.StartedAt, &t.FinishedAt, &t.TotalIPs, &t.OnlineCount, &t.ErrorMsg)
			tasks = append(tasks, t)
		}
	}
	if tasks == nil {
		tasks = []ST{}
	}
	c.JSON(http.StatusOK, gin.H{"data": tasks, "total": total, "page": page})
}

func handleScanTaskDetail(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var t struct {
		ID          uint       `json:"id"`
		Subnet      string     `json:"subnet"`
		ScanType    string     `json:"scan_type"`
		Status      string     `json:"status"`
		StartedAt   *time.Time `json:"started_at"`
		FinishedAt  *time.Time `json:"finished_at"`
		TotalIPs    int        `json:"total_ips"`
		OnlineCount int        `json:"online_count"`
		ErrorMsg    string     `json:"error_message"`
	}
	err := database.DB.QueryRow(
		"SELECT id, subnet, scan_type, status, started_at, finished_at, total_ips, online_count, error_msg FROM scan_tasks WHERE id = ?", id,
	).Scan(&t.ID, &t.Subnet, &t.ScanType, &t.Status, &t.StartedAt, &t.FinishedAt, &t.TotalIPs, &t.OnlineCount, &t.ErrorMsg)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	c.JSON(http.StatusOK, t)
}

func handleGetScanConfig(c *gin.Context) {
	var cfg struct {
		ID               uint   `json:"id"`
		Subnets          string `json:"subnets"`
		Methods          string `json:"methods"`
		IntervalSeconds  int    `json:"interval_seconds"`
		WorkerCount      int    `json:"worker_count"`
		TimeoutMs        int    `json:"timeout_ms"`
		OfflineThreshold int    `json:"offline_threshold"`
		TCPPorts         string `json:"tcp_ports"`
		Enabled          bool   `json:"enabled"`
	}
	database.DB.QueryRow(
		"SELECT id, subnets, methods, interval_seconds, worker_count, timeout_ms, offline_threshold, tcp_ports, enabled FROM scan_configs LIMIT 1",
	).Scan(&cfg.ID, &cfg.Subnets, &cfg.Methods, &cfg.IntervalSeconds, &cfg.WorkerCount, &cfg.TimeoutMs, &cfg.OfflineThreshold, &cfg.TCPPorts, &cfg.Enabled)
	c.JSON(http.StatusOK, cfg)
}

type UpdateScanConfigRequest struct {
	Subnets          *string `json:"subnets"`
	Methods          *string `json:"methods"`
	IntervalSeconds  *int    `json:"interval_seconds"`
	WorkerCount      *int    `json:"worker_count"`
	TimeoutMs        *int    `json:"timeout_ms"`
	OfflineThreshold *int    `json:"offline_threshold"`
	TCPPorts         *string `json:"tcp_ports"`
	Enabled          *bool   `json:"enabled"`
}

func handleUpdateScanConfig(c *gin.Context) {
	var req UpdateScanConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求数据格式错误"})
		return
	}

	sets := []string{}
	args := []interface{}{}
	if req.Subnets != nil {
		var s []string
		if json.Unmarshal([]byte(*req.Subnets), &s) == nil {
			sets = append(sets, "subnets = ?")
			args = append(args, *req.Subnets)
		}
	}
	if req.Methods != nil {
		var m []string
		if json.Unmarshal([]byte(*req.Methods), &m) == nil {
			sets = append(sets, "methods = ?")
			args = append(args, *req.Methods)
		}
	}
	if req.IntervalSeconds != nil {
		sets = append(sets, "interval_seconds = ?")
		args = append(args, *req.IntervalSeconds)
	}
	if req.WorkerCount != nil {
		sets = append(sets, "worker_count = ?")
		args = append(args, *req.WorkerCount)
	}
	if req.TimeoutMs != nil {
		sets = append(sets, "timeout_ms = ?")
		args = append(args, *req.TimeoutMs)
	}
	if req.OfflineThreshold != nil {
		sets = append(sets, "offline_threshold = ?")
		args = append(args, *req.OfflineThreshold)
	}
	if req.TCPPorts != nil {
		var p []int
		if json.Unmarshal([]byte(*req.TCPPorts), &p) == nil {
			sets = append(sets, "tcp_ports = ?")
			args = append(args, *req.TCPPorts)
		}
	}
	if req.Enabled != nil {
		enabledVal := 0
		if *req.Enabled {
			enabledVal = 1
		}
		sets = append(sets, "enabled = ?")
		args = append(args, enabledVal)
	}

	if len(sets) > 0 {
		database.DB.Exec("UPDATE scan_configs SET "+strings.Join(sets, ", "), args...)
	}
	c.JSON(http.StatusOK, gin.H{"message": "扫描配置已更新"})
}

func handleGetSubnets(c *gin.Context, scannerSvc *scanner.ScannerService) {
	subnets, err := scannerSvc.GetLocalSubnets()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取网段失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"subnets": subnets})
}
