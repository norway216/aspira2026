package events

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EventBus is an in-memory publish/subscribe event bus.
// It serves as the foundation for the architecture's event-driven design (§5.7).
// In production, this would be replaced by Kafka/NATS with persistent storage,
// but the interface and event schema remain the same.
type EventBus struct {
	subscribers map[string][]EventHandler
	mu          sync.RWMutex
	eventLog    []Event        // In-memory event log for replay/testing
	logMu       sync.RWMutex
	lastHash    string         // Hash chain pointer
}

// NewEventBus creates a new event bus with an empty subscriber map.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]EventHandler),
		eventLog:    make([]Event, 0, 1024),
		lastHash:    "0000000000000000000000000000000000000000000000000000000000000000",
	}
}

// Subscribe registers a handler for a specific event type.
// Returns an unsubscribe function.
func (eb *EventBus) Subscribe(eventType string, handler EventHandler) func() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.subscribers[eventType] == nil {
		eb.subscribers[eventType] = make([]EventHandler, 0)
	}
	eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)

	// Return unsubscribe function
	idx := len(eb.subscribers[eventType]) - 1
	return func() {
		eb.mu.Lock()
		defer eb.mu.Unlock()
		handlers := eb.subscribers[eventType]
		if idx < len(handlers) {
			eb.subscribers[eventType] = append(handlers[:idx], handlers[idx+1:]...)
		}
	}
}

// Publish sends an event to all subscribers of its type.
// The event is enriched with ID, timestamp, and hash chain before dispatch.
func (eb *EventBus) Publish(eventType string, orderID string, payload interface{}) {
	now := time.Now().Unix()

	// Compute payload hash
	payloadBytes, _ := json.Marshal(payload)
	payloadHash := sha256.Sum256(payloadBytes)

	// Compute event hash chain
	eb.logMu.Lock()
	prevHash := eb.lastHash
	hashInput := fmt.Sprintf("%s|%s|%s|%x|%d", prevHash, eventType, orderID, payloadHash, now)
	eventHash := sha256.Sum256([]byte(hashInput))
	currHash := hex.EncodeToString(eventHash[:])
	eb.lastHash = currHash
	eb.logMu.Unlock()

	event := Event{
		EventID:       uuid.New().String(),
		OrderID:       orderID,
		EventType:     eventType,
		Payload:       payload,
		PayloadHash:   hex.EncodeToString(payloadHash[:]),
		PrevEventHash: prevHash,
		Timestamp:     now,
	}

	// Log event
	log.Printf("[EventBus] %s | order=%s | id=%s", eventType, orderID, event.EventID)

	// Store in event log
	eb.logMu.Lock()
	eb.eventLog = append(eb.eventLog, event)
	if len(eb.eventLog) > 10000 {
		eb.eventLog = eb.eventLog[1:] // Trim old events
	}
	eb.logMu.Unlock()

	// Dispatch to subscribers (non-blocking per subscriber)
	eb.mu.RLock()
	handlers := eb.subscribers[eventType]
	// Also dispatch to wildcard subscribers
	if wildcardHandlers, ok := eb.subscribers["*"]; ok {
		handlers = append(handlers, wildcardHandlers...)
	}
	eb.mu.RUnlock()

	for _, handler := range handlers {
		go func(h EventHandler, evt Event) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[EventBus] handler panic for %s: %v", eventType, r)
				}
			}()
			h(evt)
		}(handler, event)
	}
}

// GetEventLog returns a copy of the recent event log.
func (eb *EventBus) GetEventLog() []Event {
	eb.logMu.RLock()
	defer eb.logMu.RUnlock()
	cp := make([]Event, len(eb.eventLog))
	copy(cp, eb.eventLog)
	return cp
}

// GetLastHash returns the current hash chain pointer.
func (eb *EventBus) GetLastHash() string {
	eb.logMu.RLock()
	defer eb.logMu.RUnlock()
	return eb.lastHash
}

// SubscriberCount returns the number of subscribers for a given event type.
func (eb *EventBus) SubscriberCount(eventType string) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.subscribers[eventType])
}

// TotalSubscriberCount returns the total number of subscribers across all event types.
func (eb *EventBus) TotalSubscriberCount() int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	total := 0
	for _, handlers := range eb.subscribers {
		total += len(handlers)
	}
	return total
}
