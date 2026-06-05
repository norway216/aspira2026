package project

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Repository handles database operations for projects.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new project repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// FindPublished returns all published projects matching the query.
func (r *Repository) FindPublished(ctx context.Context, q Query) ([]Project, int64, error) {
	return r.find(ctx, q, true)
}

// FindAll returns all projects for admin use.
func (r *Repository) FindAll(ctx context.Context, q Query) ([]Project, int64, error) {
	return r.find(ctx, q, false)
}

func (r *Repository) find(ctx context.Context, q Query, publishedOnly bool) ([]Project, int64, error) {
	where := []string{}
	args := []interface{}{}

	if publishedOnly {
		where = append(where, "status = ?")
		args = append(args, string(StatusPublished))
	} else if q.Status != "" {
		where = append(where, "status = ?")
		args = append(args, q.Status)
	}

	if q.Category != "" {
		where = append(where, "category = ?")
		args = append(args, q.Category)
	}

	if q.Keyword != "" {
		where = append(where, "(title LIKE ? OR summary LIKE ?)")
		kw := "%" + q.Keyword + "%"
		args = append(args, kw, kw)
	}

	if q.Featured != nil {
		where = append(where, "featured = ?")
		args = append(args, *q.Featured)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	// Count total.
	var total int64
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM projects %s", whereClause)
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count projects: %w", err)
	}

	// Fetch page.
	orderBy := q.OrderBy
	if orderBy == "" {
		orderBy = "sort_order ASC, published_at DESC"
	}

	offset := (q.Page - 1) * q.PageSize
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(
		"SELECT id, title, slug, summary, description, category, status, cover_image, tech_stack, github_url, demo_url, content_markdown, content_html, featured, sort_order, view_count, published_at, created_at, updated_at FROM projects %s ORDER BY %s LIMIT ? OFFSET ?",
		whereClause, orderBy,
	)
	args = append(args, q.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query projects: %w", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		var publishedAt sql.NullTime
		var desc, cat, cover, tech, github, demo, cm, ch sql.NullString

		if err := rows.Scan(
			&p.ID, &p.Title, &p.Slug, &p.Summary, &desc, &cat,
			&p.Status, &cover, &tech, &github, &demo,
			&cm, &ch, &p.Featured, &p.SortOrder, &p.ViewCount,
			&publishedAt, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan project: %w", err)
		}

		p.Description = desc.String
		p.Category = cat.String
		p.CoverImage = cover.String
		p.TechStack = tech.String
		p.GitHubURL = github.String
		p.DemoURL = demo.String
		p.ContentMarkdown = cm.String
		p.ContentHTML = ch.String
		if publishedAt.Valid {
			p.PublishedAt = &publishedAt.Time
		}

		projects = append(projects, p)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration: %w", err)
	}

	return projects, total, nil
}

// FindBySlug finds a project by its slug.
func (r *Repository) FindBySlug(ctx context.Context, slug string) (*Project, error) {
	query := `SELECT id, title, slug, summary, description, category, status, cover_image,
		tech_stack, github_url, demo_url, content_markdown, content_html,
		featured, sort_order, view_count, published_at, created_at, updated_at
		FROM projects WHERE slug = ?`

	var p Project
	var publishedAt sql.NullTime
	var desc, cat, cover, tech, github, demo, cm, ch sql.NullString

	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&p.ID, &p.Title, &p.Slug, &p.Summary, &desc, &cat,
		&p.Status, &cover, &tech, &github, &demo,
		&cm, &ch, &p.Featured, &p.SortOrder, &p.ViewCount,
		&publishedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find project by slug: %w", err)
	}

	p.Description = desc.String
	p.Category = cat.String
	p.CoverImage = cover.String
	p.TechStack = tech.String
	p.GitHubURL = github.String
	p.DemoURL = demo.String
	p.ContentMarkdown = cm.String
	p.ContentHTML = ch.String
	if publishedAt.Valid {
		p.PublishedAt = &publishedAt.Time
	}

	return &p, nil
}

