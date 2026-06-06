package contracts

import (
	"fmt"
	"sync"
	"time"
)

// SettlementProof records a payment settlement on-chain.
// Per architecture §7.4: stores receipt hash (not the receipt itself),
// payment channel name, and confirmation timestamp.
type SettlementProof struct {
	OrderID        string `json:"order_id"`
	ChannelName    string `json:"channel_name"`
	ReceiptHash    string `json:"receipt_hash"`
	ReceiptURIHash string `json:"receipt_uri_hash"`
	Signer         string `json:"signer"`
	ConfirmedAt    int64  `json:"confirmed_at"`
}

// SettlementProofRegistry stores payment settlement proofs on-chain.
// Equivalent to the Solidity SettlementProofRegistry in architecture §7.4.
// The actual receipt documents are stored in encrypted object storage (MinIO/S3);
// only the hash is recorded on chain for verification.
type SettlementProofRegistry struct {
	mu      sync.RWMutex
	proofs  map[string]*SettlementProof // orderID → proof
	eventCh chan SettlementEvent
}

// SettlementEvent is emitted when a settlement proof is registered.
type SettlementEvent struct {
	OrderID     string
	ChannelName string
}

// NewSettlementProofRegistry creates a new registry.
func NewSettlementProofRegistry() *SettlementProofRegistry {
	return &SettlementProofRegistry{
		proofs:  make(map[string]*SettlementProof),
		eventCh: make(chan SettlementEvent, 256),
	}
}

// RegisterProof stores a settlement proof on chain.
func (r *SettlementProofRegistry) RegisterProof(orderID, channelName, receiptHash, receiptURIHash, signer string) (*SettlementProof, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.proofs[orderID]; exists {
		return nil, fmt.Errorf("SETTLEMENT_ALREADY_REGISTERED: %s", orderID)
	}

	proof := &SettlementProof{
		OrderID:        orderID,
		ChannelName:    channelName,
		ReceiptHash:    receiptHash,
		ReceiptURIHash: receiptURIHash,
		Signer:         signer,
		ConfirmedAt:    time.Now().Unix(),
	}

	r.proofs[orderID] = proof

	select {
	case r.eventCh <- SettlementEvent{OrderID: orderID, ChannelName: channelName}:
	default:
	}

	return proof, nil
}

// GetProof retrieves a settlement proof from chain.
func (r *SettlementProofRegistry) GetProof(orderID string) (*SettlementProof, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	proof, exists := r.proofs[orderID]
	if !exists {
		return nil, fmt.Errorf("settlement proof not found: %s", orderID)
	}
	return proof, nil
}

// VerifyProof checks that the provided receipt hash matches the on-chain record.
func (r *SettlementProofRegistry) VerifyProof(orderID, receiptHash string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	proof, exists := r.proofs[orderID]
	if !exists {
		return false, fmt.Errorf("settlement proof not found: %s", orderID)
	}

	return proof.ReceiptHash == receiptHash, nil
}

// IsSettled checks if an order has a settlement proof on chain.
func (r *SettlementProofRegistry) IsSettled(orderID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.proofs[orderID]
	return exists
}

// Events returns the event channel.
func (r *SettlementProofRegistry) Events() <-chan SettlementEvent {
	return r.eventCh
}
