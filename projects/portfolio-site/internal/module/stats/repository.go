package stats

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Repository handles database operations for statistics.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new stats repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// RecordPageView inserts a page view record.
func (r *Repository) RecordPageView(ctx context.Context, pv *PageView) error {
	query := `INSERT INTO page_views (path, referrer, user_agent, ip_hash, visitor_id, is_unique, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		pv.Path, pv.Referrer, pv.UserAgent, pv.IPHash, pv.VisitorID, pv.IsUnique, now,
	)
	if err != nil {
		return fmt.Errorf("record page view: %w", err)
	}
	id, _ := result.LastInsertId()
	pv.ID = id
	pv.CreatedAt = now
	return nil
}

// IsUniqueVisitor checks if a visitor ID has visited a path today.
func (r *Repository) IsUniqueVisitor(ctx context.Context, visitorID, path string) (bool, error) {
	today := time.Now().Format("2006-01-02")
	var count int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM page_views
		WHERE visitor_id = ? AND path = ? AND is_unique = 1 AND date(created_at) = ?`,
		visitorID, path, today,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// GetOverview returns the statistics overview.
func (r *Repository) GetOverview(ctx context.Context) (*Overview, error) {
	overview := &Overview{}

	// Total PV.
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM page_views").Scan(&overview.TotalPV); err != nil {
		return nil, err
	}

	// Total UV.
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(DISTINCT visitor_id) FROM page_views WHERE visitor_id != ''",
	).Scan(&overview.TotalUV); err != nil {
		return nil, err
	}

	// Today's PV.
	today := time.Now().Format("2006-01-02")
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM page_views WHERE date(created_at) = ?", today,
	).Scan(&overview.TodayPV); err != nil {
		return nil, err
	}

	// Today's UV.
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(DISTINCT visitor_id) FROM page_views WHERE visitor_id != '' AND date(created_at) = ?", today,
	).Scan(&overview.TodayUV); err != nil {
		return nil, err
	}

	// Top pages.
	topPages, err := r.GetTopPages(ctx, 10)
	if err != nil {
		return nil, err
	}
	overview.TopPages = topPages

	// Daily stats (last 30 days).
	dailyStats, err := r.GetDailyStats(ctx, 30)
	if err != nil {
		return nil, err
	}
	overview.DailyStats = dailyStats

	// Referrer stats.
	refStats, err := r.GetReferrerStats(ctx, 10)
	if err != nil {
		return nil, err
	}
	overview.ReferrerStats = refStats

	return overview, nil
}

// GetTopPages returns the most viewed pages.
func (r *Repository) GetTopPages(ctx context.Context, limit int) ([]PageStat, error) {
	query := `SELECT path, COUNT(*) as view_count FROM page_views
		GROUP BY path ORDER BY view_count DESC LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []PageStat
	for rows.Next() {
		var s PageStat
		if err := rows.Scan(&s.Path, &s.ViewCount); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetDailyStats returns daily aggregated statistics.
func (r *Repository) GetDailyStats(ctx context.Context, days int) ([]DailyStat, error) {
	query := `SELECT date(created_at) as date,
		COUNT(*) as pv,
		COUNT(DISTINCT visitor_id) as uv
		FROM page_views
		WHERE created_at >= datetime('now', ?)
		GROUP BY date(created_at)
		ORDER BY date DESC`

	rows, err := r.db.QueryContext(ctx, query, fmt.Sprintf("-%d days", days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []DailyStat
	for rows.Next() {
		var s DailyStat
		var d sql.NullString
		if err := rows.Scan(&d, &s.PV, &s.UV); err != nil {
			return nil, err
		}
		s.Date = d.String
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetReferrerStats returns referrer statistics.
func (r *Repository) GetReferrerStats(ctx context.Context, limit int) ([]ReferrerStat, error) {
	query := `SELECT COALESCE(NULLIF(referrer, ''), '(direct)') as ref, COUNT(*) as cnt
		FROM page_views
		GROUP BY COALESCE(NULLIF(referrer, ''), '(direct)')
		ORDER BY cnt DESC LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []ReferrerStat
	for rows.Next() {
		var s ReferrerStat
		if err := rows.Scan(&s.Referrer, &s.Count); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetPathStats returns stats for a specific path over the last N days.
func (r *Repository) GetPathStats(ctx context.Context, path string, days int) ([]DailyStat, error) {
	query := `SELECT date(created_at) as date,
		COUNT(*) as pv,
		COUNT(DISTINCT visitor_id) as uv
		FROM page_views
		WHERE path = ? AND created_at >= datetime('now', ?)
		GROUP BY date(created_at)
		ORDER BY date DESC`

	rows, err := r.db.QueryContext(ctx, query, path, fmt.Sprintf("-%d days", days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []DailyStat
	for rows.Next() {
		var s DailyStat
		var d sql.NullString
		if err := rows.Scan(&d, &s.PV, &s.UV); err != nil {
			return nil, err
		}
		s.Date = d.String
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// CleanOldRecords removes page view records older than the specified number of days.
func (r *Repository) CleanOldRecords(ctx context.Context, olderThanDays int) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		"DELETE FROM page_views WHERE created_at < datetime('now', ?)",
		fmt.Sprintf("-%d days", olderThanDays),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetDashboardData returns data for the admin dashboard.
func (r *Repository) GetDashboardData(ctx context.Context, db *sql.DB) (*DashboardData, error) {
	d := &DashboardData{}

	// Project counts.
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM projects").Scan(&d.ProjectCount)

	// Article counts.
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM articles").Scan(&d.ArticleCount)
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM articles WHERE status = 'published'").Scan(&d.PublishedCount)
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM articles WHERE status = 'draft'").Scan(&d.DraftCount)

	// Page views.
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM page_views").Scan(&d.TotalPV)
	today := time.Now().Format("2006-01-02")
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM page_views WHERE date(created_at) = ?", today).Scan(&d.TodayPV)

	return d, nil
}
