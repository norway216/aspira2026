package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// CLI Flags
var (
	mode        = flag.String("mode", "payment", "Test mode: payment, query, mixed")
	concurrency = flag.Int("concurrency", 20, "Number of concurrent workers")
	reqRate     = flag.Int("rate", 0, "Target requests per second (0 = unlimited)")
	duration    = flag.Duration("duration", 30*time.Second, "Test duration")
	rampUp      = flag.Duration("ramp-up", 5*time.Second, "Ramp-up duration")
	targetURL   = flag.String("target", "http://localhost:8080", "Gateway base URL")
	authToken   = flag.String("auth-token", "", "Pre-generated JWT token")
	reportFile  = flag.String("report", "", "Output report JSON file path")
	skipVerify  = flag.Bool("skip-verify", true, "Skip TLS certificate verification")
	loginUser   = flag.String("username", "admin", "Login username")
	loginPass   = flag.String("password", "admin123", "Login password")
)

// Transaction request
type TxnRequest struct {
	PayerAccountID string `json:"payer_account_id"`
	PayeeAccountID string `json:"payee_account_id"`
	SourceCurrency string `json:"source_currency"`
	TargetCurrency string `json:"target_currency"`
	SourceAmount   int64  `json:"source_amount"`
	Fee            int64   `json:"fee"`
	Description    string `json:"description"`
	ReferenceID    string `json:"reference_id"`
}

// Stats collector
type BenchStats struct {
	mu          sync.Mutex
	latencies   []float64
	errors      atomic.Int64
	successes   atomic.Int64
	skipped     atomic.Int64
	statusCodes map[int]int64
	startTime   time.Time
	tpsWindow   map[int64]int64
}

func NewBenchStats() *BenchStats {
	return &BenchStats{
		statusCodes: make(map[int]int64),
		startTime:   time.Now(),
		tpsWindow:   make(map[int64]int64),
	}
}

func (s *BenchStats) Record(latency float64, err error, code int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		s.errors.Add(1)
	} else if code == 422 {
		// Business rejection (insufficient funds, missing rate, etc.) — not a system error
		s.skipped.Add(1)
	} else if code >= 400 {
		s.errors.Add(1)
	} else {
		s.successes.Add(1)
		s.latencies = append(s.latencies, latency)
	}
	s.statusCodes[code]++

	sec := int64(time.Since(s.startTime).Seconds())
	s.tpsWindow[sec]++
}

func (s *BenchStats) GetSnapshot() (successes, errors, skipped int64) {
	return s.successes.Load(), s.errors.Load(), s.skipped.Load()
}

// Report
type BenchReport struct {
	Mode         string           `json:"mode"`
	DurationSec  float64          `json:"duration_sec"`
	Concurrency  int              `json:"concurrency"`
	TargetRate   int              `json:"target_rate"`
	TotalReqs    int64            `json:"total_requests"`
	TotalErrors  int64            `json:"total_errors"`
	TotalSkipped int64            `json:"total_skipped"`
	SuccessRate  float64          `json:"success_rate"`
	AvgTPS       float64          `json:"avg_tps"`
	PeakTPS      float64          `json:"peak_tps"`
	AvgLatencyMs float64          `json:"latency_ms_avg"`
	P50Ms        float64          `json:"latency_ms_p50"`
	P90Ms        float64          `json:"latency_ms_p90"`
	P95Ms        float64          `json:"latency_ms_p95"`
	P99Ms        float64          `json:"latency_ms_p99"`
	MaxMs        float64          `json:"latency_ms_max"`
	MinMs        float64          `json:"latency_ms_min"`
	StatusCodes  map[string]int64 `json:"status_codes"`
}

