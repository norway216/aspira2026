package util

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// HashIP creates a SHA-256 hash of an IP address with a salt for privacy.
func HashIP(ip, salt string) string {
	h := sha256.New()
	h.Write([]byte(ip + salt))
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// GenerateToken creates a cryptographically random token.
func GenerateToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// MustGenerateToken creates a token or panics.
func MustGenerateToken(length int) string {
	t, err := GenerateToken(length)
	if err != nil {
		panic(fmt.Sprintf("failed to generate token: %v", err))
	}
	return t
}

// FormatTime formats a time for display.
func FormatTime(t time.Time, layout string) string {
	return t.Format(layout)
}

// FormatDateShort formats a date as "2006-01-02".
func FormatDateShort(t time.Time) string {
	return t.Format("2006-01-02")
}

// FormatDateLong formats a date as "January 2, 2006".
func FormatDateLong(t time.Time) string {
	return t.Format("January 2, 2006")
}

// FormatDateTime formats a datetime as "2006-01-02 15:04".
func FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

// NowPtr returns a pointer to the current time.
func NowPtr() *time.Time {
	now := time.Now()
	return &now
}

// TimePtr returns a pointer to the given time.
func TimePtr(t time.Time) *time.Time {
	return &t
}

// StringPtr returns a pointer to the given string.
func StringPtr(s string) *string {
	return &s
}

// Int64Ptr returns a pointer to the given int64.
func Int64Ptr(i int64) *int64 {
	return &i
}

// BoolPtr returns a pointer to the given bool.
func BoolPtr(b bool) *bool {
	return &b
}
