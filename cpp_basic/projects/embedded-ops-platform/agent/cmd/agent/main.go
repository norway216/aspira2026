package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/embedded-ops-platform/agent/internal/collector"
	"github.com/embedded-ops-platform/agent/internal/command"
	"github.com/embedded-ops-platform/agent/internal/config"
	"github.com/embedded-ops-platform/agent/internal/heartbeat"
	"github.com/embedded-ops-platform/agent/internal/register"
	"github.com/embedded-ops-platform/agent/pkg/machineid"
)

// Compile-time reference to ensure unused imports are kept.
var _ = filepath.Join
var _ = os.Stat

var (
	configPath  = flag.String("config", "/etc/embedded-agent/config.yaml", "Path to config file")
	versionFlag = flag.Bool("version", false, "Show agent version")
)

const agentVersion = "1.0.0"

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Printf("Embedded Agent v%s\n", agentVersion)
		os.Exit(0)
	}

	// Seed random
	rand.Seed(time.Now().UnixNano())

	log.Printf("Embedded Agent v%s starting...\n", agentVersion)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("Warning: could not load config from %s: %v (using defaults)\n", *configPath, err)
		cfg = config.DefaultConfig()
	}

	// Ensure data directory exists
	if err := config.EnsureDir(cfg.Agent.DataDir); err != nil {
		log.Printf("Warning: could not create data dir: %v\n", err)
	}

	// Collect machine IDs
	machineID := machineid.GetMachineID()
	macAddresses := machineid.GetMACAddresses()

	// Try to load saved identity
	identity, err := config.LoadIdentity(cfg.Security.TokenFile)
	if err != nil {
		log.Printf("No saved identity found, attempting registration...\n")
		identity = registerAndRetry(cfg, machineID, macAddresses)
		if identity == nil {
			log.Fatalf("Failed to register agent")
		}
		log.Printf("Registration successful: device_id=%s\n", identity.DeviceID)
	} else {
		log.Printf("Loaded saved identity: device_id=%s\n", identity.DeviceID)
	}

	// Create sender for heartbeats and metrics
	sender := heartbeat.NewSender(cfg.Server.URL, identity.DeviceID, identity.Token)

	// Create command dispatcher
	dispatcher := command.NewDispatcher(cfg.Command.AllowReboot)

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start heartbeat loop
	go heartbeatLoop(ctx, cfg, sender)

	// Start metrics collection loop
	go metricsLoop(ctx, cfg, sender)

	// Start task polling loop
	go taskPollLoop(ctx, cfg, sender, dispatcher)

	log.Printf("Agent running. Device ID: %s\n", identity.DeviceID)

	// Wait for shutdown signal
	<-sigCh
	log.Println("Shutting down...")
	cancel()
	time.Sleep(1 * time.Second)
	log.Println("Agent stopped")
}

func registerAndRetry(cfg *config.Config, machineIDStr string, macs []string) *config.AgentIdentity {
	info := collector.CollectBasicInfo(machineIDStr, macs)
	infoMap := map[string]interface{}{
		"hostname":        info.Hostname,
		"machine_id":      info.MachineID,
		"mac_addresses":   info.MACAddresses,
		"arch":            info.Arch,
		"os_name":         info.OSName,
		"kernel_version":  info.KernelVersion,
		"board_type":      info.BoardType,
		"agent_version":   info.AgentVersion,
	}

	maxAttempts := 10
	baseDelay := 2 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, err := register.TryRegister(cfg, infoMap)
		if err == nil && result.Success {
			return result.Identity
		}

		delay := baseDelay * time.Duration(1<<uint(attempt-1))
		if delay > 60*time.Second {
			delay = 60 * time.Second
		}
		jitter := time.Duration(rand.Float64() * float64(delay) * 0.5)

		log.Printf("Registration attempt %d/%d failed: %v (retrying in %v)\n",
			attempt, maxAttempts, err, delay+jitter)
		time.Sleep(delay + jitter)
	}

	return nil
}

func heartbeatLoop(ctx context.Context, cfg *config.Config, sender *heartbeat.Sender) {
	interval := time.Duration(cfg.Collector.HeartbeatIntervalSec) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}

	// Add jitter to initial delay
	jitter := time.Duration(rand.Float64() * float64(interval) * 0.5)
	time.Sleep(jitter)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := sender.SendHeartbeat(); err != nil {
				log.Printf("Heartbeat error: %v\n", err)
			}
		}
	}
}

func metricsLoop(ctx context.Context, cfg *config.Config, sender *heartbeat.Sender) {
	interval := time.Duration(cfg.Collector.MetricsIntervalSec) * time.Second
	if interval <= 0 {
		interval = 10 * time.Second
	}

	// Add jitter
	jitter := time.Duration(rand.Float64() * float64(interval) * 0.5)
	time.Sleep(jitter)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := collector.CollectMetrics()
			if err := sender.SendMetrics(
				metrics.CPUUsage,
				metrics.MemoryUsage,
				metrics.DiskUsage,
				metrics.LoadAvg1m,
				metrics.Temperature,
				metrics.NetworkRx,
				metrics.NetworkTx,
			); err != nil {
				log.Printf("Metrics error: %v\n", err)
			}
		}
	}
}

func taskPollLoop(ctx context.Context, cfg *config.Config, sender *heartbeat.Sender, dispatcher *command.Dispatcher) {
	if !cfg.Command.EnableRemoteCommand {
		log.Println("Remote commands disabled")
		return
	}

	interval := 3 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tasks, err := sender.PullTasks()
			if err != nil {
				// Don't log polling errors too frequently
				continue
			}

			for _, taskData := range tasks {
				task := parseTask(taskData)
				if task == nil {
					continue
				}

				log.Printf("Executing command: %s (task_id=%s)\n", task.Action, task.TaskID)
				result := dispatcher.Dispatch(task)
				result.TaskID = task.TaskID

				if err := sender.SubmitTaskResult(result.TaskID, result.Status, result.Stdout, result.Stderr, result.ExitCode); err != nil {
					log.Printf("Failed to submit task result: %v\n", err)
				} else {
					log.Printf("Command result submitted: %s (status=%s)\n", task.TaskID, result.Status)
				}
			}
		}
	}
}

func parseTask(data map[string]interface{}) *command.Task {
	if data == nil {
		return nil
	}

	taskID, _ := data["task_id"].(string)
	action, _ := data["action"].(string)
	if taskID == "" || action == "" {
		return nil
	}

	params, _ := data["params"].(map[string]interface{})

	return &command.Task{
		TaskID: taskID,
		Action: action,
		Params: params,
	}
}

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}