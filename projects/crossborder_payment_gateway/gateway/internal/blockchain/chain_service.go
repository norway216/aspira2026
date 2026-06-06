package blockchain

import (
	"fmt"
	"log"
	"time"

	"github.com/aspira/crossborder-payment-gateway/internal/blockchain/contracts"
	"github.com/aspira/crossborder-payment-gateway/internal/blockchain/core"
	"github.com/aspira/crossborder-payment-gateway/internal/events"
	"github.com/aspira/crossborder-payment-gateway/internal/models"
	"github.com/google/uuid"
)

// ChainService is the unified entry point for all blockchain operations.
// It wraps the blockchain core, block producer, and all smart contracts,
// providing a clean API for the gateway to interact with the chain.
type ChainService struct {
	Blockchain    *core.Blockchain
	Producer      *core.BlockProducer

	OrderRegistry    *contracts.OrderRegistry
	QuoteCommitment  *contracts.QuoteCommitment
	StateMachine     *contracts.PaymentStateMachine
	SettlementProof  *contracts.SettlementProofRegistry
	AuditLedger      *contracts.AuditLedger
	MerkleAnchor     *contracts.MerkleAnchor

	eventBus *events.EventBus
}

// ChainStatus contains summary information about the chain.
type ChainStatus struct {
	Height          uint64 `json:"height"`
	CurrentHash     string `json:"current_hash"`
	TotalBlocks     int    `json:"total_blocks"`
	OrdersOnChain   int    `json:"orders_on_chain"`
	QuotesCommitted int    `json:"quotes_committed"`
	StateTransitions int   `json:"state_transitions"`
	AuditEvents     int    `json:"audit_events"`
	SettlementProofs int   `json:"settlement_proofs"`
	MerkleAnchors   int    `json:"merkle_anchors"`
	MempoolSize     int    `json:"mempool_size"`
	ChainVerified   bool   `json:"chain_verified"`
}

// NewChainService creates and initializes the blockchain service.
// It creates the genesis block and starts the block producer.
func NewChainService(eventBus *events.EventBus) *ChainService {
	cs := &ChainService{
		Blockchain:       core.NewBlockchain(),
		OrderRegistry:    contracts.NewOrderRegistry(),
		QuoteCommitment:  contracts.NewQuoteCommitment(),
		StateMachine:     contracts.NewPaymentStateMachine(),
		SettlementProof:  contracts.NewSettlementProofRegistry(),
		AuditLedger:      contracts.NewAuditLedger(),
		MerkleAnchor:     contracts.NewMerkleAnchor(),
		eventBus:         eventBus,
	}

	// Create block producer with default config
	producerConfig := core.DefaultProducerConfig()
	cs.Producer = core.NewBlockProducer(cs.Blockchain, producerConfig)

	// Auto-anchor Merkle root every 10 blocks
	cs.Producer.SetOnNewBlock(func(block *core.Block) {
		cs.onBlockProduced(block)
	})

	// Start block production
	cs.Producer.Start()

	log.Println("[AspiraConsortium·Svc] Blockchain service initialized")
	return cs
}

// onBlockProduced is called after each new block is mined.
// It handles Merkle anchoring and audit event updates.
func (cs *ChainService) onBlockProduced(block *core.Block) {
	// Update audit events with Merkle root
	cs.AuditLedger.SetMerkleRoot(block.Index, block.MerkleRoot)

	// Anchor Merkle root every 10 blocks (§7.5 periodic anchoring)
	if block.Index%10 == 0 {
		anchor := cs.MerkleAnchor.AnchorRoot(block.Index, block.MerkleRoot)
		log.Printf("[AspiraConsortium·Svc] Merkle root anchored: block=%d, root=%s",
			anchor.BlockHeight, anchor.MerkleRoot[:16])

		cs.eventBus.Publish(events.EventSettlementProofCreated, "chain", map[string]interface{}{
			"block_height": block.Index,
			"merkle_root":  block.MerkleRoot,
		})
	}
}