func (s *BenchStats) GenerateReport(mode string, concurrency, targetRate int) *BenchReport {
	s.mu.Lock()
	defer s.mu.Unlock()

	duration := time.Since(s.startTime).Seconds()
	suc := s.successes.Load()
	err := s.errors.Load()
	skp := s.skipped.Load()
	total := suc + err + skp

	avgTPS := float64(0)
	if duration > 0 {
		avgTPS = float64(total) / duration
	}

	var peakTPS float64
	for _, c := range s.tpsWindow {
		if float64(c) > peakTPS {
			peakTPS = float64(c)
		}
	}

	sorted := make([]float64, len(s.latencies))
	copy(sorted, s.latencies)
	sort.Float64s(sorted)

	var avg, p50, p90, p95, p99, maxV, minV float64
	if len(sorted) > 0 {
		var sum float64
		for _, l := range sorted {
			sum += l
		}
		avg = sum / float64(len(sorted))
		maxV = sorted[len(sorted)-1]
		minV = sorted[0]
		idx := len(sorted) - 1
		if idx < 0 {
			idx = 0
		}
		p50 = sorted[idx*50/100]
		p90 = sorted[idx*90/100]
		p95 = sorted[idx*95/100]
		p99 = sorted[idx*99/100]
	}

	successRate := float64(0)
	if total > 0 {
		successRate = float64(suc) / float64(total) * 100
		// Clamp to [0, 100] to prevent floating-point / race artifacts exceeding 100%
		if successRate > 100 {
			successRate = 100
		}
	}

	codes := make(map[string]int64)
	for code, count := range s.statusCodes {
		codes[fmt.Sprintf("%d", code)] = count
	}

	return &BenchReport{
		Mode:         mode,
		DurationSec:  duration,
		Concurrency:  concurrency,
		TargetRate:   targetRate,
		TotalReqs:    total,
		TotalErrors:  err,
		TotalSkipped: skp,
		SuccessRate:  successRate,
		AvgTPS:       avgTPS,
		PeakTPS:      peakTPS,
		AvgLatencyMs: avg,
		P50Ms:        p50,
		P90Ms:        p90,
		P95Ms:        p95,
		P99Ms:        p99,
		MaxMs:        maxV,
		MinMs:        minV,
		StatusCodes:  codes,
	}
}

// HTTP Client
type BenchClient struct {
	client  *http.Client
	baseURL string
	token   string
}

func NewBenchClient(baseURL, token string, skipVerify bool) *BenchClient {
	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: skipVerify},
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 200,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
	}
	return &BenchClient{
		client: &http.Client{Transport: tr, Timeout: 30 * time.Second},
		baseURL: baseURL,
		token:   token,
	}
}

func (c *BenchClient) Login(username, password string) (string, error) {
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := c.client.Post(c.baseURL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("login response parse: %w", err)
	}
	token, ok := result["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("no access_token in login response")
	}
	c.token = token
	return token, nil
}

