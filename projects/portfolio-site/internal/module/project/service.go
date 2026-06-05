package project

import (
	"context"
	"fmt"
	"time"

	"portfolio-site/internal/util"
)

// Service handles business logic for projects.
type Service struct {
	repo *Repository
}

// NewService creates a new project service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ListPublished returns published projects with pagination.
func (s *Service) ListPublished(ctx context.Context, q Query) ([]Project, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 12
	}
	return s.repo.FindPublished(ctx, q)
}

// ListAll returns all projects for admin use.
func (s *Service) ListAll(ctx context.Context, q Query) ([]Project, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	return s.repo.FindAll(ctx, q)
}

// GetBySlug returns a published project by slug.
func (s *Service) GetBySlug(ctx context.Context, slug string) (*Project, error) {
	project, err := s.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if project.Status != StatusPublished {
		return nil, ErrNotFound
	}
	// Increment view count asynchronously.
	go func() {
		_ = s.repo.IncrementViewCount(context.Background(), project.ID)
	}()
	return project, nil
}

// GetByID returns a project by ID.
func (s *Service) GetByID(ctx context.Context, id int64) (*Project, error) {
	return s.repo.FindByID(ctx, id)
}

// Create creates a new project.
func (s *Service) Create(ctx context.Context, input CreateInput) (*Project, error) {
	if input.Slug == "" {
		input.Slug = util.Slugify(input.Title)
	}

	p := &Project{
		Title:           input.Title,
		Slug:            input.Slug,
		Summary:         input.Summary,
		Description:     input.Description,
		Category:        input.Category,
		Status:          StatusDraft,
		CoverImage:      input.CoverImage,
		TechStack:       input.TechStack,
		GitHubURL:       input.GitHubURL,
		DemoURL:         input.DemoURL,
		ContentMarkdown: input.ContentMarkdown,
		ContentHTML:     input.ContentHTML,
		Featured:        input.Featured,
		SortOrder:       input.SortOrder,
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return p, nil
}

// Update updates an existing project.
func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*Project, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Title != nil {
		p.Title = *input.Title
	}
	if input.Slug != nil {
		p.Slug = *input.Slug
	}
	if input.Summary != nil {
		p.Summary = *input.Summary
	}
	if input.Description != nil {
		p.Description = *input.Description
	}
	if input.Category != nil {
		p.Category = *input.Category
	}
	if input.CoverImage != nil {
		p.CoverImage = *input.CoverImage
	}
	if input.TechStack != nil {
		p.TechStack = *input.TechStack
	}
	if input.GitHubURL != nil {
		p.GitHubURL = *input.GitHubURL
	}
	if input.DemoURL != nil {
		p.DemoURL = *input.DemoURL
	}
	if input.ContentMarkdown != nil {
		p.ContentMarkdown = *input.ContentMarkdown
	}
	if input.ContentHTML != nil {
		p.ContentHTML = *input.ContentHTML
	}
	if input.Featured != nil {
		p.Featured = *input.Featured
	}
	if input.SortOrder != nil {
		p.SortOrder = *input.SortOrder
	}

	if err := s.repo.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}
	return p, nil
}

// Delete removes a project.
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// Publish publishes a project.
func (s *Service) Publish(ctx context.Context, id int64) error {
	now := time.Now()
	return s.repo.UpdateStatus(ctx, id, StatusPublished, &now)
}

// Unpublish sets a project back to draft.
func (s *Service) Unpublish(ctx context.Context, id int64) error {
	return s.repo.UpdateStatus(ctx, id, StatusDraft, nil)
}

// Archive archives a project.
func (s *Service) Archive(ctx context.Context, id int64) error {
	return s.repo.UpdateStatus(ctx, id, StatusArchived, nil)
}

// GetFeatured returns featured projects.
func (s *Service) GetFeatured(ctx context.Context, limit int) ([]Project, error) {
	featured := true
	q := Query{
		Page:     1,
		PageSize: limit,
		Featured: &featured,
		Status:   string(StatusPublished),
		OrderBy:  "sort_order ASC, published_at DESC",
	}
	projects, _, err := s.repo.FindPublished(ctx, q)
	return projects, err
}

// GetRecent returns recent projects.
func (s *Service) GetRecent(ctx context.Context, limit int) ([]Project, error) {
	return s.repo.GetRecent(ctx, limit)
}

// GetCount returns project count by status.
func (s *Service) GetCount(ctx context.Context, status string) (int64, error) {
	return s.repo.Count(ctx, status)
}
