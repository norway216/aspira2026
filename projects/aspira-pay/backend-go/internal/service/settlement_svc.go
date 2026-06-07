package service

import (
	"fmt"
	"log"

	"github.com/aspira/aspira-pay/internal/domain/ledger"
	"github.com/aspira/aspira-pay/internal/domain/settlement"
	"github.com/aspira/aspira-pay/internal/repository"
	"github.com/aspira/aspira-pay/pkg/crypto"
	"github.com/aspira/aspira-pay/pkg/idgen"
)

// SettlementService handles double-entry ledger generation and settlement batching.
type SettlementService struct {
	db *repository.DB
}

// NewSettlementService creates a new SettlementService.
func NewSettlementService(db *repository.DB) *SettlementService {
	return &SettlementService{db: db}
}

// SettlePayment generates double-entry ledger entries for a completed payment.
// This is the core accounting function that ensures 借贷平衡 (debit-credit balance).
func (s *SettlementService) SettlePayment(paymentID string) error {
	order, err := s.db.GetPaymentOrder(paymentID)
	if err != nil {
		return fmt.Errorf("cannot find payment: %w", err)
	}

	// Check no existing entries (idempotency)
	existing, _ := s.db.GetLedgerEntriesByPayment(paymentID)
	if len(existing) > 0 {
		log.Printf("Settlement: ledger entries already exist for %s, skipping", paymentID)
		return nil
	}

	eventID := idgen.EventID()

	// Get account balances BEFORE changes
	senderAccount, err := s.db.GetAccountByUserAndCurrency(order.SenderUserID, order.SourceCurrency)
	if err != nil {
		return fmt.Errorf("sender account not found: %w", err)
	}

	receiverAccount, err := s.db.GetAccountByUserAndCurrency(order.ReceiverUserID, order.TargetCurrency)
	if err != nil {
		return fmt.Errorf("receiver account not found: %w", err)
	}

	// Platform settlement & fee accounts
	settlementAccountID := "sys_settlement_" + order.SourceCurrency
	feeAccountID := "sys_fee_income_" + order.SourceCurrency

	settlementAccount, err := s.db.GetAccount(settlementAccountID)
	if err != nil {
		return fmt.Errorf("settlement account not found: %w", err)
	}

	feeAccount, err := s.db.GetAccount(feeAccountID)
	if err != nil {
		return fmt.Errorf("fee account not found: %w", err)
	}

	// Build double-entry journal following architecture doc §4.8:
	//
	// Entry 1: Debit sender available, Credit settlement intermediary
	//   Dr: Sender Available  (sourceAmount + feeAmount) — decreases available
	//   Cr: Settlement Intermediary                       — increases settlement
	//
	// Entry 2: Debit settlement, Credit receiver
	//   Dr: Settlement Intermediary (sourceAmount)        — moves to receiver
	//   Cr: Receiver Available (targetAmount)             — increases available
	//
	// Entry 3: Fee allocation
	//   Dr: Settlement Intermediary (feeAmount)           — moves fee
	//   Cr: Platform Fee Income (feeAmount)               — increases fee income

	totalSenderDebit := order.SourceAmount + order.FeeAmount

	entries := []ledger.Entry{
		// Entry 1: Sender → Settlement (Dr: Sender, Cr: Settlement)
		{
			EntryID:      idgen.EntryID(paymentID, 1),
			EventID:      eventID,
			PaymentID:    paymentID,
			AccountID:    senderAccount.AccountID,
			Currency:     order.SourceCurrency,
			Direction:    ledger.DirectionDebit,
			Amount:       totalSenderDebit,
			BalanceAfter: senderAccount.AvailableBalance - totalSenderDebit,
			Description:  fmt.Sprintf("Payment %s: sender debit", paymentID),
		},
		{
			EntryID:      idgen.EntryID(paymentID, 2),
			EventID:      eventID,
			PaymentID:    paymentID,
			AccountID:    settlementAccountID,
			Currency:     order.SourceCurrency,
			Direction:    ledger.DirectionCredit,
			Amount:       totalSenderDebit,
			BalanceAfter: settlementAccount.AvailableBalance + totalSenderDebit,
			Description:  fmt.Sprintf("Payment %s: settlement credit (source)", paymentID),
		},

		// Entry 2: Settlement → Receiver (Dr: Settlement, Cr: Receiver)
		{
			EntryID:      idgen.EntryID(paymentID, 3),
			EventID:      eventID,
			PaymentID:    paymentID,
			AccountID:    settlementAccountID,
			Currency:     order.SourceCurrency,
			Direction:    ledger.DirectionDebit,
			Amount:       order.SourceAmount,
			BalanceAfter: settlementAccount.AvailableBalance + order.FeeAmount, // After removing sourceAmount
			Description:  fmt.Sprintf("Payment %s: settlement debit (to receiver)", paymentID),
		},
		{
			EntryID:      idgen.EntryID(paymentID, 4),
			EventID:      eventID,
			PaymentID:    paymentID,
			AccountID:    receiverAccount.AccountID,
			Currency:     order.TargetCurrency,
			Direction:    ledger.DirectionCredit,
			Amount:       order.TargetAmount,
			BalanceAfter: receiverAccount.AvailableBalance + order.TargetAmount,
			Description:  fmt.Sprintf("Payment %s: receiver credit", paymentID),
		},

		// Entry 3: Fee allocation (Dr: Settlement, Cr: Fee Income)
		{
			EntryID:      idgen.EntryID(paymentID, 5),
			EventID:      eventID,
			PaymentID:    paymentID,
			AccountID:    settlementAccountID,
			Currency:     order.SourceCurrency,
			Direction:    ledger.DirectionDebit,
			Amount:       order.FeeAmount,
			BalanceAfter: settlementAccount.AvailableBalance, // After fee removed
			Description:  fmt.Sprintf("Payment %s: settlement fee debit", paymentID),
		},
		{
			EntryID:      idgen.EntryID(paymentID, 6),
			EventID:      eventID,
			PaymentID:    paymentID,
			AccountID:    feeAccountID,
			Currency:     order.SourceCurrency,
			Direction:    ledger.DirectionCredit,
			Amount:       order.FeeAmount,
			BalanceAfter: feeAccount.AvailableBalance + order.FeeAmount,
			Description:  fmt.Sprintf("Payment %s: fee income credit", paymentID),
		},
	}

	// Validate double-entry balance
	if !ledger.CheckBalance(entries) {
		return fmt.Errorf("ledger imbalance: debit != credit for payment %s", paymentID)
	}

	// Insert all entries in a single transaction
	if err := s.db.InsertLedgerEntries(entries); err != nil {
		return fmt.Errorf("cannot insert ledger entries: %w", err)
	}

	log.Printf("Settlement: %d ledger entries created for payment %s (借贷平衡 verified)", len(entries), paymentID)
	return nil
}

