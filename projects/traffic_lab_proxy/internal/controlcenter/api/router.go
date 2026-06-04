package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
	"github.com/aspira2026/traffic_lab_proxy/internal/controlcenter/db"
	"github.com/aspira2026/traffic_lab_proxy/internal/controlcenter/scheduler"
	"github.com/aspira2026/traffic_lab_proxy/internal/controlcenter/web"
)

// NewRouter creates and configures the chi router with all routes and middleware.
func NewRouter(database db.DB, cfg *common.ControlCenterConfig, eng *scheduler.Engine) *chi.Mux {
	r := chi.NewRouter()

	// Base middleware
	r.Use(chimw.RealIP)
	r.Use(LoggingMiddleware)
	r.Use(RecoveryMiddleware)
	r.Use(CORSMiddleware)
	r.Use(MetricsMiddleware)

	// Handlers
	authH := &AuthHandler{AdminUser: cfg.AdminUser, AdminPass: cfg.AdminPass}
	userH := &UserHandler{DB: database}
	nodeH := &NodeHandler{DB: database}
	trafficH := &TrafficHandler{DB: database}
	policyH := &PolicyHandler{DB: database}
	schedH := &SchedulerHandler{Engine: eng}

	// Web dashboard handler
	webHandler, err := web.NewHandler(database, eng)
	if err != nil {
		panic("failed to create web handler: " + err.Error())
	}

	// Node secret lookup for HMAC auth
	nodeSecretLookup := func(nodeID string) (string, error) {
		node, err := database.GetNode(context.Background(), nodeID)
		if err != nil {
			return "", err
		}
		return node.NodeSecret, nil
	}

	adminAuth := AdminAuthMiddleware(cfg.AdminUser, cfg.AdminPass)
	nodeAuth := NodeAuthMiddleware(nodeSecretLookup)

	// Static files (public)
	r.Handle("/static/*", webHandler.ServeStatic())

	// ── Web Dashboard routes ───────────────────────────
	r.Group(func(r chi.Router) {
		r.Use(webAuthMiddleware(cfg.AdminUser, cfg.AdminPass))

		r.Get("/", webHandler.Dashboard)
		r.Get("/admin/dashboard", webHandler.Dashboard)
		r.Get("/admin/nodes", webHandler.NodesPage)
		r.Get("/admin/users", webHandler.UsersPage)
		r.Get("/admin/policies", webHandler.PoliciesPage)
		r.Get("/admin/traffic", webHandler.TrafficPage)
		r.Get("/admin/scheduler", webHandler.SchedulerPage)
	})

	// Login page (no auth required)
	r.Get("/admin/login", webHandler.LoginPage)

	// ── API routes ─────────────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Post("/admin/login", authH.Login)
		r.Post("/admin/logout", authH.Logout)

		// Admin routes
		r.Group(func(r chi.Router) {
			r.Use(adminAuth)

			// Users
			r.Post("/users", userH.Create)
			r.Get("/users", userH.List)
			r.Get("/users/{id}", userH.Get)
			r.Put("/users/{id}", userH.Update)
			r.Delete("/users/{id}", userH.Delete)

			// Nodes (admin management)
			r.Get("/nodes", nodeH.List)
			r.Get("/nodes/{node_id}", nodeH.Get)
			r.Post("/nodes/register", nodeH.Register) // Admin can register nodes
			r.Post("/nodes/{node_id}/disable", nodeH.Disable)
			r.Post("/nodes/{node_id}/enable", nodeH.Enable)

			// Policies
			r.Get("/policies", policyH.List)
			r.Put("/policies/{user_id}", policyH.Upsert)
			r.Delete("/policies/{user_id}", policyH.Delete)

			// Traffic
			r.Get("/traffic/logs", trafficH.ListLogs)
			r.Get("/traffic/users", trafficH.UserAllTraffic)
			r.Get("/traffic/recent", trafficH.RecentTraffic)

			// Scheduler
			r.Get("/schedule", schedH.Schedule)
			r.Get("/scheduler/events", schedH.ListEvents)
			r.Post("/scheduler/strategy", schedH.SetStrategy)
		})

		// Node routes (HMAC auth)
		r.Group(func(r chi.Router) {
			r.Use(nodeAuth)

			r.Post("/nodes/heartbeat", nodeH.Heartbeat)
			r.Post("/traffic/report", trafficH.Report)
			r.Get("/nodes/{node_id}/policies", nodeH.GetPolicies)
		})
	})

	// Prometheus metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	return r
}

// webAuthMiddleware is like admin auth but redirects to login for browser requests.
func webAuthMiddleware(adminUser, adminPass string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check session cookie
			cookie, err := r.Cookie("admin_session")
			if err == nil && cookie.Value != "" {
				decoded := decodeSessionCookie(cookie.Value)
				if decoded == adminUser {
					ctx := context.WithValue(r.Context(), ctxAdminUser, adminUser)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// Check Basic auth
			user, pass, ok := r.BasicAuth()
			if ok && user == adminUser && pass == adminPass {
				ctx := context.WithValue(r.Context(), ctxAdminUser, adminUser)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Redirect to login for browser requests
			http.Redirect(w, r, "/admin/login", http.StatusFound)
		})
	}
}
