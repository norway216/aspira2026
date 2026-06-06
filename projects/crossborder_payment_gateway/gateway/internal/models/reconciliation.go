package models

import "time"

// ReconciliationStatus represents the match state of a reconciliation record.
type ReconciliationStatus string

const (
	RecPending    ReconciliationStatus = "pending"    // Awaiting reconciliation
	RecMatched    ReconciliationStatus = "matched"    // All three sources agree
	RecMismatched ReconciliationStatus = "mismatched" // Discrepancy found
	RecError      ReconciliationStatus = "error"      // Reconciliation processing error
)

// ReconciliationRecord stores a single reconciliation entry per architecture §13.
// It compares three data sources: internal order, payment channel receipt, and
// on-chain state (when available).
type ReconciliationRecord struct {
	ID             int64                 `json:"id" db:"id"`
	OrderID        string                `json:"order_id" db:"order_id"`
	TransactionID  string                `json:"transaction_id" db:"transaction_id"`
	InternalAmount int64                 `json:"internal_amount" db:"internal_amount"`
	ChannelAmount  int64                 `json:"channel_amount" db:"channel_amount"`
	ChainTxnID     string                `json:"chain_txn_id" db:"chain_txn_id"`
	Currency       string                `json:"currency" db:"currency"`
	ChannelName    string                `json:"channel_name" db:"channel_name"`
	ChannelRef     string                `json:"channel_ref" db:"channel_ref"`
	MatchStatus    ReconciliationStatus  `json:"match_status" db:"match_status"`
	Discrepancy    string                `json:"discrepancy" db:"discrepancy"`
	ResolvedAt     *time.Time            `json:"resolved_at,omitempty" db:"resolved_at"`
	CreatedAt      time.Time             `json:"created_at" db:"created_at"`
}

// ReconciliationReport summarizes the results of a reconciliation run.
type ReconciliationReport struct {
	TotalChecked   int64                  `json:"total_checked"`
	Matched        int64                  `json:"matched"`
	Mismatched     int64                  `json:"mismatched"`
	Pending        int64                  `json:"pending"`
	Errors         int64                  `json:"errors"`
	Records        []ReconciliationRecord `json:"records"`
	RunAt          time.Time              `json:"run_at"`
}

// ReconciliationRunRequest triggers a reconciliation run.
type ReconciliationRunRequest struct {
	StartDate string `json:"start_date"` // Optional: filter by date range
	EndDate   string `json:"end_date"`
}
