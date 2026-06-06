package handler

import (
	"net/http"

	"github.com/aspira/crossborder-payment-gateway/internal/database"
	"github.com/aspira/crossborder-payment-gateway/internal/engine"
	"github.com/aspira/crossborder-payment-gateway/internal/models"
	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	db     database.DB
	engine *engine.EngineClient
}

func NewDashboardHandler(db database.DB, engineClient *engine.EngineClient) *DashboardHandler {
	return &DashboardHandler{db: db, engine: engineClient}
}

func (h *DashboardHandler) GetDashboard(c *gin.Context) {
	stats, err := h.db.GetDashboardStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	stats.EngineConnected = h.engine.IsEnabled() && h.engine.IsConnected()

	c.JSON(http.StatusOK, stats)
}

func (h *DashboardHandler) GetTPSHistory(c *gin.Context) {
	points, err := h.db.GetTPSHistory(60)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if points == nil {
		points = []models.TPSDataPoint{}
	}

	c.JSON(http.StatusOK, gin.H{"points": points})
}

func (h *DashboardHandler) GetRecentTransactions(c *gin.Context) {
	txns, err := h.db.GetRecentTransactions(50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if txns == nil {
		txns = []models.Transaction{}
	}

	c.JSON(http.StatusOK, gin.H{"transactions": txns})
}
