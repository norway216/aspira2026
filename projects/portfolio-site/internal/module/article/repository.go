package article

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Repository handles database operations for articles.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new article repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// FindPublished returns published articles matching the query.
func (r *Repository) FindPublished(ctx context.Context, q Query) ([]Article, int64, error) {
	return r.find(ctx, q, true)
}

// FindAll returns all articles for admin use.
func (r *Repository) FindAll(ctx context.Context, q Query) ([]Article, int64, error) {
	return r.find(ctx, q, false)
}

func (r *Repository) find(ctx context.Context, q Query, publishedOnly bool) ([]Article, int64, error) {
	where := []string{}
	args := []interface{}{}

	if publishedOnly {
		where = append(where, "a.status = ?")
		args = append(args, string(StatusPublished))
	} else if q.Status != "" {
		where = append(where, "a.status = ?")
		args = append(args, q.Status)
	}

	if q.Category != "" {
		where = append(where, "c.slug = ?")
		args = append(args, q.Category)
	}

	if q.Keyword != "" {
		where = append(where, "(a.title LIKE ? OR a.summary LIKE ? OR a.content_markdown LIKE ?)")
		kw := "%" + q.Keyword + "%"
		args = append(args, kw, kw, kw)
	}

	if q.Featured != nil {
		where = append(where, "a.is_featured = ?")
		args = append(args, *q.Featured)
	}

	// Handle tag filtering.
	joinClause := "LEFT JOIN article_categories c ON a.category_id = c.id"
	if q.Tag != "" {
		joinClause += " JOIN article_tags at2 ON a.id = at2.article_id JOIN tags t ON at2.tag_id = t.id"
		where = append(where, "t.slug = ?")
		args = append(args, q.Tag)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	countSQL := fmt.Sprintf("SELECT COUNT(DISTINCT a.id) FROM articles a %s %s", joinClause, whereClause)
	var total int64
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count articles: %w", err)
	}

	orderBy := q.OrderBy
	if orderBy == "" {
		orderBy = "a.published_at DESC"
	}

	offset := (q.Page - 1) * q.PageSize
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(
		`SELECT a.id, a.title, a.slug, a.summary, a.category_id, a.status,
		a.cover_image, a.content_markdown, a.content_html, a.toc_html,
		a.reading_time, a.word_count, a.view_count, a.is_featured,
		a.seo_title, a.seo_description, a.og_image, a.canonical_url,
		a.published_at, a.created_at, a.updated_at,
		c.id, c.name, c.slug, c.description, c.sort_order
		FROM articles a %s %s ORDER BY %s LIMIT ? OFFSET ?`,
		joinClause, whereClause, orderBy,
	)
	args = append(args, q.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query articles: %w", err)
	}
	defer rows.Close()

	var articles []Article
	for rows.Next() {
		var a Article
		var cat Category
		var catID, catSortOrder sql.NullInt64
		var catName, catSlug, catDesc sql.NullString
		var cover, tocHTML, seoTitle, seoDesc, ogImg, canon sql.NullString
		var publishedAt sql.NullTime

		if err := rows.Scan(
			&a.ID, &a.Title, &a.Slug, &a.Summary, &catID, &a.Status,
			&cover, &a.ContentMarkdown, &a.ContentHTML, &tocHTML,
			&a.ReadingTime, &a.WordCount, &a.ViewCount, &a.IsFeatured,
			&seoTitle, &seoDesc, &ogImg, &canon,
			&publishedAt, &a.CreatedAt, &a.UpdatedAt,
			&cat.ID, &catName, &catSlug, &catDesc, &catSortOrder,
		); err != nil {
			return nil, 0, fmt.Errorf("scan article: %w", err)
		}

		a.CoverImage = cover.String
		a.TOCHTML = tocHTML.String
		a.SEOTitle = seoTitle.String
		a.SEODescription = seoDesc.String
		a.OGImage = ogImg.String
		a.CanonicalURL = canon.String
		if publishedAt.Valid {
			a.PublishedAt = &publishedAt.Time
		}
		if catID.Valid {
			a.CategoryID = catID.Int64
			cat.Name = catName.String
			cat.Slug = catSlug.String
			cat.Description = catDesc.String
			cat.SortOrder = int(catSortOrder.Int64)
			a.Category = &cat
		}

		// Load tags.
		tags, err := r.GetTagsForArticle(ctx, a.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("load tags for article %d: %w", a.ID, err)
		}
		a.Tags = tags

		articles = append(articles, a)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration: %w", err)
	}

	return articles, total, nil
}

