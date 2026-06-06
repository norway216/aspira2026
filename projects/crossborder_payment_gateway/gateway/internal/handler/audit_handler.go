package handler

import (
	"net/http"
	"strconv"

	"github.com/aspira/crossborder-payment-gateway/internal/database"
	"github.com/aspira/crossborder-payment-gateway/internal/models"
	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	db database.DB
}

func NewAuditHandler(db database.DB) *AuditHandler {
	return &AuditHandler{db: db}
}

func (h *AuditHandler) ListAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	query := database.AuditLogQuery{
		Action:    c.Query("action"),
		Resource:  c.Query("resource"),
		UserID:    c.Query("user_id"),
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
		Page:      page,
		PageSize:  pageSize,
	}

	logs, total, err := h.db.ListAuditLogs(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if logs == nil {
		logs = []models.AuditLog{}
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