func (c *BenchClient) CreateTransaction(req *TxnRequest) (int, float64, error) {
	data, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", c.baseURL+"/api/v1/transactions", bytes.NewReader(data))
	httpReq.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	start := time.Now()
	resp, err := c.client.Do(httpReq)
	latency := float64(time.Since(start).Microseconds()) / 1000.0
	if err != nil {
		return 0, latency, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, latency, nil
}

func (c *BenchClient) GetTransactions() (int, float64, error) {
	httpReq, _ := http.NewRequest("GET", c.baseURL+"/api/v1/transactions?page=1&page_size=5", nil)
	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	start := time.Now()
	resp, err := c.client.Do(httpReq)
	latency := float64(time.Since(start).Microseconds()) / 1000.0
	if err != nil {
		return 0, latency, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, latency, nil
}

// Rate limiter (token bucket)
type RateLimiter struct {
	rate     float64
	maxTokens float64
	tokens   float64
	lastTime time.Time
	mu       sync.Mutex
}

func NewRateLimiter(rps int) *RateLimiter {
	return &RateLimiter{
		rate:     float64(rps),
		maxTokens: float64(rps),
		tokens:   float64(rps),
		lastTime: time.Now(),
	}
}

func (rl *RateLimiter) Wait() {
	if rl.rate <= 0 {
		return
	}
	rl.mu.Lock()
	now := time.Now()
	elapsed := now.Sub(rl.lastTime).Seconds()
	rl.tokens += elapsed * rl.rate
	if rl.tokens > rl.maxTokens {
		rl.tokens = rl.maxTokens
	}
	rl.lastTime = now

	if rl.tokens < 1 {
		sleepTime := time.Duration((1 - rl.tokens) / rl.rate * float64(time.Second))
		rl.mu.Unlock()
		time.Sleep(sleepTime)
	} else {
		rl.tokens--
		rl.mu.Unlock()
	}
}

// Workload generators
// Currency -> payer (merchant-1) / payee (merchant-2) account mapping
var currencyAccounts = map[string]struct{ payer, payee string }{
	"USD": {"acct-001", "acct-003"},
	"CNY": {"acct-002", "acct-004"},
	"EUR": {"acct-005", "acct-014"},
	"JPY": {"acct-006", "acct-015"},
	"GBP": {"acct-007", "acct-016"},
	"CHF": {"acct-008", "acct-017"},
	"CAD": {"acct-009", "acct-001"}, // fallback payee: USD account
	"AUD": {"acct-010", "acct-018"},
	"NZD": {"acct-011", "acct-010"}, // fallback payee: AUD account
	"SGD": {"acct-012", "acct-019"},
	"HKD": {"acct-013", "acct-002"}, // fallback payee: CNY account
}

var (
	reqCounter int64
	currencies = []string{"USD", "CNY", "EUR", "JPY", "GBP", "CHF", "CAD", "AUD", "NZD", "SGD", "HKD"}
)

func genPaymentRequest() *TxnRequest {
	c := atomic.AddInt64(&reqCounter, 1)

	// Pick source currency and look up its accounts
	srcIdx := c % int64(len(currencies))
	src := currencies[srcIdx]

	// Pick a different target currency (rotate through available)
	tgtIdx := (srcIdx + 1 + c/10) % int64(len(currencies))
	tgt := currencies[tgtIdx]

	srcAccts := currencyAccounts[src]
	tgtAccts := currencyAccounts[tgt]

	return &TxnRequest{
		PayerAccountID: srcAccts.payer,
		PayeeAccountID: tgtAccts.payee,
		SourceCurrency: src,
		TargetCurrency: tgt,
		SourceAmount:   10000 + c%90000,
		Fee:            100 + c%400,
		Description:    fmt.Sprintf("Benchmark #%d", c),
		ReferenceID:    fmt.Sprintf("BENCH-%s-%d", uuid.New().String()[:8], c),
	}
}

func runWorker(client *BenchClient, stats *BenchStats, limiter *RateLimiter, stopCh <-chan struct{}) {
	for {
		select {
		case <-stopCh:
			return
		default:
		}

		limiter.Wait()

		var code int
		var latency float64
		var err error

		switch *mode {
		case "payment":
			req := genPaymentRequest()
			code, latency, err = client.CreateTransaction(req)
		case "query":
			code, latency, err = client.GetTransactions()
		case "mixed":
			c := atomic.AddInt64(&reqCounter, 1)
			if c%5 == 0 {
				code, latency, err = client.GetTransactions()
			} else {
				req := genPaymentRequest()
				code, latency, err = client.CreateTransaction(req)
			}
		default:
			req := genPaymentRequest()
			code, latency, err = client.CreateTransaction(req)
		}

		stats.Record(latency, err, code)
	}
}

func main() {
	flag.Parse()

	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║     Aspira Cross-Border Payment Gateway Benchmark        ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Printf("  Target:      %s\n", *targetURL)
	fmt.Printf("  Mode:        %s\n", *mode)
	fmt.Printf("  Concurrency: %d workers\n", *concurrency)
	fmt.Printf("  Rate limit:  %d req/s (0=unlimited)\n", *reqRate)
	fmt.Printf("  Duration:    %v\n", *duration)
	fmt.Printf("  Ramp-up:     %v\n", *rampUp)
	fmt.Println()

	// Create client
	client := NewBenchClient(*targetURL, *authToken, *skipVerify)

	// Auto-login if no token
	if client.token == "" {
		log.Printf("Authenticating as %s...", *loginUser)
		token, err := client.Login(*loginUser, *loginPass)
		if err != nil {
			log.Fatalf("Login failed: %v", err)
		}
		log.Printf("✓ Authenticated (token: %s...)", token[:20])
	} else {
		log.Printf("✓ Using provided auth token")
	}

	// Rate limiter
	limiter := NewRateLimiter(*reqRate)

	// Stats
	stats := NewBenchStats()

	// Start workers with ramp-up
	var wg sync.WaitGroup
	stopCh := make(chan struct{})
	startTime := time.Now()

	rampInterval := time.Duration(0)
	if *concurrency > 1 && *rampUp > 0 {
		rampInterval = *rampUp / time.Duration(*concurrency)
	}

	workersStarted := int32(0)
	go func() {
		for i := 0; i < *concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				runWorker(client, stats, limiter, stopCh)
			}()
			atomic.AddInt32(&workersStarted, 1)
			if rampInterval > 0 {
				time.Sleep(rampInterval)
			}
		}
	}()

	// Progress display
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			elapsed := time.Since(startTime).Seconds()
			successes, errors, skipped := stats.GetSnapshot()
			total := successes + errors + skipped
			tps := float64(0)
			if elapsed > 0 {
				tps = float64(total) / elapsed
			}
			w := atomic.LoadInt32(&workersStarted)
			rate := float64(0)
			if total > 0 && elapsed > 0 {
				rate = float64(successes) / float64(total) * 100
				// Clamp to [0, 100] to prevent display artifacts
				if rate > 100 {
					rate = 100
				}
			}
			fmt.Printf("\r[%5.1fs] Workers: %3d | Reqs: %8d | Errors: %5d | Skip: %5d | TPS: %8.1f | OK: %5.1f%%",
				elapsed, w, total, errors, skipped, tps, rate)
		}
	}()

	// Run for duration
	time.Sleep(*duration)
	close(stopCh)
	wg.Wait()
	ticker.Stop()

	elapsed := time.Since(startTime)
	fmt.Printf("\r[%5.1fs] Benchmark complete.                      \n", elapsed.Seconds())

	// Generate and print report
	report := stats.GenerateReport(*mode, *concurrency, *reqRate)

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("                   BENCHMARK RESULTS")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("  Duration:        %8.2f s\n", report.DurationSec)
	fmt.Printf("  Concurrency:     %8d workers\n", report.Concurrency)
	fmt.Printf("  Total Requests:  %8d\n", report.TotalReqs)
	fmt.Printf("  Total Errors:    %8d\n", report.TotalErrors)
	fmt.Printf("  Success Rate:    %7.2f %%\n", report.SuccessRate)
	fmt.Println("───────────────────────────────────────────────────────────")
	fmt.Printf("  Avg TPS:         %8.1f req/s\n", report.AvgTPS)
	fmt.Printf("  Peak TPS:        %8.1f req/s\n", report.PeakTPS)
	fmt.Println("───────────────────────────────────────────────────────────")
	fmt.Printf("  Latency Avg:     %8.2f ms\n", report.AvgLatencyMs)
	fmt.Printf("  Latency P50:     %8.2f ms\n", report.P50Ms)
	fmt.Printf("  Latency P90:     %8.2f ms\n", report.P90Ms)
	fmt.Printf("  Latency P95:     %8.2f ms\n", report.P95Ms)
	fmt.Printf("  Latency P99:     %8.2f ms\n", report.P99Ms)
	fmt.Printf("  Latency Max:     %8.2f ms\n", report.MaxMs)
	fmt.Printf("  Latency Min:     %8.2f ms\n", report.MinMs)
	fmt.Println("───────────────────────────────────────────────────────────")
	fmt.Println("  Status Codes:")
	for code, count := range report.StatusCodes {
		fmt.Printf("    HTTP %s: %d\n", code, count)
	}
	fmt.Println("═══════════════════════════════════════════════════════════")

	// Save report file
	if *reportFile != "" {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			log.Printf("Marshal error: %v", err)
		} else if err := os.WriteFile(*reportFile, data, 0644); err != nil {
			log.Printf("Write error: %v", err)
		} else {
			log.Printf("✓ Report saved to %s", *reportFile)
		}
	}
}