// FindBySlug finds an article by its slug.
func (r *Repository) FindBySlug(ctx context.Context, slug string) (*Article, error) {
	query := `SELECT a.id, a.title, a.slug, a.summary, a.category_id, a.status,
		a.cover_image, a.content_markdown, a.content_html, a.toc_html,
		a.reading_time, a.word_count, a.view_count, a.is_featured,
		a.seo_title, a.seo_description, a.og_image, a.canonical_url,
		a.published_at, a.created_at, a.updated_at,
		c.id, c.name, c.slug, c.description, c.sort_order
		FROM articles a
		LEFT JOIN article_categories c ON a.category_id = c.id
		WHERE a.slug = ?`

	var a Article
	var cat Category
	var catID, catSortOrder sql.NullInt64
	var catName, catSlug, catDesc sql.NullString
	var cover, tocHTML, seoTitle, seoDesc, ogImg, canon sql.NullString
	var publishedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&a.ID, &a.Title, &a.Slug, &a.Summary, &catID, &a.Status,
		&cover, &a.ContentMarkdown, &a.ContentHTML, &tocHTML,
		&a.ReadingTime, &a.WordCount, &a.ViewCount, &a.IsFeatured,
		&seoTitle, &seoDesc, &ogImg, &canon,
		&publishedAt, &a.CreatedAt, &a.UpdatedAt,
		&cat.ID, &catName, &catSlug, &catDesc, &catSortOrder,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find article by slug: %w", err)
	}

	a.CoverImage = cover.String
	a.TOCHTML = tocHTML.String
	a.SEOTitle = seoTitle.String
	a.SEODescription = seoDesc.String
	a.OGImage = ogImg.String
	a.CanonicalURL = canon.String
	if publishedAt.Valid {
		a.PublishedAt = &publishedAt.Time
	}
	if catID.Valid {
		a.CategoryID = catID.Int64
		cat.Name = catName.String
		cat.Slug = catSlug.String
		cat.Description = catDesc.String
		cat.SortOrder = int(catSortOrder.Int64)
		a.Category = &cat
	}

	tags, err := r.GetTagsForArticle(ctx, a.ID)
	if err != nil {
		return nil, fmt.Errorf("load tags: %w", err)
	}
	a.Tags = tags

	return &a, nil
}

// FindByID finds an article by its ID.
func (r *Repository) FindByID(ctx context.Context, id int64) (*Article, error) {
	query := `SELECT a.id, a.title, a.slug, a.summary, a.category_id, a.status,
		a.cover_image, a.content_markdown, a.content_html, a.toc_html,
		a.reading_time, a.word_count, a.view_count, a.is_featured,
		a.seo_title, a.seo_description, a.og_image, a.canonical_url,
		a.published_at, a.created_at, a.updated_at,
		c.id, c.name, c.slug, c.description, c.sort_order
		FROM articles a
		LEFT JOIN article_categories c ON a.category_id = c.id
		WHERE a.id = ?`

	var a Article
	var cat Category
	var catID, catSortOrder sql.NullInt64
	var catName, catSlug, catDesc sql.NullString
	var cover, tocHTML, seoTitle, seoDesc, ogImg, canon sql.NullString
	var publishedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&a.ID, &a.Title, &a.Slug, &a.Summary, &catID, &a.Status,
		&cover, &a.ContentMarkdown, &a.ContentHTML, &tocHTML,
		&a.ReadingTime, &a.WordCount, &a.ViewCount, &a.IsFeatured,
		&seoTitle, &seoDesc, &ogImg, &canon,
		&publishedAt, &a.CreatedAt, &a.UpdatedAt,
		&cat.ID, &catName, &catSlug, &catDesc, &catSortOrder,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find article by id: %w", err)
	}

	a.CoverImage = cover.String
	a.TOCHTML = tocHTML.String
	a.SEOTitle = seoTitle.String
	a.SEODescription = seoDesc.String
	a.OGImage = ogImg.String
	a.CanonicalURL = canon.String
	if publishedAt.Valid {
		a.PublishedAt = &publishedAt.Time
	}
	if catID.Valid {
		a.CategoryID = catID.Int64
		cat.Name = catName.String
		cat.Slug = catSlug.String
		cat.Description = catDesc.String
		cat.SortOrder = int(catSortOrder.Int64)
		a.Category = &cat
	}

	tags, err := r.GetTagsForArticle(ctx, a.ID)
	if err != nil {
		return nil, fmt.Errorf("load tags: %w", err)
	}
	a.Tags = tags

	return &a, nil
}

