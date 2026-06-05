package article

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"

	"portfolio-site/internal/middleware"
	"portfolio-site/internal/module/audit"
	"portfolio-site/internal/render"
	"portfolio-site/internal/util"
)

// AdminHandler handles admin article CRUD operations.
type AdminHandler struct {
	svc      *Service
	auditSvc *audit.Service
	render   *render.Renderer
	logger   *slog.Logger
	db       *sql.DB
}

// NewAdminHandler creates a new admin article handler.
func NewAdminHandler(svc *Service, auditSvc *audit.Service, render *render.Renderer, logger *slog.Logger, db *sql.DB) *AdminHandler {
	return &AdminHandler{
		svc:      svc,
		auditSvc: auditSvc,
		render:   render,
		logger:   logger,
		db:       db,
	}
}

// Dashboard renders the admin dashboard.
func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	articleCount, _ := h.svc.GetCount(r.Context(), "")
	publishedCount, _ := h.svc.GetCount(r.Context(), "published")
	draftCount, _ := h.svc.GetCount(r.Context(), "draft")

	var projectCount int64
	h.db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM projects").Scan(&projectCount)

	var totalPV int64
	h.db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM page_views").Scan(&totalPV)

	recentArticles, _ := h.svc.GetRecent(r.Context(), 5)

	h.render.Render(w, "dashboard", map[string]interface{}{
		"Title":          "Dashboard - Aspira Admin",
		"ArticleCount":   articleCount,
		"PublishedCount": publishedCount,
		"DraftCount":     draftCount,
		"ProjectCount":   projectCount,
		"TotalPV":        totalPV,
		"RecentArticles": recentArticles,
	})
}

// ListArticles displays the admin article list.
func (h *AdminHandler) ListArticles(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	status := r.URL.Query().Get("status")

	q := Query{
		Page:     page,
		PageSize: 20,
		Status:   status,
		OrderBy:  "a.created_at DESC",
	}

	articles, total, err := h.svc.ListAll(r.Context(), q)
	if err != nil {
		h.logger.Error("failed to list articles for admin", "error", err)
		h.render.ServerError(w)
		return
	}

	h.render.Render(w, "article_list", map[string]interface{}{
		"Title":    "Articles - Aspira Admin",
		"Articles": articles,
		"Total":    total,
		"Page":     page,
		"Status":   status,
	})
}

// NewArticle displays the article creation form.
func (h *AdminHandler) NewArticle(w http.ResponseWriter, r *http.Request) {
	categories, _ := h.svc.GetAllCategories(r.Context())
	tags, _ := h.svc.GetAllTags(r.Context())

	h.render.Render(w, "article_form", map[string]interface{}{
		"Title":      "New Article - Aspira Admin",
		"Categories": categories,
		"Tags":       tags,
		"IsNew":      true,
	})
}

// CreateArticle creates a new article.
func (h *AdminHandler) CreateArticle(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render.ErrorPage(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	title := r.FormValue("title")
	slug := r.FormValue("slug")
	summary := r.FormValue("summary")
	contentMarkdown := r.FormValue("content_markdown")
	catID, _ := strconv.ParseInt(r.FormValue("category_id"), 10, 64)

	if slug == "" {
		slug = util.Slugify(title)
	}

	wordCount := util.WordCount(contentMarkdown)
	readingTime := util.ReadingTime(wordCount)

	input := CreateInput{
		Title:           title,
		Slug:            slug,
		Summary:         summary,
		CategoryID:      catID,
		ContentMarkdown: contentMarkdown,
		ContentHTML:     contentMarkdown, // Simplified; real app would render Markdown.
		WordCount:       wordCount,
		ReadingTime:     readingTime,
	}

	article, err := h.svc.Create(r.Context(), input)
	if err != nil {
		h.logger.Error("failed to create article", "error", err)
		h.render.ErrorPage(w, http.StatusInternalServerError, "Failed to create article")
		return
	}

	// Record audit.
	h.auditSvc.RecordFromRequest(r.Context(), r, audit.LogInput{
		ActorID:      middleware.GetUserID(r.Context()),
		ActorName:    middleware.GetUserName(r.Context()),
		Action:       audit.ActionCreate,
		ResourceType: audit.ResourceArticle,
		ResourceID:   article.ID,
		Detail:       audit.Logf("Created article: %s", article.Title),
	})

	http.Redirect(w, r, "/admin/articles", http.StatusSeeOther)
}

// EditArticle displays the article edit form.
func (h *AdminHandler) EditArticle(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.render.NotFound(w)
		return
	}

	article, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		h.render.NotFound(w)
		return
	}

	categories, _ := h.svc.GetAllCategories(r.Context())
	tags, _ := h.svc.GetAllTags(r.Context())

	h.render.Render(w, "article_form", map[string]interface{}{
		"Title":      "Edit Article - Aspira Admin",
		"Article":    article,
		"Categories": categories,
		"Tags":       tags,
		"IsNew":      false,
	})
}

