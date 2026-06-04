package counter

import (
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// TrafficCounter tracks per-user and node-wide traffic statistics using atomic counters.
type TrafficCounter struct {
	mu       sync.RWMutex
	users    map[string]*userTraffic
	nodeWide nodeTraffic
}

type userTraffic struct {
	upload    atomic.Int64
	download  atomic.Int64
	conns     atomic.Int64
}

type nodeTraffic struct {
	totalBytes atomic.Int64
	lastBytes  atomic.Int64 // For rate calculation
	lastTime   time.Time
	mu         sync.Mutex
}

// TrafficSnapshot captures traffic data for a single user since the last snapshot.
type TrafficSnapshot struct {
	UserID          string
	UploadBytes     int64
	DownloadBytes   int64
	ConnectionCount int64
}

// NewTrafficCounter creates a new traffic counter.
func NewTrafficCounter() *TrafficCounter {
	return &TrafficCounter{
		users: make(map[string]*userTraffic),
		nodeWide: nodeTraffic{
			lastTime: time.Now(),
		},
	}
}

// AddUploadBytes records upload bytes for a user.
func (tc *TrafficCounter) AddUploadBytes(userID string, n int64) {
	tc.ensureUser(userID)
	tc.users[userID].upload.Add(n)
	tc.nodeWide.totalBytes.Add(n)
}

// AddDownloadBytes records download bytes for a user.
func (tc *TrafficCounter) AddDownloadBytes(userID string, n int64) {
	tc.ensureUser(userID)
	tc.users[userID].download.Add(n)
	tc.nodeWide.totalBytes.Add(n)
}

// AddConnection increments the connection count for a user.
func (tc *TrafficCounter) AddConnection(userID string) {
	tc.ensureUser(userID)
	tc.users[userID].conns.Add(1)
}

// SnapshotAndReset atomically captures all per-user traffic counts and resets them.
func (tc *TrafficCounter) SnapshotAndReset() []TrafficSnapshot {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	snapshots := make([]TrafficSnapshot, 0, len(tc.users))
	for userID, ut := range tc.users {
		up := ut.upload.Swap(0)
		down := ut.download.Swap(0)
		conn := ut.conns.Swap(0)

		if up == 0 && down == 0 && conn == 0 {
			continue
		}

		snapshots = append(snapshots, TrafficSnapshot{
			UserID:          userID,
			UploadBytes:     up,
			DownloadBytes:   down,
			ConnectionCount: conn,
		})
	}

	// Clean up users with no activity (they'll be recreated on next use)
	for userID, ut := range tc.users {
		if ut.upload.Load() == 0 && ut.download.Load() == 0 && ut.conns.Load() == 0 {
			delete(tc.users, userID)
		}
	}

	return snapshots
}

// NodeRxMbps calculates the current receive rate in Mbps.
func (tc *TrafficCounter) NodeRxMbps() float64 {
	// Approximate from snapshots
	tc.nodeWide.mu.Lock()
	defer tc.nodeWide.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tc.nodeWide.lastTime).Seconds()
	if elapsed < 0.1 {
		return 0
	}

	currentBytes := tc.nodeWide.totalBytes.Load()
	delta := float64(currentBytes - tc.nodeWide.lastBytes.Load())
	tc.nodeWide.lastBytes.Store(currentBytes)
	tc.nodeWide.lastTime = now

	return (delta * 8) / (elapsed * 1_000_000) // Mbps
}

// TotalBytes returns the total bytes processed by this node.
func (tc *TrafficCounter) TotalBytes() int64 {
	return tc.nodeWide.totalBytes.Load()
}

func (tc *TrafficCounter) ensureUser(userID string) {
	tc.mu.RLock()
	_, ok := tc.users[userID]
	tc.mu.RUnlock()

	if !ok {
		tc.mu.Lock()
		// Double-check after acquiring write lock
		if _, ok := tc.users[userID]; !ok {
			tc.users[userID] = &userTraffic{}
		}
		tc.mu.Unlock()
	}
}

// ────────────────────────────────────────────────────────────
// io wrappers for automatic counting
// ────────────────────────────────────────────────────────────

// CountingReader wraps an io.Reader and counts bytes read.
type CountingReader struct {
	io.Reader
	counter  *TrafficCounter
	userID   string
	isUpload bool // true = upload (client->target), false = download (target->client)
}

func (cr *CountingReader) Read(p []byte) (int, error) {
	n, err := cr.Reader.Read(p)
	if n > 0 {
		if cr.isUpload {
			cr.counter.AddUploadBytes(cr.userID, int64(n))
		} else {
			cr.counter.AddDownloadBytes(cr.userID, int64(n))
		}
	}
	return n, err
}

// NewCountingReader creates a reader that counts bytes.
func (tc *TrafficCounter) NewCountingReader(userID string, r io.Reader, isUpload bool) io.Reader {
	return &CountingReader{
		Reader:   r,
		counter:  tc,
		userID:   userID,
		isUpload: isUpload,
	}
}

// CountingWriter wraps an io.Writer and counts bytes written.
type CountingWriter struct {
	io.Writer
	counter  *TrafficCounter
	userID   string
	isUpload bool
}

func (cw *CountingWriter) Write(p []byte) (int, error) {
	n, err := cw.Writer.Write(p)
	if n > 0 {
		if cw.isUpload {
			cw.counter.AddUploadBytes(cw.userID, int64(n))
		} else {
			cw.counter.AddDownloadBytes(cw.userID, int64(n))
		}
	}
	return n, err
}

// NewCountingWriter creates a writer that counts bytes.
func (tc *TrafficCounter) NewCountingWriter(userID string, w io.Writer, isUpload bool) io.Writer {
	return &CountingWriter{
		Writer:   w,
		counter:  tc,
		userID:   userID,
		isUpload: isUpload,
	}
}
