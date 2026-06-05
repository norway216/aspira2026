package stats

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Service handles statistics business logic.
type Service struct {
	repo     *Repository
	salt     string
}

// NewService creates a new stats service.
func NewService(repo *Repository, salt string) *Service {
	if salt == "" {
		salt = "portfolio-stats-salt"
	}
	return &Service{repo: repo, salt: salt}
}

// RecordView records a page view with privacy-preserving IP hashing.
func (s *Service) RecordView(ctx context.Context, r *http.Request, path string) {
	ip := extractIP(r)
	visitorID := s.generateVisitorID(r)

	isUnique, err := s.repo.IsUniqueVisitor(ctx, visitorID, path)
	if err != nil {
		isUnique = true // Assume unique on error.
	}

	pv := &PageView{
		Path:      path,
		Referrer:  truncateString(r.Referer(), 512),
		UserAgent: truncateString(r.UserAgent(), 512),
		IPHash:    s.hashIP(ip),
		VisitorID: visitorID,
		IsUnique:  isUnique,
	}

	_ = s.repo.RecordPageView(ctx, pv)
}

// GetOverview returns overall statistics.
func (s *Service) GetOverview(ctx context.Context) (*Overview, error) {
	return s.repo.GetOverview(ctx)
}

// GetPathStats returns stats for a specific path.
func (s *Service) GetPathStats(ctx context.Context, path string, days int) ([]DailyStat, error) {
	if days <= 0 {
		days = 30
	}
	return s.repo.GetPathStats(ctx, path, days)
}

// CleanOldRecords removes records older than the specified number of days.
func (s *Service) CleanOldRecords(ctx context.Context, olderThanDays int) (int64, error) {
	return s.repo.CleanOldRecords(ctx, olderThanDays)
}

// hashIP creates a privacy-preserving hash of an IP address.
func (s *Service) hashIP(ip string) string {
	h := sha256.New()
	h.Write([]byte(ip + s.salt))
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// generateVisitorID creates a semi-unique visitor identifier.
func (s *Service) generateVisitorID(r *http.Request) string {
	parts := []string{
		extractIP(r),
		r.UserAgent(),
	}
	data := strings.Join(parts, "|")
	h := sha256.New()
	h.Write([]byte(data + s.salt))
	return hex.EncodeToString(h.Sum(nil))[:24]
}

// extractIP extracts the client IP from the request.
func extractIP(r *http.Request) string {
	// Check X-Forwarded-For header.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	// Check X-Real-IP.
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr.
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// truncateString truncates a string to maxLen characters.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// BackgroundCleanup runs a periodic cleanup of old stats records.
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
				// Log error but don't stop the background worker.
				fmt.Printf("stats cleanup error: %v\n", err)
			} else if count > 0 {
				fmt.Printf("stats cleanup: removed %d old records\n", count)
			}
		}
	}
}
