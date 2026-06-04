package web

import (
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/aspira2026/traffic_lab_proxy/internal/controlcenter/db"
	"github.com/aspira2026/traffic_lab_proxy/internal/controlcenter/scheduler"
)

// Handler serves the web dashboard pages.
type Handler struct {
	DB       db.DB
	Engine   *scheduler.Engine
	tmpl     *template.Template
	staticFS fs.FS
}

// NewHandler creates a new web handler with parsed templates.
func NewHandler(database db.DB, eng *scheduler.Engine) (*Handler, error) {
	tmpl := template.New("").Funcs(template.FuncMap{
		"formatBytes": formatBytes,
		"add":         func(a, b int64) int64 { return a + b },
	})

	tplFS := TemplatesFS()

	// Walk all .html files in the templates directory
	err := fs.WalkDir(tplFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".html" {
			return nil
		}

		content, err := fs.ReadFile(tplFS, path)
		if err != nil {
			return fmt.Errorf("read template %s: %w", path, err)
		}

		name := filepath.Base(path)
		name = name[:len(name)-len(filepath.Ext(name))] // strip .html
		_, err = tmpl.New(name).Parse(string(content))
		if err != nil {
			return fmt.Errorf("parse template %s: %w", name, err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("load templates: %w", err)
	}

	return &Handler{
		DB:       database,
		Engine:   eng,
		tmpl:     tmpl,
		staticFS: StaticFS(),
	}, nil
}

// ServeStatic returns an http.Handler for serving static files.
func (h *Handler) ServeStatic() http.Handler {
	return http.FileServer(http.FS(h.staticFS))
}

// ────────────────────────────────────────────────────────────
// Page Handlers
// ────────────────────────────────────────────────────────────

func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	// If already logged in, redirect to dashboard
	if _, err := r.Cookie("admin_session"); err == nil {
		http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
		return
	}
	h.render(w, "login.html", nil)
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	stats, _ := h.DB.GetDashboardStats(r.Context())
	nodes, _ := h.DB.ListNodes(r.Context())
	events, _ := h.Engine.ListEvents(r.Context(), 20)

	h.render(w, "dashboard.html", map[string]interface{}{
		"ActivePage":      "dashboard",
		"Stats":           stats,
		"Nodes":           nodes,
		"SchedulerEvents": events,
	})
}

func (h *Handler) NodesPage(w http.ResponseWriter, r *http.Request) {
	nodes, _ := h.DB.ListNodes(r.Context())
	h.render(w, "nodes.html", map[string]interface{}{
		"ActivePage": "nodes",
		"Nodes":      nodes,
	})
}

func (h *Handler) UsersPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, "users.html", map[string]interface{}{
		"ActivePage": "users",
	})
}

func (h *Handler) PoliciesPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, "policies.html", map[string]interface{}{
		"ActivePage": "policies",
	})
}

func (h *Handler) TrafficPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, "traffic.html", map[string]interface{}{
		"ActivePage": "traffic",
	})
}

func (h *Handler) SchedulerPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, "scheduler.html", map[string]interface{}{
		"ActivePage": "scheduler",
	})
}

// ────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────

func (h *Handler) render(w http.ResponseWriter, name string, data interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}
	// Strip .html extension for template lookup
	tmplName := name[:len(name)-len(filepath.Ext(name))]
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, tmplName, data); err != nil {
		slog.Error("template render error", "template", name, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func formatBytes(bytes interface{}) string {
	var b int64
	switch v := bytes.(type) {
	case int64:
		b = v
	case int:
		b = int64(v)
	case float64:
		b = int64(v)
	default:
		return "0 B"
	}

	if b == 0 {
		return "0 B"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