// Create inserts a new article.
func (r *Repository) Create(ctx context.Context, a *Article, tagIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()
	query := `INSERT INTO articles (title, slug, summary, category_id, status,
		cover_image, content_markdown, content_html, toc_html,
		reading_time, word_count, is_featured,
		seo_title, seo_description, og_image, canonical_url,
		published_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var catID interface{}
	if a.CategoryID > 0 {
		catID = a.CategoryID
	}

	result, err := tx.ExecContext(ctx, query,
		a.Title, a.Slug, a.Summary, catID, a.Status,
		a.CoverImage, a.ContentMarkdown, a.ContentHTML, a.TOCHTML,
		a.ReadingTime, a.WordCount, a.IsFeatured,
		a.SEOTitle, a.SEODescription, a.OGImage, a.CanonicalURL,
		a.PublishedAt, now, now,
	)
	if err != nil {
		return fmt.Errorf("create article: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	a.ID = id
	a.CreatedAt = now
	a.UpdatedAt = now

	// Insert tags.
	if len(tagIDs) > 0 {
		for _, tagID := range tagIDs {
			if _, err := tx.ExecContext(ctx,
				"INSERT OR IGNORE INTO article_tags (article_id, tag_id) VALUES (?, ?)",
				a.ID, tagID); err != nil {
				return fmt.Errorf("insert article tag: %w", err)
			}
		}
	}

	return tx.Commit()
}

// Update updates an existing article.
func (r *Repository) Update(ctx context.Context, a *Article, tagIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	a.UpdatedAt = time.Now()
	query := `UPDATE articles SET title=?, slug=?, summary=?, category_id=?, status=?,
		cover_image=?, content_markdown=?, content_html=?, toc_html=?,
		reading_time=?, word_count=?, is_featured=?,
		seo_title=?, seo_description=?, og_image=?, canonical_url=?,
		published_at=?, updated_at=? WHERE id=?`

	var catID interface{}
	if a.CategoryID > 0 {
		catID = a.CategoryID
	}

	_, err = tx.ExecContext(ctx, query,
		a.Title, a.Slug, a.Summary, catID, a.Status,
		a.CoverImage, a.ContentMarkdown, a.ContentHTML, a.TOCHTML,
		a.ReadingTime, a.WordCount, a.IsFeatured,
		a.SEOTitle, a.SEODescription, a.OGImage, a.CanonicalURL,
		a.PublishedAt, a.UpdatedAt, a.ID,
	)
	if err != nil {
		return fmt.Errorf("update article: %w", err)
	}

	// Update tags if provided.
	if tagIDs != nil {
		if _, err := tx.ExecContext(ctx, "DELETE FROM article_tags WHERE article_id = ?", a.ID); err != nil {
			return fmt.Errorf("delete article tags: %w", err)
		}
		for _, tagID := range tagIDs {
			if _, err := tx.ExecContext(ctx,
				"INSERT OR IGNORE INTO article_tags (article_id, tag_id) VALUES (?, ?)",
				a.ID, tagID); err != nil {
				return fmt.Errorf("insert article tag: %w", err)
			}
		}
	}

	return tx.Commit()
}

// Delete removes an article by ID.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM articles WHERE id = ?", id)
	return err
}

// UpdateStatus changes the status of an article.
func (r *Repository) UpdateStatus(ctx context.Context, id int64, status Status, publishedAt *time.Time) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE articles SET status=?, published_at=?, updated_at=? WHERE id=?",
		string(status), publishedAt, time.Now(), id)
	return err
}

// IncrementViewCount increments the view count.
func (r *Repository) IncrementViewCount(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "UPDATE articles SET view_count = view_count + 1 WHERE id = ?", id)
	return err
}

// Count returns article count.
func (r *Repository) Count(ctx context.Context, status string) (int64, error) {
	var count int64
	query := "SELECT COUNT(*) FROM articles"
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

// GetRecent returns the most recent articles.
func (r *Repository) GetRecent(ctx context.Context, limit int) ([]Article, error) {
	query := `SELECT id, title, slug, status, created_at FROM articles ORDER BY created_at DESC LIMIT ?`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []Article
	for rows.Next() {
		var a Article
		if err := rows.Scan(&a.ID, &a.Title, &a.Slug, &a.Status, &a.CreatedAt); err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}
	return articles, rows.Err()
}

// GetTagsForArticle returns tags for a given article.
func (r *Repository) GetTagsForArticle(ctx context.Context, articleID int64) ([]Tag, error) {
	query := `SELECT t.id, t.name, t.slug, t.created_at
		FROM tags t JOIN article_tags at2 ON t.id = at2.tag_id
		WHERE at2.article_id = ? ORDER BY t.name`

	rows, err := r.db.QueryContext(ctx, query, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// FindAllCategories returns all article categories.
func (r *Repository) FindAllCategories(ctx context.Context) ([]Category, error) {
	query := "SELECT id, name, slug, description, sort_order, created_at, updated_at FROM article_categories ORDER BY sort_order ASC, name ASC"
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []Category
	for rows.Next() {
		var c Category
		var desc sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &desc, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.Description = desc.String
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

// FindAllTags returns all tags.
func (r *Repository) FindAllTags(ctx context.Context) ([]Tag, error) {
	query := "SELECT id, name, slug, created_at FROM tags ORDER BY name ASC"
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// CreateCategory creates a new article category.
func (r *Repository) CreateCategory(ctx context.Context, c *Category) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx,
		"INSERT INTO article_categories (name, slug, description, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		c.Name, c.Slug, c.Description, c.SortOrder, now, now)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	c.ID = id
	c.CreatedAt = now
	c.UpdatedAt = now
	return nil
}

// CreateTag creates a new tag.
func (r *Repository) CreateTag(ctx context.Context, t *Tag) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx,
		"INSERT INTO tags (name, slug, created_at) VALUES (?, ?, ?)",
		t.Name, t.Slug, now)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	t.ID = id
	t.CreatedAt = now
	return nil
}

// GetArchive returns articles grouped by year and month.
func (r *Repository) GetArchive(ctx context.Context) (map[int]map[int][]Article, error) {
	query := `SELECT id, title, slug, published_at FROM articles
		WHERE status = 'published' AND published_at IS NOT NULL
		ORDER BY published_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	archive := make(map[int]map[int][]Article)
	for rows.Next() {
		var a Article
		var pubAt time.Time
		if err := rows.Scan(&a.ID, &a.Title, &a.Slug, &pubAt); err != nil {
			return nil, err
		}
		a.PublishedAt = &pubAt
		year := pubAt.Year()
		month := int(pubAt.Month())
		if archive[year] == nil {
			archive[year] = make(map[int][]Article)
		}
		archive[year][month] = append(archive[year][month], a)
	}
	return archive, rows.Err()
}

// ErrNotFound is returned when an article is not found.
var ErrNotFound = fmt.Errorf("article not found")
