package handler

import (
	"net/http"
	"time"

	"github.com/aspira/crossborder-payment-gateway/internal/database"
	"github.com/aspira/crossborder-payment-gateway/internal/exchange"
	"github.com/aspira/crossborder-payment-gateway/internal/models"
	"github.com/gin-gonic/gin"
)

type ExchangeHandler struct {
	db          database.DB
	rateService *exchange.RateService
}

func NewExchangeHandler(db database.DB, rateService *exchange.RateService) *ExchangeHandler {
	return &ExchangeHandler{db: db, rateService: rateService}
}

func (h *ExchangeHandler) ListRates(c *gin.Context) {
	rates, err := h.db.GetExchangeRates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rates == nil {
		rates = []models.ExchangeRate{}
	}
	c.JSON(http.StatusOK, gin.H{"rates": rates})
}

func (h *ExchangeHandler) UpsertRate(c *gin.Context) {
	var req models.UpsertExchangeRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	rate := &models.ExchangeRate{
		Source:     req.Source,
		Target:     req.Target,
		Rate:       req.Rate,
		Bid:        req.Bid,
		Ask:        req.Ask,
		SourceName: req.SourceName,
		CreatedAt:  time.Now(),
	}

	if rate.Bid == 0 {
		rate.Bid = rate.Rate * 0.999
	}
	if rate.Ask == 0 {
		rate.Ask = rate.Rate * 1.001
	}

	if err := h.db.UpsertExchangeRate(rate); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rate": rate})
}

// GetLiveRates returns the cached live exchange rates.
func (h *ExchangeHandler) GetLiveRates(c *gin.Context) {
	rates := h.rateService.GetCachedRates()
	cacheAge := h.rateService.GetCacheAge()
	c.JSON(http.StatusOK, gin.H{
		"rates":      rates,
		"base":       "USD",
		"count":      len(rates),
		"cache_age":  cacheAge.String(),
		"cached_at":  time.Now().Add(-cacheAge).Format(time.RFC3339),
	})
}

// RefreshRates forces a refresh of live exchange rates from the API.
func (h *ExchangeHandler) RefreshRates(c *gin.Context) {
	rates, err := h.rateService.FetchRates()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "rates refreshed",
		"count":   len(rates),
	})
}
