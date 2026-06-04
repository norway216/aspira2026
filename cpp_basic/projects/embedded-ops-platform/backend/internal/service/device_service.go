package service

import (
	"context"
	"sync"
	"time"

	"github.com/embedded-ops-platform/backend/internal/model"
	"github.com/embedded-ops-platform/backend/internal/repository"
	"github.com/embedded-ops-platform/backend/internal/ws"
	"github.com/embedded-ops-platform/backend/pkg/logger"
)

// DeviceService handles business logic for device management.
type DeviceService struct {
	db     *repository.PostgresRepo
	redis  *repository.RedisRepo
	hub    *ws.Hub
	mu     sync.Mutex
	buffer *metricsBuffer
}

type metricsBuffer struct {
	mu      sync.Mutex
	metrics []*model.DeviceMetric
}

// NewDeviceService creates a new device service.
func NewDeviceService(db *repository.PostgresRepo, redis *repository.RedisRepo, hub *ws.Hub) *DeviceService {
	svc := &DeviceService{
		db:     db,
		redis:  redis,
		hub:    hub,
		buffer: &metricsBuffer{metrics: make([]*model.DeviceMetric, 0, 1000)},
	}

	// Start background flush for metrics
	go svc.flushMetricsLoop()
	// Start background flush for heartbeat
	go svc.flushHeartbeatsLoop()

	return svc
}

// HandleHeartbeat processes an agent heartbeat.
func (s *DeviceService) HandleHeartbeat(ctx context.Context, req *model.HeartbeatRequest) error {
	ttl := 20 * time.Second
	s.redis.MarkOnline(ctx, req.DeviceID, ttl)

	// Cache for async DB update
	s.mu.Lock()
	s.buffer.metrics = append(s.buffer.metrics, &model.DeviceMetric{
		DeviceID: req.DeviceID,
	})
	s.mu.Unlock()

	// Broadcast online status
	s.hub.Broadcast(ws.MsgTypeDeviceOnline, map[string]interface{}{
		"device_id": req.DeviceID,
		"timestamp": time.Now().Unix(),
	})

	return nil
}

// HandleMetrics processes agent metric reports with batching.
func (s *DeviceService) HandleMetrics(ctx context.Context, req *model.MetricsRequest) error {
	metric := &model.DeviceMetric{
		DeviceID:    req.DeviceID,
		CPUUsage:    req.CPUUsage,
		MemoryUsage: req.MemoryUsage,
		DiskUsage:   req.DiskUsage,
		LoadAvg1m:   req.LoadAvg1m,
		Temperature: req.Temperature,
		GPULoad:     req.GPULoad,
		NetworkRX:   req.NetworkRX,
		NetworkTX:   req.NetworkTX,
	}
	s.buffer.mu.Lock()
	s.buffer.metrics = append(s.buffer.metrics, metric)
	s.buffer.mu.Unlock()

	// Broadcast metrics update to frontend
	s.hub.Broadcast(ws.MsgTypeMetricsUpdate, map[string]interface{}{
		"device_id":    req.DeviceID,
		"cpu_usage":    req.CPUUsage,
		"memory_usage": req.MemoryUsage,
		"disk_usage":   req.DiskUsage,
		"temperature":  req.Temperature,
		"timestamp":    time.Now().Unix(),
	})

	return nil
}

// flushMetricsLoop periodically flushes buffered metrics to PostgreSQL.
func (s *DeviceService) flushMetricsLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.buffer.mu.Lock()
		metrics := s.buffer.metrics
		s.buffer.metrics = make([]*model.DeviceMetric, 0, 1000)
		s.buffer.mu.Unlock()

		if len(metrics) == 0 {
			continue
		}

		// Filter to only metrics with actual data (not just heartbeats)
		realMetrics := make([]*model.DeviceMetric, 0, len(metrics))
		for _, m := range metrics {
			if m.CPUUsage > 0 || m.MemoryUsage > 0 || m.DiskUsage > 0 || m.Temperature > 0 {
				realMetrics = append(realMetrics, m)
			}
		}

		if len(realMetrics) == 0 {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := s.db.BatchInsertMetrics(ctx, realMetrics); err != nil {
			logger.Warn("Failed to flush metrics", logger.ErrField(err))
			// Re-buffer on failure
			s.buffer.mu.Lock()
			s.buffer.metrics = append(s.buffer.metrics, realMetrics...)
			s.buffer.mu.Unlock()
		}
		cancel()
	}
}

// flushHeartbeatsLoop periodically syncs Redis online status to PostgreSQL.
func (s *DeviceService) flushHeartbeatsLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		onlineDevices, err := s.redis.GetAllOnlineDevices(ctx)
		cancel()
		if err != nil {
			logger.Warn("Failed to get online devices for flush", logger.ErrField(err))
			continue
		}

		updates := make(map[string]time.Time, len(onlineDevices))
		for deviceID, ts := range onlineDevices {
			updates[deviceID] = time.Unix(ts, 0)
		}

		if len(updates) > 0 {
			ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
			if err := s.db.BatchUpdateLastSeen(ctx2, updates); err != nil {
				logger.Warn("Failed to batch update last_seen", logger.ErrField(err))
			}
			cancel2()
		}
	}
}

// CheckOfflineDevices marks devices as offline if they haven't sent heartbeats.
func (s *DeviceService) CheckOfflineDevices(ctx context.Context) {
	staleDevices := s.redis.RemoveOfflineDevices(ctx, 25*time.Second)
	for _, deviceID := range staleDevices {
		if err := s.db.UpdateDeviceStatus(ctx, deviceID, "offline", time.Now()); err != nil {
			logger.Warn("Failed to mark device offline",
				logger.String("device_id", deviceID),
				logger.ErrField(err))
		}
		s.hub.Broadcast(ws.MsgTypeDeviceOffline, map[string]interface{}{
			"device_id": deviceID,
			"timestamp": time.Now().Unix(),
		})
	}
}

// ListDevices returns paginated device list.
func (s *DeviceService) ListDevices(ctx context.Context, page, pageSize int) ([]*model.Device, int, error) {
	offset := (page - 1) * pageSize
	devices, total, err := s.db.ListDevices(ctx, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}
	// Enrich with online status
	for _, d := range devices {
		if s.redis.IsOnline(ctx, d.DeviceID) {
			d.Status = "online"
		} else {
			d.Status = "offline"
		}
	}
	return devices, total, nil
}

// GetDevice returns device details including latest metrics.
func (s *DeviceService) GetDevice(ctx context.Context, deviceID string) (*model.Device, *model.DeviceMetric, error) {
	device, err := s.db.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, nil, err
	}
	if s.redis.IsOnline(ctx, deviceID) {
		device.Status = "online"
	} else {
		device.Status = "offline"
	}
	metric, err := s.db.GetLatestMetrics(ctx, deviceID)
	if err != nil {
		metric = nil // Metrics are optional
	}
	return device, metric, nil
}

// GetOnlineCount returns the number of online devices.
func (s *DeviceService) GetOnlineCount(ctx context.Context) (int, error) {
	return s.redis.GetOnlineCount(ctx)
}

// GetTotalCount returns the total number of registered devices.
func (s *DeviceService) GetTotalCount(ctx context.Context) (int, error) {
	_, total, err := s.db.ListDevices(ctx, 0, 1)
	return total, err
}