package limiter

import (
	"context"
	"math"
	"sync"
	"time"
)

// TokenBucket implements the token bucket rate limiting algorithm.
// It allows bursts up to the bucket capacity while maintaining a long-term rate limit.
type TokenBucket struct {
	mu         sync.Mutex
	rate       float64 // tokens per second (bytes per second)
	capacity   float64 // maximum tokens (burst capacity in bytes)
	tokens     float64 // current tokens
	lastRefill time.Time
}

// NewTokenBucket creates a new token bucket.
// rateBytesPerSec: sustained rate in bytes per second
// burstBytes: maximum burst in bytes
func NewTokenBucket(rateBytesPerSec, burstBytes float64) *TokenBucket {
	if burstBytes < rateBytesPerSec {
		burstBytes = rateBytesPerSec // burst must be at least the rate
	}
	return &TokenBucket{
		rate:       rateBytesPerSec,
		capacity:   burstBytes,
		tokens:     burstBytes, // Start with full bucket
		lastRefill: time.Now(),
	}
}

// refill adds tokens based on elapsed time. Must be called with mu held.
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens = math.Min(tb.capacity, tb.tokens+elapsed*tb.rate)
	tb.lastRefill = now
}

// Consume attempts to consume the given number of bytes from the bucket.
// Returns true if enough tokens were available, false otherwise.
func (tb *TokenBucket) Consume(bytes int64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= float64(bytes) {
		tb.tokens -= float64(bytes)
		return true
	}
	return false
}

// ConsumeWait attempts to consume tokens, blocking until enough are available
// or the context is cancelled.
func (tb *TokenBucket) ConsumeWait(ctx context.Context, bytes int64) error {
	for {
		if tb.Consume(bytes) {
			return nil
		}

		// Calculate how long to wait for enough tokens
		tb.mu.Lock()
		tb.refill()
		needed := float64(bytes) - tb.tokens
		waitTime := time.Duration(needed / tb.rate * float64(time.Second))
		tb.mu.Unlock()

		// Add a small buffer to the wait time
		waitTime += 10 * time.Millisecond
		if waitTime > time.Second {
			waitTime = time.Second // Cap at 1s to check for cancellation
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Try again
		}
	}
}

// Rate returns the configured rate in bytes per second.
func (tb *TokenBucket) Rate() float64 {
	return tb.rate
}

// Capacity returns the configured burst capacity in bytes.
func (tb *TokenBucket) Capacity() float64 {
	return tb.capacity
}

// Available returns the current number of available tokens.
func (tb *TokenBucket) Available() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	return tb.tokens
}

// SetRate updates the rate limit. Thread-safe.
func (tb *TokenBucket) SetRate(rateBytesPerSec float64) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.rate = rateBytesPerSec
}

// SetBurst updates the burst capacity. Thread-safe.
func (tb *TokenBucket) SetBurst(burstBytes float64) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.capacity = burstBytes
	if tb.tokens > burstBytes {
		tb.tokens = burstBytes
	}
}
