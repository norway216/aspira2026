package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
)

// MerkleTree is a binary Merkle tree using SHA-256.
// It supports proof generation and verification for any leaf transaction.
type MerkleTree struct {
	Leaves []string   // Transaction hashes at the leaf level
	Layers [][]string // All layers of the tree; Layers[0] = leaves, Layers[n] = root (single element)
	Root   string
}

// MerkleProof contains the information needed to verify a leaf's inclusion.
type MerkleProof struct {
	TxHash string   `json:"tx_hash"`
	Index  int      `json:"index"`
	Proof  []string `json:"proof"`
	Root   string   `json:"root"`
}

// NewMerkleTree builds a Merkle tree from transaction hashes.
// If the number of leaves is odd, the last leaf is duplicated (perfect binary tree).
func NewMerkleTree(txHashes []string) *MerkleTree {
	if len(txHashes) == 0 {
		// Empty tree — use a single zero-hash leaf
		txHashes = []string{zeroHash()}
	}

	mt := &MerkleTree{
		Leaves: make([]string, len(txHashes)),
		Layers: make([][]string, 0),
	}
	copy(mt.Leaves, txHashes)

	// Build layers bottom-up
	currentLayer := make([]string, len(mt.Leaves))
	copy(currentLayer, mt.Leaves)
	mt.Layers = append(mt.Layers, currentLayer)

	for len(currentLayer) > 1 {
		nextLayer := buildParentLayer(currentLayer)
		mt.Layers = append(mt.Layers, nextLayer)
		currentLayer = nextLayer
	}

	if len(currentLayer) == 1 {
		mt.Root = currentLayer[0]
	} else {
		mt.Root = zeroHash()
	}

	return mt
}

// GenerateProof creates a Merkle proof for a specific transaction hash.
// Returns nil if the hash is not found in the leaves.
func (mt *MerkleTree) GenerateProof(txHash string) *MerkleProof {
	// Find the index of the leaf
	index := -1
	for i, leaf := range mt.Leaves {
		if leaf == txHash {
			index = i
			break
		}
	}
	if index < 0 {
		return nil
	}

	proof := make([]string, 0)
	currentIdx := index

	// Walk up the tree, collecting sibling hashes
	for layerIdx := 0; layerIdx < len(mt.Layers)-1; layerIdx++ {
		layer := mt.Layers[layerIdx]
		siblingIdx := currentIdx ^ 1 // XOR toggles the last bit (0<->1, 2<->3, ...)

		if siblingIdx < len(layer) {
			proof = append(proof, layer[siblingIdx])
		} else {
			// Odd leaf — duplicate self
			proof = append(proof, layer[currentIdx])
		}

		currentIdx = currentIdx / 2
	}

	return &MerkleProof{
		TxHash: txHash,
		Index:  index,
		Proof:  proof,
		Root:   mt.Root,
	}
}

// VerifyProof verifies a Merkle inclusion proof.
// Returns true if the proof is valid for the given root.
func VerifyProof(txHash string, root string, proof []string, index int) bool {
	currentHash := txHash
	currentIdx := index

	for _, siblingHash := range proof {
		siblingIdx := currentIdx ^ 1

		var combined string
		if siblingIdx > currentIdx {
			// Sibling is to the right
			combined = currentHash + siblingHash
		} else {
			// Sibling is to the left
			combined = siblingHash + currentHash
		}

		hash := sha256Hex(combined)
		currentHash = hash
		currentIdx = currentIdx / 2
	}

	return currentHash == root
}

// GetHeight returns the height of the tree (number of layers - 1).
func (mt *MerkleTree) GetHeight() int {
	return len(mt.Layers) - 1
}

// GetLeafCount returns the number of leaves in the tree.
func (mt *MerkleTree) GetLeafCount() int {
	return len(mt.Leaves)
}

// buildParentLayer computes the parent layer from a child layer.
// If odd number of elements, the last one is duplicated.
func buildParentLayer(children []string) []string {
	parentSize := (len(children) + 1) / 2
	parents := make([]string, 0, parentSize)

	for i := 0; i < len(children); i += 2 {
		left := children[i]
		right := left
		if i+1 < len(children) {
			right = children[i+1]
		}
		combined := left + right
		parents = append(parents, sha256Hex(combined))
	}

	return parents
}

// sha256Hex computes the SHA-256 hash of a string and returns it as hex.
func sha256Hex(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// zeroHash returns a 64-character zero hash string.
func zeroHash() string {
	return hex.EncodeToString(make([]byte, 32))
}

// ComputeTxHash computes a transaction hash from its ID and type.
func ComputeTxHash(txID, txType, orderID string, timestamp int64) string {
	input := fmt.Sprintf("%s|%s|%s|%d", txID, txType, orderID, timestamp)
	return sha256Hex(input)
}

// MerkleProofToJSON converts a MerkleProof to a map suitable for JSON serialization.
func (mp *MerkleProof) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"tx_hash":       mp.TxHash,
		"index":         mp.Index,
		"proof":         mp.Proof,
		"root":          mp.Root,
		"proof_length":  len(mp.Proof),
		"tree_depth":    int(math.Ceil(math.Log2(float64(mp.Index + 1)))) + 1,
	}
}
