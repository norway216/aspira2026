package models

import "time"

type Account struct {
	ID              string    `json:"id" db:"id"`
	MerchantID      string    `json:"merchant_id" db:"merchant_id"`
	Currency        string    `json:"currency" db:"currency"`
	Balance         int64     `json:"balance" db:"balance"`
	ReservedBalance int64     `json:"reserved_balance" db:"reserved_balance"`
	Status          string    `json:"status" db:"status"`
	DailyLimit      int64     `json:"daily_limit" db:"daily_limit"`
	DailyUsed       int64     `json:"daily_used" db:"daily_used"`
	MonthlyLimit    int64     `json:"monthly_limit" db:"monthly_limit"`
	MonthlyUsed     int64     `json:"monthly_used" db:"monthly_used"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

type AccountLedgerEntry struct {
	TransactionID string    `json:"transaction_id"`
	Amount        int64     `json:"amount"`
	Type          string    `json:"type"` // credit, debit
	BalanceAfter  int64     `json:"balance_after"`
	CreatedAt     time.Time `json:"created_at"`
}
