package channel

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"time"
)

// MockBankChannel simulates a bank payment connector.
// In production, this would integrate with actual banking APIs (SWIFT, ACH, SEPA, etc.).
type MockBankChannel struct {
	name    string
	healthy bool
	latency time.Duration // Simulated processing latency
}

func NewMockBankChannel(name string) *MockBankChannel {
	return &MockBankChannel{
		name:    name,
		healthy: true,
		latency: 200 * time.Millisecond,
	}
}

func (m *MockBankChannel) ExecutePayment(instruction PaymentInstruction) (*PaymentResult, error) {
	if !m.healthy {
		return nil, fmt.Errorf("bank channel %s is unhealthy", m.name)
	}

	// Simulate bank processing time
	time.Sleep(m.latency)

	channelRef := fmt.Sprintf("BANK-%s-%d", instruction.OrderID[:8], time.Now().UnixNano())
	receiptHash := m.computeReceiptHash(channelRef, instruction)

	log.Printf("[Bank:%s] Payment executed: %s %d %s → %s, ref=%s",
		m.name, instruction.SourceCurrency, instruction.Amount,
		instruction.TargetCurrency, channelRef)

	return &PaymentResult{
		ChannelRef:  channelRef,
		Status:      "completed",
		Amount:      instruction.TargetAmount,
		Fee:         instruction.Fee,
		ReceiptHash: receiptHash,
		ReceiptURI:  fmt.Sprintf("bank://%s/receipts/%s", m.name, channelRef),
		ExecutedAt:  time.Now(),
		ConfirmedAt: time.Now(),
	}, nil
}

func (m *MockBankChannel) CheckStatus(channelRef string) (*PaymentResult, error) {
	// Simulated: always returns completed for mock
	return &PaymentResult{
		ChannelRef: channelRef,
		Status:     "completed",
		ExecutedAt: time.Now().Add(-1 * time.Minute),
	}, nil
}

func (m *MockBankChannel) Refund(channelRef string, amount int64, reason string) (*PaymentResult, error) {
	refundRef := fmt.Sprintf("REFUND-%s-%d", channelRef, time.Now().UnixNano())
	receiptHash := m.computeReceiptHash(refundRef, PaymentInstruction{Amount: amount})

	log.Printf("[Bank:%s] Refund processed: %s, amount=%d, reason=%s",
		m.name, channelRef, amount, reason)

	return &PaymentResult{
		ChannelRef:  refundRef,
		Status:      "completed",
		Amount:      amount,
		ReceiptHash: receiptHash,
		ReceiptURI:  fmt.Sprintf("bank://%s/receipts/%s", m.name, refundRef),
		ExecutedAt:  time.Now(),
		ConfirmedAt: time.Now(),
	}, nil
}

func (m *MockBankChannel) GetChannelName() string { return m.name }
func (m *MockBankChannel) IsHealthy() bool         { return m.healthy }

func (m *MockBankChannel) computeReceiptHash(ref string, inst PaymentInstruction) string {
	input := fmt.Sprintf("%s|%s|%d|%s|%d", ref, inst.SourceCurrency, inst.Amount, inst.TargetCurrency, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// MockPSPChannel simulates a Payment Service Provider connector.
type MockPSPChannel struct {
	name       string
	healthy    bool
	successRate float64 // 0.0 to 1.0 probability of success
}

func NewMockPSPChannel(name string) *MockPSPChannel {
	return &MockPSPChannel{
		name:        name,
		healthy:     true,
		successRate: 0.95, // 95% success rate for realism
	}
}

func (m *MockPSPChannel) ExecutePayment(instruction PaymentInstruction) (*PaymentResult, error) {
	if !m.healthy {
		return nil, fmt.Errorf("PSP channel %s is unhealthy", m.name)
	}

	channelRef := fmt.Sprintf("PSP-%s-%d", instruction.OrderID[:8], time.Now().UnixNano())

	// Simulate occasional failures for realism
	if rand.Float64() > m.successRate {
		return &PaymentResult{
			ChannelRef:   channelRef,
			Status:       "failed",
			ErrorMessage: "simulated PSP processing failure",
			ExecutedAt:   time.Now(),
		}, fmt.Errorf("PSP payment failed: simulated error")
	}

	receiptHash := m.computeReceiptHash(channelRef, instruction)

	log.Printf("[PSP:%s] Payment executed: %s %d → %s %d, ref=%s",
		m.name, instruction.SourceCurrency, instruction.Amount,
		instruction.TargetCurrency, instruction.TargetAmount, channelRef)

	return &PaymentResult{
		ChannelRef:  channelRef,
		Status:      "completed",
		Amount:      instruction.TargetAmount,
		Fee:         instruction.Fee,
		ReceiptHash: receiptHash,
		ReceiptURI:  fmt.Sprintf("psp://%s/receipts/%s", m.name, channelRef),
		ExecutedAt:  time.Now(),
		ConfirmedAt: time.Now(),
	}, nil
}

func (m *MockPSPChannel) CheckStatus(channelRef string) (*PaymentResult, error) {
	return &PaymentResult{
		ChannelRef: channelRef,
		Status:     "completed",
		ExecutedAt: time.Now().Add(-1 * time.Minute),
	}, nil
}

func (m *MockPSPChannel) Refund(channelRef string, amount int64, reason string) (*PaymentResult, error) {
	refundRef := fmt.Sprintf("REFUND-PSP-%s-%d", channelRef, time.Now().UnixNano())
	receiptHash := m.computeReceiptHash(refundRef, PaymentInstruction{Amount: amount})

	log.Printf("[PSP:%s] Refund processed: %s, amount=%d, reason=%s",
		m.name, channelRef, amount, reason)

	return &PaymentResult{
		ChannelRef:  refundRef,
		Status:      "completed",
		Amount:      amount,
		ReceiptHash: receiptHash,
		ReceiptURI:  fmt.Sprintf("psp://%s/receipts/%s", m.name, refundRef),
		ExecutedAt:  time.Now(),
		ConfirmedAt: time.Now(),
	}, nil
}

func (m *MockPSPChannel) GetChannelName() string { return m.name }
func (m *MockPSPChannel) IsHealthy() bool         { return m.healthy }

func (m *MockPSPChannel) computeReceiptHash(ref string, inst PaymentInstruction) string {
	input := fmt.Sprintf("%s|%s|%d|%d", ref, inst.SourceCurrency, inst.Amount, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
