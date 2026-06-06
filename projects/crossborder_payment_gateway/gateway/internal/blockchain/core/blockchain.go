package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"
)

// GenesisPrevHash is the all-zero hash used as the previous hash for the genesis block.
const GenesisPrevHash = "0000000000000000000000000000000000000000000000000000000000000000"

// Blockchain manages the Append-only chain of blocks.
// It stores blocks in memory for fast access and provides persistence hooks.
type Blockchain struct {
	mu         sync.RWMutex
	blocks     []*Block           // In-memory block storage (indexed by height)
	height     uint64             // Current chain height (last block index)
	currentHash string            // Hash of the most recent block
	txIndex    map[string]*TxLoc  // txID → (block height, tx index within block)
}

// TxLoc locates a transaction reference in the chain.
type TxLoc struct {
	BlockHeight uint64
	TxIndex     int
}

// NewBlockchain creates a new blockchain with a genesis block.
func NewBlockchain() *Blockchain {
	bc := &Blockchain{
		blocks:  make([]*Block, 0),
		txIndex: make(map[string]*TxLoc),
	}

	// Create genesis block
	genesis := bc.createGenesisBlock()
	bc.blocks = append(bc.blocks, genesis)
	bc.height = 0
	bc.currentHash = genesis.Hash

	log.Printf("[Aspira Consortium Chain] Genesis block created: height=0, hash=%s", genesis.Hash[:16])
	return bc
}

// createGenesisBlock creates the first block in the chain.
func (bc *Blockchain) createGenesisBlock() *Block {
	genesisTx := TxRef{
		TxID:    "genesis",
		TxHash:  ComputeTxHash("genesis", "genesis", "", time.Now().Unix()),
		TxType:  "genesis",
		OrderID: "",
	}

	tree := NewMerkleTree([]string{genesisTx.TxHash})
	block := NewBlock(0, GenesisPrevHash, []TxRef{genesisTx}, tree.Root, StandardBlockSigner)
	block.SetStateRoot(zeroHash())

	return block
}

// AddBlock creates a new block with the given transactions and appends it to the chain.
// Returns the new block and any error.
func (bc *Blockchain) AddBlock(txRefs []TxRef) (*Block, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if len(txRefs) == 0 {
		return nil, fmt.Errorf("cannot create block with no transactions")
	}

	// Build Merkle tree from transaction hashes
	txHashes := make([]string, len(txRefs))
	for i, tx := range txRefs {
		txHashes[i] = tx.TxHash
	}
	tree := NewMerkleTree(txHashes)

	newHeight := bc.height + 1
	block := NewBlock(newHeight, bc.currentHash, txRefs, tree.Root, StandardBlockSigner)

	// Compute state root from contract state (simplified: hash of block height + merkle root)
	stateInput := fmt.Sprintf("%d|%s|%d", newHeight, tree.Root, len(txRefs))
	stateHash := sha256.Sum256([]byte(stateInput))
	block.SetStateRoot(hex.EncodeToString(stateHash[:]))

	// Verify the block before adding
	if err := block.Verify(); err != nil {
		return nil, fmt.Errorf("block verification failed: %w", err)
	}

	// Append to chain
	bc.blocks = append(bc.blocks, block)
	bc.height = newHeight
	bc.currentHash = block.Hash

	// Index transactions
	for i, tx := range txRefs {
		bc.txIndex[tx.TxID] = &TxLoc{
			BlockHeight: newHeight,
			TxIndex:     i,
		}
	}

	log.Printf("[Aspira Consortium Chain] Block %d created: %d txns, merkle=%s, hash=%s",
		newHeight, block.TxCount, block.MerkleRoot[:16], block.Hash[:16])

	return block, nil
}

// GetBlock returns the block at the given height.
func (bc *Blockchain) GetBlock(height uint64) (*Block, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if height > bc.height {
		return nil, fmt.Errorf("block height %d exceeds chain height %d", height, bc.height)
	}

	return bc.blocks[height], nil
}

// GetBlockByHash returns the block with the given hash.
func (bc *Blockchain) GetBlockByHash(hash string) (*Block, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	for _, block := range bc.blocks {
		if block.Hash == hash {
			return block, nil
		}
	}

	return nil, fmt.Errorf("block not found: %s", hash[:16])
}

// GetLatestBlock returns the most recent block.
func (bc *Blockchain) GetLatestBlock() *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if len(bc.blocks) == 0 {
		return nil
	}
	return bc.blocks[len(bc.blocks)-1]
}

// GetTransactionProof generates a Merkle proof for a transaction.
func (bc *Blockchain) GetTransactionProof(txID string) (*MerkleProof, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	loc, ok := bc.txIndex[txID]
	if !ok {
		return nil, fmt.Errorf("transaction not found on chain: %s", txID)
	}

	block := bc.blocks[loc.BlockHeight]

	txHashes := make([]string, len(block.Transactions))
	for i, tx := range block.Transactions {
		txHashes[i] = tx.TxHash
	}

	tree := NewMerkleTree(txHashes)
	proof := tree.GenerateProof(txHashes[loc.TxIndex])
	if proof == nil {
		return nil, fmt.Errorf("failed to generate Merkle proof for tx %s", txID)
	}

	return proof, nil
}

// VerifyChain validates the entire chain from genesis to tip.
// Returns nil if all blocks are valid, or an error describing the first invalid block.
func (bc *Blockchain) VerifyChain() error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	for i := uint64(0); i <= bc.height; i++ {
		block := bc.blocks[i]

		// Verify internal consistency
		if err := block.Verify(); err != nil {
			return fmt.Errorf("block %d verification failed: %w", i, err)
		}

		// Verify chain continuity
		if i > 0 {
			prevBlock := bc.blocks[i-1]
			if block.PrevHash != prevBlock.Hash {
				return fmt.Errorf("chain broken at block %d: prev_hash %s != expected %s",
					i, block.PrevHash[:16], prevBlock.Hash[:16])
			}
		}
	}

	return nil
}

// Height returns the current chain height.
func (bc *Blockchain) Height() uint64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.height
}

// CurrentHash returns the hash of the latest block.
func (bc *Blockchain) CurrentHash() string {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.currentHash
}

// GetAllBlocks returns a copy of all blocks in memory.
func (bc *Blockchain) GetAllBlocks() []*Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	cp := make([]*Block, len(bc.blocks))
	copy(cp, bc.blocks)
	return cp
}

// IsTxOnChain checks if a transaction ID is recorded on the chain.
func (bc *Blockchain) IsTxOnChain(txID string) bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	_, ok := bc.txIndex[txID]
	return ok
}
