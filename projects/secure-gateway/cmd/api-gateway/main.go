package main

import (
	"context"
	"database/sql"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/gin-gonic/gin"
	"github.com/secure-gateway/internal/auth"
	"github.com/secure-gateway/internal/config"
	"github.com/secure-gateway/internal/handler"
	"github.com/secure-gateway/internal/middleware"
	"github.com/secure-gateway/pkg/limiter"
	"github.com/secure-gateway/pkg/logger"
)

func main() {
	configPath := flag.String("config", "configs/api-gateway.yaml", "Path to config file")
	flag.Parse()

	// Load config
	cfg, err := config.LoadAPIGatewayConfig(*configPath)
	if err != nil {
		// Try alternate path
		cfg, err = config.LoadAPIGatewayConfig("/app/configs/api-gateway.yaml")
		if err != nil {
			logger.Init("info", "")
			logger.Fatal("Failed to load config", "error", err)
		}
	}

	// Init logger
	logger.Init(cfg.Log.Level, cfg.Log.File)
	defer logger.Sync()

	logger.Info("Starting API Gateway", "listen", cfg.Server.Listen)

	// Connect to database
	db, err := sql.Open(cfg.Database.Driver, cfg.Database.DSN)
	if err != nil {
		logger.Fatal("Failed to open database", "error", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		logger.Fatal("Failed to ping database", "error", err)
	}
	logger.Info("Database connected")

	// Auto migrate tables
	if err := autoMigrate(db); err != nil {
		logger.Fatal("Failed to migrate database", "error", err)
	}

	// Seed default admin
	seedDefaultAdmin(db)

	// Init JWT manager
	jwtManager, err := auth.NewJWTManager(
		cfg.JWT.Issuer,
		cfg.JWT.AccessTokenTTL,
		cfg.JWT.RefreshTokenTTL,
		cfg.JWT.SigningKey,
		cfg.JWT.RefreshSigningKey,
	)
	if err != nil {
		logger.Fatal("Failed to init JWT manager", "error", err)
	}

	// Init rate limiter
	var rl *limiter.RateLimiter
	if cfg.RateLimit.Enabled {
		rl = limiter.NewRateLimiter(cfg.RateLimit.RequestsPerMinute)
		rl.CleanupStale(5 * time.Minute)
	}

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.LoggerMiddleware())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "api-gateway"})
	})

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Auth routes (no auth required)
		authH := handler.NewAuthHandler(jwtManager, db)
		v1.POST("/auth/login", rlMiddleware(rl), authH.Login)
		v1.POST("/auth/refresh", authH.Refresh)

		// Authenticated routes
		authG := v1.Group("")
		authG.Use(middleware.AuthMiddleware(jwtManager))
		{
			authG.POST("/auth/logout", authH.Logout)
			authG.GET("/auth/profile", authH.GetProfile)

			// Dashboard (authenticated, admin-only for some)
			dashH := handler.NewDashboardHandler(db)
			authG.GET("/dashboard/stats", dashH.GetStats)
			authG.GET("/dashboard/distribution", dashH.GetNodeDistribution)
			authG.GET("/dashboard/traffic-history", dashH.GetTrafficHistory)
			authG.GET("/dashboard/top-users", dashH.GetTopUsers)

			// User management (admin only)
			userH := handler.NewUserHandler(db)
			usersG := authG.Group("/users")
			usersG.Use(middleware.AdminMiddleware())
			{
				usersG.GET("", userH.ListUsers)
				usersG.POST("", userH.CreateUser)
				usersG.PUT("/:id", userH.UpdateUser)
				usersG.DELETE("/:id", userH.DeleteUser)
			}

			// Node management
			nodeH := handler.NewNodeHandler(db)
			nodesG := authG.Group("/nodes")
			{
				nodesG.GET("", nodeH.ListNode)
				nodesG.GET("/:node_id/metrics", nodeH.GetNodeMetrics)
				nodesG.GET("/:node_id/metrics/history", nodeH.GetNodeMetricsHistory)
			}

			// Traffic records
			trafficH := handler.NewTrafficHandler(db)
			authG.GET("/traffic", trafficH.ListTraffic)
			authG.GET("/traffic/summary", trafficH.GetTrafficSummary)

			// Audit logs (admin only)
			authG.GET("/audit-logs", middleware.AdminMiddleware(), func(c *gin.Context) {
				rows, err := db.Query(
					`SELECT id, user_id, action, resource, ip_addr, user_agent, detail, created_at
					 FROM audit_logs ORDER BY created_at DESC LIMIT 100`,
				)
				if err != nil {
					c.JSON(500, gin.H{"code": 500, "message": "Query failed"})
					return
				}
				defer rows.Close()

				type AuditLog struct {
					ID        int64      `json:"id"`
					UserID    *int64     `json:"user_id,omitempty"`
					Action    string     `json:"action"`
					Resource  string     `json:"resource,omitempty"`
					IPAddr    string     `json:"ip_addr,omitempty"`
					UserAgent string     `json:"user_agent,omitempty"`
					Detail    string     `json:"detail,omitempty"`
					CreatedAt time.Time  `json:"created_at"`
				}

				var logs []AuditLog
				for rows.Next() {
					var l AuditLog
					if err := rows.Scan(&l.ID, &l.UserID, &l.Action, &l.Resource, &l.IPAddr, &l.UserAgent, &l.Detail, &l.CreatedAt); err != nil {
						continue
					}
					logs = append(logs, l)
				}

				c.JSON(200, gin.H{"code": 200, "message": "Success", "data": logs})
			})
		}

		// Node registration and heartbeat (authenticated by node secret)
		nodeH := handler.NewNodeHandler(db)
		v1.POST("/nodes/register", nodeH.RegisterNode)
		v1.POST("/nodes/heartbeat", nodeH.Heartbeat)
	}

	// HTTP server
	srv := &http.Server{
		Addr:         cfg.Server.Listen,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logger.Info("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			logger.Fatal("Server forced to shutdown", "error", err)
		}
		logger.Info("Server exited")
	}()

	// Start server
	if cfg.Server.TLSEnabled {
		logger.Info("Starting HTTPS server", "addr", cfg.Server.Listen)
		if err := srv.ListenAndServeTLS(cfg.Server.TLSCert, cfg.Server.TLSKey); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server error", "error", err)
		}
	} else {
		logger.Info("Starting HTTP server", "addr", cfg.Server.Listen)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server error", "error", err)
		}
	}
}

