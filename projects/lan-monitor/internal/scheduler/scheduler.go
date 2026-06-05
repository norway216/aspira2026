package scheduler

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"lan-monitor/internal/config"
	"lan-monitor/internal/database"
	"lan-monitor/internal/scanner"
	"lan-monitor/internal/websocket"
)

type Scheduler struct {
	scanner *scanner.ScannerService
	hub     *websocket.Hub
	cfg     *config.Config
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

func New(scannerSvc *scanner.ScannerService, hub *websocket.Hub, cfg *config.Config) *Scheduler {
	return &Scheduler{
		scanner: scannerSvc,
		hub:     hub,
		cfg:     cfg,
		stopCh:  make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	if s.cfg.Scan.Enabled {
		s.wg.Add(1)
		go s.scanLoop()
	}

	s.wg.Add(1)
	go s.trafficCollectLoop()

	s.wg.Add(1)
	go s.dataCleanupLoop()

	log.Println("[Scheduler] All tasks started")
}

func (s *Scheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	log.Println("[Scheduler] All tasks stopped")
}

func (s *Scheduler) scanLoop() {
	defer s.wg.Done()
	interval := time.Duration(s.cfg.Scan.IntervalSeconds) * time.Second
	time.Sleep(3 * time.Second)
	log.Println("[Scheduler] Running initial scan...")
	s.scanner.RunScan()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("[Scheduler] Starting periodic scan...")
			s.scanner.RunScan()
		case <-s.stopCh:
			return
		}
	}
}

func (s *Scheduler) trafficCollectLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.collectSystemTraffic()
		case <-s.stopCh:
			return
		}
	}
}

func (s *Scheduler) collectSystemTraffic() {
	stats, err := readNetworkStats()
	if err != nil {
		return
	}

	now := time.Now()
	for _, stat := range stats {
		database.DB.Exec(
			"INSERT INTO system_traffics (rx_bytes, tx_bytes, timestamp) VALUES (?, ?, ?)",
			stat.rxBytes, stat.txBytes, now)
	}
}

func (s *Scheduler) dataCleanupLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupOldData()
		case <-s.stopCh:
			return
		}
	}
}

func (s *Scheduler) cleanupOldData() {
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	database.DB.Exec("DELETE FROM traffic_records WHERE timestamp < ?", cutoff)
	database.DB.Exec("DELETE FROM system_traffics WHERE timestamp < ?", cutoff)

	scanCutoff := time.Now().Add(-30 * 24 * time.Hour)
	database.DB.Exec("DELETE FROM scan_tasks WHERE finished_at < ?", scanCutoff)

	log.Println("[Scheduler] Data cleanup completed")
}

type netStat struct {
	rxBytes int64
	txBytes int64
}

func readNetworkStats() (map[string]netStat, error) {
	result := make(map[string]netStat)

	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return result, nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum <= 2 {
			continue
		}
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		name := strings.TrimSuffix(fields[0], ":")
		rx, _ := strconv.ParseInt(fields[1], 10, 64)
		tx, _ := strconv.ParseInt(fields[9], 10, 64)

		result[name] = netStat{rxBytes: rx, txBytes: tx}
	}

	return result, nil
}
