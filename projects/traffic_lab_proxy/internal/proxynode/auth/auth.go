package auth

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"sync"
)

// Authenticator validates proxy connection credentials against a local policy cache.
type Authenticator struct {
	mu     sync.RWMutex
	tokens map[string]*UserCredential // token -> credential
	users  map[string]*UserCredential // username -> credential
}

// UserCredential holds the essential authentication data for a proxy user.
type UserCredential struct {
	UserID         string
	Token          string
	MaxRateMbps    int
	BurstMbps      int
	MaxConnections int
	IsActive       bool
}

// NewAuthenticator creates a new authenticator.
func NewAuthenticator() *Authenticator {
	return &Authenticator{
		tokens: make(map[string]*UserCredential),
		users:  make(map[string]*UserCredential),
	}
}

// UpdateCredentials replaces the entire credential cache atomically.
// Called by the policy fetcher when policies change.
func (a *Authenticator) UpdateCredentials(creds []*UserCredential) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.tokens = make(map[string]*UserCredential, len(creds))
	a.users = make(map[string]*UserCredential, len(creds))

	for _, c := range creds {
		a.tokens[c.Token] = c
		a.users[c.UserID] = c
	}
}

// Authenticate validates an HTTP CONNECT request and returns the user ID and target address.
// Format: CONNECT host:port HTTP/1.1\r\nProxy-Authorization: Basic base64(username:token)
func (a *Authenticator) Authenticate(conn net.Conn) (userID, target string, err error) {
	reader := bufio.NewReader(conn)

	// Read CONNECT line
	connectLine, err := reader.ReadString('\n')
	if err != nil {
		return "", "", fmt.Errorf("read connect line: %w", err)
	}

	connectLine = strings.TrimSpace(connectLine)
	parts := strings.SplitN(connectLine, " ", 3)
	if len(parts) < 2 || strings.ToUpper(parts[0]) != "CONNECT" {
		return "", "", fmt.Errorf("invalid CONNECT request: %s", connectLine)
	}

	target = parts[1]

	// Read headers until empty line
	var username, token string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", "", fmt.Errorf("read headers: %w", err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break // End of headers
		}

		// Parse Proxy-Authorization header
		if strings.HasPrefix(strings.ToLower(line), "proxy-authorization:") {
			authValue := strings.TrimSpace(line[len("proxy-authorization:"):])
			username, token, err = parseBasicAuth(authValue)
			if err != nil {
				return "", "", fmt.Errorf("parse auth: %w", err)
			}
		}
	}

	if username == "" || token == "" {
		return "", "", fmt.Errorf("missing Proxy-Authorization header")
	}

	// Validate credentials
	a.mu.RLock()
	defer a.mu.RUnlock()

	// First try token lookup directly
	cred, ok := a.tokens[token]
	if !ok {
		// Try username lookup
		cred, ok = a.users[username]
	}

	if !ok {
		return "", "", fmt.Errorf("invalid credentials")
	}

	if !cred.IsActive {
		return "", "", fmt.Errorf("user %s is disabled", username)
	}

	if cred.Token != token {
		return "", "", fmt.Errorf("token mismatch for user %s", username)
	}

	return cred.UserID, target, nil
}

// parseBasicAuth parses a Basic authentication header value.
func parseBasicAuth(authValue string) (username, password string, err error) {
	// Expected: "Basic base64encoded"
	parts := strings.SplitN(authValue, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "basic" {
		return "", "", fmt.Errorf("unsupported auth scheme")
	}

	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", fmt.Errorf("invalid base64: %w", err)
	}

	creds := strings.SplitN(string(decoded), ":", 2)
	if len(creds) != 2 {
		return "", "", fmt.Errorf("invalid credentials format")
	}

	return creds[0], creds[1], nil
}