// FindByID finds a project by its ID.
func (r *Repository) FindByID(ctx context.Context, id int64) (*Project, error) {
	query := `SELECT id, title, slug, summary, description, category, status, cover_image,
		tech_stack, github_url, demo_url, content_markdown, content_html,
		featured, sort_order, view_count, published_at, created_at, updated_at
		FROM projects WHERE id = ?`

	var p Project
	var publishedAt sql.NullTime
	var desc, cat, cover, tech, github, demo, cm, ch sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.Title, &p.Slug, &p.Summary, &desc, &cat,
		&p.Status, &cover, &tech, &github, &demo,
		&cm, &ch, &p.Featured, &p.SortOrder, &p.ViewCount,
		&publishedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find project by id: %w", err)
	}

	p.Description = desc.String
	p.Category = cat.String
	p.CoverImage = cover.String
	p.TechStack = tech.String
	p.GitHubURL = github.String
	p.DemoURL = demo.String
	p.ContentMarkdown = cm.String
	p.ContentHTML = ch.String
	if publishedAt.Valid {
		p.PublishedAt = &publishedAt.Time
	}

	return &p, nil
}

// Create inserts a new project.
func (r *Repository) Create(ctx context.Context, p *Project) error {
	query := `INSERT INTO projects (title, slug, summary, description, category, status,
		cover_image, tech_stack, github_url, demo_url, content_markdown, content_html,
		featured, sort_order, published_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		p.Title, p.Slug, p.Summary, p.Description, p.Category, p.Status,
		p.CoverImage, p.TechStack, p.GitHubURL, p.DemoURL,
		p.ContentMarkdown, p.ContentHTML,
		p.Featured, p.SortOrder, p.PublishedAt, now, now,
	)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	p.ID = id
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

// Update updates an existing project.
func (r *Repository) Update(ctx context.Context, p *Project) error {
	query := `UPDATE projects SET title=?, slug=?, summary=?, description=?, category=?,
		status=?, cover_image=?, tech_stack=?, github_url=?, demo_url=?,
		content_markdown=?, content_html=?, featured=?, sort_order=?,
		published_at=?, updated_at=?
		WHERE id=?`

	p.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, query,
		p.Title, p.Slug, p.Summary, p.Description, p.Category,
		p.Status, p.CoverImage, p.TechStack, p.GitHubURL, p.DemoURL,
		p.ContentMarkdown, p.ContentHTML, p.Featured, p.SortOrder,
		p.PublishedAt, p.UpdatedAt, p.ID,
	)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	return nil
}

// Delete removes a project by ID.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM projects WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}

// UpdateStatus changes the status of a project.
func (r *Repository) UpdateStatus(ctx context.Context, id int64, status Status, publishedAt *time.Time) error {
	query := `UPDATE projects SET status=?, published_at=?, updated_at=? WHERE id=?`
	_, err := r.db.ExecContext(ctx, query, string(status), publishedAt, time.Now(), id)
	if err != nil {
		return fmt.Errorf("update project status: %w", err)
	}
	return nil
}

// IncrementViewCount increments the view count for a project.
func (r *Repository) IncrementViewCount(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "UPDATE projects SET view_count = view_count + 1 WHERE id = ?", id)
	return err
}

// Count returns the total number of projects.
func (r *Repository) Count(ctx context.Context, status string) (int64, error) {
	var count int64
	query := "SELECT COUNT(*) FROM projects"
	args := []interface{}{}
	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// GetRecent returns the most recent projects.
func (r *Repository) GetRecent(ctx context.Context, limit int) ([]Project, error) {
	query := `SELECT id, title, slug, status, created_at FROM projects ORDER BY created_at DESC LIMIT ?`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Title, &p.Slug, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// ErrNotFound is returned when a project is not found.
var ErrNotFound = fmt.Errorf("project not found")
