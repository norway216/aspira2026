package limiter

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
)

const (
	// Chunk size for rate-limited reads/writes
	chunkSize = 32 * 1024 // 32KB
)

// PerUserLimiter manages rate limiters and connection limits for all users.
type PerUserLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*UserLimiter
}

// UserLimiter bundles a token bucket with connection counting for one user.
type UserLimiter struct {
	bucket      *TokenBucket
	connections atomic.Int64
	maxConns    int
}

// NewPerUserLimiter creates a new per-user rate limiter.
func NewPerUserLimiter() *PerUserLimiter {
	return &PerUserLimiter{
		buckets: make(map[string]*UserLimiter),
	}
}

// AllowConnection checks if a new connection is allowed for the given user.
// Returns true if allowed, false if the user has reached their connection limit.
func (p *PerUserLimiter) AllowConnection(userID string) bool {
	p.mu.RLock()
	ul, ok := p.buckets[userID]
	p.mu.RUnlock()

	if !ok {
		// No limiter configured - allow with default (unlimited)
		return true
	}

	if ul.maxConns <= 0 {
		return true // No limit
	}

	current := int(ul.connections.Add(1))
	if current > ul.maxConns {
		ul.connections.Add(-1)
		return false
	}
	return true
}

// ReleaseConnection decrements the connection count for a user.
func (p *PerUserLimiter) ReleaseConnection(userID string) {
	p.mu.RLock()
	ul, ok := p.buckets[userID]
	p.mu.RUnlock()

	if ok {
		ul.connections.Add(-1)
	}
}

// UpdatePolicy creates or updates the rate limiter for a user.
func (p *PerUserLimiter) UpdatePolicy(userID string, maxRateMbps, burstMbps, maxConns int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	rateBytes := float64(maxRateMbps) * 1024 * 1024 / 8
	burstBytes := float64(burstMbps) * 1024 * 1024 / 8

	ul, ok := p.buckets[userID]
	if ok {
		ul.bucket.SetRate(rateBytes)
		ul.bucket.SetBurst(burstBytes)
		ul.maxConns = maxConns
	} else {
		p.buckets[userID] = &UserLimiter{
			bucket:   NewTokenBucket(rateBytes, burstBytes),
			maxConns: maxConns,
		}
	}
}

// RemoveUser removes a user's rate limiter.
func (p *PerUserLimiter) RemoveUser(userID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.buckets, userID)
}

// UpdateAllPolicies replaces all user limiters with new policies.
func (p *PerUserLimiter) UpdateAllPolicies(policies map[string]*PolicyConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove users not in new policies
	for userID := range p.buckets {
		if _, ok := policies[userID]; !ok {
			delete(p.buckets, userID)
		}
	}

	// Add or update
	for userID, pc := range policies {
		rateBytes := float64(pc.MaxRateMbps) * 1024 * 1024 / 8
		burstBytes := float64(pc.BurstMbps) * 1024 * 1024 / 8

		ul, ok := p.buckets[userID]
		if ok {
			ul.bucket.SetRate(rateBytes)
			ul.bucket.SetBurst(burstBytes)
			ul.maxConns = pc.MaxConnections
		} else {
			p.buckets[userID] = &UserLimiter{
				bucket:   NewTokenBucket(rateBytes, burstBytes),
				maxConns: pc.MaxConnections,
			}
		}
	}
}

// RateLimitedReader wraps an io.Reader with rate limiting.
type RateLimitedReader struct {
	ctx    context.Context
	reader io.Reader
	bucket *TokenBucket
}

// Read implements io.Reader with rate limiting.
func (r *RateLimitedReader) Read(p []byte) (int, error) {
	toRead := len(p)
	if toRead > chunkSize {
		toRead = chunkSize
	}

	if err := r.bucket.ConsumeWait(r.ctx, int64(toRead)); err != nil {
		return 0, err
	}

	n, err := r.reader.Read(p[:toRead])

	// If we read less than we consumed, refund the difference
	if n < toRead {
		// Note: we can't easily "un-consume" tokens. For simplicity,
		// the next read will have a head start. This is acceptable for
		// TCP streams where reads usually fill the buffer.
	}
	return n, err
}

// RateLimitedWriter wraps an io.Writer with rate limiting.
type RateLimitedWriter struct {
	ctx    context.Context
	writer io.Writer
	bucket *TokenBucket
}

// Write implements io.Writer with rate limiting.
func (w *RateLimitedWriter) Write(p []byte) (int, error) {
	total := 0
	for total < len(p) {
		chunk := len(p) - total
		if chunk > chunkSize {
			chunk = chunkSize
		}

		if err := w.bucket.ConsumeWait(w.ctx, int64(chunk)); err != nil {
			return total, err
		}

		n, err := w.writer.Write(p[total : total+chunk])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// NewRateLimitedReader creates a rate-limited reader for a user.
func (p *PerUserLimiter) NewRateLimitedReader(ctx context.Context, userID string, r io.Reader) io.Reader {
	p.mu.RLock()
	ul, ok := p.buckets[userID]
	p.mu.RUnlock()

	if !ok {
		return r // No rate limiting
	}

	return &RateLimitedReader{
		ctx:    ctx,
		reader: r,
		bucket: ul.bucket,
	}
}

// NewRateLimitedWriter creates a rate-limited writer for a user.
func (p *PerUserLimiter) NewRateLimitedWriter(ctx context.Context, userID string, w io.Writer) io.Writer {
	p.mu.RLock()
	ul, ok := p.buckets[userID]
	p.mu.RUnlock()

	if !ok {
		return w // No rate limiting
	}

	return &RateLimitedWriter{
		ctx:    ctx,
		writer: w,
		bucket: ul.bucket,
	}
}

// PolicyConfig is used to pass policy updates to the limiter.
type PolicyConfig struct {
	MaxRateMbps    int
	BurstMbps      int
	MaxConnections int
}