// UpdateArticle updates an existing article.
func (h *AdminHandler) UpdateArticle(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	if err := r.ParseForm(); err != nil {
		h.render.ErrorPage(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	title := r.FormValue("title")
	slug := r.FormValue("slug")
	summary := r.FormValue("summary")
	contentMarkdown := r.FormValue("content_markdown")
	catID, _ := strconv.ParseInt(r.FormValue("category_id"), 10, 64)

	wordCount := util.WordCount(contentMarkdown)
	readingTime := util.ReadingTime(wordCount)

	input := UpdateInput{
		Title:           &title,
		Slug:            &slug,
		Summary:         &summary,
		CategoryID:      &catID,
		ContentMarkdown: &contentMarkdown,
		ContentHTML:     &contentMarkdown,
		WordCount:       &wordCount,
		ReadingTime:     &readingTime,
	}

	article, err := h.svc.Update(r.Context(), id, input)
	if err != nil {
		h.logger.Error("failed to update article", "error", err)
		h.render.ErrorPage(w, http.StatusInternalServerError, "Failed to update article")
		return
	}

	h.auditSvc.RecordFromRequest(r.Context(), r, audit.LogInput{
		ActorID:      middleware.GetUserID(r.Context()),
		ActorName:    middleware.GetUserName(r.Context()),
		Action:       audit.ActionUpdate,
		ResourceType: audit.ResourceArticle,
		ResourceID:   article.ID,
		Detail:       audit.Logf("Updated article: %s", article.Title),
	})

	http.Redirect(w, r, "/admin/articles", http.StatusSeeOther)
}

// DeleteArticle deletes an article.
func (h *AdminHandler) DeleteArticle(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	article, _ := h.svc.GetByID(r.Context(), id)
	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.logger.Error("failed to delete article", "error", err)
		h.render.ServerError(w)
		return
	}

	detail := audit.Logf("Deleted article ID=%d", id)
	if article != nil {
		detail = audit.Logf("Deleted article: %s", article.Title)
	}

	h.auditSvc.RecordFromRequest(r.Context(), r, audit.LogInput{
		ActorID:      middleware.GetUserID(r.Context()),
		ActorName:    middleware.GetUserName(r.Context()),
		Action:       audit.ActionDelete,
		ResourceType: audit.ResourceArticle,
		ResourceID:   id,
		Detail:       detail,
	})

	http.Redirect(w, r, "/admin/articles", http.StatusSeeOther)
}

// PublishArticle publishes an article.
func (h *AdminHandler) PublishArticle(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	if err := h.svc.Publish(r.Context(), id); err != nil {
		h.logger.Error("failed to publish article", "error", err)
		h.render.ServerError(w)
		return
	}

	article, _ := h.svc.GetByID(r.Context(), id)
	detail := audit.Logf("Published article ID=%d", id)
	if article != nil {
		detail = audit.Logf("Published article: %s", article.Title)
	}

	h.auditSvc.RecordFromRequest(r.Context(), r, audit.LogInput{
		ActorID:      middleware.GetUserID(r.Context()),
		ActorName:    middleware.GetUserName(r.Context()),
		Action:       audit.ActionPublish,
		ResourceType: audit.ResourceArticle,
		ResourceID:   id,
		Detail:       detail,
	})

	http.Redirect(w, r, "/admin/articles", http.StatusSeeOther)
}
