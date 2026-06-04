package policy

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
	"github.com/aspira2026/traffic_lab_proxy/internal/proxynode/auth"
	"github.com/aspira2026/traffic_lab_proxy/internal/proxynode/limiter"
)

// Cache holds the latest policies fetched from the control center.
type Cache struct {
	mu       sync.RWMutex
	policies map[string]*common.UserPolicySummary
	global   common.GlobalPolicy
	nodeID   string
}

// NewCache creates a new policy cache.
func NewCache(nodeID string) *Cache {
	return &Cache{
		policies: make(map[string]*common.UserPolicySummary),
		nodeID:   nodeID,
	}
}

// GetUserPolicy returns the cached policy for a user.
func (c *Cache) GetUserPolicy(userID string) (*common.UserPolicySummary, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p, ok := c.policies[userID]
	return p, ok
}

// GlobalPolicy returns the cached global (node-level) policy.
func (c *Cache) GlobalPolicy() common.GlobalPolicy {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.global
}

// ReplaceAll atomically replaces the entire policy set.
func (c *Cache) ReplaceAll(policies []common.UserPolicySummary, global common.GlobalPolicy) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.policies = make(map[string]*common.UserPolicySummary, len(policies))
	for i := range policies {
		c.policies[policies[i].UserID] = &policies[i]
	}
	c.global = global
}

// UpdateAuthenticator updates the auth cache with current user credentials.
func (c *Cache) UpdateAuthenticator(authenticator *auth.Authenticator) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	creds := make([]*auth.UserCredential, 0, len(c.policies))
	for _, p := range c.policies {
		creds = append(creds, &auth.UserCredential{
			UserID:         p.UserID,
			Token:          p.Token,
			MaxRateMbps:    p.MaxRateMbps,
			BurstMbps:      p.BurstMbps,
			MaxConnections: p.MaxConnections,
			IsActive:       p.Status == "active",
		})
	}
	authenticator.UpdateCredentials(creds)
}

// UpdateLimiter updates the rate limiter with current policies.
func (c *Cache) UpdateLimiter(pl *limiter.PerUserLimiter) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	policies := make(map[string]*limiter.PolicyConfig, len(c.policies))
	for _, p := range c.policies {
		policies[p.UserID] = &limiter.PolicyConfig{
			MaxRateMbps:    p.MaxRateMbps,
			BurstMbps:      p.BurstMbps,
			MaxConnections: p.MaxConnections,
		}
	}
	pl.UpdateAllPolicies(policies)
}

// PolicyFetcher periodically fetches policies from the control center.
type PolicyFetcher struct {
	client   *http.Client
	cache    *Cache
	auth     *auth.Authenticator
	limiter  *limiter.PerUserLimiter
	baseURL  string
	nodeID   string
	secret   string
	interval time.Duration
}

// NewPolicyFetcher creates a policy fetcher.
func NewPolicyFetcher(baseURL, nodeID, secret string, cache *Cache,
	auth *auth.Authenticator, limiter *limiter.PerUserLimiter, interval time.Duration) *PolicyFetcher {

	return &PolicyFetcher{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:    cache,
		auth:     auth,
		limiter:  limiter,
		baseURL:  baseURL,
		nodeID:   nodeID,
		secret:   secret,
		interval: interval,
	}
}

// Start begins periodic policy fetching. Runs until ctx is cancelled.
func (pf *PolicyFetcher) Start(ctx context.Context) {
	// Fetch immediately on startup
	pf.fetch(ctx)

	ticker := time.NewTicker(pf.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pf.fetch(ctx)
		}
	}
}

func (pf *PolicyFetcher) fetch(ctx context.Context) {
	url := fmt.Sprintf("%s/api/v1/nodes/%s/policies", pf.baseURL, pf.nodeID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		slog.Error("policy fetcher: create request", "error", err)
		return
	}

	// Add node auth headers
	body := []byte{}
	sig := computeHMAC(body, pf.secret)
	req.Header.Set("X-Node-Id", pf.nodeID)
	req.Header.Set("X-Node-Signature", sig)

	resp, err := pf.client.Do(req)
	if err != nil {
		slog.Error("policy fetcher: request failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		slog.Error("policy fetcher: non-200 response", "status", resp.StatusCode, "body", string(respBody))
		return
	}

	var policyResp common.PolicyResponse
	if err := json.NewDecoder(resp.Body).Decode(&policyResp); err != nil {
		slog.Error("policy fetcher: decode error", "error", err)
		return
	}

	// Update caches
	pf.cache.ReplaceAll(policyResp.UserPolicies, policyResp.GlobalPolicy)
	pf.cache.UpdateAuthenticator(pf.auth)
	pf.cache.UpdateLimiter(pf.limiter)

	slog.Info("policies updated",
		"users", len(policyResp.UserPolicies),
		"max_bw", policyResp.GlobalPolicy.MaxNodeBandwidthMbps,
		"max_conns", policyResp.GlobalPolicy.MaxNodeConnections,
	)
}

func computeHMAC(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Ensure unused import doesn't error
var _ = bytes.NewReader
