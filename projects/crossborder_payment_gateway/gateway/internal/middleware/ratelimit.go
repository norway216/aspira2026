package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
	rate     float64
	capacity int
}

type tokenBucket struct {
	tokens   float64
	lastTime time.Time
}

func NewRateLimiter(rate float64, capacity int) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*tokenBucket),
		rate:     rate,
		capacity: capacity,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		for ip, bucket := range rl.buckets {
			if time.Since(bucket.lastTime) > 5*time.Minute {
				delete(rl.buckets, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[ip]
	now := time.Now()

	if !exists {
		bucket = &tokenBucket{tokens: float64(rl.capacity), lastTime: now}
		rl.buckets[ip] = bucket
	}

	elapsed := now.Sub(bucket.lastTime).Seconds()
	bucket.tokens += elapsed * rl.rate
	if bucket.tokens > float64(rl.capacity) {
		bucket.tokens = float64(rl.capacity)
	}
	bucket.lastTime = now

	if bucket.tokens >= 1 {
		bucket.tokens--
		return true
	}
	return false
}

func RateLimit(rate float64, capacity int) gin.HandlerFunc {
	rl := NewRateLimiter(rate, capacity)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !rl.allow(ip) {
			c.AbortWithStatusJSON(429, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
