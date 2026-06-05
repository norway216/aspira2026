package api

import (
	"io/fs"
	"net/http"
	"strings"

	"lan-monitor/internal/config"
	"lan-monitor/internal/middleware"
	"lan-monitor/internal/scanner"
	"lan-monitor/internal/websocket"

	"github.com/gin-gonic/gin"
)

func NewRouter(cfg *config.Config, scannerSvc *scanner.ScannerService, hub *websocket.Hub, staticFS fs.FS, indexHTML string) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())
	r.Use(middleware.AuditLogMiddleware())

	// API routes
	api := r.Group("/api/v1")
	{
		// Auth (no authentication required)
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/login", handleLogin)
			authGroup.POST("/refresh", handleRefresh)
		}

		// Agent endpoints (optional auth via token in body)
		agentGroup := api.Group("/agents")
		{
			agentGroup.POST("/register", handleAgentRegister)
			agentGroup.POST("/heartbeat", handleAgentHeartbeat)
			agentGroup.POST("/metrics", handleAgentMetrics)
		}

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.AuthRequired())
		{
			protected.GET("/auth/profile", handleProfile)
			protected.POST("/auth/logout", handleLogout)

			// Dashboard
			protected.GET("/dashboard", handleDashboard)

			// Devices
			deviceGroup := protected.Group("/devices")
			{
				deviceGroup.GET("", handleDeviceList)
				deviceGroup.GET("/online", handleOnlineDevices)
				deviceGroup.GET("/:id", handleDeviceDetail)
				deviceGroup.PUT("/:id", handleDeviceUpdate)
				deviceGroup.DELETE("/:id", middleware.AdminRequired(), handleDeviceDelete)
				deviceGroup.POST("/:id/ignore", handleDeviceIgnore)
				deviceGroup.POST("/:id/label", handleDeviceLabel)
				deviceGroup.GET("/:id/events", handleDeviceEvents)
				deviceGroup.GET("/:id/traffic", handleDeviceTraffic)
			}

			// Scans
			scanGroup := protected.Group("/scans")
			{
				scanGroup.POST("/start", middleware.AdminRequired(), func(c *gin.Context) {
					handleScanStart(c, scannerSvc)
				})
				scanGroup.GET("/tasks", handleScanTasks)
				scanGroup.GET("/tasks/:id", handleScanTaskDetail)
				scanGroup.GET("/config", handleGetScanConfig)
				scanGroup.PUT("/config", middleware.AdminRequired(), handleUpdateScanConfig)
				scanGroup.GET("/subnets", func(c *gin.Context) {
					handleGetSubnets(c, scannerSvc)
				})
			}

			// Traffic
			trafficGroup := protected.Group("/traffic")
			{
				trafficGroup.GET("/devices/:id/realtime", handleDeviceRealtimeTraffic)
				trafficGroup.GET("/devices/:id/history", handleDeviceTrafficHistory)
				trafficGroup.GET("/rank", handleTrafficRank)
				trafficGroup.GET("/summary", handleTrafficSummary)
			}

			// Alerts
			alertGroup := protected.Group("/alerts")
			{
				alertGroup.GET("", handleAlertList)
				alertGroup.GET("/:id", handleAlertDetail)
				alertGroup.PUT("/:id", middleware.AdminRequired(), handleAlertUpdate)
				alertGroup.POST("/:id/resolve", middleware.AdminRequired(), handleAlertResolve)
				alertGroup.GET("/stats", handleAlertStats)
			}

			// Users (admin only)
			userGroup := protected.Group("/users")
			{
				userGroup.GET("", handleUserList)
				userGroup.POST("", middleware.AdminRequired(), handleUserCreate)
				userGroup.PUT("/:id", middleware.AdminRequired(), handleUserUpdate)
				userGroup.DELETE("/:id", middleware.SuperAdminRequired(), handleUserDelete)
				userGroup.GET("/roles", handleRoleList)
			}

			// Audit logs (admin only)
			protected.GET("/audit", middleware.AdminRequired(), handleAuditLogs)

			// System
			protected.GET("/system/info", handleSystemInfo)
		}
	}

	// WebSocket
	r.GET("/ws", func(c *gin.Context) {
		hub.HandleWebSocket(c.Writer, c.Request)
	})

	// Serve static files (embedded) — use sub-filesystem rooted at web/static
	staticSub, _ := fs.Sub(staticFS, "web/static")
	r.GET("/static/*filepath", func(c *gin.Context) {
		path := c.Param("filepath")
		if path == "" || path == "/" {
			c.Status(http.StatusNotFound)
			return
		}
		path = path[1:] // strip leading slash
		data, err := fs.ReadFile(staticSub, path)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		contentType := "application/octet-stream"
		switch {
		case strings.HasSuffix(path, ".css"):
			contentType = "text/css; charset=utf-8"
		case strings.HasSuffix(path, ".js"):
			contentType = "application/javascript; charset=utf-8"
		case strings.HasSuffix(path, ".html"):
			contentType = "text/html; charset=utf-8"
		case strings.HasSuffix(path, ".png"):
			contentType = "image/png"
		case strings.HasSuffix(path, ".svg"):
			contentType = "image/svg+xml"
		case strings.HasSuffix(path, ".ico"):
			contentType = "image/x-icon"
		case strings.HasSuffix(path, ".json"):
			contentType = "application/json"
		}
		c.Header("Cache-Control", "public, max-age=3600")
		c.Data(http.StatusOK, contentType, data)
	})

	// Serve index.html for SPA
	r.GET("/", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, indexHTML)
	})

	// SPA fallback - serve index.html for all non-API, non-static routes
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if len(path) >= 4 && (path[:4] == "/api" || path[:4] == "/ws/") {
			c.JSON(404, gin.H{"error": "Not found"})
			return
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, indexHTML)
	})

	return r
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
