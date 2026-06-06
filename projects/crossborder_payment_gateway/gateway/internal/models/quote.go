package models

import "time"

// QuoteStatus represents the lifecycle of a quote.
type QuoteStatus string

const (
	QuotePending   QuoteStatus = "pending"   // Created, awaiting acceptance
	QuoteAccepted  QuoteStatus = "accepted"  // Accepted by merchant, rate locked
	QuoteExpired   QuoteStatus = "expired"   // Validity period elapsed
	QuoteCancelled QuoteStatus = "cancelled" // Cancelled by merchant or system
	QuoteExecuted  QuoteStatus = "executed"  // Quote was used to create an order
)

// Quote represents an exchange rate quote provided to a merchant.
// Per architecture §5.3.1 QuoteCommitment contract.
type Quote struct {
	ID                 string      `json:"id" db:"id"`
	MerchantID         string      `json:"merchant_id" db:"merchant_id"`
	SourceCurrency     string      `json:"source_currency" db:"source_currency"`
	TargetCurrency     string      `json:"target_currency" db:"target_currency"`
	SourceAmount       int64       `json:"source_amount" db:"source_amount"`
	TargetAmount       int64       `json:"target_amount" db:"target_amount"`
	ExchangeRate       float64     `json:"exchange_rate" db:"exchange_rate"`
	Fee                int64       `json:"fee" db:"fee"`
	ProviderID         string      `json:"provider_id" db:"provider_id"`
	ExpiresAt          time.Time   `json:"expires_at" db:"expires_at"`
	Status             QuoteStatus `json:"status" db:"status"`
	RateCommitmentHash string      `json:"rate_commitment_hash" db:"rate_commitment_hash"`
	Signature          string      `json:"signature,omitempty" db:"signature"`
	CreatedAt          time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at" db:"updated_at"`
}

// CreateQuoteRequest is the merchant's request for a new quote.
type CreateQuoteRequest struct {
	SourceCurrency string `json:"source_currency" binding:"required"`
	TargetCurrency string `json:"target_currency" binding:"required"`
	SourceAmount   int64  `json:"source_amount" binding:"required"`
	MerchantID     string `json:"merchant_id"`
}

// QuoteResponse wraps a quote in the API response.
type QuoteResponse struct {
	Quote Quote `json:"quote"`
}

// QuoteListResponse is the API response for listing quotes.
type QuoteListResponse struct {
	Quotes   []Quote `json:"quotes"`
	Total    int64   `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
}

// AcceptQuoteRequest is the request to lock in a quote.
type AcceptQuoteRequest struct {
	OrderReference string `json:"order_reference"` // Optional reference for the resulting order
}
