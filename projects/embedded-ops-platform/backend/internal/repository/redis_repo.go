package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/embedded-ops-platform/backend/pkg/config"
	"github.com/embedded-ops-platform/backend/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// OnlineStatus represents device online status from Redis.
type OnlineStatus struct {
	DeviceID     string
	LastSeenUnix int64
	TTL          time.Duration
}

// RedisRepo handles online status caching and rate limiting.
type RedisRepo struct {
	client *redis.Client
	mu     sync.RWMutex
	// Local fallback cache when Redis is down
	localCache map[string]time.Time
	redisUp    bool
}

// NewRedisRepo creates a new Redis repository.
func NewRedisRepo(ctx context.Context, cfg config.RedisConfig) *RedisRepo {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: 10,
	})

	rr := &RedisRepo{
		client:     client,
		localCache: make(map[string]time.Time),
		redisUp:    true,
	}

	// Check connectivity
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Warn("Redis not available, using local fallback", logger.ErrField(err))
		rr.redisUp = false
	} else {
		logger.Info("Connected to Redis", logger.Int("pool_size", cfg.PoolSize))
	}

	return rr
}

// Close closes the Redis connection.
func (r *RedisRepo) Close() {
	_ = r.client.Close()
}

// MarkOnline sets a device's online status in Redis with TTL.
func (r *RedisRepo) MarkOnline(ctx context.Context, deviceID string, ttl time.Duration) {
	if r.redisUp {
		err := r.client.Set(ctx, onlineKey(deviceID), time.Now().Unix(), ttl).Err()
		if err != nil {
			logger.Warn("Redis MarkOnline failed, falling back to local",
				logger.String("device_id", deviceID),
				logger.ErrField(err))
			r.redisUp = false
		} else {
			return
		}
	}
	// Local fallback
	r.mu.Lock()
	r.localCache[deviceID] = time.Now()
	r.mu.Unlock()
}

// IsOnline checks if a device is online.
func (r *RedisRepo) IsOnline(ctx context.Context, deviceID string) bool {
	if r.redisUp {
		_, err := r.client.Get(ctx, onlineKey(deviceID)).Result()
		if err == nil {
			return true
		}
		if err != redis.Nil {
			logger.Warn("Redis IsOnline failed, falling back to local",
				logger.String("device_id", deviceID),
				logger.ErrField(err))
			r.redisUp = false
		}
	}
	// Local fallback
	r.mu.RLock()
	t, ok := r.localCache[deviceID]
	r.mu.RUnlock()
	if !ok {
		return false
	}
	return time.Since(t) < 30*time.Second
}

// GetAllOnlineDevices returns all device IDs that are currently marked online.
func (r *RedisRepo) GetAllOnlineDevices(ctx context.Context) (map[string]int64, error) {
	if r.redisUp {
		keys, err := r.client.Keys(ctx, "online:device:*").Result()
		if err != nil {
			logger.Warn("Redis GetAllOnlineDevices failed", logger.ErrField(err))
			r.redisUp = false
		} else {
			result := make(map[string]int64, len(keys))
			for _, key := range keys {
				deviceID := key[14:] // strip "online:device:"
				val, err := r.client.Get(ctx, key).Int64()
				if err == nil {
					result[deviceID] = val
				}
			}
			return result, nil
		}
	}
	// Local fallback
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]int64, len(r.localCache))
	for id, t := range r.localCache {
		if time.Since(t) < 30*time.Second {
			result[id] = t.Unix()
		}
	}
	return result, nil
}

// RemoveOfflineDevices cleans up old entries and returns stale device IDs.
func (r *RedisRepo) RemoveOfflineDevices(ctx context.Context, maxAge time.Duration) []string {
	var staleDevices []string
	if r.redisUp {
		keys, err := r.client.Keys(ctx, "online:device:*").Result()
		if err != nil {
			return nil
		}
		for _, key := range keys {
			val, err := r.client.Get(ctx, key).Int64()
			if err != nil {
				continue
			}
			if time.Since(time.Unix(val, 0)) > maxAge {
				deviceID := key[14:]
				staleDevices = append(staleDevices, deviceID)
				r.client.Del(ctx, key)
			}
		}
	}
	r.mu.Lock()
	for id, t := range r.localCache {
		if time.Since(t) > maxAge {
			staleDevices = append(staleDevices, id)
			delete(r.localCache, id)
		}
	}
	r.mu.Unlock()
	return staleDevices
}

// GetOnlineCount returns the number of currently online devices.
func (r *RedisRepo) GetOnlineCount(ctx context.Context) (int, error) {
	if r.redisUp {
		keys, err := r.client.Keys(ctx, "online:device:*").Result()
		if err != nil {
			return 0, err
		}
		return len(keys), nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.localCache), nil
}

// RateLimit checks if an action has exceeded its rate limit.
func (r *RedisRepo) RateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if !r.redisUp {
		// When Redis is down, allow through (better than blocking everything)
		return true, nil
	}
	count, err := r.client.Incr(ctx, rateLimitKey(key)).Result()
	if err != nil {
		return true, nil
	}
	if count == 1 {
		r.client.Expire(ctx, rateLimitKey(key), window)
	}
	if count > int64(limit) {
		return false, fmt.Errorf("rate limit exceeded")
	}
	return true, nil
}

func onlineKey(deviceID string) string {
	return fmt.Sprintf("online:device:%s", deviceID)
}

func rateLimitKey(key string) string {
	return fmt.Sprintf("ratelimit:%s", key)
}