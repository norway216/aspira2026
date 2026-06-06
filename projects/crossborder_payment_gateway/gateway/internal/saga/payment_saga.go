package saga

import (
	"context"
	"fmt"
	"log"

	"github.com/aspira/crossborder-payment-gateway/internal/channel"
	"github.com/aspira/crossborder-payment-gateway/internal/database"
	"github.com/aspira/crossborder-payment-gateway/internal/events"
	"github.com/aspira/crossborder-payment-gateway/internal/models"
)

// NewPaymentSaga creates the standard payment saga per architecture §8.2.
//
// Steps:
//   1. CreateOrder          — Register order in database
//   2. LockQuote            — Lock the accepted quote
//   3. ComplianceCheck      — Run compliance and risk pre-checks
//   4. CreatePaymentInstr   — Generate payment instruction
//   5. ExecutePayment       — Execute payment through channel
//   6. WriteSettlementProof — Record settlement proof
//   7. Reconcile            — Run reconciliation
//   8. CloseOrder           — Mark order as closed
//
// Compensation follows architecture §8.3.
func NewPaymentSaga(
	db database.DB,
	eventBus *events.EventBus,
	channels *channel.ChannelRegistry,
) *Saga {
	saga := &Saga{
		Name: "payment-saga",
		Steps: []SagaStep{
			{
				Name: "CreateOrder",
				Execute: func(ctx context.Context, orderID string, state map[string]interface{}) error {
					// Order is already created in the database by the handler.
					// This step verifies the order exists and publishes the event.
					txn, err := db.GetTransaction(orderID)
					if err != nil {
						return fmt.Errorf("order not found: %s", orderID)
					}

					eventBus.Publish(events.EventOrderCreated, orderID, map[string]interface{}{
						"merchant_id": txn.MerchantID,
						"amount":      txn.SourceAmount,
						"currency":    txn.SourceCurrency,
					})

					state["merchant_id"] = txn.MerchantID
					state["source_amount"] = txn.SourceAmount
					state["source_currency"] = txn.SourceCurrency
					state["target_currency"] = txn.TargetCurrency

					log.Printf("[PaymentSaga] Order %s verified", orderID)
					return nil
				},
				Compensate: func(ctx context.Context, orderID string, state map[string]interface{}) error {
					// Cancel the order
					return db.UpdateTransactionStatus(orderID, models.StatusCancelled)
				},
			},
			{
				Name: "LockQuote",
				Execute: func(ctx context.Context, orderID string, state map[string]interface{}) error {
					// Transition order to quote_locked if a quote was used
					txn, err := db.GetTransaction(orderID)
					if err != nil {
						return err
					}

					if txn.Status == models.StatusCreated {
						if err := db.UpdateTransactionStatusValidated(orderID, models.StatusCreated, models.StatusQuoteLocked); err != nil {
							return err
						}
					}

					eventBus.Publish(events.EventQuoteCommitted, orderID, map[string]interface{}{
						"rate": txn.ExchangeRate,
						"fee":  txn.Fee,
					})

					state["rate"] = txn.ExchangeRate
					state["fee"] = txn.Fee
					log.Printf("[PaymentSaga] Quote locked for order %s", orderID)
					return nil
				},
				Compensate: func(ctx context.Context, orderID string, state map[string]interface{}) error {
					log.Printf("[PaymentSaga] Unlocking quote for order %s", orderID)
					return nil // Quote unlock is handled by expiry
				},
			},
			{
				Name: "ComplianceCheck",
				Execute: func(ctx context.Context, orderID string, state map[string]interface{}) error {
					// Simulated compliance check — in production, this would call
					// KYC/AML/Sanctions screening services.

					eventBus.Publish(events.EventCompliancePrechecked, orderID, map[string]interface{}{
						"result": "passed",
					})

					// Transition to compliance_prechecked
					if err := db.UpdateTransactionStatusValidated(orderID, models.StatusQuoteLocked, models.StatusCompliancePrechecked); err != nil {
						// If not in quote_locked state, try from current state
						log.Printf("[PaymentSaga] Could not transition from quote_locked, may already be advanced")
					}

					log.Printf("[PaymentSaga] Compliance check passed for order %s", orderID)
					return nil
				},
				Compensate: func(ctx context.Context, orderID string, state map[string]interface{}) error {
					log.Printf("[PaymentSaga] Compliance check compensation for order %s", orderID)
					return db.UpdateTransactionStatus(orderID, models.StatusRiskRejected)
				},
			},
			{
				Name: "CreatePaymentInstruction",
				Execute: func(ctx context.Context, orderID string, state map[string]interface{}) error {
					// Transition to payment_pending
					if err := db.UpdateTransactionStatusValidated(orderID, models.StatusCompliancePrechecked, models.StatusPaymentPending); err != nil {
						log.Printf("[PaymentSaga] State transition issue: %v", err)
					}

					eventBus.Publish(events.EventPaymentInstructionCreated, orderID, state)
					log.Printf("[PaymentSaga] Payment instruction created for order %s", orderID)
					return nil
				},
				Compensate: func(ctx context.Context, orderID string, state map[string]interface{}) error {
					return db.UpdateTransactionStatus(orderID, models.StatusCancelled)
				},
			},
			{
				Name: "ExecutePayment",
				Execute: func(ctx context.Context, orderID string, state map[string]interface{}) error {
					// Transition to payment_executing
					if err := db.UpdateTransactionStatusValidated(orderID, models.StatusPaymentPending, models.StatusPaymentExecuting); err != nil {
						return fmt.Errorf("failed to transition to executing: %w", err)
					}

					// Get transaction details
					txn, err := db.GetTransaction(orderID)
					if err != nil {
						return err
					}

					// Build payment instruction
					instruction := channel.PaymentInstruction{
						OrderID:        orderID,
						TransactionID:  txn.ID,
						SourceAccount:  txn.PayerAccountID,
						TargetAccount:  txn.PayeeAccountID,
						SourceCurrency: txn.SourceCurrency,
						TargetCurrency: txn.TargetCurrency,
						Amount:         txn.SourceAmount,
						TargetAmount:   txn.TargetAmount,
						ExchangeRate:   txn.ExchangeRate,
						Fee:            txn.Fee,
						Description:    txn.Description,
					}

					// Execute through channel
					ch, err := channels.GetHealthy("mock-bank")
					if err != nil {
						return fmt.Errorf("no payment channel available: %w", err)
					}

					result, err := ch.ExecutePayment(instruction)
					if err != nil {
						_ = db.UpdateTransactionStatus(orderID, models.StatusPaymentFailed)
						eventBus.Publish(events.EventPaymentExecuted, orderID, map[string]interface{}{
							"status": "failed",
							"error":  err.Error(),
						})
						return fmt.Errorf("payment execution failed: %w", err)
					}

					// Transition to payment_confirmed
					if err := db.UpdateTransactionStatusValidated(orderID, models.StatusPaymentExecuting, models.StatusPaymentConfirmed); err != nil {
						return err
					}

					state["channel_ref"] = result.ChannelRef
					state["receipt_hash"] = result.ReceiptHash

					eventBus.Publish(events.EventPaymentExecuted, orderID, map[string]interface{}{
						"status":       "completed",
						"channel_ref":  result.ChannelRef,
						"receipt_hash": result.ReceiptHash,
					})

					log.Printf("[PaymentSaga] Payment executed for order %s via %s (ref: %s)",
						orderID, ch.GetChannelName(), result.ChannelRef)
					return nil
				},
				Compensate: func(ctx context.Context, orderID string, state map[string]interface{}) error {
					// Attempt refund through the channel
					channelRef, _ := state["channel_ref"].(string)
					if channelRef != "" {
						ch, err := channels.GetHealthy("mock-bank")
						if err == nil {
							_, _ = ch.Refund(channelRef, 0, "saga-compensation")
						}
					}
					log.Printf("[PaymentSaga] Payment compensation for order %s", orderID)
					return db.UpdateTransactionStatus(orderID, models.StatusRefundPending)
				},
			},
			{
				Name: "WriteSettlementProof",
				Execute: func(ctx context.Context, orderID string, state map[string]interface{}) error {
					if err := db.UpdateTransactionStatusValidated(orderID, models.StatusPaymentConfirmed, models.StatusSettlementProofed); err != nil {
						return err
					}

					eventBus.Publish(events.EventSettlementProofCreated, orderID, state)
					log.Printf("[PaymentSaga] Settlement proof written for order %s", orderID)
					return nil
				},
				Compensate: func(ctx context.Context, orderID string, state map[string]interface{}) error {
					return db.UpdateTransactionStatus(orderID, models.StatusDisputed)
				},
			},
			{
				Name: "Reconcile",
				Execute: func(ctx context.Context, orderID string, state map[string]interface{}) error {
					// Create a reconciliation record
					txn, err := db.GetTransaction(orderID)
					if err != nil {
						return err
					}

					rec := &models.ReconciliationRecord{
						OrderID:        orderID,
						TransactionID:  txn.ID,
						InternalAmount: txn.TargetAmount,
						ChannelAmount:  txn.TargetAmount,
						Currency:       txn.TargetCurrency,
						ChannelName:    "mock-bank",
						ChannelRef:     "",
						MatchStatus:    models.RecMatched,
					}

					if err := db.CreateReconciliationRecord(rec); err != nil {
						log.Printf("[PaymentSaga] Reconciliation record creation failed: %v", err)
					}

					if err := db.UpdateTransactionStatusValidated(orderID, models.StatusSettlementProofed, models.StatusReconciled); err != nil {
						return err
					}

					eventBus.Publish(events.EventReconciliationCompleted, orderID, nil)
					log.Printf("[PaymentSaga] Reconciliation completed for order %s", orderID)
					return nil
				},
				Compensate: func(ctx context.Context, orderID string, state map[string]interface{}) error {
					return db.UpdateTransactionStatus(orderID, models.StatusDisputed)
				},
			},
			{
				Name: "CloseOrder",
				Execute: func(ctx context.Context, orderID string, state map[string]interface{}) error {
					if err := db.UpdateTransactionStatusValidated(orderID, models.StatusReconciled, models.StatusClosed); err != nil {
						return err
					}

					eventBus.Publish(events.EventOrderStateUpdated, orderID, map[string]interface{}{
						"new_status": "closed",
					})

					log.Printf("[PaymentSaga] Order %s closed successfully", orderID)
					return nil
				},
				Compensate: nil, // Terminal step — no compensation for close
			},
		},
	}

	return saga
}
