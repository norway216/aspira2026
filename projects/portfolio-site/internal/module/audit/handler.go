package audit

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"portfolio-site/internal/render"
)

// Handler handles audit log HTTP requests.
type Handler struct {
	svc    *Service
	render *render.Renderer
	logger *slog.Logger
}

// NewHandler creates a new audit handler.
func NewHandler(svc *Service, render *render.Renderer, logger *slog.Logger) *Handler {
	return &Handler{
		svc:    svc,
		render: render,
		logger: logger,
	}
}

// ListPage renders the audit log viewer page.
func (h *Handler) ListPage(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	q := Query{
		Page:         page,
		PageSize:     50,
		Action:       r.URL.Query().Get("action"),
		ResourceType: r.URL.Query().Get("resource_type"),
		Status:       r.URL.Query().Get("status"),
		DateFrom:     r.URL.Query().Get("date_from"),
		DateTo:       r.URL.Query().Get("date_to"),
		OrderBy:      "created_at DESC",
	}

	logs, total, err := h.svc.Query(r.Context(), q)
	if err != nil {
		h.logger.Error("failed to query audit logs", "error", err)
		h.render.ErrorPage(w, http.StatusInternalServerError, "Failed to load audit logs")
		return
	}

	// Get action summary for filtering.
	actionSummary, _ := h.svc.GetActionSummary(r.Context(), 30)

	h.render.Render(w, "audit_logs", map[string]interface{}{
		"Title":         "Audit Logs - Aspira Admin",
		"Logs":          logs,
		"Total":         total,
		"Page":          page,
		"Query":         q,
		"ActionSummary": actionSummary,
	})
}

// ListAPI returns audit logs as JSON.
func (h *Handler) ListAPI(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	q := Query{
		Page:         page,
		PageSize:     50,
		Action:       r.URL.Query().Get("action"),
		ResourceType: r.URL.Query().Get("resource_type"),
		Status:       r.URL.Query().Get("status"),
		DateFrom:     r.URL.Query().Get("date_from"),
		DateTo:       r.URL.Query().Get("date_to"),
	}

	logs, total, err := h.svc.Query(r.Context(), q)
	if err != nil {
		http.Error(w, `{"error":"failed to query audit logs"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":  logs,
		"total": total,
		"page":  page,
	})
}

// SummaryAPI returns action summary counts.
func (h *Handler) SummaryAPI(w http.ResponseWriter, r *http.Request) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 {
		days = 7
	}

	summary, err := h.svc.GetActionSummary(r.Context(), days)
	if err != nil {
		http.Error(w, `{"error":"failed to get summary"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}
