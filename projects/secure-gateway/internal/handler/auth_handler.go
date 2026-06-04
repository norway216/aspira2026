package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/secure-gateway/internal/auth"
	"github.com/secure-gateway/internal/model"
	"github.com/secure-gateway/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	jwtManager *auth.JWTManager
	db         *sql.DB // used for user queries
}

// NewAuthHandler creates a new authentication handler.
func NewAuthHandler(jwtManager *auth.JWTManager, db *sql.DB) *AuthHandler {
	return &AuthHandler{jwtManager: jwtManager, db: db}
}

// Login authenticates a user and returns tokens.
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "Invalid request: " + err.Error()})
		return
	}

	// Query user from database
	var user model.User
	err := h.db.QueryRow(
		"SELECT id, username, password_hash, role, status FROM users WHERE username = $1",
		req.Username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.Status)
	if err != nil {
		logger.Warn("Login failed - user not found", "username", req.Username)
		c.JSON(http.StatusUnauthorized, model.APIResponse{Code: 401, Message: "Invalid username or password"})
		return
	}

	if user.Status != model.StatusActive {
		logger.Warn("Login failed - user disabled", "username", req.Username)
		c.JSON(http.StatusForbidden, model.APIResponse{Code: 403, Message: "Account is disabled"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		logger.Warn("Login failed - wrong password", "username", req.Username)
		c.JSON(http.StatusUnauthorized, model.APIResponse{Code: 401, Message: "Invalid username or password"})
		return
	}

	accessToken, expiresIn, err := h.jwtManager.GenerateAccessToken(user.ID, user.Username, user.Role)
	if err != nil {
		logger.Error("Failed to generate access token", "error", err)
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "Internal server error"})
		return
	}

	refreshToken, _, err := h.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		logger.Error("Failed to generate refresh token", "error", err)
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "Internal server error"})
		return
	}

	// Log audit
	_, _ = h.db.Exec(
		"INSERT INTO audit_logs (user_id, action, resource, ip_addr, user_agent) VALUES ($1, $2, $3, $4, $5)",
		user.ID, "login", "/api/v1/auth/login", c.ClientIP(), c.Request.UserAgent(),
	)

	logger.Info("User logged in", "user", user.Username, "role", user.Role)
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    200,
		Message: "Login successful",
		Data: model.LoginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    expiresIn,
		},
	})
}

// Refresh issues a new access token using a refresh token.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req model.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "Invalid request"})
		return
	}

	userID, err := h.jwtManager.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.APIResponse{Code: 401, Message: "Invalid or expired refresh token"})
		return
	}

	// Fetch user
	var user model.User
	err = h.db.QueryRow(
		"SELECT id, username, role, status FROM users WHERE id = $1",
		userID,
	).Scan(&user.ID, &user.Username, &user.Role, &user.Status)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.APIResponse{Code: 401, Message: "User not found"})
		return
	}

	if user.Status != model.StatusActive {
		c.JSON(http.StatusForbidden, model.APIResponse{Code: 403, Message: "Account is disabled"})
		return
	}

	accessToken, expiresIn, err := h.jwtManager.GenerateAccessToken(user.ID, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    200,
		Message: "Token refreshed",
		Data: model.LoginResponse{
			AccessToken:  accessToken,
			RefreshToken: req.RefreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    expiresIn,
		},
	})
}

// Logout handles user logout (placeholder for token blacklist).
func (h *AuthHandler) Logout(c *gin.Context) {
	// In production, add the access token to a blacklist in Redis
	username, _ := c.Get("username")
	logger.Info("User logged out", "user", username)

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    200,
		Message: "Logout successful",
	})
}

// GetProfile returns the current user's profile.
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var user model.User
	err := h.db.QueryRow(
		"SELECT id, username, role, status, traffic_quota_bytes, traffic_used_bytes, created_at FROM users WHERE id = $1",
		userID,
	).Scan(&user.ID, &user.Username, &user.Role, &user.Status, &user.TrafficQuotaBytes, &user.TrafficUsedBytes, &user.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "Failed to get profile"})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    200,
		Message: "Success",
		Data:    user,
	})
}