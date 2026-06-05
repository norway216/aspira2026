package project

import "time"

// Status represents the publication status of a project.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusArchived  Status = "archived"
)

// Project represents a portfolio project.
type Project struct {
	ID              int64      `json:"id"`
	Title           string     `json:"title"`
	Slug            string     `json:"slug"`
	Summary         string     `json:"summary"`
	Description     string     `json:"description,omitempty"`
	Category        string     `json:"category,omitempty"`
	Status          Status     `json:"status"`
	CoverImage      string     `json:"cover_image,omitempty"`
	TechStack       string     `json:"tech_stack,omitempty"`
	GitHubURL       string     `json:"github_url,omitempty"`
	DemoURL         string     `json:"demo_url,omitempty"`
	ContentMarkdown string     `json:"content_markdown,omitempty"`
	ContentHTML     string     `json:"content_html,omitempty"`
	Featured        bool       `json:"featured"`
	SortOrder       int        `json:"sort_order"`
	ViewCount       int64      `json:"view_count"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// CreateInput is used when creating a project.
type CreateInput struct {
	Title           string
	Slug            string
	Summary         string
	Description     string
	Category        string
	CoverImage      string
	TechStack       string
	GitHubURL       string
	DemoURL         string
	ContentMarkdown string
	ContentHTML     string
	Featured        bool
	SortOrder       int
}

// UpdateInput is used when updating a project.
type UpdateInput struct {
	Title           *string
	Slug            *string
	Summary         *string
	Description     *string
	Category        *string
	CoverImage      *string
	TechStack       *string
	GitHubURL       *string
	DemoURL         *string
	ContentMarkdown *string
	ContentHTML     *string
	Featured        *bool
	SortOrder       *int
}

// Query is used for filtering and pagination of projects.
type Query struct {
	Page     int
	PageSize int
	Status   string
	Category string
	Featured *bool
	Keyword  string
	OrderBy  string
}

// DefaultQuery returns a default query with sensible values.
func DefaultQuery() Query {
	return Query{
		Page:     1,
		PageSize: 12,
		Status:   string(StatusPublished),
		OrderBy:  "sort_order ASC, published_at DESC",
	}
}
