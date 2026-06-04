package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// TokenClaims holds the JWT custom claims.
type TokenClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT token operations.
type JWTManager struct {
	issuer           string
	signingKey       []byte
	refreshSigningKey []byte
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
}

// NewJWTManager creates a new JWT manager.
func NewJWTManager(issuer, accessTTL, refreshTTL, signingKey, refreshKey string) (*JWTManager, error) {
	accessDuration, err := time.ParseDuration(accessTTL)
	if err != nil {
		return nil, fmt.Errorf("invalid access token TTL: %w", err)
	}
	refreshDuration, err := time.ParseDuration(refreshTTL)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token TTL: %w", err)
	}

	return &JWTManager{
		issuer:            issuer,
		signingKey:        []byte(signingKey),
		refreshSigningKey: []byte(refreshKey),
		accessTokenTTL:    accessDuration,
		refreshTokenTTL:   refreshDuration,
	}, nil
}

// GenerateAccessToken creates a new access token.
func (m *JWTManager) GenerateAccessToken(userID int64, username, role string) (string, int64, error) {
	now := time.Now()
	expiresAt := now.Add(m.accessTokenTTL)

	claims := TokenClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Subject:   fmt.Sprintf("%d", userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.signingKey)
	if err != nil {
		return "", 0, err
	}

	return signed, expiresAt.Unix(), nil
}

// GenerateRefreshToken creates a new refresh token.
func (m *JWTManager) GenerateRefreshToken(userID int64) (string, int64, error) {
	now := time.Now()
	expiresAt := now.Add(m.refreshTokenTTL)

	claims := jwt.RegisteredClaims{
		Issuer:    m.issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		Subject:   fmt.Sprintf("%d", userID),
		ID:        fmt.Sprintf("refresh-%d-%d", userID, now.Unix()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.refreshSigningKey)
	if err != nil {
		return "", 0, err
	}

	return signed, expiresAt.Unix(), nil
}

// ValidateAccessToken validates an access token and returns its claims.
func (m *JWTManager) ValidateAccessToken(tokenStr string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &TokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.signingKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token and returns the user ID.
func (m *JWTManager) ValidateRefreshToken(tokenStr string) (int64, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.refreshSigningKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return 0, ErrExpiredToken
		}
		return 0, ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return 0, ErrInvalidToken
	}

	var userID int64
	_, err = fmt.Sscanf(claims.Subject, "%d", &userID)
	if err != nil {
		return 0, ErrInvalidToken
	}

	return userID, nil
}