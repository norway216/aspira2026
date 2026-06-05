package project

import (
	"log/slog"
	"net/http"
	"strconv"

	"portfolio-site/internal/module/audit"
	"portfolio-site/internal/render"
)

// Handler handles project HTTP requests.
type Handler struct {
	svc      *Service
	auditSvc *audit.Service
	render   *render.Renderer
	logger   *slog.Logger
}

// NewHandler creates a new project handler.
func NewHandler(svc *Service, auditSvc *audit.Service, render *render.Renderer, logger *slog.Logger) *Handler {
	return &Handler{
		svc:      svc,
		auditSvc: auditSvc,
		render:   render,
		logger:   logger,
	}
}

// List displays the public project list.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	category := r.URL.Query().Get("category")

	q := Query{
		Page:     page,
		PageSize: 12,
		Category: category,
		Status:   string(StatusPublished),
		OrderBy:  "sort_order ASC, published_at DESC",
	}

	projects, total, err := h.svc.ListPublished(r.Context(), q)
	if err != nil {
		h.logger.Error("failed to list projects", "error", err)
		h.render.ServerError(w)
		return
	}

	h.render.Render(w, "projects", map[string]interface{}{
		"Title":       "Projects - Aspira Studio",
		"Description": "Engineering projects and case studies.",
		"Projects":    projects,
		"Total":       total,
		"Page":        page,
		"Category":    category,
	})
}

// Detail displays a single project.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	project, err := h.svc.GetBySlug(r.Context(), slug)
	if err != nil {
		h.render.NotFound(w)
		return
	}

	h.render.Render(w, "project_detail", map[string]interface{}{
		"Title":       project.Title + " - Aspira Studio",
		"Description": project.Summary,
		"Project":     project,
	})
}
