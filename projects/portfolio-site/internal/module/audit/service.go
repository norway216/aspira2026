package audit

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Service handles audit logging business logic.
type Service struct {
	repo   *Repository
	logger *slog.Logger
}

// NewService creates a new audit service.
func NewService(repo *Repository, logger *slog.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// Record creates an audit log entry for an action.
// This is the primary method for tracking admin operations.
func (s *Service) Record(ctx context.Context, entry LogInput) {
	log := &Log{
		ActorID:      entry.ActorID,
		ActorName:    entry.ActorName,
		Action:       entry.Action,
		ResourceType: entry.ResourceType,
		ResourceID:   entry.ResourceID,
		Detail:       entry.Detail,
		IPAddress:    entry.IPAddress,
		UserAgent:    truncateStr(entry.UserAgent, 512),
		Status:       entry.Status,
	}
	if log.Status == "" {
		log.Status = StatusSuccess
	}

	if err := s.repo.Create(ctx, log); err != nil {
		s.logger.Error("failed to write audit log",
			"action", entry.Action,
			"resource", entry.ResourceType,
			"error", err,
		)
		return
	}

	// Also write to structured logger for real-time monitoring.
	s.logger.Info("audit",
		"action", entry.Action,
		"resource_type", entry.ResourceType,
		"resource_id", entry.ResourceID,
		"actor", entry.ActorName,
		"status", entry.Status,
		"detail", entry.Detail,
	)
}

// RecordFromRequest is a convenience method that extracts IP and User-Agent from an HTTP request.
func (s *Service) RecordFromRequest(ctx context.Context, r *http.Request, entry LogInput) {
	entry.IPAddress = extractIP(r)
	entry.UserAgent = r.UserAgent()
	s.Record(ctx, entry)
}

// Query returns audit logs matching the query.
func (s *Service) Query(ctx context.Context, q Query) ([]Log, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 50
	}
	return s.repo.Find(ctx, q)
}

// GetActionSummary returns action counts for the last N days.
func (s *Service) GetActionSummary(ctx context.Context, days int) (map[string]int64, error) {
	if days <= 0 {
		days = 7
	}
	return s.repo.GetActionSummary(ctx, days)
}

// CleanOldRecords removes audit logs older than the specified days.
func (s *Service) CleanOldRecords(ctx context.Context, olderThanDays int) (int64, error) {
	return s.repo.CleanOldRecords(ctx, olderThanDays)
}

// BackgroundCleanup runs periodic cleanup of old audit logs.
func (s *Service) BackgroundCleanup(ctx context.Context, olderThanDays int, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, err := s.repo.CleanOldRecords(ctx, olderThanDays)
			if err != nil {
				s.logger.Error("audit cleanup error", "error", err)
			} else if count > 0 {
				s.logger.Info("audit cleanup completed", "records_removed", count)
			}
		}
	}
}

// LogInput is the input for creating an audit log entry.
type LogInput struct {
	ActorID      int64
	ActorName    string
	Action       string
	ResourceType string
	ResourceID   int64
	Detail       string
	IPAddress    string
	UserAgent    string
	Status       string
}

// Logf creates a detail string using fmt.Sprintf.
func Logf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

// extractIP extracts the real client IP from the request.
func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
