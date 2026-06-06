package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// ============================================================
// CLI Flags
// ============================================================
var (
	targetURL   = flag.String("target", "http://localhost:8080", "Gateway base URL")
	batchSize   = flag.Int("batch", 5, "Transactions per batch per worker")
	batchPause  = flag.Duration("pause", 1*time.Second, "Pause between batches per worker")
	balance     = flag.Int64("balance", 10_000_000, "Initial account balance per currency")
	reportEvery = flag.Duration("report", 10*time.Second, "Stats report interval")
	loginUser   = flag.String("username", "admin", "Login username")
	loginPass   = flag.String("password", "admin123", "Login password")
	skipVerify  = flag.Bool("skip-verify", true, "Skip TLS verification")
)

// ============================================================
// Currency configuration — 8 workers, 8 currencies
// ============================================================
var currencies = []string{"USD", "HKD", "SGD", "JPY", "AUD", "EUR", "CNY", "CAD"}

// Payer accounts (merchant-1) — one per currency, each holds 10,000,000
var payerAccounts = map[string]string{
	"USD": "acct-001",
	"HKD": "acct-013",
	"SGD": "acct-012",
	"JPY": "acct-006",
	"AUD": "acct-010",
	"EUR": "acct-005",
	"CNY": "acct-002",
	"CAD": "acct-009",
}

// Payee accounts (merchant-2) — target accounts per currency
var payeeAccounts = map[string]string{
	"USD": "acct-003",
	"HKD": "acct-002", // CNY fallback
	"SGD": "acct-019",
	"JPY": "acct-015",
	"AUD": "acct-018",
	"EUR": "acct-014",
	"CNY": "acct-004",
	"CAD": "acct-001", // USD fallback
}

// ============================================================
// WorkerStats — per-worker metrics
// ============================================================
type WorkerStats struct {
	Currency   string
	Success    atomic.Int64
	Errors     atomic.Int64
	Skipped    atomic.Int64
	TotalAmt   atomic.Int64
	LastError  string
	LastErrAt  time.Time
	mu         sync.Mutex
}

func (s *WorkerStats) RecordSuccess() {
	s.Success.Add(1)
}
func (s *WorkerStats) RecordError(errMsg string) {
	s.Errors.Add(1)
	s.mu.Lock()
	s.LastError = errMsg
	s.LastErrAt = time.Now()
	s.mu.Unlock()
}
func (s *WorkerStats) RecordSkip() { s.Skipped.Add(1) }
func (s *WorkerStats) AddAmount(amt int64) { s.TotalAmt.Add(amt) }

// ============================================================
// HTTP Client with connection pool
// ============================================================
type GatewayClient struct {
	client  *http.Client
	baseURL string
	token   string
}

func NewGatewayClient(baseURL, token string) *GatewayClient {
	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: *skipVerify},
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 200,
		MaxConnsPerHost:     200,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
	}
	return &GatewayClient{
		client: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
		baseURL: baseURL,
		token:   token,
	}
}

