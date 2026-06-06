package exchange

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// RateService fetches and caches live exchange rates from an external API.
// All rates are expressed as: 1 USD = X [currency]
// Fallback to DB rates if API is unavailable.
type RateService struct {
	apiURL   string
	ttl      time.Duration
	enabled  bool
	client   *http.Client
	cache    map[string]float64
	lastFetch time.Time
	mu       sync.RWMutex

	// Fallback: direct rates from DB for known pairs
	dbFallback DBFallback
}

// DBFallback provides exchange rates from the local database when the API is unavailable.
type DBFallback interface {
	GetExchangeRate(source, target string) (*DBRate, error)
}

// DBRate mirrors models.ExchangeRate without creating a circular import.
type DBRate struct {
	Rate float64
}

// APIResponse is the JSON structure from open.er-api.com.
type APIResponse struct {
	Result      string             `json:"result"`
	BaseCode    string             `json:"base_code"`
	Rates       map[string]float64 `json:"rates"`
	TimeLastUpdate int64           `json:"time_last_update_unix"`
}

func NewRateService(apiURL string, ttl time.Duration, enabled bool) *RateService {
	return &RateService{
		apiURL:  apiURL,
		ttl:     ttl,
		enabled: enabled,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: make(map[string]float64),
	}
}

// SetDBFallback sets the database fallback for when the API is unavailable.
func (s *RateService) SetDBFallback(fb DBFallback) {
	s.dbFallback = fb
}

// FetchRates calls the external API and returns all rates.
// Rates are keyed by currency code, value = amount of that currency per 1 USD.
func (s *RateService) FetchRates() (map[string]float64, error) {
	if !s.enabled {
		return nil, fmt.Errorf("exchange rate service is disabled")
	}

	resp, err := s.client.Get(s.apiURL)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}

	if apiResp.Result != "success" {
		return nil, fmt.Errorf("API result: %s", apiResp.Result)
	}

	s.mu.Lock()
	s.cache = apiResp.Rates
	s.lastFetch = time.Now()
	s.mu.Unlock()

	log.Printf("[Exchange] Fetched %d live rates from API (base: %s)", len(apiResp.Rates), apiResp.BaseCode)
	return apiResp.Rates, nil
}

// GetRate returns the USD-to-currency rate. Returns 0 if not found.
// Auto-refreshes if cache is expired.
func (s *RateService) GetRate(currency string) (float64, error) {
	if currency == "USD" {
		return 1.0, nil
	}

	s.mu.RLock()
	rate, ok := s.cache[currency]
	cacheAge := time.Since(s.lastFetch)
	s.mu.RUnlock()

	if !ok || (s.ttl > 0 && cacheAge > s.ttl) {
		// Cache miss or expired — try refresh
		if _, err := s.FetchRates(); err != nil {
			// If refresh fails, try stale cache
			s.mu.RLock()
			rate, ok = s.cache[currency]
			s.mu.RUnlock()
			if !ok {
				return 0, fmt.Errorf("rate not found for %s (API unavailable)", currency)
			}
			log.Printf("[Exchange] Using stale cache for %s: %.4f", currency, rate)
			return rate, nil
		}
		// Re-read from refreshed cache
		s.mu.RLock()
		rate, ok = s.cache[currency]
		s.mu.RUnlock()
		if !ok {
			return 0, fmt.Errorf("rate not found for %s", currency)
		}
	}

	return rate, nil
}

// StartAutoRefresh runs a background goroutine that periodically refreshes rates.
func (s *RateService) StartAutoRefresh(interval time.Duration) {
	if !s.enabled {
		log.Println("[Exchange] Service disabled, skipping auto-refresh")
		return
	}

	// Initial fetch
	if _, err := s.FetchRates(); err != nil {
		log.Printf("[Exchange] Initial fetch failed: %v (will retry)", err)
	} else {
		log.Println("[Exchange] Initial rates loaded successfully")
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := s.FetchRates(); err != nil {
				log.Printf("[Exchange] Auto-refresh failed: %v", err)
			}
		}
	}()
	log.Printf("[Exchange] Auto-refresh started (interval: %v)", interval)
}

