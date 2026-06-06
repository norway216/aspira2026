package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// TxRef is a lightweight reference to a transaction stored on-chain.
// The actual transaction data (amounts, identities, bank accounts) is stored
// off-chain in SQLite. The chain only stores hashes and references.
type TxRef struct {
	TxID    string `json:"tx_id"`    // Transaction ID (matches database transactions.id)
	TxHash  string `json:"tx_hash"`  // SHA-256 hash of the transaction
	TxType  string `json:"tx_type"`  // Event type (order.create, payment.execute, etc.)
	OrderID string `json:"order_id"` // Associated order ID
}

// Block represents a single block in the Aspira Consortium Chain.
// Per architecture §3, blocks contain Merkle roots of batched transactions,
// but never raw sensitive data (amounts, identities, bank details).
type Block struct {
	Index        uint64  `json:"index"`
	Timestamp    int64   `json:"timestamp"`
	Transactions []TxRef `json:"transactions"`
	PrevHash     string  `json:"prev_hash"`
	MerkleRoot   string  `json:"merkle_root"`
	StateRoot    string  `json:"state_root"`
	Hash         string  `json:"hash"`
	Nonce        uint64  `json:"nonce"`
	Signer       string  `json:"signer"`
	TxCount      int     `json:"tx_count"`
}

// NewBlock creates a new block with the given parameters.
// The block hash is computed as SHA-256(index + prevHash + merkleRoot + timestamp).
func NewBlock(index uint64, prevHash string, txRefs []TxRef, merkleRoot string, signer string) *Block {
	timestamp := time.Now().Unix()

	txCount := len(txRefs)
	dbTxRefs := make([]TxRef, txCount)
	copy(dbTxRefs, txRefs)

	block := &Block{
		Index:        index,
		Timestamp:    timestamp,
		Transactions: dbTxRefs,
		PrevHash:     prevHash,
		MerkleRoot:   merkleRoot,
		StateRoot:    "", // Computed after state transitions
		Nonce:        0,
		Signer:       signer,
		TxCount:      txCount,
	}

	block.Hash = block.computeHash()
	return block
}

// computeHash calculates the block hash from its header fields.
// Hash = SHA-256(index + prevHash + merkleRoot + stateRoot + timestamp + nonce + signer)
func (b *Block) computeHash() string {
	input := fmt.Sprintf("%d|%s|%s|%s|%d|%d|%s",
		b.Index, b.PrevHash, b.MerkleRoot, b.StateRoot,
		b.Timestamp, b.Nonce, b.Signer)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// Verify checks the block's internal consistency:
// - Hash matches recomputed hash
// - Merkle root matches the transaction list
// - Non-negative index (genesis = 0)
func (b *Block) Verify() error {
	// Verify hash integrity
	expectedHash := b.computeHash()
	if b.Hash != expectedHash {
		return fmt.Errorf("block hash mismatch: expected %s, got %s", expectedHash, b.Hash)
	}

	// Verify Merkle root against transaction list
	txHashes := make([]string, len(b.Transactions))
	for i, tx := range b.Transactions {
		txHashes[i] = tx.TxHash
	}
	tree := NewMerkleTree(txHashes)
	if b.MerkleRoot != tree.Root {
		return fmt.Errorf("merkle root mismatch: expected %s, got %s", tree.Root, b.MerkleRoot)
	}

	// Verify transaction count
	if b.TxCount != len(b.Transactions) {
		return fmt.Errorf("tx_count mismatch: header says %d, actual %d", b.TxCount, len(b.Transactions))
	}

	return nil
}

// SetStateRoot sets the state root hash after all contract state transitions.
func (b *Block) SetStateRoot(stateRoot string) {
	b.StateRoot = stateRoot
	b.Hash = b.computeHash() // Recompute hash since stateRoot affects it
}

// StandardBlockSigner is the default signer for PoA consensus (aspira-core-node).
const StandardBlockSigner = "aspira-core-node"
