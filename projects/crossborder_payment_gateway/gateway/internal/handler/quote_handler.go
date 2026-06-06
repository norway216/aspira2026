package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/aspira/crossborder-payment-gateway/internal/database"
	"github.com/aspira/crossborder-payment-gateway/internal/exchange"
	"github.com/aspira/crossborder-payment-gateway/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// QuoteHandler handles quote requests per architecture §5.1 and §7.2.
// Quotes are exchange rate commitments that merchants can lock before
// creating a payment order. The rate commitment hash prevents quote
// providers from reneging on a quote.
type QuoteHandler struct {
	db          database.DB
	rateService *exchange.RateService
}

func NewQuoteHandler(db database.DB, rateService *exchange.RateService) *QuoteHandler {
	return &QuoteHandler{db: db, rateService: rateService}
}

// CreateQuote handles POST /api/v1/quote
// Generates a quote with exchange rate, fee, and validity period.
// The rate commitment hash binds the provider to this specific rate.
func (h *QuoteHandler) CreateQuote(c *gin.Context) {
	var req models.CreateQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Get merchant ID from context (JWT) or request
	merchantID, _ := c.Get("user_id")
	if req.MerchantID != "" {
		merchantID = req.MerchantID
	}
	if merchantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "merchant_id is required"})
		return
	}

	// Compute exchange rate using the two-hop USD conversion
	usdAmount, targetAmount, srcToUsd, usdToTgt, convErr :=
		h.rateService.Convert(req.SourceAmount, 0, req.SourceCurrency, req.TargetCurrency)
	if convErr != nil {
		// Fallback: try direct DB rate
		rate, err := h.db.GetExchangeRate(req.SourceCurrency, req.TargetCurrency)
		if err != nil {
			if req.SourceCurrency == req.TargetCurrency {
				rate = &models.ExchangeRate{Rate: 1.0}
			} else {
				c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "cannot compute exchange rate: " + convErr.Error()})
				return
			}
		}
		targetAmount = int64(float64(req.SourceAmount) * rate.Rate)
		_ = usdAmount
		_ = srcToUsd
		_ = usdToTgt
	}

	// Calculate fee (0.5% for demo, configurable in production)
	fee := int64(float64(req.SourceAmount) * 0.005)
	effectiveRate := float64(targetAmount) / float64(req.SourceAmount-fee)
	if req.SourceAmount-fee <= 0 {
		effectiveRate = 0
	}

	// Compute rate commitment hash per architecture §7.2 and §10.2
	// commitment = Hash(provider_id + source_currency + target_currency + rate + amount + timestamp + salt)
	salt := uuid.New().String()[:8]
	commitInput := fmt.Sprintf("%s|%s|%s|%.6f|%d|%d|%s",
		"aspira-core", req.SourceCurrency, req.TargetCurrency,
		effectiveRate, req.SourceAmount, time.Now().Unix(), salt)
	commitHash := sha256.Sum256([]byte(commitInput))

	now := time.Now()
	expiresAt := now.Add(5 * time.Minute) // 5-minute quote validity

	quote := &models.Quote{
		ID:                 fmt.Sprintf("QTE-%s-%d-%s", req.SourceCurrency, now.Unix(), uuid.New().String()[:8]),
		MerchantID:         merchantID.(string),
		SourceCurrency:     req.SourceCurrency,
		TargetCurrency:     req.TargetCurrency,
		SourceAmount:       req.SourceAmount,
		TargetAmount:       targetAmount,
		ExchangeRate:       effectiveRate,
		Fee:                fee,
		ProviderID:         "aspira-core",
		ExpiresAt:          expiresAt,
		Status:             models.QuotePending,
		RateCommitmentHash: hex.EncodeToString(commitHash[:]),
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := h.db.CreateQuote(quote); err != nil {
		log.Printf("[Quote] Failed to create quote: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create quote"})
		return
	}

	log.Printf("[Quote] Created quote %s: %s %d %s → %d %s (rate=%.6f, expires=%s)",
		quote.ID, req.SourceCurrency, req.SourceAmount, req.TargetCurrency,
		targetAmount, req.TargetCurrency, effectiveRate, expiresAt.Format(time.RFC3339))

	c.JSON(http.StatusOK, models.QuoteResponse{Quote: *quote})
}

// GetQuote handles GET /api/v1/quote/:id
func (h *QuoteHandler) GetQuote(c *gin.Context) {
	id := c.Param("id")
	quote, err := h.db.GetQuote(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "quote not found"})
		return
	}

	// Check expiry
	if quote.Status == models.QuotePending && time.Now().After(quote.ExpiresAt) {
		_ = h.db.UpdateQuoteStatus(id, models.QuoteExpired)
		quote.Status = models.QuoteExpired
	}

	c.JSON(http.StatusOK, models.QuoteResponse{Quote: *quote})
}

// AcceptQuote handles POST /api/v1/quote/:id/accept
// Locks the quote so it can be used to create an order.
func (h *QuoteHandler) AcceptQuote(c *gin.Context) {
	id := c.Param("id")

	var req models.AcceptQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Body is optional for accept
	}

	quote, err := h.db.GetQuote(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "quote not found"})
		return
	}

	if quote.Status != models.QuotePending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quote is not in pending status: " + string(quote.Status)})
		return
	}

	if time.Now().After(quote.ExpiresAt) {
		_ = h.db.UpdateQuoteStatus(id, models.QuoteExpired)
		c.JSON(http.StatusGone, gin.H{"error": "quote has expired"})
		return
	}

	if err := h.db.AcceptQuote(id, req.OrderReference); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "failed to accept quote (may already be accepted): " + err.Error()})
		return
	}

	quote.Status = models.QuoteAccepted
	log.Printf("[Quote] Quote %s accepted by merchant %s", id, quote.MerchantID)

	c.JSON(http.StatusOK, gin.H{
		"message": "quote accepted",
		"quote":   quote,
	})
}

// ListQuotes handles GET /api/v1/quotes
func (h *QuoteHandler) ListQuotes(c *gin.Context) {
	merchantID, _ := c.Get("user_id")
	if mid := c.Query("merchant_id"); mid != "" {
		merchantID = mid
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	quotes, total, err := h.db.ListQuotes(
		func() string {
			if mid, ok := merchantID.(string); ok {
				return mid
			}
			return ""
		}(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if quotes == nil {
		quotes = []models.Quote{}
	}

	// Auto-expire stale quotes in the result
	now := time.Now()
	for i := range quotes {
		if quotes[i].Status == models.QuotePending && now.After(quotes[i].ExpiresAt) {
			quotes[i].Status = models.QuoteExpired
		}
	}

	c.JSON(http.StatusOK, models.QuoteListResponse{
		Quotes:   quotes,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// CancelQuote handles POST /api/v1/quote/:id/cancel
func (h *QuoteHandler) CancelQuote(c *gin.Context) {
	id := c.Param("id")

	quote, err := h.db.GetQuote(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "quote not found"})
		return
	}

	if quote.Status != models.QuotePending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only pending quotes can be cancelled"})
		return
	}

	if err := h.db.UpdateQuoteStatus(id, models.QuoteCancelled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel quote"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "quote cancelled"})
}
