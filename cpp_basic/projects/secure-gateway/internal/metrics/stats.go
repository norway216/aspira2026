package metrics

import (
	"sync/atomic"
	"time"
)

// ConnectionStats tracks connection and traffic statistics for a node.
type ConnectionStats struct {
	activeConns     int64
	totalConns      int64
	rxBytes         int64
	txBytes         int64
	rxBytesPerSec   int64
	txBytesPerSec   int64
	lastRxBytes     int64
	lastTxBytes     int64
	lastSampleTime  time.Time
	authFailedTotal int64
	errorTotal      int64
}

// NewConnectionStats creates a new stats tracker.
func NewConnectionStats() *ConnectionStats {
	return &ConnectionStats{
		lastSampleTime: time.Now(),
	}
}

// IncActiveConns increments the active connection counter.
func (s *ConnectionStats) IncActiveConns() {
	atomic.AddInt64(&s.activeConns, 1)
	atomic.AddInt64(&s.totalConns, 1)
}

// DecActiveConns decrements the active connection counter.
func (s *ConnectionStats) DecActiveConns() {
	atomic.AddInt64(&s.activeConns, -1)
}

// AddRxBytes adds received bytes.
func (s *ConnectionStats) AddRxBytes(n int64) {
	atomic.AddInt64(&s.rxBytes, n)
}

// AddTxBytes adds transmitted bytes.
func (s *ConnectionStats) AddTxBytes(n int64) {
	atomic.AddInt64(&s.txBytes, n)
}

// IncAuthFailed increments auth failure count.
func (s *ConnectionStats) IncAuthFailed() {
	atomic.AddInt64(&s.authFailedTotal, 1)
}

// IncErrors increments error count.
func (s *ConnectionStats) IncErrors() {
	atomic.AddInt64(&s.errorTotal, 1)
}

// Snapshot returns the current stats and calculates per-second rates.
func (s *ConnectionStats) Snapshot() (activeConns int64, rxPerSec, txPerSec int64) {
	activeConns = atomic.LoadInt64(&s.activeConns)
	currentRx := atomic.LoadInt64(&s.rxBytes)
	currentTx := atomic.LoadInt64(&s.txBytes)
	now := time.Now()

	elapsed := now.Sub(s.lastSampleTime).Seconds()
	if elapsed > 0 {
		rxPerSec = int64(float64(currentRx-s.lastRxBytes) / elapsed)
		txPerSec = int64(float64(currentTx-s.lastTxBytes) / elapsed)
	}

	// Update for next sample
	s.lastRxBytes = currentRx
	s.lastTxBytes = currentTx
	s.lastSampleTime = now

	return
}

// GetAll returns all stats values.
func (s *ConnectionStats) GetAll() (activeConns, totalConns, rxBytes, txBytes, authFailed, errors int64) {
	activeConns = atomic.LoadInt64(&s.activeConns)
	totalConns = atomic.LoadInt64(&s.totalConns)
	rxBytes = atomic.LoadInt64(&s.rxBytes)
	txBytes = atomic.LoadInt64(&s.txBytes)
	authFailed = atomic.LoadInt64(&s.authFailedTotal)
	errors = atomic.LoadInt64(&s.errorTotal)
	return
}