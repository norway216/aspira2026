package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/aspira/crossborder-payment-gateway/internal/database"
	"github.com/aspira/crossborder-payment-gateway/internal/engine"
	"github.com/aspira/crossborder-payment-gateway/internal/models"
	"github.com/aspira/crossborder-payment-gateway/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TransactionHandler struct {
	db     database.DB
	engine *engine.EngineClient
	wsHub  *websocket.Hub
}

func NewTransactionHandler(db database.DB, engineClient *engine.EngineClient, wsHub *websocket.Hub) *TransactionHandler {
	return &TransactionHandler{db: db, engine: engineClient, wsHub: wsHub}
}

func (h *TransactionHandler) CreateTransaction(c *gin.Context) {
	var req models.CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	merchantID, _ := c.Get("user_id")

	// Try external engine first
	if h.engine.IsEnabled() && h.engine.IsConnected() {
		payload := engine.TransactionPayload{
			TransactionID:  uuid.New().String(),
			MerchantID:     merchantID.(string),
			PayerAccountID: req.PayerAccountID,
			PayeeAccountID: req.PayeeAccountID,
			SourceCurrency: req.SourceCurrency,
			TargetCurrency: req.TargetCurrency,
			SourceAmount:   req.SourceAmount,
			Fee:            req.Fee,
			ReferenceID:    req.ReferenceID,
			Timestamp:      time.Now().UTC().Format(time.RFC3339),
		}

		payloadJSON, _ := json.Marshal(payload)
		engReq := engine.EngineRequest{
			ID:      uuid.New().String(),
			Type:    "process_transaction",
			Payload: payloadJSON,
		}

		resp, err := h.engine.Send(engReq)
		if err == nil && resp.Status == "ok" {
			var result engine.TransactionResult
			json.Unmarshal(resp.Payload, &result)

			txn := h.buildTransaction(result, req, merchantID.(string), payload.TransactionID)
			if err := h.db.CreateTransaction(txn); err != nil {
				log.Printf("Failed to save transaction: %v", err)
			}
			h.wsHub.BroadcastTransactionUpdate(txn)
			c.JSON(http.StatusOK, models.TransactionResponse{Transaction: *txn})
			return
		}
		log.Printf("Engine error: %v, falling back to internal processing", err)
	}

	// Internal processing (fallback)
	txn, err := h.processInternally(req, merchantID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.TransactionResponse{Transaction: *txn})
}

