package stats

import "time"

// PageView represents a single page view record.
type PageView struct {
	ID        int64     `json:"id"`
	Path      string    `json:"path"`
	Referrer  string    `json:"referrer,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	IPHash    string    `json:"ip_hash,omitempty"`
	VisitorID string    `json:"visitor_id,omitempty"`
	IsUnique  bool      `json:"is_unique"`
	CreatedAt time.Time `json:"created_at"`
}

// Overview contains summary statistics.
type Overview struct {
	TotalPV       int64            `json:"total_pv"`
	TotalUV       int64            `json:"total_uv"`
	TodayPV       int64            `json:"today_pv"`
	TodayUV       int64            `json:"today_uv"`
	TopPages      []PageStat       `json:"top_pages"`
	DailyStats    []DailyStat      `json:"daily_stats"`
	ReferrerStats []ReferrerStat   `json:"referrer_stats"`
}

// PageStat holds per-page statistics.
type PageStat struct {
	Path      string `json:"path"`
	ViewCount int64  `json:"view_count"`
}

// DailyStat holds daily aggregated statistics.
type DailyStat struct {
	Date      string `json:"date"`
	PV        int64  `json:"pv"`
	UV        int64  `json:"uv"`
}

// ReferrerStat holds referrer statistics.
type ReferrerStat struct {
	Referrer  string `json:"referrer"`
	Count     int64  `json:"count"`
}

// DashboardData holds data for the admin dashboard.
type DashboardData struct {
	ProjectCount   int64  `json:"project_count"`
	ArticleCount   int64  `json:"article_count"`
	PublishedCount int64  `json:"published_count"`
	DraftCount     int64  `json:"draft_count"`
	TotalPV        int64  `json:"total_pv"`
	TodayPV        int64  `json:"today_pv"`
	RecentArticles []RecentItem `json:"recent_articles"`
	RecentProjects []RecentItem `json:"recent_projects"`
}

// RecentItem holds a simplified recent item.
type RecentItem struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
