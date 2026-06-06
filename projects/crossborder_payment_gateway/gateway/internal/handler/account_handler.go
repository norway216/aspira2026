package handler

import (
	"net/http"

	"github.com/aspira/crossborder-payment-gateway/internal/database"
	"github.com/aspira/crossborder-payment-gateway/internal/models"
	"github.com/gin-gonic/gin"
)

type AccountHandler struct {
	db database.DB
}

func NewAccountHandler(db database.DB) *AccountHandler {
	return &AccountHandler{db: db}
}

func (h *AccountHandler) ListAccounts(c *gin.Context) {
	merchantID := c.Query("merchant_id")
	role, _ := c.Get("role")

	// Non-admin users can only see their own accounts
	if role != "admin" {
		userID, _ := c.Get("user_id")
		merchantID = userID.(string)
	}

	accounts, err := h.db.ListAccounts(merchantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if accounts == nil {
		accounts = []models.Account{}
	}

	c.JSON(http.StatusOK, gin.H{"accounts": accounts})
}

func (h *AccountHandler) GetAccount(c *gin.Context) {
	id := c.Param("id")
	acct, err := h.db.GetAccount(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"account": acct})
}

func (h *AccountHandler) GetAccountLedger(c *gin.Context) {
	id := c.Param("id")

	// Return transactions where this account is payer or payee
	query := database.TransactionQuery{
		Page:     1,
		PageSize: 50,
	}
	txns, total, err := h.db.ListTransactions(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Filter transactions involving this account
	var ledger []gin.H
	for _, txn := range txns {
		if txn.PayerAccountID == id || txn.PayeeAccountID == id {
			entryType := "debit"
			amount := -txn.SourceAmount
			if txn.PayeeAccountID == id {
				entryType = "credit"
				amount = txn.TargetAmount
			}
			ledger = append(ledger, gin.H{
				"transaction_id": txn.ID,
				"amount":         amount,
				"type":           entryType,
				"status":         txn.Status,
				"created_at":     txn.CreatedAt,
			})
		}
	}

	if ledger == nil {
		ledger = make([]gin.H, 0)
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": ledger,
		"total":   total,
	})
}
