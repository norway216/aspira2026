package article

import "time"

// Status represents the publication status of an article.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusArchived  Status = "archived"
)

// Article represents a blog article / writing.
type Article struct {
	ID              int64      `json:"id"`
	Title           string     `json:"title"`
	Slug            string     `json:"slug"`
	Summary         string     `json:"summary"`
	CategoryID      int64      `json:"category_id"`
	Category        *Category  `json:"category,omitempty"`
	Tags            []Tag      `json:"tags,omitempty"`
	CoverImage      string     `json:"cover_image,omitempty"`
	ContentMarkdown string     `json:"content_markdown"`
	ContentHTML     string     `json:"content_html"`
	TOCHTML         string     `json:"toc_html,omitempty"`
	ReadingTime     int        `json:"reading_time"`
	WordCount       int        `json:"word_count"`
	Status          Status     `json:"status"`
	IsFeatured      bool       `json:"is_featured"`
	ViewCount       int64      `json:"view_count"`
	SEOTitle        string     `json:"seo_title,omitempty"`
	SEODescription  string     `json:"seo_description,omitempty"`
	OGImage         string     `json:"og_image,omitempty"`
	CanonicalURL    string     `json:"canonical_url,omitempty"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Category represents an article category.
type Category struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description,omitempty"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Tag represents a tag that can be applied to articles and projects.
type Tag struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateInput is used when creating an article.
type CreateInput struct {
	Title           string
	Slug            string
	Summary         string
	CategoryID      int64
	TagIDs          []int64
	CoverImage      string
	ContentMarkdown string
	ContentHTML     string
	TOCHTML         string
	ReadingTime     int
	WordCount       int
	IsFeatured      bool
	SEOTitle        string
	SEODescription  string
	OGImage         string
}

// UpdateInput is used when updating an article.
type UpdateInput struct {
	Title           *string
	Slug            *string
	Summary         *string
	CategoryID      *int64
	TagIDs          []int64
	CoverImage      *string
	ContentMarkdown *string
	ContentHTML     *string
	TOCHTML         *string
	ReadingTime     *int
	WordCount       *int
	IsFeatured      *bool
	SEOTitle        *string
	SEODescription  *string
	OGImage         *string
}

// Query is used for filtering and pagination.
type Query struct {
	Page      int
	PageSize  int
	Category  string
	Tag       string
	Keyword   string
	Status    string
	Featured  *bool
	OrderBy   string
}

// DefaultQuery returns sensible defaults.
func DefaultQuery() Query {
	return Query{
		Page:     1,
		PageSize: 10,
		Status:   string(StatusPublished),
		OrderBy:  "published_at DESC",
	}
}
