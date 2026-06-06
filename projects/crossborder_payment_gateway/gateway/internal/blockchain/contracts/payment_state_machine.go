package contracts

import (
	"fmt"
	"sync"
	"time"

	"github.com/aspira/crossborder-payment-gateway/internal/models"
)

// StateTransition records an immutable state change on chain.
// Per architecture §7.3: every state transition is recorded with
// timestamp, signer, and proof for complete auditability.
type StateTransition struct {
	ID          int64                      `json:"id"`
	OrderID     string                     `json:"order_id"`
	FromStatus  models.TransactionStatus   `json:"from_status"`
	ToStatus    models.TransactionStatus   `json:"to_status"`
	ProofHash   string                     `json:"proof_hash"`
	Signer      string                     `json:"signer"`
	BlockHeight uint64                     `json:"block_height"`
	CreatedAt   int64                      `json:"created_at"`
}

// PaymentStateMachine is the on-chain state machine contract.
// Equivalent to the Solidity PaymentStateMachine in architecture §7.3.
// It strictly enforces the state transition table.
type PaymentStateMachine struct {
	mu           sync.RWMutex
	transitions  map[string][]*StateTransition // orderID → transition history
	currentState map[string]models.TransactionStatus
	eventCh      chan StateEvent
}

// StateEvent is emitted when a state transition occurs.
type StateEvent struct {
	OrderID    string
	FromStatus models.TransactionStatus
	ToStatus   models.TransactionStatus
}

// NewPaymentStateMachine creates a new state machine contract.
func NewPaymentStateMachine() *PaymentStateMachine {
	return &PaymentStateMachine{
		transitions:  make(map[string][]*StateTransition),
		currentState: make(map[string]models.TransactionStatus),
		eventCh:      make(chan StateEvent, 256),
	}
}

// Transition attempts to change the state of an order.
// It validates the transition against the allowed transitions table (§7.3).
// This is the on-chain equivalent of models.ValidateTransition().
func (sm *PaymentStateMachine) Transition(orderID string, fromStatus, toStatus models.TransactionStatus, proofHash, signer string, blockHeight uint64) (*StateTransition, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Get current state (or use provided fromStatus for initial transition)
	current, exists := sm.currentState[orderID]
	if exists {
		// State already exists on chain — verify the fromStatus matches
		if current != fromStatus && fromStatus != "" {
			return nil, fmt.Errorf("STATE_MISMATCH: expected current state %s, got %s", current, fromStatus)
		}
	}

	// Validate the transition per architecture §7.3
	// Empty fromStatus means initial state creation (first transition for this order)
	if fromStatus != "" {
		if err := models.ValidateTransition(fromStatus, toStatus); err != nil {
			return nil, fmt.Errorf("ILLEGAL_TRANSITION: %w", err)
		}
	}

	now := time.Now().Unix()

	transition := &StateTransition{
		OrderID:     orderID,
		FromStatus:  fromStatus,
		ToStatus:    toStatus,
		ProofHash:   proofHash,
		Signer:      signer,
		BlockHeight: blockHeight,
		CreatedAt:   now,
	}

	// Record the transition
	sm.transitions[orderID] = append(sm.transitions[orderID], transition)
	sm.currentState[orderID] = toStatus

	// Emit event
	select {
	case sm.eventCh <- StateEvent{OrderID: orderID, FromStatus: fromStatus, ToStatus: toStatus}:
	default:
	}

	return transition, nil
}

// GetCurrentState returns the current on-chain state of an order.
func (sm *PaymentStateMachine) GetCurrentState(orderID string) (models.TransactionStatus, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.currentState[orderID]
	if !exists {
		return "", fmt.Errorf("no state found for order: %s", orderID)
	}
	return state, nil
}

// GetStateHistory returns the complete state transition history for an order.
func (sm *PaymentStateMachine) GetStateHistory(orderID string) []*StateTransition {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	history := sm.transitions[orderID]
	if history == nil {
		return []*StateTransition{}
	}

	// Return a copy
	cp := make([]*StateTransition, len(history))
	copy(cp, history)
	return cp
}

// VerifyStateHistory verifies the state transition chain for an order.
// Returns true if every transition follows the allowed transitions.
func (sm *PaymentStateMachine) VerifyStateHistory(orderID string) (bool, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	history := sm.transitions[orderID]
	if len(history) == 0 {
		return true, nil
	}

	for _, t := range history {
		if err := models.ValidateTransition(t.FromStatus, t.ToStatus); err != nil {
			return false, fmt.Errorf("invalid transition in history for %s: %s -> %s (%w)",
				orderID, t.FromStatus, t.ToStatus, err)
		}
	}

	return true, nil
}

// Events returns the event channel.
func (sm *PaymentStateMachine) Events() <-chan StateEvent {
	return sm.eventCh
}

// GetTransitionCount returns the total number of state transitions on chain.
func (sm *PaymentStateMachine) GetTransitionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	total := 0
	for _, history := range sm.transitions {
		total += len(history)
	}
	return total
}
