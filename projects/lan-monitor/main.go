package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"lan-monitor/internal/api"
	"lan-monitor/internal/config"
	"lan-monitor/internal/database"
	"lan-monitor/internal/scanner"
	"lan-monitor/internal/scheduler"
	"lan-monitor/internal/websocket"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("╔══════════════════════════════════════════╗")
	log.Println("║   局域网设备监控管理系统 v1.0.0         ║")
	log.Println("║   LAN Monitor & Management System      ║")
	log.Println("╚══════════════════════════════════════════╝")

	// 1. Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("[WARN] 无法加载配置文件 %s: %v", *configPath, err)
		log.Println("[INFO] 使用默认配置...")
		cfg = &config.Config{
			Server: config.ServerConfig{Addr: ":8080", Mode: "debug"},
			Scan: config.ScanConfig{
				Enabled:          true,
				IntervalSeconds:  60,
				Subnets:          []string{"auto"},
				Methods:          []string{"arp", "ping", "tcp"},
				WorkerCount:      100,
				TimeoutMs:        800,
				OfflineThreshold: 3,
				TCPPorts:         []int{22, 80, 443, 8080, 3389},
			},
			Database: config.DatabaseConfig{Path: "./data/lan_monitor.db"},
			JWT: config.JWTConfig{
				Secret:                   "lan-monitor-secret-change-in-production",
				AccessTokenExpireMinutes: 60,
				RefreshTokenExpireHours:  168,
			},
			Alert: config.AlertConfig{
				NewDeviceEnabled: true,
				OfflineEnabled:   true,
			},
		}
		config.AppConfig = cfg
	}

	log.Printf("[INFO] 服务器地址: %s", cfg.Server.Addr)
	log.Printf("[INFO] 数据库路径: %s", cfg.Database.Path)
	log.Printf("[INFO] 扫描间隔: %d 秒", cfg.Scan.IntervalSeconds)

	// 2. Initialize database
	log.Println("[INFO] 初始化数据库...")
	db, err := database.Init(cfg.Database.Path)
	if err != nil {
		log.Fatalf("[FATAL] 数据库初始化失败: %v", err)
	}
	log.Println("[INFO] 数据库初始化完成")
	_ = db

	// 3. Initialize WebSocket hub
	log.Println("[INFO] 初始化 WebSocket Hub...")
	hub := websocket.NewHub()
	api.SetAgentHub(hub)
	log.Println("[INFO] WebSocket Hub 就绪")

	// 4. Initialize scanner
	log.Println("[INFO] 初始化扫描服务...")
	scannerSvc := scanner.NewScannerService(&cfg.Scan, hub)

	// Detect local subnets
	subnets, _ := scannerSvc.GetLocalSubnets()
	log.Printf("[INFO] 检测到本地网段: %v", subnets)

	// 5. Initialize scheduler
	log.Println("[INFO] 启动定时任务调度器...")
	sched := scheduler.New(scannerSvc, hub, cfg)
	sched.Start()
	log.Println("[INFO] 定时任务调度器已启动")

	// 6. Setup HTTP router
	log.Println("[INFO] 初始化路由...")
	router := api.NewRouter(cfg, scannerSvc, hub, StaticFS, IndexHTML)

	// 7. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("[INFO] Web 服务启动于 http://0.0.0.0%s", cfg.Server.Addr)
		log.Println("[INFO] 默认管理员账户: admin / admin123")
		log.Println("[INFO] 按 Ctrl+C 停止服务")

		if err := router.Run(cfg.Server.Addr); err != nil {
			log.Fatalf("[FATAL] Web 服务启动失败: %v", err)
		}
	}()

	<-quit
	log.Println("[INFO] 正在关闭服务...")
	sched.Stop()
	log.Println("[INFO] 服务已关闭")
}
