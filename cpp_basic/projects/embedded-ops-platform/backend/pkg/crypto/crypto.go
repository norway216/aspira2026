package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateToken generates a cryptographically secure random token in hex.
func GenerateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// HashToken creates a SHA-256 HMAC hash of the token with the given secret.
func HashToken(token, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyToken verifies a token against its hash.
func VerifyToken(token, hash, secret string) bool {
	expected := HashToken(token, secret)
	return hmac.Equal([]byte(expected), []byte(hash))
}

// GenerateDeviceID creates a device ID in the format: dev-{type}-{random-hex}
func GenerateDeviceID(boardType string) (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate device ID: %w", err)
	}
	shortType := boardType
	if len(shortType) > 8 {
		shortType = shortType[:8]
	}
	return fmt.Sprintf("dev-%s-%s", shortType, hex.EncodeToString(bytes)), nil
}