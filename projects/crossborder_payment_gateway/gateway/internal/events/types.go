package events

// Event types match the architecture §5.7 Kafka topic design.
// Each event has a globally unique event_id, is bound to an order_id,
// includes a prev_event_hash for hash-chain integrity, and can be
// batched into Merkle roots for on-chain anchoring.
const (
	// Quote lifecycle
	EventQuoteRequested  = "quote.requested"
	EventQuoteGenerated  = "quote.generated"
	EventQuoteCommitted  = "quote.committed.onchain"
	EventQuoteAccepted   = "quote.accepted"
	EventQuoteExpired    = "quote.expired"

	// Order lifecycle
	EventOrderCreated     = "order.created"
	EventOrderStateUpdated = "order.state.updated"

	// Compliance & risk
	EventRiskPrechecked     = "risk.prechecked"
	EventCompliancePrechecked = "compliance.prechecked"

	// Payment lifecycle
	EventPaymentInstructionCreated = "payment.instruction.created"
	EventPaymentExecuted           = "payment.executed"
	EventPaymentCallbackReceived   = "payment.callback.received"

	// Settlement & proof
	EventSettlementProofCreated = "settlement.proof.created"

	// Reconciliation
	EventReconciliationStarted   = "reconciliation.started"
	EventReconciliationCompleted = "reconciliation.completed"

	// Refund
	EventRefundRequested = "refund.requested"
	EventRefundCompleted = "refund.completed"

	// Audit
	EventAuditCreated = "audit.event.created"

	// Dispute
	EventDisputeOpened = "dispute.opened"
	EventDisputeClosed = "dispute.closed"
)

// Event represents a single event in the system per architecture §12.2.
// Events form a hash chain: each event contains the hash of the previous
// event, creating an immutable audit trail that can be anchored on-chain.
type Event struct {
	EventID       string      `json:"event_id"`
	OrderID       string      `json:"order_id"`
	EventType     string      `json:"event_type"`
	Payload       interface{} `json:"payload"`
	PayloadHash   string      `json:"payload_hash"`
	PrevEventHash string      `json:"prev_event_hash"`
	Timestamp     int64       `json:"timestamp"`
	Signature     string      `json:"signature,omitempty"`
}

// EventHandler is a function that processes an event.
type EventHandler func(event Event)
