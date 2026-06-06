package channel

import (
	"fmt"
	"time"
)

// PaymentChannel defines the interface for executing real money movement
// through financial institutions. Per architecture §5.4, the payment
// execution layer remains centralized and compliant while upstream
// stages (quote, order, reconciliation) are decentralized.
type PaymentChannel interface {
	// ExecutePayment initiates a payment through the channel.
	// Returns a channel-specific payment reference on success.
	ExecutePayment(instruction PaymentInstruction) (*PaymentResult, error)

	// CheckStatus queries the current state of a payment.
	CheckStatus(channelRef string) (*PaymentResult, error)

	// Refund reverses a previously executed payment.
	Refund(channelRef string, amount int64, reason string) (*PaymentResult, error)

	// GetChannelName returns the channel type identifier.
	GetChannelName() string

	// IsHealthy returns whether the channel is operational.
	IsHealthy() bool
}

// PaymentInstruction contains all details needed to execute a payment.
type PaymentInstruction struct {
	OrderID        string            `json:"order_id"`
	TransactionID  string            `json:"transaction_id"`
	SourceAccount  string            `json:"source_account"`
	TargetAccount  string            `json:"target_account"`
	SourceCurrency string            `json:"source_currency"`
	TargetCurrency string            `json:"target_currency"`
	Amount         int64             `json:"amount"`       // In source currency minor units
	TargetAmount   int64             `json:"target_amount"` // In target currency minor units
	ExchangeRate   float64           `json:"exchange_rate"`
	Fee            int64             `json:"fee"`
	Description    string            `json:"description"`
	Metadata       map[string]string `json:"metadata"`
	CreatedAt      time.Time         `json:"created_at"`
}

// PaymentResult is the outcome of a payment execution or status check.
type PaymentResult struct {
	ChannelRef    string    `json:"channel_ref"`    // Channel-specific reference ID
	Status        string    `json:"status"`         // pending/processing/completed/failed
	Amount        int64     `json:"amount"`          // Actual amount processed
	Fee           int64     `json:"fee"`             // Actual fee charged
	ReceiptHash   string    `json:"receipt_hash"`    // Hash of the payment receipt
	ReceiptURI    string    `json:"receipt_uri"`     // URI to retrieve full receipt
	ExecutedAt    time.Time `json:"executed_at"`
	ConfirmedAt   time.Time `json:"confirmed_at,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
}

// ChannelType enumerates the payment channel types per architecture §5.4.
type ChannelType string

const (
	ChannelBank       ChannelType = "bank"
	ChannelPSP        ChannelType = "psp"
	ChannelSWIFT      ChannelType = "swift"
	ChannelCard       ChannelType = "card"
	ChannelLocalRail  ChannelType = "local_rail"
	ChannelStablecoin ChannelType = "stablecoin"
	ChannelFXProvider ChannelType = "fx_provider"
)

// String returns the string representation of a channel type.
func (ct ChannelType) String() string {
	return string(ct)
}

// Validate checks that the channel type is known.
func (ct ChannelType) Validate() error {
	switch ct {
	case ChannelBank, ChannelPSP, ChannelSWIFT, ChannelCard,
		ChannelLocalRail, ChannelStablecoin, ChannelFXProvider:
		return nil
	default:
		return fmt.Errorf("unknown channel type: %s", ct)
	}
}