// ProcessTransactionOnChain registers a transaction on the blockchain.
// This is called automatically when a transaction is created through the gateway.
// Per architecture §5.3.1: orders are registered on-chain with hashed data only.
func (cs *ChainService) ProcessTransactionOnChain(txn *models.Transaction) error {
	orderID := txn.ID

	// 1. Register order on chain (§7.1 OrderRegistry)
	//    Only hashes are stored; amounts and identities stay off-chain
	customerID := txn.PayerAccountID
	orderRecord, err := cs.OrderRegistry.CreateOrder(
		orderID, txn.MerchantID, customerID, txn.SourceAmount, "",
	)
	if err != nil {
		return fmt.Errorf("chain: order registration failed: %w", err)
	}

	// 2. Record initial state transition (§7.3 PaymentStateMachine)
	_, err = cs.StateMachine.Transition(
		orderID,
		"", // No previous state for initial creation
		txn.Status,
		orderRecord.OrderHash,
		core.StandardBlockSigner,
		cs.Blockchain.Height()+1,
	)
	if err != nil {
		log.Printf("[AspiraConsortium·Svc] Initial state transition failed: %v", err)
	}

	// 3. Record audit event (§7.5 AuditLedger)
	operatorHash := contracts.CommitHash(txn.MerchantID + fmt.Sprintf("%d", time.Now().Unix()))
	_, err = cs.AuditLedger.RecordEvent(
		uuid.New().String(),
		orderID,
		"order.created",
		orderRecord.OrderHash,
		operatorHash,
		cs.Blockchain.Height()+1,
	)
	if err != nil {
		log.Printf("[AspiraConsortium·Svc] Audit event recording failed: %v", err)
	}

	// 4. Add transaction reference to the mempool for inclusion in next block
	txHash := core.ComputeTxHash(orderID, "order.created", orderID, time.Now().Unix())
	txRef := core.TxRef{
		TxID:    orderID,
		TxHash:  txHash,
		TxType:  "order.created",
		OrderID: orderID,
	}
	cs.Producer.EnqueueTx(txRef)

	// 5. Publish event
	cs.eventBus.Publish(events.EventOrderCreated, orderID, map[string]interface{}{
		"order_hash":        orderRecord.OrderHash,
		"merchant_hash":     orderRecord.MerchantHash,
		"amount_commitment": orderRecord.AmountCommitment,
		"status":            string(orderRecord.Status),
	})

	log.Printf("[AspiraConsortium·Svc] Transaction %s processed on chain: order=%s, status=%s",
		txn.ID[:8], orderRecord.OrderID, orderRecord.Status)

	return nil
}

// TransitionStateOnChain records a state transition for an order on chain.
func (cs *ChainService) TransitionStateOnChain(orderID string, fromStatus, toStatus models.TransactionStatus) error {
	proofHash := contracts.CommitHash(fmt.Sprintf("%s|%s|%s|%d", orderID, fromStatus, toStatus, time.Now().Unix()))

	transition, err := cs.StateMachine.Transition(
		orderID,
		fromStatus,
		toStatus,
		proofHash,
		core.StandardBlockSigner,
		cs.Blockchain.Height()+1,
	)
	if err != nil {
		return fmt.Errorf("chain: state transition failed: %w", err)
	}

	// Update order registry
	if err := cs.OrderRegistry.UpdateStatus(orderID, toStatus); err != nil {
		log.Printf("[AspiraConsortium·Svc] Order status update on chain failed: %v", err)
	}

	// Record audit event
	operatorHash := contracts.CommitHash(core.StandardBlockSigner)
	cs.AuditLedger.RecordEvent(
		uuid.New().String(),
		orderID,
		"order.state.updated",
		proofHash,
		operatorHash,
		cs.Blockchain.Height()+1,
	)

	// Add to mempool
	txHash := core.ComputeTxHash(orderID, "order.state.updated", orderID, time.Now().Unix())
	cs.Producer.EnqueueTx(core.TxRef{
		TxID:    orderID + "-" + string(toStatus),
		TxHash:  txHash,
		TxType:  "order.state.updated",
		OrderID: orderID,
	})

	cs.eventBus.Publish(events.EventOrderStateUpdated, orderID, map[string]interface{}{
		"from_status": string(fromStatus),
		"to_status":   string(toStatus),
		"transition":  transition,
	})

	log.Printf("[AspiraConsortium·Svc] State transition on chain: %s: %s -> %s", orderID, fromStatus, toStatus)
	return nil
}

