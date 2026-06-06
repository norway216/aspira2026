package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/aspira/crossborder-payment-gateway/config"
	"github.com/aspira/crossborder-payment-gateway/internal/auth"
	"github.com/aspira/crossborder-payment-gateway/internal/database"
	"github.com/aspira/crossborder-payment-gateway/internal/engine"
	"github.com/aspira/crossborder-payment-gateway/internal/handler"
	"github.com/aspira/crossborder-payment-gateway/internal/middleware"
	"github.com/aspira/crossborder-payment-gateway/internal/websocket"
	"github.com/gin-gonic/gin"
)

func main() {
	// Determine dashboard path
	// Try relative paths: ../dashboard, dashboard
	dashboardDir := "../dashboard"
	if _, err := os.Stat(dashboardDir); os.IsNotExist(err) {
		dashboardDir = "dashboard"
	}

	// Load config
	cfgPath := "configs/config.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create data directory for SQLite
	if cfg.Database.Driver == "sqlite" {
		dir := filepath.Dir(cfg.Database.DSN)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Warning: cannot create data dir: %v", err)
		}
	}

	// Initialize database
	db, err := database.NewSQLite(cfg.Database.DSN)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	log.Println("Database initialized")

	// Initialize JWT manager
	jwtMgr := auth.NewJWTManager(cfg.Auth.JWTSecret, cfg.Auth.TokenExpiry, cfg.Auth.RefreshExpiry)

	// Initialize engine client
	engineClient := engine.NewEngineClient(cfg.Engine.Addr, cfg.Engine.Enabled, cfg.Engine.Timeout)
	if cfg.Engine.Enabled {
		if err := engineClient.Connect(); err != nil {
			log.Printf("Warning: Engine connection failed: %v (using internal fallback)", err)
		} else {
			log.Printf("Connected to C++ engine at %s", cfg.Engine.Addr)
		}
	}

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()
	log.Println("WebSocket hub started")

	// Initialize handlers
	authH := handler.NewAuthHandler(db, jwtMgr)
	txnH := handler.NewTransactionHandler(db, engineClient, wsHub)
	acctH := handler.NewAccountHandler(db)
	merchantH := handler.NewMerchantHandler(db)
	dashboardH := handler.NewDashboardHandler(db, engineClient)
	auditH := handler.NewAuditHandler(db)
	exchangeH := handler.NewExchangeHandler(db)

	// Setup Gin
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// Global middleware
	r.Use(middleware.CORS())
	r.Use(middleware.AuditLog(db))
	r.Use(middleware.RateLimit(100, 200))

	// WebSocket endpoint
	r.GET("/ws", func(c *gin.Context) {
		wsHub.HandleWebSocket(c.Writer, c.Request)
	})

	// API routes
	api := r.Group("/api/v1")
	{
		// Public auth routes
		api.POST("/auth/login", authH.Login)
		api.POST("/auth/refresh", authH.Refresh)

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.AuthRequired(jwtMgr))
		{
			// Auth
			protected.GET("/auth/profile", authH.Profile)
			protected.POST("/auth/logout", authH.Logout)

			// Dashboard
			protected.GET("/dashboard", dashboardH.GetDashboard)
			protected.GET("/dashboard/tps-history", dashboardH.GetTPSHistory)
			protected.GET("/dashboard/volume-history", dashboardH.GetVolumeHistory)
			protected.GET("/dashboard/recent-transactions", dashboardH.GetRecentTransactions)

			// Transactions
			protected.GET("/transactions", txnH.ListTransactions)
			protected.POST("/transactions", txnH.CreateTransaction)
			protected.GET("/transactions/stats", txnH.GetStats)
			protected.GET("/transactions/:id", txnH.GetTransaction)
			protected.POST("/transactions/:id/refund", txnH.RefundTransaction)

			// Accounts
			protected.GET("/accounts", acctH.ListAccounts)
			protected.GET("/accounts/:id", acctH.GetAccount)
			protected.GET("/accounts/:id/ledger", acctH.GetAccountLedger)

			// Merchants (admin only for CUD)
			protected.GET("/merchants", merchantH.ListMerchants)
			protected.POST("/merchants", middleware.AdminRequired(), merchantH.CreateMerchant)
			protected.PUT("/merchants/:id", middleware.AdminRequired(), merchantH.UpdateMerchant)
			protected.DELETE("/merchants/:id", middleware.AdminRequired(), merchantH.DeleteMerchant)

			// Audit
			protected.GET("/audit", auditH.ListAuditLogs)

			// Exchange rates
			protected.GET("/exchange-rates", exchangeH.ListRates)
			protected.POST("/exchange-rates", middleware.AdminRequired(), exchangeH.UpsertRate)
		}
	}

	// Serve static files from dashboard directory
	r.Static("/static", filepath.Join(dashboardDir, "static"))

	// SPA fallback: serve index.html for all non-API, non-static routes
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ws") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		indexPath := filepath.Join(dashboardDir, "index.html")
		c.File(indexPath)
	})

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Graceful shutdown
	go func() {
		log.Printf("═══════════════════════════════════════════════════════")
		log.Printf("  Aspira Cross-Border Payment Gateway")
		log.Printf("  Dashboard: http://localhost%s", addr)
		log.Printf("  API Base:  http://localhost%s/api/v1", addr)
		log.Printf("  WebSocket: ws://localhost%s/ws", addr)
		log.Printf("═══════════════════════════════════════════════════════")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Start background stats broadcaster
	go broadcastStats(wsHub, db, engineClient)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	engineClient.Close()
	log.Println("Server stopped")
}

func broadcastStats(wsHub *websocket.Hub, db database.DB, engineClient *engine.EngineClient) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats, err := db.GetDashboardStats()
		if err != nil {
			continue
		}
		stats.EngineConnected = engineClient.IsEnabled() && engineClient.IsConnected()
		wsHub.BroadcastDashboardStats(stats)
		wsHub.BroadcastTPSUpdate(stats.CurrentTPS)
		wsHub.BroadcastEngineHealth(map[string]interface{}{
			"connected":      stats.EngineConnected,
			"active_conns":   wsHub.GetActiveConnections(),
			"engine_enabled": engineClient.IsEnabled(),
		})
	}
}
