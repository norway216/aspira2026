package client

import (
	"math/rand"
	"time"
)

// RetryConfig holds configuration for retry logic.
type RetryConfig struct {
	MaxRetries    int
	BaseInterval  time.Duration
	MaxInterval   time.Duration
	JitterFactor  float64
}

// DefaultRetryConfig returns a default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   0, // unlimited (loop forever)
		BaseInterval: 1 * time.Second,
		MaxInterval:  60 * time.Second,
		JitterFactor: 0.5,
	}
}

// DoWithRetry executes a function with exponential backoff and jitter.
func DoWithRetry(fn func() error, cfg RetryConfig) error {
	attempt := 0
	for {
		err := fn()
		if err == nil {
			return nil
		}

		attempt++
		if cfg.MaxRetries > 0 && attempt > cfg.MaxRetries {
			return err
		}

		// Exponential backoff with jitter
		multiplier := int64(1) << uint(attempt-1)
		delay := time.Duration(float64(cfg.BaseInterval) * float64(multiplier))
		if delay > cfg.MaxInterval {
			delay = cfg.MaxInterval
		}

		// Add jitter
		jitter := time.Duration(float64(delay) * cfg.JitterFactor * (rand.Float64()*2 - 1))
		delay += jitter
		if delay < 0 {
			delay = cfg.MaxInterval / 2
		}

		time.Sleep(delay)
	}
}