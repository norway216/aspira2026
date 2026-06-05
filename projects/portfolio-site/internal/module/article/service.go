package article

import (
	"context"
	"fmt"
	"time"

	"portfolio-site/internal/util"
)

// Service handles business logic for articles.
type Service struct {
	repo *Repository
}

// NewService creates a new article service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ListPublished returns published articles.
func (s *Service) ListPublished(ctx context.Context, q Query) ([]Article, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 50 {
		q.PageSize = 10
	}
	return s.repo.FindPublished(ctx, q)
}

// ListAll returns all articles for admin.
func (s *Service) ListAll(ctx context.Context, q Query) ([]Article, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 50 {
		q.PageSize = 20
	}
	return s.repo.FindAll(ctx, q)
}

// GetPublishedBySlug returns a published article.
func (s *Service) GetPublishedBySlug(ctx context.Context, slug string) (*Article, error) {
	article, err := s.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if article.Status != StatusPublished {
		return nil, ErrNotFound
	}
	// Async view count.
	go func() {
		_ = s.repo.IncrementViewCount(context.Background(), article.ID)
	}()
	return article, nil
}

// GetByID returns any article by ID (admin).
func (s *Service) GetByID(ctx context.Context, id int64) (*Article, error) {
	return s.repo.FindByID(ctx, id)
}

// Create creates a new article draft.
func (s *Service) Create(ctx context.Context, input CreateInput) (*Article, error) {
	if input.Slug == "" {
		input.Slug = util.Slugify(input.Title)
	}

	readingTime := input.ReadingTime
	if readingTime <= 0 && input.WordCount > 0 {
		readingTime = util.ReadingTime(input.WordCount)
	}

	a := &Article{
		Title:           input.Title,
		Slug:            input.Slug,
		Summary:         input.Summary,
		CategoryID:      input.CategoryID,
		Status:          StatusDraft,
		CoverImage:      input.CoverImage,
		ContentMarkdown: input.ContentMarkdown,
		ContentHTML:     input.ContentHTML,
		TOCHTML:         input.TOCHTML,
		ReadingTime:     readingTime,
		WordCount:       input.WordCount,
		IsFeatured:      input.IsFeatured,
		SEOTitle:        input.SEOTitle,
		SEODescription:  input.SEODescription,
		OGImage:         input.OGImage,
	}

	if err := s.repo.Create(ctx, a, input.TagIDs); err != nil {
		return nil, fmt.Errorf("create article: %w", err)
	}
	return a, nil
}

// Update updates an existing article.
func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*Article, error) {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Title != nil {
		a.Title = *input.Title
	}
	if input.Slug != nil {
		a.Slug = *input.Slug
	}
	if input.Summary != nil {
		a.Summary = *input.Summary
	}
	if input.CategoryID != nil {
		a.CategoryID = *input.CategoryID
	}
	if input.CoverImage != nil {
		a.CoverImage = *input.CoverImage
	}
	if input.ContentMarkdown != nil {
		a.ContentMarkdown = *input.ContentMarkdown
	}
	if input.ContentHTML != nil {
		a.ContentHTML = *input.ContentHTML
	}
	if input.TOCHTML != nil {
		a.TOCHTML = *input.TOCHTML
	}
	if input.ReadingTime != nil {
		a.ReadingTime = *input.ReadingTime
	}
	if input.WordCount != nil {
		a.WordCount = *input.WordCount
	}
	if input.IsFeatured != nil {
		a.IsFeatured = *input.IsFeatured
	}
	if input.SEOTitle != nil {
		a.SEOTitle = *input.SEOTitle
	}
	if input.SEODescription != nil {
		a.SEODescription = *input.SEODescription
	}
	if input.OGImage != nil {
		a.OGImage = *input.OGImage
	}

	if err := s.repo.Update(ctx, a, input.TagIDs); err != nil {
		return nil, fmt.Errorf("update article: %w", err)
	}
	return a, nil
}

// Delete removes an article.
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// Publish publishes an article.
func (s *Service) Publish(ctx context.Context, id int64) error {
	now := time.Now()
	return s.repo.UpdateStatus(ctx, id, StatusPublished, &now)
}

// Unpublish sets back to draft.
func (s *Service) Unpublish(ctx context.Context, id int64) error {
	return s.repo.UpdateStatus(ctx, id, StatusDraft, nil)
}

// Archive archives an article.
func (s *Service) Archive(ctx context.Context, id int64) error {
	return s.repo.UpdateStatus(ctx, id, StatusArchived, nil)
}

// GetFeatured returns featured articles.
func (s *Service) GetFeatured(ctx context.Context, limit int) ([]Article, error) {
	featured := true
	q := Query{
		Page:     1,
		PageSize: limit,
		Featured: &featured,
		Status:   string(StatusPublished),
		OrderBy:  "published_at DESC",
	}
	articles, _, err := s.repo.FindPublished(ctx, q)
	return articles, err
}

// GetRecent returns recent articles.
func (s *Service) GetRecent(ctx context.Context, limit int) ([]Article, error) {
	return s.repo.GetRecent(ctx, limit)
}

// GetCount returns article count by status.
func (s *Service) GetCount(ctx context.Context, status string) (int64, error) {
	return s.repo.Count(ctx, status)
}

// GetAllCategories returns all categories.
func (s *Service) GetAllCategories(ctx context.Context) ([]Category, error) {
	return s.repo.FindAllCategories(ctx)
}

// GetAllTags returns all tags.
func (s *Service) GetAllTags(ctx context.Context) ([]Tag, error) {
	return s.repo.FindAllTags(ctx)
}

// GetArchive returns the archive structure.
func (s *Service) GetArchive(ctx context.Context) (map[int]map[int][]Article, error) {
	return s.repo.GetArchive(ctx)
}
