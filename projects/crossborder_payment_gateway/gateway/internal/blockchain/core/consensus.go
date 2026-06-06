package core

import (
	"log"
	"sync"
	"time"
)

// BlockProducerConfig controls block production behavior.
type BlockProducerConfig struct {
	BlockInterval  time.Duration // Minimum time between blocks (default: 5s)
	MaxTxPerBlock  int           // Maximum transactions per block (default: 100)
	MinTxPerBlock  int           // Minimum transactions before producing a block (default: 1)
	Signer         string        // Node identity for block signing
}

// DefaultProducerConfig returns sensible defaults for block production.
func DefaultProducerConfig() BlockProducerConfig {
	return BlockProducerConfig{
		BlockInterval: 5 * time.Second,
		MaxTxPerBlock: 100,
		MinTxPerBlock: 1,
		Signer:        StandardBlockSigner,
	}
}

// BlockProducer manages asynchronous block production.
// Transactions are queued in a mempool and batched into blocks
// either when the mempool is full or the block interval elapses.
type BlockProducer struct {
	config    BlockProducerConfig
	chain     *Blockchain
	mempool   []TxRef
	mu        sync.Mutex
	running   bool
	stopCh    chan struct{}
	newBlockCh chan *Block // Notifies listeners when a new block is produced
	onNewBlock func(*Block) // Callback when a block is produced (for persistence)
}

// NewBlockProducer creates a new block producer.
func NewBlockProducer(chain *Blockchain, config BlockProducerConfig) *BlockProducer {
	if config.BlockInterval == 0 {
		config.BlockInterval = 5 * time.Second
	}
	if config.MaxTxPerBlock == 0 {
		config.MaxTxPerBlock = 100
	}
	if config.MinTxPerBlock == 0 {
		config.MinTxPerBlock = 1
	}
	if config.Signer == "" {
		config.Signer = StandardBlockSigner
	}

	return &BlockProducer{
		config:     config,
		chain:      chain,
		mempool:    make([]TxRef, 0, config.MaxTxPerBlock),
		stopCh:     make(chan struct{}),
		newBlockCh: make(chan *Block, 16),
	}
}

// Start begins asynchronous block production.
func (bp *BlockProducer) Start() {
	bp.mu.Lock()
	if bp.running {
		bp.mu.Unlock()
		return
	}
	bp.running = true
	bp.mu.Unlock()

	go bp.productionLoop()
	log.Printf("[AspiraConsortium·Producer] Started: interval=%v, max_tx=%d",
		bp.config.BlockInterval, bp.config.MaxTxPerBlock)
}

// Stop halts block production.
func (bp *BlockProducer) Stop() {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	if !bp.running {
		return
	}
	bp.running = false
	close(bp.stopCh)
	log.Println("[AspiraConsortium·Producer] Stopped")
}

// EnqueueTx adds a transaction to the mempool.
// If the mempool reaches MaxTxPerBlock, a block is produced immediately.
func (bp *BlockProducer) EnqueueTx(tx TxRef) {
	bp.mu.Lock()
	bp.mempool = append(bp.mempool, tx)
	shouldProduce := len(bp.mempool) >= bp.config.MaxTxPerBlock
	bp.mu.Unlock()

	if shouldProduce {
		bp.produceBlock()
	}
}

// ForceProduce triggers immediate block production regardless of mempool size.
func (bp *BlockProducer) ForceProduce() {
	bp.produceBlock()
}

// SetOnNewBlock sets a callback that is invoked each time a block is produced.
func (bp *BlockProducer) SetOnNewBlock(callback func(*Block)) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.onNewBlock = callback
}

// NewBlockChan returns a channel that receives newly produced blocks.
func (bp *BlockProducer) NewBlockChan() <-chan *Block {
	return bp.newBlockCh
}

// MempoolSize returns the current number of pending transactions.
func (bp *BlockProducer) MempoolSize() int {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return len(bp.mempool)
}

// productionLoop runs in a goroutine, triggering block production
// at the configured interval if the mempool has transactions.
func (bp *BlockProducer) productionLoop() {
	ticker := time.NewTicker(bp.config.BlockInterval)
	defer ticker.Stop()

	for {
		select {
		case <-bp.stopCh:
			// Produce one final block with any remaining transactions
			bp.produceBlock()
			return

		case <-ticker.C:
			bp.mu.Lock()
			mempoolLen := len(bp.mempool)
			bp.mu.Unlock()

			if mempoolLen >= bp.config.MinTxPerBlock {
				bp.produceBlock()
			}
		}
	}
}

// produceBlock takes transactions from the mempool, creates a block,
// adds it to the chain, and notifies listeners.
func (bp *BlockProducer) produceBlock() {
	bp.mu.Lock()

	if len(bp.mempool) == 0 {
		bp.mu.Unlock()
		return
	}

	// Take up to MaxTxPerBlock transactions from the mempool
	count := bp.config.MaxTxPerBlock
	if count > len(bp.mempool) {
		count = len(bp.mempool)
	}

	txBatch := make([]TxRef, count)
	copy(txBatch, bp.mempool[:count])

	// Remove processed transactions from mempool
	bp.mempool = bp.mempool[count:]
	bp.mu.Unlock()

	// Create and add the block to the chain
	block, err := bp.chain.AddBlock(txBatch)
	if err != nil {
		log.Printf("[AspiraConsortium·Producer] Failed to create block: %v", err)
		// Put transactions back in mempool
		bp.mu.Lock()
		bp.mempool = append(txBatch, bp.mempool...)
		bp.mu.Unlock()
		return
	}

	// Notify listeners
	if bp.onNewBlock != nil {
		bp.onNewBlock(block)
	}

	// Non-blocking send to channel
	select {
	case bp.newBlockCh <- block:
	default:
	}

	log.Printf("[AspiraConsortium·Producer] Block %d produced: %d txns (mempool: %d remaining)",
		block.Index, block.TxCount, len(bp.mempool))
}