// CreateSettlementBatch creates or updates a settlement batch for a currency.
func (s *SettlementService) CreateSettlementBatch(currency string) (*settlement.Batch, error) {
	// Check for existing open batch
	batch, _ := s.db.GetOpenSettlementBatch(currency)
	if batch != nil {
		return batch, nil
	}

	batch = &settlement.Batch{
		BatchID:    idgen.BatchID(),
		Currency:   currency,
		Status:     settlement.BatchOpen,
	}

	if err := s.db.CreateSettlementBatch(batch); err != nil {
		return nil, fmt.Errorf("cannot create settlement batch: %w", err)
	}

	return batch, nil
}

// CloseSettlementBatch closes a batch and computes the Merkle root.
func (s *SettlementService) CloseSettlementBatch(batchID string) error {
	batch, err := s.db.GetSettlementBatch(batchID)
	if err != nil {
		return err
	}

	if batch.Status != settlement.BatchOpen {
		return fmt.Errorf("batch is not open: %s", batch.Status)
	}

	// Compute Merkle root of all entries (simplified — in production, collect all event hashes)
	ledgerRootHash := crypto.SHA256(fmt.Sprintf("batch:%s:debit:%d:credit:%d:count:%d",
		batch.BatchID, batch.TotalDebit, batch.TotalCredit, batch.EntryCount))

	if err := s.db.UpdateSettlementBatchStatus(batchID, string(settlement.BatchClosed)); err != nil {
		return err
	}

	// Update ledger root hash
	query := `UPDATE settlement_batches SET ledger_root_hash = $1 WHERE batch_id = $2`
	_, err = s.db.Exec(query, ledgerRootHash, batchID)
	return err
}

// GetBatch retrieves a settlement batch.
func (s *SettlementService) GetBatch(batchID string) (*settlement.Batch, error) {
	return s.db.GetSettlementBatch(batchID)
}

// ListBatches returns paginated settlement batches.
func (s *SettlementService) ListBatches(page, pageSize int) ([]settlement.Batch, int64, error) {
	return s.db.ListSettlementBatches(page, pageSize)
}

// GetLedgerForPayment returns the full ledger for a payment.
func (s *SettlementService) GetLedgerForPayment(paymentID string) (*ledger.LedgerSummary, error) {
	return s.db.GetLedgerSummary(paymentID)
}
