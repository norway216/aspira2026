package app

import (
	"context"
	"fmt"
	"log/slog"

	"portfolio-site/internal/config"
	"portfolio-site/internal/database"
	"portfolio-site/internal/middleware"
	"portfolio-site/internal/module/article"
	"portfolio-site/internal/module/audit"
	"portfolio-site/internal/module/auth"
	"portfolio-site/internal/module/page"
	"portfolio-site/internal/module/project"
	"portfolio-site/internal/module/stats"
	"portfolio-site/internal/render"
)

// Application holds all dependencies for the web application.
type Application struct {
	cfg    *config.Config
	logger *slog.Logger
	db     *database.DB

	// Renderer.
	Renderer *render.Renderer

	// Middleware.
	AuthMW   *middleware.Auth
	CSRFMW   *middleware.CSRF
	RateLimiter *middleware.RateLimiter

	// Services.
	ProjectSvc *project.Service
	ArticleSvc *article.Service
	AuthSvc    *auth.Service
	StatsSvc   *stats.Service
	AuditSvc   *audit.Service

	// Handlers.
	PageHandler       *page.Handler
	ProjectHandler    *project.Handler
	ArticleHandler    *article.Handler
	ArticleAdminHandler *article.AdminHandler
	AuthHandler       *auth.Handler
	StatsHandler      *stats.Handler
	AuditHandler      *audit.Handler
}

// New creates a new Application with all dependencies wired up.
func New(cfg *config.Config, logger *slog.Logger) (*Application, error) {
	// Initialize database.
	db, err := database.New(database.DriverConfig{
		Driver:          cfg.Database.Driver,
		DSN:             cfg.Database.DSN,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}

	// Run migrations.
	if err := db.RunMigrations(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize renderer.
	renderer, err := render.NewRenderer("web/templates")
	if err != nil {
		logger.Warn("template renderer init warning (templates may not exist yet)", "error", err)
		// Create a minimal renderer that won't fail.
		renderer = &render.Renderer{}
	}

	// Initialize repositories.
	projectRepo := project.NewRepository(db.DB)
	articleRepo := article.NewRepository(db.DB)
	authRepo := auth.NewRepository(db.DB)
	statsRepo := stats.NewRepository(db.DB)
	auditRepo := audit.NewRepository(db.DB)

	// Initialize services.
	projectSvc := project.NewService(projectRepo)
	articleSvc := article.NewService(articleRepo)
	authSvc := auth.NewService(authRepo)
	statsSvc := stats.NewService(statsRepo, cfg.Session.Secret)
	auditSvc := audit.NewService(auditRepo, logger)

	// Ensure default admin user exists.
	if err := authSvc.EnsureDefaultAdmin(context.Background()); err != nil {
		logger.Warn("failed to ensure default admin", "error", err)
	}

	// Initialize middleware.
	authMW := middleware.NewAuth(cfg.Session.Name)
	csrfMW := middleware.NewCSRF()
	rateLimiter := middleware.NewRateLimiter(10, 50, logger)

	// Initialize handlers.
	pageHandler := page.NewHandler(projectSvc, articleSvc, renderer, logger, cfg.Site.Title)
	projectHandler := project.NewHandler(projectSvc, auditSvc, renderer, logger)
	articleHandler := article.NewHandler(articleSvc, renderer, logger)
	articleAdminHandler := article.NewAdminHandler(articleSvc, auditSvc, renderer, logger, db.DB)
	authHandler := auth.NewHandler(authSvc, auditSvc, authMW, renderer, logger)
	statsHandler := stats.NewHandler(statsSvc, renderer, logger)
	auditHandler := audit.NewHandler(auditSvc, renderer, logger)

	app := &Application{
		cfg:    cfg,
		logger: logger,
		db:     db,

		Renderer:    renderer,
		AuthMW:      authMW,
		CSRFMW:      csrfMW,
		RateLimiter: rateLimiter,

		ProjectSvc: projectSvc,
		ArticleSvc: articleSvc,
		AuthSvc:    authSvc,
		StatsSvc:   statsSvc,
		AuditSvc:   auditSvc,

		PageHandler:        pageHandler,
		ProjectHandler:     projectHandler,
		ArticleHandler:     articleHandler,
		ArticleAdminHandler: articleAdminHandler,
		AuthHandler:        authHandler,
		StatsHandler:       statsHandler,
		AuditHandler:       auditHandler,
	}

	return app, nil
}

// Shutdown gracefully shuts down the application.
func (a *Application) Shutdown() error {
	if a.db != nil {
		a.logger.Info("closing database connection")
		return a.db.Close()
	}
	return nil
}

// Config returns the application configuration.
func (a *Application) Config() *config.Config {
	return a.cfg
}

// Logger returns the application logger.
func (a *Application) Logger() *slog.Logger {
	return a.logger
}
