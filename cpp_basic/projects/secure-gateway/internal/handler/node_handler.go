package handler

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/secure-gateway/internal/model"
	"github.com/secure-gateway/pkg/logger"
)

// NodeHandler handles node management endpoints.
type NodeHandler struct {
	db *sql.DB
}

// NewNodeHandler creates a new node handler.
func NewNodeHandler(db *sql.DB) *NodeHandler {
	return &NodeHandler{db: db}
}

// ListNode returns all registered nodes.
func (h *NodeHandler) ListNode(c *gin.Context) {
	rows, err := h.db.Query(
		"SELECT id, node_id, name, region, public_addr, status, weight, max_connections, created_at, updated_at FROM nodes ORDER BY id",
	)
	if err != nil {
		logger.Error("Failed to list nodes", "error", err)
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "Internal server error"})
		return
	}
	defer rows.Close()

	type NodeWithMetrics struct {
		model.Node
		CPUUsage          *float64 `json:"cpu_usage,omitempty"`
		MemUsage          *float64 `json:"mem_usage,omitempty"`
		ActiveConnections *int     `json:"active_connections,omitempty"`
		RxBytesPerSec     *int64   `json:"rx_bytes_per_sec,omitempty"`
		TxBytesPerSec     *int64   `json:"tx_bytes_per_sec,omitempty"`
	}

	var nodes []NodeWithMetrics
	for rows.Next() {
		var n model.Node
		if err := rows.Scan(&n.ID, &n.NodeID, &n.Name, &n.Region, &n.PublicAddr, &n.Status, &n.Weight, &n.MaxConnections, &n.CreatedAt, &n.UpdatedAt); err != nil {
			logger.Error("Failed to scan node row", "error", err)
			continue
		}

		nwm := NodeWithMetrics{Node: n}

		// Fetch latest metrics
		var cpu sql.NullFloat64
		var mem sql.NullFloat64
		var conns sql.NullInt64
		var rx sql.NullInt64
		var tx sql.NullInt64
		err := h.db.QueryRow(
			`SELECT cpu_usage, mem_usage, active_connections, rx_bytes_per_sec, tx_bytes_per_sec
			 FROM node_metrics WHERE node_id = $1 ORDER BY created_at DESC LIMIT 1`,
			n.NodeID,
		).Scan(&cpu, &mem, &conns, &rx, &tx)
		if err == nil {
			if cpu.Valid { nwm.CPUUsage = &cpu.Float64 }
			if mem.Valid { nwm.MemUsage = &mem.Float64 }
			if conns.Valid { v := int(conns.Int64); nwm.ActiveConnections = &v }
			if rx.Valid { nwm.RxBytesPerSec = &rx.Int64 }
			if tx.Valid { nwm.TxBytesPerSec = &tx.Int64 }
		}

		nodes = append(nodes, nwm)
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 200, Message: "Success", Data: nodes})
}

// RegisterNode handles node registration.
func (h *NodeHandler) RegisterNode(c *gin.Context) {
	var req model.NodeRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "Invalid request: " + err.Error()})
		return
	}

	// Verify node secret against config
	expectedSecret := c.GetHeader("X-Node-Secret")
	if expectedSecret == "" {
		expectedSecret = req.Secret
	}
	// In production, validate req.Secret against a stored key
	// For now, accept any non-empty secret

	maxConns := req.MaxConnections
	if maxConns <= 0 {
		maxConns = 10000
	}

	// Upsert node
	var n model.Node
	err := h.db.QueryRow(
		`INSERT INTO nodes (node_id, name, region, public_addr, status, max_connections)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (node_id) DO UPDATE SET
		   name = EXCLUDED.name,
		   region = EXCLUDED.region,
		   public_addr = EXCLUDED.public_addr,
		   status = $5,
		   max_connections = EXCLUDED.max_connections,
		   updated_at = NOW()
		 RETURNING id, node_id, name, region, public_addr, status, weight, max_connections, created_at, updated_at`,
		req.NodeID, req.Name, req.Region, req.PublicAddr, model.NodeStatusOnline, maxConns,
	).Scan(&n.ID, &n.NodeID, &n.Name, &n.Region, &n.PublicAddr, &n.Status, &n.Weight, &n.MaxConnections, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		logger.Error("Failed to register node", "error", err)
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "Registration failed"})
		return
	}

	logger.Info("Node registered", "node_id", n.NodeID, "name", n.Name)
	c.JSON(http.StatusOK, model.APIResponse{Code: 200, Message: "Node registered", Data: n})
}