func (c *GatewayClient) Login(username, password string) (string, error) {
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := c.client.Post(c.baseURL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("login: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	token, ok := result["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("no access_token in response")
	}
	c.token = token
	return token, nil
}

type TxnRequest struct {
	PayerAccountID string `json:"payer_account_id"`
	PayeeAccountID string `json:"payee_account_id"`
	SourceCurrency string `json:"source_currency"`
	TargetCurrency string `json:"target_currency"`
	SourceAmount   int64  `json:"source_amount"`
	Fee            int64  `json:"fee"`
	Description    string `json:"description"`
	ReferenceID    string `json:"reference_id"`
}

type TxnResponse struct {
	Transaction struct {
		ID             string `json:"id"`
		Status         string `json:"status"`
		SourceAmount   int64  `json:"source_amount"`
		TargetAmount   int64  `json:"target_amount"`
		SourceCurrency string `json:"source_currency"`
		TargetCurrency string `json:"target_currency"`
	} `json:"transaction"`
	Error string `json:"error,omitempty"`
}

func (c *GatewayClient) CreateTransaction(req *TxnRequest) (*TxnResponse, error) {
	// Retry up to 3 times for transient errors (SQLITE_BUSY, network)
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 50ms, 150ms
			time.Sleep(time.Duration(50*(1<<(attempt-1))) * time.Millisecond)
		}

		data, _ := json.Marshal(req)
		httpReq, _ := http.NewRequest("POST", c.baseURL+"/api/v1/transactions", bytes.NewReader(data))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
		httpReq.Header.Set("X-Request-Id", req.ReferenceID)
		httpReq.Header.Set("X-Merchant-Id", "merchant-001")

		resp, err := c.client.Do(httpReq)
		if err != nil {
			lastErr = err
			continue
		}

		var txnResp TxnResponse
		if err := json.NewDecoder(resp.Body).Decode(&txnResp); err != nil {
			resp.Body.Close()
			lastErr = fmt.Errorf("decode: %w", err)
			continue
		}
		resp.Body.Close()

		// Retry on 5xx (server errors including SQLITE_BUSY which returns 500)
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf(txnResp.Error)
			continue
		}

		if txnResp.Error != "" {
			// 4xx errors are non-retriable (validation, insufficient funds, etc.)
			return &txnResp, fmt.Errorf(txnResp.Error)
		}
		return &txnResp, nil
	}
	return nil, lastErr
}

// ============================================================
// Worker — one per currency
// ============================================================
func runWorker(currency string, client *GatewayClient, stats *WorkerStats, stopCh <-chan struct{}) {
	payer := payerAccounts[currency]

	// Build target currency list (all except own)
	targets := make([]string, 0, 7)
	for _, c := range currencies {
		if c != currency {
			targets = append(targets, c)
		}
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(len(currency))))
	counter := int64(0)

	for {
		select {
		case <-stopCh:
			return
		default:
		}

		// Process a batch
		for i := 0; i < *batchSize; i++ {
			select {
			case <-stopCh:
				return
			default:
			}

			counter++
			// Pick random target currency
			target := targets[rng.Intn(len(targets))]
			payee := payeeAccounts[target]

			// Random amount: 100 ~ 5000
			amount := int64(100 + rng.Intn(4901))
			fee := int64(1 + rng.Intn(20))
			ref := fmt.Sprintf("W-%s-%d-%d", currency, time.Now().UnixNano(), counter)

			req := &TxnRequest{
				PayerAccountID: payer,
				PayeeAccountID: payee,
				SourceCurrency: currency,
				TargetCurrency: target,
				SourceAmount:   amount,
				Fee:            fee,
				Description:    fmt.Sprintf("[%s→%s] batch", currency, target),
				ReferenceID:    ref,
			}

			resp, err := client.CreateTransaction(req)
			if err != nil {
				stats.RecordError(err.Error())
			} else if resp.Transaction.Status == "payment_confirmed" || resp.Transaction.Status == "completed" {
				stats.RecordSuccess()
				stats.AddAmount(amount)
			} else {
				stats.RecordSkip()
			}
		}

		// Pause between batches
		select {
		case <-stopCh:
			return
		case <-time.After(*batchPause + time.Duration(rng.Intn(200))*time.Millisecond):
		}
	}
}

// ============================================================
// Stats Display
// ============================================================
var colorCodes = map[string]string{
	"USD": "\033[0;32m", // Green
	"HKD": "\033[0;36m", // Cyan
	"SGD": "\033[0;34m", // Blue
	"JPY": "\033[0;35m", // Magenta
	"AUD": "\033[1;33m", // Yellow
	"EUR": "\033[0;36m", // Cyan
	"CNY": "\033[0;31m", // Red
	"CAD": "\033[0;35m", // Magenta
}
const colorReset = "\033[0m"
const colorCyan = "\033[0;36m"
const colorYellow = "\033[1;33m"
const colorGreen = "\033[0;32m"

