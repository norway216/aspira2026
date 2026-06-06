package models

import "fmt"

// AllowedTransitions defines the valid state transitions per architecture §7.3.
// Any transition not listed here is illegal and will be rejected.
var AllowedTransitions = map[TransactionStatus][]TransactionStatus{
	StatusCreated:              {StatusQuoteLocked, StatusCancelled, StatusRiskRejected},
	StatusQuoteLocked:          {StatusCompliancePrechecked, StatusRiskRejected, StatusCancelled},
	StatusCompliancePrechecked: {StatusPaymentPending, StatusManualReview, StatusRiskRejected},
	StatusPaymentPending:       {StatusPaymentExecuting, StatusCancelled},
	StatusPaymentExecuting:     {StatusPaymentConfirmed, StatusPaymentFailed},
	StatusPaymentConfirmed:     {StatusSettlementProofed, StatusRefundPending, StatusDisputed, StatusFrozen},
	StatusSettlementProofed:    {StatusReconciled, StatusDisputed},
	StatusReconciled:           {StatusClosed, StatusDisputed},
	StatusClosed:               {}, // Terminal state

	// Error / exception state transitions
	StatusRiskRejected:  {},
	StatusPaymentFailed: {StatusRefundPending, StatusManualReview, StatusPaymentPending}, // Retry allowed
	StatusRefundPending: {StatusRefunded, StatusPaymentFailed},
	StatusRefunded:      {StatusClosed},
	StatusDisputed:      {StatusClosed, StatusManualReview, StatusRefundPending},
	StatusFrozen:        {StatusManualReview, StatusCancelled, StatusRefundPending},
	StatusManualReview:  {StatusPaymentPending, StatusCancelled, StatusRiskRejected, StatusFrozen},
	StatusCancelled:     {StatusClosed},
}

// TerminalStates are states from which no further positive transitions are possible.
var TerminalStates = map[TransactionStatus]bool{
	StatusClosed:       true,
	StatusRiskRejected: true,
	StatusRefunded:     true,
}

// ValidateTransition checks if a state transition is allowed.
func ValidateTransition(from, to TransactionStatus) error {
	if from == to {
		return nil // Idempotent — same state is always allowed
	}

	allowed, ok := AllowedTransitions[from]
	if !ok {
		return fmt.Errorf("unknown current state: %s", from)
	}

	for _, s := range allowed {
		if s == to {
			return nil
		}
	}

	return fmt.Errorf("illegal state transition: %s -> %s", from, to)
}

// GetNextStates returns all valid next states from the given state.
func GetNextStates(status TransactionStatus) []TransactionStatus {
	if allowed, ok := AllowedTransitions[status]; ok {
		return allowed
	}
	return nil
}

// IsTerminal returns true if the state is terminal (no further transitions).
func IsTerminal(status TransactionStatus) bool {
	return TerminalStates[status]
}

// IsPositiveFlow returns true if the status is part of the normal payment flow.
func IsPositiveFlow(status TransactionStatus) bool {
	switch status {
	case StatusCreated, StatusQuoteLocked, StatusCompliancePrechecked,
		StatusPaymentPending, StatusPaymentExecuting, StatusPaymentConfirmed,
		StatusSettlementProofed, StatusReconciled, StatusClosed:
		return true
	default:
		return false
	}
}

// IsExceptionFlow returns true if the status is an exception/error state.
func IsExceptionFlow(status TransactionStatus) bool {
	return !IsPositiveFlow(status)
}
