package handler

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/aspira/crossborder-payment-gateway/internal/database"
	"github.com/aspira/crossborder-payment-gateway/internal/models"
	"github.com/gin-gonic/gin"
)

// ReconciliationHandler handles reconciliation operations per architecture §13.
// Reconciliation compares internal order data against payment channel receipts
// and on-chain state to detect discrepancies.
type ReconciliationHandler struct {
	db database.DB
}

func NewReconciliationHandler(db database.DB) *ReconciliationHandler {
	return &ReconciliationHandler{db: db}
}

// GetReport handles GET /api/v1/reconciliation/report
// Returns a summary of reconciliation status.
func (h *ReconciliationHandler) GetReport(c *gin.Context) {
	matched, mismatched, pending, errors, err := h.db.GetReconciliationSummary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	records, total, err := h.db.GetReconciliationRecords(status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if records == nil {
		records = []models.ReconciliationRecord{}
	}

	c.JSON(http.StatusOK, models.ReconciliationReport{
		TotalChecked: total,
		Matched:      matched,
		Mismatched:   mismatched,
		Pending:      pending,
		Errors:       errors,
		Records:      records,
		RunAt:        time.Now(),
	})
}

// RunReconciliation handles POST /api/v1/reconciliation/run
// Triggers a manual reconciliation run comparing transactions against
// expected channel results (three-way reconciliation per architecture §13.1).
func (h *ReconciliationHandler) RunReconciliation(c *gin.Context) {
	var req models.ReconciliationRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Body is optional
	}

	log.Println("[Reconciliation] Starting manual reconciliation run...")

	// Get recent confirmed transactions for reconciliation
	txns, _, err := h.db.ListTransactions(database.TransactionQuery{
		Page:     1,
		PageSize: 200,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch transactions: " + err.Error()})
		return
	}

	checked := int64(0)
	matched := int64(0)
	mismatched := int64(0)

	for _, txn := range txns {
		// Skip non-settled transactions
		if txn.Status != models.StatusPaymentConfirmed &&
			txn.Status != models.StatusSettlementProofed &&
			txn.Status != models.StatusReconciled &&
			txn.Status != models.StatusClosed {
			continue
		}

		checked++

		// For three-way reconciliation per architecture §13.1:
		// Internal order amount vs expected channel amount
		// In production, this would also query the payment channel and blockchain state
		internalAmt := txn.TargetAmount
		channelAmt := txn.TargetAmount // In mock, assume channel matches

		matchStatus := models.RecMatched
		discrepancy := ""

		if internalAmt != channelAmt {
			matchStatus = models.RecMismatched
			discrepancy = "amount mismatch: internal=" + strconv.FormatInt(internalAmt, 10) +
				" vs channel=" + strconv.FormatInt(channelAmt, 10)
			mismatched++
		} else {
			matched++
		}

		rec := &models.ReconciliationRecord{
			OrderID:        txn.ID,
			TransactionID:  txn.ID,
			InternalAmount: internalAmt,
			ChannelAmount:  channelAmt,
			ChainTxnID:     "",
			Currency:       txn.TargetCurrency,
			ChannelName:    "mock",
			ChannelRef:     txn.ReferenceID,
			MatchStatus:    matchStatus,
			Discrepancy:    discrepancy,
			CreatedAt:      time.Now(),
		}

		if err := h.db.CreateReconciliationRecord(rec); err != nil {
			log.Printf("[Reconciliation] Failed to create record for %s: %v", txn.ID, err)
		}

		// If matched, transition to reconciled state
		if matchStatus == models.RecMatched && txn.Status == models.StatusPaymentConfirmed {
			if err := h.db.UpdateTransactionStatusValidated(txn.ID, models.StatusPaymentConfirmed, models.StatusSettlementProofed); err != nil {
				log.Printf("[Reconciliation] State transition failed for %s: %v", txn.ID, err)
			}
		}
	}

	log.Printf("[Reconciliation] Run complete: %d checked, %d matched, %d mismatched",
		checked, matched, mismatched)

	c.JSON(http.StatusOK, gin.H{
		"message":    "reconciliation run complete",
		"checked":    checked,
		"matched":    matched,
		"mismatched": mismatched,
	})
}

// GetDiscrepancies handles GET /api/v1/reconciliation/discrepancies
// Returns only mismatched reconciliation records.
func (h *ReconciliationHandler) GetDiscrepancies(c *gin.Context) {
	records, total, err := h.db.GetReconciliationRecords(string(models.RecMismatched), 1, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if records == nil {
		records = []models.ReconciliationRecord{}
	}

	c.JSON(http.StatusOK, gin.H{
		"discrepancies": records,
		"total":         total,
	})
}
