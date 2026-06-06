package models

import "time"

type TransactionStatus string

const (
	TxnPending    TransactionStatus = "pending"
	TxnProcessing TransactionStatus = "processing"
	TxnCompleted  TransactionStatus = "completed"
	TxnFailed     TransactionStatus = "failed"
	TxnRefunded   TransactionStatus = "refunded"
)

type Transaction struct {
	ID              string            `json:"id" db:"id"`
	MerchantID      string            `json:"merchant_id" db:"merchant_id"`
	PayerAccountID  string            `json:"payer_account_id" db:"payer_account_id"`
	PayeeAccountID  string            `json:"payee_account_id" db:"payee_account_id"`
	SourceCurrency  string            `json:"source_currency" db:"source_currency"`
	TargetCurrency  string            `json:"target_currency" db:"target_currency"`
	SourceAmount    int64             `json:"source_amount" db:"source_amount"`
	TargetAmount    int64             `json:"target_amount" db:"target_amount"`
	ExchangeRate    float64           `json:"exchange_rate" db:"exchange_rate"`
	Fee             int64             `json:"fee" db:"fee"`
	Status          TransactionStatus `json:"status" db:"status"`
	Description     string            `json:"description" db:"description"`
	ReferenceID     string            `json:"reference_id" db:"reference_id"`
	CallbackURL     string            `json:"callback_url" db:"callback_url"`
	HashChainPrev   string            `json:"hash_chain_prev" db:"hash_chain_prev"`
	HashChainCurr   string            `json:"hash_chain_curr" db:"hash_chain_curr"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
}

type CreateTransactionRequest struct {
	MerchantID      string `json:"merchant_id"`
	PayerAccountID  string `json:"payer_account_id"`
	PayeeAccountID  string `json:"payee_account_id"`
	SourceCurrency  string `json:"source_currency"`
	TargetCurrency  string `json:"target_currency"`
	SourceAmount    int64  `json:"source_amount"`
	Fee             int64  `json:"fee"`
	Description     string `json:"description"`
	ReferenceID     string `json:"reference_id"`
	CallbackURL     string `json:"callback_url"`
}

type TransactionResponse struct {
	Transaction Transaction `json:"transaction"`
}

type TransactionListResponse struct {
	Transactions []Transaction `json:"transactions"`
	Total        int64         `json:"total"`
	Page         int           `json:"page"`
	PageSize     int           `json:"page_size"`
}

type TransactionStats struct {
	TotalCount    int64   `json:"total_count"`
	TodayCount    int64   `json:"today_count"`
	TodayVolume   int64   `json:"today_volume"`
	SuccessRate   float64 `json:"success_rate"`
	PendingCount  int64   `json:"pending_count"`
	FailedCount   int64   `json:"failed_count"`
	RefundedCount int64   `json:"refunded_count"`
}

type RefundRequest struct {
	Reason string `json:"reason"`
}
