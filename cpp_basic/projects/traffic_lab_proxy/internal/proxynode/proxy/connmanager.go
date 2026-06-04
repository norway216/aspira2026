package proxy

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ConnectionInfo tracks metadata for an active connection.
type ConnectionInfo struct {
	ID        string
	UserID    string
	Target    string
	StartTime time.Time
}

// ConnectionManager limits the number of concurrent connections using a semaphore pattern.
type ConnectionManager struct {
	mu          sync.RWMutex
	connections map[string]*ConnectionInfo
	sem         chan struct{}
	maxConns    int
	count       atomic.Int64
	idCounter   atomic.Int64
}

// NewConnectionManager creates a connection manager with the given maximum connections.
func NewConnectionManager(maxConns int) *ConnectionManager {
	if maxConns <= 0 {
		maxConns = 1000
	}
	return &ConnectionManager{
		connections: make(map[string]*ConnectionInfo),
		sem:         make(chan struct{}, maxConns),
		maxConns:    maxConns,
	}
}

// Acquire tries to reserve a connection slot. Returns a connection ID and nil error
// on success. Returns error if the context is cancelled or the server is at capacity.
func (cm *ConnectionManager) Acquire(ctx context.Context) (string, error) {
	select {
	case cm.sem <- struct{}{}:
		// Slot acquired
	case <-ctx.Done():
		return "", ctx.Err()
	}

	id := cm.idCounter.Add(1)
	connID := fmt.Sprintf("conn-%d-%d", time.Now().UnixNano(), id)
	cm.count.Add(1)

	cm.mu.Lock()
	cm.connections[connID] = &ConnectionInfo{
		ID:        connID,
		StartTime: time.Now(),
	}
	cm.mu.Unlock()

	return connID, nil
}

// Release frees a connection slot. Must be called for every successful Acquire.
func (cm *ConnectionManager) Release(connID string) {
	cm.mu.Lock()
	delete(cm.connections, connID)
	cm.mu.Unlock()

	cm.count.Add(-1)
	<-cm.sem // Release semaphore slot
}

// SetUserID associates a user ID with a connection for tracking purposes.
func (cm *ConnectionManager) SetUserID(connID, userID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if ci, ok := cm.connections[connID]; ok {
		ci.UserID = userID
	}
}

// SetTarget associates a target address with a connection.
func (cm *ConnectionManager) SetTarget(connID, target string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if ci, ok := cm.connections[connID]; ok {
		ci.Target = target
	}
}

// CurrentConnections returns the current number of active connections.
func (cm *ConnectionManager) CurrentConnections() int {
	return int(cm.count.Load())
}

// MaxConnections returns the maximum allowed connections.
func (cm *ConnectionManager) MaxConnections() int {
	return cm.maxConns
}

// Utilization returns the connection utilization ratio (0.0 - 1.0).
func (cm *ConnectionManager) Utilization() float64 {
	return float64(cm.CurrentConnections()) / float64(cm.maxConns)
}

// ActiveConnections returns a copy of all active connection info.
func (cm *ConnectionManager) ActiveConnections() []ConnectionInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make([]ConnectionInfo, 0, len(cm.connections))
	for _, ci := range cm.connections {
		result = append(result, *ci)
	}
	return result
}

// Drain gracefully waits for active connections to close within the given deadline.
// Returns the number of connections remaining after the deadline.
func (cm *ConnectionManager) Drain(deadline time.Duration) int {
	deadlineCh := time.After(deadline)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		count := cm.CurrentConnections()
		if count == 0 {
			return 0
		}

		select {
		case <-deadlineCh:
			return count
		case <-ticker.C:
			// Continue waiting
		}
	}
}
