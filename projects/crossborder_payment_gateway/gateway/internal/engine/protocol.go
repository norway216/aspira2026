package engine

import "encoding/json"

type EngineRequest struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type EngineResponse struct {
	ID      string          `json:"id"`
	Status  string          `json:"status"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   *EngineError    `json:"error,omitempty"`
}

type EngineError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retriable bool   `json:"retriable"`
}

type TransactionPayload struct {
	TransactionID   string `json:"transaction_id"`
	MerchantID      string `json:"merchant_id"`
	PayerAccountID  string `json:"payer_account_id"`
	PayeeAccountID  string `json:"payee_account_id"`
	SourceCurrency  string `json:"source_currency"`
	TargetCurrency  string `json:"target_currency"`
	SourceAmount    int64  `json:"source_amount"`
	Fee             int64  `json:"fee"`
	ReferenceID     string `json:"reference_id"`
	Timestamp       string `json:"timestamp"`
}

type TransactionResult struct {
	TransactionID  string  `json:"transaction_id"`
	TargetAmount   int64   `json:"target_amount"`
	ExchangeRate   float64 `json:"exchange_rate"`
	Fee            int64   `json:"fee"`
	Status         string  `json:"status"`
	HashChainCurr  string  `json:"hash_chain_current"`
	ProcessedAt    string  `json:"processed_at"`
}