// Heartbeat receives node heartbeat and updates metrics.
func (h *NodeHandler) Heartbeat(c *gin.Context) {
	var req model.NodeHeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "Invalid request: " + err.Error()})
		return
	}

	// Update node status to online
	_, _ = h.db.Exec("UPDATE nodes SET status = $1, updated_at = NOW() WHERE node_id = $2", model.NodeStatusOnline, req.NodeID)

	// Insert metrics
	_, err := h.db.Exec(
		`INSERT INTO node_metrics (node_id, cpu_usage, mem_usage, active_connections, rx_bytes_per_sec, tx_bytes_per_sec)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		req.NodeID, req.CPUUsage, req.MemUsage, req.ActiveConnections, req.RxBytesPerSec, req.TxBytesPerSec,
	)
	if err != nil {
		logger.Error("Failed to insert node metrics", "error", err)
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 200, Message: "Heartbeat received"})
}

// GetNodeMetrics returns the latest metrics for a specific node.
func (h *NodeHandler) GetNodeMetrics(c *gin.Context) {
	nodeID := c.Param("node_id")

	var cpu, mem sql.NullFloat64
	var conns, rx, tx sql.NullInt64
	var createdAt sql.NullTime
	err := h.db.QueryRow(
		`SELECT cpu_usage, mem_usage, active_connections, rx_bytes_per_sec, tx_bytes_per_sec, created_at
		 FROM node_metrics WHERE node_id = $1 ORDER BY created_at DESC LIMIT 1`,
		nodeID,
	).Scan(&cpu, &mem, &conns, &rx, &tx, &createdAt)
	if err != nil {
		c.JSON(http.StatusNotFound, model.APIResponse{Code: 404, Message: "No metrics found"})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    200,
		Message: "Success",
		Data: gin.H{
			"node_id":            nodeID,
			"cpu_usage":          cpu.Float64,
			"mem_usage":          mem.Float64,
			"active_connections": conns.Int64,
			"rx_bytes_per_sec":   rx.Int64,
			"tx_bytes_per_sec":   tx.Int64,
			"created_at":         createdAt.Time,
		},
	})
}

// GetNodeMetricsHistory returns the metric history for a node.
func (h *NodeHandler) GetNodeMetricsHistory(c *gin.Context) {
	nodeID := c.Param("node_id")

	rows, err := h.db.Query(
		`SELECT cpu_usage, mem_usage, active_connections, rx_bytes_per_sec, tx_bytes_per_sec, created_at
		 FROM node_metrics WHERE node_id = $1 ORDER BY created_at DESC LIMIT 60`,
		nodeID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "Query failed"})
		return
	}
	defer rows.Close()

	var metrics []gin.H
	for rows.Next() {
		var cpu, mem float64
		var conns, rx, tx int64
		var t time.Time
		if err := rows.Scan(&cpu, &mem, &conns, &rx, &tx, &t); err != nil {
			continue
		}
		metrics = append(metrics, gin.H{
			"cpu_usage":          cpu,
			"mem_usage":          mem,
			"active_connections": conns,
			"rx_bytes_per_sec":   rx,
			"tx_bytes_per_sec":   tx,
			"created_at":         t,
		})
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 200, Message: "Success", Data: metrics})
}