package app

import (
	"net/http"

	"portfolio-site/internal/middleware"
)

// Router returns the fully configured HTTP handler with all routes.
func (a *Application) Router() http.Handler {
	mux := http.NewServeMux()

	// --- Static files (must use method+path pattern to avoid conflicts) ---
	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))
	mux.Handle("HEAD /static/", http.StripPrefix("/static/", fs))

	// --- Public routes ---
	// Home page. "{$}" means exact match, preventing conflict with /static/
	mux.HandleFunc("GET /{$}", a.PageHandler.Home)

	// Projects.
	mux.HandleFunc("GET /projects", a.ProjectHandler.List)
	mux.HandleFunc("GET /projects/{slug}", a.ProjectHandler.Detail)

	// Writings (blog).
	mux.HandleFunc("GET /writings", a.ArticleHandler.List)
	mux.HandleFunc("GET /writings/{slug}", a.ArticleHandler.Detail)
	mux.HandleFunc("GET /writings/archive", a.ArticleHandler.Archive)

	// Static pages.
	mux.HandleFunc("GET /about", a.PageHandler.About)
	mux.HandleFunc("GET /resume", a.PageHandler.Resume)
	mux.HandleFunc("GET /contact", a.PageHandler.Contact)
	mux.HandleFunc("POST /contact", a.PageHandler.SubmitContact)

	// SEO.
	mux.HandleFunc("GET /rss.xml", a.PageHandler.RSS)
	mux.HandleFunc("GET /sitemap.xml", a.PageHandler.Sitemap)
	mux.HandleFunc("GET /robots.txt", a.PageHandler.RobotsTxt)

	// --- Admin sub-router ---
	adminMux := http.NewServeMux()

	// Public admin routes (no auth required).
	adminMux.HandleFunc("GET /admin/login", a.AuthHandler.LoginPage)
	adminMux.HandleFunc("POST /admin/login", a.AuthHandler.Login)

	// Protected admin routes.
	protectedMux := http.NewServeMux()

	// Dashboard.
	protectedMux.HandleFunc("GET /admin", a.ArticleAdminHandler.Dashboard)
	protectedMux.HandleFunc("GET /admin/", a.ArticleAdminHandler.Dashboard)

	// Articles CRUD.
	protectedMux.HandleFunc("GET /admin/articles", a.ArticleAdminHandler.ListArticles)
	protectedMux.HandleFunc("GET /admin/articles/new", a.ArticleAdminHandler.NewArticle)
	protectedMux.HandleFunc("POST /admin/articles", a.ArticleAdminHandler.CreateArticle)
	protectedMux.HandleFunc("GET /admin/articles/{id}/edit", a.ArticleAdminHandler.EditArticle)
	protectedMux.HandleFunc("POST /admin/articles/{id}", a.ArticleAdminHandler.UpdateArticle)
	protectedMux.HandleFunc("POST /admin/articles/{id}/delete", a.ArticleAdminHandler.DeleteArticle)
	protectedMux.HandleFunc("POST /admin/articles/{id}/publish", a.ArticleAdminHandler.PublishArticle)

	// Stats.
	protectedMux.HandleFunc("GET /admin/stats", a.StatsHandler.OverviewPage)

	// API for stats.
	protectedMux.HandleFunc("GET /api/stats/overview", a.StatsHandler.OverviewAPI)
	protectedMux.HandleFunc("GET /api/stats/daily", a.StatsHandler.DailyStatsAPI)
	protectedMux.HandleFunc("GET /api/stats/path", a.StatsHandler.PathStatsAPI)

	// Audit logs.
	protectedMux.HandleFunc("GET /admin/audit", a.AuditHandler.ListPage)
	protectedMux.HandleFunc("GET /api/audit", a.AuditHandler.ListAPI)
	protectedMux.HandleFunc("GET /api/audit/summary", a.AuditHandler.SummaryAPI)

	// Logout.
	protectedMux.HandleFunc("POST /admin/logout", a.AuthHandler.Logout)

	// 404 for unmatched admin routes.
	protectedMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		a.Renderer.NotFound(w)
	})

	// Wrap protected routes with auth middleware.
	protectedHandler := a.AuthMW.RequireLogin(protectedMux)

	// Mount protected routes under adminMux.
	// The auth middleware passes through /admin/login and /admin/logout to adminMux,
	// and requires login for everything else under /admin/.
	adminMux.Handle("/admin/login", adminMux) // already handled above
	adminMux.Handle("/admin/", protectedHandler)
	adminMux.Handle("/admin", protectedHandler)
	adminMux.Handle("/api/", protectedHandler)

	// Mount the entire admin sub-router at the root level.
	// adminMux handles all /admin/... and /api/... paths.
	mux.Handle("/admin/", adminMux)
	mux.Handle("/admin", adminMux)
	mux.Handle("/api/", adminMux)

	// Catch-all 404 for unmatched routes on main mux.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		a.Renderer.NotFound(w)
	})

	// --- Apply global middleware ---
	var handler http.Handler = mux

	// Stats tracking (records page views).
	handler = middleware.StatsTracking(a.StatsSvc)(handler)

	// CSRF protection.
	handler = a.CSRFMW.Protect(handler)

	// Rate limiting.
	handler = a.RateLimiter.Limit(handler)

	// Recovery from panics.
	handler = middleware.Recovery(a.logger)(handler)

	// Request logging (outermost - runs first, completes last).
	handler = middleware.Logger(a.logger)(handler)

	return handler
}
