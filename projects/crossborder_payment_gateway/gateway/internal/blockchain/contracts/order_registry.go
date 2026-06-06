package contracts

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/aspira/crossborder-payment-gateway/internal/models"
)

// OrderRecord stores an order on the Aspira Consortium Chain.
// Per architecture §7.1 and §10.2: only hashes and commitments are stored on-chain.
// Sensitive data (amounts, identities, bank accounts) stays in the off-chain SQLite database.
type OrderRecord struct {
	OrderID          string                  `json:"order_id"`
	OrderHash        string                  `json:"order_hash"`
	MerchantHash     string                  `json:"merchant_hash"`
	CustomerHash     string                  `json:"customer_hash"`
	AmountCommitment string                  `json:"amount_commitment"`
	QuoteID          string                  `json:"quote_id"`
	Status           models.TransactionStatus `json:"status"`
	CreatedAt        int64                   `json:"created_at"`
	UpdatedAt        int64                   `json:"updated_at"`
}

// OrderRegistry is the on-chain smart contract for order management.
// Equivalent to the Solidity OrderRegistry contract in architecture §7.1.
type OrderRegistry struct {
	mu      sync.RWMutex
	orders  map[string]*OrderRecord // orderID → OrderRecord
	eventCh chan OrderEvent
}

// OrderEvent is emitted when an order is created or updated on chain.
type OrderEvent struct {
	OrderID string
	Type    string // "created", "updated"
	Status  models.TransactionStatus
}

// NewOrderRegistry creates a new OrderRegistry contract instance.
func NewOrderRegistry() *OrderRegistry {
	return &OrderRegistry{
		orders:  make(map[string]*OrderRecord),
		eventCh: make(chan OrderEvent, 256),
	}
}

// CreateOrder registers a new order on the chain.
// Only hashes are stored; raw data stays off-chain.
// Per §7.1: "require(orders[orderId].createdAt == 0, "ORDER_EXISTS")"
func (r *OrderRegistry) CreateOrder(orderID, merchantID, customerID string, amount int64, quoteID string) (*OrderRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate orders (§7.1)
	if _, exists := r.orders[orderID]; exists {
		return nil, fmt.Errorf("ORDER_EXISTS: %s", orderID)
	}

	now := time.Now().Unix()

	// Compute hashes and commitments (§10.2)
	// commitment = Hash(value + salt), where salt is a random component
	salt := fmt.Sprintf("%d", now)
	merchantHash := CommitHash(merchantID + salt)
	customerHash := CommitHash(customerID + salt)
	amountCommitment := CommitHash(fmt.Sprintf("%d%s", amount, salt))
	orderHash := CommitHash(fmt.Sprintf("%s|%s|%s|%s|%d",
		orderID, merchantHash, customerHash, amountCommitment, now))

	record := &OrderRecord{
		OrderID:          orderID,
		OrderHash:        orderHash,
		MerchantHash:     merchantHash,
		CustomerHash:     customerHash,
		AmountCommitment: amountCommitment,
		QuoteID:          quoteID,
		Status:           models.StatusCreated,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	r.orders[orderID] = record

	// Emit event (non-blocking)
	select {
	case r.eventCh <- OrderEvent{OrderID: orderID, Type: "created", Status: models.StatusCreated}:
	default:
	}

	return record, nil
}

// GetOrder retrieves an order from the chain.
func (r *OrderRegistry) GetOrder(orderID string) (*OrderRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, exists := r.orders[orderID]
	if !exists {
		return nil, fmt.Errorf("order not found on chain: %s", orderID)
	}
	return record, nil
}

// UpdateStatus updates the status of an on-chain order.
func (r *OrderRegistry) UpdateStatus(orderID string, newStatus models.TransactionStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, exists := r.orders[orderID]
	if !exists {
		return fmt.Errorf("order not found on chain: %s", orderID)
	}

	record.Status = newStatus
	record.UpdatedAt = time.Now().Unix()

	select {
	case r.eventCh <- OrderEvent{OrderID: orderID, Type: "updated", Status: newStatus}:
	default:
	}

	return nil
}

// OrderExists checks if an order is already registered on chain.
func (r *OrderRegistry) OrderExists(orderID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.orders[orderID]
	return exists
}

// GetOrderCount returns the total number of orders on chain.
func (r *OrderRegistry) GetOrderCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.orders)
}

// Events returns the event channel for order events.
func (r *OrderRegistry) Events() <-chan OrderEvent {
	return r.eventCh
}

// GetAllOrders returns a copy of all on-chain orders.
func (r *OrderRegistry) GetAllOrders() []*OrderRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	orders := make([]*OrderRecord, 0, len(r.orders))
	for _, o := range r.orders {
		orders = append(orders, o)
	}
	return orders
}

// CommitHash computes a SHA-256 commitment hash.
// commitment = SHA-256(value)
func CommitHash(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}
