package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/embedded-ops-platform/backend/internal/repository"
	"github.com/gin-gonic/gin"
)

// AlertAPI handles alert endpoints.
type AlertAPI struct {
	db *repository.PostgresRepo
}

// NewAlertAPI creates a new alert API handler.
func NewAlertAPI(db *repository.PostgresRepo) *AlertAPI {
	return &AlertAPI{db: db}
}

// ListAlerts returns active alerts.
func (a *AlertAPI) ListAlerts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	alerts, total, err := a.db.ListAlerts(ctx, offset, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts":    alerts,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ResolveAlert resolves an alert.
func (a *AlertAPI) ResolveAlert(c *gin.Context) {
	alertID, err := strconv.ParseInt(c.Param("alert_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert ID"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := a.db.ResolveAlert(ctx, alertID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "resolved"})
}

// RegisterRoutes registers all alert API routes.
func (a *AlertAPI) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/alerts", a.ListAlerts)
	rg.PUT("/alerts/:alert_id/resolve", a.ResolveAlert)
}