// RegisterSettlementOnChain records a payment settlement proof on chain.
func (cs *ChainService) RegisterSettlementOnChain(orderID, channelName, receiptHash string) error {
	receiptURIHash := contracts.CommitHash(fmt.Sprintf("offchain://receipts/%s/%s", channelName, orderID))

	_, err := cs.SettlementProof.RegisterProof(
		orderID, channelName, receiptHash, receiptURIHash, core.StandardBlockSigner,
	)
	if err != nil {
		return fmt.Errorf("chain: settlement proof failed: %w", err)
	}

	// Add to mempool
	txHash := core.ComputeTxHash(orderID, "settlement.proof", orderID, time.Now().Unix())
	cs.Producer.EnqueueTx(core.TxRef{
		TxID:    orderID + "-settlement",
		TxHash:  txHash,
		TxType:  "settlement.proof.created",
		OrderID: orderID,
	})

	cs.eventBus.Publish(events.EventSettlementProofCreated, orderID, map[string]interface{}{
		"channel_name": channelName,
		"receipt_hash": receiptHash,
	})

	return nil
}

// CommitQuoteOnChain locks a quote on chain to prevent reneging.
func (cs *ChainService) CommitQuoteOnChain(quote *models.Quote, providerID, signature string) error {
	_, err := cs.QuoteCommitment.CommitQuote(
		quote.ID, providerID, quote.SourceCurrency, quote.TargetCurrency,
		quote.ExchangeRate, quote.Fee, quote.ExpiresAt.Unix(), signature,
	)
	if err != nil {
		return fmt.Errorf("chain: quote commitment failed: %w", err)
	}

	cs.eventBus.Publish(events.EventQuoteCommitted, quote.ID, map[string]interface{}{
		"quote_id":  quote.ID,
		"merchant":  quote.MerchantID,
		"rate":      quote.ExchangeRate,
	})

	return nil
}

// GetChainStatus returns a summary of the current blockchain state.
func (cs *ChainService) GetChainStatus() *ChainStatus {
	verified := cs.Blockchain.VerifyChain() == nil

	return &ChainStatus{
		Height:           cs.Blockchain.Height(),
		CurrentHash:      cs.Blockchain.CurrentHash(),
		TotalBlocks:      len(cs.Blockchain.GetAllBlocks()),
		OrdersOnChain:    cs.OrderRegistry.GetOrderCount(),
		QuotesCommitted:  0, // TODO: add count method
		StateTransitions: cs.StateMachine.GetTransitionCount(),
		AuditEvents:      cs.AuditLedger.GetEventCount(),
		SettlementProofs: 0, // TODO: add count method
		MerkleAnchors:    cs.MerkleAnchor.GetAnchorCount(),
		MempoolSize:      cs.Producer.MempoolSize(),
		ChainVerified:    verified,
	}
}

// GetOrderOnChain retrieves the on-chain record for an order.
func (cs *ChainService) GetOrderOnChain(orderID string) (*contracts.OrderRecord, error) {
	return cs.OrderRegistry.GetOrder(orderID)
}

// GetOrderStateHistory returns the complete state transition history for an order.
func (cs *ChainService) GetOrderStateHistory(orderID string) []*contracts.StateTransition {
	return cs.StateMachine.GetStateHistory(orderID)
}

// GetAuditTrail returns the audit event chain for an order.
func (cs *ChainService) GetAuditTrail(orderID string) []*contracts.AuditEvent {
	return cs.AuditLedger.GetEventsByOrder(orderID)
}

// GetMerkleProof generates a Merkle proof for a transaction.
func (cs *ChainService) GetMerkleProof(txID string) (*core.MerkleProof, error) {
	return cs.Blockchain.GetTransactionProof(txID)
}

// VerifyChainIntegrity performs a full verification of the blockchain.
func (cs *ChainService) VerifyChainIntegrity() error {
	return cs.Blockchain.VerifyChain()
}

// VerifyAuditChain verifies the audit hash chain for an order.
func (cs *ChainService) VerifyAuditChain(orderID string) (bool, error) {
	return cs.AuditLedger.VerifyEventChain(orderID)
}

// VerifyStateHistory verifies all state transitions for an order.
func (cs *ChainService) VerifyStateHistory(orderID string) (bool, error) {
	return cs.StateMachine.VerifyStateHistory(orderID)
}

// Stop shuts down the blockchain service gracefully.
func (cs *ChainService) Stop() {
	cs.Producer.Stop()
	log.Println("[AspiraConsortium·Svc] Blockchain service stopped")
}
