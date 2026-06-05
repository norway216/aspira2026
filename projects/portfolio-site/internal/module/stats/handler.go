package stats

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"portfolio-site/internal/render"
)

// Handler handles statistics HTTP requests.
type Handler struct {
	svc    *Service
	render *render.Renderer
	logger *slog.Logger
}

// NewHandler creates a new stats handler.
func NewHandler(svc *Service, render *render.Renderer, logger *slog.Logger) *Handler {
	return &Handler{
		svc:    svc,
		render: render,
		logger: logger,
	}
}

// OverviewPage renders the stats overview page.
func (h *Handler) OverviewPage(w http.ResponseWriter, r *http.Request) {
	overview, err := h.svc.GetOverview(r.Context())
	if err != nil {
		h.logger.Error("failed to get stats overview", "error", err)
		h.render.ErrorPage(w, http.StatusInternalServerError, "Failed to load statistics")
		return
	}

	h.render.Render(w, "stats_overview", map[string]interface{}{
		"Title":    "Statistics - Aspira Admin",
		"Overview": overview,
	})
}

// OverviewAPI returns stats as JSON (for API/charts).
func (h *Handler) OverviewAPI(w http.ResponseWriter, r *http.Request) {
	overview, err := h.svc.GetOverview(r.Context())
	if err != nil {
		h.logger.Error("failed to get stats overview", "error", err)
		http.Error(w, `{"error":"failed to load stats"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(overview); err != nil {
		h.logger.Error("failed to encode stats response", "error", err)
	}
}

// PathStatsAPI returns stats for a specific path.
func (h *Handler) PathStatsAPI(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, `{"error":"path parameter required"}`, http.StatusBadRequest)
		return
	}

	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 {
		days = 30
	}

	stats, err := h.svc.GetPathStats(r.Context(), path, days)
	if err != nil {
		h.logger.Error("failed to get path stats", "error", err)
		http.Error(w, `{"error":"failed to load path stats"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// DailyStatsAPI returns daily stats.
func (h *Handler) DailyStatsAPI(w http.ResponseWriter, r *http.Request) {
	overview, err := h.svc.GetOverview(r.Context())
	if err != nil {
		h.logger.Error("failed to get daily stats", "error", err)
		http.Error(w, `{"error":"failed to load daily stats"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(overview.DailyStats)
}
