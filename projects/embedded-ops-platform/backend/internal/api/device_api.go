package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/embedded-ops-platform/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// DeviceAPI handles device management endpoints.
type DeviceAPI struct {
	deviceService *service.DeviceService
}

// NewDeviceAPI creates a new device API handler.
func NewDeviceAPI(deviceService *service.DeviceService) *DeviceAPI {
	return &DeviceAPI{deviceService: deviceService}
}

// ListDevices returns a paginated list of devices.
func (a *DeviceAPI) ListDevices(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	devices, total, err := a.deviceService.ListDevices(ctx, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	onlineCount, _ := a.deviceService.GetOnlineCount(ctx)
	totalCount, _ := a.deviceService.GetTotalCount(ctx)

	c.JSON(http.StatusOK, gin.H{
		"devices":      devices,
		"total":        total,
		"page":         page,
		"page_size":    pageSize,
		"online_count": onlineCount,
		"total_count":  totalCount,
	})
}

// GetDevice returns detailed info for a single device.
func (a *DeviceAPI) GetDevice(c *gin.Context) {
	deviceID := c.Param("device_id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	device, metrics, err := a.deviceService.GetDevice(ctx, deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device":  device,
		"metrics": metrics,
	})
}

// GetDeviceMetrics returns metric history for a device.
func (a *DeviceAPI) GetDeviceMetrics(c *gin.Context) {
	deviceID := c.Param("device_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "60"))

	// In a full implementation, this would query the metric store
	c.JSON(http.StatusOK, gin.H{
		"device_id": deviceID,
		"metrics":   []interface{}{},
		"limit":     limit,
	})
}

// GetDashboardStats returns aggregated statistics for the dashboard.
func (a *DeviceAPI) GetDashboardStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	onlineCount, _ := a.deviceService.GetOnlineCount(ctx)
	totalCount, _ := a.deviceService.GetTotalCount(ctx)
	offlineCount := 0
	if totalCount > onlineCount {
		offlineCount = totalCount - onlineCount
	}

	c.JSON(http.StatusOK, gin.H{
		"online_count":  onlineCount,
		"offline_count": offlineCount,
		"total_count":   totalCount,
	})
}

// RegisterRoutes registers all device API routes.
func (a *DeviceAPI) RegisterRoutes(rg *gin.RouterGroup) {
	devices := rg.Group("/devices")
	{
		devices.GET("", a.ListDevices)
		devices.GET("/stats", a.GetDashboardStats)
		devices.GET("/:device_id", a.GetDevice)
		devices.GET("/:device_id/metrics", a.GetDeviceMetrics)
	}
}

