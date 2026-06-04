package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/embedded-ops-platform/backend/internal/repository"
	"github.com/gin-gonic/gin"
)

// LogAPI handles device log endpoints.
type LogAPI struct {
	db *repository.PostgresRepo
}

// NewLogAPI creates a new log API handler.
func NewLogAPI(db *repository.PostgresRepo) *LogAPI {
	return &LogAPI{db: db}
}

// GetLogs returns logs for a device.
func (a *LogAPI) GetLogs(c *gin.Context) {
	deviceID := c.Param("device_id")
	logType := c.DefaultQuery("type", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if limit < 1 || limit > 500 {
		limit = 100
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	logs, err := a.db.GetDeviceLogs(ctx, deviceID, logType, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"limit": limit,
	})
}

// RegisterRoutes registers all log API routes.
func (a *LogAPI) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/devices/:device_id/logs", a.GetLogs)
}