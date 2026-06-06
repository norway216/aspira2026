package contracts

import (
	"fmt"
	"sync"
	"time"
)

// QuoteCommitmentRecord stores a locked quote on-chain.
// Per architecture §7.2: prevents quote providers from reneging on a quoted rate.
type QuoteCommitmentRecord struct {
	QuoteID          string `json:"quote_id"`
	ProviderHash     string `json:"provider_hash"`
	SrcCurrencyHash  string `json:"src_currency_hash"`
	TgtCurrencyHash  string `json:"tgt_currency_hash"`
	RateCommitment   string `json:"rate_commitment"`
	FeeCommitment    string `json:"fee_commitment"`
	ExpireAt         int64  `json:"expire_at"`
	Signature        string `json:"signature"`
	CreatedAt        int64  `json:"created_at"`
}

// QuoteCommitment is the on-chain contract for locking exchange rate quotes.
// Equivalent to the Solidity QuoteCommitment contract in architecture §7.2.
type QuoteCommitment struct {
	mu      sync.RWMutex
	quotes  map[string]*QuoteCommitmentRecord
	eventCh chan QuoteEvent
}

// QuoteEvent is emitted when a quote is committed on chain.
type QuoteEvent struct {
	QuoteID string
	Type    string // "committed", "expired"
}

// NewQuoteCommitment creates a new QuoteCommitment contract.
func NewQuoteCommitment() *QuoteCommitment {
	return &QuoteCommitment{
		quotes:  make(map[string]*QuoteCommitmentRecord),
		eventCh: make(chan QuoteEvent, 256),
	}
}

// CommitQuote locks a quote on-chain, preventing the provider from reneging.
// Per §7.2: stores rate_commitment, fee_commitment, and expiry time.
func (qc *QuoteCommitment) CommitQuote(quoteID, providerID string, srcCurrency, tgtCurrency string, rate float64, fee int64, expireAt int64, signature string) (*QuoteCommitmentRecord, error) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	if _, exists := qc.quotes[quoteID]; exists {
		return nil, fmt.Errorf("QUOTE_ALREADY_COMMITTED: %s", quoteID)
	}

	now := time.Now().Unix()

	// Hash sensitive fields (§10.2)
	providerHash := CommitHash(providerID + fmt.Sprintf("%d", now))
	srcCurrencyHash := CommitHash(srcCurrency + fmt.Sprintf("%d", now))
	tgtCurrencyHash := CommitHash(tgtCurrency + fmt.Sprintf("%d", now))
	rateCommitment := CommitHash(fmt.Sprintf("%.6f|%d|%d", rate, now, fee))
	feeCommitment := CommitHash(fmt.Sprintf("%d|%d", fee, now))

	record := &QuoteCommitmentRecord{
		QuoteID:         quoteID,
		ProviderHash:    providerHash,
		SrcCurrencyHash: srcCurrencyHash,
		TgtCurrencyHash: tgtCurrencyHash,
		RateCommitment:  rateCommitment,
		FeeCommitment:   feeCommitment,
		ExpireAt:        expireAt,
		Signature:       signature,
		CreatedAt:       now,
	}

	qc.quotes[quoteID] = record

	select {
	case qc.eventCh <- QuoteEvent{QuoteID: quoteID, Type: "committed"}:
	default:
	}

	return record, nil
}

// GetQuote retrieves a committed quote from chain.
func (qc *QuoteCommitment) GetQuote(quoteID string) (*QuoteCommitmentRecord, error) {
	qc.mu.RLock()
	defer qc.mu.RUnlock()

	record, exists := qc.quotes[quoteID]
	if !exists {
		return nil, fmt.Errorf("quote not found on chain: %s", quoteID)
	}
	return record, nil
}

// IsExpired checks if a quote has expired.
func (qc *QuoteCommitment) IsExpired(quoteID string) (bool, error) {
	qc.mu.RLock()
	defer qc.mu.RUnlock()

	record, exists := qc.quotes[quoteID]
	if !exists {
		return true, fmt.Errorf("quote not found: %s", quoteID)
	}

	return time.Now().Unix() > record.ExpireAt, nil
}

// IsQuoteCommitted checks if a quote is already on chain.
func (qc *QuoteCommitment) IsQuoteCommitted(quoteID string) bool {
	qc.mu.RLock()
	defer qc.mu.RUnlock()
	_, exists := qc.quotes[quoteID]
	return exists
}

// Events returns the event channel.
func (qc *QuoteCommitment) Events() <-chan QuoteEvent {
	return qc.eventCh
}
