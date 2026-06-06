package blockchain

import (
	"net/http"
	"strconv"

	"github.com/aspira/crossborder-payment-gateway/internal/blockchain/core"
	"github.com/gin-gonic/gin"
)

// ChainHandler provides HTTP API endpoints for blockchain interaction.
type ChainHandler struct {
	service *ChainService
}

// NewChainHandler creates a new chain API handler.
func NewChainHandler(service *ChainService) *ChainHandler {
	return &ChainHandler{service: service}
}

// GetChainStatus handles GET /api/v1/chain/status
func (h *ChainHandler) GetChainStatus(c *gin.Context) {
	status := h.service.GetChainStatus()
	c.JSON(http.StatusOK, status)
}

// ListBlocks handles GET /api/v1/chain/blocks
func (h *ChainHandler) ListBlocks(c *gin.Context) {
	blocks := h.service.Blockchain.GetAllBlocks()

	// Convert to lightweight block summaries
	type BlockSummary struct {
		Height     uint64 `json:"height"`
		Hash       string `json:"hash"`
		PrevHash   string `json:"prev_hash"`
		MerkleRoot string `json:"merkle_root"`
		TxCount    int    `json:"tx_count"`
		Timestamp  int64  `json:"timestamp"`
		Signer     string `json:"signer"`
	}

	summaries := make([]BlockSummary, 0, len(blocks))
	for _, b := range blocks {
		summaries = append(summaries, BlockSummary{
			Height:     b.Index,
			Hash:       b.Hash,
			PrevHash:   b.PrevHash,
			MerkleRoot: b.MerkleRoot,
			TxCount:    b.TxCount,
			Timestamp:  b.Timestamp,
			Signer:     b.Signer,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"blocks":     summaries,
		"total":      len(summaries),
		"chain_hash": h.service.Blockchain.CurrentHash(),
	})
}

// GetBlock handles GET /api/v1/chain/blocks/:height
func (h *ChainHandler) GetBlock(c *gin.Context) {
	heightStr := c.Param("height")
	height, err := strconv.ParseUint(heightStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid block height"})
		return
	}

	block, err := h.service.Blockchain.GetBlock(height)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, block)
}

// GetBlockByHash handles GET /api/v1/chain/blocks/hash/:hash
func (h *ChainHandler) GetBlockByHash(c *gin.Context) {
	hash := c.Param("hash")

	block, err := h.service.Blockchain.GetBlockByHash(hash)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, block)
}

// GetOrderOnChain handles GET /api/v1/chain/orders/:id
func (h *ChainHandler) GetOrderOnChain(c *gin.Context) {
	orderID := c.Param("id")

	order, err := h.service.GetOrderOnChain(orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

// GetOrderStateHistory handles GET /api/v1/chain/orders/:id/state-history
func (h *ChainHandler) GetOrderStateHistory(c *gin.Context) {
	orderID := c.Param("id")

	history := h.service.GetOrderStateHistory(orderID)

	c.JSON(http.StatusOK, gin.H{
		"order_id":   orderID,
		"transitions": history,
		"count":      len(history),
	})
}

// GetMerkleProof handles GET /api/v1/chain/proof/:tx_id
func (h *ChainHandler) GetMerkleProof(c *gin.Context) {
	txID := c.Param("tx_id")

	proof, err := h.service.GetMerkleProof(txID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, proof.ToMap())
}

// VerifyMerkleProof handles POST /api/v1/chain/proof/verify
func (h *ChainHandler) VerifyMerkleProof(c *gin.Context) {
	var req struct {
		TxHash string   `json:"tx_hash" binding:"required"`
		Root   string   `json:"root" binding:"required"`
		Proof  []string `json:"proof" binding:"required"`
		Index  int      `json:"index"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	valid := core.VerifyProof(req.TxHash, req.Root, req.Proof, req.Index)

	c.JSON(http.StatusOK, gin.H{
		"valid":   valid,
		"tx_hash": req.TxHash,
		"root":    req.Root,
	})
}

// GetAuditTrail handles GET /api/v1/chain/audit/:order_id
func (h *ChainHandler) GetAuditTrail(c *gin.Context) {
	orderID := c.Param("order_id")

	events := h.service.GetAuditTrail(orderID)

	c.JSON(http.StatusOK, gin.H{
		"order_id": orderID,
		"events":   events,
		"count":    len(events),
	})
}

// VerifyAuditChain handles POST /api/v1/chain/audit/verify/:order_id
func (h *ChainHandler) VerifyAuditChain(c *gin.Context) {
	orderID := c.Param("order_id")

	valid, err := h.service.VerifyAuditChain(orderID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"order_id": orderID,
			"valid":    false,
			"error":    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order_id": orderID,
		"valid":    valid,
	})
}

// VerifyChain handles POST /api/v1/chain/verify
func (h *ChainHandler) VerifyChain(c *gin.Context) {
	err := h.service.VerifyChainIntegrity()

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid":  false,
			"error":  err.Error(),
			"height": h.service.Blockchain.Height(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":  true,
		"height": h.service.Blockchain.Height(),
		"hash":   h.service.Blockchain.CurrentHash(),
	})
}

// GetSettlementProof handles GET /api/v1/chain/settlement/:order_id
func (h *ChainHandler) GetSettlementProof(c *gin.Context) {
	orderID := c.Param("order_id")

	proof, err := h.service.SettlementProof.GetProof(orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, proof)
}

// GetLatestAnchor handles GET /api/v1/chain/anchor/latest
func (h *ChainHandler) GetLatestAnchor(c *gin.Context) {
	anchor := h.service.MerkleAnchor.GetLatestAnchor()
	if anchor == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no anchors yet"})
		return
	}

	c.JSON(http.StatusOK, anchor)
}

// GetAnchors handles GET /api/v1/chain/anchors
func (h *ChainHandler) GetAnchors(c *gin.Context) {
	anchors := h.service.MerkleAnchor.GetAllAnchors()

	c.JSON(http.StatusOK, gin.H{
		"anchors": anchors,
		"count":   len(anchors),
	})
}
