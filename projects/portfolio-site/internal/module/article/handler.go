package article

import (
	"log/slog"
	"net/http"
	"strconv"

	"portfolio-site/internal/render"
)

// Handler handles article HTTP requests.
type Handler struct {
	svc    *Service
	render *render.Renderer
	logger *slog.Logger
}

// NewHandler creates a new article handler.
func NewHandler(svc *Service, render *render.Renderer, logger *slog.Logger) *Handler {
	return &Handler{
		svc:    svc,
		render: render,
		logger: logger,
	}
}

// List displays the public article list.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	category := r.URL.Query().Get("category")
	tag := r.URL.Query().Get("tag")

	q := Query{
		Page:     page,
		PageSize: 10,
		Category: category,
		Tag:      tag,
		Status:   string(StatusPublished),
		OrderBy:  "published_at DESC",
	}

	articles, total, err := h.svc.ListPublished(r.Context(), q)
	if err != nil {
		h.logger.Error("failed to list articles", "error", err)
		h.render.ServerError(w)
		return
	}

	categories, _ := h.svc.GetAllCategories(r.Context())

	h.render.Render(w, "writings", map[string]interface{}{
		"Title":       "Writings - Aspira Studio",
		"Description": "Engineering notes and field records.",
		"Articles":    articles,
		"Total":       total,
		"Page":        page,
		"Category":    category,
		"Tag":         tag,
		"Categories":  categories,
	})
}

// Detail displays a single article.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	article, err := h.svc.GetPublishedBySlug(r.Context(), slug)
	if err != nil {
		h.render.NotFound(w)
		return
	}

	h.render.Render(w, "writing_detail", map[string]interface{}{
		"Title":       article.Title + " - Aspira Studio",
		"Description": article.Summary,
		"Article":     article,
	})
}

// Archive displays the article archive.
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	archive, err := h.svc.GetArchive(r.Context())
	if err != nil {
		h.logger.Error("failed to get archive", "error", err)
		h.render.ServerError(w)
		return
	}

	h.render.Render(w, "archive", map[string]interface{}{
		"Title":       "Archive - Aspira Studio",
		"Description": "Article archive by year and month.",
		"Archive":     archive,
	})
}
