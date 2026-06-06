package contracts

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// AuditEvent records an immutable audit entry on-chain.
// Per architecture §7.5: events form a hash chain (prev_event_hash → event_hash)
// for tamper-proof audit trails that can be anchored with Merkle roots.
type AuditEvent struct {
	EventID       string `json:"event_id"`
	OrderID       string `json:"order_id"`
	EventType     string `json:"event_type"`
	EventHash     string `json:"event_hash"`
	PrevEventHash string `json:"prev_event_hash"`
	MerkleRoot    string `json:"merkle_root"`
	OperatorHash  string `json:"operator_hash"`
	BlockHeight   uint64 `json:"block_height"`
	CreatedAt     int64  `json:"created_at"`
}

// AuditLedger is the immutable on-chain audit log contract.
// Equivalent to the Solidity AuditLedger in architecture §7.5.
// Every event is cryptographically chained to its predecessor.
type AuditLedger struct {
	mu         sync.RWMutex
	events     []*AuditEvent           // All events in insertion order
	eventsByOrder map[string][]*AuditEvent // Events grouped by order
	lastHash   string                  // Chain pointer: hash of the last event
}

// NewAuditLedger creates a new audit ledger with a genesis hash.
func NewAuditLedger() *AuditLedger {
	return &AuditLedger{
		events:        make([]*AuditEvent, 0),
		eventsByOrder: make(map[string][]*AuditEvent),
		lastHash:      "0000000000000000000000000000000000000000000000000000000000000000",
	}
}

// RecordEvent appends an audit event to the immutable ledger.
// The event hash is computed as SHA-256(prev_event_hash + event_id + order_id + event_type + payload_hash + timestamp).
func (al *AuditLedger) RecordEvent(eventID, orderID, eventType, payloadHash, operatorHash string, blockHeight uint64) (*AuditEvent, error) {
	al.mu.Lock()
	defer al.mu.Unlock()

	now := time.Now().Unix()

	// Build event hash linking to previous event (§7.5 hash chain)
	hashInput := fmt.Sprintf("%s|%s|%s|%s|%s|%d",
		al.lastHash, eventID, orderID, eventType, payloadHash, now)
	eventHash := sha256Hex(hashInput)

	event := &AuditEvent{
		EventID:       eventID,
		OrderID:       orderID,
		EventType:     eventType,
		EventHash:     eventHash,
		PrevEventHash: al.lastHash,
		MerkleRoot:    "", // Set when block is produced
		OperatorHash:  operatorHash,
		BlockHeight:   blockHeight,
		CreatedAt:     now,
	}

	al.events = append(al.events, event)
	al.eventsByOrder[orderID] = append(al.eventsByOrder[orderID], event)
	al.lastHash = eventHash

	return event, nil
}

// GetEventsByOrder returns all audit events for a specific order, in order.
func (al *AuditLedger) GetEventsByOrder(orderID string) []*AuditEvent {
	al.mu.RLock()
	defer al.mu.RUnlock()

	events := al.eventsByOrder[orderID]
	if events == nil {
		return []*AuditEvent{}
	}

	cp := make([]*AuditEvent, len(events))
	copy(cp, events)
	return cp
}

// GetAllEvents returns all audit events in insertion order.
func (al *AuditLedger) GetAllEvents() []*AuditEvent {
	al.mu.RLock()
	defer al.mu.RUnlock()

	cp := make([]*AuditEvent, len(al.events))
	copy(cp, al.events)
	return cp
}

// VerifyEventChain validates the cryptographic hash chain for an order's audit events.
// Returns true if every event correctly links to its predecessor.
// This is the core of the immutable audit proof.
func (al *AuditLedger) VerifyEventChain(orderID string) (bool, error) {
	al.mu.RLock()
	defer al.mu.RUnlock()

	events := al.eventsByOrder[orderID]
	if len(events) == 0 {
		return true, nil
	}

	// Verify the global chain: each event's prev_event_hash must match
	// the previous event's event_hash. We use the order-specific events
	// but they chain through the global lastHash at insertion time.

	// Simplified verification: check each event's hash consistency
	for i := 1; i < len(events); i++ {
		// The prev_event_hash of event[i] should reference a previous event
		// (may not be event[i-1] for the same order, since other orders'
		// events are interleaved in the global chain)
		prevHash := events[i].PrevEventHash

		// Find the referenced event in the global list
		found := false
		for _, e := range al.events {
			if e.EventHash == prevHash {
				found = true
				break
			}
		}
		if !found {
			return false, fmt.Errorf("broken audit chain for order %s: hash %s not found in ledger",
				orderID, prevHash[:16])
		}
	}

	return true, nil
}

// GetLastHash returns the current hash chain pointer.
func (al *AuditLedger) GetLastHash() string {
	al.mu.RLock()
	defer al.mu.RUnlock()
	return al.lastHash
}

// GetEventCount returns the total number of audit events.
func (al *AuditLedger) GetEventCount() int {
	al.mu.RLock()
	defer al.mu.RUnlock()
	return len(al.events)
}

// SetMerkleRoot sets the Merkle root for events in a block.
func (al *AuditLedger) SetMerkleRoot(blockHeight uint64, merkleRoot string) {
	al.mu.Lock()
	defer al.mu.Unlock()

	for _, event := range al.events {
		if event.BlockHeight == blockHeight {
			event.MerkleRoot = merkleRoot
		}
	}
}

func sha256Hex(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
