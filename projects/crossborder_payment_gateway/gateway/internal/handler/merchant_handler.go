package handler

import (
	"net/http"
	"time"

	"github.com/aspira/crossborder-payment-gateway/internal/database"
	"github.com/aspira/crossborder-payment-gateway/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MerchantHandler struct {
	db database.DB
}

func NewMerchantHandler(db database.DB) *MerchantHandler {
	return &MerchantHandler{db: db}
}

func (h *MerchantHandler) ListMerchants(c *gin.Context) {
	merchants, err := h.db.ListMerchants()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if merchants == nil {
		merchants = []models.Merchant{}
	}
	c.JSON(http.StatusOK, gin.H{"merchants": merchants})
}

func (h *MerchantHandler) CreateMerchant(c *gin.Context) {
	var req models.CreateMerchantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.DailyLimit == 0 {
		req.DailyLimit = 100000000
	}
	if req.MonthlyLimit == 0 {
		req.MonthlyLimit = 1000000000
	}

	now := time.Now()
	merchant := &models.Merchant{
		ID:           "merchant-" + uuid.New().String()[:8],
		Name:         req.Name,
		Status:       "active",
		DailyLimit:   req.DailyLimit,
		MonthlyLimit: req.MonthlyLimit,
		CallbackURL:  req.CallbackURL,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.db.CreateMerchant(merchant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"merchant": merchant})
}

func (h *MerchantHandler) UpdateMerchant(c *gin.Context) {
	id := c.Param("id")

	existing, err := h.db.GetMerchant(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "merchant not found"})
		return
	}

	var req models.UpdateMerchantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Status != "" {
		existing.Status = req.Status
	}
	if req.DailyLimit > 0 {
		existing.DailyLimit = req.DailyLimit
	}
	if req.MonthlyLimit > 0 {
		existing.MonthlyLimit = req.MonthlyLimit
	}
	if req.CallbackURL != "" {
		existing.CallbackURL = req.CallbackURL
	}
	existing.UpdatedAt = time.Now()

	if err := h.db.UpdateMerchant(id, existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"merchant": existing})
}

func (h *MerchantHandler) DeleteMerchant(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.DeleteMerchant(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "merchant deactivated"})
}