func rlMiddleware(rl *limiter.RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rl == nil {
			c.Next()
			return
		}
		if !rl.Allow(c.ClientIP()) {
			c.JSON(http.StatusTooManyRequests, gin.H{"code": 429, "message": "Too many requests"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func autoMigrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY,
		username VARCHAR(64) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(32) NOT NULL DEFAULT 'user',
		status VARCHAR(32) NOT NULL DEFAULT 'active',
		traffic_quota_bytes BIGINT NOT NULL DEFAULT 0,
		traffic_used_bytes BIGINT NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	CREATE TABLE IF NOT EXISTS nodes (
		id BIGSERIAL PRIMARY KEY,
		node_id VARCHAR(128) NOT NULL UNIQUE,
		name VARCHAR(128) NOT NULL,
		region VARCHAR(64) NOT NULL,
		public_addr VARCHAR(255) NOT NULL,
		status VARCHAR(32) NOT NULL DEFAULT 'offline',
		weight INT NOT NULL DEFAULT 100,
		max_connections INT NOT NULL DEFAULT 10000,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	CREATE TABLE IF NOT EXISTS node_metrics (
		id BIGSERIAL PRIMARY KEY,
		node_id VARCHAR(128) NOT NULL,
		cpu_usage DOUBLE PRECISION NOT NULL,
		mem_usage DOUBLE PRECISION NOT NULL,
		active_connections INT NOT NULL,
		rx_bytes_per_sec BIGINT NOT NULL,
		tx_bytes_per_sec BIGINT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	CREATE TABLE IF NOT EXISTS traffic_records (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT NOT NULL,
		node_id VARCHAR(128) NOT NULL,
		rx_bytes BIGINT NOT NULL DEFAULT 0,
		tx_bytes BIGINT NOT NULL DEFAULT 0,
		duration_seconds BIGINT NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	CREATE TABLE IF NOT EXISTS audit_logs (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT,
		action VARCHAR(128) NOT NULL,
		resource VARCHAR(255),
		ip_addr VARCHAR(64),
		user_agent TEXT,
		detail TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	`
	_, err := db.Exec(schema)
	if err != nil {
		return err
	}
	logger.Info("Database migration completed")
	return nil
}

func seedDefaultAdmin(db *sql.DB) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil || count > 0 {
		return
	}

	hash, err := auth.HashPassword("admin123")
	if err != nil {
		logger.Error("Failed to hash default password", "error", err)
		return
	}

	_, err = db.Exec("INSERT INTO users (username, password_hash, role, status) VALUES ($1, $2, $3, $4)",
		"admin", hash, "admin", "active")
	if err != nil {
		logger.Error("Failed to seed admin user", "error", err)
		return
	}
	logger.Info("Default admin user created (admin / admin123)")
}