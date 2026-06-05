package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"portfolio-site/internal/app"
	"portfolio-site/internal/config"
)

func main() {
	// Load configuration.
	cfg := config.Load()

	// Initialize structured JSON logger.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting portfolio site",
		"addr", cfg.Server.Addr,
		"db_driver", cfg.Database.Driver,
		"site_title", cfg.Site.Title,
	)

	// Create application with all dependencies.
	application, err := app.New(cfg, logger)
	if err != nil {
		logger.Error("failed to create application", "error", err)
		os.Exit(1)
	}

	// Configure HTTP server with sensible timeouts for high concurrency.
	server := &http.Server{
		Addr:           cfg.Server.Addr,
		Handler:        application.Router(),
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		IdleTimeout:    cfg.Server.IdleTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	// Start background cleanup workers.
	ctx, cancelCleanup := context.WithCancel(context.Background())
	defer cancelCleanup()

	// Clean old page view records (older than 90 days) daily.
	go application.StatsSvc.BackgroundCleanup(ctx, 90, 24*time.Hour)

	// Clean old audit logs (older than 365 days) daily.
	go application.AuditSvc.BackgroundCleanup(ctx, 365, 24*time.Hour)

	// Start server in a goroutine.
	go func() {
		logger.Info("server started",
			"addr", cfg.Server.Addr,
			"read_timeout", cfg.Server.ReadTimeout,
			"write_timeout", cfg.Server.WriteTimeout,
			"idle_timeout", cfg.Server.IdleTimeout,
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("received shutdown signal", "signal", sig.String())

	// Give outstanding requests a deadline to complete.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	// Gracefully shut down the HTTP server.
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}

	// Shutdown application (close DB connections, etc.).
	if err := application.Shutdown(); err != nil {
		logger.Error("application shutdown error", "error", err)
	}

	logger.Info("server stopped gracefully")
}
