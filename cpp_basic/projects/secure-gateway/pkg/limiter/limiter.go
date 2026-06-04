package limiter

import (
	"sync"
	"time"
)

// RateLimiter is a simple token-bucket rate limiter.
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string]*bucket
	rate     int           // requests per interval
	interval time.Duration // time window
}

type bucket struct {
	count    int
	resetAt  time.Time
}

// NewRateLimiter creates a new rate limiter with the given rate per minute.
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 120
	}
	return &RateLimiter{
		requests: make(map[string]*bucket),
		rate:     requestsPerMinute,
		interval: time.Minute,
	}
}

// Allow checks if a key is allowed to proceed.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.requests[key]

	if !exists || now.After(b.resetAt) {
		rl.requests[key] = &bucket{
			count:   1,
			resetAt: now.Add(rl.interval),
		}
		return true
	}

	if b.count >= rl.rate {
		return false
	}

	b.count++
	return true
}

// CleanupStale periodically removes stale entries to prevent memory leaks.
func (rl *RateLimiter) CleanupStale(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			rl.mu.Lock()
			now := time.Now()
			for key, b := range rl.requests {
				if now.After(b.resetAt) {
					delete(rl.requests, key)
				}
			}
			rl.mu.Unlock()
		}
	}()
}