// Convert performs a two-hop conversion: Source → USD → Target.
// Step 1: usdAmount = (sourceAmount - fee) / usdRate[sourceCurrency]
// Step 2: targetAmount = usdAmount * usdRate[targetCurrency]
// Returns the USD amount, target amount, and both conversion rates.
// Falls back to DB direct conversion if API rates are unavailable.
func (s *RateService) Convert(sourceAmount, fee int64, sourceCurrency, targetCurrency string) (usdAmount, targetAmount int64, srcToUsd, usdToTgt float64, err error) {
	netAmount := sourceAmount - fee
	if netAmount <= 0 {
		return 0, 0, 0, 0, fmt.Errorf("net amount is zero or negative after fee")
	}

	// Same currency — no conversion needed
	if sourceCurrency == targetCurrency {
		usdAmount = netAmount
		if sourceCurrency == "USD" {
			return usdAmount, netAmount, 1.0, 1.0, nil
		}
		// Try to get the USD rate for this currency
		rate, rErr := s.GetRate(sourceCurrency)
		if rErr == nil && rate > 0 {
			usdAmount = int64(float64(netAmount) / rate)
			return usdAmount, netAmount, 1.0 / rate, 1.0, nil
		}
		// Fallback: use DB
		if s.dbFallback != nil {
			dbRate, dbErr := s.dbFallback.GetExchangeRate(sourceCurrency, "USD")
			if dbErr == nil && dbRate.Rate > 0 {
				srcToUsd = dbRate.Rate
				usdAmount = int64(float64(netAmount) * srcToUsd)
				return usdAmount, netAmount, srcToUsd, 1.0, nil
			}
		}
		return 0, netAmount, 0, 0, nil
	}

	// Try live rates first
	srcRate, err1 := s.GetRate(sourceCurrency)
	tgtRate, err2 := s.GetRate(targetCurrency)

	if err1 == nil && err2 == nil && srcRate > 0 && tgtRate > 0 {
		// Live rates: srcRate = USD→Source, tgtRate = USD→Target
		srcToUsd = 1.0 / srcRate // e.g., 1/7.25 = 0.1379 (1 CNY = 0.1379 USD)
		usdToTgt = tgtRate        // e.g., 0.92 (1 USD = 0.92 EUR)
		usdAmount = int64(float64(netAmount) * srcToUsd)
		targetAmount = int64(float64(usdAmount) * usdToTgt)
		return usdAmount, targetAmount, srcToUsd, usdToTgt, nil
	}

	// Fallback: use database rates with USD as intermediate
	if s.dbFallback != nil {
		// Step 1: Source → USD
		srcToUsdRate := 1.0
		if sourceCurrency != "USD" {
			dbRate, dbErr := s.dbFallback.GetExchangeRate(sourceCurrency, "USD")
			if dbErr != nil {
				// Try reverse
				dbRate, dbErr = s.dbFallback.GetExchangeRate("USD", sourceCurrency)
				if dbErr != nil {
					return 0, 0, 0, 0, fmt.Errorf("no rate for %s→USD: %w", sourceCurrency, dbErr)
				}
				srcToUsdRate = 1.0 / dbRate.Rate
			} else {
				srcToUsdRate = dbRate.Rate
			}
		}
		usdAmount = int64(float64(netAmount) * srcToUsdRate)

		// Step 2: USD → Target
		usdToTgtRate := 1.0
		if targetCurrency != "USD" {
			dbRate, dbErr := s.dbFallback.GetExchangeRate("USD", targetCurrency)
			if dbErr != nil {
				dbRate, dbErr = s.dbFallback.GetExchangeRate(targetCurrency, "USD")
				if dbErr != nil {
					return 0, 0, 0, 0, fmt.Errorf("no rate for USD→%s: %w", targetCurrency, dbErr)
				}
				usdToTgtRate = 1.0 / dbRate.Rate
			} else {
				usdToTgtRate = dbRate.Rate
			}
		}
		targetAmount = int64(float64(usdAmount) * usdToTgtRate)

		log.Printf("[Exchange] Using DB fallback: %s→USD rate=%.4f, USD→%s rate=%.4f",
			sourceCurrency, srcToUsdRate, targetCurrency, usdToTgtRate)
		return usdAmount, targetAmount, srcToUsdRate, usdToTgtRate, nil
	}

	srcRateErr := err1
	if srcRateErr == nil {
		srcRateErr = err2
	}
	return 0, 0, 0, 0, fmt.Errorf("conversion failed: %v", srcRateErr)
}

// GetCacheAge returns how long ago the cache was last updated.
func (s *RateService) GetCacheAge() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.lastFetch)
}

// GetCachedRates returns a copy of the cached rates map.
func (s *RateService) GetCachedRates() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]float64, len(s.cache))
	for k, v := range s.cache {
		cp[k] = v
	}
	return cp
}