func (h *TransactionHandler) processInternally(req models.CreateTransactionRequest, merchantID string) (*models.Transaction, error) {
	// Validate payer account
	payerAcct, err := h.db.GetAccount(req.PayerAccountID)
	if err != nil {
		return nil, fmt.Errorf("payer account not found: %s", req.PayerAccountID)
	}
	if payerAcct.Status != "active" {
		return nil, fmt.Errorf("payer account is %s", payerAcct.Status)
	}

	// Validate payee account
	payeeAcct, err := h.db.GetAccount(req.PayeeAccountID)
	if err != nil {
		return nil, fmt.Errorf("payee account not found: %s", req.PayeeAccountID)
	}
	if payeeAcct.Status != "active" {
		return nil, fmt.Errorf("payee account is %s", payeeAcct.Status)
	}

	// Check balance
	totalDebit := req.SourceAmount
	if payerAcct.Balance-payerAcct.ReservedBalance < totalDebit {
		return nil, fmt.Errorf("insufficient funds: balance=%d, needed=%d", payerAcct.Balance, totalDebit)
	}

	// Get exchange rate
	rate, err := h.db.GetExchangeRate(req.SourceCurrency, req.TargetCurrency)
	if err != nil {
		if req.SourceCurrency == req.TargetCurrency {
			rate = &models.ExchangeRate{Rate: 1.0}
		} else {
			return nil, fmt.Errorf("exchange rate not found: %s->%s", req.SourceCurrency, req.TargetCurrency)
		}
	}

	// Calculate target amount
	targetAmount := int64(float64(req.SourceAmount-req.Fee) * rate.Rate)

	// Generate hash chain
	txnID := uuid.New().String()
	prevHash := h.getLastHash()
	hashInput := fmt.Sprintf("%s|%s|%d|%s|%d|%d", prevHash, txnID, req.SourceAmount, req.TargetCurrency, targetAmount, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(hashInput))
	currHash := hex.EncodeToString(hash[:])

	// Update balances atomically (simplified - in production use transactions)
	newPayerBalance := payerAcct.Balance - totalDebit
	if err := h.db.UpdateAccountBalance(req.PayerAccountID, newPayerBalance, payerAcct.ReservedBalance, payerAcct.DailyUsed+totalDebit, payerAcct.MonthlyUsed+totalDebit); err != nil {
		return nil, fmt.Errorf("failed to update payer balance: %w", err)
	}

	newPayeeBalance := payeeAcct.Balance + targetAmount
	if err := h.db.UpdateAccountBalance(req.PayeeAccountID, newPayeeBalance, payeeAcct.ReservedBalance, payeeAcct.DailyUsed, payeeAcct.MonthlyUsed); err != nil {
		// Rollback payer
		h.db.UpdateAccountBalance(req.PayerAccountID, payerAcct.Balance, payerAcct.ReservedBalance, payerAcct.DailyUsed, payerAcct.MonthlyUsed)
		return nil, fmt.Errorf("failed to update payee balance: %w", err)
	}

	now := time.Now()
	txn := &models.Transaction{
		ID:              txnID,
		MerchantID:      merchantID,
		PayerAccountID:  req.PayerAccountID,
		PayeeAccountID:  req.PayeeAccountID,
		SourceCurrency:  req.SourceCurrency,
		TargetCurrency:  req.TargetCurrency,
		SourceAmount:    req.SourceAmount,
		TargetAmount:    targetAmount,
		ExchangeRate:    rate.Rate,
		Fee:             req.Fee,
		Status:          models.TxnCompleted,
		Description:     req.Description,
		ReferenceID:     req.ReferenceID,
		CallbackURL:     req.CallbackURL,
		HashChainPrev:   prevHash,
		HashChainCurr:   currHash,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := h.db.CreateTransaction(txn); err != nil {
		return nil, fmt.Errorf("failed to save transaction: %w", err)
	}

	// Broadcast via WebSocket
	h.wsHub.BroadcastTransactionUpdate(txn)

	return txn, nil
}

func (h *TransactionHandler) getLastHash() string {
	txns, _, _ := h.db.ListTransactions(database.TransactionQuery{Page: 1, PageSize: 1})
	if len(txns) > 0 {
		return txns[0].HashChainCurr
	}
	return "0000000000000000000000000000000000000000000000000000000000000000"
}

func (h *TransactionHandler) buildTransaction(result engine.TransactionResult, req models.CreateTransactionRequest, merchantID, txnID string) *models.Transaction {
	now := time.Now()
	return &models.Transaction{
		ID:              txnID,
		MerchantID:      merchantID,
		PayerAccountID:  req.PayerAccountID,
		PayeeAccountID:  req.PayeeAccountID,
		SourceCurrency:  req.SourceCurrency,
		TargetCurrency:  req.TargetCurrency,
		SourceAmount:    req.SourceAmount,
		TargetAmount:    result.TargetAmount,
		ExchangeRate:    result.ExchangeRate,
		Fee:             result.Fee,
		Status:          models.TransactionStatus(result.Status),
		Description:     req.Description,
		ReferenceID:     req.ReferenceID,
		CallbackURL:     req.CallbackURL,
		HashChainPrev:   "",
		HashChainCurr:   result.HashChainCurr,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func (h *TransactionHandler) ListTransactions(c *gin.Context) {
	query := database.TransactionQuery{
		Status:     c.Query("status"),
		MerchantID: c.Query("merchant_id"),
		StartDate:  c.Query("start_date"),
		EndDate:    c.Query("end_date"),
		Search:     c.Query("search"),
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	query.Page = page
	query.PageSize = pageSize

	txns, total, err := h.db.ListTransactions(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if txns == nil {
		txns = []models.Transaction{}
	}

	c.JSON(http.StatusOK, models.TransactionListResponse{
		Transactions: txns,
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
	})
}

func (h *TransactionHandler) GetTransaction(c *gin.Context) {
	id := c.Param("id")
	txn, err := h.db.GetTransaction(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	c.JSON(http.StatusOK, models.TransactionResponse{Transaction: *txn})
}

func (h *TransactionHandler) RefundTransaction(c *gin.Context) {
	id := c.Param("id")
	txn, err := h.db.GetTransaction(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}

	if txn.Status != models.TxnCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only completed transactions can be refunded"})
		return
	}

	var req models.RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = "manual refund"
	}

	// Reverse the transaction
	now := time.Now()
	refundID := uuid.New().String()

	prevHash := h.getLastHash()
	hashInput := fmt.Sprintf("%s|%s|%d|%s|%d|%d", prevHash, refundID, txn.TargetAmount, txn.SourceCurrency, txn.SourceAmount, now.UnixNano())
	hash := sha256.Sum256([]byte(hashInput))
	currHash := hex.EncodeToString(hash[:])

	refund := &models.Transaction{
		ID:              refundID,
		MerchantID:      txn.MerchantID,
		PayerAccountID:  txn.PayeeAccountID,
		PayeeAccountID:  txn.PayerAccountID,
		SourceCurrency:  txn.TargetCurrency,
		TargetCurrency:  txn.SourceCurrency,
		SourceAmount:    txn.TargetAmount,
		TargetAmount:    txn.SourceAmount,
		ExchangeRate:    1.0 / txn.ExchangeRate,
		Fee:             0,
		Status:          models.TxnCompleted,
		Description:     fmt.Sprintf("Refund for %s: %s", id, req.Reason),
		ReferenceID:     fmt.Sprintf("REFUND-%s", id),
		CallbackURL:     txn.CallbackURL,
		HashChainPrev:   prevHash,
		HashChainCurr:   currHash,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := h.db.CreateTransaction(refund); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save refund"})
		return
	}

	// Update accounts
	payerAcct, _ := h.db.GetAccount(refund.PayerAccountID)
	if payerAcct != nil {
		h.db.UpdateAccountBalance(refund.PayerAccountID, payerAcct.Balance+refund.SourceAmount, payerAcct.ReservedBalance, payerAcct.DailyUsed, payerAcct.MonthlyUsed)
	}
	payeeAcct, _ := h.db.GetAccount(refund.PayeeAccountID)
	if payeeAcct != nil {
		h.db.UpdateAccountBalance(refund.PayeeAccountID, payeeAcct.Balance-refund.TargetAmount, payeeAcct.ReservedBalance, payeeAcct.DailyUsed, payeeAcct.MonthlyUsed)
	}

	// Mark original as refunded
	h.db.UpdateTransactionStatus(id, models.TxnRefunded)

	h.wsHub.BroadcastTransactionUpdate(refund)

	c.JSON(http.StatusOK, models.TransactionResponse{Transaction: *refund})
}

func (h *TransactionHandler) GetStats(c *gin.Context) {
	stats, err := h.db.GetDashboardStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
