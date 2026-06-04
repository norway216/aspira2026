package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/secure-gateway/internal/model"
)

// TrafficHandler handles traffic stat endpoints.
type TrafficHandler struct {
	db *sql.DB
}

// NewTrafficHandler creates a new traffic handler.
func NewTrafficHandler(db *sql.DB) *TrafficHandler {
	return &TrafficHandler{db: db}
}

// ListTraffic returns traffic records with optional filters.
func (h *TrafficHandler) ListTraffic(c *gin.Context) {
	userIDStr := c.Query("user_id")
	limitStr := c.DefaultQuery("limit", "50")

	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var rows *sql.Rows
	var err error

	if userIDStr != "" {
		userID, _ := strconv.ParseInt(userIDStr, 10, 64)
		rows, err = h.db.Query(
			`SELECT id, user_id, node_id, rx_bytes, tx_bytes, duration_seconds, created_at
			 FROM traffic_records WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`,
			userID, limit,
		)
	} else {
		rows, err = h.db.Query(
			`SELECT id, user_id, node_id, rx_bytes, tx_bytes, duration_seconds, created_at
			 FROM traffic_records ORDER BY created_at DESC LIMIT $1`,
			limit,
		)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "Query failed"})
		return
	}
	defer rows.Close()

	var records []model.TrafficRecord
	for rows.Next() {
		var r model.TrafficRecord
		if err := rows.Scan(&r.ID, &r.UserID, &r.NodeID, &r.RxBytes, &r.TxBytes, &r.DurationSeconds, &r.CreatedAt); err != nil {
			continue
		}
		records = append(records, r)
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 200, Message: "Success", Data: records})
}

// GetTrafficSummary returns aggregated traffic data.
func (h *TrafficHandler) GetTrafficSummary(c *gin.Context) {
	var totalRx, totalTx sql.NullInt64
	var totalDuration sql.NullInt64

	h.db.QueryRow("SELECT COALESCE(SUM(rx_bytes), 0), COALESCE(SUM(tx_bytes), 0), COALESCE(SUM(duration_seconds), 0) FROM traffic_records").Scan(&totalRx, &totalTx, &totalDuration)

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    200,
		Message: "Success",
		Data: gin.H{
			"total_rx_bytes":    totalRx.Int64,
			"total_tx_bytes":    totalTx.Int64,
			"total_duration_seconds": totalDuration.Int64,
		},
	})
}