func displayStats(workers map[string]*WorkerStats, startTime time.Time) {
	elapsed := time.Since(startTime).Seconds()

	fmt.Println()
	fmt.Printf("%s══════════════════════════════════════════════════════════%s\n", colorCyan, colorReset)
	fmt.Printf("%s  Aspira Pay — Multi-Currency Workers | %6.1fs elapsed%s\n", colorCyan, elapsed, colorReset)
	fmt.Printf("%s══════════════════════════════════════════════════════════%s\n", colorCyan, colorReset)

	var totalS, totalE, totalSk, totalAmt int64
	for _, cur := range currencies {
		s := workers[cur]
		succ := s.Success.Load()
		errs := s.Errors.Load()
		skip := s.Skipped.Load()
		amt := s.TotalAmt.Load()
		totalS += succ
		totalE += errs
		totalSk += skip
		totalAmt += amt

		total := succ + errs + skip
		rate := "0.0"
		if total > 0 {
			rate = fmt.Sprintf("%.1f", float64(succ)/float64(total)*100)
		}

		color := colorCodes[cur]
		fmt.Printf("  %s[%-3s]%s total=%6d  OK=%6d  ERR=%5d  skip=%4d  rate=%s%%  vol=%-8d\n",
			color, cur, colorReset, total, succ, errs, skip, rate, amt)
	}

	totalAll := totalS + totalE + totalSk
	avgTPS := 0.0
	if elapsed > 0 {
		avgTPS = float64(totalAll) / elapsed
	}
	rateAll := "0.0"
	if totalAll > 0 {
		rateAll = fmt.Sprintf("%.1f", float64(totalS)/float64(totalAll)*100)
	}

	fmt.Printf("  %s──────────────────────────────────────────────────────%s\n", colorCyan, colorReset)
	fmt.Printf("  %s[ALL]%s total=%-6d OK=%-6d ERR=%-5d skip=%-4d rate=%s%%  TPS=%.0f\n",
		colorYellow, colorReset, totalAll, totalS, totalE, totalSk, rateAll, avgTPS)
	fmt.Printf("  %sTotal Volume: %d across all currencies%s\n", colorGreen, totalAmt, colorReset)
	fmt.Printf("%s══════════════════════════════════════════════════════════%s\n", colorCyan, colorReset)
}

// ============================================================
// Main
// ============================================================
func main() {
	flag.Parse()

	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║   Aspira Pay — Multi-Currency Continuous Trader (Go)    ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Printf("  Target:       %s\n", *targetURL)
	fmt.Printf("  Workers:      8 (USD HKD SGD JPY AUD EUR CNY CAD)\n")
	fmt.Printf("  Batch size:   %d txns/batch\n", *batchSize)
	fmt.Printf("  Batch pause:  %v\n", *batchPause)
	fmt.Printf("  Balance:      %d per account\n", *balance)
	fmt.Println()

	// Create HTTP client and authenticate
	client := NewGatewayClient(*targetURL, "")
	log.Printf("Authenticating as %s...", *loginUser)
	token, err := client.Login(*loginUser, *loginPass)
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}
	log.Printf("✓ Authenticated (token: %s...)", token[:16])

	// Initialize worker stats
	workers := make(map[string]*WorkerStats)
	for _, cur := range currencies {
		workers[cur] = &WorkerStats{Currency: cur}
	}

	// Start all 8 workers
	stopCh := make(chan struct{})
	var wg sync.WaitGroup
	startTime := time.Now()

	for _, cur := range currencies {
		wg.Add(1)
		go func(currency string) {
			defer wg.Done()
			runWorker(currency, client, workers[currency], stopCh)
		}(cur)
	}

	log.Printf("✓ 8 workers started. Press Ctrl+C to stop.\n")

	// Periodic stats reporter
	go func() {
		ticker := time.NewTicker(*reportEvery)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				displayStats(workers, startTime)
			}
		}
	}()

	// Wait for Ctrl+C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\n\nShutting down...")
	close(stopCh)
	wg.Wait()

	// Final stats
	displayStats(workers, startTime)
	fmt.Println("\n✓ All workers stopped.")
}
