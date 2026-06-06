package contracts

import (
	"sync"
	"time"
)

// MerkleAnchorRecord stores a periodic Merkle root anchor on chain.
// Per architecture §7.5 and §3: Merkle roots are periodically anchored
// to provide a public verifiable checkpoint. In production, these anchors
// would be published to a public L2 blockchain for cross-chain verification.
type MerkleAnchorRecord struct {
	ID          int64  `json:"id"`
	BlockHeight uint64 `json:"block_height"`
	MerkleRoot  string `json:"merkle_root"`
	AnchoredAt  int64  `json:"anchored_at"`
}

// MerkleAnchor is the on-chain Merkle root anchoring contract.
// Provides periodic checkpoints that can be used for cross-chain anchoring
// to public blockchains for additional trust guarantees.
type MerkleAnchor struct {
	mu      sync.RWMutex
	anchors []*MerkleAnchorRecord
}

// NewMerkleAnchor creates a new anchor registry.
func NewMerkleAnchor() *MerkleAnchor {
	return &MerkleAnchor{
		anchors: make([]*MerkleAnchorRecord, 0),
	}
}

// AnchorRoot stores a Merkle root checkpoint.
func (ma *MerkleAnchor) AnchorRoot(blockHeight uint64, merkleRoot string) *MerkleAnchorRecord {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	record := &MerkleAnchorRecord{
		BlockHeight: blockHeight,
		MerkleRoot:  merkleRoot,
		AnchoredAt:  time.Now().Unix(),
	}

	// Assign a sequential ID
	record.ID = int64(len(ma.anchors) + 1)
	ma.anchors = append(ma.anchors, record)

	return record
}

// GetLatestAnchor returns the most recent anchor.
func (ma *MerkleAnchor) GetLatestAnchor() *MerkleAnchorRecord {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	if len(ma.anchors) == 0 {
		return nil
	}
	return ma.anchors[len(ma.anchors)-1]
}

// GetAnchor retrieves a specific anchor by block height.
func (ma *MerkleAnchor) GetAnchor(blockHeight uint64) (*MerkleAnchorRecord, error) {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	for _, anchor := range ma.anchors {
		if anchor.BlockHeight == blockHeight {
			return anchor, nil
		}
	}
	return nil, nil // Not found is not an error
}

// GetAllAnchors returns all Merkle root anchors.
func (ma *MerkleAnchor) GetAllAnchors() []*MerkleAnchorRecord {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	cp := make([]*MerkleAnchorRecord, len(ma.anchors))
	copy(cp, ma.anchors)
	return cp
}

// GetAnchorCount returns the number of anchors.
func (ma *MerkleAnchor) GetAnchorCount() int {
	ma.mu.RLock()
	defer ma.mu.RUnlock()
	return len(ma.anchors)
}
