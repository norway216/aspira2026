package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/embedded-ops-platform/backend/internal/api"
	"github.com/embedded-ops-platform/backend/internal/middleware"
	"github.com/embedded-ops-platform/backend/internal/repository"
	"github.com/embedded-ops-platform/backend/internal/service"
	"github.com/embedded-ops-platform/backend/internal/ws"
	"github.com/embedded-ops-platform/backend/pkg/config"
	"github.com/embedded-ops-platform/backend/pkg/logger"
	"github.com/embedded-ops-platform/backend/pkg/token"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger.Init(cfg.Server.Mode)
	defer logger.Sync()

	logger.Info("Starting Embedded Ops Platform Server",
		logger.String("host", cfg.Server.Host),
		logger.Int("port", cfg.Server.Port),
	)

	// Initialize database
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := repository.NewPostgresRepo(ctx, cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database", logger.ErrField(err))
	}
	defer db.Close()

	// Initialize schema
	if err := db.InitSchema(context.Background()); err != nil {
		logger.Fatal("Failed to initialize schema", logger.ErrField(err))
	}

	// Seed default data
	if err := db.SeedDefaultData(context.Background()); err != nil {
		logger.Warn("Failed to seed default data", logger.ErrField(err))
	}

	// Initialize Redis
	redisRepo := repository.NewRedisRepo(context.Background(), cfg.Redis)

	// Initialize WebSocket hub
	hub := ws.NewHub()
	go hub.Run()
	defer hub.Stop()

	// Initialize services
	registerSvc := service.NewRegisterService(db, cfg.JWT.Secret)
	deviceSvc := service.NewDeviceService(db, redisRepo, hub)
	commandSvc := service.NewCommandService(db, hub)

	// Initialize JWT token manager
	tokenManager := token.NewManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenTTL,
		cfg.JWT.RefreshTokenTTL,
	)

	// Initialize APIs
	authAPI := api.NewAuthAPI(db, tokenManager)
	agentAPI := api.NewAgentAPI(registerSvc, deviceSvc, commandSvc)
	deviceAPI := api.NewDeviceAPI(deviceSvc)
	commandAPI := api.NewCommandAPI(commandSvc)
	logAPI := api.NewLogAPI(db)
	alertAPI := api.NewAlertAPI(db)

	// Setup Gin router
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()

	// Recovery middleware
	router.Use(gin.Recovery())
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.LoggerMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		})
	})

	// WebSocket endpoint for frontend
	router.GET("/ws", func(c *gin.Context) {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.Warn("WebSocket upgrade failed", logger.ErrField(err))
			return
		}

		client := &ws.Client{
			Conn:   conn,
			Send:   make(chan []byte, 256),
			Hub:    hub,
			UserID: "anonymous",
		}
		hub.Register(client)
		go client.WritePump()
		go client.ReadPump()
	})

	// API v1 routes
	v1 := router.Group("/api/v1")

	// Public routes (no auth required)
	v1.POST("/agents/register", agentAPI.Register)
	v1.POST("/auth/login", authAPI.Login)

	// Protected routes (user JWT required)
	protected := v1.Group("")
	protected.Use(middleware.AuthMiddleware(tokenManager))
	{
		protected.GET("/auth/me", authAPI.Me)
		protected.GET("/devices", deviceAPI.ListDevices)
		protected.GET("/devices/stats", deviceAPI.GetDashboardStats)
		protected.GET("/devices/:device_id", deviceAPI.GetDevice)
		protected.GET("/devices/:device_id/metrics", deviceAPI.GetDeviceMetrics)
		protected.GET("/devices/:device_id/commands", commandAPI.ListCommands)
		protected.POST("/devices/:device_id/commands", commandAPI.CreateCommand)
		protected.GET("/devices/:device_id/logs", logAPI.GetLogs)
		protected.GET("/commands/actions", commandAPI.ListActions)
		protected.GET("/alerts", alertAPI.ListAlerts)
		protected.PUT("/alerts/:alert_id/resolve", alertAPI.ResolveAlert)
	}

	// Agent authenticated routes (agent token required)
	agentAuth := v1.Group("")
	agentAuth.Use(middleware.AgentAuthMiddleware(registerSvc.ValidateAgentToken))
	{
		agentAuth.POST("/agents/heartbeat", agentAPI.Heartbeat)
		agentAuth.POST("/agents/metrics", agentAPI.SubmitMetrics)
		agentAuth.GET("/agents/tasks/pull", agentAPI.PullTasks)
	}

	// Also allow agent auth on task result submission with token in body
	agentAuth.POST("/agents/tasks/:task_id/result", agentAPI.SubmitTaskResult)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start background goroutine for offline detection
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			ctx := context.Background()
			deviceSvc.CheckOfflineDevices(ctx)
		}
	}()

	// Start server in background
	go func() {
		logger.Info(fmt.Sprintf("Server listening on %s:%d", cfg.Server.Host, cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", logger.ErrField(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("Server forced to shutdown", logger.ErrField(err))
	}

	logger.Info("Server exited")
}